<!--
SPDX-FileCopyrightText: Elias Mueller

SPDX-License-Identifier: MIT
-->

# CLI Design

## 2026-04-24

### Synopsis

```
qmlimportsort [flags] <path>...
qmlimportsort --stdin [flags]
```

Each `<path>` is a file or directory. Directories are walked recursively.

### Invocations

| Command                        | Behavior                                                         |
| ------------------------------ | ---------------------------------------------------------------- |
| `qmlimportsort a.qml`          | Format `a.qml` in place.                                         |
| `qmlimportsort a.qml b.qml`    | Format both in place.                                            |
| `qmlimportsort src/`           | Recurse under `src/`, format every `*.qml` file in place.        |
| `qmlimportsort src/ main.qml`  | Mix: format files under `src/` and `main.qml` in place.          |
| `qmlimportsort --stdin`        | Read stdin, write formatted content to stdout.                   |
| `qmlimportsort --check src/`   | Dry-run. Print paths that would change to stdout. Exit 1 if any. |
| `qmlimportsort --stdout a.qml` | Print formatted content of `a.qml` to stdout. Don't write.       |
| `qmlimportsort --version`      | Print version, exit 0.                                           |
| `qmlimportsort --help`         | Print usage, exit 0.                                             |
| `qmlimportsort` (no args)      | Print usage to stderr, exit 2.                                   |

### Flags

| Flag               | Short | Purpose                                                                                                        |
| ------------------ | ----- | -------------------------------------------------------------------------------------------------------------- |
| `--check`          | `-c`  | Don't write. Print paths that would change to stdout, one per line. Exit 1 if any.                             |
| `--stdout`         |       | Don't write. Print formatted content to stdout. Single file only (see restrictions).                           |
| `--stdin`          |       | Read from stdin, write to stdout. Mutually exclusive with positional paths.                                    |
| `--library-prefix` |       | Additional prefix to classify as a library import. Repeatable. Overrides the default dot-heuristic.            |
| `--module-prefix`  |       | Additional prefix to classify as a module import. Repeatable. Overrides the default bare-identifier heuristic. |
| `--version`        |       | Print version, exit 0.                                                                                         |
| `--help`           | `-h`  | Print usage, exit 0.                                                                                           |

### Flag combinations

- `--check` and `--stdout` are mutually exclusive (usage error, exit 2).
- `--stdin` is mutually exclusive with positional paths (usage error, exit 2).
- `--stdin` combined with `--check`: dry-run on stdin content. Exit 1 if it would change, 0 otherwise. No output on
  stdout.
- `--stdin` combined with `--stdout`: redundant but allowed — stdin already goes to stdout.
- `--stdout` requires exactly one input that is a file. Passing a directory, or more than one path, with `--stdout` is a
  usage error (exit 2). Rationale: no unambiguous way to concatenate multiple files' output.
- Prefix values are trimmed of leading and trailing whitespace before validation and matching (`--library-prefix="  Foo  "` is treated as `Foo`).
- Prefix values are validated (usage error, exit 2) if any of the following holds:
  - an empty prefix is given (or one that is all whitespace, which trims to empty),
  - a prefix starts with `.`,
  - a prefix starts with `Qt` or `qt`,
  - a prefix equals `pragma`,
  - two prefixes overlap — either identical or one-is-a-prefix-of-the-other — whether within one flag or across `--library-prefix` and `--module-prefix`.
    The error names the specific prefix(es) involved.

### Classification override (`--library-prefix` / `--module-prefix`)

Both flags are repeatable. Each occurrence adds one literal prefix that is matched byte-for-byte against the normalized import text (the content after `import ` with whitespace trimmed). Include a trailing `.` in the prefix to create a boundary: `--module-prefix=MyCorp.` matches `import MyCorp.Foo` but not `import MyCorpExternal`.

User prefixes take precedence over the default dot/bare heuristic, so they can both promote (make a bare name count as library) and demote (make a dotted name count as module). The category precedence order is: pragma, qt, `--library-prefix`, `--module-prefix`, default library (dotted), default module (bare), relative.

Example:

```
qmlimportsort --module-prefix=AppComponents. --library-prefix=MyLib src/
```

Anything starting with `AppComponents.` is classified as a module; `MyLib` (bare) is classified as a library.

### Directory walking

- **Recurse** into directories fully.
- **Match**: files whose name ends in `.qml` (case-sensitive).
- **Skip** any directory or file whose name starts with `.` during recursion (e.g. `.git`, `.venv`, `.idea`). Explicitly
  passing such a path as an argument still processes it — the skip rule applies only while walking.
- **Don't follow symlinks** (neither directory nor file symlinks).
- **Empty results** (directory with no matching `.qml` files): silent success, exit 0.

### Error handling

- **Per-file parse/IO errors**: log to stderr, continue with remaining inputs.
- **Path doesn't exist**: treated as a per-input error (stderr + continue + exit 2 at end), not a fatal abort.
- **Exit code**: if any input produced an error, exit 2 after processing all inputs. Otherwise exit 0 (or 1 in `--check`
  mode if changes would be needed).

### Exit codes

| Code | Meaning                                                                           |
| ---- | --------------------------------------------------------------------------------- |
| 0    | Success. Write mode: all inputs processed cleanly. Check mode: no changes needed. |
| 1    | `--check` mode only: at least one file would change.                              |
| 2    | Usage error, or one or more inputs failed (missing file, parse error, IO error).  |

### Behavior guarantees

- **Idempotent**: `qmlimportsort x.qml && qmlimportsort --check x.qml` always exits 0 on the second call.
- **Atomic writes**: write formatted content to a temp file in the same directory, then `rename(2)` over the original.
  Prevents truncation on crash.
- **Preserve**: line endings (`\n` / `\r\n` / `\r`) and file mode.
- **No backups**: no `.bak` files. Users have VCS.
- **No progress output** on success in write mode — silence means success. Errors go to stderr.
- **Processing order**: inputs are processed in the order given on the command line. Within a directory, entries are
  processed in lexical order (sorted) so `--check` output is deterministic.

### Out of scope (deferred)

- Subcommands (`format`, `check`, etc.) — can be added later non-breakingly if a genuinely new verb shows up.
- `.gitignore` / `.qmlimportsortignore` support.
- `--exclude` / `--include` patterns.
- `--follow-symlinks`.
- Configurable file extensions.
- Parallel processing.
