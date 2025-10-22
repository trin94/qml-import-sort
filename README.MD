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

The tool organizes imports in the following order, with blank lines between groups:

1. Pragmas (e.g., pragma Singleton)
2. Qt imports (e.g., import QtQuick, import QtQuick.Controls)
3. Library imports (e.g., import org.kde.kirigami)
4. Module imports (e.g., import MyModule)
5. Relative imports (e.g., import "../components")

Within each group, imports are sorted alphabetically.

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

import org.kde.kirigami as Kirigami

import MyModule

import "../components"

QtObject {}
```

## Installation

### Pre-built Binaries

Download pre-built binaries from the releases page.

## Usage

    USAGE:
       qmlimportsort [flags] [files...]

    FLAGS:
       --in-place, -i  modify files in-place (only valid with files)
       --help, -h      show help
       --version, -v   print the version

### Examples

Process files and output to stdout:

    qmlimportsort Main.qml Component.qml

Modify files in-place:

    qmlimportsort -i Main.qml Component.qml

Process from stdin:

    cat Main.qml | qmlimportsort

Process all QML files in a directory:

    find . -name "*.qml" -exec qmlimportsort -i {} +

## Development

### Prerequisites

- Go 1.24 or later

### Building

    go build

### Running Tests

    go test ./...

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

[MIT License](LICENSE)
