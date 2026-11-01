package stream

import (
	"api/types"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{
	Transport: &http.Transport{
		DialContext: (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
	},
}

type wireUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type wireMetrics struct {
	TTFTMs  int64 `json:"ttft_ms"`
	TotalMs int64 `json:"total_ms"`
}

type wireChunk struct {
	Content string       `json:"content"`
	Done    bool         `json:"done"`
	Error   string       `json:"error,omitempty"`
	Usage   *wireUsage   `json:"usage,omitempty"`
	Metrics *wireMetrics `json:"metrics,omitempty"`
}

type idleTimeoutReader struct {
	r       io.ReadCloser
	timeout time.Duration
	timer   *time.Timer
}

func writeChunk(w http.ResponseWriter, f http.Flusher, c wireChunk) {
	b, _ := json.Marshal(c)
	fmt.Fprintf(w, "data:%s\n\n", b)
	f.Flush()
}

func writeErrorChunk(w http.ResponseWriter, f http.Flusher, msg string) {
	writeChunk(w, f, wireChunk{Done: true, Error: msg})
}

func (r *idleTimeoutReader) Read(p []byte) (int, error) {
	if r.timer == nil {
		r.timer = time.AfterFunc(r.timeout, func() {
			r.r.Close()
		})
	} else {
		r.timer.Reset(r.timeout)
	}
	return r.r.Read(p)
}

// Stream executes the full lifecycle of a single proxied request:
// build upstream request → read SSE → normalize → flush to client.
func Stream(ctx context.Context, w http.ResponseWriter, provider types.Provider, req types.ChatRequest) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported: ResponseWriter does not implement http.Flusher")
	}

	upstreamReq, err := provider.BuildRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	resp, err := httpClient.Do(upstreamReq)
	if err != nil {
		return fmt.Errorf("upstream request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upstream %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	start := time.Now()
	var ttft time.Duration
	var accUsage types.Usage
	body := &idleTimeoutReader{r: resp.Body, timeout: 30 * time.Second}
	scanner := bufio.NewScanner(body)
	var eventType string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			eventType = ""
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		chunk, err := provider.ParseEvent(eventType, data)
		if err != nil {
			writeErrorChunk(w, flusher, err.Error())
			return nil
		}
		if chunk == nil {
			continue
		}

		if chunk.Usage != nil {
			if chunk.Usage.InputTokens > 0 {
				accUsage.InputTokens = chunk.Usage.InputTokens
			}
			if chunk.Usage.OutputTokens > 0 {
				accUsage.OutputTokens = chunk.Usage.OutputTokens
			}
		}
		if chunk.Err != nil {
			writeErrorChunk(w, flusher, chunk.Err.Error())
			return nil
		}
		if ttft == 0 && chunk.Content != "" {
			ttft = time.Since(start)
		}
		if chunk.Done {
			total := time.Since(start)
			slog.Info("stream complete",
				"provider", req.Provider,
				"model", req.Model,
				"ttfs_ms", ttft.Milliseconds(),
				"total_ms", total.Milliseconds(),
				"input_tokens", accUsage.InputTokens,
				"output_tokens", accUsage.OutputTokens)
			writeChunk(w, flusher, wireChunk{
				Done:    true,
				Usage:   &wireUsage{InputTokens: accUsage.InputTokens, OutputTokens: accUsage.OutputTokens},
				Metrics: &wireMetrics{TTFTMs: ttft.Milliseconds(), TotalMs: total.Milliseconds()},
			})
			return nil
		}

		if chunk.Content == "" {
			continue
		}
		writeChunk(w, flusher, wireChunk{Content: chunk.Content})
	}
	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			slog.Info("client disconnected", "provider", req.Provider, "model", req.Model)
			return nil
		}
		writeErrorChunk(w, flusher, fmt.Sprintf("stream read error: %s", err.Error()))
		return nil
	}
	writeErrorChunk(w, flusher, "stream ended without completion signal")
	return nil
}
