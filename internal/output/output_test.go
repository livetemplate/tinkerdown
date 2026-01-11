package output

import (
	"context"
	"errors"
	"testing"
)

// mockOutput is a test implementation of Output
type mockOutput struct {
	name     string
	messages []string
	closed   bool
	sendErr  error
}

func (m *mockOutput) Name() string {
	return m.name
}

func (m *mockOutput) Send(ctx context.Context, message string) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.messages = append(m.messages, message)
	return nil
}

func (m *mockOutput) Close() error {
	m.closed = true
	return nil
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()
	mock := &mockOutput{name: "test"}

	r.Register("test", mock)

	got, ok := r.Get("test")
	if !ok {
		t.Fatal("expected output to be found")
	}
	if got != mock {
		t.Error("expected same output instance")
	}
}

func TestRegistry_Get_NotFound(t *testing.T) {
	r := NewRegistry()

	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("expected output not to be found")
	}
}

func TestRegistry_SendAll(t *testing.T) {
	r := NewRegistry()
	mock1 := &mockOutput{name: "mock1"}
	mock2 := &mockOutput{name: "mock2"}

	r.Register("mock1", mock1)
	r.Register("mock2", mock2)

	ctx := context.Background()
	err := r.SendAll(ctx, "test message")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock1.messages) != 1 || mock1.messages[0] != "test message" {
		t.Errorf("mock1 did not receive message: %v", mock1.messages)
	}
	if len(mock2.messages) != 1 || mock2.messages[0] != "test message" {
		t.Errorf("mock2 did not receive message: %v", mock2.messages)
	}
}

func TestRegistry_SendAll_WithErrors(t *testing.T) {
	r := NewRegistry()
	mock1 := &mockOutput{name: "mock1"}
	mock2 := &mockOutput{name: "mock2", sendErr: errors.New("send failed")}

	r.Register("mock1", mock1)
	r.Register("mock2", mock2)

	ctx := context.Background()
	err := r.SendAll(ctx, "test message")
	if err == nil {
		t.Fatal("expected error")
	}

	// mock1 should still have received the message
	if len(mock1.messages) != 1 {
		t.Errorf("mock1 should have received message: %v", mock1.messages)
	}
}

func TestRegistry_Close(t *testing.T) {
	r := NewRegistry()
	mock1 := &mockOutput{name: "mock1"}
	mock2 := &mockOutput{name: "mock2"}

	r.Register("mock1", mock1)
	r.Register("mock2", mock2)

	err := r.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !mock1.closed {
		t.Error("mock1 should be closed")
	}
	if !mock2.closed {
		t.Error("mock2 should be closed")
	}
}

func TestNewFromConfig_Slack(t *testing.T) {
	// Set required env var
	t.Setenv("SLACK_WEBHOOK_URL", "https://hooks.slack.com/test")

	cfg := Config{
		Type:    "slack",
		Channel: "#test-channel",
	}

	output, err := NewFromConfig("test-slack", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	slackOutput, ok := output.(*SlackOutput)
	if !ok {
		t.Fatal("expected SlackOutput type")
	}

	if slackOutput.Name() != "slack" {
		t.Errorf("expected name 'slack', got %q", slackOutput.Name())
	}
	if slackOutput.Channel() != "#test-channel" {
		t.Errorf("expected channel '#test-channel', got %q", slackOutput.Channel())
	}
}

func TestNewFromConfig_Email(t *testing.T) {
	// Set required env vars
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_FROM", "noreply@example.com")

	cfg := Config{
		Type:    "email",
		To:      "user@example.com",
		Subject: "Test Alert",
	}

	output, err := NewFromConfig("test-email", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	emailOutput, ok := output.(*EmailOutput)
	if !ok {
		t.Fatal("expected EmailOutput type")
	}

	if emailOutput.Name() != "email" {
		t.Errorf("expected name 'email', got %q", emailOutput.Name())
	}
	if emailOutput.To() != "user@example.com" {
		t.Errorf("expected to 'user@example.com', got %q", emailOutput.To())
	}
	if emailOutput.Subject() != "Test Alert" {
		t.Errorf("expected subject 'Test Alert', got %q", emailOutput.Subject())
	}
}

func TestNewFromConfig_UnsupportedType(t *testing.T) {
	cfg := Config{
		Type: "unsupported",
	}

	_, err := NewFromConfig("test", cfg)
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}
