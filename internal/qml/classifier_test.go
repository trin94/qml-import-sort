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
			name:            "empty prefix is rejected",
			opts:            Options{FirstPartyPrefixes: []string{""}},
			wantErrContains: "empty prefix",
		},
		{
			name:            "all-whitespace prefix is trimmed to empty and rejected",
			opts:            Options{FirstPartyPrefixes: []string{"   "}},
			wantErrContains: "empty prefix",
		},
		{
			name:            "prefix starting with '.' is rejected",
			opts:            Options{FirstPartyPrefixes: []string{".foo"}},
			wantErrContains: `".foo"`,
		},
		{
			name:            "prefix equal to 'pragma' is rejected",
			opts:            Options{FirstPartyPrefixes: []string{"pragma"}},
			wantErrContains: `"pragma"`,
		},
		{
			name:            "prefix starting with 'Qt' is rejected",
			opts:            Options{FirstPartyPrefixes: []string{"QtCustom"}},
			wantErrContains: `"QtCustom"`,
		},
		{
			name:            "prefix starting with 'qt' is rejected",
			opts:            Options{FirstPartyPrefixes: []string{"qtcustom"}},
			wantErrContains: `"qtcustom"`,
		},
		{
			name:            "prefix with surrounding whitespace is trimmed before the Qt-start check",
			opts:            Options{FirstPartyPrefixes: []string{"  QtCustom  "}},
			wantErrContains: `"QtCustom"`,
		},
		{
			name:            "prefix equal to 'QML' is rejected",
			opts:            Options{FirstPartyPrefixes: []string{"QML"}},
			wantErrContains: `"QML"`,
		},
		{
			name:            "prefix starting with 'QML.' is rejected",
			opts:            Options{FirstPartyPrefixes: []string{"QML.Foo"}},
			wantErrContains: `"QML.Foo"`,
		},
		{
			name: "prefix merely starting with QML is accepted",
			opts: Options{FirstPartyPrefixes: []string{"QMLFoo"}},
		},
		{
			name:            "duplicate prefix is rejected and names the prefix",
			opts:            Options{FirstPartyPrefixes: []string{"Foo", "Foo"}},
			wantErrContains: `"Foo"`,
		},
		{
			name:            "prefix-of-prefix overlap is rejected and names both (shorter first)",
			opts:            Options{FirstPartyPrefixes: []string{"Foo", "Foo.Bar"}},
			wantErrContains: `"Foo.Bar"`,
		},
		{
			name:            "prefix-of-prefix overlap is detected regardless of order (longer first)",
			opts:            Options{FirstPartyPrefixes: []string{"Foo.Bar", "Foo"}},
			wantErrContains: `"Foo.Bar"`,
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
