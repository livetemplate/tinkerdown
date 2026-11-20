package source

import (
	"os"
	"path/filepath"

	"github.com/livetemplate/tinkerdown/internal/config"
)

// resolveSitePath returns p as is when absolute, otherwise relative to siteDir.
func resolveSitePath(siteDir, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(siteDir, p)
}

// expandEnv returns env with $VAR references in each value expanded.
func expandEnv(env map[string]string) map[string]string {
	out := make(map[string]string, len(env))
	for k, v := range env {
		out[k] = os.ExpandEnv(v)
	}
	return out
}

// requireExec rejects running a command unless --allow-exec was given. what
// names the kind of source in the message, e.g. "exec sources".
func requireExec(name, field, what string) error {
	if config.IsExecAllowed() {
		return nil
	}
	return &ValidationError{
		Source: name,
		Field:  field,
		Reason: what + " are disabled by default for security. Use --allow-exec flag to enable.",
	}
}
