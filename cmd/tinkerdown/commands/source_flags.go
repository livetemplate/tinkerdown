package commands

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/livetemplate/tinkerdown/internal/config"
)

// sourceName is what a --source name may hold. Names end up in filter
// expressions (source = NAME), CSS classes and tab labels, so they stay plain.
var sourceName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

// parseSourceFlag parses a --source value: NAME=COMMAND for a stream source
// that runs COMMAND, or NAME=file:PATH for one that follows PATH. It splits on
// the first "=" only, so a command may contain "=".
func parseSourceFlag(v string) (string, config.SourceConfig, error) {
	name, value, ok := strings.Cut(v, "=")
	if !ok || value == "" {
		return "", config.SourceConfig{}, fmt.Errorf("--source %q: want NAME=COMMAND or NAME=file:PATH", v)
	}
	if !sourceName.MatchString(name) {
		return "", config.SourceConfig{}, fmt.Errorf("--source %q: name %q must be 1-32 letters, digits, _ or -", v, name)
	}
	if path, isFile := strings.CutPrefix(value, "file:"); isFile {
		if path == "" {
			return "", config.SourceConfig{}, fmt.Errorf("--source %q: file: needs a path", v)
		}
		return name, config.SourceConfig{Type: "stream", File: path}, nil
	}
	return name, config.SourceConfig{Type: "stream", Cmd: value}, nil
}

// sourceFlagValue turns a source back into the value parseSourceFlag reads.
func sourceFlagValue(name string, src config.SourceConfig) string {
	if src.File != "" {
		return name + "=file:" + src.File
	}
	return name + "=" + src.Cmd
}

// shellQuote quotes s for a POSIX shell, so a printed command line runs the
// same values when pasted back.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
