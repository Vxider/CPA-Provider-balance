package main

import "testing"

func TestParseBalanceUsage(t *testing.T) {
	v := map[string]any{
		"remaining":  12.5,
		"total_used": 3.2,
		"limit":      20.0,
		"unit":       "USD",
	}
	p, ok := parseBalanceValue(shapeUsage, v)
	if !ok {
		t.Fatal("expected ok")
	}
	if p.remaining == nil || *p.remaining != 12.5 {
		t.Fatalf("remaining=%v", p.remaining)
	}
	if p.total == nil || *p.total != 20.0 {
		t.Fatalf("total=%v", p.total)
	}
	if p.unit != "USD" {
		t.Fatalf("unit=%q", p.unit)
	}
}

func TestParseBalanceZhipu(t *testing.T) {
	v := map[string]any{
		"data": map[string]any{
			"total_balance": 500.0,
			"used":          120.0,
		},
	}
	p, ok := parseBalanceValue(shapeZhipu, v)
	if !ok {
		t.Fatal("expected ok")
	}
	if p.remaining == nil || *p.remaining != 500.0 {
		t.Fatalf("remaining=%v", p.remaining)
	}
	if p.used == nil || *p.used != 120.0 {
		t.Fatalf("used=%v", p.used)
	}
	if p.unit != "CNY" {
		t.Fatalf("unit=%q", p.unit)
	}
}

func TestParseBalanceRemainingDerived(t *testing.T) {
	v := map[string]any{
		"quota": map[string]any{
			"limit": 100.0,
			"used":  30.0,
		},
	}
	p, ok := parseBalanceValue(shapeUsage, v)
	if !ok {
		t.Fatal("expected ok")
	}
	if p.remaining == nil || *p.remaining != 70.0 {
		t.Fatalf("remaining=%v", p.remaining)
	}
}

func TestProviderKindFromBaseURL(t *testing.T) {
	cases := map[string]string{
		"https://open.bigmodel.cn/api/paas/v4": "bigmodel",
		"https://api.openai.com/v1":            "provider",

		"https://www.vastnum.com/v1": "vastnum",
		"https://aifxo.com/v1":       "aifxo",
	}
	for base, want := range cases {
		got := providerKindFromBaseURL(base)
		if got != want {
			t.Errorf("kind(%q)=%q want %q", base, got, want)
		}
	}
}

func TestIsZhipuBaseURL(t *testing.T) {
	if !isZhipuBaseURL("https://open.bigmodel.cn/api/paas/v4") {
		t.Fatal("bigmodel should be zhipu")
	}
	if isZhipuBaseURL("https://api.openai.com/v1") {
		t.Fatal("openai should not be zhipu")
	}
}

func TestEndpointsForZhipu(t *testing.T) {
	eps := endpointsFor("https://open.bigmodel.cn/api/paas/v4")
	if len(eps) == 0 || eps[0].shape != shapeZhipu {
		t.Fatalf("expected zhipu endpoints, got %+v", eps)
	}
}

func TestMaskKey(t *testing.T) {
	got := maskKey("sk-abcdefghij123")
	if got != "sk-a...j123" {
		t.Fatalf("mask=%q", got)
	}
}

func TestDedupeProviders(t *testing.T) {
	in := []providerEntry{
		{Name: "a", BaseURL: "https://x", APIKey: "sk-abcdefghij1"},
		{Name: "a", BaseURL: "https://x", APIKey: "sk-abcdefghij1"},
	}
	out := dedupeProviders(in)
	if len(out) != 1 {
		t.Fatalf("dedupe len=%d", len(out))
	}
}

func TestExtractSectionFlatAPIKey(t *testing.T) {
	yml := []byte(`
codex-api-key:
  - api-key: sk-testkey1234
    base-url: https://api.example.com
`)
	// exercise the yaml.Node path used in production.
	cfg, err := parseForTest(yml)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg) != 1 || cfg[0].APIKey != "sk-testkey1234" {
		t.Fatalf("got %+v", cfg)
	}
}

func TestExtractSectionFromBytesUsesRequestedSection(t *testing.T) {
	yml := []byte(`
codex-api-key:
  - api-key: sk-codex1234
    base-url: https://api.codex.example.com
gemini-api-key:
  - api-key: sk-gemini1234
    base-url: https://api.gemini.example.com
`)
	cfg, err := extractSectionFromBytes(yml, "gemini-api-key")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg) != 1 || cfg[0].APIKey != "sk-gemini1234" {
		t.Fatalf("got %+v", cfg)
	}
}

func parseForTest(data []byte) ([]providerEntry, error) {
	return extractSectionFromBytes(data, "codex-api-key")
}
