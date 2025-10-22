// SPDX-FileCopyrightText: Elias Mueller
//
// SPDX-License-Identifier: MIT

package internal

import (
	"fmt"
	"io"
)

func ProcessFiles(files []string, inPlace bool) error {
	files, err := resolveFiles(files)
	if err != nil {
		return err
	}
	for _, file := range files {
		doc, err := newDocument(file)
		if err != nil {
			return err
		}
		if err := doc.organize(); err != nil {
			return err
		}
		if inPlace {
			if err := doc.writeBack(); err != nil {
				return err
			}
		} else {
			fmt.Print(doc.string())
		}
	}
	return nil
}

func ProcessStdIn(reader io.Reader) error {
	doc, err := newDocumentFromReader(reader)
	if err != nil {
		return err
	}
	if err := doc.organize(); err != nil {
		return err
	}
	fmt.Print(doc.string())
	return nil
}

func processLines(lines []string) ([]string, error) {
	start, end, err := identifyRelevantLines(lines)
	if err != nil {
		return nil, err
	}
	organized, err := organizeQmlHeaderStatements(lines[start : end+1])
	if err != nil {
		return nil, err
	}

	// calculate capacity
	capacity := len(lines[:start]) + len(organized) + len(lines[end+1:])
	if start > 0 {
		capacity++
	}
	if end+1 < len(lines) {
		capacity++
	}

	// join final lines
	result := make([]string, 0, capacity)
	result = append(result, lines[:start]...)
	if start > 0 {
		result = append(result, "")
	}
	result = append(result, organized...)
	if end+1 < len(lines) {
		result = append(result, "")
	}
	result = append(result, lines[end+1:]...)
	return result, nil
}
