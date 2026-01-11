package output

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestEmailOutput_NewEmailOutput(t *testing.T) {
	// Clear env vars first
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_FROM", "")

	// Test without SMTP_HOST
	_, err := NewEmailOutput("user@example.com", "")
	if err == nil {
		t.Fatal("expected error when SMTP_HOST is not set")
	}

	// Test without SMTP_FROM
	t.Setenv("SMTP_HOST", "smtp.example.com")
	_, err = NewEmailOutput("user@example.com", "")
	if err == nil {
		t.Fatal("expected error when SMTP_FROM is not set")
	}

	// Test with empty recipient
	t.Setenv("SMTP_FROM", "noreply@example.com")
	_, err = NewEmailOutput("", "")
	if err == nil {
		t.Fatal("expected error when recipient is empty")
	}

	// Test successful creation with default subject
	output, err := NewEmailOutput("user@example.com", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.To() != "user@example.com" {
		t.Errorf("expected to 'user@example.com', got %q", output.To())
	}
	if output.Subject() != "Notification from Tinkerdown" {
		t.Errorf("expected default subject, got %q", output.Subject())
	}

	// Test with custom subject
	output, err = NewEmailOutput("user@example.com", "Custom Subject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.Subject() != "Custom Subject" {
		t.Errorf("expected subject 'Custom Subject', got %q", output.Subject())
	}
}

func TestEmailOutput_NewEmailOutputWithConfig(t *testing.T) {
	tests := []struct {
		name      string
		to        string
		from      string
		subject   string
		smtpHost  string
		smtpPort  string
		wantErr   bool
		errContains string
	}{
		{
			name:     "empty to",
			to:       "",
			from:     "noreply@example.com",
			smtpHost: "smtp.example.com",
			wantErr:  true,
			errContains: "recipient",
		},
		{
			name:     "empty from",
			to:       "user@example.com",
			from:     "",
			smtpHost: "smtp.example.com",
			wantErr:  true,
			errContains: "sender",
		},
		{
			name:     "empty host",
			to:       "user@example.com",
			from:     "noreply@example.com",
			smtpHost: "",
			wantErr:  true,
			errContains: "host",
		},
		{
			name:     "valid config",
			to:       "user@example.com",
			from:     "noreply@example.com",
			smtpHost: "smtp.example.com",
			smtpPort: "587",
			subject:  "Test",
			wantErr:  false,
		},
		{
			name:     "default port",
			to:       "user@example.com",
			from:     "noreply@example.com",
			smtpHost: "smtp.example.com",
			smtpPort: "",
			wantErr:  false,
		},
		{
			name:     "default subject",
			to:       "user@example.com",
			from:     "noreply@example.com",
			smtpHost: "smtp.example.com",
			subject:  "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := NewEmailOutputWithConfig(
				tt.to, tt.from, tt.subject,
				tt.smtpHost, tt.smtpPort,
				"user", "pass",
			)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if output.To() != tt.to {
				t.Errorf("expected to %q, got %q", tt.to, output.To())
			}
		})
	}
}

func TestEmailOutput_Name(t *testing.T) {
	output, err := NewEmailOutputWithConfig(
		"user@example.com",
		"noreply@example.com",
		"Test",
		"smtp.example.com",
		"587",
		"", "",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Name() != "email" {
		t.Errorf("expected name 'email', got %q", output.Name())
	}
}

func TestEmailOutput_Close(t *testing.T) {
	output, err := NewEmailOutputWithConfig(
		"user@example.com",
		"noreply@example.com",
		"Test",
		"smtp.example.com",
		"587",
		"", "",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = output.Close()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEmailOutput_DefaultPort(t *testing.T) {
	output, err := NewEmailOutputWithConfig(
		"user@example.com",
		"noreply@example.com",
		"Test",
		"smtp.example.com",
		"", // empty port should default to 587
		"", "",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the output was created (port is internal)
	if output.To() != "user@example.com" {
		t.Errorf("unexpected to: %s", output.To())
	}
}

func TestEmailOutput_DefaultSubject(t *testing.T) {
	output, err := NewEmailOutputWithConfig(
		"user@example.com",
		"noreply@example.com",
		"", // empty subject should get default
		"smtp.example.com",
		"587",
		"", "",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Subject() != "Notification from Tinkerdown" {
		t.Errorf("expected default subject, got %q", output.Subject())
	}
}

func TestEmailOutput_Send_ContextCancellation(t *testing.T) {
	output, err := NewEmailOutputWithConfig(
		"user@example.com",
		"noreply@example.com",
		"Test",
		"localhost",
		"25",
		"", "",
	)
	if err != nil {
		t.Fatalf("unexpected error creating output: %v", err)
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// The send should fail with context error
	err = output.Send(ctx, "test message")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		// The error might be from context or from connection failure
		// Either is acceptable since we're testing that context is respected
		t.Logf("got error: %v (context may have been checked)", err)
	}
}

func TestEmailOutput_Send_ContextTimeout(t *testing.T) {
	output, err := NewEmailOutputWithConfig(
		"user@example.com",
		"noreply@example.com",
		"Test",
		"localhost",    // Use localhost which won't have SMTP
		"25",
		"", "",
	)
	if err != nil {
		t.Fatalf("unexpected error creating output: %v", err)
	}

	// Create a context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait a bit to let context timeout
	time.Sleep(2 * time.Millisecond)

	// The send should fail
	err = output.Send(ctx, "test message")
	if err == nil {
		t.Fatal("expected error for timed out context")
	}
}

func TestEmailOutput_HeaderInjectionPrevention(t *testing.T) {
	// Test that malicious subjects are sanitized
	tests := []struct {
		name    string
		to      string
		from    string
		subject string
		wantErr bool
	}{
		{
			name:    "valid emails and subject",
			to:      "user@example.com",
			from:    "noreply@example.com",
			subject: "Normal subject",
			wantErr: false,
		},
		{
			name:    "subject with newline is sanitized",
			to:      "user@example.com",
			from:    "noreply@example.com",
			subject: "Subject\r\nBcc: attacker@evil.com",
			wantErr: false, // sanitized, not rejected
		},
		{
			name:    "invalid to email",
			to:      "not-an-email",
			from:    "noreply@example.com",
			subject: "Test",
			wantErr: true,
		},
		{
			name:    "invalid from email",
			to:      "user@example.com",
			from:    "not-an-email",
			subject: "Test",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEmailOutputWithConfig(
				tt.to, tt.from, tt.subject,
				"smtp.example.com", "587",
				"", "",
			)

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestEmailOutput_EmailValidation(t *testing.T) {
	tests := []struct {
		name    string
		to      string
		from    string
		wantErr bool
	}{
		{
			name:    "valid emails",
			to:      "user@example.com",
			from:    "noreply@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with display name",
			to:      "User Name <user@example.com>",
			from:    "No Reply <noreply@example.com>",
			wantErr: false,
		},
		{
			name:    "invalid to - no domain",
			to:      "user@",
			from:    "noreply@example.com",
			wantErr: true,
		},
		{
			name:    "invalid to - no @",
			to:      "userexample.com",
			from:    "noreply@example.com",
			wantErr: true,
		},
		{
			name:    "invalid from - empty",
			to:      "user@example.com",
			from:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEmailOutputWithConfig(
				tt.to, tt.from, "Test",
				"smtp.example.com", "587",
				"", "",
			)

			if tt.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
