package stream

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MaxHTTPFile caps the size of a .http file the reader reads.
const MaxHTTPFile = 4 * 1024 * 1024

// HTTPRequest is one request in a .http file, as written.
type HTTPRequest struct {
	Name    string // from the ### line or # @name; "" when neither is set
	Method  string
	URL     string
	Headers string // one per line
	Body    string
}

// ParseHTTPFile reads the subset of the .http format the VS Code REST Client
// and JetBrains IDEs share: ### separates requests and may name the next
// one, # @name also names it, # and // start comments, @name = value defines
// a variable. A request is its request line, then headers up to the first
// blank line, then the body up to the next ###. It returns the requests and
// the variables, or the first line it can't read.
func ParseHTTPFile(text string) ([]HTTPRequest, map[string]string, error) {
	var reqs []HTTPRequest
	vars := map[string]string{}
	var cur *HTTPRequest // the request being read, once its request line is seen
	var headers, body []string
	inHeaders := false
	name := ""
	finish := func() {
		if cur != nil {
			cur.Headers = strings.Join(headers, "\n")
			cur.Body = strings.TrimRight(strings.Join(body, "\n"), "\n")
			reqs = append(reqs, *cur)
		}
		cur, headers, body, inHeaders = nil, nil, nil, false
	}
	for i, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		n := i + 1
		trimmed := strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(trimmed, "###"); ok {
			finish()
			name = strings.TrimSpace(rest)
			continue
		}
		if cur != nil && !inHeaders {
			body = append(body, line) // the body is kept as written
			continue
		}
		switch {
		case cur != nil && trimmed == "":
			inHeaders = false // the blank line before the body
			continue
		case cur != nil && isComment(trimmed):
			continue // REST Client allows comments between headers
		case cur != nil:
			if !headerLine.MatchString(trimmed) {
				return nil, nil, fmt.Errorf("line %d: %q is not a header (Name: value); put a blank line before the body", n, trimmed)
			}
			headers = append(headers, trimmed)
			continue
		case trimmed == "":
			continue
		}
		if m := nameTag.FindStringSubmatch(trimmed); m != nil {
			name = m[1]
			continue
		}
		if isComment(trimmed) {
			continue
		}
		if m := varLine.FindStringSubmatch(trimmed); m != nil {
			vars[m[1]] = strings.TrimSpace(m[2])
			continue
		}
		method, url, ok := parseRequestLine(trimmed)
		if !ok {
			return nil, nil, fmt.Errorf("line %d: %q is not a request line (METHOD URL)", n, trimmed)
		}
		cur = &HTTPRequest{Name: name, Method: method, URL: url}
		inHeaders, name = true, ""
	}
	finish()
	return reqs, vars, nil
}

var (
	nameTag    = regexp.MustCompile(`^(?:#|//)\s*@name\s+(\S+)$`)
	varLine    = regexp.MustCompile(`^@([A-Za-z_][A-Za-z0-9_.-]*)\s*=\s*(.*)$`)
	headerLine = regexp.MustCompile(`^[!#$%&'*+.^_` + "`" + `|~0-9A-Za-z-]+:`)
	varRef     = regexp.MustCompile(`\{\{\s*(\$processEnv\s+)?([A-Za-z_][A-Za-z0-9_.-]*)\s*\}\}`)
)

// isComment reports whether a trimmed line is a # or // comment.
func isComment(trimmed string) bool {
	return strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//")
}

// parseRequestLine reads a request line: METHOD URL, an optional HTTP
// version after the URL, or a bare URL, which means GET. ok is false when the
// line is not a request line.
func parseRequestLine(line string) (method, url string, ok bool) {
	parts := strings.Fields(line)
	if n := len(parts); n > 1 && strings.HasPrefix(parts[n-1], "HTTP/") {
		parts = parts[:n-1]
	}
	switch {
	case len(parts) == 1 && bareURL.MatchString(parts[0]):
		return "GET", parts[0], true
	case len(parts) == 2 && httpMethods[strings.ToUpper(parts[0])]:
		return strings.ToUpper(parts[0]), parts[1], true
	}
	return "", "", false
}

// httpMethods are the methods a request line may name. A fixed list catches
// a typo such as FETCH as an error instead of sending it.
var httpMethods = map[string]bool{
	"GET": true, "HEAD": true, "POST": true, "PUT": true, "PATCH": true,
	"DELETE": true, "OPTIONS": true, "TRACE": true, "CONNECT": true,
}

// bareURL is a request line with no method: an http(s) URL, or one starting
// with a {{variable}} such as {{host}}/users.
var bareURL = regexp.MustCompile(`^(?i:https?://|\{\{)`)

