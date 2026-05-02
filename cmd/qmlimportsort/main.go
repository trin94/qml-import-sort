// SPDX-FileCopyrightText: Elias Mueller
//
// SPDX-License-Identifier: MIT

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/trin94/qml-import-sort/internal/fs"
	"github.com/trin94/qml-import-sort/internal/qml"
)

// version is overridden at build time via -ldflags="-X main.version=...".
var version = "dev"

const usage = `qmlimportsort — sort and group QML imports

USAGE:
   qmlimportsort [flags] <path>...
   qmlimportsort --stdin [flags]

FLAGS:
   --check, -c              Don't write. Print paths that would change to
                            stdout, one per line. Exit 1 if any.
   --stdout                 Don't write. Print formatted content to stdout.
                            Single file only.
   --stdin                  Read from stdin, write to stdout. Mutually
                            exclusive with positional paths.
   --first-party-prefix     Prefix that marks a non-Qt, non-relative
                            import as first-party. Repeatable.
   --version                Print version, exit 0.
   --help, -h               Print this help, exit 0.
`

type stringList []string

func (s *stringList) String() string     { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error { *s = append(*s, v); return nil }

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("qmlimportsort", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() { _, _ = fmt.Fprint(stderr, usage) }

	var (
		check, stdoutMode, stdinMode bool
		showVersion, showHelp, showH bool
		prefixes                     stringList
	)
	flags.BoolVar(&check, "check", false, "")
	flags.BoolVar(&check, "c", false, "")
	flags.BoolVar(&stdoutMode, "stdout", false, "")
	flags.BoolVar(&stdinMode, "stdin", false, "")
	flags.Var(&prefixes, "first-party-prefix", "")
	flags.BoolVar(&showVersion, "version", false, "")
	flags.BoolVar(&showHelp, "help", false, "")
	flags.BoolVar(&showH, "h", false, "")

	if err := flags.Parse(args); err != nil {
		return 2
	}

	if showVersion {
		_, _ = fmt.Fprintln(stdout, version)
		return 0
	}
	if showHelp || showH {
		_, _ = fmt.Fprint(stdout, usage)
		return 0
	}

	paths := flags.Args()

	if check && stdoutMode {
		_, _ = fmt.Fprintln(stderr, "qmlimportsort: --check and --stdout are mutually exclusive")
		return 2
	}
	if stdinMode && len(paths) > 0 {
		_, _ = fmt.Fprintln(stderr, "qmlimportsort: --stdin cannot be combined with positional paths")
		return 2
	}
	if !stdinMode && len(paths) == 0 {
		_, _ = fmt.Fprint(stderr, usage)
		return 2
	}

	classifier, err := qml.Compile(qml.Options{FirstPartyPrefixes: []string(prefixes)})
	if err != nil {
		reportErr(stderr, err)
		return 2
	}

	switch {
	case stdinMode:
		return runStdin(stdin, stdout, stderr, classifier, check)
	case stdoutMode:
		return runStdout(paths, stdout, stderr, classifier)
	case check:
		return runCheck(paths, stdout, stderr, classifier)
	default:
		return runWrite(paths, stderr, classifier)
	}
}

// reportErr writes a formatted error to stderr. The write error itself
// is discarded — there is nowhere meaningful to report a failed stderr
// write.
func reportErr(stderr io.Writer, err error) {
	_, _ = fmt.Fprintf(stderr, "qmlimportsort: %v\n", err)
}

func runStdin(stdin io.Reader, stdout, stderr io.Writer, c *qml.Classifier, check bool) int {
	dst := stdout
	if check {
		dst = io.Discard
	}
	changed, err := fs.FormatStream(stdin, dst, c)
	if err != nil {
		reportErr(stderr, err)
		return 2
	}
	if check && changed {
		return 1
	}
	return 0
}

func runStdout(paths []string, stdout, stderr io.Writer, c *qml.Classifier) int {
	if len(paths) != 1 {
		_, _ = fmt.Fprintln(stderr, "qmlimportsort: --stdout requires exactly one file")
		return 2
	}
	path := paths[0]
	info, err := os.Stat(path)
	if err != nil {
		reportErr(stderr, err)
		return 2
	}
	if info.IsDir() {
		_, _ = fmt.Fprintln(stderr, "qmlimportsort: --stdout requires a file, not a directory")
		return 2
	}
	if err := fs.FormatFileTo(path, stdout, c); err != nil {
		reportErr(stderr, err)
		return 2
	}
	return 0
}

func runCheck(paths []string, stdout, stderr io.Writer, c *qml.Classifier) int {
	anyChanged, anyError, stdoutFailed := false, false, false

	// report records a per-file result. On a write failure to stdout
	// (e.g. broken pipe) it sets stdoutFailed so callers can stop
	// iterating instead of looping over every remaining file.
	report := func(p string, changed bool, err error) {
		if err != nil {
			reportErr(stderr, err)
			anyError = true
			return
		}
		if !changed {
			return
		}
		anyChanged = true
		if _, werr := fmt.Fprintln(stdout, p); werr != nil {
			reportErr(stderr, werr)
			anyError = true
			stdoutFailed = true
		}
	}

	for _, root := range paths {
		if stdoutFailed {
			break
		}
		info, err := os.Stat(root)
		if err != nil {
			report(root, false, err)
			continue
		}
		if !info.IsDir() {
			changed, err := fs.CheckFile(root, c)
			report(root, changed, err)
			continue
		}
		walkErr := fs.WalkQMLFiles(root, func(p string) error {
			if stdoutFailed {
				return filepath.SkipAll
			}
			changed, err := fs.CheckFile(p, c)
			report(p, changed, err)
			return nil
		})
		if walkErr != nil {
			reportErr(stderr, walkErr)
			anyError = true
		}
	}
	if anyError {
		return 2
	}
	if anyChanged {
		return 1
	}
	return 0
}

func runWrite(paths []string, stderr io.Writer, c *qml.Classifier) int {
	anyError := false
	report := func(err error) {
		if err == nil {
			return
		}
		reportErr(stderr, err)
		anyError = true
	}
	for _, root := range paths {
		info, err := os.Stat(root)
		if err != nil {
			report(err)
			continue
		}
		if !info.IsDir() {
			_, err := fs.FormatFile(root, c)
			report(err)
			continue
		}
		walkErr := fs.WalkQMLFiles(root, func(p string) error {
			_, err := fs.FormatFile(p, c)
			report(err)
			return nil
		})
		if walkErr != nil {
			report(walkErr)
		}
	}
	if anyError {
		return 2
	}
	return 0
}
