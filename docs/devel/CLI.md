<!--
SPDX-FileCopyrightText: Elias Mueller

SPDX-License-Identifier: MIT
-->

# CLI Design

The flag list, synopsis, and exact mode behavior live in `qmlimportsort --help` and `cmd/qmlimportsort/main.go`. This document covers user-facing contracts and deliberate non-features.

## Exit codes

| Code | Meaning                                                                          |
| ---- | -------------------------------------------------------------------------------- |
| 0    | Success. Write mode: nothing needed to change. Check mode: nothing would change. |
| 1    | At least one file changed (write mode) or would change (check mode).             |
| 2    | Usage error, or one or more inputs failed (missing file, parse error, IO error). |

`--stdin` and `--stdout` (without `--check`) always exit 0 on success — they are pipe-style modes; the user already sees the output and shell `&&` chains shouldn't break on a transformation.

## Behavior guarantees

- **Idempotent**: `qmlimportsort x.qml && qmlimportsort --check x.qml` always exits 0 on the second call.
- **Atomic writes**: formatted content goes to a temp file in the same directory, then `rename(2)` over the original. Prevents truncation on crash.
- **Preserves** line endings (`\n` / `\r\n` / `\r`) and file mode.
- **No backups**: no `.bak` files. Users have VCS.
- **Silent on success** in write mode. Errors go to stderr.
- **Processing order**: inputs in the order given on the command line; entries within a directory in lexical order, so `--check` output is deterministic.
- **`--stdout` requires exactly one file**: there is no unambiguous way to concatenate multiple files' output.

## Out of scope (deferred)

- Subcommands (`format`, `check`, etc.) — can be added later non-breakingly if a genuinely new verb shows up.
- `.gitignore` / `.qmlimportsortignore` support.
- `--exclude` / `--include` patterns.
- `--follow-symlinks`.
- Configurable file extensions.
- Parallel processing.
