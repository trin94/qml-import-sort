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
	//   - prefix starting with "."
	//   - prefix starting with "Qt" or "qt", equal to "QML", or
	//     starting with "QML." (Qt-owned names always stay in the Qt
	//     section)
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
			if p == "" {
				return nil, errors.New("qml.Compile: empty prefix")
			}
			if strings.HasPrefix(p, ".") {
				return nil, fmt.Errorf("qml.Compile: prefix %q starts with %q", p, ".")
			}
			if isQtReservedPrefix(p) {
				return nil, fmt.Errorf("qml.Compile: prefix %q is reserved (Qt-owned names always stay in the Qt section)", p)
			}
			if seen[p] {
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
