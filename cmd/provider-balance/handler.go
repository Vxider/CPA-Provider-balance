package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
	"gopkg.in/yaml.v3"
)

// handleManagement dispatches an inbound management/resource HTTP request that
// the host routed to this plugin via MethodManagementHandle. It returns an
// envelope-wrapped pluginapi.ManagementResponse (ok=true, result=<resp>), which
// the host decodes with callPlugin[pluginapi.ManagementResponse].
func handleManagement(raw []byte) ([]byte, error) {
	var req rpcManagementRequest
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &req); err != nil {
			return managementEnvelope(http.StatusBadRequest, map[string]any{"error": "invalid request: " + err.Error()})
		}
	}
	pathLower := strings.ToLower(req.Path)
	if strings.HasSuffix(pathLower, pluginResourcePing) {
		return serveConnectivityJSON(req.Query)
	}
	accept := strings.ToLower(strings.Join(req.Headers.Values("Accept"), ", "))
	if strings.HasSuffix(pathLower, pluginManagementPath) || strings.HasSuffix(pathLower, pluginResourceJSON) {
		return serveBalanceJSON()
	}
	if strings.Contains(accept, "application/json") && !strings.Contains(accept, "text/html") {
		return serveBalanceJSON()
	}
	return serveBalanceHTML()
}

// rpcManagementRequest mirrors the host's wire shape (embedded ManagementRequest
// plus an opaque host callback id we ignore here).
type rpcManagementRequest struct {
	pluginapi.ManagementRequest
	HostCallbackID string `json:"host_callback_id,omitempty"`
}

func managementEnvelope(status int, body any) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	resp := pluginapi.ManagementResponse{
		StatusCode: status,
		Headers: http.Header{
			"Content-Type":  []string{"application/json; charset=utf-8"},
			"Cache-Control": []string{"no-store"},
		},
		Body: payload,
	}
	return okEnvelope(resp)
}

func managementEnvelopeHTML(status int, body string) ([]byte, error) {
	resp := pluginapi.ManagementResponse{
		StatusCode: status,
		Headers: http.Header{
			"Content-Type":  []string{"text/html; charset=utf-8"},
			"Cache-Control": []string{"no-store"},
		},
		Body: []byte(body),
	}
	return okEnvelope(resp)
}

func collectProviders(cfg pluginConfig) []providerEntry {
	out := make([]providerEntry, 0, len(cfg.ExtraProviders))
	if cfg.AutoDetectCPAConfig {
		out = append(out, providersFromCPAConfig()...)
	}
	out = append(out, cfg.ExtraProviders...)
	return dedupeProviders(out)
}

func serveBalanceJSON() ([]byte, error) {
	cfg := loadedConfig()
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	providers := collectProviders(cfg)
	reports := queryProviderBalances(providers, timeout, cfg.AutoDetectCPAConfig)
	envelope := map[string]any{
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"count":        len(reports),
		"providers":    reports,
	}
	return managementEnvelope(http.StatusOK, envelope)
}

func serveBalanceHTML() ([]byte, error) {
	return managementEnvelopeHTML(http.StatusOK, renderDashboardHTML())
}

func serveConnectivityJSON(query url.Values) ([]byte, error) {
	cfg := loadedConfig()
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	providers := collectProviders(cfg)
	// When the caller supplies provider / base_url query params, narrow to a
	// single provider so the frontend can stream results per row.
	if name := query.Get("provider"); name != "" || query.Get("base_url") != "" {
		providers = filterProviders(providers, query.Get("provider"), query.Get("base_url"))
	}
	reports := queryProviderConnectivity(providers, timeout, cfg.AutoDetectCPAConfig)
	envelope := map[string]any{
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"count":        len(reports),
		"providers":    reports,
	}
	return managementEnvelope(http.StatusOK, envelope)
}

// filterProviders narrows the collected providers to those matching the given
// display name and base URL, using the same normalization the JSON report
// produces so the frontend can round-trip a row's identity.
func filterProviders(providers []providerEntry, name, baseURL string) []providerEntry {
	var out []providerEntry
	for _, p := range providers {
		base := normalizeBaseURL(p.BaseURL)
		if name != "" && displayNameFor(p, base) != name {
			continue
		}
		if baseURL != "" && base != baseURL {
			continue
		}
		out = append(out, p)
	}
	return out
}

func providersFromCPAConfig() []providerEntry {
	configPath := findCPAConfigPath()
	if configPath == "" {
		return nil
	}
	data, err := readConfigFile(configPath)
	if err != nil {
		return nil
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return nil
	}
	mapping := root.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil
	}
	sections := mappingToMap(mapping)
	var out []providerEntry
	out = append(out, extractSection(sections, "openai-compatibility", providerKindOpenAICompat)...)
	out = append(out, extractSection(sections, "codex-api-key", providerKindCodexKey)...)
	out = append(out, extractSection(sections, "gemini-api-key", providerKindGeminiKey)...)
	out = append(out, extractSection(sections, "claude-api-key", providerKindClaudeKey)...)
	out = append(out, extractSection(sections, "vertex-api-key", providerKindVertexKey)...)
	return out
}
