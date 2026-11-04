package provider

import (
	"net/http"
	"sync/atomic"
)

type AnthropicProvider struct {
	apiKeys []string
	baseURL string
	client  *http.Client
	counter atomic.Uint64
}

func NewAnthropic(apiKeys []string, baseURL string) *AnthropicProvider {
	return &AnthropicProvider{apiKeys: apiKeys, baseURL: baseURL, client: &http.Client{}}
}

func (p *AnthropicProvider) Name() string         { return "anthropic" }
func (p *AnthropicProvider) Client() *http.Client { return p.client }
func (p *AnthropicProvider) BaseURL() string      { return p.baseURL }
func (p *AnthropicProvider) NextAPIKey() string {
	if len(p.apiKeys) == 0 {
		return ""
	}
	n := p.counter.Add(1) - 1
	return p.apiKeys[n%uint64(len(p.apiKeys))]
}
