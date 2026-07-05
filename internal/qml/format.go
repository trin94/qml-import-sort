// SPDX-FileCopyrightText: Elias Mueller
//
// SPDX-License-Identifier: MIT

package qml

import (
	"fmt"
	"slices"
	"strings"
)

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
	if len(src) == 0 {
		return nil, nil
	}
	if c == nil {
		c = &Classifier{}
	}

	lineEnding := detectLineEnding(src)
	lines, hadTrailingLE := splitLines(src, lineEnding)
	lines = stripLeadingBlanks(lines)
	if len(lines) == 0 {
		return nil, nil
	}

	firstBlock, lastBlock := findBlockRange(lines)

	var preamble, hoisted, block, body []string
	if firstBlock == -1 {
		body = lines
	} else {
		preamble = stripTrailingBlanks(lines[:firstBlock])
		body = stripLeadingBlanks(lines[lastBlock+1:])

		tokens, err := tokenizeBlock(lines[firstBlock:lastBlock+1], c)
		if err != nil {
			return nil, err
		}
		hoisted = extractComments(tokens)
		block = emitGroups(groupAndSort(tokens))
	}

	var out []string
	for _, section := range [][]string{preamble, hoisted, block, body} {
		if len(section) == 0 {
			continue
		}
		if len(out) > 0 {
			out = append(out, "")
		}
		out = append(out, section...)
	}

	s := strings.Join(out, lineEnding)
	if hadTrailingLE {
		s += lineEnding
	}
	return []byte(s), nil
}

func detectLineEnding(src []byte) string {
	for i := 0; i < len(src); i++ {
		if src[i] == '\r' {
			if i+1 < len(src) && src[i+1] == '\n' {
				return "\r\n"
			}
			return "\r"
		}
		if src[i] == '\n' {
			return "\n"
		}
	}
	return "\n"
}

func splitLines(src []byte, lineEnding string) ([]string, bool) {
	s := string(src)
	hadTrailing := strings.HasSuffix(s, lineEnding)
	if hadTrailing {
		s = s[:len(s)-len(lineEnding)]
	}
	if s == "" {
		return nil, hadTrailing
	}
	return strings.Split(s, lineEnding), hadTrailing
}

func isBlank(line string) bool {
	return strings.TrimSpace(line) == ""
}

func stripLeadingBlanks(lines []string) []string {
	i := 0
	for i < len(lines) && isBlank(lines[i]) {
		i++
	}
	return lines[i:]
}

func stripTrailingBlanks(lines []string) []string {
	i := len(lines)
	for i > 0 && isBlank(lines[i-1]) {
		i--
	}
	return lines[:i]
}

func startsWithKeyword(line, keyword string) bool {
	t := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(t, keyword) || len(t) == len(keyword) {
		return false
	}
	next := t[len(keyword)]
	return next == ' ' || next == '\t'
}

func isPragmaOrImport(line string) bool {
	return startsWithKeyword(line, "pragma") || startsWithKeyword(line, "import")
}

func findBlockRange(lines []string) (first, last int) {
	first, last = -1, -1
	for i, line := range lines {
		if isPragmaOrImport(line) {
			if first == -1 {
				first = i
			}
			last = i
		}
	}
	return
}

type lineKind int

const (
	kindLineComment lineKind = iota
	kindBlockComment
	kindPragma
	kindQt
	kindThirdParty
	kindFirstParty
	kindRelative
)

type blockToken struct {
	kind lineKind
	text string // canonical form for normal lines; original (with leading whitespace) for block comments
}

// extractComments returns the canonical text of every comment line in
// tokens, preserving input order. Block-comment lines retain their
// original leading whitespace; line comments are emitted with leading
// whitespace stripped (handled at classification time).
func extractComments(tokens []blockToken) []string {
	var out []string
	for _, t := range tokens {
		if t.kind == kindLineComment || t.kind == kindBlockComment {
			out = append(out, t.text)
		}
	}
	return out
}

// emitOrder fixes the sequence in which import categories appear in the
// output. Comments are extracted separately by extractComments and
// emitted as a hoisted section between preamble and the import groups.
var emitOrder = []lineKind{kindPragma, kindQt, kindThirdParty, kindFirstParty, kindRelative}

// groupAndSort buckets tokens by category, sorts each bucket by text in
// byte order, removes byte-identical duplicates, and returns one slice
// per category in emitOrder. Comments are dropped.
func groupAndSort(tokens []blockToken) [][]blockToken {
	groups := make(map[lineKind][]blockToken)
	for _, t := range tokens {
		if t.kind == kindLineComment || t.kind == kindBlockComment {
			continue
		}
		groups[t.kind] = append(groups[t.kind], t)
	}
	result := make([][]blockToken, len(emitOrder))
	for i, k := range emitOrder {
		g := groups[k]
		slices.SortFunc(g, func(a, b blockToken) int { return strings.Compare(a.text, b.text) })
		g = slices.CompactFunc(g, func(a, b blockToken) bool { return a.text == b.text })
		result[i] = g
	}
	return result
}

