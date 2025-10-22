// SPDX-FileCopyrightText: Elias Mueller
//
// SPDX-License-Identifier: MIT

package internal

import (
	"reflect"
	"testing"
)

func TestIsQtImport(t *testing.T) {
	for _, test := range []struct {
		line     string
		expected bool
	}{
		{line: "import QtCore", expected: true},
		{line: "import QtGraphicalEffects", expected: true},
		{line: "import QtLocation", expected: true},
		{line: "import QtMultimedia", expected: true},
		{line: "import QtNetwork", expected: true},
		{line: "import QtPositioning", expected: true},
		{line: "import Qt.", expected: true},
		{line: "import QtQml", expected: true},
		{line: "import QtQml //", expected: true},
		{line: "import QtQml /*", expected: true},
		{line: "import QtQml.", expected: true},
		{line: "import QtQml 2", expected: true},
		{line: "import QtQuick", expected: true},
		{line: "import QtQuick.", expected: true},
		{line: "import QtQuick 2", expected: true},
		{line: "import QtTest", expected: true},
		{line: "import QtTest 2", expected: true},

		{line: "", expected: false},
		{line: "import Qt", expected: false},
		{line: "import MyModule", expected: false},
		{line: "import io.github.what", expected: false},
		{line: "import \"\"", expected: false},
	} {
		actual := isQtImport(test.line)
		if actual != test.expected {
			t.Errorf("isQtImport(%q) = %v; expected %v", test.line, actual, test.expected)
		}
	}
}

func TestIsLibraryImport(t *testing.T) {
	for _, test := range []struct {
		line     string
		expected bool
	}{
		{line: "import io.github.what", expected: true},
		{line: "import io.github.what //", expected: true},

		{line: "", expected: false},
		{line: "import Qt", expected: false},
		{line: "import MyModule", expected: false},
		{line: "import \"\"", expected: false},
	} {
		actual := isLibraryImport(test.line)
		if actual != test.expected {
			t.Errorf("isLibraryImport(%q) = %v; expected %v", test.line, actual, test.expected)
		}
	}
}

func TestIsModuleImport(t *testing.T) {
	for _, test := range []struct {
		line     string
		expected bool
	}{
		{line: "import pyobjects //", expected: true},
		{line: "import MyModule", expected: true},
		{line: "import MyModule //", expected: true},
		{line: "import MyModule // io.github", expected: true},

		{line: "", expected: false},
		{line: "import pyobjects.x", expected: false},
		{line: "import io.github", expected: false},
		{line: "import io.github.what //", expected: false},
		{line: "import \"\"", expected: false},
	} {
		actual := isModuleImport(test.line)
		if actual != test.expected {
			t.Errorf("isModuleImport(%q) = %v; expected %v", test.line, actual, test.expected)
		}
	}
}

func TestIsRelativeImport(t *testing.T) {
	for _, test := range []struct {
		line     string
		expected bool
	}{
		{line: "import '.", expected: true},
		{line: "import \".", expected: true},
		{line: "import \"\"", expected: true},

		{line: "", expected: false},
		{line: "import pyobjects", expected: false},
		{line: "import io.github.what", expected: false},
	} {
		actual := isRelativeImport(test.line)
		if actual != test.expected {
			t.Errorf("isRelativeImport(%q) = %v; expected %v", test.line, actual, test.expected)
		}
	}
}

