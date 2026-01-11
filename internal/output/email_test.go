package output

import (
	"testing"
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
