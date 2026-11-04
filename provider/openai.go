package provider

import (
	"net/http"
	"sync/atomic"
)

type OpenAIProvider struct {
	apiKeys []string
	baseURL string
	client  *http.Client
	counter atomic.Uint64
}

func NewOpenAI(apiKeys []string, baseURL string) *OpenAIProvider {
	return &OpenAIProvider{apiKeys: apiKeys, baseURL: baseURL, client: &http.Client{}}
}

func (p *OpenAIProvider) Name() string         { return "openai" }
func (p *OpenAIProvider) Client() *http.Client { return p.client }
func (p *OpenAIProvider) BaseURL() string      { return p.baseURL }
func (p *OpenAIProvider) NextAPIKey() string {
	if len(p.apiKeys) == 0 {
		return ""
	}
	n := p.counter.Add(1) - 1
	return p.apiKeys[n%uint64(len(p.apiKeys))]
}
