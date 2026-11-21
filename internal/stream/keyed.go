package stream

import (
	"fmt"
	"slices"
	"strings"
)

// MaxCollected caps the text a keyed row gathers in "collected"; the end is
// kept, since that is where a failure usually prints.
const MaxCollected = 64 * 1024

// keyedRows holds a keyed merge's view: one row per distinct key, in the
// order keys first appeared. A new row for a known key replaces the old one in
// place, so a page's keyed range sends an update for it rather than a move.
// Guarded by hub.mu, like the rest of the buffer.
type keyedRows struct {
	spec  KeySpec
	rows  []Row          // first-seen order
	index map[string]int // key -> position in rows
}

// add folds r into the view. Past size keys, the key updated least recently
// is dropped.
func (k *keyedRows) add(r Row, size int) {
	key := k.keyOf(r)
	i, known := k.index[key]
	if k.spec.Drop.Matches(r) {
		if known {
			k.remove(i)
		}
		return
	}
	row := make(Row, len(r)+2)
	for f, v := range r {
		row[f] = v
	}
	row["key"] = key
	if known {
		if k.spec.Collect != "" {
			prev := k.rows[i]["collected"]
			if k.spec.Reset.Matches(r) {
				prev = nil
			}
			row["collected"] = appendCollected(prev, FieldPath(r, k.spec.Collect))
		}
		k.rows[i] = row
		return
	}
	if k.spec.Collect != "" {
		row["collected"] = appendCollected(nil, FieldPath(r, k.spec.Collect))
	}
	if len(k.rows) >= size {
		k.evictOldest()
	}
	k.index[key] = len(k.rows)
	k.rows = append(k.rows, row)
}

// keyOf hashes the row's key fields into the row's "key". A missing field
// counts as empty, so a row without one still gets a key of its own (go
// test's package rows). The hash is hex, so a page can use it as data-key
// whatever the fields hold: quotes or other characters in a test name broke
// the client's lookup of the row to update or remove.
func (k *keyedRows) keyOf(r Row) string {
	parts := make([]string, len(k.spec.Paths))
	for i, p := range k.spec.Paths {
		if v := FieldPath(r, p); v != nil {
			parts[i] = fmt.Sprint(v)
		}
	}
	return hashKey(strings.Join(parts, "\x1f"))
}

// Match is a PATH=VALUE condition on a row, where VALUE may list
// alternatives as a|b. The zero Match matches nothing. It is deliberately
// smaller than the filter language tabs use: one field, exact string values.
type Match struct {
	Path   string
	Values []string
}

// ParseMatch reads PATH=VALUE. It reports false when "=" is missing or either
// side is empty; the caller checks that Path is a field path.
func ParseMatch(s string) (Match, bool) {
	path, value, ok := strings.Cut(s, "=")
	if !ok || path == "" || value == "" {
		return Match{}, false
	}
	return Match{Path: path, Values: strings.Split(value, "|")}, true
}

// Matches reports whether r's field at m.Path is one of m.Values.
func (m Match) Matches(r Row) bool {
	if m.Path == "" {
		return false
	}
	switch v := FieldPath(r, m.Path).(type) {
	case nil:
		return false
	case string:
		return slices.Contains(m.Values, v)
	default:
		return slices.Contains(m.Values, fmt.Sprint(v))
	}
}

func (m Match) String() string {
	if m.Path == "" {
		return ""
	}
	return m.Path + "=" + strings.Join(m.Values, "|")
}

// evictOldest drops the row whose latest update has the lowest seq.
func (k *keyedRows) evictOldest() {
	oldest := 0
	for i, r := range k.rows {
		if seqOf(r) < seqOf(k.rows[oldest]) {
			oldest = i
		}
	}
	k.remove(oldest)
}

// remove deletes the row at i and reindexes the rows after it.
func (k *keyedRows) remove(i int) {
	delete(k.index, k.rows[i]["key"].(string))
	k.rows = slices.Delete(k.rows, i, i+1)
	for j := i; j < len(k.rows); j++ {
		k.index[k.rows[j]["key"].(string)] = j
	}
}

func seqOf(r Row) int64 {
	s, _ := r["seq"].(int64)
	return s
}

func appendCollected(prev, add interface{}) string {
	s, _ := prev.(string)
	if add != nil {
		s += fmt.Sprint(add)
	}
	if len(s) > MaxCollected {
		s = s[len(s)-MaxCollected:]
	}
	return s
}

// FieldPath reads a row field by path: NAME is a row field, and fields.NAME a
// field parsed from a JSON or logfmt line. It returns nil when absent.
func FieldPath(r Row, path string) interface{} {
	if name, ok := strings.CutPrefix(path, "fields."); ok {
		fields, _ := r["fields"].(map[string]interface{})
		return fields[name]
	}
	return r[path]
}
