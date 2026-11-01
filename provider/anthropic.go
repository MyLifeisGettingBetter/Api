package provider

import (
	"api/types"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Anthropic struct {
	APIKey string
}

type anthropicRequest struct {
	Model     string          `json:"model"`
	Messages  []types.Message `json:"messages"`
	MaxTokens int             `json:"max_tokens"`
	Stream    bool            `json:"stream"`
}

type anthropicContentBlockDelta struct {
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
}

type anthropicMessageStart struct {
	Message struct {
		Usage struct {
			InputTokens int `json:"input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

type anthropicMessageDelta struct {
	Usage struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type anthropicError struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (a *Anthropic) BuildRequest(ctx context.Context, req types.ChatRequest) (*http.Request, error) {
	body := anthropicRequest{
		Model:     req.Model,
		Messages:  req.Messages,
		MaxTokens: 4096,
		Stream:    true,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("anthropic: marshal request: %w", err)
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("anthropic: build request: %w", err)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("x-api-key", a.APIKey)
	r.Header.Set("anthropic-version", "2023-06-01")
	return r, nil
}

func (a *Anthropic) ParseEvent(eventType, data string) (*types.Chunk, error) {
	switch eventType {
	case "content_block_delta":
		var payload anthropicContentBlockDelta
		if err := json.Unmarshal([]byte(data), &payload); err != nil {
			return nil, fmt.Errorf("anthropic: parse content_block_delta: %w", err)
		}
		return &types.Chunk{Content: payload.Delta.Text}, nil
	case "message_start":
		var payload anthropicMessageStart
		if err := json.Unmarshal([]byte(data), &payload); err != nil {
			return nil, fmt.Errorf("anthropic: parse message_start: %w", err)
		}
		return &types.Chunk{
			Usage: &types.Usage{
				InputTokens: payload.Message.Usage.InputTokens,
			},
		}, nil
	case "message_delta":
		var payload anthropicMessageDelta
		if err := json.Unmarshal([]byte(data), &payload); err != nil {
			return nil, fmt.Errorf("anthropic: parse message_delta: %w", err)
		}
		return &types.Chunk{Usage: &types.Usage{OutputTokens: payload.Usage.OutputTokens}}, nil
	case "message_stop":
		return &types.Chunk{Done: true}, nil
	case "error":
		var payload anthropicError
		if err := json.Unmarshal([]byte(data), &payload); err != nil {
			return nil, fmt.Errorf("anthropic: parse error event: %w", err)
		}
		return &types.Chunk{Err: fmt.Errorf("anthropic: %s", payload.Error.Message)}, nil
	default:
		return nil, nil
	}
}
