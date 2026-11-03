package main

import (
	"api/types"
	"net/http"
)

func handlerHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handlerHealth)
	mux.HandleFunc("POST /chat", types.ChatHandler) // 注册
}
