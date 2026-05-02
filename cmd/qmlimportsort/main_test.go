// SPDX-FileCopyrightText: Elias Mueller
//
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	unformatted = "import QtQuick\nimport QtQml\n\nRectangle {}\n"
	formatted   = "import QtQml\nimport QtQuick\n\nRectangle {}\n"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

type result struct {
	code   int
	stdout string
	stderr string
}

func runCmd(t *testing.T, args []string, stdin string) result {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, strings.NewReader(stdin), &out, &errb)
	return result{code: code, stdout: out.String(), stderr: errb.String()}
}

func TestNoArgsPrintsUsageExit2(t *testing.T) {
	r := runCmd(t, nil, "")
	if r.code != 2 {
		t.Errorf("code = %d, want 2", r.code)
	}
	if !strings.Contains(r.stderr, "USAGE") {
		t.Errorf("stderr should contain usage; got %q", r.stderr)
	}
}

func TestVersion(t *testing.T) {
	r := runCmd(t, []string{"--version"}, "")
	if r.code != 0 {
		t.Errorf("code = %d, want 0", r.code)
	}
	if strings.TrimSpace(r.stdout) == "" {
		t.Errorf("expected version on stdout")
	}
}

func TestHelp(t *testing.T) {
	r := runCmd(t, []string{"--help"}, "")
	if r.code != 0 {
		t.Errorf("code = %d, want 0", r.code)
	}
	if !strings.Contains(r.stdout, "USAGE") {
		t.Errorf("expected usage on stdout; got %q", r.stdout)
	}
}

func TestStdinTransforms(t *testing.T) {
	r := runCmd(t, []string{"--stdin"}, unformatted)
	if r.code != 0 {
		t.Errorf("code = %d, want 0; stderr=%q", r.code, r.stderr)
	}
	if r.stdout != formatted {
		t.Errorf("stdout = %q, want %q", r.stdout, formatted)
	}
}

func TestStdinCheckChangedExit1(t *testing.T) {
	r := runCmd(t, []string{"--stdin", "--check"}, unformatted)
	if r.code != 1 {
		t.Errorf("code = %d, want 1", r.code)
	}
	if r.stdout != "" {
		t.Errorf("stdout should be empty in --stdin --check; got %q", r.stdout)
	}
}

func TestStdinCheckUnchangedExit0(t *testing.T) {
	r := runCmd(t, []string{"--stdin", "--check"}, formatted)
	if r.code != 0 {
		t.Errorf("code = %d, want 0", r.code)
	}
}

func TestStdinWithPathIsUsageError(t *testing.T) {
	r := runCmd(t, []string{"--stdin", "x.qml"}, "")
	if r.code != 2 {
		t.Errorf("code = %d, want 2", r.code)
	}
}

func TestCheckAndStdoutMutuallyExclusive(t *testing.T) {
	r := runCmd(t, []string{"--check", "--stdout", "x.qml"}, "")
	if r.code != 2 {
		t.Errorf("code = %d, want 2", r.code)
	}
	if !strings.Contains(r.stderr, "mutually exclusive") {
		t.Errorf("stderr = %q", r.stderr)
	}
}

func TestFormatFileInPlace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.qml")
	writeFile(t, path, unformatted)

	r := runCmd(t, []string{path}, "")
	if r.code != 0 {
		t.Errorf("code = %d, want 0; stderr=%q", r.code, r.stderr)
	}
	if got := readFile(t, path); got != formatted {
		t.Errorf("file = %q, want %q", got, formatted)
	}
}

