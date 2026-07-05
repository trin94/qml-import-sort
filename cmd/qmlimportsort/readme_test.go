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

// TestReadmeExamples runs every marked example in README.MD against the
// real CLI so the documentation cannot drift from the implementation.
//
// Convention: an "<!-- test:input -->" marker precedes the fenced block
// holding the example source file; each "<!-- test:expect -->" marker
// precedes a fenced block holding the expected output of the last
// inline `qmlimportsort … <file>` command mentioned before the marker.
// The command's trailing file argument is replaced by --stdin when it
// runs, feeding the input block and comparing stdout byte-for-byte.
func TestReadmeExamples(t *testing.T) {
	const (
		inputMarker  = "<!-- test:input -->"
		expectMarker = "<!-- test:expect -->"
	)
	commandRe := regexp.MustCompile("`(qmlimportsort [^`]+)`")

	readme := readFile(t, filepath.Join("..", "..", "README.MD"))

	inputParts := strings.Split(readme, inputMarker)
	if len(inputParts) != 2 {
		t.Fatalf("README must contain exactly one %q marker, found %d", inputMarker, len(inputParts)-1)
	}
	input := firstQMLFence(t, inputParts[1])

	segments := strings.Split(readme, expectMarker)
	if len(segments) < 2 {
		t.Fatalf("README contains no %q marker", expectMarker)
	}
	for i := 1; i < len(segments); i++ {
		commands := commandRe.FindAllStringSubmatch(segments[i-1], -1)
		if commands == nil {
			t.Fatalf("no inline `qmlimportsort …` command found before %q marker %d", expectMarker, i)
		}
		command := commands[len(commands)-1][1]
		expected := firstQMLFence(t, segments[i])

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

// firstQMLFence returns the content of the first ```qml code fence in s,
// including the trailing newline.
func firstQMLFence(t *testing.T, s string) string {
	t.Helper()
	const open = "```qml\n"
	_, after, ok := strings.Cut(s, open)
	if !ok {
		t.Fatalf("no %sfence found after marker", open)
	}
	body := after
	j := strings.Index(body, "\n```")
	if j < 0 {
		t.Fatalf("unterminated %sfence after marker", open)
	}
	return body[:j+1]
}
