<!--
SPDX-FileCopyrightText: Elias Mueller

SPDX-License-Identifier: MIT
-->

# qmlimportsort

A tool to organize and sort QML import statements.

> [!CAUTION]
> This tool is in early development. Use at your own risk.

> [!IMPORTANT]
> This is a vibe-coded project — most of the code was written collaboratively with an AI assistant.

> [!NOTE]
> Comments between two imports are hoisted into a single comment section above the imports — they do not travel with their original import line through sorting. To attach a note to a specific import, put it as a trailing comment on the same line (e.g. `import QtQuick // note`).

## Example

Imports are grouped into pragmas, Qt, third-party, first-party (opt-in), and relative — sorted and de-duplicated within each group, separated by a blank line.

Before:

```qml
// SPDX-FileCopyrightText: Jane Doe
//
// SPDX-License-Identifier: MIT
import "../components"
import MyModule
import QtQuick.Controls
import org.kde.kirigami as Kirigami
import QtQuick
pragma Singleton
QtObject {}
```

After:

```qml
// SPDX-FileCopyrightText: Jane Doe
//
// SPDX-License-Identifier: MIT

pragma Singleton

import QtQuick
import QtQuick.Controls

import MyModule
import org.kde.kirigami as Kirigami

import "../components"

QtObject {}
```

## Usage

```shell
# Format files or directories in place
qmlimportsort Main.qml
qmlimportsort src/

# Dry run — print paths that would change, exit 1 if any
qmlimportsort --check src/

# Print formatted content to stdout, leave file untouched
qmlimportsort --stdout Main.qml

# Pipe through stdin/stdout
cat Main.qml | qmlimportsort --stdin

# Mark imports owned by your project as first-party (repeatable)
qmlimportsort --first-party-prefix=io.github.mpvqc. src/
```

Run `qmlimportsort --help` for the full flag list.

## Development

Requires Go 1.24+.

```shell
just build
just test
```

## License

[MIT License](LICENSE)
