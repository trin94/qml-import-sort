// SPDX-FileCopyrightText: Elias Mueller
//
// SPDX-License-Identifier: MIT

package qml

import "errors"

// Format sorts, groups, and deduplicates QML imports in src, returning
// the formatted bytes. src is not modified.
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
// Returns an error if a line inside the block cannot be classified as
// a pragma, import, comment, or blank line.
func Format(src []byte) ([]byte, error) {
	return nil, errors.New("qml.Format: not implemented")
}
