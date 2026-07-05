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

`--stdin` (without `--check`) and `--stdout` always exit 0 on success — they are pipe-style modes; the user already sees the output and shell `&&` chains shouldn't break on a transformation. Combining `--stdout` with `--check` is a usage error.

## Behavior guarantees

- **Idempotent**: `qmlimportsort x.qml && qmlimportsort --check x.qml` always exits 0 on the second call.
- **Atomic writes**: formatted content goes to a temp file in the same directory, then `rename(2)` over the original. Prevents truncation on crash.
- **Preserves** line endings (`\n` / `\r\n` / `\r`) and file mode.
- **No backups**: no `.bak` files. Users have VCS.
- **Silent on success** in write mode. Errors go to stderr.
- **Processing order**: inputs in the order given on the command line; entries within a directory in lexical order, so `--check` output is deterministic.
- **`--stdout` requires exactly one file**: there is no unambiguous way to concatenate multiple files' output.

## Import grouping (`--group`)

Imports are emitted in this order, separated by blank lines and each sorted and deduplicated: **pragmas**, **Qt**, **default** (everything no group claims), **custom sections**, **relative** (quoted paths).

`--group <prefix>[,<prefix>...]` declares one custom section holding every import that starts with one of its prefixes. Repeat the flag to declare more sections; they appear in the order given.

```shell
# One section for company libraries, one for the app's own modules
qmlimportsort --group Company.Shared.,Company.Widgets. --group MyApp. src/

# The same prefixes as three separate sections
qmlimportsort --group Company.Shared. --group Company.Widgets. --group MyApp. src/
```

The first command formats a file like this:

```qml
pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Controls

import org.kde.kirigami as Kirigami

import Company.Shared.Logging
import Company.Widgets.Buttons

import MyApp.Views

import "./components"
```

The second command puts every namespace in its own section:

```qml
pragma ComponentBehavior: Bound

import QtQuick
import QtQuick.Controls

import org.kde.kirigami as Kirigami

import Company.Shared.Logging

import Company.Widgets.Buttons

import MyApp.Views

import "./components"
```

End a prefix with `.` to avoid catching sibling modules: `MyApp.` matches `MyApp.Views` but not `MyAppExtras`. Invalid flags (empty, `.`-leading, Qt-reserved, or duplicate prefixes) exit 2 before any file is touched.

Full classification and validation semantics live in [INTERNAL_API.md](INTERNAL_API.md).

## Out of scope (deferred)

- Config file (YAML or TOML). Planned — the repeatable `--group` flag is designed to map 1:1 onto it (schema sketch in [INTERNAL_API.md](INTERNAL_API.md)); CLI flags will take precedence when both are present.
- Named prefix groups. Useful only once a config file exists (readability, error messages).
- Subcommands (`format`, `check`, etc.) — can be added later non-breakingly if a genuinely new verb shows up.
- `.gitignore` / `.qmlimportsortignore` support.
- `--exclude` / `--include` patterns.
- `--follow-symlinks`.
- Configurable file extensions.
- Parallel processing.
