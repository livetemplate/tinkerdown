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
	row := make(Row, len(r)+2)
	for f, v := range r {
		row[f] = v
	}
	row["key"] = key
	if i, ok := k.index[key]; ok {
		if k.spec.Collect != "" {
			prev := k.rows[i]["collected"]
			if k.resets(r) {
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

// resets reports whether r starts its key's collected text over.
func (k *keyedRows) resets(r Row) bool {
	if k.spec.ResetPath == "" {
		return false
	}
	v := FieldPath(r, k.spec.ResetPath)
	return v != nil && slices.Contains(strings.Split(k.spec.ResetValue, "|"), fmt.Sprint(v))
}

// evictOldest drops the row whose latest update has the lowest seq.
func (k *keyedRows) evictOldest() {
	oldest := 0
	for i, r := range k.rows {
		if seqOf(r) < seqOf(k.rows[oldest]) {
			oldest = i
		}
	}
	delete(k.index, k.rows[oldest]["key"].(string))
	k.rows = append(k.rows[:oldest], k.rows[oldest+1:]...)
	for i := oldest; i < len(k.rows); i++ {
		k.index[k.rows[i]["key"].(string)] = i
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
