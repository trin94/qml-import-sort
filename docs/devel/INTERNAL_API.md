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

Exports exactly one function. All parsing, classification, and reassembly logic stays unexported.

```go
// Format sorts and groups QML imports in src, returning the formatted bytes.
// The input's line endings (\n, \r\n, or \r) are detected and preserved
// in the output. src is not modified.
//
// Comments inside the import block are preserved: each comment line is
// attached to the following import and travels with it through sorting.
//
// A file with no imports (and no pragmas) is a valid input and is
// returned unchanged.
//
// Returns an error if the input cannot be parsed — specifically, if a
// line inside the pragma/import block cannot be classified as a pragma,
// import, blank line, or comment.
func Format(src []byte) ([]byte, error)
```

The package will need unexported helpers for (a) detecting the input's line ending, (b) locating the pragma/import block within the document, (c) classifying each line into one of the import categories (pragma, Qt, library, module, relative), and (d) reassembling the categories back into output bytes. The exact shape of these helpers is an implementation detail and not part of the API contract.

**Implementation approach: line tokenizer.** A single pass over the import block produces a slice of tokens of the form `{kind, text, leadingComments []string}`, where `kind` is one of the five categories. Sorting and grouping operate on that slice, and output is reassembled by walking the sorted tokens.

Comments inside the block are preserved: while scanning, contiguous comment lines accumulate into a buffer and attach as `leadingComments` to the next import or pragma token. On emit, each token writes its leading comments (in original order) before the import line itself. Sorting operates only on `token.text`, so comments travel with their import to its final sorted position.

**Why this shape**

- Bytes-in/bytes-out is the simplest possible contract. Any caller that can produce bytes (file, stdin, in-memory buffer, test fixture) can use it.
- Line-ending detection is content inspection, not I/O — it belongs with the pure core.
- Single export forces the public surface to stay small. Anything else is an implementation detail.

______________________________________________________________________

### Package `internal/fs`

The I/O shell. Wraps `qml.Format` with the file operations the CLI needs.

```go
// FormatStream reads QML content from src, formats it via qml.Format,
// and writes the result to dst.
// Returns (changed, err) where changed reports whether the formatted
// output differs byte-for-byte from the input.
//
// Used by: --stdin (dst = os.Stdout), --stdin --check (dst = io.Discard).
func FormatStream(src io.Reader, dst io.Writer) (changed bool, err error)

// FormatFile formats path in place using an atomic write (temp file
// in the same directory + rename).
// Returns (changed, err) where changed reports whether the file's
// content on disk differs after formatting.
// File mode is preserved across the rename.
//
// Used by: default write mode.
func FormatFile(path string) (changed bool, err error)

// CheckFile reports whether formatting path would change its content.
// Does not write.
//
// Used by: --check.
func CheckFile(path string) (wouldChange bool, err error)

// FormatFileTo reads path, formats it, writes to dst.
// Does not modify the file on disk.
//
// Used by: --stdout.
func FormatFileTo(path string, dst io.Writer) error

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

Each flag mode in [CLI.md](CLI.md) maps to a small combination of the primitives above. `main` contains no formatting logic — only dispatch.

| Mode                            | Composition                                                                                                                                                                  |
| ------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `qmlimportsort a.qml`           | `fs.FormatFile("a.qml")`                                                                                                                                                     |
| `qmlimportsort src/`            | `fs.WalkQMLFiles("src/", fs.FormatFile)` — except `fn` needs to match `func(string) error`, so wrap: `fn := func(p string) error { _, err := fs.FormatFile(p); return err }` |
| `qmlimportsort --check src/`    | `fs.WalkQMLFiles("src/", func(p) { if change, _ := fs.CheckFile(p); change { print(p); anyChanged = true } })`                                                               |
| `qmlimportsort --stdout a.qml`  | `fs.FormatFileTo("a.qml", os.Stdout)`                                                                                                                                        |
| `qmlimportsort --stdin`         | `fs.FormatStream(os.Stdin, os.Stdout)`                                                                                                                                       |
| `qmlimportsort --stdin --check` | `changed, _ := fs.FormatStream(os.Stdin, io.Discard)` — exit 1 if `changed`                                                                                                  |

______________________________________________________________________

### Implementation note

The refactor is a fresh rewrite — no code from the current `internal/` or `cmd/` tree is being ported over. The existing files (`internal/qml.go`, `internal/orchestrator.go`, `internal/document.go`, `internal/file.go`, `internal/qml_test.go`, `cmd/qmlimportsort/main.go`) will be replaced wholesale. Tests are rewritten against the new API surface, not adapted from the old ones.

______________________________________________________________________

### Open points deliberately not specified here

- **Concrete error types / sentinel errors**: not needed yet. If CI tooling grows to want to distinguish parse errors from IO errors, we can introduce `var ErrParse = errors.New(...)` later.
- **Streaming / chunked formatting**: not needed. QML files are small; read fully into memory.
- **Context / cancellation**: no long-running operations; no `context.Context` in the API.
