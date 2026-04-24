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

Exports one function and one struct. All parsing, classification, and reassembly logic stays unexported.

```go
type Options struct {
    LibraryPrefixes []string
    ModulePrefixes  []string
}

func Format(src []byte, opts Options) ([]byte, error)
```

`Options{}` (zero value) applies only the default heuristics. `LibraryPrefixes` and `ModulePrefixes` are literal prefix overrides on the default classifier — caller-supplied strings that force a specific category when the import text starts with any of them. Include a trailing `.` to create a boundary (e.g. `MyCorp.` matches `MyCorp.Foo` but not `MyCorpX`).

Prefixes are trimmed of leading and trailing whitespace before validation and matching — `"  MyLib  "` is treated as `"MyLib"`. Trimming happens before every rule below, so an all-whitespace prefix is rejected as empty.

Format validates the Options at entry and returns an error (naming the offending prefix) if any rule is violated:

- No prefix may be the empty string (after trimming).
- No prefix may start with `.`.
- No prefix may start with `Qt` or `qt` (Qt imports are their own category and not overridable via prefix).
- No prefix may equal `pragma` (pragma is a separate category, not overridable).
- No two prefixes may **overlap** — i.e. be identical, or one a prefix of the other — whether within the same list or across `LibraryPrefixes` and `ModulePrefixes`. Identical duplicates are reported as `"duplicate"`; non-identical overlaps as `"overlapping"`.

These rules are enforced uniformly: a conflict between two entries in the same list and a conflict across the two lists are both errors with the same reasoning — "either prefix could match the same import, and there's no sensible default for which wins."

**Classification precedence**: pragma → qt → user `LibraryPrefixes` → user `ModulePrefixes` → default library (dotted) → default module (bare) → relative → error. User prefixes are checked *before* the default dot/bare heuristics so callers can both promote (bare → library) and demote (dotted → module) symmetrically.

The full behavior contract — block boundaries, category ordering, sorting, deduplication, comment and blank-line handling, whitespace normalization, error cases — lives in the godoc on `Format` in [`internal/qml/format.go`](../../internal/qml/format.go). The prose below covers the _implementation_ approach, not the contract.

The package will need unexported helpers for (a) detecting the input's line ending, (b) locating the pragma/import block within the document, (c) classifying each line into one of the import categories (pragma, Qt, library, module, relative), and (d) reassembling the categories back into output bytes. The exact shape of these helpers is an implementation detail and not part of the API contract.

**Implementation approach: line tokenizer.** A single pass over the import block produces a slice of tokens of the form `{kind, text, leadingComments []string, leadingBlank bool}`, where `kind` is one of the five categories. Sorting and grouping operate on that slice, and output is reassembled by walking the sorted tokens.

Comments inside the block are preserved: while scanning, contiguous comment lines accumulate into a buffer and attach as `leadingComments` to the next import or pragma token. On emit, each token writes its leading comments (in original order) before the import line itself. Sorting operates only on `token.text`, so comments travel with their import to its final sorted position.

Blank lines inside the block are preserved in spirit, not position. While scanning, a blank line sets `leadingBlank = true` on the next non-blank token (multiple consecutive blanks collapse to a single `true`). On emit, blank lines within a category follow an **all-or-nothing** rule: if *any* token in the category had `leadingBlank = true` in the input, the output inserts a blank line between every adjacent pair of tokens in that category. Otherwise, the category is emitted tightly packed. This captures the user's intent ("I want breathing room in this group") without trying to preserve exact pre-sort positions, which become ambiguous after reordering.

Structural separators around and between categories are independent of `leadingBlank` and always normalized:

- Exactly one blank line between the preamble and the block (if the preamble is non-empty).
- Exactly one blank line between adjacent category groups.
- Exactly one blank line between the block and the body (if the body is non-empty).

Multiple blank lines in any of these separator positions in the input collapse to one. Blank lines *inside* the preamble (between comment lines, before trailing blanks) are preserved byte-for-byte — the preamble is passthrough except for its trailing blank lines, which are part of the preamble-to-block separator and get normalized.

