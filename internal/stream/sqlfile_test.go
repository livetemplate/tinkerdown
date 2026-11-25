package stream

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

const sampleSQL = `-- Queries for the admin page.
PRAGMA foreign_keys = ON;

-- name: users :many
-- Everyone, newest first.
SELECT id, name
FROM users
ORDER BY id DESC;

-- name: get-user^
SELECT * FROM users WHERE id = 1;

-- name: empty
-- nothing here yet

-- :name close-stale :! :n
UPDATE orders SET status = 'closed' WHERE age > 30;
`

func TestParseSQLFile(t *testing.T) {
	got := ParseSQLFile(sampleSQL, "admin")
	want := []SQLQuery{
		{Name: "users", Line: 4, SQL: "-- Everyone, newest first.\nSELECT id, name\nFROM users\nORDER BY id DESC;"},
		{Name: "get-user", Line: 10, SQL: "SELECT * FROM users WHERE id = 1;"},
		{Name: "close-stale", Line: 16, SQL: "UPDATE orders SET status = 'closed' WHERE age > 30;"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("queries =\n%+v\nwant\n%+v", got, want)
	}

	// The classic layout: one query per file, named after the file.
	got = ParseSQLFile("\uFEFF-- Active users\r\nSELECT * FROM users\r\nWHERE active;\r\n", "users")
	if len(got) != 1 || got[0].Name != "users" || got[0].Line != 1 || got[0].SQL != "-- Active users\nSELECT * FROM users\nWHERE active;" {
		t.Fatalf("a file with no header = %+v", got)
	}
	for _, text := range []string{"", "\n\n", "-- only a comment\n"} {
		if got := ParseSQLFile(text, "x"); len(got) != 0 {
			t.Fatalf("%q gave %+v, want no query", text, got)
		}
	}
	// A name that is only operator characters is kept as written.
	if got := ParseSQLFile("-- name: !\nSELECT 1", "x"); len(got) != 1 || got[0].Name != "!" {
		t.Fatalf("got %+v", got)
	}
}

func TestQueryKind(t *testing.T) {
	for sql, want := range map[string]string{
		"SELECT 1":                         "select",
		"-- Everyone\nSELECT * FROM users": "select",
		"/* the old ones */ DELETE FROM t": "delete",
		"  update t set a = 1":             "update",
		"-- a\n-- b\n\n  Insert INTO t":    "insert",
	} {
		if got := queryKind(sql); got != want {
			t.Errorf("queryKind(%q) = %q, want %q", sql, got, want)
		}
	}
}

// sqlRows starts a hub reading spec through a keyed merge, as a page does.
func sqlRows(t *testing.T, spec InputSpec) (*Hub, *Buffer) {
	t.Helper()
	old := pollInterval
	pollInterval = 20 * time.Millisecond
	t.Cleanup(func() { pollInterval = old })
	h := NewHub()
	t.Cleanup(func() { h.Close() })
	in, err := h.Input(spec)
	if err != nil {
		t.Fatal(err)
	}
	m, _ := h.Merge("queries", []*Buffer{in}, 0, KeySpec{Paths: []string{"fields.entry"}})
	sub := m.Subscribe()
	t.Cleanup(sub.Close)
	return h, m
}

// names lists the rows' query names, with their entries and files.
func names(m *Buffer) []string {
	var out []string
	for _, r := range m.Snapshot() {
		f := r["fields"].(map[string]interface{})
		out = append(out, f["entry"].(string)+" "+f["file"].(string)+" "+f["name"].(string))
	}
	return out
}

func TestSQLReaderFollowsTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "admin.sql")
	if err := os.WriteFile(path, []byte(sampleSQL), 0o644); err != nil {
		t.Fatal(err)
	}
	h, m := sqlRows(t, InputSpec{Name: "admin", SQL: path})
	waitFor(t, "three queries", func() bool { return len(m.Snapshot()) == 3 })

	r := m.Snapshot()[0]
	f := r["fields"].(map[string]interface{})
	if f["entry"] != "admin/1" || f["file"] != "admin.sql" || f["name"] != "users" || f["line"] != 4 ||
		f["sql"] != "-- Everyone, newest first.\nSELECT id, name\nFROM users\nORDER BY id DESC;" ||
		!strings.Contains(f["text"].(string), "users -- Everyone") {
		t.Fatalf("fields = %v", f)
	}
	if r["msg"] != "users" || r["line"] != "SELECT id, name" || r["source"] != "admin" {
		t.Fatalf("row = %v", r)
	}
	if p, _ := h.Private(r); p.Values["sql"] != f["sql"] || len(p.Values) != 1 {
		t.Fatalf("private = %v", p)
	}

	// Removing a query removes its row and its private value, so a click on
	// it finds nothing.
	stale := m.Snapshot()[2]
	time.Sleep(10 * time.Millisecond) // a new modification time
	os.WriteFile(path, []byte("-- name: only\nSELECT 2;\n"), 0o644)
	waitFor(t, "one query", func() bool { return len(m.Snapshot()) == 1 })
	if _, ok := h.Private(stale); ok {
		t.Fatal("a removed query's SQL is still published")
	}
	if got := names(m); got[0] != "admin/1 admin.sql only" {
		t.Fatalf("rows = %q", got)
	}
}

