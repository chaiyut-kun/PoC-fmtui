w# fmtui - System Architecture Plan

## Current Architecture

```
┌─────────────────────────┐
│         Terminal        │
│   keyboard + screen I/O │
└───────────┬─────────────┘
            │
            v
┌─────────────────────────┐
│      Bubble Tea App     │
│   main.go -> tea.Program│
└───────────┬─────────────┘
            │
            v
┌─────────────────────────┐
│       model.go          │
│─────────────────────────│
│ model state             │
│ - currentDir            │
│ - filesPath             │
│ - cursor                │
│ - selected              │
│ - showHidden            │
│ - inputMode/inputText   │
│ - scroll/error state    │
│                         │
│ Update(msg)             │
│ View()                  │
│ adjustScroll()          │
└───────────┬─────────────┘
            │
            v
┌─────────────────────────┐
│      utilities.go       │
│─────────────────────────│
│ reLoadDir()             │
│ reloadCurrentDir()      │
│ isDir()                 │
│ findAbsPath()           │
│ CheckDirFilesExt()      │
│ copyToClipboard()       │
│ temp file helpers       │
└───┬───┬─────────────────┘
    │   │
    │   └──────────────────────┐
    │                          │
    v                          v
┌──────────┐            ┌──────────────────┐
│Filesystem│            │ Clipboard / Temp │
│os.ReadDir│            │ atotto/clipboard │
│os.Stat   │            │ /tmp/fmtui-dir   │
│os.Create │            └──────────────────┘
│os.Mkdir  │
└──────────┘
```

## Target Architecture

### Layer Diagram

```
┌──────────────────────────────────────────────────┐
│                  Terminal UI                     │
│            Bubble Tea Program / View             │
├────────────────────┬─────────────────────────────┤
│                    │                             │
│                    v                             │
│  ┌──────────────────────────────────────────┐   │
│  │            Presentation Layer            │   │
│  │──────────────────────────────────────────│   │
│  │ tui/model                                │   │
│  │ - Update()                               │   │
│  │ - View()                                 │   │
│  │ - input mode, cursor, scroll             │   │
│  │ - translates keypresses into actions     │   │
│  └────────────────────┬─────────────────────┘   │
│                       │                         │
│                       v                         │
│  ┌──────────────────────────────────────────┐   │
│  │            Application Layer             │   │
│  │──────────────────────────────────────────│   │
│  │ app/service                              │   │
│  │ - LoadDirectory(path, showHidden)        │   │
│  │ - NavigateInto(name)                     │   │
│  │ - NavigateUp()                           │   │
│  │ - ToggleHidden()                         │   │
│  │ - CreateFile(name)                       │   │
│  │ - CreateDir(name)                        │   │
│  │ - CopySelectedPath()                     │   │
│  └────────────────────┬─────────────────────┘   │
│                       │                         │
│                       v                         │
│  ┌──────────────────────────────────────────┐   │
│  │              Domain Layer                │   │
│  │──────────────────────────────────────────│   │
│  │ domain/model                             │   │
│  │ - CurrentDir  string                     │   │
│  │ - Entries     []Entry                    │   │
│  │ - Cursor      int                        │   │
│  │ - Selection   map[int]struct{}           │   │
│  │ - ShowHidden  bool                       │   │
│  │ - Error       error                      │   │
│  │                                          │   │
│  │ Entry struct                             │   │
│  │ - Name     string                        │   │
│  │ - Path     string                        │   │
│  │ - IsDir    bool                          │   │
│  │ - IsHidden bool                          │   │
│  │ - Kind     string (ext / special name)   │   │
│  └────────────────────┬─────────────────────┘   │
│                       │                         │
│                       v                         │
│  ┌──────────────────────────────────────────┐   │
│  │          Infrastructure Layer            │   │
│  │──────────────────────────────────────────│   │
│  │ infra/filesystem                         │   │
│  │ - ReadDir(path) → []Entry                │   │
│  │ - Stat(path) → FileInfo                  │   │
│  │ - CreateFile(path)                       │   │
│  │ - CreateDir(path)                        │   │
│  │                                          │   │
│  │ infra/clipboard                          │   │
│  │ - Copy(text) error                       │   │
│  │                                          │   │
│  │ infra/handoff                            │   │
│  │ - WriteChosenDir(dir) error              │   │
│  │ - ClearChosenDir() error                 │   │
│  └──────────────────────────────────────────┘   │
│                                                  │
└──────────────────────────────────────────────────┘
```

### Data Flow

```
User key press
     │
     v
Bubble Tea Update()
     │
     v
App action (navigate, create, toggle, copy)
     │
     v
Filesystem / Clipboard operation
     │
     v
State update (reload entries, set cursor)
     │
     v
View render (icons, colors, scroll)
     │
     v
Terminal output
```

### Component Breakdown (Future Package Structure)

```
main.go
go.mod
Plan.MD
internal/
  tui/
    model.go       ─ Bubble Tea model, Update(), View()
    view.go        ─ rendering helpers, icon/color selection
    update.go      ─ key handler dispatch
  app/
    navigator.go   ─ LoadDir, NavigateInto, NavigateUp
    actions.go     ─ CreateFile, CreateDir, CopyPath
  domain/
    entry.go       ─ Entry struct (Name, Path, IsDir, IsHidden, Kind)
    state.go       ─ AppState struct
  infra/
    filesystem.go  ─ ReadDir, Stat, CreateFile, CreateDir
    clipboard.go   ─ Copy via atotto/clipboard
    handoff.go     ─ Temp file write/clear for shell cd
  ui/
    icons.go       ─ Nerd Font icon constants
    colors.go      ─ ANSI color constants
    status.go      ─ Status bar / error display helpers
```

## Key Improvement: Structured Entries

**Current:**
```go
filesPath []string
```

**Target:**
```go
type Entry struct {
    Name     string
    Path     string
    IsDir    bool
    IsHidden bool
    Kind     string
}
```

**Benefit:** Simplifies hidden file filtering, icon selection, directory
coloring, traversal, and future features like sort/filter/search.

## Dependency Map

```
┌──────────────────────────────────────┐
│        charm.land/bubbletea/v2       │
│  (TUI framework: event loop, render) │
├──────────────┬───────────────────────┤
│              │                       │
│  github.com/atotto/clipboard        │
│  (system clipboard access)          │
│              │                       │
│              └─── Terminal libs ─────┤
│                   (ansi, terminfo,   │
│                    runewidth, etc.)  │
└──────────────────────────────────────┘
```

## Shell Integration

```
┌─────────────────────┐
│    User's Shell     │
│   (zsh / bash)      │
│                     │
│  fm() {             │
│    fmtui            │
│    dir=$(cat /tmp/  │
│      fmtui-dir)     │
│    cd "$dir"        │
│  }                  │
└──────────┬──────────┘
           │ reads from
           v
┌─────────────────────┐
│   /tmp/fmtui-dir    │
│  (temp file handoff)│
└──────────┬──────────┘
           │ written by
           v
┌─────────────────────┐
│      fmtui app      │
│  - 'd' writes path  │
│  - 'q' clears file  │
└─────────────────────┘
```

## Implementation Priorities

1. **Refactor to `[]Entry`** — foundation for all future features
2. **Extract service layer** — move filesystem/OS calls out of `Update()`
3. **Clean package structure** — reorganize files as shown above
4. **Add features on stable base** — search, filter, bookmarks, file preview