**Leading blank lines at the top of the file** are stripped entirely. The preamble effectively starts at the first non-blank line; nothing meaningful lives above it.

**Block boundaries**: the pragma/import block starts at the first pragma or import line and ends at the **last** pragma or import line (inclusive). Comments that appear before the first pragma/import are preamble (this is what keeps license headers put regardless of whether the user included a blank line between them and the imports). Comments between pragma/import lines attach as leading comments to the following import or pragma. Comments after the last pragma/import are body. Under this rule, an "orphan" comment — one with no following import — never exists inside the block; it is always either preamble or body.

**Duplicates are removed.** Two tokens are considered duplicates when their normalized `text` is byte-identical (same kind, same whitespace-normalized import/pragma line, same alias if any). The first occurrence wins and keeps its `leadingComments` and `leadingBlank`; subsequent occurrences are dropped entirely, including any comments attached to them. Applies to both pragmas and imports.

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
// and writes the result to dst. opts is forwarded to qml.Format.
// Returns (changed, err) where changed reports whether the formatted
// output differs byte-for-byte from the input.
//
// Used by: --stdin (dst = os.Stdout), --stdin --check (dst = io.Discard).
func FormatStream(src io.Reader, dst io.Writer, opts qml.Options) (changed bool, err error)

// FormatFile formats path in place using an atomic write (temp file
// in the same directory + rename). opts is forwarded to qml.Format.
// Returns (changed, err) where changed reports whether the file's
// content on disk differs after formatting.
// File mode is preserved across the rename.
//
// Used by: default write mode.
func FormatFile(path string, opts qml.Options) (changed bool, err error)

// CheckFile reports whether formatting path would change its content.
// Does not write. opts is forwarded to qml.Format.
//
// Used by: --check.
func CheckFile(path string, opts qml.Options) (wouldChange bool, err error)

// FormatFileTo reads path, formats it, writes to dst.
// Does not modify the file on disk. opts is forwarded to qml.Format.
//
// Used by: --stdout.
func FormatFileTo(path string, dst io.Writer, opts qml.Options) error

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

Each flag mode in [CLI.md](CLI.md) maps to a small combination of the primitives above. `main` contains no formatting logic — only dispatch. `main` builds a `qml.Options` from `--library-prefix` / `--module-prefix` flags and threads the same `opts` value through every fs call in one invocation.

| Mode                            | Composition                                                                                                |
| ------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `qmlimportsort a.qml`           | `fs.FormatFile("a.qml", opts)`                                                                             |
| `qmlimportsort src/`            | `fs.WalkQMLFiles("src/", func(p) { _, err := fs.FormatFile(p, opts); return err })`                        |
| `qmlimportsort --check src/`    | `fs.WalkQMLFiles("src/", func(p) { if c, _ := fs.CheckFile(p, opts); c { print(p); anyChanged = true } })` |
| `qmlimportsort --stdout a.qml`  | `fs.FormatFileTo("a.qml", os.Stdout, opts)`                                                                |
| `qmlimportsort --stdin`         | `fs.FormatStream(os.Stdin, os.Stdout, opts)`                                                               |
| `qmlimportsort --stdin --check` | `changed, _ := fs.FormatStream(os.Stdin, io.Discard, opts)` — exit 1 if `changed`                          |

______________________________________________________________________

### Implementation note

The refactor is a fresh rewrite — no code from the current `internal/` or `cmd/` tree is being ported over. The existing files (`internal/qml.go`, `internal/orchestrator.go`, `internal/document.go`, `internal/file.go`, `internal/qml_test.go`, `cmd/qmlimportsort/main.go`) will be replaced wholesale. Tests are rewritten against the new API surface, not adapted from the old ones.

______________________________________________________________________

### Open points deliberately not specified here

- **Concrete error types / sentinel errors**: not needed yet. If CI tooling grows to want to distinguish parse errors from IO errors, we can introduce `var ErrParse = errors.New(...)` later.
- **Streaming / chunked formatting**: not needed. QML files are small; read fully into memory.
- **Context / cancellation**: no long-running operations; no `context.Context` in the API.
