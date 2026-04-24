// SPDX-FileCopyrightText: Elias Mueller
//
// SPDX-License-Identifier: MIT

package qml

import "errors"

// Options configure Format's classifier. The zero value applies only the
// default heuristics.
//
// LibraryPrefixes and ModulePrefixes let the caller override the default
// classifier for specific imports. When an import's normalized text
// starts with any string in LibraryPrefixes, it is classified as library
// regardless of whether its name contains a dot. Similarly for
// ModulePrefixes. See Format for the full precedence rules.
//
// A prefix is a literal byte-level string match against the import text
// (the content after "import " with leading/trailing whitespace stripped
// and internal whitespace collapsed). Include a trailing "." to match
// only dotted subpaths (e.g. "MyCorp." matches "MyCorp.Foo" but not
// "MyCorpX").
//
// Prefixes are trimmed of leading and trailing whitespace before
// validation and matching, so "  MyLib  " is treated as "MyLib". This
// trim happens before all validation rules below, which means an
// all-whitespace prefix is rejected as empty and a prefix like
// "  Qt  " is rejected for starting with "Qt".
//
// Format validates the Options at entry and returns an error if any of
// the following rules is violated; every error names the specific
// offending prefix(es):
//
//   - No prefix may be the empty string (after trimming).
//   - No prefix may start with "." (e.g. ".foo" is rejected).
//   - No prefix may start with "Qt" or "qt" — Qt imports are their own
//     category and are not overridable via prefix.
//   - No prefix may equal "pragma" (pragma is a separate category,
//     not overridable via prefix).
//   - Within a single list, no two prefixes may overlap — meaning one
//     is a prefix of the other, or the two are identical. Duplicates
//     are reported as "duplicate"; non-identical overlaps (e.g. "Foo"
//     and "Foo.Bar") are reported as "overlapping".
//   - Across the two lists, the same rule applies: a prefix in
//     LibraryPrefixes and a prefix in ModulePrefixes may not overlap
//     with each other. This catches both exact matches (the prefix
//     appears in both) and one-is-a-prefix-of-the-other cases.
type Options struct {
	LibraryPrefixes []string
	ModulePrefixes  []string
}

// Format sorts, groups, and deduplicates QML imports in src, returning
// the formatted bytes. src is not modified. opts configures the
// classifier; Options{} preserves the default behavior.
//
// Line endings are detected from the input — the first \n, \r\n, or \r
// encountered wins — and that ending is used as the separator throughout
// the output. If the input contains other line-ending bytes later
// (genuinely mixed line endings), they remain as character content on
// their lines; for pragma and import lines, trailing-whitespace
// normalization will strip such stray bytes when they sit at line ends.
//
// A QML document is split into three parts: the preamble (everything
// before the first pragma or import, including any comment lines that
// sit immediately above it), the block (from the first to the last
// pragma/import, inclusive), and the body (everything after). The
// preamble and body are passed through untouched, with two exceptions:
// leading blank lines at the top of the file are stripped, and the
// blank-line separators around the block are normalized to exactly one
// blank line. This means license headers always stay put — whether or
// not the author put a blank line between the header and the imports,
// the header is preamble and the formatter supplies the blank line.
//
// Within the block, each pragma or import is classified into one of
// five categories and emitted in this fixed order:
//
//  1. pragma
//  2. qt       — e.g. import QtQuick, import QtQuick.Controls, import Qt5Compat.*
//  3. library  — dotted module path, e.g. import io.github.me
//  4. module   — bare identifier, e.g. import MyModule
//  5. relative — quoted path, e.g. import "./components"
//
// Classification precedence runs top-to-bottom; the first rule that
// matches wins:
//
//  1. pragma keyword             → pragma
//  2. Qt[A-Z0-9.] pattern        → qt
//  3. opts.LibraryPrefixes match → library (override)
//  4. opts.ModulePrefixes match  → module  (override)
//  5. text contains a dot        → library (default heuristic)
//  6. text is a bare identifier  → module  (default heuristic)
//  7. text starts with " or '    → relative
//  8. none of the above          → error
//
// User-configured prefixes are checked before the default dot/bare
// heuristics, so a caller can both promote (make a bare name count as
// library) and demote (make a dotted name count as module).
//
// Within each category, entries are sorted by normalized text in byte
// order — case-sensitive, so "A" < "a" per ASCII. Duplicates (by
// normalized text) are removed; the first occurrence wins and keeps
// its attached comments, subsequent duplicates are dropped along with
// any comments attached to them.
//
// Comments inside the block attach to the following pragma or import
// and travel with it through sorting. Consecutive comments stay
// together as a group.
//
// Blank lines inside the block use an all-or-nothing rule per category:
// if any blank line appears between imports of a category in the input,
// the output inserts a blank line between every adjacent pair of
// imports in that category. Otherwise, that category is emitted
// tightly packed. Structural separators — between preamble and block,
// between category groups, and between block and body — are always
// normalized to exactly one blank line; multiple blanks in these
// positions collapse to one.
//
// Pragma and import lines are emitted canonical: leading and trailing
// whitespace is stripped, and runs of whitespace between tokens are
// collapsed to single spaces. Whitespace inside quoted strings is
// preserved — the quoted path is treated as a single token, so
// `import    "./my  file.qml"` becomes `import "./my  file.qml"`.
// A trailing comment on the same line as an import or pragma (e.g.
// `import QtQuick 2.15 // note`) is treated as part of that line's
// text: it is kept, it participates in sorting, and the same
// whitespace-collapse rule applies to it. To have a comment that
// travels independently, put it on its own line.
//
// Line comments (//) on their own line are emitted canonical, with
// leading whitespace stripped. Block-comment lines — those whose
// trimmed content starts with /*, *, or */ — preserve their leading
// whitespace so that multi-line block comment alignment survives.
// Whitespace-only lines count as blank.
//
// A file with no imports (and no pragmas) is valid input and is
// returned unchanged.
//
// Returns an error if opts fails validation (see Options for the full
// list of rules) or if a line inside the block cannot be classified as
// a pragma, import, comment, or blank line.
func Format(src []byte, opts Options) ([]byte, error) {
	return nil, errors.New("qml.Format: not implemented")
}
