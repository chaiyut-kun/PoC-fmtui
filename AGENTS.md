# AGENTS.md — fmtui / file-list

Guidance for AI agents and human contributors working on this repository.
Source documents: `.gemini/PROJECT_KNOWLEDGE.md` (deep invariants), `Architecture.md`, `Plan.md`, `README.md`.

---

## What this is

**fmtui** (module `file-list`) — a proof-of-concept terminal file manager for Linux, written in Go, built on the Charm TUI stack v2:
- `charm.land/bubbletea/v2` (Elm-style state loop)
- `charm.land/lipgloss/v2` (styling/text ink)
- `charm.land/bubbles/v2` (textinput)
- `github.com/atotto/clipboard` (system clipboard)

3-pane TUI: shortcuts/bookmarks sidebar, file explorer, preview + bottom status/clipboard bars. Vim-ish keys (`hjkl`), search, copy/move/delete/rename, undo/redo, quick-find overlay (`ctrl+f`), multi-tab, mouse support, and a `cd`-on-exit shell handoff.

## Quick start

```bash
# Build (canonical — binary MUST be named fm)
go build -o fm ./cmd/fm

# Run
go run ./cmd/fm          # or: make dev

# Test (headless, exercises model + view directly)
go test ./...
go test -v -run TestUndoRedo ./internal/tui/

# Regenerate local knowledge graph & docs index
#   run the /understand skill (see "Knowledge graph" below)
```

Requires Go >= 1.25 (per `go.mod`; tested with 1.26). Nerd Fonts recommended for icons.

> **Known gotcha:** `scripts/build.sh` currently outputs `fmtui` and `Makefile bd` calls it — this conflicts with the `fm` invariant and the `cd-on-exit` wrapper, which invokes `fm`. Prefer the canonical `go build -o fm ./cmd/fm` until `build.sh` is aligned.

## Repository map

```
cmd/fm/main.go           Entry point: tea.NewProgram(tui.InitialModel()).Run()
internal/tui/            Everything else (single Go package `tui`)
  model.go               State: model, TabState, ActionRecord, InitialModel()
  model_methods.go       dir reload, tab refresh, search filter
  update.go              Root Update(): routes msgs to active-pane handler
  update_explorer.go     explorer keys: nav, select, copy/move, create, undo/redo
  update_input.go        shortcut pane + search + text-input prompts + quick-find keys
  update_menus.go        modal menus: copy, delete, chmod, open-with, help, bookmarks
  view.go                Single View() renders all panes/overlays
  utilities.go           helpers: clipboard, preview, metadata, formatting
  fs_operations.go       copy/move/delete/trash/chmod + undo/redo executors
  quick_find.go          ctrl+f fuzzy search over home/cwd/bookmarks
  constant.go, config.go, bookmarks.go, cmdline.go, help_data.go
  tui_test.go            60+ headless tests
scripts/                 build.sh, cd-on-exit.{sh,ps1}, add-wrapper-script.ps1
diagrams/                Mermaid + HTML architecture diagrams
prototypes/              .tui static screen mockups
migrate/lipgloss-migration.md
```

Layers (from the knowledge graph): CLI Entry → State & Model → Input Handling → Rendering → Filesystem Services → Tests; plus Documentation, Diagrams/Prototypes, Build Config, Shell Integration.

## Runtime model (Elm architecture)

State lives in a single immutable-ish `model` value. Every input returns an updated model.

1. `InitialModel()` seeds state (cwd listing, tabs, shortcuts, bookmarks, config).
2. `Update(msg)` dispatches to exactly one handler, in strict priority (see `update.go`):
   ```
   ctrl+c quit → quick-find → any menu → shortcut pane → search → text input → explorer keys
   ```
3. `View()` paints from model state. Keep `Update`/`View` free of blocking I/O — heavy work (clipboard copy, quick-find walk) runs in goroutines that return messages via channels.

## Non-negotiable invariants

1. **Binary name is `fm`.** Never rename.
2. **Shell handoff contract.** Exit-`e` writes cwd to `/tmp/fmtui-dir` (`writeTempDirFile`); quit `q`/`Q`/`ctrl+c` clears it (`clearTempDirFile`). Never drop either call — the `fm()` shell wrapper depends on it.
3. **Layout math + mouse hit-testing must agree.** `view.go` width/ratio math and `update.go` mouse-coordinate detection must stay in sync (prefer extracting to a shared calculator).
4. **Input modes are mutually exclusive.** When a modal/input/search is open, explorer keys must never fire explorer actions.
5. **Destructive ops push an `ActionRecord` to `undoStack`** (and clear `redoStack`): delete → trash (`gio trash` / `~/.local/share/Trash`); permanent delete → `os.RemoveAll`; rename/creates/chmod reversible.
6. **No blocking I/O in `Update()`/`View()`.** Use goroutines + channels (see the 50MB clipboard-copy job pattern).

## Contribution workflow

1. Read the invariants above and `.gemini/PROJECT_KNOWLEDGE.md` before editing.
2. Build + test first: `go build -o fm ./cmd/fm && go test ./...`.
3. Follow the headless test pattern in `tui_test.go` (drive `InitialModel()` with `m.Update(tea.KeyPressMsg{...})`, assert on state). New behaviours are expected to ship with a test.
4. Keep files structured as above — don't merge the `update_*`/`view`/`model` split into one god file.
5. Any plan/diagram/design doc for this project lives in `.gemini/` (or `diagrams/` for diagrams).
6. Known debt / refactor roadmap: `Plan.md` and `.gemini/PROJECT_KNOWLEDGE.md` §6. Top items:
   - decompose the ~930-line `View()` god function into per-pane renderers
   - replace the 14+ modal booleans with an `ActiveModal` enum
   - extract a shared `calculateLayout()` so view + mouse-hit-test agree

## Data written by the app

- `~/.config/fmtui/config.json` — user settings (`AllowedHiddenFiles`)
- `~/.config/fmtui/bookmarks.json` — bookmarked directories
- `/tmp/fmtui-dir` — shell handoff file (app-written, wrapper-read)

## Knowledge graph & docs

- Machine-readable graph: `.ua/knowledge-graph.json` (nodes/edges/layers/tour), refreshed by the **/understand** skill.
- Architecture detail: `Architecture.md`, `Plan.md`, `diagrams/`, `.gemini/PROJECT_KNOWLEDGE.md`.
- Interactive dashboard (points at `.ua/knowledge-graph.json`):
  ```bash
  cd ~/.understand-anything-plugin/packages/dashboard
  GRAPH_DIR="<this repo>" npx vite --host 127.0.0.1   # URL printed with access token
  ```