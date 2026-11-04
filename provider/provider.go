package provider

import (
	"fmt"
	"net/http"
	"os"

	"github.com/pelletier/go-toml/v2"
)

type Provider interface {
	Name() string
	Client() *http.Client
	BaseURL() string
	NextAPIKey() string //返回本次请求要使用的API Key
}

type ProviderConfig struct {
	Name    string   `toml:"name"`
	Type    string   `toml:"type"`
	BaseURL string   `toml:"base_url"`
	APIKey  []string `toml:"api_key"`
	Models  []string `toml:"models"`
}

type ProviderInfos struct {
	Providers []ProviderConfig `toml:"providers"`
}

func LoadProviders() (*Router, error) {
	var cfg ProviderInfos
	path := "./conf/providers.toml"
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	r := New()
	for _, pc := range cfg.Providers {
		var p Provider
		switch pc.Type {
		case "openai":
			p = NewOpenAI(pc.APIKey, pc.BaseURL)
		case "anthropic":
			p = NewAnthropic(pc.APIKey, pc.BaseURL)
		default:
			return nil, fmt.Errorf("unknown provider type %q", pc.Type)
		}
		for _, model := range pc.Models {
			r.Register(model, p)
		}
	}
	return r, nil
}
