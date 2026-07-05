<!--
SPDX-FileCopyrightText: Elias Mueller

SPDX-License-Identifier: MIT
-->

# Internal API Design

## Principles

- **Pure core, I/O shell.** The parsing/formatting logic takes bytes in and returns bytes out. Everything else (files, directories, stdin, atomic writes) is a thin I/O layer on top.
- **Compiler-enforced layering.** The pure core lives in a package that does not import `os` or `io`. The Go compiler prevents accidental coupling.
- **`main` is a dispatcher.** Flag parsing, exit codes, and stderr messages live in `main`. All tool policy (file filters, dotfile skip, atomic writes, line-ending preservation) lives in `internal`.

## Package layout

```
internal/
├── qml/     # Pure: formatting logic. No I/O imports.
└── fs/      # I/O shell: file reads, directory walks, atomic writes, stdin/stdout glue.
```

`main` imports `internal/fs` and `internal/qml` (for `Compile`). `internal/fs` imports `internal/qml`. `internal/qml` imports only the stdlib's pure packages.

## Why `Compile` is separate from `Format`

The CLI walks many files in one invocation and validates options once. `qml.Compile` does the validation and returns a `*Classifier`; `qml.Format` trusts its input. The CLI calls `Compile` at flag-parse time and treats a `Compile` error as a usage error (exit 2), so a bad `--group` fails before any file is touched.

The full API surface and behavior contract live in the godoc on the `qml` and `fs` packages.

## Import classification and grouping

The output order of the import block is fixed: pragmas, Qt, default, custom `--group` sections (in flag order), relative. Only the custom sections are configurable.

Classification does not depend on flag order — it is decided per import:

- **Relative** matches by syntax: the import target is a quoted path.
- **Qt** matches modules shipped with Qt: names starting with `Qt` followed by an uppercase letter, digit, or dot (`QtQuick`, `Qt.labs.settings`, `Qt5Compat.GraphicalEffects`), plus the base language module `QML` and its `QML.` subpaths.
- **Custom sections** match by the longest matching prefix across *all* `--group` flags. A prefix is a literal match against the whitespace-normalized text after `import`; a trailing `.` creates a namespace boundary (`io.github.mpvqc.` matches `io.github.mpvqc.Foo` but not `io.github.mpvqcExternal`). Prefixes are trimmed of surrounding whitespace.
- **Default** takes whatever is left.

Validation happens in `qml.Compile`, so a bad group definition is a usage error (exit 2) before any file is touched:

- a group with no prefixes at all
- empty prefix
- a prefix that can never match an import: internal tabs or runs of spaces (import text is normalized to single spaces), or a module-name part that is not a valid QML-name prefix (ASCII letter or underscore first; then letters, digits, underscores, dots)
- Qt-reserved prefix: starting with `Qt` or `qt`, equal to `QML`, or starting with `QML.` — the Qt/QML namespace cannot be grouped
- the same prefix listed twice anywhere

Overlapping (non-identical) prefixes are legal — the longest match wins, and because exact duplicates are rejected there are no ties.

The repeatable flag maps 1:1 onto a list in the planned YAML/TOML config file:

```yaml
groups:
  - prefixes: [Company.Shared., Company.Widgets.]
  - prefixes: [MyApp.]
```

## How the CLI modes compose

Each flag mode in [CLI.md](CLI.md) maps to a small combination of `fs` primitives. `main` contains no formatting logic — only dispatch. It builds a `qml.Options` from `--group` flags, calls `qml.Compile` once, and threads the resulting `*qml.Classifier` through every `fs` call.

| Mode                            | Composition                                                                                               |
| ------------------------------- | --------------------------------------------------------------------------------------------------------- |
| `qmlimportsort a.qml`           | `fs.FormatFile("a.qml", c)`                                                                               |
| `qmlimportsort src/`            | `fs.WalkQMLFiles("src/", func(p) { _, err := fs.FormatFile(p, c); return err })`                          |
| `qmlimportsort --check src/`    | `fs.WalkQMLFiles("src/", func(p) { if ch, _ := fs.CheckFile(p, c); ch { print(p); anyChanged = true } })` |
| `qmlimportsort --stdout a.qml`  | `fs.FormatFileTo("a.qml", os.Stdout, c)`                                                                  |
| `qmlimportsort --stdin`         | `fs.FormatStream(os.Stdin, os.Stdout, c)`                                                                 |
| `qmlimportsort --stdin --check` | `changed, _ := fs.FormatStream(os.Stdin, io.Discard, c)` — exit 1 if `changed`                            |

## Open points deliberately not specified

- **Concrete error types / sentinel errors**: not needed yet. If CI tooling grows to want to distinguish parse errors from IO errors, we can introduce `var ErrParse = errors.New(...)` later.
- **Streaming / chunked formatting**: not needed. QML files are small; read fully into memory.
- **Context / cancellation**: no long-running operations; no `context.Context` in the API.
