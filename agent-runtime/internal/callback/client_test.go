package callback

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestSendLog(t *testing.T) {
	var received logPayload
	var authHeader string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "test-token", "run-1", "step-1")
	err := c.SendLog(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("SendLog returned error: %v", err)
	}

	if authHeader != "Bearer test-token" {
		t.Errorf("expected auth header %q, got %q", "Bearer test-token", authHeader)
	}
	if len(received.Lines) != 1 || received.Lines[0] != "hello world" {
		t.Errorf("expected lines %v, got %v", []string{"hello world"}, received.Lines)
	}
}

func TestSendCost(t *testing.T) {
	var received costPayload
	var reqPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "tok", "run-42", "step-7")
	err := c.SendCost(context.Background(), 1000, 500, "claude-sonnet-4-6", 0.0105)
	if err != nil {
		t.Fatalf("SendCost returned error: %v", err)
	}

	expectedPath := "/internal/agent/callback/runs/run-42/steps/step-7/cost"
	if reqPath != expectedPath {
		t.Errorf("expected path %q, got %q", expectedPath, reqPath)
	}
	if received.InputTokens != 1000 {
		t.Errorf("expected input_tokens 1000, got %d", received.InputTokens)
	}
	if received.OutputTokens != 500 {
		t.Errorf("expected output_tokens 500, got %d", received.OutputTokens)
	}
	if received.Model != "claude-sonnet-4-6" {
		t.Errorf("expected model %q, got %q", "claude-sonnet-4-6", received.Model)
	}
	if received.CostUSD != 0.0105 {
		t.Errorf("expected cost_usd 0.0105, got %f", received.CostUSD)
	}
}

func TestSendStatus(t *testing.T) {
	var received statusPayload
	var contentType string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "tok", "run-1", "step-1")
	err := c.SendStatus(context.Background(), 1, "process crashed")
	if err != nil {
		t.Fatalf("SendStatus returned error: %v", err)
	}

	if contentType != "application/json" {
		t.Errorf("expected content-type %q, got %q", "application/json", contentType)
	}
	if received.ExitCode != 1 {
		t.Errorf("expected exit_code 1, got %d", received.ExitCode)
	}
	if received.Error != "process crashed" {
		t.Errorf("expected error %q, got %q", "process crashed", received.Error)
	}
}

func TestRetryOn500(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(srv.URL, "tok", "run-1", "step-1")
	err := c.SendLog(context.Background(), "retrying")
	if err != nil {
		t.Fatalf("SendLog should have succeeded after retries: %v", err)
	}

	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestNoRetryOn400(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	c := New(srv.URL, "tok", "run-1", "step-1")
	err := c.SendLog(context.Background(), "bad request")
	if err == nil {
		t.Fatal("SendLog should have returned an error for 400")
	}

	if attempts.Load() != 1 {
		t.Errorf("expected 1 attempt (no retry on 400), got %d", attempts.Load())
	}
}

func TestRetryExhausted(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(srv.URL, "tok", "run-1", "step-1")
	err := c.SendLog(context.Background(), "always fails")
	if err == nil {
		t.Fatal("SendLog should have returned an error after exhausting retries")
	}

	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts.Load())
	}
}
