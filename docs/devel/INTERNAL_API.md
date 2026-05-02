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

`main` imports `internal/fs`. `internal/fs` imports `internal/qml`. `internal/qml` imports only the stdlib's pure packages.

## Why `Compile` is separate from `Format`

The CLI walks many files in one invocation and validates options once. `qml.Compile` does the validation and returns a `*Classifier`; `qml.Format` trusts its input. The CLI calls `Compile` at flag-parse time and treats a `Compile` error as a usage error (exit 2), so a bad `--first-party-prefix` fails before any file is touched.

The full API surface and behavior contract live in the godoc on the `qml` and `fs` packages.

## How the CLI modes compose

Each flag mode in [CLI.md](CLI.md) maps to a small combination of `fs` primitives. `main` contains no formatting logic — only dispatch. It builds a `qml.Options` from `--first-party-prefix` flags, calls `qml.Compile` once, and threads the resulting `*qml.Classifier` through every `fs` call.

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
