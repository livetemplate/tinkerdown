package stream

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// lineSplitter turns written bytes into lines for emit. A line longer than
// MaxLineBytes is emitted cut, with truncated=true, and the rest of it is
// dropped, so one huge line never stops the stream. A partial last line waits
// for its newline; Flush emits it.
type lineSplitter struct {
	pending    []byte
	discarding bool // dropping the rest of a line already emitted truncated
	emit       func(line string, truncated bool)
}

func (s *lineSplitter) Write(p []byte) (int, error) {
	n := len(p)
	for len(p) > 0 {
		i := bytes.IndexByte(p, '\n')
		if i < 0 {
			s.add(p)
			break
		}
		s.add(p[:i])
		if !s.discarding {
			s.emit(string(bytes.TrimRight(s.pending, "\r")), false)
		}
		s.pending, s.discarding = s.pending[:0], false
		p = p[i+1:]
	}
	return n, nil
}

func (s *lineSplitter) add(p []byte) {
	if s.discarding {
		return
	}
	s.pending = append(s.pending, p...)
	if len(s.pending) > MaxLineBytes {
		s.emit(string(s.pending[:MaxLineBytes]), true)
		s.pending, s.discarding = s.pending[:0], true
	}
}

// Flush emits a partial last line, for a stream that has ended.
func (s *lineSplitter) Flush() {
	if len(s.pending) > 0 && !s.discarding {
		s.emit(string(bytes.TrimRight(s.pending, "\r")), false)
	}
	s.pending, s.discarding = s.pending[:0], false
}

// parsers lists, per format, the structured parsers tried in order. A format
// whose list ends in detectLevel falls back to plain text with a guessed level.
var parsers = map[string][]func(string) map[string]interface{}{
	"auto":   {parseJSON, parseLogfmt},
	"json":   {parseJSON},
	"logfmt": {parseLogfmt},
	"lines":  nil,
}

// newRow parses one line in the given (normalized) format. The hub adds seq,
// source and stream when it publishes the row.
func newRow(line string, truncated bool, format string, now time.Time) Row {
	ts, body := now, line
	if format != "lines" {
		// `docker logs --timestamps` and `kubectl logs --timestamps` prefix each
		// line with an RFC 3339 timestamp; use it as the row time.
		if sp := strings.IndexByte(body, ' '); sp > 0 {
			if t, ok := parseTime(body[:sp]); ok {
				ts, body = t, body[sp+1:]
			}
		}
	}

	var fields map[string]interface{}
	for _, parse := range parsers[format] {
		if fields = parse(body); fields != nil {
			break
		}
	}
	if fields == nil {
		level := ""
		if format == "auto" {
			level = detectLevel(body)
		}
		return makeRow(ts, level, body, line, truncated, nil)
	}

	level, _ := takeString(fields, "level", "lvl", "severity")
	msg, ok := takeString(fields, "msg", "message")
	if !ok {
		msg = body
	}
	if v, ok := takeString(fields, "time", "ts", "timestamp"); ok {
		if t, ok := parseTime(v); ok {
			ts = t
		}
	}
	return makeRow(ts, level, msg, line, truncated, fields)
}

// makeRow builds the row shape shared by parsed lines and system rows. A row
// always has a fields map, empty for plain text and system rows, so a
// template can read .fields.NAME on any row.
func makeRow(ts time.Time, level, msg, line string, truncated bool, fields map[string]interface{}) Row {
	if fields == nil {
		fields = map[string]interface{}{}
	}
	return Row{
		"ts":        ts.Format(time.RFC3339Nano),
		"time":      ts.Local().Format("15:04:05.000"),
		"level":     normalizeLevel(level),
		"msg":       msg,
		"line":      line,
		"truncated": truncated,
		"fields":    fields,
	}
}

func parseTime(s string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339Nano, s)
	return t, err == nil
}

func parseJSON(s string) map[string]interface{} {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "{") {
		return nil
	}
	var m map[string]interface{}
	if json.Unmarshal([]byte(s), &m) != nil {
		return nil
	}
	return m
}

