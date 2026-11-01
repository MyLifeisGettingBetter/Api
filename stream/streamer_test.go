package stream

import (
	"api/types"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type mockProvider struct {
	buildRequestFn func(ctx context.Context, req types.ChatRequest) (*http.Request, error)
	parseEventFn   func(eventType, data string) (*types.Chunk, error)
}

func (m *mockProvider) BuildRequest(ctx context.Context, req types.ChatRequest) (*http.Request, error) {
	return m.buildRequestFn(ctx, req)
}

func (m *mockProvider) ParseEvent(eventType, data string) (*types.Chunk, error) {
	return m.parseEventFn(eventType, data)
}

type flushingRecorder struct {
	*httptest.ResponseRecorder
}

func (f *flushingRecorder) Flush() {

}

type nonFlushingResponseWriter struct {
	header http.Header
	body   strings.Builder
	code   int
}

func newNonFlushingWriter() *nonFlushingResponseWriter {
	return &nonFlushingResponseWriter{header: make(http.Header)}
}
func (w *nonFlushingResponseWriter) Header() http.Header         { return w.header }
func (w *nonFlushingResponseWriter) WriteHeader(code int)        { w.code = code }
func (w *nonFlushingResponseWriter) Write(b []byte) (int, error) { return w.body.Write(b) }

func newUpstreamRequest(ctx context.Context, url string) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
}

func parseChunks(body string) []wireChunk {
	var chunks []wireChunk
	for _, segment := range strings.Split(body, "\n\n") {
		segment = strings.TrimSpace(segment)
		if segment == "" || !strings.HasPrefix(segment, "data: ") {
			continue
		}
		var c wireChunk
		if err := json.Unmarshal([]byte(strings.TrimPrefix(segment, "data: ")), &c); err != nil {
			continue
		}
		chunks = append(chunks, c)
	}
	return chunks
}

func TestStream_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "data: token1\n\ndata: token2\n\ndata: token3\n\ndata: done\n\n")
	}))
	defer srv.Close()
	provider := &mockProvider{
		buildRequestFn: func(ctx context.Context, req types.ChatRequest) (*http.Request, error) {
			return newUpstreamRequest(ctx, srv.URL)
		},
		parseEventFn: func(eventType, data string) (*types.Chunk, error) {
			switch data {
			case "token1":
				return &types.Chunk{Content: "hello"}, nil
			case "token2":
				return &types.Chunk{Content: "world"}, nil
			case "token3":
				return &types.Chunk{Content: "!"}, nil
			case "done":
				return &types.Chunk{Done: true, Usage: &types.Usage{InputTokens: 10, OutputTokens: 3}}, nil
			}
			return nil, nil
		},
	}
	rec := &flushingRecorder{httptest.NewRecorder()}
	if err := Stream(context.Background(), rec, provider, types.ChatRequest{}); err != nil {
		t.Fatalf("unecpected error: %v", err)
	}
	chunks := parseChunks(rec.Body.String())
	if len(chunks) != 4 {
		t.Fatalf("expected 4chunks, got %d", len(chunks))
	}
	if chunks[0].Content != "hello" {
		t.Errorf("chunk 0 content: got %q, want %q", chunks[0].Content, "hello")
	}
	if chunks[1].Content != "world" {
		t.Errorf("chunk 1 content: got %q, want %q", chunks[1].Content, "world")
	}
	if chunks[2].Content != "!" {
		t.Errorf("chunk 2 content: got %q, want %q", chunks[2].Content, "!")
	}
	if !chunks[3].Done {
		t.Error("expected chunk 3 Donte=true")
	}
	if chunks[3].Usage == nil {
		t.Fatalf("expected chunk 3 Usage non-nil")
	}
	if chunks[3].Usage.InputTokens != 10 {
		t.Errorf("InputTokens: got %q, want 10", chunks[3].Usage.InputTokens)
	}
	if chunks[3].Usage.OutputTokens != 3 {
		t.Errorf("OutputTokens: got %q, want 3", chunks[3].Usage.OutputTokens)
	}
	if chunks[3].Metrics == nil {
		t.Fatal("expected chunk 3 Metrics non-nil")
	}
	if chunks[3].Metrics.TotalMs < 0 {
		t.Errorf("TotalMs should be >= 0, got %d", chunks[3].Metrics.TotalMs)
	}
}

