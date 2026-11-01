package types

import (
	"context"
	"net/http"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Provider string    `json:"provider"`
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Usage struct {
	InputTokens  int
	OutputTokens int
}

type Chunk struct {
	Content string // Token text
	Done    bool   // signals the stream is over
	Err     error  // any mid-stream errors
	Usage   *Usage // only populated in the final chunk
}

type Metrics struct {
	TTFT  time.Duration // Time To First Token
	Total time.Duration // Time from request to stream close
}

type Provider interface {
	BuildRequest(ctx context.Context, req ChatRequest) (*http.Request, error)
	ParseEvent(eventType, data string) (*Chunk, error)
}