func TestOrganizeQmlHeaderStatements(t *testing.T) {
	for _, test := range []struct {
		input    []string
		expected []string
	}{
		{
			input:    []string{},
			expected: []string{},
		},
		{
			input: []string{
				"import QtQuick",
			},
			expected: []string{
				"import QtQuick",
			},
		},
		{
			input: []string{
				" import QtQuick",
				"import Qt5Compat.GraphicalEffects",
			},
			expected: []string{
				"import Qt5Compat.GraphicalEffects",
				"import QtQuick",
			},
		},
		{
			input: []string{
				" //",
				" import QtQuick",
				" */",
				" *",
				"/*",
				"// import QtQuick.Controls",
			},
			expected: []string{
				"import QtQuick",
			},
		},
		{
			input: []string{
				"",
				"pragma ComponentBehavior: Bound",
				" import QtQuick",
				"// import QtQuick.Controls",
				"import \"../views\"",
			},
			expected: []string{
				"pragma ComponentBehavior: Bound",
				"",
				"import QtQuick",
				"",
				"import \"../views\"",
			},
		},
		{
			input: []string{
				"import  pyobjects",
				" import QtQuick.Window",
				" import   io.github.whatever.MyTheme",
				" import   IO.github.whatever.MyTheme",
				"",
				"import QtQuick",
				"import \"../views\"",
				" import QtQuick.Layouts",
				"  import   QtQuick.Controls.Material  ",
				"pragma ComponentBehavior: Bound",
				"",
				"",
				"",
			},
			expected: []string{
				"pragma ComponentBehavior: Bound",
				"",
				"import QtQuick",
				"import QtQuick.Controls.Material",
				"import QtQuick.Layouts",
				"import QtQuick.Window",
				"",
				"import IO.github.whatever.MyTheme",
				"import io.github.whatever.MyTheme",
				"",
				"import pyobjects",
				"",
				"import \"../views\"",
			},
		},
	} {
		actual, err := organizeQmlHeaderStatements(test.input)
		if err != nil || !reflect.DeepEqual(actual, test.expected) {
			t.Errorf(`organizeQmlHeaderStatements(%v) = %v, want %v, error %v`, test.input, actual, test.expected, err)
		}
	}

}

func TestIdentifyRelevantLines(t *testing.T) {
	for _, test := range []struct {
		input            []string
		expectedStartIdx int
		expectedEndIdx   int
	}{
		{
			input: []string{
				"import QtQuick",
				"QtObject{",
			},
			expectedStartIdx: 0,
			expectedEndIdx:   0,
		},
		{
			input: []string{
				" import QtQuick",
				"import QtQuick.Controls",
				"QtObject {",
			},
			expectedStartIdx: 0,
			expectedEndIdx:   1,
		},
		{
			input: []string{
				"import QtQuick 2.0",
				"import QtQuick.Controls",
				"TextInput{",
			},
			expectedStartIdx: 0,
			expectedEndIdx:   1,
		},
		{
			input: []string{
				" import QtQuick",
				" import QtQuick.Controls",
				" ",
				" ",
				"QtObject {",
			},
			expectedStartIdx: 0,
			expectedEndIdx:   3,
		},
		{
			input: []string{
				" import QtQuick",
				" // comment",
				" import QtQuick.Controls",
				" ",
				" ",
				" Loader {",
			},
			expectedStartIdx: 0,
			expectedEndIdx:   4,
		},
		{
			input: []string{
				"pragma Singleton",
				"",
				"import QtQuick",
				"import QtQuick.Controls",
				"",
				"QtObject {",
			},
			expectedStartIdx: 0,
			expectedEndIdx:   4,
		},
		{
			input: []string{
				"",
				"",
				"import QtQuick",
				"",
				"// inline comment",
				"import QtQuick.Controls",
				"import MyModule",
				"",
				"",
				"Item {",
			},
			expectedStartIdx: 0,
			expectedEndIdx:   8,
		},
		{
			input: []string{
				"// Header comment",
				"",
				"pragma Singleton",
				"",
				"",
				"import QtQuick",
				"",
				"QtObject {",
			},
			expectedStartIdx: 1,
			expectedEndIdx:   6,
		},
		{
			input: []string{
				"/**",
				" * MOCKED-FileCopyrightText: Jane Doe",
				" *",
				" * MOCKED-License-Identifier: MIT",
				" */",
				" ",
				" ",
				" import QtQuick",
				" import QtQuick.Controls",
				" ",
				" ",
				"QtObject {",
			},
			expectedStartIdx: 5,
			expectedEndIdx:   10,
		},
		{
			input: []string{
				"",
				" // MOCKED-FileCopyrightText: Jane Doe",
				" // ",
				" // MOCKED-License-Identifier: MIT",
				" ",
				" ",
				" import QtQuick",
				" import QtQuick.Controls",
				" ",
				" ",
				"QtObject {",
			},
			expectedStartIdx: 4,
			expectedEndIdx:   9,
		},
	} {
		start, end, err := identifyRelevantLines(test.input)
		if err != nil || start != test.expectedStartIdx || end != test.expectedEndIdx {
			t.Errorf(`identifyRelevantLines(%v) = [%d, %d), want [%d, %d), error %v`,
				test.input, start, end, test.expectedStartIdx, test.expectedEndIdx, err)
		}
	}

}
