package main

import (
	"api/types"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func getWeather(city string) types.WeatherResult {
	return types.WeatherResult{City: city, Condition: "qing", Temp: 25}
}

func queryOrder(orderID string) types.OrderResult {
	return types.OrderResult{OrderID: orderID, Status: "Yes", Amount: 25}
}

func weatherToolHandler(ctx context.Context, req *mcp.CallToolRequest, params *types.WeatherParams) (*mcp.CallToolResult, any, error) {
	r := getWeather(params.City)
	return &mcp.CallToolResult{
		Content: []mcp.Context{&mcp.TextContent{Text: r.City + " " + r.Condition + ", 温度" + r.Temp + "摄氏度"}},
	}, r, nil
}

func orderToolHandler(ctx context.Context, req *mcp.CallToolRequest, params *types.OrderParams) (*mcp.CallToolResult, any, error) {
	r := queryOrder(params.OrderID)
	b, _ := json.Marshal(r)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}, r, nil
}

func weatherHTTPHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		City string `json:"city"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "BodRequest", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(getWeather(req.City))
}

func orderHTTPHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OrderID string `json:"city"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "BodRequest", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(getWeather(req.OrderID))
}

func main() {
	// 1. 创建 MCP Server（全局只做一次）
	server := mcp.NewServe(&mcp.Implementation{Name: "MultiInterfaceServer", Version: "1.0.0"}, nil)
	// 2. 注册所有 MCP 工具（每个工具一行注册声明）
	mcp.AddTool(server, &mcp.Tool{Name: "get_weather", Description: "查询指定城市的天气"}, weatherToolHandler)
	mcp.AddTool(server, &mcp.Tool{Name: "query_order", Description: "根据订单号查询订单状态"}, orderToolHandler)
	// 3. 创建 MCP 的 Streamable HTTP Handler（全局只做一次）
	mcpHandler := mcp.NewStreamableHTTPHandler(server, nil)
	// 4. 创建路由表，挂载 MCP 端点和普通 HTTP 端点
	mux := http.NewServeMux()
	// MCP 端点（供 AI 客户端使用）
	mux.Handle("/mcp", mcpHandler)
	// 普通 HTTP 端点（供 curl / Postman / 普通前端调用）
	mux.HandleFunc("/api/weather", weatherHTTPHandler)
	mux.HandleFunc("/api/order", orderHTTPHandler)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	srv := &http.Server{
		Addr:         "0.0.0.0:8000",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