func TestSQLReaderFollowsAFolder(t *testing.T) {
	dir := t.TempDir()
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("b.sql", "-- name: b1\nSELECT 1;\n-- name: b2\nSELECT 2;\n")
	write("a.SQL", "SELECT 'a';\n")
	write("notes.txt", "SELECT 'not a query';\n")
	write(".#a.sql", "SELECT 'a lock file';\n")
	os.Symlink(filepath.Join(dir, "gone.sql"), filepath.Join(dir, "dangling.sql"))
	os.MkdirAll(filepath.Join(dir, "sub.sql"), 0o755)
	write("sub.sql/deep.sql", "SELECT 'in a subfolder';\n")

	_, m := sqlRows(t, InputSpec{Name: "q", SQL: dir})
	want := []string{"q/1 a.SQL a", "q/2 b.sql b1", "q/3 b.sql b2"}
	waitFor(t, "the folder's queries", func() bool { return slices.Equal(names(m), want) })
	if m.Err() != "" {
		t.Fatalf("err = %q", m.Err())
	}

	// Adding, editing and removing a file each update the rows. Editing a file
	// leaves the folder's own modification time alone.
	write("c.sql", "SELECT 'c';\n")
	waitFor(t, "the new file", func() bool { return len(m.Snapshot()) == 4 })
	write("b.sql", "-- name: b1\nSELECT 1;\n")
	want = []string{"q/1 a.SQL a", "q/2 b.sql b1", "q/3 c.sql c"}
	waitFor(t, "the edit", func() bool { return slices.Equal(names(m), want) })
	os.Remove(filepath.Join(dir, "a.SQL"))
	want = []string{"q/1 b.sql b1", "q/2 c.sql c"}
	waitFor(t, "the removal", func() bool { return slices.Equal(names(m), want) })
}

func TestSQLReaderRefusesTooMuch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.sql")
	if err := os.WriteFile(path, []byte("SELECT '"+strings.Repeat("x", MaxReadFile)+"';"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, m := sqlRows(t, InputSpec{Name: "big", SQL: path})
	waitFor(t, "the size error", func() bool { return strings.Contains(m.Err(), "is over 4194304 bytes") })

	// Folder files count together.
	half := []byte("SELECT '" + strings.Repeat("x", MaxReadFile/2) + "';")
	os.Remove(path)
	os.WriteFile(filepath.Join(dir, "1.sql"), half, 0o644)
	os.WriteFile(filepath.Join(dir, "2.sql"), half, 0o644)
	_, m = sqlRows(t, InputSpec{Name: "two", SQL: dir})
	waitFor(t, "the folder size error", func() bool { return strings.Contains(m.Err(), "over 4194304 bytes together") })
}

func TestHTTPReaderRefusesAFolder(t *testing.T) {
	h := NewHub()
	defer h.Close()
	in, err := h.Input(InputSpec{Name: "api", HTTP: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	sub := in.Subscribe()
	defer sub.Close()
	waitFor(t, "the folder error", func() bool { return strings.Contains(in.Err(), "is a folder, not a file") })
}
