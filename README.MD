<!--
SPDX-FileCopyrightText: Elias Mueller

SPDX-License-Identifier: MIT
-->

# qmlimportsort

A tool to organize and sort QML import statements.

> [!CAUTION]
> This tool is in early development. Use at your own risk.
> The CLI is not stable yet — flags and behavior may change in breaking ways until version 1.0.0.

> [!IMPORTANT]
> This is a vibe-coded project — most of the code was written collaboratively with an AI assistant.

> [!NOTE]
> Comments between two imports are hoisted into a single comment section above the imports — they do not travel with their original import line through sorting. To attach a note to a specific import, put it as a trailing comment on the same line (e.g. `import QtQuick // note`).

## Examples

Imports are grouped into pragmas, Qt, default, custom sections (opt-in), and relative — sorted and de-duplicated within each group, separated by a blank line.

Given this file:

<!-- test:input -->
```qml
// SPDX-FileCopyrightText: Jane Doe
//
// SPDX-License-Identifier: MIT
import QtQuick.Controls
import "../components"
import MyApp.Views
import org.kde.kirigami as Kirigami
import QtQuick
import Company.Widgets 1.2
import MyApp.Backend
import QtQuick
pragma ComponentBehavior: Bound
import Qt.labs.settings

Item {}
```

`qmlimportsort Main.qml` applies the default grouping — everything that is not Qt or relative shares one section:

<!-- test:expect -->
```qml
// SPDX-FileCopyrightText: Jane Doe
//
// SPDX-License-Identifier: MIT

pragma ComponentBehavior: Bound

import Qt.labs.settings
import QtQuick
import QtQuick.Controls

import Company.Widgets 1.2
import MyApp.Backend
import MyApp.Views
import org.kde.kirigami as Kirigami

import "../components"

Item {}
```

`qmlimportsort --group MyApp. Main.qml` gives the project's own modules their own section:

<!-- test:expect -->
```qml
// SPDX-FileCopyrightText: Jane Doe
//
// SPDX-License-Identifier: MIT

pragma ComponentBehavior: Bound

import Qt.labs.settings
import QtQuick
import QtQuick.Controls

import Company.Widgets 1.2
import org.kde.kirigami as Kirigami

import MyApp.Backend
import MyApp.Views

import "../components"

Item {}
```

`qmlimportsort --group Company. --group MyApp. Main.qml` splits further — one section per namespace, in flag order:

<!-- test:expect -->
```qml
// SPDX-FileCopyrightText: Jane Doe
//
// SPDX-License-Identifier: MIT

pragma ComponentBehavior: Bound

import Qt.labs.settings
import QtQuick
import QtQuick.Controls

import org.kde.kirigami as Kirigami

import Company.Widgets 1.2

import MyApp.Backend
import MyApp.Views

import "../components"

Item {}
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

# Give your project's modules their own section (repeatable, comma-separated)
qmlimportsort --group io.github.mpvqc. src/
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