func TestCheckReportsChangedPaths(t *testing.T) {
	dir := t.TempDir()
	changedPath := filepath.Join(dir, "a.qml")
	cleanPath := filepath.Join(dir, "b.qml")
	writeFile(t, changedPath, unformatted)
	writeFile(t, cleanPath, formatted)

	r := runCmd(t, []string{"--check", dir}, "")
	if r.code != 1 {
		t.Errorf("code = %d, want 1", r.code)
	}
	if !strings.Contains(r.stdout, changedPath) {
		t.Errorf("stdout should include %s; got %q", changedPath, r.stdout)
	}
	if strings.Contains(r.stdout, cleanPath) {
		t.Errorf("stdout should not include clean %s; got %q", cleanPath, r.stdout)
	}
	// File on disk unchanged after --check.
	if got := readFile(t, changedPath); got != unformatted {
		t.Errorf("--check modified file: %q", got)
	}
}

func TestCheckCleanDirExit0(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.qml"), formatted)

	r := runCmd(t, []string{"--check", dir}, "")
	if r.code != 0 {
		t.Errorf("code = %d, want 0; stderr=%q", r.code, r.stderr)
	}
}

func TestStdoutSingleFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.qml")
	writeFile(t, path, unformatted)

	r := runCmd(t, []string{"--stdout", path}, "")
	if r.code != 0 {
		t.Errorf("code = %d, want 0; stderr=%q", r.code, r.stderr)
	}
	if r.stdout != formatted {
		t.Errorf("stdout = %q, want %q", r.stdout, formatted)
	}
	if got := readFile(t, path); got != unformatted {
		t.Errorf("--stdout modified file: %q", got)
	}
}

func TestStdoutRejectsDirectory(t *testing.T) {
	dir := t.TempDir()
	r := runCmd(t, []string{"--stdout", dir}, "")
	if r.code != 2 {
		t.Errorf("code = %d, want 2", r.code)
	}
}

func TestStdoutRejectsMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.qml")
	b := filepath.Join(dir, "b.qml")
	writeFile(t, a, formatted)
	writeFile(t, b, formatted)

	r := runCmd(t, []string{"--stdout", a, b}, "")
	if r.code != 2 {
		t.Errorf("code = %d, want 2", r.code)
	}
}

func TestMissingPathIsExit2(t *testing.T) {
	r := runCmd(t, []string{filepath.Join(t.TempDir(), "nope.qml")}, "")
	if r.code != 2 {
		t.Errorf("code = %d, want 2", r.code)
	}
}

func TestBadPrefixExit2(t *testing.T) {
	r := runCmd(t, []string{"--first-party-prefix=QtFoo", "--stdin"}, formatted)
	if r.code != 2 {
		t.Errorf("code = %d, want 2", r.code)
	}
	if !strings.Contains(r.stderr, "QtFoo") {
		t.Errorf("stderr should name offending prefix; got %q", r.stderr)
	}
}

func TestFirstPartyPrefixIsApplied(t *testing.T) {
	in := "import QtQuick\nimport com.acme.widgets\nimport io.github.someone\n\nRectangle {}\n"
	want := "import QtQuick\n\nimport io.github.someone\n\nimport com.acme.widgets\n\nRectangle {}\n"
	r := runCmd(t, []string{"--first-party-prefix=com.acme.", "--stdin"}, in)
	if r.code != 0 {
		t.Errorf("code = %d, want 0; stderr=%q", r.code, r.stderr)
	}
	if r.stdout != want {
		t.Errorf("stdout =\n%q\nwant\n%q", r.stdout, want)
	}
}

func TestWriteWalksDirectoryAndFormatsAll(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.qml")
	sub := filepath.Join(dir, "sub", "b.qml")
	writeFile(t, a, unformatted)
	writeFile(t, sub, unformatted)

	r := runCmd(t, []string{dir}, "")
	if r.code != 0 {
		t.Errorf("code = %d, want 0; stderr=%q", r.code, r.stderr)
	}
	if got := readFile(t, a); got != formatted {
		t.Errorf("a.qml not formatted: %q", got)
	}
	if got := readFile(t, sub); got != formatted {
		t.Errorf("sub/b.qml not formatted: %q", got)
	}
}
