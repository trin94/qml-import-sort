// SPDX-FileCopyrightText: Elias Mueller
//
// SPDX-License-Identifier: MIT

// Package fs is the I/O shell on top of the pure qml package. It reads
// files / streams, calls qml.Format, and (for write modes) lays the
// result back to disk via an atomic temp-file-then-rename.
package fs

import (
	"bytes"
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/trin94/qml-import-sort/internal/qml"
)

// FormatStream reads QML content from src, formats it via qml.Format,
// and writes the result to dst. c is forwarded to qml.Format.
// Returns (changed, err) where changed reports whether the formatted
// output differs byte-for-byte from the input.
func FormatStream(src io.Reader, dst io.Writer, c *qml.Classifier) (bool, error) {
	in, err := io.ReadAll(src)
	if err != nil {
		return false, err
	}
	out, changed, err := readAndFormat(in, c)
	if err != nil {
		return false, err
	}
	if _, err := dst.Write(out); err != nil {
		return changed, err
	}
	return changed, nil
}

// FormatFile formats path in place using an atomic write (temp file in
// the same directory + rename). c is forwarded to qml.Format. Returns
// (changed, err) where changed reports whether the file's content on
// disk differs after formatting. File mode is preserved across the
// rename.
func FormatFile(path string, c *qml.Classifier) (bool, error) {
	in, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}
	out, changed, err := readAndFormat(in, c)
	if err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}
	if !changed {
		return false, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}
	if err := writeAtomic(path, out, info.Mode().Perm()); err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}
	return true, nil
}

// CheckFile reports whether formatting path would change its content.
// Does not write. c is forwarded to qml.Format.
func CheckFile(path string, c *qml.Classifier) (bool, error) {
	in, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}
	_, changed, err := readAndFormat(in, c)
	if err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}
	return changed, nil
}

// FormatFileTo reads path, formats it, writes to dst. Does not modify
// the file on disk. c is forwarded to qml.Format.
func FormatFileTo(path string, dst io.Writer, c *qml.Classifier) error {
	in, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	out, _, err := readAndFormat(in, c)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if _, err := dst.Write(out); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// WalkQMLFiles walks root recursively, calling fn(path) for each
// regular file whose name ends in ".qml". Entries whose name begins
// with "." are skipped during descent. Symlinks are not followed.
//
// The root argument itself is processed regardless of a leading dot
// (explicit paths bypass the skip rule).
//
// If fn returns an error, the walk stops and that error is returned.
func WalkQMLFiles(root string, fn func(path string) error) error {
	return filepath.WalkDir(root, func(path string, d iofs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != root && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".qml") {
			return nil
		}
		return fn(path)
	})
}

// readAndFormat runs qml.Format on in and reports whether the output
// differs byte-for-byte from the input.
func readAndFormat(in []byte, c *qml.Classifier) ([]byte, bool, error) {
	out, err := qml.Format(in, c)
	if err != nil {
		return nil, false, err
	}
	return out, !bytes.Equal(in, out), nil
}

// writeAtomic writes data to a temp file in the same directory as path
// and renames it over path. The temp file is removed on any failure.
func writeAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".qmlimportsort-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	success := false
	defer func() {
		tmp.Close()
		if !success {
			os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	success = true
	return nil
}
