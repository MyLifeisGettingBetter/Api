package provider

import (
	"api/types"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type OpenAI struct {
	APIKey string
}

type openAIRequest struct {
	Model         string              `json:"model"`
	Messages      []types.Message     `json:"messages"`
	Stream        bool                `json:"stream"`
	StreamOptions openAIStreamOptions `json:"stream_options"`
}

type openAIStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type openAIResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (o *OpenAI) BuildRequest(ctx context.Context, req types.ChatRequest) (*http.Request, error) {
	body := openAIRequest{
		Model:         req.Model,
		Messages:      req.Messages,
		Stream:        true,
		StreamOptions: openAIStreamOptions{IncludeUsage: true},
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openai: marshal request: %w", err)
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("openai: build request: %w", err)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+o.APIKey)
	return r, nil
}

func (o *OpenAI) ParseEvent(_ string, data string) (*types.Chunk, error) {
	if data == "[DONE]" {
		return &types.Chunk{Done: true}, nil
	}
	var resp openAIResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return nil, fmt.Errorf("openai: parse event: %w", err)
	}
	chunk := &types.Chunk{}
	if len(resp.Choices) > 0 {
		chunk.Content = resp.Choices[0].Delta.Content
	}

	if resp.Usage != nil {
		chunk.Usage = &types.Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		}
	}
	return chunk, nil
}