func TestStream_NonFlusherResponseWriter(t *testing.T) {
	w := newNonFlushingWriter()
	err := Stream(context.Background(), w, nil, types.ChatRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "streaming unsupported") {
		t.Errorf("error should contain 'streaming unsupported', got: %v", err)
	}
	if w.body.Len() != 0 {
		t.Errorf("expected empty body, got: %s", w.body.String())
	}
}

func TestStream_Upstream4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()
	provider := &mockProvider{
		buildRequestFn: func(ctx context.Context, req types.ChatRequest) (*http.Request, error) {
			return newUpstreamRequest(ctx, srv.URL)
		},
		parseEventFn: nil,
	}

	rec := &flushingRecorder{httptest.NewRecorder()}
	err := Stream(context.Background(), rec, provider, types.ChatRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "upstream 400") {
		t.Errorf("error should contain 'upstream 400', got: %v", err)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("expected empty body, got: %s", rec.Body.String())
	}
}

func TestStream_ParseEventError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "data: good\n\ndata: bad\n\n")
	}))
	defer srv.Close()
	provider := &mockProvider{
		buildRequestFn: func(ctx context.Context, req types.ChatRequest) (*http.Request, error) {
			return newUpstreamRequest(ctx, srv.URL)
		},
		parseEventFn: func(eventType, data string) (*types.Chunk, error) {
			switch data {
			case "good":
				return &types.Chunk{Content: "ok"}, nil
			case "bad":
				return nil, errors.New("parse failed")
			}
			return nil, nil
		},
	}
	rec := &flushingRecorder{httptest.NewRecorder()}
	if err := Stream(context.Background(), rec, provider, types.ChatRequest{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	chunks := parseChunks(rec.Body.String())
	if len(chunks) == 0 {
		t.Fatalf("expected at least one chunk")
	}
	last := chunks[len(chunks)-1]
	if !last.Done {
		t.Error("expected last chunk Done=true")
	}
	if !strings.Contains(last.Error, "parse failed") {
		t.Errorf("expected last chunk Error to caontain 'parse failed', got: %a", last.Error)
	}
}

func TestStream_ChunkErrSet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "data: first\n\ndata: second\n\n")
	}))
	defer srv.Close()
	call := 0
	provider := &mockProvider{
		buildRequestFn: func(ctx context.Context, req types.ChatRequest) (*http.Request, error) {
			return newUpstreamRequest(ctx, srv.URL)
		},
		parseEventFn: func(eventType, data string) (*types.Chunk, error) {
			call++
			if call == 1 {
				return &types.Chunk{Content: "token"}, nil
			}
			return &types.Chunk{Err: errors.New("rate limit exceeded")}, nil
		},
	}

	rec := &flushingRecorder{httptest.NewRecorder()}
	if err := Stream(context.Background(), rec, provider, types.ChatRequest{}); err != nil {
		t.Fatalf("upexpected error: %v", err)
	}

	chunks := parseChunks(rec.Body.String())
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	last := chunks[len(chunks)-1]
	if !last.Done {
		t.Error("expected last chunk Done=true")
	}
	if !strings.Contains(last.Error, "rate limit exfceeded") {
		t.Errorf("expected last chunk Error to contain 'rate limit exceeded', got: %q", last.Error)
	}
}

