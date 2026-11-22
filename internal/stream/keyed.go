package stream

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
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
	rows  []Row          // first-seen order, or spec.Sort's
	index map[string]int // key -> position in rows
}

// add folds r into the view. Past size keys, the key updated least recently
// is dropped.
func (k *keyedRows) add(r Row, size int) {
	key := k.keyOf(r)
	i, known := k.index[key]
	if k.spec.Drop.Matches(r) {
		// A drop for a key never seen, or already dropped, changes nothing.
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
		if o := k.spec.Sort; o.Path != "" && o.compare(FieldPath(k.rows[i], o.Path), FieldPath(row, o.Path)) != 0 {
			// Its sort value changed: move it. Two O(n) reindexes, bounded
			// by the buffer size.
			k.remove(i)
			k.insert(row)
			return
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
	k.insert(row)
}

// insert places row at the end, or with Sort set, after every row that
// sorts before it or equal to it, so ties keep the order rows arrived in.
func (k *keyedRows) insert(row Row) {
	at := len(k.rows)
	if o := k.spec.Sort; o.Path != "" {
		v := FieldPath(row, o.Path)
		at = sort.Search(len(k.rows), func(j int) bool { return o.compare(v, FieldPath(k.rows[j], o.Path)) < 0 })
	}
	k.rows = slices.Insert(k.rows, at, row)
	k.reindex(at)
}

// reindex points the index at rows from position from on.
func (k *keyedRows) reindex(from int) {
	for j := from; j < len(k.rows); j++ {
		k.index[k.rows[j]["key"].(string)] = j
	}
}

// Order sorts a keyed merge's rows by the field at Path, descending with
// Desc; see compare for how values order.
type Order struct {
	Path string
	Desc bool
}

// ParseOrder parses PATH, or -PATH for descending.
func ParseOrder(s string) (Order, bool) {
	path, desc := strings.CutPrefix(s, "-")
	if path == "" {
		return Order{}, false
	}
	return Order{Path: path, Desc: desc}, true
}

func (o Order) String() string {
	if o.Desc {
		return "-" + o.Path
	}
	return o.Path
}

// compare orders two sort values: negative when a sorts first. A missing
// value sorts last in either direction. Otherwise numbers, including numeric
// text such as logfmt's, compare as numbers and sort before other text, which
// compares as text: the same numbers a where-clause filter sees.
func (o Order) compare(a, b interface{}) int {
	if a == nil || b == nil {
		return cmp.Compare(boolInt(a == nil), boolInt(b == nil))
	}
	x, xok := number(a)
	y, yok := number(b)
	var c int
	switch {
	case xok && yok:
		c = cmp.Compare(x, y)
	case xok != yok:
		c = cmp.Compare(boolInt(yok), boolInt(xok)) // a number before text
	default:
		c = strings.Compare(text(a), text(b))
	}
	if o.Desc {
		c = -c
	}
	return c
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func text(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

// number returns v as a number when it is one, or text that parses as one.
func number(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case string:
		// Plain decimal text only: ParseFloat also takes "inf", "nan" and
		// hex, which would make a status such as "inf" sort as a number.
		if t := strings.TrimSpace(n); decimal.MatchString(t) {
			f, err := strconv.ParseFloat(t, 64)
			return f, err == nil
		}
	}
	return 0, false
}

var decimal = regexp.MustCompile(`^[-+]?(\d+\.?\d*|\.\d+)([eE][-+]?\d+)?$`)

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

// ParseMatch reads PATH=VALUE. It reports false when "=" is missing, PATH is
// empty, or an alternative is empty (a||b); the caller checks that Path is a
// field path. "|" can't be escaped.
func ParseMatch(s string) (Match, bool) {
	path, value, ok := strings.Cut(s, "=")
	values := strings.Split(value, "|")
	if !ok || path == "" || slices.Contains(values, "") {
		return Match{}, false
	}
	return Match{Path: path, Values: values}, true
}

// Matches reports whether r's field at m.Path is one of m.Values. A number is
// compared as it prints, so 0 matches the value "0".
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
	k.reindex(i)
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
