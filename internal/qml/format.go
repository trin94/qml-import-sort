// SPDX-FileCopyrightText: Elias Mueller
//
// SPDX-License-Identifier: MIT

package qml

import "errors"

// Format sorts, groups, and deduplicates QML imports in src, returning
// the formatted bytes. src is not modified. c selects the classifier;
// a nil c is equivalent to one compiled from Options{}.
//
// The document is split into preamble (everything before the first
// pragma/import), block (first to last pragma/import, inclusive), and
// body (everything after). Preamble and body pass through untouched
// except: leading blank lines at the top of the file are stripped,
// and the blank-line separator around the block is normalized to one
// blank. License headers always stay in the preamble — whether or not
// the author left a blank line before the imports, the formatter
// supplies one.
//
// The block is classified into five categories — pragma, qt,
// third-party, first-party, relative — and emitted in that fixed
// order. Within each, entries sort by normalized text in byte order
// (case-sensitive, so "A" < "a"). Duplicates are removed; the first
// occurrence wins. Empty categories emit nothing.
//
// Comments inside the block are hoisted: they are collected in input
// order and emitted as a single section between the preamble and the
// imports. They do not travel with any particular import. Comments
// before the first pragma/import are preamble; comments after the
// last are body. Comment lines are independent of duplicate detection —
// every comment from inside the block appears in the hoisted section,
// even if it sat above a duplicate. To annotate a specific import,
// use a trailing comment on the import line (e.g. `import QtQuick
// // note`); that text is part of the line and stays attached
// through sorting.
//
// Blank lines inside the block are stripped — categories are always
// emitted tightly packed. Structural separators between adjacent
// non-empty sections are normalized to exactly one blank line;
// multiple blanks collapse to one.
//
// Pragma and import lines are emitted canonical: leading/trailing
// whitespace stripped, internal whitespace collapsed to single
// spaces. Whitespace inside quoted relative-import paths is preserved.
// Line comments are emitted with leading whitespace stripped;
// block-comment lines (those whose trimmed content starts with /*, *,
// or */) preserve their leading whitespace so multi-line alignment
// survives. Whitespace-only lines count as blank.
//
// Line endings are detected from the input — the first \n, \r\n, or
// \r wins — and used throughout the output.
//
// A file with no imports (and no pragmas) is returned unchanged.
//
// Returns an error if a line inside the block is not a pragma, import,
// comment, or blank. Options validation errors come from Compile, not
// Format.
func Format(src []byte, c *Classifier) ([]byte, error) {
	return nil, errors.New("qml.Format: not implemented")
}
