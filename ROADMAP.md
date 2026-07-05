<!--
SPDX-FileCopyrightText: Elias Mueller

SPDX-License-Identifier: MIT
-->

# Roadmap

## Refactor

1. **Design the CLI** — ✅ Done. Agreed surface: write-by-default, `--check` / `--stdout` / `--stdin` modes, recursive directory walking, skip dotfiles, atomic writes. Contracts in [docs/devel/CLI.md](docs/devel/CLI.md); the flag surface lives in `--help`.
2. **Design the internal API** — ✅ Done. Split into `internal/qml` (pure `Format([]byte)`) and `internal/fs` (I/O shell: walker, atomic writes, stream/file helpers). `main` is a thin dispatcher. Full spec in [docs/devel/INTERNAL_API.md](docs/devel/INTERNAL_API.md).
