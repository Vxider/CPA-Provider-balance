# cpa-provider-balance

A CLIProxyAPI plugin that surfaces AI provider API-key balances in the
management panel via a dedicated dashboard page and JSON API.

## What it does

- Auto-detects API-key providers from the running CPA `config.yaml`
  (`openai-compatibility`, `codex-api-key`, `gemini-api-key`,
  `claude-api-key`, `vertex-api-key`).
- Queries each provider's balance endpoint concurrently.
- Exposes a browser-navigable dashboard at
  `/v0/resource/plugins/provider-balance/balance` (no management key needed).
- Exposes a JSON API at `/v0/resource/plugins/provider-balance/balance.json`.
- Exposes a management-authenticated JSON route at
  `/v0/management/provider-balance`.

## Build

```bash
make build      # builds bin/provider-balance.so
make install    # copies to cliproxyapi/plugins/linux/arm64/
make test       # runs unit tests
```

Requires Go 1.26+ and the CLIProxyAPI v7.2.22 SDK.

## Config

Add to `cliproxyapi/config.yaml` under `plugins.configs`:

```yaml
provider-balance:
  enabled: true
  timeout_seconds: 8
  auto_detect_cpa_config: true
  extra_providers:
    - name: "zhipu-glm"
      base_url: "https://open.bigmodel.cn/api/paas/v4"
      api_key: "YOUR_ZHIPU_KEY"
      disabled: false
```

Restart CPA after changing config. The plugin appears as a "Provider Balance"
menu entry in the management panel resource pages.

## Supported balance endpoints

The plugin probes these endpoints per provider (in order):
- `/v1/usage` (OpenAI-compatible usage shape)
- `/v1/dashboard/billing/credit_grants` (credit grants shape)
- `/dashboard/billing/credit_grants`
- `/v1/dashboard/billing/subscription` (subscription shape)
- `/dashboard/billing/subscription`

For Zhipu (bigmodel.cn) hosts, it probes Zhipu-specific paths:
- `/api/paas/v1/usage`
- `/api/paas/v3/usage`
- `/v1/usage`

## Files

- `cmd/provider-balance/main.go` - C ABI, plugin registration, management routes
- `cmd/provider-balance/handler.go` - HTTP request dispatch, JSON/HTML responses
- `cmd/provider-balance/balance.go` - Balance query logic and parsing
- `cmd/provider-balance/cpa_config.go` - CPA config.yaml auto-detection
- `cmd/provider-balance/dashboard.go` - HTML dashboard page
- `cmd/provider-balance/config.go` - Plugin config types
- `cmd/provider-balance/logic_test.go` - Unit tests