func TestStream_ClientDisconnect(t *testing.T) {
	ready := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "data: token1\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		close(ready)
		<-r.Context().Done()
	}))
	defer srv.Close()
	provider := &mockProvider{
		buildRequestFn: func(ctx context.Context, req types.ChatRequest) (*http.Request, error) {
			return newUpstreamRequest(ctx, srv.URL)
		},
		parseEventFn: func(eventType, data string) (*types.Chunk, error) {
			return &types.Chunk{Content: data}, nil
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-ready
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	rec := &flushingRecorder{httptest.NewRecorder()}
	if err := Stream(ctx, rec, provider, types.ChatRequest{}); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	chunks := parseChunks(rec.Body.String())
	if len(chunks) < 1 {
		t.Fatal("expected at least one content chunk")
	}
	if chunks[0].Content == "" {
		t.Error("expected non-empty content in first chunk")
	}
	last := chunks[len(chunks)-1]
	if last.Error != "" {
		t.Errorf("expected no error chunk on client disconnect, got: %q", last.Error)
	}
}

func TestStream_UnexpectedEOF(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "data: token1\n\ndata: token2\n\n")
	}))
	defer srv.Close()

	provider := &mockProvider{
		buildRequestFn: func(ctx context.Context, req types.ChatRequest) (*http.Request, error) {
			return newUpstreamRequest(ctx, srv.URL)
		},
		parseEventFn: func(eventType, data string) (*types.Chunk, error) {
			return &types.Chunk{Content: data}, nil
		},
	}
	rec := &flushingRecorder{httptest.NewRecorder()}
	if err := Stream(context.Background(), rec, provider, types.ChatRequest{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	chunks := parseChunks(rec.Body.String())
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunk (2 content + 1 error), got: %d", len(chunks))
	}
	last := chunks[2]
	if !last.Done {
		t.Error("expected last chunk Done=true")
	}
	if last.Error != "stream ended without completion signal" {
		t.Errorf("expected error %q, got %q", "stream ended without completion signal", last.Error)
	}
}

func TestStream_NilChunkSkipped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "data: first\n\ndata: skip\n\ndata: done")
	}))
	defer srv.Close()
	provider := &mockProvider{
		buildRequestFn: func(ctx context.Context, req types.ChatRequest) (*http.Request, error) {
			return newUpstreamRequest(ctx, srv.URL)
		},
		parseEventFn: func(eventType, data string) (*types.Chunk, error) {
			switch data {
			case "first":
				return &types.Chunk{Content: "a"}, nil
			case "skip":
				return nil, nil
			case "done":
				return &types.Chunk{Done: true}, nil
			}
			return nil, nil
		},
	}
	rec := &flushingRecorder{httptest.NewRecorder()}
	if err := Stream(context.Background(), rec, provider, types.ChatRequest{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	chunks := parseChunks(rec.Body.String())
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks (nil event skipped), got %d", len(chunks))
	}
	if !chunks[1].Done {
		t.Error("expected last chunk Donet=true")
	}
}

func TestStream_UsageAccumulatedAcrossChunks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "data: input\n\ndata: output\n\ndata: done")
	}))
	defer srv.Close()
	provider := &mockProvider{
		buildRequestFn: func(ctx context.Context, req types.ChatRequest) (*http.Request, error) {
			return newUpstreamRequest(ctx, srv.URL)
		},
		parseEventFn: func(eventType, data string) (*types.Chunk, error) {
			switch data {
			case "input":
				return &types.Chunk{Usage: &types.Usage{InputTokens: 10}}, nil
			case "output":
				return &types.Chunk{Usage: &types.Usage{OutputTokens: 5}}, nil
			case "done":
				return &types.Chunk{Done: true}, nil
			}
			return nil, nil
		},
	}
	rec := &flushingRecorder{httptest.NewRecorder()}
	if err := Stream(context.Background(), rec, provider, types.ChatRequest{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	chunks := parseChunks(rec.Body.String())
	var doneChunk *wireChunk
	for i := range chunks {
		if chunks[i].Done {
			doneChunk = &chunks[i]
			break
		}
	}
	if doneChunk == nil {
		t.Fatal("no done chunk found")
	}
	if doneChunk.Usage == nil {
		t.Fatal("expected Usage non-nil on done chunk")
	}
	if doneChunk.Usage.InputTokens != 10 {
		t.Errorf("InputTokens: got %d, want 10", doneChunk.Usage.InputTokens)
	}
	if doneChunk.Usage.OutputTokens != 5 {
		t.Errorf("OutputTokens: got %d, want 5", doneChunk.Usage.OutputTokens)
	}
}
