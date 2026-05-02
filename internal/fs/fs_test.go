// SPDX-FileCopyrightText: Elias Mueller
//
// SPDX-License-Identifier: MIT

package fs

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

const (
	unformatted = "import QtQuick\nimport QtQml\n\nRectangle {}\n"
	formatted   = "import QtQml\nimport QtQuick\n\nRectangle {}\n"
)

func writeQML(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func readQML(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func TestFormatStream_TransformsContent(t *testing.T) {
	src := strings.NewReader(unformatted)
	var dst bytes.Buffer
	changed, err := FormatStream(src, &dst, nil)
	if err != nil {
		t.Fatalf("FormatStream: %v", err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if dst.String() != formatted {
		t.Errorf("got %q, want %q", dst.String(), formatted)
	}
}

func TestFormatStream_AlreadyFormatted(t *testing.T) {
	src := strings.NewReader(formatted)
	var dst bytes.Buffer
	changed, err := FormatStream(src, &dst, nil)
	if err != nil {
		t.Fatalf("FormatStream: %v", err)
	}
	if changed {
		t.Error("expected changed=false")
	}
	if dst.String() != formatted {
		t.Errorf("got %q, want %q", dst.String(), formatted)
	}
}

func TestFormatFile_WritesWhenChanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.qml")
	writeQML(t, path, unformatted, 0o644)

	changed, err := FormatFile(path, nil)
	if err != nil {
		t.Fatalf("FormatFile: %v", err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if got := readQML(t, path); got != formatted {
		t.Errorf("got %q, want %q", got, formatted)
	}
}

func TestFormatFile_DoesNotWriteWhenUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.qml")
	writeQML(t, path, formatted, 0o644)

	info1, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// Force an mtime gap so a hypothetical rewrite would be visible.
	time.Sleep(10 * time.Millisecond)

	changed, err := FormatFile(path, nil)
	if err != nil {
		t.Fatalf("FormatFile: %v", err)
	}
	if changed {
		t.Error("expected changed=false")
	}
	info2, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info2.ModTime().Equal(info1.ModTime()) {
		t.Errorf("file was rewritten (mtime changed: %v -> %v)", info1.ModTime(), info2.ModTime())
	}
}

func TestFormatFile_PreservesMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.qml")
	// On Windows, Go's file-mode bits are largely synthesized from the
	// read-only attribute, so 0o640 on disk reads back as 0o666. The
	// preservation property we actually care about is "mode after
	// FormatFile == mode before FormatFile" — capture whatever the OS
	// stored and compare against that.
	writeQML(t, path, unformatted, 0o640)
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := FormatFile(path, nil); err != nil {
		t.Fatalf("FormatFile: %v", err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if after.Mode().Perm() != before.Mode().Perm() {
		t.Errorf("mode = %o, want %o (preserved from before format)",
			after.Mode().Perm(), before.Mode().Perm())
	}
}

func TestFormatFile_MissingFile(t *testing.T) {
	_, err := FormatFile(filepath.Join(t.TempDir(), "nope.qml"), nil)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "nope.qml") {
		t.Errorf("error should name the path: %v", err)
	}
}

func TestCheckFile_DetectsChangesWithoutWriting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.qml")
	writeQML(t, path, unformatted, 0o644)

	changed, err := CheckFile(path, nil)
	if err != nil {
		t.Fatalf("CheckFile: %v", err)
	}
	if !changed {
		t.Error("expected changed=true")
	}
	if got := readQML(t, path); got != unformatted {
		t.Errorf("CheckFile modified the file: %q", got)
	}
}

func TestFormatFileTo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.qml")
	writeQML(t, path, unformatted, 0o644)

	var dst bytes.Buffer
	if err := FormatFileTo(path, &dst, nil); err != nil {
		t.Fatalf("FormatFileTo: %v", err)
	}
	if dst.String() != formatted {
		t.Errorf("got %q, want %q", dst.String(), formatted)
	}
	if got := readQML(t, path); got != unformatted {
		t.Errorf("FormatFileTo modified the file: %q", got)
	}
}

func TestWalkQMLFiles_FiltersExtensionAndSkipsHidden(t *testing.T) {
	dir := t.TempDir()
	for _, rel := range []string{
		"a.qml",
		"b.txt",
		".hidden.qml",
		"sub/c.qml",
		".hidden_dir/d.qml",
		"sub/nested/e.qml",
		"sub/.hidden_in_sub/f.qml",
	} {
		writeQML(t, filepath.Join(dir, rel), "// content\n", 0o644)
	}

	var found []string
	if err := WalkQMLFiles(dir, func(p string) error {
		rel, _ := filepath.Rel(dir, p)
		found = append(found, filepath.ToSlash(rel))
		return nil
	}); err != nil {
		t.Fatalf("WalkQMLFiles: %v", err)
	}

	want := []string{"a.qml", "sub/c.qml", "sub/nested/e.qml"}
	slices.Sort(found)
	slices.Sort(want)
	if !slices.Equal(found, want) {
		t.Errorf("found %v, want %v", found, want)
	}
}

func TestWalkQMLFiles_RootBypassesDotRule(t *testing.T) {
	dir := t.TempDir()
	hiddenDir := filepath.Join(dir, ".hidden_dir")
	writeQML(t, filepath.Join(hiddenDir, "x.qml"), "// content\n", 0o644)

	var found []string
	if err := WalkQMLFiles(hiddenDir, func(p string) error {
		found = append(found, filepath.Base(p))
		return nil
	}); err != nil {
		t.Fatalf("WalkQMLFiles: %v", err)
	}
	if !slices.Equal(found, []string{"x.qml"}) {
		t.Errorf("found %v, want [x.qml]", found)
	}
}

func TestWalkQMLFiles_DoesNotFollowSymlinks(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	writeQML(t, filepath.Join(target, "real.qml"), "// content\n", 0o644)
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	var found []string
	if err := WalkQMLFiles(dir, func(p string) error {
		rel, _ := filepath.Rel(dir, p)
		found = append(found, filepath.ToSlash(rel))
		return nil
	}); err != nil {
		t.Fatalf("WalkQMLFiles: %v", err)
	}
	want := []string{"target/real.qml"}
	slices.Sort(found)
	if !slices.Equal(found, want) {
		t.Errorf("found %v, want %v (symlinked dir should not be descended)", found, want)
	}
}

func TestFormatFile_AtomicWriteLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.qml")
	writeQML(t, path, unformatted, 0o644)

	if _, err := FormatFile(path, nil); err != nil {
		t.Fatalf("FormatFile: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".qmlimportsort-") {
			t.Errorf("temp file leaked: %s", e.Name())
		}
	}
}