// Fill replaces each {{name}} in s with vars[name], and each
// {{$processEnv NAME}} with the environment variable NAME, as the VS Code
// REST Client does. A plain {{name}} never reads the environment, so a file
// gets only the variables it asks for by name. A variable's value is filled
// too, so @token = {{$processEnv TOKEN}} works; an environment value is used
// as it is. Fill reports the first reference it couldn't fill.
func Fill(s string, vars map[string]string) (string, error) {
	budget := MaxHTTPFile
	return fill(s, vars, 0, &budget)
}

// maxVarDepth bounds how deep variables may refer to variables, which also
// stops a cycle such as @a = {{b}} and @b = {{a}}. Fan-out is bounded
// separately: every value substituted, at any depth, spends one shared
// budget of MaxHTTPFile bytes, so @a = {{b}}{{b}}… can't grow a fill
// exponentially.
const maxVarDepth = 10

func fill(s string, vars map[string]string, depth int, budget *int) (string, error) {
	var err error
	out := varRef.ReplaceAllStringFunc(s, func(ref string) string {
		if err != nil {
			return ref
		}
		m := varRef.FindStringSubmatch(ref)
		if m[1] != "" {
			v, ok := os.LookupEnv(m[2])
			if !ok {
				err = fmt.Errorf("unset environment variable %s", m[2])
			}
			return v
		}
		v, ok := vars[m[2]]
		switch {
		case !ok:
			err = fmt.Errorf("unset variable %s", m[2])
			return ref
		case depth == maxVarDepth:
			err = fmt.Errorf("variable %s refers to variables more than %d deep, or to itself", m[2], maxVarDepth)
			return ref
		}
		if v, err = fill(v, vars, depth+1, budget); err != nil {
			return ref
		}
		if *budget -= len(v); *budget < 0 {
			err = fmt.Errorf("variables fill in to over %d MiB", MaxHTTPFile>>20)
			return ref
		}
		return v
	})
	return out, err
}

// readHTTPFile follows a .http file: it reads it whole at start and whenever
// its size or modification time changes, and each read replaces the input's
// rows. A file that doesn't parse keeps the last good rows and reports why.
func readHTTPFile(ctx context.Context, spec InputSpec, b *Buffer) {
	tick := time.NewTicker(pollInterval)
	defer tick.Stop()
	var seen os.FileInfo
	var last []byte // the content last read, so a touch changes nothing
	for {
		info, err := os.Stat(spec.HTTP)
		switch {
		case err != nil:
			b.hub.setErr(ctx, b, "waiting for "+spec.HTTP)
			seen = nil
		case info.Size() > MaxHTTPFile:
			b.hub.setErr(ctx, b, fmt.Sprintf("%s is over %d bytes", spec.HTTP, MaxHTTPFile))
		case seen == nil || info.Size() != seen.Size() || !info.ModTime().Equal(seen.ModTime()):
			seen = info
			last = readHTTPOnce(ctx, spec, b, last)
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// readHTTPOnce reads the file and, unless its content is last's, replaces
// the rows. It returns the content it read.
func readHTTPOnce(ctx context.Context, spec InputSpec, b *Buffer, last []byte) []byte {
	data, err := os.ReadFile(spec.HTTP)
	if err != nil {
		b.hub.setErr(ctx, b, err.Error())
		return nil
	}
	if last != nil && bytes.Equal(data, last) {
		return last
	}
	reqs, vars, err := ParseHTTPFile(string(data))
	if err != nil {
		b.hub.setErr(ctx, b, err.Error())
		return data
	}
	now := b.hub.now()
	rows := make([]Row, len(reqs))
	privs := make([]Private, len(reqs))
	for i, q := range reqs {
		fields := map[string]interface{}{
			"entry":   strconv.Itoa(i + 1),
			"name":    q.Name,
			"method":  q.Method,
			"url":     q.URL,
			"headers": q.Headers,
			"body":    q.Body,
			"text":    strings.Join([]string{q.Name, q.Method, q.URL}, " "),
		}
		line := q.Method + " " + q.URL
		rows[i] = makeRow(now, "", q.Name, line, false, fields)
		privs[i] = fillRequest(q, vars)
	}
	b.hub.replace(ctx, b, "file", rows, privs)
	b.hub.setErr(ctx, b, "")
	return data
}

// fillRequest fills a request's variables into the values a command gets.
// A variable set nowhere leaves no values and says which.
func fillRequest(q HTTPRequest, vars map[string]string) Private {
	values := map[string]string{"method": q.Method}
	for _, f := range [][2]string{{"url", q.URL}, {"headers", q.Headers}, {"body", q.Body}} {
		filled, err := Fill(f[1], vars)
		if err != nil {
			return Private{Err: err.Error()}
		}
		values[f[0]] = filled
	}
	return Private{Values: values}
}
