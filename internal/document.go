// SPDX-FileCopyrightText: Elias Mueller
//
// SPDX-License-Identifier: MIT

package internal

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type document struct {
	path       string
	lineEnding string
	lines      []string
}

func newDocument(path string) (*document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, errorWithPath(err, path)
	}
	lineEnding := detectLineEnding(content)
	lines := strings.Split(string(content), lineEnding)
	return &document{
		path:       path,
		lineEnding: lineEnding,
		lines:      lines,
	}, nil
}

func newDocumentFromReader(r io.Reader) (*document, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	lineEnding := detectLineEnding(content)
	lines := strings.Split(string(content), lineEnding)
	return &document{
		path:       "",
		lineEnding: lineEnding,
		lines:      lines,
	}, nil
}

func (d *document) organize() error {
	var err error
	d.lines, err = processLines(d.lines)
	return errorWithPath(err, d.path)
}

func (d *document) writeBack() error {
	err := os.WriteFile(d.path, []byte(strings.Join(d.lines, d.lineEnding)), 0600)
	return errorWithPath(err, d.path)
}

func (d *document) string() string {
	return strings.Join(d.lines, d.lineEnding)
}

func errorWithPath(err error, path string) error {
	if err == nil {
		return nil
	}
	if path == "" {
		return err
	}
	return fmt.Errorf("file: %s: %v", path, err.Error())
}
