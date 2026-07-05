// SPDX-FileCopyrightText: Elias Mueller
//
// SPDX-License-Identifier: MIT

package main

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var commandRe = regexp.MustCompile("`(qmlimportsort [^`]+)`")

// TestReadmeExamples runs every marked example in README.MD against the
// real CLI so the documentation cannot drift from the implementation.
//
// Convention: a "<!-- test:input -->" marker is directly followed by the
// fenced block holding the example source file; each "<!-- test:expect -->"
// marker is directly followed by a fenced block holding the expected
// output of the example command. The command is the single inline
// `qmlimportsort … <file>` span in the paragraph directly above the
// marker; its trailing file argument is replaced by --stdin when it
// runs, feeding the input block and comparing stdout byte-for-byte.
func TestReadmeExamples(t *testing.T) {
	const (
		inputMarker  = "<!-- test:input -->"
		expectMarker = "<!-- test:expect -->"
	)

	// Normalize line endings so a CRLF checkout (e.g. Windows CI with
	// autocrlf) parses and compares the same as an LF one.
	readme := strings.ReplaceAll(readFile(t, filepath.Join("..", "..", "README.MD")), "\r\n", "\n")

	inputParts := strings.Split(readme, inputMarker)
	if len(inputParts) != 2 {
		t.Fatalf("README must contain exactly one %q marker, found %d", inputMarker, len(inputParts)-1)
	}
	input := fenceDirectlyAfter(t, inputParts[1], inputMarker)

	segments := strings.Split(readme, expectMarker)
	if len(segments) < 2 {
		t.Fatalf("README contains no %q marker", expectMarker)
	}
	for i := 1; i < len(segments); i++ {
		command := commandAbove(t, segments[i-1], expectMarker, i)
		expected := fenceDirectlyAfter(t, segments[i], expectMarker)

		t.Run(command, func(t *testing.T) {
			args := strings.Fields(command)[1:]
			last := len(args) - 1
			if last < 0 || strings.HasPrefix(args[last], "-") {
				t.Fatalf("command %q must end with a file argument", command)
			}
			args[last] = "--stdin"

			r := runCmd(t, args, input)
			if r.code != 0 {
				t.Fatalf("exit code = %d, stderr = %q", r.code, r.stderr)
			}
			if r.stdout != expected {
				t.Errorf("README output drifted from the implementation\nwant:\n%s\ngot:\n%s", expected, r.stdout)
			}
		})
	}
}

// commandAbove extracts the example command from the paragraph directly
// above occurrence n of marker (segment is the text before it).
// Requiring exactly one command there keeps prose mentioning other
// commands from silently swapping the command under test.
func commandAbove(t *testing.T, segment, marker string, n int) string {
	t.Helper()
	paragraphs := strings.Split(strings.TrimSpace(segment), "\n\n")
	paragraph := paragraphs[len(paragraphs)-1]
	commands := commandRe.FindAllStringSubmatch(paragraph, -1)
	if len(commands) != 1 {
		t.Fatalf("want exactly one inline `qmlimportsort …` command in the paragraph above %q marker %d, found %d", marker, n, len(commands))
	}
	return commands[0][1]
}

// fenceDirectlyAfter returns the content of the ```qml fence that must
// open directly after the marker (only blank lines in between), so a
// deleted or drifted fence fails loudly instead of silently picking up
// a later block.
func fenceDirectlyAfter(t *testing.T, s, marker string) string {
	t.Helper()
	const open = "```qml\n"
	head, body, ok := strings.Cut(s, open)
	if !ok || strings.TrimSpace(head) != "" {
		t.Fatalf("want a %sfence directly after the %q marker", open, marker)
	}
	j := strings.Index(body, "\n```")
	if j < 0 {
		t.Fatalf("unterminated %sfence after the %q marker", open, marker)
	}
	return body[:j+1]
}
