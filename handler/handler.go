package handler

import (
	"api/types"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type StreamFunc func(ctx context.Context, w http.ResponseWriter, provider types.Provider, req types.ChatRequest) error

func ChatHandler(providers map[string]types.Provider, streamFn StreamFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		if req.Provider == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "provider is required"})
			return
		}

		if req.Model == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "model is required"})
			return
		}

		if len(req.Messages) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "messages is required"})
		}

		p, ok := providers[req.Provider]
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("unknown provider: %s", req.Provider)})
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		if err := streamFn(r.Context(), w, p, req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		}
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
