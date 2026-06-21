package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type balanceReport struct {
	Provider  string   `json:"provider"`
	Kind      string   `json:"kind"`
	BaseURL   string   `json:"base_url"`
	APIKey    string   `json:"api_key"`
	Status    string   `json:"status"`
	Remaining *float64 `json:"remaining,omitempty"`
	Used      *float64 `json:"used,omitempty"`
	Total     *float64 `json:"total,omitempty"`
	Unit      string   `json:"unit"`
	Note      string   `json:"note"`
}

type balanceEndpoint struct {
	path  string
	shape balanceShape
}

type balanceShape int

const (
	shapeUsage balanceShape = iota
	shapeCreditGrants
	shapeSubscription
	shapeZhipu
)

var defaultEndpoints = []balanceEndpoint{
	{"/v1/usage", shapeUsage},
	{"/v1/dashboard/billing/credit_grants", shapeCreditGrants},
	{"/dashboard/billing/credit_grants", shapeCreditGrants},
	{"/v1/dashboard/billing/subscription", shapeSubscription},
	{"/dashboard/billing/subscription", shapeSubscription},
}

// queryProviderBalances queries each provider concurrently and returns reports.
// includeKind flags whether the kind label should be set (auto-detected entries).
func queryProviderBalances(providers []providerEntry, timeout time.Duration, includeKind bool) []balanceReport {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		reports = make([]balanceReport, 0, len(providers))
	)
	for _, p := range providers {
		if p.Disabled || p.BaseURL == "" || p.APIKey == "" {
			continue
		}
		wg.Add(1)
		go func(p providerEntry) {
			defer wg.Done()
			report := queryProviderBalance(p, timeout)
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

func queryProviderBalance(p providerEntry, timeout time.Duration) balanceReport {
	base := normalizeBaseURL(p.BaseURL)
	report := balanceReport{
		Provider: displayNameFor(p, base),
		BaseURL:  base,
		APIKey:   maskKey(p.APIKey),
		Status:   "Err",
		Unit:     "-",
		Note:     "-",
	}

	client := &http.Client{Timeout: timeout}
	endpoints := endpointsFor(base)

	var errors []string
	for _, ep := range endpoints {
		u, err := url.JoinPath(base, ep.path)
		if err != nil {
			continue
		}
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u, nil)
		if err != nil {
			errors = append(errors, ep.path+": "+err.Error())
			continue
		}
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
		req.Header.Set("Accept", "application/json")
		if isZhipuBaseURL(base) {
			req.Header.Set("Authorization", "Bearer "+p.APIKey)
		}
		resp, err := client.Do(req)
		if err != nil {
			errors = append(errors, ep.path+": "+compactErr(err))
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		_ = resp.Body.Close()
		if resp.StatusCode >= 500 {
			errors = append(errors, fmt.Sprintf("%s HTTP %d", ep.path, resp.StatusCode))
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			errors = append(errors, fmt.Sprintf("%s HTTP %d: %s", ep.path, resp.StatusCode, preview(body)))
			continue
		}
		var value map[string]any
		if err := json.Unmarshal(body, &value); err != nil {
			errors = append(errors, ep.path+": parse: "+compactErr(err))
			continue
		}
		parsed, ok := parseBalanceValue(ep.shape, value)
		if !ok {
			errors = append(errors, ep.path+": unsupported response")
			continue
		}
		report.Status = "OK"
		report.Remaining = parsed.remaining
		report.Used = parsed.used
		report.Total = parsed.total
		report.Unit = parsed.unit
		report.Note = parsed.note
		return report
	}
	if len(errors) > 0 {
		report.Note = strings.Join(errors, " | ")
	} else {
		report.Note = "no balance endpoint attempted"
	}
	return report
}

type parsedBalance struct {
	remaining *float64
	used      *float64
	total     *float64
	unit      string
	note      string
}

func parseBalanceValue(shape balanceShape, value map[string]any) (parsedBalance, bool) {
	switch shape {
	case shapeUsage:
		remaining := firstNumber(value, "remaining", "remain", "balance", "available", "quota_remaining", "total_available", "credit")
		used := pathNumber(value, "subscription", "monthly_usage_usd")
		if used == nil {
			used = pathNumber(value, "quota", "used")
		}
		if used == nil {
			used = pathNumber(value, "usage", "total", "actual_cost")
		}
		if used == nil {
			used = firstNumber(value, "total_used", "quota_used", "used")
		}
		total := pathNumber(value, "subscription", "monthly_limit_usd")
		if total == nil {
			total = pathNumber(value, "quota", "limit")
		}
		if total == nil {
			total = firstNumber(value, "limit", "total", "quota", "granted")
		}
		if remaining == nil && total != nil && used != nil {
			v := *total - *used
			if v < 0 {
				v = 0
			}
			remaining = &v
		}
		unit := firstString(value, "unit", "currency")
		if unit == "" {
			unit = "USD"
		}
		note := firstString(value, "message", "status", "planName", "plan")
		if note == "" {
			note = "-"
		}
		return parsedBalance{remaining, used, total, unit, note}, true
	case shapeCreditGrants:
		total := firstNumber(value, "total_granted", "granted", "hard_limit_usd", "system_hard_limit_usd")
		used := firstNumber(value, "total_used", "used")
		available := firstNumber(value, "total_available", "available", "remaining")
		remaining := available
		if remaining == nil && total != nil && used != nil {
			v := *total - *used
			remaining = &v
		}
		unit := firstString(value, "currency", "unit")
		if unit == "" {
			unit = "USD"
		}
		return parsedBalance{remaining, used, total, unit, "-"}, true
	case shapeSubscription:
		total := firstNumber(value, "hard_limit_usd", "system_hard_limit_usd")
		if total == nil {
			return parsedBalance{}, false
		}
		return parsedBalance{nil, nil, total, "USD", "subscription"}, true
	case shapeZhipu:
		// Zhipu BigModel /v1/usage or paas balance shape: {"data":{...}} or top-level.
		obj := value
		if inner, ok := value["data"].(map[string]any); ok {
			obj = inner
		}
		remaining := firstNumber(obj, "total_balance", "balance", "remaining", "available", "total_available")
		used := firstNumber(obj, "used", "total_used")
		total := firstNumber(obj, "total_granted", "granted", "total", "limit")
		if remaining == nil && total != nil && used != nil {
			v := *total - *used
			remaining = &v
		}
		unit := firstString(obj, "unit", "currency")
		if unit == "" {
			unit = "CNY"
		}
		if remaining == nil && used == nil && total == nil {
			return parsedBalance{}, false
		}
		return parsedBalance{remaining, used, total, unit, "zhipu"}, true
	}
	return parsedBalance{}, false
}

// endpointsFor returns the endpoint probe list for a provider. Zhipu (bigmodel)
// gets a dedicated endpoint first.
func endpointsFor(base string) []balanceEndpoint {
	if isZhipuBaseURL(base) {
		return []balanceEndpoint{
			{"/api/paas/v1/usage", shapeZhipu},
			{"/api/paas/v3/usage", shapeZhipu},
			{"/v1/usage", shapeZhipu},
		}
	}
	return defaultEndpoints
}

func isZhipuBaseURL(base string) bool {
	host := hostOf(base)
	return strings.Contains(host, "bigmodel") || strings.Contains(host, "zhipuai")
}

func providerKindFromBaseURL(base string) string {
	host := hostOf(base)
	host = strings.TrimPrefix(host, "www.")
	if host == "" {
		return "provider"
	}
	if isIPAddress(host) {
		return "local"
	}
	parts := strings.Split(host, ".")
	if len(parts) == 1 {
		return parts[0]
	}
	if len(parts) >= 2 {
		candidate := parts[len(parts)-2]
		if !isGenericHostPart(candidate) {
			return candidate
		}
	}
	for _, part := range parts {
		if !isGenericHostPart(part) {
			return part
		}
	}
	return "provider"
}

// isIPAddress reports whether s looks like a bare IPv4 address.
func isIPAddress(s string) bool {
	dots := 0
	for _, r := range s {
		if r == '.' {
			dots++
		} else if r < '0' || r > '9' {
			return false
		}
	}
	return dots == 3 && len(s) >= 7
}

func isGenericHostPart(value string) bool {
	switch value {
	case "api", "apis", "openai", "chat", "gateway", "proxy", "relay", "ai", "llm",
		"com", "cn", "net", "org", "io", "top", "cc", "vip":
		return true
	}
	return false
}

func normalizeBaseURL(base string) string {
	base = strings.TrimSpace(base)
	base = strings.TrimRight(base, "/")
	return base
}

func displayNameFor(p providerEntry, base string) string {
	if p.Name != "" {
		return p.Name
	}
	if kind := providerKindFromBaseURL(base); kind != "" {
		return kind
	}
	return base
}

func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return rawURL
	}
	return u.Hostname()
}

func maskKey(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func preview(b []byte) string {
	s := string(b)
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 120 {
		s = s[:120] + "..."
	}
	return s
}

func compactErr(err error) string {
	s := err.Error()
	if len(s) > 160 {
		s = s[:160] + "..."
	}
	return s
}

func firstNumber(value map[string]any, keys ...string) *float64 {
	for _, k := range keys {
		if v, ok := value[k]; ok {
			if f, ok := toFloat(v); ok {
				return &f
			}
		}
	}
	return nil
}

func pathNumber(value map[string]any, path ...string) *float64 {
	current := value
	for i, key := range path {
		v, ok := current[key]
		if !ok {
			return nil
		}
		if i == len(path)-1 {
			if f, ok := toFloat(v); ok {
				return &f
			}
			return nil
		}
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		current = m
	}
	return nil
}

func firstString(value map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := value[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case string:
		var f float64
		_, err := fmt.Sscanf(n, "%f", &f)
		return f, err == nil
	}
	return 0, false
}
