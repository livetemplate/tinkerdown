package config

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

// TDName is what a value's name may hold. Its variable is TD_<name>, so the
// prefix keeps a value from replacing PATH or another variable a command uses.
var TDName = regexp.MustCompile(`^[A-Za-z0-9_]{1,32}$`)

// MaxTDValue caps a value's length in bytes.
const MaxTDValue = 256

// CheckTDValue refuses a value a command could misread: empty, over
// MaxTDValue bytes, starting with "-" (an option, even when quoted), or
// holding a control character.
func CheckTDValue(name, v string) error {
	switch {
	case v == "":
		return fmt.Errorf("value %s is empty", name)
	case len(v) > MaxTDValue:
		return fmt.Errorf("value %s is over %d bytes", name, MaxTDValue)
	case strings.HasPrefix(v, "-"):
		return fmt.Errorf("value %s starts with -", name)
	case strings.IndexFunc(v, unicode.IsControl) >= 0:
		return fmt.Errorf("value %s holds a control character", name)
	}
	return nil
}

// TDEnv turns values into TD_<name>=value environment entries, sorted by
// name. A name TDName refuses is skipped, so page markup such as a
// hyphenated data-* attribute doesn't block the command; a value under a
// valid name that CheckTDValue refuses fails the whole set.
func TDEnv(values map[string]interface{}) ([]string, error) {
	var env []string
	for _, name := range slices.Sorted(maps.Keys(values)) {
		if !TDName.MatchString(name) {
			continue
		}
		v, err := TDValue(values, name)
		if err != nil {
			return nil, err
		}
		env = append(env, "TD_"+name+"="+v)
	}
	return env, nil
}

// TDValue reads a click's value by name as text, missing as empty, and
// checks it with CheckTDValue.
func TDValue(values map[string]interface{}, name string) (string, error) {
	v := ""
	if values[name] != nil {
		v = fmt.Sprint(values[name])
	}
	return v, CheckTDValue(name, v)
}
