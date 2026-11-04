package provider

import (
	"strings"
	"sync"
)

type Router struct {
	mu       sync.RWMutex
	exact    map[string]Provider
	prefixes []prefixRouter
	suffixes []suffixRouter
}

type prefixRouter struct {
	prefix   string
	provider Provider
}

type suffixRouter struct {
	suffix   string
	provider Provider
}

func New() *Router {
	return &Router{exact: make(map[string]Provider)}
}

func (r *Router) Register(model string, p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !strings.Contains(model, "*") {
		r.exact[model] = p
	} else if strings.HasPrefix(model, "*") {
		r.prefixes = append(r.prefixes, prefixRouter{prefix: strings.TrimSuffix(model, "*"), provider: p})
	} else if strings.HasSuffix(model, "*") {
		r.suffixes = append(r.suffixes, suffixRouter{suffix: strings.TrimPrefix(model, "*"), provider: p})
	}
}

func (r *Router) Find(model string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if p, ok := r.exact[model]; ok {
		return p, true
	}

	for _, pr := range r.prefixes {
		if strings.HasPrefix(model, pr.prefix) {
			return pr.provider, true
		}
	}

	for _, su := range r.suffixes {
		if strings.HasSuffix(model, su.suffix) {
			return su.provider, true
		}
	}
	return nil, false
}
