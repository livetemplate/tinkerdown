package output

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// slackWebhookURLPrefix is the required prefix for Slack webhook URLs.
// This prevents data exfiltration to non-Slack endpoints.
const slackWebhookURLPrefix = "https://hooks.slack.com/"

// validateSlackWebhookURL ensures the webhook URL is a valid Slack webhook.
func validateSlackWebhookURL(url string) error {
	if !strings.HasPrefix(url, slackWebhookURLPrefix) {
		return fmt.Errorf("invalid Slack webhook URL: must start with %s", slackWebhookURLPrefix)
	}
	return nil
}

// SlackOutput sends notifications to a Slack channel via webhook.
type SlackOutput struct {
	channel    string
	webhookURL string
	client     *http.Client
}

// slackPayload represents the Slack webhook request body.
type slackPayload struct {
	Channel string `json:"channel,omitempty"`
	Text    string `json:"text"`
}

// NewSlackOutput creates a new Slack output.
// The webhook URL is read from the SLACK_WEBHOOK_URL environment variable.
// Channel should be in the format "#channel-name".
func NewSlackOutput(channel string) (*SlackOutput, error) {
	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if webhookURL == "" {
		return nil, fmt.Errorf("SLACK_WEBHOOK_URL environment variable not set")
	}
	if err := validateSlackWebhookURL(webhookURL); err != nil {
		return nil, err
	}

	if channel == "" {
		return nil, fmt.Errorf("slack channel is required")
	}

	return &SlackOutput{
		channel:    channel,
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// NewSlackOutputWithURL creates a Slack output with an explicit webhook URL.
// The URL must be a valid Slack webhook URL (starting with https://hooks.slack.com/).
// For testing, use NewSlackOutputForTesting instead.
func NewSlackOutputWithURL(channel, webhookURL string) (*SlackOutput, error) {
	if webhookURL == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}
	if err := validateSlackWebhookURL(webhookURL); err != nil {
		return nil, err
	}

	if channel == "" {
		return nil, fmt.Errorf("slack channel is required")
	}

	return &SlackOutput{
		channel:    channel,
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// NewSlackOutputForTesting creates a Slack output for testing purposes.
// This bypasses webhook URL validation to allow mock servers.
// Do not use in production code.
func NewSlackOutputForTesting(channel, webhookURL string) (*SlackOutput, error) {
	if webhookURL == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}

	if channel == "" {
		return nil, fmt.Errorf("slack channel is required")
	}

	return &SlackOutput{
		channel:    channel,
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// Name returns "slack".
func (s *SlackOutput) Name() string {
	return "slack"
}

// Send posts a message to the Slack channel.
func (s *SlackOutput) Send(ctx context.Context, message string) error {
	payload := slackPayload{
		Channel: s.channel,
		Text:    message,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack API error: status %d", resp.StatusCode)
	}

	return nil
}

// Close is a no-op for Slack output.
func (s *SlackOutput) Close() error {
	return nil
}

// Channel returns the configured channel name.
func (s *SlackOutput) Channel() string {
	return s.channel
}
