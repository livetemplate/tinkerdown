package output

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSlackOutput_NewSlackOutput(t *testing.T) {
	// Test without SLACK_WEBHOOK_URL
	_, err := NewSlackOutput("#test")
	if err == nil {
		t.Fatal("expected error when SLACK_WEBHOOK_URL is not set")
	}

	// Test with invalid webhook URL (not a Slack URL)
	t.Setenv("SLACK_WEBHOOK_URL", "https://evil.com/webhook")
	_, err = NewSlackOutput("#test")
	if err == nil {
		t.Fatal("expected error when webhook URL is not a Slack URL")
	}

	// Test with empty channel
	t.Setenv("SLACK_WEBHOOK_URL", "https://hooks.slack.com/test")
	_, err = NewSlackOutput("")
	if err == nil {
		t.Fatal("expected error when channel is empty")
	}

	// Test successful creation
	output, err := NewSlackOutput("#test-channel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Channel() != "#test-channel" {
		t.Errorf("expected channel '#test-channel', got %q", output.Channel())
	}
}

func TestSlackOutput_NewSlackOutputWithURL(t *testing.T) {
	// Test with empty URL
	_, err := NewSlackOutputWithURL("#test", "")
	if err == nil {
		t.Fatal("expected error when URL is empty")
	}

	// Test with invalid webhook URL (not a Slack URL)
	_, err = NewSlackOutputWithURL("#test", "https://evil.com/webhook")
	if err == nil {
		t.Fatal("expected error when webhook URL is not a Slack URL")
	}

	// Test with empty channel
	_, err = NewSlackOutputWithURL("", "https://hooks.slack.com/test")
	if err == nil {
		t.Fatal("expected error when channel is empty")
	}

	// Test successful creation with valid Slack URL
	output, err := NewSlackOutputWithURL("#test-channel", "https://hooks.slack.com/services/T123/B456/abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Channel() != "#test-channel" {
		t.Errorf("expected channel '#test-channel', got %q", output.Channel())
	}
}

func TestSlackOutput_NewSlackOutputForTesting(t *testing.T) {
	// Test with empty URL
	_, err := NewSlackOutputForTesting("#test", "")
	if err == nil {
		t.Fatal("expected error when URL is empty")
	}

	// Test with empty channel
	_, err = NewSlackOutputForTesting("", "https://hooks.slack.com/test")
	if err == nil {
		t.Fatal("expected error when channel is empty")
	}

	// Test successful creation
	output, err := NewSlackOutputForTesting("#test-channel", "https://hooks.slack.com/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Channel() != "#test-channel" {
		t.Errorf("expected channel '#test-channel', got %q", output.Channel())
	}
}

func TestSlackOutput_Name(t *testing.T) {
	output, err := NewSlackOutputForTesting("#test", "https://hooks.slack.com/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Name() != "slack" {
		t.Errorf("expected name 'slack', got %q", output.Name())
	}
}

func TestSlackOutput_Send(t *testing.T) {
	var receivedPayload slackPayload

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	output, err := NewSlackOutputForTesting("#test-channel", server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	err = output.Send(ctx, "Hello from Tinkerdown!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedPayload.Channel != "#test-channel" {
		t.Errorf("expected channel '#test-channel', got %q", receivedPayload.Channel)
	}
	if receivedPayload.Text != "Hello from Tinkerdown!" {
		t.Errorf("expected text 'Hello from Tinkerdown!', got %q", receivedPayload.Text)
	}
}

func TestSlackOutput_Send_Error(t *testing.T) {
	// Create test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	output, err := NewSlackOutputForTesting("#test", server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx := context.Background()
	err = output.Send(ctx, "test")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestSlackOutput_Send_ContextCancellation(t *testing.T) {
	// Create test server that never responds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // Wait for context cancellation
	}))
	defer server.Close()

	output, err := NewSlackOutputForTesting("#test", server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = output.Send(ctx, "test")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestSlackOutput_Close(t *testing.T) {
	output, err := NewSlackOutputForTesting("#test", "https://hooks.slack.com/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = output.Close()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