// emitGroups flattens grouped tokens into output lines, inserting one
// blank line between adjacent non-empty groups.
func emitGroups(groups [][]blockToken) []string {
	var out []string
	for _, g := range groups {
		if len(g) == 0 {
			continue
		}
		if len(out) > 0 {
			out = append(out, "")
		}
		for _, t := range g {
			out = append(out, t.text)
		}
	}
	return out
}

func tokenizeBlock(lines []string, c *Classifier) ([]blockToken, error) {
	out := make([]blockToken, 0, len(lines))
	for _, line := range lines {
		if isBlank(line) {
			continue
		}
		tok, err := classifyLine(line, c)
		if err != nil {
			return nil, err
		}
		out = append(out, tok)
	}
	return out, nil
}

func classifyLine(line string, c *Classifier) (blockToken, error) {
	trimmed := strings.TrimLeft(line, " \t")
	switch {
	case strings.HasPrefix(trimmed, "/*"),
		strings.HasPrefix(trimmed, "*/"),
		strings.HasPrefix(trimmed, "*"):
		return blockToken{kind: kindBlockComment, text: line}, nil
	case strings.HasPrefix(trimmed, "//"):
		return blockToken{kind: kindLineComment, text: trimmed}, nil
	case startsWithKeyword(line, "pragma"):
		return classifyPragma(line)
	case startsWithKeyword(line, "import"):
		return classifyImport(line, c)
	}
	return blockToken{}, fmt.Errorf("qml.Format: unrecognized line in block: %q", line)
}

func classifyPragma(line string) (blockToken, error) {
	tokens, err := splitImportTokens(line)
	if err != nil {
		return blockToken{}, fmt.Errorf("qml.Format: pragma %q: %w", line, err)
	}
	if len(tokens) < 2 {
		return blockToken{}, fmt.Errorf("qml.Format: pragma without name: %q", line)
	}
	return blockToken{kind: kindPragma, text: strings.Join(tokens, " ")}, nil
}

func classifyImport(line string, c *Classifier) (blockToken, error) {
	tokens, err := splitImportTokens(line)
	if err != nil {
		return blockToken{}, fmt.Errorf("qml.Format: import %q: %w", line, err)
	}
	if len(tokens) < 2 {
		return blockToken{}, fmt.Errorf("qml.Format: import without name: %q", line)
	}

	name := tokens[1]
	canonical := strings.Join(tokens, " ")

	if name[0] == '"' || name[0] == '\'' {
		return blockToken{kind: kindRelative, text: canonical}, nil
	}

	if !isValidQMLName(name) {
		return blockToken{}, fmt.Errorf("qml.Format: invalid import name %q", name)
	}

	if isQtName(name) {
		return blockToken{kind: kindQt, text: canonical}, nil
	}

	matchText := strings.Join(tokens[1:], " ")
	for _, p := range c.firstPartyPrefixes {
		if strings.HasPrefix(matchText, p) {
			return blockToken{kind: kindFirstParty, text: canonical}, nil
		}
	}
	return blockToken{kind: kindThirdParty, text: canonical}, nil
}

// splitImportTokens splits a pragma or import line into whitespace-
// separated tokens, treating any quoted string ("..." or '...') as a
// single token whose internal whitespace is preserved verbatim. Stray
// line-ending bytes (\r, \n) — which can appear when the input has
// mixed line endings — count as whitespace.
func splitImportTokens(s string) ([]string, error) {
	isWS := func(c byte) bool { return c == ' ' || c == '\t' || c == '\r' || c == '\n' }
	var tokens []string
	i := 0
	for i < len(s) {
		for i < len(s) && isWS(s[i]) {
			i++
		}
		if i >= len(s) {
			break
		}
		if s[i] == '"' || s[i] == '\'' {
			quote := s[i]
			j := i + 1
			for j < len(s) && s[j] != quote {
				j++
			}
			if j >= len(s) {
				return nil, fmt.Errorf("unterminated quoted string")
			}
			tokens = append(tokens, s[i:j+1])
			i = j + 1
		} else {
			j := i
			for j < len(s) && !isWS(s[j]) {
				j++
			}
			tokens = append(tokens, s[i:j])
			i = j
		}
	}
	return tokens, nil
}

func isValidQMLName(s string) bool {
	if s == "" {
		return false
	}
	if !isLetter(s[0]) && s[0] != '_' {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !isLetter(c) && !isDigit(c) && c != '_' && c != '.' {
			return false
		}
	}
	return true
}

func isQtName(name string) bool {
	if name == "QML" || strings.HasPrefix(name, "QML.") {
		return true
	}
	if !strings.HasPrefix(name, "Qt") || len(name) < 3 {
		return false
	}
	c := name[2]
	return (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.'
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}
