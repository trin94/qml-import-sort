// SPDX-FileCopyrightText: Elias Mueller
//
// SPDX-License-Identifier: MIT

package qml

import (
	"strings"
	"testing"
)

func TestCompile(t *testing.T) {
	cases := []struct {
		name            string
		opts            Options
		wantErrContains string
	}{
		{
			name: "no groups is valid",
			opts: Options{},
		},
		{
			name:            "empty prefix is rejected",
			opts:            Options{Groups: [][]string{{""}}},
			wantErrContains: "empty prefix",
		},
		{
			name:            "all-whitespace prefix is trimmed to empty and rejected",
			opts:            Options{Groups: [][]string{{"   "}}},
			wantErrContains: "empty prefix",
		},
		{
			name:            "group without prefixes is rejected",
			opts:            Options{Groups: [][]string{{}}},
			wantErrContains: "no prefixes",
		},
		{
			name:            "prefix starting with '.' is rejected",
			opts:            Options{Groups: [][]string{{".foo"}}},
			wantErrContains: `".foo"`,
		},
		{
			name:            "prefix starting with 'Qt' is rejected",
			opts:            Options{Groups: [][]string{{"QtCustom"}}},
			wantErrContains: `"QtCustom"`,
		},
		{
			name:            "prefix starting with 'qt' is rejected",
			opts:            Options{Groups: [][]string{{"qtcustom"}}},
			wantErrContains: `"qtcustom"`,
		},
		{
			name:            "prefix equal to 'QML' is rejected",
			opts:            Options{Groups: [][]string{{"QML"}}},
			wantErrContains: `"QML"`,
		},
		{
			name:            "prefix starting with 'QML.' is rejected",
			opts:            Options{Groups: [][]string{{"QML.Foo"}}},
			wantErrContains: `"QML.Foo"`,
		},
		{
			name: "prefix merely starting with QML is accepted",
			opts: Options{Groups: [][]string{{"QMLFoo"}}},
		},
		{
			name:            "prefix with surrounding whitespace is trimmed before the Qt-start check",
			opts:            Options{Groups: [][]string{{"  QtCustom  "}}},
			wantErrContains: `"QtCustom"`,
		},
		{
			name:            "duplicate prefix within a group is rejected",
			opts:            Options{Groups: [][]string{{"Foo", "Foo"}}},
			wantErrContains: `"Foo"`,
		},
		{
			name:            "duplicate prefix across groups is rejected",
			opts:            Options{Groups: [][]string{{"Foo"}, {"Foo"}}},
			wantErrContains: `"Foo"`,
		},
		{
			name:            "duplicates are detected after trimming",
			opts:            Options{Groups: [][]string{{"Foo"}, {"  Foo  "}}},
			wantErrContains: `"Foo"`,
		},
		{
			name: "overlapping prefixes in one group are accepted",
			opts: Options{Groups: [][]string{{"Foo.", "Foo.Bar."}}},
		},
		{
			name: "overlapping prefixes across groups are accepted",
			opts: Options{Groups: [][]string{{"Foo."}, {"Foo.Bar."}}},
		},
		{
			name:            "Qt-reserved rejection names the namespace policy",
			opts:            Options{Groups: [][]string{{"QtCustom"}}},
			wantErrContains: "reserved",
		},
		{
			name:            "prefix with a run of spaces can never match and is rejected",
			opts:            Options{Groups: [][]string{{"My  Lib"}}},
			wantErrContains: `"My  Lib"`,
		},
		{
			name:            "prefix containing a tab can never match and is rejected",
			opts:            Options{Groups: [][]string{{"My\tLib"}}},
			wantErrContains: "can never match",
		},
		{
			name:            "prefix with non-ASCII module name can never match and is rejected",
			opts:            Options{Groups: [][]string{{"Café."}}},
			wantErrContains: `"Café."`,
		},
		{
			name:            "prefix starting with a digit can never match and is rejected",
			opts:            Options{Groups: [][]string{{"9foo"}}},
			wantErrContains: `"9foo"`,
		},
		{
			name:            "prefix starting with a quote can never match and is rejected",
			opts:            Options{Groups: [][]string{{`"x`}}},
			wantErrContains: "can never match",
		},
		{
			name: "prefix 'pragma' is accepted since a module may be named pragma",
			opts: Options{Groups: [][]string{{"pragma"}}},
		},
		{
			name: "underscore-leading prefix is accepted",
			opts: Options{Groups: [][]string{{"_private."}}},
		},
		{
			name: "prefix extending past the module name is accepted",
			opts: Options{Groups: [][]string{{"MyLib 1.0"}}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Compile(tc.opts)
			if tc.wantErrContains == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErrContains)
			}
			if !strings.Contains(err.Error(), tc.wantErrContains) {
				t.Errorf("error message %q does not contain %q", err.Error(), tc.wantErrContains)
			}
		})
	}
}
