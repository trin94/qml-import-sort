<!--
SPDX-FileCopyrightText: Elias Mueller

SPDX-License-Identifier: MIT
-->

# Internal API Design

## 2026-04-24

### Principles

- **Pure core, I/O shell.** The parsing/formatting logic takes bytes in and returns bytes out. Everything else (files, directories, stdin, atomic writes) is a thin I/O layer on top.
- **Compiler-enforced layering.** The pure core lives in a package that does not import `os` or `io`. The Go compiler prevents accidental coupling.
- **`main` is a dispatcher.** Flag parsing, exit codes, and stderr messages live in `main`. All tool policy (file filters, dotfile skip, atomic writes, line-ending preservation) lives in `internal`.

### Package layout

```
internal/
├── qml/     # Pure: formatting logic. No I/O imports.
└── fs/      # I/O shell: file reads, directory walks, atomic writes, stdin/stdout glue.
```

`main` imports `internal/fs`. `internal/fs` imports `internal/qml`. `internal/qml` imports only the stdlib's pure packages (`strings`, `regexp`, `sort`/`slices`, `bytes`, `errors`, `fmt`).

______________________________________________________________________

### Package `internal/qml`

Exports two struct types and three functions. All parsing, classification, and reassembly logic stays unexported.

```go
type Options struct {
    FirstPartyPrefixes []string
}

type Classifier struct { /* unexported */ }

func Compile(opts Options) (*Classifier, error)
func MustCompile(opts Options) *Classifier
func Format(src []byte, c *Classifier) ([]byte, error)
```

`Options` is the user-facing knob. `Classifier` is the validated, immutable form built by `Compile`; pass it to `Format`. A nil `*Classifier` is equivalent to one compiled from `Options{}`, i.e. defaults only.

The split exists so that batch processing (the CLI walks many files in one invocation) validates options once and reuses the result. `Compile` returns the validation error; `Format` trusts its input. The CLI calls `Compile` at flag-parse time and treats a `Compile` error as a usage error (exit 2), so a bad `--first-party-prefix` fails before any file is touched.

`Options{}` (zero value) puts every non-Qt, non-relative import into a single third-party group. `FirstPartyPrefixes` lets the caller carve a first-party group out of that default — caller-supplied strings that promote any import whose normalized text starts with one of them into the first-party category. Include a trailing `.` to create a boundary (e.g. `MyCorp.` matches `MyCorp.Foo` but not `MyCorpX`).

Prefixes are trimmed of leading and trailing whitespace before validation and matching — `"  MyLib  "` is treated as `"MyLib"`. Trimming happens before every rule below, so an all-whitespace prefix is rejected as empty.

`Compile` returns an error (naming the offending prefix) if any rule is violated:

- No prefix may be the empty string (after trimming).
- No prefix may start with `.`.
- No prefix may start with `Qt` or `qt` (Qt imports are their own category and not overridable via prefix).
- No prefix may equal `pragma` (pragma is a separate category, not overridable).
- No two prefixes in `FirstPartyPrefixes` may **overlap** — i.e. be identical, or one a prefix of the other. Identical duplicates are reported as `"duplicate"`; non-identical overlaps as `"overlapping"`.

**Classification precedence**: pragma → qt → relative → `FirstPartyPrefixes` match → third-party (everything else) → error (when the line is not a valid pragma, import, comment, or blank). The model is semantic, not syntactic: there is no dotted-vs-bare heuristic. A bare identifier and a dotted path are both third-party by default; the caller decides which (if any) become first-party.

The full behavior contract — block boundaries, category ordering, sorting, deduplication, comment and blank-line handling, whitespace normalization, error cases — lives in the godoc on `Format` in [`internal/qml/format.go`](../../internal/qml/format.go). The prose below covers the _implementation_ approach, not the contract.

The package will need unexported helpers for (a) detecting the input's line ending, (b) locating the pragma/import block within the document, (c) classifying each line into one of the import categories (pragma, Qt, third-party, first-party, relative), and (d) reassembling the categories back into output bytes. The exact shape of these helpers is an implementation detail and not part of the API contract.

