// SPDX-FileCopyrightText: Elias Mueller
//
// SPDX-License-Identifier: MIT

package internal

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	normalizeImportWhitespace = regexp.MustCompile("^import\\s+")
	firstStatement            = regexp.MustCompile("^[a-zA-Z0-9.]+\\s+\\{.*")

	qtImportRegex       = regexp.MustCompile("^import\\s+Qt[A-Z5.].*")
	libraryImportRegex  = regexp.MustCompile("^import\\s+[a-zA-Z]+\\..*")
	moduleImportRegex   = regexp.MustCompile("^import\\s+[a-zA-Z]\\w*(\\s|$)")
	relativeImportRegex = regexp.MustCompile("^import\\s+([\"']).*")
)

// identifyRelevantLines returns the start index (inclusive) and end index (inclusive)
func identifyRelevantLines(lines []string) (start, end int, err error) {
	var consecutiveBlankLines int
	var startFound bool

	for index, line := range lines {
		line = strings.TrimSpace(line)

		if !startFound {
			if line == "" {
				consecutiveBlankLines++
			} else if isComment(line) {
				consecutiveBlankLines = 0
			} else if isPragma(line) || strings.HasPrefix(line, "import ") {
				start = index - consecutiveBlankLines
				startFound = true
			}
			continue
		}

		if firstStatement.MatchString(line) {
			return start, index - 1, nil
		}
	}

	return -1, -1, errors.New("could not identify relevant lines")
}

// isComment returns if a line starts with //, /*, *, or */
func isComment(line string) bool {
	return strings.HasPrefix(line, "//") ||
		strings.HasPrefix(line, "/*") ||
		strings.HasPrefix(line, "*") ||
		strings.HasPrefix(line, "*/")
}

func isPragma(line string) bool {
	return strings.HasPrefix(line, "pragma ")
}

func organizeQmlHeaderStatements(lines []string) ([]string, error) {
	pragmas := make([]string, 0, len(lines))
	qt := make([]string, 0, len(lines))
	library := make([]string, 0, len(lines))
	module := make([]string, 0, len(lines))
	relative := make([]string, 0, len(lines))

	// identify import type
	for i, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" || isComment(line) {
			continue
		}

		line = normalizeImportWhitespace.ReplaceAllString(line, "import ")

		if isPragma(line) {
			pragmas = append(pragmas, line)
		} else if isQtImport(line) {
			qt = append(qt, line)
		} else if isLibraryImport(line) {
			library = append(library, line)
		} else if isModuleImport(line) {
			module = append(module, line)
		} else if isRelativeImport(line) {
			relative = append(relative, line)
		} else {
			return nil, fmt.Errorf("cannot identify import type (one of qt, library, module, relative) in line %d: '%s'", i, line)
		}
	}

	var sections [][]string
	if len(pragmas) > 0 {
		sections = append(sections, pragmas)
	}
	if len(qt) > 0 {
		sections = append(sections, qt)
	}
	if len(library) > 0 {
		sections = append(sections, library)
	}
	if len(module) > 0 {
		sections = append(sections, module)
	}
	if len(relative) > 0 {
		sections = append(sections, relative)
	}

	// sort & calculate size needed
	sizeRequirement := 0
	for _, section := range sections {
		sort.Strings(section)
		sizeRequirement += len(section)
	}

	if len(sections) > 1 {
		sizeRequirement += len(sections) - 1
	}

	// concat imports
	result := make([]string, 0, sizeRequirement)
	for idx, section := range sections {
		result = append(result, section...)
		if idx != len(sections)-1 {
			result = append(result, "")
		}
	}

	return result, nil
}

// isQtImport returns true if a line matches something like "import Qt...", false else.
func isQtImport(line string) bool {
	return qtImportRegex.MatchString(line)
}

// isLibraryImport returns true if a line matches something like "import io.github.whatever", false else.
func isLibraryImport(line string) bool {
	return libraryImportRegex.MatchString(line)
}

// isModuleImport returns true if a line matches something like "import MyModule" or "import mymodule", false else.
func isModuleImport(line string) bool {
	return moduleImportRegex.MatchString(line)
}

// isRelativeImport returns true if a line matches something like >import "< or >import '<
func isRelativeImport(line string) bool {
	return relativeImportRegex.MatchString(line)
}
