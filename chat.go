package main

import (
	"encoding/json"
	"net/http"
)

type ContentBlock struct {
	Type     string    `json:"type"` // text/image_url/input_audio
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
	VideoURL *VideoURL `json:"video_url,omitempty"`
}

type VideoURL struct {
	URL string `json:"url"` // http(s) URL或者base64 data URI
}

type Function struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"` //JSON Schema对象
}

type ToolCall struct {
	ID       string       `json:"ID"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // OpenAI返回字符串化JSON
}

type ResponseFormat struct {
	Type string `json:"type"` // text/json_object/json_schema
}

type Tool struct {
	Type     string   `json:"type"` // function
	Function Function `json:"function"`
}

type Message struct {
	Role    string `json:"role"`    // system/user/assistant/tool
	Content any    `json:"content"` // string 或 []ContentBlock(多模态)
	Name    string `json:"name,omitempty"`
}

type ImageURL struct {
	URL    string `json:"url"`              // http(s) URL或者base64 data URI
	Detail string `json:"detail,omitempty"` // auto/low/high
}

type GatewayRequest struct {
	Provider string    `json:"provider"`
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Steam    bool      `json:"stream,omitempty"`
	//采样与生成控制
	Temperature      *float64 `json:"temperature,omitempty"`
	TopP             *float64 `json:"top_p,omitempty"`
	MaxTokens        *int     `json:"max_tokens,omitempty"`
	Stop             []string `json:"stop,omitempty"`
	PresencePenalty  *float64 `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`
	//工具调用
	Tools            []Tool `json:"tools,omitempty"`
	ToolChoice       any    `json:"tool_choice,omitempty"`
	ParalleToolCalls *bool  `json:"parallel_tool_calls,omitempty"`
	// 结构化输出(OpenAI的response_format)
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
	// 透传字段：各厂商特有参数放置位置，避免污染核心结构
	// 例如Qwen的result_format、top_k、repetition_penalty等
	Extra map[string]any `json:"-"`
}

func ChatHandler(w http.ResponseWriter, r *http.Request) {
	var req GatewayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if err := checkRequest(req); err != nil { //对用户请求参数进行统一校验
		return
	}



}

func checkRequest(p GatewayRequest) error {
	return nil
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