**Implementation approach: line tokenizer.** A single pass over the import block produces a slice of tokens of the form `{kind, text}`, where `kind` is one of the five categories, plus a separate flat slice of comment lines collected from within the block. Blank lines inside the block are dropped during tokenization. Sorting and grouping operate on the token slice; comments are emitted as a hoisted section between preamble and imports, in input order.

Comments inside the block do **not** travel with imports. Comments between pragma/import lines are accumulated in input order and emitted as a single comment section between the preamble and the import groups. Comments before the first pragma/import are preamble, and comments after the last are body — those positions are unchanged. To annotate a specific import, use a trailing `// note` on the import line itself (it becomes part of the line text and stays attached through sorting).

Blank lines inside the block are dropped — categories are always emitted tightly packed. The blank-line treatment lives entirely in the structural separators between sections.

Structural separators around and between sections are always normalized:

- Exactly one blank line between the preamble and the next non-empty section (the hoisted-comment section if any, otherwise the imports), when the preamble is non-empty.
- Exactly one blank line between the hoisted-comment section and the imports (when both are non-empty).
- Exactly one blank line between adjacent non-empty category groups.
- Exactly one blank line between the block and the body (when the body is non-empty).

Multiple blank lines in any of these separator positions in the input collapse to one. Blank lines *inside* the preamble (between comment lines, before trailing blanks) are preserved byte-for-byte — the preamble is passthrough except for its trailing blank lines, which are part of the preamble-to-block separator and get normalized.

**Leading blank lines at the top of the file** are stripped entirely. The preamble effectively starts at the first non-blank line; nothing meaningful lives above it.

**Block boundaries**: the pragma/import block starts at the first pragma or import line and ends at the **last** pragma or import line (inclusive). Comments that appear before the first pragma/import are preamble (this is what keeps license headers put regardless of whether the user included a blank line between them and the imports). Comments between pragma/import lines are hoisted out of the block — see the hoist rule below. Comments after the last pragma/import are body.

**Duplicates are removed.** Two tokens are considered duplicates when their normalized `text` is byte-identical (same kind, same whitespace-normalized import/pragma line, same alias if any). The first occurrence wins; subsequent occurrences are dropped from the token slice. Comments are independent of duplicate detection — every comment line collected from inside the block goes into the hoisted section in input order, even if it sat above a duplicate import. Applies to both pragmas and imports.

**Whitespace inside lines** is handled per line-kind:

- **Pragma and import lines**: classified after trimming leading whitespace, so indented imports are recognized as imports. On emit they are rendered canonical — no leading whitespace, no trailing whitespace, and runs of internal whitespace (spaces or tabs between tokens) collapsed to single spaces. `import  QtQuick   2.15` and `import\tQtQuick\t2.15` both become `import QtQuick 2.15`. Whitespace inside quoted strings is preserved — quoted paths are single tokens, so `import    "./my  file.qml"` becomes `import "./my  file.qml"` with the internal double-space intact. A trailing comment on the same line as an import or pragma is kept as part of that line's text; it participates in sorting and in whitespace collapse. Users who want a comment to travel independently should put it on its own line.
- **Line comments (`//`)**: trimmed and emitted canonical. Leading whitespace is stripped; an indented `// note` becomes just `// note`.
- **Block-comment lines** — those whose trimmed content starts with `/*`, `*`, or `*/`: preserved byte-for-byte, including leading whitespace. This is what keeps multi-line block comment `*` alignment intact.
- **Whitespace-only lines** (any mix of spaces and tabs, nothing else) count as blank lines and feed into the blank-line rules above.

**Why this shape**

- Bytes-in/bytes-out is the simplest possible contract. Any caller that can produce bytes (file, stdin, in-memory buffer, test fixture) can use it.
- Line-ending detection is content inspection, not I/O — it belongs with the pure core.
- Single export forces the public surface to stay small. Anything else is an implementation detail.

______________________________________________________________________

### Package `internal/fs`

The I/O shell. Wraps `qml.Format` with the file operations the CLI needs.

