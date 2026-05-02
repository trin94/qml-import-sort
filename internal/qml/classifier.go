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
// in a single third-party group.
type Options struct {

	// FirstPartyPrefixes carves a first-party group out of the default
	// third-party set. The classifier checks these rules top-to-bottom;
	// the first match wins:
	//
	//  1. pragma keyword                → pragma
	//  2. Qt[A-Z0-9.] pattern           → qt
	//  3. text starts with " or '       → relative
	//  4. FirstPartyPrefixes match      → first-party
	//  5. otherwise (valid identifier)  → third-party
	//
	// There is no dotted-vs-bare heuristic: a bare identifier and a
	// dotted path are both third-party by default, and the caller
	// decides which (if any) become first-party.
	//
	// A prefix is a literal byte-level string match against the import
	// text (the content after "import " with leading/trailing
	// whitespace stripped and internal whitespace collapsed). Include
	// a trailing "." to match only dotted subpaths — "MyCorp." matches
	// "MyCorp.Foo" but not "MyCorpX".
	//
	// Prefixes are trimmed of leading and trailing whitespace before
	// validation and matching. Trim happens before every rule below,
	// so "   " is rejected as empty and "  Qt  " is rejected for
	// starting with "Qt".
	//
	// Format validates each prefix at entry and returns an error
	// naming the offending prefix(es) if any rule is violated:
	//
	//   - empty (after trimming)
	//   - starts with "."
	//   - starts with "Qt" or "qt" (Qt is its own category)
	//   - equals "pragma" (pragma is its own category)
	//   - any two prefixes overlap — identical, or one is a prefix of
	//     the other (reported as "duplicate" or "overlapping"
	//     respectively)
	FirstPartyPrefixes []string
}

// Classifier is a compiled, validated form of Options. Build one with
// Compile (or MustCompile) and pass it to Format; a nil *Classifier is
// equivalent to one compiled from Options{}, i.e. defaults only.
//
// Compile once and reuse across files. The returned Classifier is
// immutable and safe for concurrent use.
type Classifier struct {
	firstPartyPrefixes []string
}

// Compile validates opts and returns a Classifier suitable for Format.
// See Options.FirstPartyPrefixes for the validation rules; Compile
// returns an error naming the offending prefix(es) when any rule is
// violated.
func Compile(opts Options) (*Classifier, error) {
	prefixes := make([]string, 0, len(opts.FirstPartyPrefixes))
	for _, raw := range opts.FirstPartyPrefixes {
		p := strings.TrimSpace(raw)
		if p == "" {
			return nil, errors.New("qml.Compile: empty prefix")
		}
		if strings.HasPrefix(p, ".") {
			return nil, fmt.Errorf("qml.Compile: prefix %q starts with %q", p, ".")
		}
		if strings.HasPrefix(p, "Qt") || strings.HasPrefix(p, "qt") {
			return nil, fmt.Errorf("qml.Compile: prefix %q starts with Qt/qt (Qt is its own category)", p)
		}
		if p == "pragma" {
			return nil, fmt.Errorf("qml.Compile: prefix %q is reserved (pragma is its own category)", p)
		}
		prefixes = append(prefixes, p)
	}
	for i, a := range prefixes {
		for _, b := range prefixes[i+1:] {
			if a == b {
				return nil, fmt.Errorf("qml.Compile: duplicate prefix %q", a)
			}
			short, long := a, b
			if len(b) < len(a) {
				short, long = b, a
			}
			if strings.HasPrefix(long, short) {
				return nil, fmt.Errorf("qml.Compile: overlapping prefixes %q and %q (%q is a prefix of %q)", a, b, short, long)
			}
		}
	}
	return &Classifier{firstPartyPrefixes: prefixes}, nil
}

// MustCompile is like Compile but panics if opts fail validation.
// Intended for caller-controlled inputs and tests.
func MustCompile(opts Options) *Classifier {
	c, err := Compile(opts)
	if err != nil {
		panic(err)
	}
	return c
}
