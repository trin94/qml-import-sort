// SPDX-FileCopyrightText: Elias Mueller
//
// SPDX-License-Identifier: MIT

package qml

import (
	"strings"
	"testing"
)

type formatCase struct {
	name            string
	lineEnding      string
	input           []string
	expected        []string
	options         Options
	wantErr         bool
	wantErrContains string
}

func runFormatCases(t *testing.T, cases []formatCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.lineEnding == "" {
				t.Fatalf("test case %q has empty lineEnding; set it explicitly", tc.name)
			}
			input := strings.Join(tc.input, tc.lineEnding) + tc.lineEnding

			c, err := Compile(tc.options)
			var got []byte
			if err == nil {
				got, err = Format([]byte(input), c)
			}

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil (output: %q)", got)
					return
				}
				if tc.wantErrContains != "" && !strings.Contains(err.Error(), tc.wantErrContains) {
					t.Errorf("error message %q does not contain %q", err.Error(), tc.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("Compile/Format returned error: %v", err)
			}

			expected := strings.Join(tc.expected, tc.lineEnding) + tc.lineEnding
			if string(got) != expected {
				t.Errorf("Format output mismatch\nwant:\n%q\ngot:\n%q", expected, got)
				return
			}

			got2, err := Format(got, c)
			if err != nil {
				t.Fatalf("second Format returned error: %v", err)
			}
			if string(got2) != string(got) {
				t.Errorf("Format is not idempotent\nfirst:\n%q\nsecond:\n%q", got, got2)
			}
		})
	}
}

