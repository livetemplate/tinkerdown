package output

import (
	"context"
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

// EmailOutput sends notifications via SMTP email.
type EmailOutput struct {
	to       string
	from     string
	subject  string
	smtpHost string
	smtpPort string
	username string
	password string
}

// NewEmailOutput creates a new email output.
// SMTP configuration is read from environment variables:
//   - SMTP_HOST: SMTP server hostname
//   - SMTP_PORT: SMTP server port (default: 587)
//   - SMTP_USER: SMTP authentication username
//   - SMTP_PASS: SMTP authentication password
//   - SMTP_FROM: Sender email address
func NewEmailOutput(to, subject string) (*EmailOutput, error) {
	if to == "" {
		return nil, fmt.Errorf("email recipient (to) is required")
	}

	host := os.Getenv("SMTP_HOST")
	if host == "" {
		return nil, fmt.Errorf("SMTP_HOST environment variable not set")
	}

	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "587" // Default to TLS port
	}

	from := os.Getenv("SMTP_FROM")
	if from == "" {
		return nil, fmt.Errorf("SMTP_FROM environment variable not set")
	}

	if subject == "" {
		subject = "Notification from Tinkerdown"
	}

	return &EmailOutput{
		to:       to,
		from:     from,
		subject:  subject,
		smtpHost: host,
		smtpPort: port,
		username: os.Getenv("SMTP_USER"),
		password: os.Getenv("SMTP_PASS"),
	}, nil
}

// NewEmailOutputWithConfig creates an email output with explicit configuration.
// This is primarily useful for testing.
func NewEmailOutputWithConfig(to, from, subject, smtpHost, smtpPort, username, password string) (*EmailOutput, error) {
	if to == "" {
		return nil, fmt.Errorf("email recipient (to) is required")
	}
	if from == "" {
		return nil, fmt.Errorf("sender email (from) is required")
	}
	if smtpHost == "" {
		return nil, fmt.Errorf("SMTP host is required")
	}
	if smtpPort == "" {
		smtpPort = "587"
	}
	if subject == "" {
		subject = "Notification from Tinkerdown"
	}

	return &EmailOutput{
		to:       to,
		from:     from,
		subject:  subject,
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		username: username,
		password: password,
	}, nil
}

// Name returns "email".
func (e *EmailOutput) Name() string {
	return "email"
}

// Send delivers an email with the given message as the body.
func (e *EmailOutput) Send(ctx context.Context, message string) error {
	// Build the email message
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("From: %s\r\n", e.from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", e.to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", e.subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(message)

	addr := e.smtpHost + ":" + e.smtpPort

	// Set up authentication if credentials are provided
	var auth smtp.Auth
	if e.username != "" && e.password != "" {
		auth = smtp.PlainAuth("", e.username, e.password, e.smtpHost)
	}

	// Send the email
	// Note: smtp.SendMail doesn't support context cancellation directly,
	// but the underlying connection has default timeouts
	err := smtp.SendMail(addr, auth, e.from, []string{e.to}, []byte(msg.String()))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// Close is a no-op for email output.
func (e *EmailOutput) Close() error {
	return nil
}

// To returns the configured recipient address.
func (e *EmailOutput) To() string {
	return e.to
}

// Subject returns the configured subject line.
func (e *EmailOutput) Subject() string {
	return e.subject
}
