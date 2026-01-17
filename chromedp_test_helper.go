//go:build !ci

package tinkerdown_test

import (
	"context"
	"os"

	"github.com/chromedp/chromedp"
)

// NewChromedpAllocator creates a chromedp allocator context with appropriate options.
// When running in CI (detected via CI or GITHUB_ACTIONS env vars), it adds the
// --no-sandbox flag to work around Chrome sandbox restrictions on newer Ubuntu versions.
func NewChromedpAllocator(ctx context.Context, opts ...chromedp.ExecAllocatorOption) (context.Context, context.CancelFunc) {
	baseOpts := chromedp.DefaultExecAllocatorOptions[:]

	// Detect CI environment and add --no-sandbox flag
	// This is required on Ubuntu 23.10+ due to AppArmor restrictions on unprivileged user namespaces
	// See: https://chromium.googlesource.com/chromium/src/+/main/docs/security/apparmor-userns-restrictions.md
	if isCI() {
		baseOpts = append(baseOpts, chromedp.Flag("no-sandbox", true))
	}

	// Always run headless in tests
	baseOpts = append(baseOpts, chromedp.Flag("headless", true))

	// Add any additional options passed by the caller
	baseOpts = append(baseOpts, opts...)

	return chromedp.NewExecAllocator(ctx, baseOpts...)
}

// isCI returns true if running in a CI environment
func isCI() bool {
	// Check common CI environment variables
	ciVars := []string{
		"CI",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"CIRCLECI",
		"TRAVIS",
		"JENKINS_URL",
	}
	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return true
		}
	}
	return false
}