// parseLogfmt accepts a line only when every token is key=value and there are
// at least two, so ordinary prose with one "a=b" in it stays plain text.
func parseLogfmt(s string) map[string]interface{} {
	s = strings.TrimSpace(s)
	if eq := strings.IndexByte(s, '='); eq <= 0 || strings.ContainsAny(s[:eq], " \"") {
		return nil // first token isn't key=value; skip the allocation
	}
	pairs := map[string]interface{}{}
	for ; s != ""; s = strings.TrimLeft(s, " ") {
		eq := strings.IndexByte(s, '=')
		if eq <= 0 || strings.ContainsAny(s[:eq], " \"") {
			return nil
		}
		key, rest := s[:eq], s[eq+1:]
		var val string
		switch {
		case strings.HasPrefix(rest, `"`):
			end := closingQuote(rest)
			if end < 0 {
				return nil
			}
			val, rest = strings.ReplaceAll(rest[1:end], `\"`, `"`), rest[end+1:]
		case strings.IndexByte(rest, ' ') >= 0:
			sp := strings.IndexByte(rest, ' ')
			val, rest = rest[:sp], rest[sp:]
		default:
			val, rest = rest, ""
		}
		pairs[key] = val
		s = rest
	}
	if len(pairs) < 2 {
		return nil
	}
	return pairs
}

func closingQuote(s string) int {
	for i := 1; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '"':
			return i
		}
	}
	return -1
}

func takeString(m map[string]interface{}, keys ...string) (string, bool) {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			delete(m, k)
			return fmt.Sprint(v), true
		}
	}
	return "", false
}

// normalizeLevel maps the many spellings of a level onto
// trace, debug, info, warn, error, fatal, or "" when unknown.
func normalizeLevel(s string) string {
	switch strings.ToLower(strings.TrimFunc(s, func(r rune) bool { return !unicode.IsLetter(r) })) {
	case "trace", "trc":
		return "trace"
	case "debug", "dbg":
		return "debug"
	case "info", "inf", "information", "notice":
		return "info"
	case "warn", "warning", "wrn":
		return "warn"
	case "error", "err", "eror":
		return "error"
	case "fatal", "crit", "critical", "panic", "emerg", "alert":
		return "fatal"
	}
	return ""
}

// detectLevel guesses the level of a plain-text line that has no structured
// level field. It returns a raw level word (normalizeLevel cleans it up) or ""
// when the line gives no clear signal.
//
// It reads only the start of the line, so prose such as "no errors found" or
// "retrying after error" stays unlabelled. Recognised starts:
//   - a syslog priority, "<3>disk full" (severity = priority % 8)
//   - a glog prefix, "E1120 19:02:11.123 ..." (I, W, E or F and the date)
//   - a Go crash, "panic: ..." or "fatal error: ..."
//   - a level word, "ERROR ...", "[warn] ...", "WARN: ...", possibly after
//     timestamp tokens such as "2026/11/20 19:02:11 ERROR ..."
func detectLevel(line string) string {
	line = strings.TrimSpace(line)
	if len(line) > 2 && line[0] == '<' {
		if end := strings.IndexByte(line, '>'); end > 1 && end < 5 {
			// Digits only, in syslog's 0-191 range: Atoi accepts "-1", and a
			// negative n would index out of range and crash the reader.
			if n, err := strconv.Atoi(line[1:end]); err == nil && line[1] != '-' && line[1] != '+' && n >= 0 && n <= 191 {
				return syslogSeverity[n%8]
			}
		}
	}
	if strings.HasPrefix(line, "panic:") || strings.HasPrefix(line, "fatal error:") {
		return "fatal"
	}
	fields := strings.Fields(line)
	if len(fields) > 0 && glogPrefix.MatchString(fields[0]) {
		return map[byte]string{'I': "info", 'W': "warn", 'E': "error", 'F': "fatal"}[fields[0][0]]
	}
	for i, tok := range fields {
		if i == 3 {
			break
		}
		if strings.ContainsAny(tok, "0123456789") {
			continue // a timestamp or date before the level
		}
		if normalizeLevel(tok) != "" {
			return tok
		}
		return "" // the first ordinary word ends the search
	}
	return ""
}

// syslogSeverity maps a syslog priority's severity (priority % 8) to a level.
var syslogSeverity = [8]string{"fatal", "fatal", "fatal", "error", "warn", "info", "info", "debug"}

// glogPrefix matches glog's level letter and date, as in "E1120".
var glogPrefix = regexp.MustCompile(`^[IWEF]\d{4}$`)
