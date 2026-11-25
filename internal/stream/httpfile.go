package stream

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

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
	return newFiller(vars).fill(s)
}

// MaxHTTPFill caps the bytes one read of a .http file may fill in, across
// its variables and every request, so the values it keeps stay bounded.
const MaxHTTPFill = 64 * 1024 * 1024

// maxVarDepth bounds how deep variables may refer to variables, so a long
// chain can't grow the stack.
const maxVarDepth = 10

// filler fills a file's variables, resolving each variable once: the work
// grows with the file, not with how often its variables nest and repeat,
// as in @a = {{b}}{{b}}… Every string it builds spends one budget.
type filler struct {
	vars   map[string]string // literal text around references is bounded by the file size, so only values spend left
	done   map[string]string
	failed map[string]error
	active []string // the variables being filled, innermost last
	left   int      // bytes the filler may still build
}

func newFiller(vars map[string]string) *filler {
	return &filler{vars: vars, done: map[string]string{}, failed: map[string]error{}, left: MaxHTTPFill}
}

func (f *filler) fill(s string) (string, error) {
	var b strings.Builder
	last := 0
	for _, m := range varRef.FindAllStringSubmatchIndex(s, -1) {
		b.WriteString(s[last:m[0]])
		last = m[1]
		name := s[m[4]:m[5]]
		var v string
		var err error
		if m[2] >= 0 { // {{$processEnv NAME}}
			var ok bool
			if v, ok = os.LookupEnv(name); !ok {
				err = fmt.Errorf("unset environment variable %s", name)
			}
		} else {
			v, err = f.value(name)
		}
		if err != nil {
			return "", err
		}
		if f.left -= len(v); f.left < 0 {
			return "", fmt.Errorf("variables fill in to over %d MiB", MaxHTTPFill>>20)
		}
		b.WriteString(v)
	}
	b.WriteString(s[last:])
	return b.String(), nil
}

// tooDeep is the depth limit's error, which value doesn't cache.
type tooDeep struct{ error }

func (e tooDeep) Unwrap() error { return e.error }

// value returns a variable's filled value, filling it the first time.
func (f *filler) value(name string) (string, error) {
	if v, ok := f.done[name]; ok {
		return v, nil
	}
	if err, ok := f.failed[name]; ok {
		return "", err
	}
	raw, ok := f.vars[name]
	switch {
	case !ok:
		return "", fmt.Errorf("unset variable %s", name)
	case slices.Contains(f.active, name):
		return "", fmt.Errorf("variable %s refers to itself", name)
	case len(f.active) == maxVarDepth:
		return "", tooDeep{fmt.Errorf("variable %s nests variables more than %d deep", name, maxVarDepth)}
	}
	f.active = append(f.active, name)
	v, err := f.fill(raw)
	f.active = f.active[:len(f.active)-1]
	if err != nil {
		// How deep a variable sits depends on where it was reached from,
		// so only the other errors, which don't, are kept.
		if !errors.As(err, new(tooDeep)) {
			f.failed[name] = err
		}
		return "", err
	}
	f.done[name] = v
	return v, nil
}

// readHTTPFile follows a .http file, one row per request.
func readHTTPFile(ctx context.Context, spec InputSpec, b *Buffer) {
	followFiles(ctx, b, spec.HTTP, "", func(files []textFile) ([]Row, []Private, error) {
		reqs, vars, err := ParseHTTPFile(string(files[0].data))
		if err != nil {
			return nil, nil, err
		}
		now := b.hub.now()
		rows := make([]Row, len(reqs))
		privs := make([]Private, len(reqs))
		vf := newFiller(vars) // one budget and one resolution for the whole file
		for i, q := range reqs {
			fields := map[string]interface{}{
				"entry":   entryID(spec.Name, i),
				"name":    q.Name,
				"method":  q.Method,
				"url":     q.URL,
				"headers": q.Headers,
				"body":    q.Body,
				"text":    strings.Join([]string{q.Name, q.Method, q.URL}, " "),
			}
			rows[i] = makeRow(now, "", q.Name, q.Method+" "+q.URL, false, fields)
			privs[i] = fillRequest(q, vf)
		}
		return rows, privs, nil
	})
}

// entryID names a row a file reader made: its source's name, "/", and its
// place from 1, so rows from two files differ and a click names one.
func entryID(source string, i int) string {
	return source + "/" + strconv.Itoa(i+1)
}

// fillRequest fills a request's variables into the values a command gets.
// A variable set nowhere leaves no values and says which.
func fillRequest(q HTTPRequest, vf *filler) Private {
	values := map[string]string{"method": q.Method}
	for _, f := range [][2]string{{"url", q.URL}, {"headers", q.Headers}, {"body", q.Body}} {
		filled, err := vf.fill(f[1])
		if err != nil {
			return Private{Err: err.Error()}
		}
		values[f[0]] = filled
	}
	return Private{Values: values}
}
