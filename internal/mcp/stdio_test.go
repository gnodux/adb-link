package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

// blockingReader is a reader that never returns any data, simulating
// a stdin that nobody writes to. It blocks until its internal context
// is done.
type blockingReader struct {
	ctx context.Context
}

func (r *blockingReader) Read(p []byte) (int, error) {
	<-r.ctx.Done()
	return 0, r.ctx.Err()
}

// TestServeStdio_ContextCancel tests that cancelling the context causes
// ServeStdio to return promptly even when the reader is blocked and no
// data is forthcoming (the core fix for the Ctrl+C hang).
func TestServeStdio_ContextCancel(t *testing.T) {
	srv := NewServer("test", "1.0.0")

	ctx, cancel := context.WithCancel(context.Background())
	in := &blockingReader{ctx: ctx}
	out := &bytes.Buffer{}

	done := make(chan error, 1)
	go func() {
		done <- srv.ServeStdio(ctx, in, out)
	}()

	// Give the goroutine time to enter the blocking read.
	time.Sleep(50 * time.Millisecond)

	// Cancel the context — this simulates Ctrl+C.
	cancel()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ServeStdio did not return within 2s after context cancellation")
	}
}

// TestServeStdio_EOF tests that ServeStdio returns nil when the reader
// reaches EOF.
func TestServeStdio_EOF(t *testing.T) {
	srv := NewServer("test", "1.0.0")

	in := strings.NewReader("")
	out := &bytes.Buffer{}

	err := srv.ServeStdio(context.Background(), in, out)
	if err != nil {
		t.Fatalf("expected nil error on EOF, got %v", err)
	}
}

// TestServeStdio_RequestResponse tests that a valid JSON-RPC request
// on the reader produces a proper response on the writer.
func TestServeStdio_RequestResponse(t *testing.T) {
	srv := NewServer("test", "1.0.0")

	req := Request{JSONRPC: "2.0", Method: "initialize"}
	req.ID, _ = json.Marshal(1)
	reqBody, _ := json.Marshal(req)
	in := strings.NewReader(string(reqBody) + "\n")
	out := &bytes.Buffer{}

	// After the response is written, the reader will hit EOF.
	err := srv.ServeStdio(context.Background(), in, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse the response from the output buffer.
	var resp Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected RPC error: %s", resp.Error.Message)
	}
}

// TestServeStdio_InvalidJSON tests that an invalid JSON line produces
// a parse error response and the loop continues until EOF.
func TestServeStdio_InvalidJSON(t *testing.T) {
	srv := NewServer("test", "1.0.0")

	in := strings.NewReader("not-json\n")
	out := &bytes.Buffer{}

	err := srv.ServeStdio(context.Background(), in, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected parse error response, got nil")
	}
	if resp.Error.Code != ErrCodeParse {
		t.Fatalf("expected error code %d, got %d", ErrCodeParse, resp.Error.Code)
	}
}

// TestServeStdio_ContextCancelDuringProcessing tests that cancelling the
// context while the server is idle (between requests) also returns promptly.
func TestServeStdio_ContextCancelDuringProcessing(t *testing.T) {
	srv := NewServer("test", "1.0.0")

	// Use a pipe so we can keep the reader open without sending data.
	pr, pw := io.Pipe()
	defer pw.Close()

	ctx, cancel := context.WithCancel(context.Background())
	out := &bytes.Buffer{}

	done := make(chan error, 1)
	go func() {
		done <- srv.ServeStdio(ctx, pr, out)
	}()

	// Let the goroutine enter the blocking read.
	time.Sleep(50 * time.Millisecond)

	cancel()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ServeStdio did not return within 2s after context cancellation")
	}
}
