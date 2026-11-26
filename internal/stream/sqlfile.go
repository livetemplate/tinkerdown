package stream

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// SQLQuery is one query in a .sql file, as written.
type SQLQuery struct {
	Name string
	SQL  string // trimmed
	Line int    // the header's line number, or 1 for a file with no header
}

// sqlHeader is a line that names the query under it: -- name: NAME, as sqlc,
// aiosql and yesql write it, or HugSQL's -- :name NAME. What follows NAME,
// such as sqlc's :one, is dropped.
var sqlHeader = regexp.MustCompile(`^--\s*(?i:name:|:name\s)\s*(\S+)`)

// ParseSQLFile reads the queries in a .sql file. A query starts at a header
// line and runs to the next one; text before the first header, and a header
// with no SQL under it, are skipped. A file with no header is one query,
// named name, as in the classic layout of one query per file. A name keeps no
// trailing operator characters (^ $ ! < * #), which aiosql and yesql use to
// say what a query returns.
func ParseSQLFile(text, name string) []SQLQuery {
	text = strings.TrimPrefix(strings.ReplaceAll(text, "\r\n", "\n"), "\uFEFF")
	lines := strings.Split(text, "\n")
	var queries []SQLQuery
	var cur *SQLQuery
	start := 0 // the first line of cur's SQL
	finish := func(end int) {
		if cur == nil {
			return
		}
		cur.SQL = strings.TrimSpace(strings.Join(lines[start:end], "\n"))
		if firstSQLLine(cur.SQL) != "" {
			queries = append(queries, *cur)
		}
	}
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "--") {
			continue
		}
		m := sqlHeader.FindStringSubmatch(t)
		if m == nil {
			continue
		}
		finish(i)
		n := strings.Clone(m[1]) // a row keeps the name, not the whole file
		if t := strings.TrimRight(n, "^$!<*#"); t != "" {
			n = t
		}
		cur, start = &SQLQuery{Name: n, Line: i + 1}, i+1
	}
	if cur == nil {
		cur = &SQLQuery{Name: name, Line: 1}
	}
	finish(len(lines))
	return queries
}

// firstSQLLine is a query's first line that isn't blank or a -- comment, the
// line its row shows; "" when the query holds no SQL.
func firstSQLLine(sql string) string {
	for line := range strings.SplitSeq(sql, "\n") {
		if t := strings.TrimSpace(line); t != "" && !strings.HasPrefix(t, "--") {
			return t
		}
	}
	return ""
}

// queryKind returns a query's kind: its first SQL keyword, lowercased, such
// as select, insert or update, after any -- and /* */ comments. A page uses it
// to mark the queries that write; tinkerdown doesn't judge that, the
// database's read-only mode does. A query with a CTE is with: what follows
// the CTE can only be found by matching its parentheses and quotes.
func queryKind(sql string) string {
	s := sql
	for {
		s = strings.TrimLeftFunc(s, unicode.IsSpace)
		switch {
		case strings.HasPrefix(s, "--"):
			_, s, _ = strings.Cut(s, "\n") // a last-line comment leaves ""
		case strings.HasPrefix(s, "/*"):
			_, rest, ok := strings.Cut(s[2:], "*/")
			if !ok {
				return ""
			}
			s = rest
		default:
			end := strings.IndexFunc(s, func(r rune) bool { return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') })
			if end < 0 {
				end = len(s)
			}
			return strings.ToLower(s[:end])
		}
	}
}

// kindWord is what a kind may be: a page puts it in a CSS class.
var kindWord = regexp.MustCompile(`^[a-z]+$`)

// readSQLFiles follows a .sql file, or a folder of them, one row per query.
// Each row's SQL reaches a clicked command as TD_sql.
func readSQLFiles(ctx context.Context, spec InputSpec, b *Buffer) {
	followFiles(ctx, b, spec.SQL, ".sql", func(files []textFile) ([]Row, []Private, error) {
		now := b.hub.now()
		var rows []Row
		var privs []Private
		for _, f := range files {
			for _, q := range ParseSQLFile(string(f.data), strings.TrimSuffix(f.name, filepath.Ext(f.name))) {
				kind := queryKind(q.SQL)
				if !kindWord.MatchString(kind) {
					kind = ""
				}
				fields := map[string]interface{}{
					"entry": EntryID(spec.Name, len(rows)),
					"file":  f.name,
					"name":  q.Name,
					"kind":  kind,
					"sql":   q.SQL,
					"line":  q.Line,
					"text":  q.Name + " " + q.SQL,
				}
				rows = append(rows, makeRow(now, "", q.Name, firstSQLLine(q.SQL), false, fields))
				privs = append(privs, Private{Values: map[string]string{"sql": q.SQL}})
			}
		}
		return rows, privs, nil
	})
}
