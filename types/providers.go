package types

type ProviderConfig struct {
	Name    string   `toml:"name"`
	Type    string   `toml:"type"`
	BaseURL string   `toml:"base_url"`
	APIKey  []string `toml:"api_key"`
	Models  []string `toml:"models"`
}
