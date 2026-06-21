package main

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	providerKindOpenAICompat = "openai-compatibility"
	providerKindCodexKey     = "codex-api-key"
	providerKindGeminiKey    = "gemini-api-key"
	providerKindClaudeKey    = "claude-api-key"
	providerKindVertexKey    = "vertex-api-key"
)

// findCPAConfigPath locates the cliproxyapi config.yaml used by the running
// server. It checks the binary's working directory and common install paths.
func findCPAConfigPath() string {
	candidates := []string{
		"/home/vxider/cliproxyapi/config.yaml",
		"./config.yaml",
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates,
			filepath.Join(filepath.Dir(exe), "config.yaml"),
			filepath.Join(filepath.Dir(exe), "..", "config.yaml"),
		)
	}
	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	return ""
}

func readConfigFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func mappingToMap(node *yaml.Node) map[string]*yaml.Node {
	out := make(map[string]*yaml.Node)
	if node == nil || node.Kind != yaml.MappingNode {
		return out
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := strings.TrimSpace(node.Content[i].Value)
		if key != "" {
			out[key] = node.Content[i+1]
		}
	}
	return out
}

// extractSection pulls provider entries from one config section. It handles both
// the api-key-entries array shape (openai-compatibility / vertex-api-key) and the
// flat api-key shape (codex-api-key / gemini-api-key / claude-api-key).
func extractSection(sections map[string]*yaml.Node, sectionName, kind string) []providerEntry {
	node, ok := sections[sectionName]
	if !ok || node == nil || node.Kind != yaml.SequenceNode {
		return nil
	}
	var out []providerEntry
	for idx, item := range node.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		fields := mappingToMap(item)
		disabled := boolField(fields, "disabled")
		baseURL := strField(fields, "base-url", "base_url", "baseURL", "baseUrl")
		displayName := strField(fields, "name", "prefix", "label")
		if displayName == "" {
			displayName = kind
		}

		if entries := fields["api-key-entries"]; entries != nil && entries.Kind == yaml.SequenceNode {
			for _, entry := range entries.Content {
				if entry.Kind != yaml.MappingNode {
					continue
				}
				ef := mappingToMap(entry)
				key := strField(ef, "api-key", "api_key", "apiKey", "key")
				if key == "" {
					continue
				}
				entryBase := strField(ef, "base-url", "base_url", "baseURL", "baseUrl")
				if entryBase == "" {
					entryBase = baseURL
				}
				entryName := strField(ef, "prefix", "name", "label")
				if entryName == "" {
					entryName = displayName
				}
				out = append(out, providerEntry{
					Name:     entryName,
					BaseURL:  entryBase,
					APIKey:   key,
					Disabled: disabled || boolField(ef, "disabled"),
				})
			}
			continue
		}

		// Flat api-keys array (codex/gemini/claude-api-key style).
		if apiKeys := fields["api-keys"]; apiKeys != nil && apiKeys.Kind == yaml.SequenceNode {
			for _, keyNode := range apiKeys.Content {
				key := strings.TrimSpace(keyNode.Value)
				if key == "" && keyNode.Kind == yaml.MappingNode {
					key = strField(mappingToMap(keyNode), "api-key", "api_key", "key")
				}
				if key == "" {
					continue
				}
				out = append(out, providerEntry{
					Name:     displayName,
					BaseURL:  baseURL,
					APIKey:   key,
					Disabled: disabled,
				})
			}
			continue
		}

		// codex-api-key uses flat `api-key:` string field.
		if key := strField(fields, "api-key", "api_key", "key"); key != "" {
			out = append(out, providerEntry{
				Name:     displayName,
				BaseURL:  baseURL,
				APIKey:   key,
				Disabled: disabled,
			})
		}
		_ = idx
	}
	return out
}

func strField(fields map[string]*yaml.Node, keys ...string) string {
	for _, k := range keys {
		if node, ok := fields[k]; ok && node != nil {
			v := strings.TrimSpace(node.Value)
			if v != "" {
				return v
			}
		}
	}
	return ""
}

func boolField(fields map[string]*yaml.Node, key string) bool {
	node, ok := fields[key]
	if !ok || node == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(node.Value), "true")
}

func dedupeProviders(in []providerEntry) []providerEntry {
	seen := make(map[string]bool, len(in))
	out := make([]providerEntry, 0, len(in))
	for _, p := range in {
		key := p.Name + "|" + p.BaseURL + "|" + maskKey(p.APIKey)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, p)
	}
	return out
}

// extractSectionFromBytes is a test helper that parses a YAML document and
// extracts one section. Kept here to keep production and test code in sync.
func extractSectionFromBytes(data []byte, sectionName string) ([]providerEntry, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return nil, nil
	}
	mapping := root.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil, nil
	}
	return extractSection(mappingToMap(mapping), sectionName, sectionName), nil
}