func TestFormatClassification(t *testing.T) {
	runFormatCases(t, []formatCase{
		{
			name:       "single qt import passes through unchanged",
			lineEnding: "\n",
			input: []string{
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "single dotted third-party import passes through unchanged",
			lineEnding: "\n",
			input: []string{
				"import io.github.me 1.0",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import io.github.me 1.0",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "single bare-identifier third-party import passes through unchanged",
			lineEnding: "\n",
			input: []string{
				"import MyModule",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import MyModule",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "single relative import with double quotes passes through unchanged",
			lineEnding: "\n",
			input: []string{
				`import "./components"`,
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				`import "./components"`,
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "single relative import with single quotes passes through unchanged",
			lineEnding: "\n",
			input: []string{
				`import './components'`,
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				`import './components'`,
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "various qt import forms classify and sort correctly",
			lineEnding: "\n",
			input: []string{
				"import QtQuick.Controls 2.15",
				"import Qt5Compat.GraphicalEffects",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import Qt5Compat.GraphicalEffects",
				"import QtQuick",
				"import QtQuick.Controls 2.15",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "qt import with dot path but no version classifies correctly",
			lineEnding: "\n",
			input: []string{
				"import QtQuick.Controls",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import QtQuick.Controls",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "dotted third-party import without version classifies correctly",
			lineEnding: "\n",
			input: []string{
				"import io.github.me",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import io.github.me",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "bare-identifier third-party import with version classifies correctly",
			lineEnding: "\n",
			input: []string{
				"import MyModule 1.0",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import MyModule 1.0",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "relative import with parent directory path classifies correctly",
			lineEnding: "\n",
			input: []string{
				`import "../lib/foo.qml"`,
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				`import "../lib/foo.qml"`,
				"",
				"Rectangle {",
				"}",
			},
		},
	})
}

func TestFormatSorting(t *testing.T) {
	runFormatCases(t, []formatCase{
		{
			name:       "two qt imports are sorted alphabetically",
			lineEnding: "\n",
			input: []string{
				"import QtQuick",
				"import QtQml",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import QtQml",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "mixed categories are grouped and ordered",
			lineEnding: "\n",
			input: []string{
				`import "./components"`,
				"import MyModule",
				"import io.github.me",
				"import QtQuick",
				"pragma Singleton",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"pragma Singleton",
				"",
				"import QtQuick",
				"",
				"import MyModule",
				"import io.github.me",
				"",
				`import "./components"`,
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "pragmas are sorted alphabetically",
			lineEnding: "\n",
			input: []string{
				"pragma Singleton",
				"pragma ComponentBehavior: Bound",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"pragma ComponentBehavior: Bound",
				"pragma Singleton",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "qt import with alias sorts after the unaliased version",
			lineEnding: "\n",
			input: []string{
				"import QtQuick as QQ",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import QtQuick",
				"import QtQuick as QQ",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "pragma appearing after imports is reordered to the top",
			lineEnding: "\n",
			input: []string{
				"import QtQuick",
				"pragma Singleton",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"pragma Singleton",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "sort is case-sensitive with uppercase before lowercase",
			lineEnding: "\n",
			input: []string{
				"import aaa",
				"import Aaa",
				"import AAA",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import AAA",
				"import Aaa",
				"import aaa",
				"",
				"Rectangle {",
				"}",
			},
		},
	})
}

func TestFormatPreamble(t *testing.T) {
	runFormatCases(t, []formatCase{
		{
			name:       "preamble with a single line comment is preserved",
			lineEnding: "\n",
			input: []string{
				"// SPDX-License-Identifier: MIT",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"// SPDX-License-Identifier: MIT",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "preamble with multiple line comments is preserved",
			lineEnding: "\n",
			input: []string{
				"// SPDX-FileCopyrightText: Elias Mueller",
				"//",
				"// SPDX-License-Identifier: MIT",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"// SPDX-FileCopyrightText: Elias Mueller",
				"//",
				"// SPDX-License-Identifier: MIT",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "preamble with a single-line block comment is preserved",
			lineEnding: "\n",
			input: []string{
				"/* SPDX-License-Identifier: MIT */",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"/* SPDX-License-Identifier: MIT */",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "preamble with a multi-line block comment is preserved",
			lineEnding: "\n",
			input: []string{
				"/*",
				" * SPDX-FileCopyrightText: Elias Mueller",
				" *",
				" * SPDX-License-Identifier: MIT",
				" */",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"/*",
				" * SPDX-FileCopyrightText: Elias Mueller",
				" *",
				" * SPDX-License-Identifier: MIT",
				" */",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "leading blank lines at the top of the file are stripped (with preamble)",
			lineEnding: "\n",
			input: []string{
				"",
				"",
				"// SPDX-License-Identifier: MIT",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"// SPDX-License-Identifier: MIT",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "leading blank lines at the top of the file are stripped (no preamble)",
			lineEnding: "\n",
			input: []string{
				"",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "blank lines inside the preamble are preserved while trailing blanks collapse",
			lineEnding: "\n",
			input: []string{
				"// Copyright notice",
				"",
				"",
				"// License text",
				"",
				"",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"// Copyright notice",
				"",
				"",
				"// License text",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "preamble with mixed line and block comments is preserved",
			lineEnding: "\n",
			input: []string{
				"/* License block */",
				"// Additional copyright notice",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"/* License block */",
				"// Additional copyright notice",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "license header without a blank line before the imports stays in the preamble",
			lineEnding: "\n",
			input: []string{
				"// SPDX-License-Identifier: MIT",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"// SPDX-License-Identifier: MIT",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
	})
}

func TestFormatLineEndings(t *testing.T) {
	runFormatCases(t, []formatCase{
		{
			name:       "CRLF line endings are preserved",
			lineEnding: "\r\n",
			input: []string{
				"import QtQuick",
				"import QtQml",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import QtQml",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "CR line endings are preserved",
			lineEnding: "\r",
			input: []string{
				"import QtQuick",
				"import QtQml",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import QtQml",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
	})
}

func TestFormatBlankLines(t *testing.T) {
	runFormatCases(t, []formatCase{
		{
			name:       "blank line is inserted between imports and body when missing",
			lineEnding: "\n",
			input: []string{
				"import QtQuick",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "multiple blank lines between preamble and imports collapse to one",
			lineEnding: "\n",
			input: []string{
				"// SPDX-License-Identifier: MIT",
				"",
				"",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"// SPDX-License-Identifier: MIT",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "multiple blank lines between imports and body collapse to one",
			lineEnding: "\n",
			input: []string{
				"import QtQuick",
				"",
				"",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "multiple blank lines between category groups collapse to one",
			lineEnding: "\n",
			input: []string{
				"import QtQuick",
				"",
				"",
				"",
				"import MyModule",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import QtQuick",
				"",
				"import MyModule",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "blanks between same-category imports are stripped",
			lineEnding: "\n",
			input: []string{
				"import YourModule",
				"",
				"",
				"",
				"import MyModule",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import MyModule",
				"import YourModule",
				"",
				"Rectangle {",
				"}",
			},
		},
	})
}

func TestFormatComments(t *testing.T) {
	runFormatCases(t, []formatCase{
		{
			name:       "comments inside the block are hoisted in input order",
			lineEnding: "\n",
			input: []string{
				"// note about YourModule",
				"import YourModule",
				"// note about MyModule",
				"import MyModule",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"// note about YourModule",
				"// note about MyModule",
				"",
				"import MyModule",
				"import YourModule",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "multiple consecutive comments are all hoisted in input order",
			lineEnding: "\n",
			input: []string{
				"// first comment for ZZZ",
				"// second comment for ZZZ",
				"import ZZZ",
				"// comment for AAA",
				"import AAA",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"// first comment for ZZZ",
				"// second comment for ZZZ",
				"// comment for AAA",
				"",
				"import AAA",
				"import ZZZ",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "comment after last import is treated as body and left untouched",
			lineEnding: "\n",
			input: []string{
				"import QtQuick",
				"// body-level comment",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import QtQuick",
				"",
				"// body-level comment",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "indented line comment inside the block has its indentation stripped when hoisted",
			lineEnding: "\n",
			input: []string{
				"import YourModule",
				"  // indented note about MyModule",
				"import MyModule",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"// indented note about MyModule",
				"",
				"import MyModule",
				"import YourModule",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "block comment inside the block is hoisted with its alignment preserved",
			lineEnding: "\n",
			input: []string{
				"import YourModule",
				"/*",
				" * note about MyModule",
				" */",
				"import MyModule",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"/*",
				" * note about MyModule",
				" */",
				"",
				"import MyModule",
				"import YourModule",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "single-line block comment inside the block is hoisted",
			lineEnding: "\n",
			input: []string{
				"import YourModule",
				"/* note about MyModule */",
				"import MyModule",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"/* note about MyModule */",
				"",
				"import MyModule",
				"import YourModule",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "comment immediately before the first import is preamble and does not move with sort",
			lineEnding: "\n",
			input: []string{
				"// leading comment",
				"import YourModule",
				"import MyModule",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"// leading comment",
				"",
				"import MyModule",
				"import YourModule",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "line that looks like an import inside a comment remains a comment and is hoisted",
			lineEnding: "\n",
			input: []string{
				"import QtQuick",
				"// import QtQml",
				"import QtQml",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"// import QtQml",
				"",
				"import QtQml",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "comment between pragmas is hoisted above the import groups",
			lineEnding: "\n",
			input: []string{
				"pragma Singleton",
				"// note about ComponentBehavior",
				"pragma ComponentBehavior: Bound",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"// note about ComponentBehavior",
				"",
				"pragma ComponentBehavior: Bound",
				"pragma Singleton",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "trailing comment on import line becomes part of the line text",
			lineEnding: "\n",
			input: []string{
				"import QtQuick // zz comment",
				"import QtQml // aa comment",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import QtQml // aa comment",
				"import QtQuick // zz comment",
				"",
				"Rectangle {",
				"}",
			},
		},
	})
}

func TestFormatWhitespace(t *testing.T) {
	runFormatCases(t, []formatCase{
		{
			name:       "leading whitespace after import keyword is normalized",
			lineEnding: "\n",
			input: []string{
				"import    QtQuick",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "indented pragma is normalized to no leading whitespace",
			lineEnding: "\n",
			input: []string{
				"  pragma Singleton",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"pragma Singleton",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "indented imports are normalized regardless of spaces or tabs",
			lineEnding: "\n",
			input: []string{
				"  import QtQuick",
				"\timport QtQml",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import QtQml",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "whitespace-only lines count as blank",
			lineEnding: "\n",
			input: []string{
				"import YourModule",
				"   ",
				"\t",
				"import MyModule",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import MyModule",
				"",
				"import YourModule",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "trailing whitespace on import lines is stripped",
			lineEnding: "\n",
			input: []string{
				"import QtQuick   ",
				"import QtQml\t",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import QtQml",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "internal whitespace inside import lines is normalized to single spaces",
			lineEnding: "\n",
			input: []string{
				"import  QtQuick",
				"import\tQtQml\tas\tQQ",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import QtQml as QQ",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "whitespace inside quoted relative-import paths is preserved",
			lineEnding: "\n",
			input: []string{
				`import    "./my  file.qml"`,
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				`import "./my  file.qml"`,
				"",
				"Rectangle {",
				"}",
			},
		},
	})
}

func TestFormatPassthrough(t *testing.T) {
	runFormatCases(t, []formatCase{
		{
			name:       "file with no imports passes through unchanged",
			lineEnding: "\n",
			input: []string{
				"Rectangle {",
				"    width: 100",
				"    height: 100",
				"}",
			},
			expected: []string{
				"Rectangle {",
				"    width: 100",
				"    height: 100",
				"}",
			},
		},
		{
			name:       "file with only pragmas and no imports is handled",
			lineEnding: "\n",
			input: []string{
				"pragma Singleton",
				"pragma ComponentBehavior: Bound",
				"",
				"QtObject {",
				"}",
			},
			expected: []string{
				"pragma ComponentBehavior: Bound",
				"pragma Singleton",
				"",
				"QtObject {",
				"}",
			},
		},
	})
}

func TestFormatDeduplication(t *testing.T) {
	runFormatCases(t, []formatCase{
		{
			name:       "duplicate imports are deduplicated",
			lineEnding: "\n",
			input: []string{
				"import QtQuick",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "duplicate pragmas are deduplicated",
			lineEnding: "\n",
			input: []string{
				"pragma Singleton",
				"pragma Singleton",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"pragma Singleton",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "duplicates are detected after whitespace normalization",
			lineEnding: "\n",
			input: []string{
				"import QtQuick",
				"import  QtQuick   2.15",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "duplicate imports are removed but every block comment is kept in the hoisted section",
			lineEnding: "\n",
			input: []string{
				"import QtQuick",
				"// first block comment",
				"import QtQuick",
				"// second block comment",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"// first block comment",
				"// second block comment",
				"",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
		},
	})
}

func TestFormatErrors(t *testing.T) {
	runFormatCases(t, []formatCase{
		{
			name:       "import with digit-prefixed module name returns an error",
			lineEnding: "\n",
			input: []string{
				"import QtQuick",
				"import 123invalid",
				"",
				"Rectangle {",
				"}",
			},
			wantErr: true,
		},
		{
			name:       "import with dash in module name returns an error",
			lineEnding: "\n",
			input: []string{
				"import QtQuick",
				"import foo-bar",
				"",
				"Rectangle {",
				"}",
			},
			wantErr: true,
		},
		{
			name:       "import with special character prefix returns an error",
			lineEnding: "\n",
			input: []string{
				"import QtQuick",
				"import @foo",
				"",
				"Rectangle {",
				"}",
			},
			wantErr: true,
		},
	})
}

func TestFormatEdgeCases(t *testing.T) {
	t.Run("empty input returns empty output without error", func(t *testing.T) {
		got, err := Format([]byte{}, nil)
		if err != nil {
			t.Fatalf("Format returned error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty output, got %q", got)
		}
	})

	t.Run("whitespace-only input returns empty output", func(t *testing.T) {
		got, err := Format([]byte("\n\n\n"), nil)
		if err != nil {
			t.Fatalf("Format returned error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty output, got %q", got)
		}
	})

	t.Run("input without trailing newline preserves that property", func(t *testing.T) {
		input := "import QtQuick\nRectangle {}"
		expected := "import QtQuick\n\nRectangle {}"
		got, err := Format([]byte(input), nil)
		if err != nil {
			t.Fatalf("Format returned error: %v", err)
		}
		if string(got) != expected {
			t.Errorf("Format output mismatch\nwant: %q\ngot:  %q", expected, got)
		}
	})

	t.Run("mixed line endings use the first-detected ending as the separator", func(t *testing.T) {
		// Input starts with \n (first-detected), but the second import uses \r\n.
		// After split by \n, the trailing \r is a trimmable whitespace byte that
		// gets stripped by the import-line normalization rule.
		input := "import QtQuick\nimport QtQml\r\n"
		expected := "import QtQml\nimport QtQuick\n"
		got, err := Format([]byte(input), nil)
		if err != nil {
			t.Fatalf("Format returned error: %v", err)
		}
		if string(got) != expected {
			t.Errorf("Format output mismatch\nwant: %q\ngot:  %q", expected, got)
		}
	})
}

func TestFormatIntegration(t *testing.T) {
	runFormatCases(t, []formatCase{
		{
			name:       "realistic file exercises preamble, all categories, comments, dedup, and whitespace normalization",
			lineEnding: "\n",
			input: []string{
				"/*",
				" * SPDX-FileCopyrightText: Elias Mueller",
				" *",
				" * SPDX-License-Identifier: MIT",
				" */",
				"",
				"pragma Singleton",
				"pragma ComponentBehavior: Bound",
				"import  QtQuick   2.15",
				`import "./components"`,
				"// note about MyLib",
				"import io.github.mylib",
				"import MyModule",
				"import QtQml",
				"import io.github.mylib",
				"",
				"Rectangle {",
				"    width: 100",
				"    // internal comment",
				"    height: 100",
				"}",
			},
			expected: []string{
				"/*",
				" * SPDX-FileCopyrightText: Elias Mueller",
				" *",
				" * SPDX-License-Identifier: MIT",
				" */",
				"",
				"// note about MyLib",
				"",
				"pragma ComponentBehavior: Bound",
				"pragma Singleton",
				"",
				"import QtQml",
				"import QtQuick",
				"",
				"import MyModule",
				"import io.github.mylib",
				"",
				`import "./components"`,
				"",
				"Rectangle {",
				"    width: 100",
				"    // internal comment",
				"    height: 100",
				"}",
			},
		},
	})
}

func TestFormatFirstPartyPrefixes(t *testing.T) {
	runFormatCases(t, []formatCase{
		{
			name:       "first-party prefix carves a bare identifier out of third-party",
			lineEnding: "\n",
			options:    Options{FirstPartyPrefixes: []string{"MyLib"}},
			input: []string{
				"import MyLib",
				"import QtQuick",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import QtQuick",
				"",
				"import MyLib",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "first-party prefix matches a dotted name",
			lineEnding: "\n",
			options:    Options{FirstPartyPrefixes: []string{"MyCorp."}},
			input: []string{
				"import MyCorp.Foo",
				"import io.github.other",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import io.github.other",
				"",
				"import MyCorp.Foo",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "trailing dot in the prefix creates a boundary that does not match siblings",
			lineEnding: "\n",
			options:    Options{FirstPartyPrefixes: []string{"MyCorp."}},
			input: []string{
				"import MyCorp.Foo",
				"import MyCorpExternal.Bar",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import MyCorpExternal.Bar",
				"",
				"import MyCorp.Foo",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "default options put every non-Qt non-relative import into a single third-party group",
			lineEnding: "\n",
			input: []string{
				"import QtQuick",
				"import io.github.other",
				"import PlainModule",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import QtQuick",
				"",
				"import PlainModule",
				"import io.github.other",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "multiple first-party prefixes are all honored",
			lineEnding: "\n",
			options:    Options{FirstPartyPrefixes: []string{"Alpha", "Beta"}},
			input: []string{
				"import Alpha",
				"import Beta",
				"import Gamma",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import Gamma",
				"",
				"import Alpha",
				"import Beta",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "first-party prefix matches imports with a trailing '//' comment",
			lineEnding: "\n",
			options:    Options{FirstPartyPrefixes: []string{"MyLib"}},
			input: []string{
				"import MyLib // project internal library",
				"import PlainModule",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import PlainModule",
				"",
				"import MyLib // project internal library",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "leading and trailing whitespace in prefixes is trimmed before use",
			lineEnding: "\n",
			options:    Options{FirstPartyPrefixes: []string{"  MyLib  "}},
			input: []string{
				"import MyLib",
				"import PlainModule",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import PlainModule",
				"",
				"import MyLib",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "first-party prefix matches against whitespace-normalized import text",
			lineEnding: "\n",
			options:    Options{FirstPartyPrefixes: []string{"MyLib 1.0"}},
			input: []string{
				"import  MyLib   1.0",
				"import OtherModule",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import OtherModule",
				"",
				"import MyLib 1.0",
				"",
				"Rectangle {",
				"}",
			},
		},
		{
			name:       "first-party imports are sorted and deduplicated within their group",
			lineEnding: "\n",
			options:    Options{FirstPartyPrefixes: []string{"MyCorp."}},
			input: []string{
				"import QtQuick",
				"import MyCorp.Charlie",
				"import MyCorp.Alpha",
				"import MyCorp.Beta",
				"import MyCorp.Alpha",
				"import io.github.other",
				"",
				"Rectangle {",
				"}",
			},
			expected: []string{
				"import QtQuick",
				"",
				"import io.github.other",
				"",
				"import MyCorp.Alpha",
				"import MyCorp.Beta",
				"import MyCorp.Charlie",
				"",
				"Rectangle {",
				"}",
			},
		},
	})
}
