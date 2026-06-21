package main

// pluginConfig is the plugin's own configuration block (plugins.configs.provider-balance).
type pluginConfig struct {
	TimeoutSeconds      int             `yaml:"timeout_seconds"`
	AutoDetectCPAConfig bool            `yaml:"auto_detect_cpa_config"`
	ExtraProviders      []providerEntry `yaml:"extra_providers"`
}

type providerEntry struct {
	Name     string `yaml:"name" json:"name"`
	BaseURL  string `yaml:"base_url" json:"base_url"`
	APIKey   string `yaml:"api_key" json:"api_key"`
	Disabled bool   `yaml:"disabled" json:"disabled"`
}

func defaultPluginConfig() pluginConfig {
	return pluginConfig{
		TimeoutSeconds:      8,
		AutoDetectCPAConfig: true,
	}
}