```go
// FormatStream reads QML content from src, formats it via qml.Format,
// and writes the result to dst. c is forwarded to qml.Format.
// Returns (changed, err) where changed reports whether the formatted
// output differs byte-for-byte from the input.
//
// Used by: --stdin (dst = os.Stdout), --stdin --check (dst = io.Discard).
func FormatStream(src io.Reader, dst io.Writer, c *qml.Classifier) (changed bool, err error)

// FormatFile formats path in place using an atomic write (temp file
// in the same directory + rename). c is forwarded to qml.Format.
// Returns (changed, err) where changed reports whether the file's
// content on disk differs after formatting.
// File mode is preserved across the rename.
//
// Used by: default write mode.
func FormatFile(path string, c *qml.Classifier) (changed bool, err error)

// CheckFile reports whether formatting path would change its content.
// Does not write. c is forwarded to qml.Format.
//
// Used by: --check.
func CheckFile(path string, c *qml.Classifier) (wouldChange bool, err error)

// FormatFileTo reads path, formats it, writes to dst.
// Does not modify the file on disk. c is forwarded to qml.Format.
//
// Used by: --stdout.
func FormatFileTo(path string, dst io.Writer, c *qml.Classifier) error

// WalkQMLFiles walks root recursively, calling fn(path) for each regular
// file whose name ends in ".qml". Entries whose name begins with "."
// are skipped during descent. Symlinks are not followed.
//
// The root argument itself is processed regardless of a leading dot
// (explicit paths bypass the skip rule).
//
// If fn returns an error, the walk stops and that error is returned.
//
// Used by: all modes that accept directory arguments.
func WalkQMLFiles(root string, fn func(path string) error) error
```

**Unexported helpers inside `fs`**:

- `writeAtomic(path string, data []byte, mode os.FileMode) error` — temp file + rename.
- `readAndFormat([]byte) (out []byte, changed bool, err error)` — shared by `FormatStream`/`FormatFile`/`CheckFile` to avoid duplicating the "did anything change?" comparison.

**Error handling**:

- All exported `fs` functions wrap path/IO errors with `fmt.Errorf("%s: %w", path, err)` so callers can use `errors.Is` / `errors.As`.
- Parse errors from `qml.Format` are wrapped with the path as well.

______________________________________________________________________

### How the CLI modes compose

Each flag mode in [CLI.md](CLI.md) maps to a small combination of the primitives above. `main` contains no formatting logic — only dispatch. `main` builds a `qml.Options` from `--first-party-prefix` flags, calls `qml.Compile` once at startup, and threads the resulting `*qml.Classifier` through every fs call in the invocation. A `Compile` error becomes a usage error (exit 2) before any file is opened.

| Mode                            | Composition                                                                                              |
| ------------------------------- | -------------------------------------------------------------------------------------------------------- |
| `qmlimportsort a.qml`           | `fs.FormatFile("a.qml", c)`                                                                              |
| `qmlimportsort src/`            | `fs.WalkQMLFiles("src/", func(p) { _, err := fs.FormatFile(p, c); return err })`                         |
| `qmlimportsort --check src/`    | `fs.WalkQMLFiles("src/", func(p) { if ch, _ := fs.CheckFile(p, c); ch { print(p); anyChanged = true } })` |
| `qmlimportsort --stdout a.qml`  | `fs.FormatFileTo("a.qml", os.Stdout, c)`                                                                 |
| `qmlimportsort --stdin`         | `fs.FormatStream(os.Stdin, os.Stdout, c)`                                                                |
| `qmlimportsort --stdin --check` | `changed, _ := fs.FormatStream(os.Stdin, io.Discard, c)` — exit 1 if `changed`                           |

______________________________________________________________________

### Implementation note

The refactor is a fresh rewrite — no code from the current `internal/` or `cmd/` tree is being ported over. The existing files (`internal/qml.go`, `internal/orchestrator.go`, `internal/document.go`, `internal/file.go`, `internal/qml_test.go`, `cmd/qmlimportsort/main.go`) will be replaced wholesale. Tests are rewritten against the new API surface, not adapted from the old ones.

______________________________________________________________________

### Open points deliberately not specified here

- **Concrete error types / sentinel errors**: not needed yet. If CI tooling grows to want to distinguish parse errors from IO errors, we can introduce `var ErrParse = errors.New(...)` later.
- **Streaming / chunked formatting**: not needed. QML files are small; read fully into memory.
- **Context / cancellation**: no long-running operations; no `context.Context` in the API.
