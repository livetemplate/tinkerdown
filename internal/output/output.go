// Package output provides notification outputs for tinkerdown.
// Outputs are destinations where notifications from Notify imperatives are sent.
package output

import (
	"context"
	"fmt"
)

// Output represents a notification destination.
// Implementations include Slack, Email, and other messaging services.
type Output interface {
	// Name returns the output identifier (e.g., "slack", "email").
	Name() string

	// Send delivers a message to the output destination.
	// The context can be used for cancellation and timeouts.
	Send(ctx context.Context, message string) error

	// Close releases any resources held by the output.
	Close() error
}

// Config represents the configuration for an output.
type Config struct {
	// Type is the output type: "slack" or "email"
	Type string `yaml:"type"`

	// Channel is the Slack channel (for slack type), e.g., "#team-updates"
	Channel string `yaml:"channel,omitempty"`

	// To is the email recipient address (for email type)
	To string `yaml:"to,omitempty"`

	// Subject is the email subject template (for email type)
	// Defaults to "Notification from Tinkerdown"
	Subject string `yaml:"subject,omitempty"`
}

// Registry manages a collection of outputs.
type Registry struct {
	outputs map[string]Output
}

// NewRegistry creates a new output registry.
func NewRegistry() *Registry {
	return &Registry{
		outputs: make(map[string]Output),
	}
}

// Register adds an output to the registry.
func (r *Registry) Register(name string, output Output) {
	r.outputs[name] = output
}

// Get retrieves an output by name.
func (r *Registry) Get(name string) (Output, bool) {
	output, ok := r.outputs[name]
	return output, ok
}

// SendAll sends a message to all registered outputs.
func (r *Registry) SendAll(ctx context.Context, message string) error {
	var errs []error
	for name, output := range r.outputs {
		if err := output.Send(ctx, message); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to send to %d outputs: %v", len(errs), errs)
	}
	return nil
}

// Close closes all registered outputs.
func (r *Registry) Close() error {
	var errs []error
	for name, output := range r.outputs {
		if err := output.Close(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to close %d outputs: %v", len(errs), errs)
	}
	return nil
}

// NewFromConfig creates an output from configuration.
// Returns an error if the output type is unsupported or configuration is invalid.
func NewFromConfig(name string, cfg Config) (Output, error) {
	switch cfg.Type {
	case "slack":
		return NewSlackOutput(cfg.Channel)
	case "email":
		return NewEmailOutput(cfg.To, cfg.Subject)
	default:
		return nil, fmt.Errorf("unsupported output type: %s", cfg.Type)
	}
}
