// SPDX-FileCopyrightText: Elias Mueller
//
// SPDX-License-Identifier: MIT

package internal

import (
	"fmt"
	"os"
	"path/filepath"
)

func resolveFiles(files []string) ([]string, error) {
	var existing []string
	for _, file := range files {
		abs, err := filepath.Abs(file)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(abs); os.IsNotExist(err) {
			return nil, fmt.Errorf("file '%s' does not exist", file)
		}
		existing = append(existing, abs)
	}
	return existing, nil
}

func detectLineEnding(content []byte) string {
	for i := 0; i < len(content); i++ {
		if content[i] == '\r' {
			if i+1 < len(content) && content[i+1] == '\n' {
				return "\r\n"
			}
			return "\r"
		}
		if content[i] == '\n' {
			return "\n"
		}
	}
	return "\n"
}
