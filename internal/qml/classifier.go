// SPDX-FileCopyrightText: Elias Mueller
//
// SPDX-License-Identifier: MIT

package qml

import (
	"errors"
	"fmt"
	"strings"
)

// Options configure Format's classifier. The zero value preserves
// Format's default behavior: every non-Qt, non-relative import lands
// in the single default section.
type Options struct {

	// Groups declares the custom sections emitted between the default
	// and relative sections, in order; each inner slice holds the
	// prefixes of one section. The classifier decides membership per
	// import:
	//
	//  1. pragma keyword                → pragma
	//  2. Qt[A-Z0-9.], QML, or QML.…    → qt
	//  3. text starts with " or '       → relative
	//  4. longest matching prefix       → that prefix's group
	//  5. otherwise (valid identifier)  → default
	//
	// A prefix is a literal byte-level string match against the import
	// text (the content after "import " with leading/trailing
	// whitespace stripped and internal whitespace collapsed). Include
	// a trailing "." to match only dotted subpaths — "io.github.mpvqc."
	// matches "io.github.mpvqc.Foo" but not "io.github.mpvqcExternal".
	//
	// Matching uses the longest prefix across all groups, so the order
	// of Groups affects only where sections appear, never which group
	// an import belongs to. Overlapping prefixes are legal; exact
	// duplicates are not, so there are no ties.
	//
	// Prefixes are trimmed of leading and trailing whitespace before
	// validation and matching.
	//
	// Compile validates each group and returns an error identifying
	// the offending prefix or group if any rule is violated:
	//
	//   - a group without prefixes
	//   - empty prefix (after trimming)
	//   - a prefix that can never match: containing a tab or a run of
	//     two or more spaces (import text is normalized to single
	//     spaces), or whose module-name part is not a valid QML-name
	//     prefix (ASCII letter or underscore first; then letters,
	//     digits, underscores, dots)
	//   - a Qt-reserved prefix — starting with "Qt" or "qt", equal to
	//     "QML", or starting with "QML." — the Qt/QML namespace cannot
	//     be grouped
	//   - the same prefix listed twice anywhere
	Groups [][]string
}

// Classifier is a compiled, validated form of Options. Build one with
// Compile and pass it to Format; a nil *Classifier is equivalent to one
// compiled from Options{}, i.e. defaults only.
//
// Compile once and reuse across files. The returned Classifier is
// immutable and safe for concurrent use.
type Classifier struct {
	groups [][]string
}

// Compile validates opts and returns a Classifier suitable for Format.
// See Options.Groups for the validation rules; Compile returns an
// error identifying the offending prefix or group when any rule is
// violated.
func Compile(opts Options) (*Classifier, error) {
	groups := make([][]string, 0, len(opts.Groups))
	seen := make(map[string]bool)
	for i, rawGroup := range opts.Groups {
		if len(rawGroup) == 0 {
			return nil, fmt.Errorf("qml.Compile: group %d has no prefixes", i+1)
		}
		prefixes := make([]string, 0, len(rawGroup))
		for _, raw := range rawGroup {
			p := strings.TrimSpace(raw)
			switch {
			case p == "":
				return nil, errors.New("qml.Compile: empty prefix")
			case strings.ContainsAny(p, "\t\r\n") || strings.Contains(p, "  "):
				return nil, fmt.Errorf("qml.Compile: prefix %q can never match: import text uses single spaces", p)
			case !isMatchablePrefix(p):
				return nil, fmt.Errorf("qml.Compile: prefix %q can never match a QML module name", p)
			case isQtReservedPrefix(p):
				return nil, fmt.Errorf("qml.Compile: prefix %q is reserved: the Qt/QML namespace cannot be grouped", p)
			case seen[p]:
				return nil, fmt.Errorf("qml.Compile: duplicate prefix %q", p)
			}
			seen[p] = true
			prefixes = append(prefixes, p)
		}
		groups = append(groups, prefixes)
	}
	return &Classifier{groups: groups}, nil
}

func isQtReservedPrefix(p string) bool {
	return strings.HasPrefix(p, "Qt") || strings.HasPrefix(p, "qt") ||
		p == "QML" || strings.HasPrefix(p, "QML.")
}

// isMatchablePrefix reports whether p could prefix the whitespace-
// normalized text of a groupable import: its module-name part must be
// a valid QML-name prefix. Anything after the first space (version,
// alias, trailing comment) is deliberately not validated.
func isMatchablePrefix(p string) bool {
	name, _, _ := strings.Cut(p, " ")
	if !isLetter(name[0]) && name[0] != '_' {
		return false
	}
	for i := 1; i < len(name); i++ {
		c := name[i]
		if !isLetter(c) && !isDigit(c) && c != '_' && c != '.' {
			return false
		}
	}
	return true
}

// matchGroup returns the index of the group holding the longest prefix
// matching text; the boolean reports whether any prefix matched.
func (c *Classifier) matchGroup(text string) (int, bool) {
	best, bestLen := -1, 0
	for i, prefixes := range c.groups {
		for _, p := range prefixes {
			if len(p) > bestLen && strings.HasPrefix(text, p) {
				best, bestLen = i, len(p)
			}
		}
	}
	return best, best >= 0
}
