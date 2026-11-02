package main

import (
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", HandlerHealth)
	mux.HandleFunc("POST /chat", ChatHandler) // 注册
}
