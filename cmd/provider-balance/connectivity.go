package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// connectivityReport is the per-provider result of a health check. Unlike
// balanceReport it focuses on whether the model actually responds, not usage.
type connectivityReport struct {
	Provider string `json:"provider"`
	Kind     string `json:"kind"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
	Status   string `json:"status"` // ok | unreachable | unauthorized | no_model | no_endpoint | error
	Latency  int64  `json:"latency_ms"`
	Code     int    `json:"code"`
	Note     string `json:"note"`
}

// queryProviderConnectivity runs a health check against every enabled provider
// concurrently. For each provider it discovers a model via /v1/models, then
// issues a minimal chat-completion request to verify the model actually
// responds.
func queryProviderConnectivity(providers []providerEntry, timeout time.Duration, includeKind bool) []connectivityReport {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		reports = make([]connectivityReport, 0, len(providers))
	)
	for _, p := range providers {
		if p.Disabled || p.BaseURL == "" || p.APIKey == "" {
			continue
		}
		wg.Add(1)
		go func(p providerEntry) {
			defer wg.Done()
			report := healthCheckProvider(p, timeout)
			if includeKind {
				report.Kind = providerKindFromBaseURL(p.BaseURL)
			}
			if report.Kind == "" {
				report.Kind = "provider"
			}
			mu.Lock()
			reports = append(reports, report)
			mu.Unlock()
		}(p)
	}
	wg.Wait()
	return reports
}

// preferredModelFor returns a model id the health check should prefer when it
// is available in the provider's model list. Codex providers are probed with
// gpt-5.5 since that is the canonical Codex model; other providers fall back to
// the first available model (empty string = no preference).
func preferredModelFor(p providerEntry) string {
	if p.Kind == providerKindCodexKey {
		return "gpt-5.5"
	}
	return ""
}

// healthCheckProvider performs the two-step health check:
//  1. GET /v1/models to discover a usable model id (also validates auth).
//  2. POST a minimal chat completion with max_tokens=1 to confirm the model
//     actually responds.
// Latency is the wall-clock time of the model (chat-completion) request.
func healthCheckProvider(p providerEntry, timeout time.Duration) connectivityReport {
	base := normalizeBaseURL(p.BaseURL)
	report := connectivityReport{
		Provider: displayNameFor(p, base),
		BaseURL:  base,
		APIKey:   maskKey(p.APIKey),
		Status:   "error",
		Note:     "-",
	}

	client := &http.Client{Timeout: timeout}

	// Step 1: discover a model.
	model, modelsCode, modelsNote := discoverModel(client, base, p.APIKey, preferredModelFor(p))
	if model == "" {
		// Classify the discovery failure so the UI can distinguish auth /
		// reachability / endpoint problems.
		switch {
		case modelsNote == "unreachable":
			report.Status = "unreachable"
		case modelsNote == "unauthorized":
			report.Status = "unauthorized"
		case modelsNote == "no_endpoint":
			report.Status = "no_endpoint"
		default:
			report.Status = "no_model"
		}
		report.Code = modelsCode
		report.Note = "no model available: " + modelsNote
		return report
	}
	report.Model = model

	// Step 2: send a minimal chat completion to verify the model responds.
	body := map[string]any{
		"model":       model,
		"messages":    []map[string]string{{"role": "user", "content": "ping"}},
		"max_tokens":  1,
		"stream":      false,
		"temperature": 0,
	}
	payload, _ := json.Marshal(body)

	u := joinAPIPath(base, "/chat/completions")
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		report.Note = "request build failed: " + compactErr(err)
		return report
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	start := time.Now()
	resp, err := client.Do(req)
	report.Latency = time.Since(start).Milliseconds()
	if err != nil {
		report.Status = "unreachable"
		report.Note = compactErr(err)
		return report
	}
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	_ = resp.Body.Close()
	report.Code = resp.StatusCode

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		report.Status = "ok"
		report.Note = "model responded"
	case resp.StatusCode == 401 || resp.StatusCode == 403:
		report.Status = "unauthorized"
		report.Note = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, preview(respBody))
	case resp.StatusCode == 404:
		report.Status = "no_endpoint"
		report.Note = "chat endpoint not found (404)"
	default:
		report.Status = "error"
		report.Note = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, preview(respBody))
	}
	return report
}

// discoverModel GETs the models list and returns a model id to probe. If
// preferred is non-empty and present in the list it is returned; otherwise the
// first available id is used. code/note describe the request outcome for error
// reporting.
func discoverModel(client *http.Client, base, apiKey, preferred string) (model string, code int, note string) {
	u := joinAPIPath(base, "/models")
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u, nil)
	if err != nil {
		return "", 0, "request build failed"
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, "unreachable"
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	_ = resp.Body.Close()
	code = resp.StatusCode

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		// ok, parse below
	case resp.StatusCode == 401 || resp.StatusCode == 403:
		return "", code, "unauthorized"
	case resp.StatusCode == 404:
		return "", code, "no_endpoint"
	default:
		return "", code, fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	var value struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		// Some hosts return a bare array.
	}
	var ids []string
	if err := json.Unmarshal(body, &value); err == nil {
		for _, m := range value.Data {
			if m.ID != "" {
				ids = append(ids, m.ID)
			}
		}
	}
	// Try bare array shape: [{"id":"..."}, ...]
	var arr []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &arr); err == nil {
		for _, m := range arr {
			if m.ID != "" {
				ids = append(ids, m.ID)
			}
		}
	}
	// Prefer the requested model when the provider actually exposes it;
	// otherwise fall back to the first available id.
	for _, id := range ids {
		if id == preferred {
			return id, code, "ok"
		}
	}
	if len(ids) > 0 {
		return ids[0], code, "ok"
	}
	return "", code, "empty models list"
}
