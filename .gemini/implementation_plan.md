# Implementation Plan - Restructure Project Layout

## Goal Description
Restructure the application files into a standard, idiomatic Go project layout. This organizes the flat directory structure, separates the entry point from the UI/business logic, and keeps components modular.

## Proposed Changes
We propose moving from a flat root layout to a standard Go structure:
- **`cmd/fm/main.go`**: The application's entry point (`package main`). It imports the TUI package and runs the Bubble Tea program.
- **`internal/tui/`**: A private internal package (`package tui`) containing all user interface, layout, and event handling logic:
  - `model.go`: Contains struct definitions (`Model`, `TabState`), helper methods, and initializers.
  - `update.go`: Contains the Bubble Tea `Update(msg tea.Msg) (tea.Model, tea.Cmd)` logic (extracted from `model.go`).
  - `view.go`: Contains the `View() tea.View` rendering logic (extracted from `model.go`).
  - `utilities.go`: Contains filesystem helper methods and operations.
  - `constant.go`: Contains colors, icons, and constants.
  - `bookmarks.go`: Contains loading/saving of bookmarks.
  - `tui_test.go` (renamed from `main_test.go`): Package-level unit tests.

### Build and Tooling Updates
- **`Makefile`**: Update targets (`dev`, `bd`) to run/build `./cmd/fm`.
- **`build.sh`**: Update to build `./cmd/fm` as `fm` and copy to `GOPATH/bin`.
- **`cd-on-exit.sh`**: Ensure it references the standard command `fm`.

---

### Step-by-Step Proposed Changes

#### [NEW] cmd/fm/main.go
Entry point for the application.
```go
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"file-list/internal/tui"
)

func main() {
	program := tea.NewProgram(tui.InitialModel())
	m, err := program.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	_ = m
}
```

#### [NEW] internal/tui/model.go
Structs, initialization, and helper methods. The `model` struct will be exported as `Model` to make it accessible to `cmd/fm/main.go`.
```go
package tui

// Imports...
// Structs (Model, TabState, ActionRecord, ShortcutEntry)
// InitialModel()
// getMaxVisibleFiles()
// adjustScroll()
```

#### [NEW] internal/tui/update.go
Extracted `Update` loop for cleaner file separation.
```go
package tui

import tea "charm.land/bubbletea/v2"

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Key and mouse events...
}
```

#### [NEW] internal/tui/view.go
Extracted `View` rendering loop.
```go
package tui

import tea "charm.land/bubbletea/v2"

func (m Model) View() tea.View {
    // Layout and rendering...
}
```

#### [NEW] internal/tui/utilities.go
Change package header to `package tui` and update method receivers to `Model`.

#### [NEW] internal/tui/constant.go
Change package header to `package tui`.

#### [NEW] internal/tui/bookmarks.go
Change package header to `package tui`.

#### [NEW] internal/tui/tui_test.go
Change package header to `package tui` and update tests to target exported `Model`.

#### [DELETE] Flat files in root:
- `main.go`
- `model.go`
- `utilities.go`
- `constant.go`
- `bookmarks.go`
- `main_test.go`

---

## Verification Plan

### Automated Tests
Run unit tests in the new package location:
```bash
go test ./internal/tui/...
```

### Manual Verification
1. **Compilation**: Run `go build -o fm ./cmd/fm` to verify compilation.
2. **Execution**: Run `./fm` and verify all functionalities (directory listing, navigation, search with active/inactive border, open menu, delete, rename, etc.) work exactly as before.
