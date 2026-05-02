<!--
SPDX-FileCopyrightText: Elias Mueller

SPDX-License-Identifier: MIT
-->

# qmlimportsort

A tool to automatically organize and sort QML import statements according to best practices.

> [!CAUTION]
> This tool is in very early development. Use at your own risk.

> [!WARNING]
> The current version of the tool removes comments between an optional license header and the start of the component.

## Features

- Automatically organizes QML imports into logical groups
- Sorts imports alphabetically within each group
- Preserves line endings across platforms (Unix, Windows, macOS)
- Supports stdin/stdout workflow or in-place file modification
- Can process multiple files at once

## Import Organization

The tool organizes imports in the following order, with blank lines between non-empty groups:

1. Pragmas (e.g., `pragma Singleton`)
2. Qt imports (e.g., `import QtQuick`, `import QtQuick.Controls`)
3. Third-party imports — anything that isn't Qt, relative, or first-party (e.g., `import org.kde.kirigami`, `import MyModule`)
4. First-party imports — opt-in, see [First-party imports](#first-party-imports) below
5. Relative imports (e.g., `import "../components"`)

Within each group, imports are sorted alphabetically (case-sensitive byte order).

### Example

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

### First-party imports

By default, every non-Qt, non-relative import is third-party. Use `--first-party-prefix` to mark imports owned by your project as first-party so they get their own group. The flag is repeatable; include a trailing `.` to limit the match to dotted subpaths.

Same input as above, run with `--first-party-prefix=MyModule`:

```qml
// SPDX-FileCopyrightText: Jane Doe
//
// SPDX-License-Identifier: MIT

pragma Singleton

import QtQuick
import QtQuick.Controls

import org.kde.kirigami as Kirigami

import MyModule

import "../components"

QtObject {}
```

`MyModule` is now in its own first-party group between third-party and relative.

## Installation

### Pre-built Binaries

Download pre-built binaries from the releases page.

## Development

### Prerequisites

- Go 1.24 or later

### Building

```shell
just build
```

### Running Tests

```shell
just test
```

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

[MIT License](LICENSE)
