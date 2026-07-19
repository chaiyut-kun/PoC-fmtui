# Implementation Plan - Update Architecture Documentation

## Goal Description
With the recent major updates (search & filter features, scroll layout alignment, and the Go project restructuring/code split), the system architecture has evolved. We need to update our architecture diagrams and documentation (`Architecture.md` and `.gemini/architecture_diagrams.md`) to accurately represent the current implementation instead of holding deprecated models.

---

## Proposed Changes

### Documentation Changes

#### [MODIFY] Architecture.md (Root)
Populate this empty root file with a high-level summary of the directory structure and package organization.
- Explain the division between the entrypoint (`cmd/fm`) and the core application (`internal/tui`).
- Document the role of the modularized TUI sub-files (`model.go`, `update.go`, `view.go`).

#### [MODIFY] .gemini/architecture_diagrams.md
Update the Mermaid architecture diagrams to depict the actual layout of the application:
- **Mermaid Block 1: Modular TUI Design**: Show how `model.go`, `update.go`, and `view.go` share package state and function calls.
- **Mermaid Block 2: Search and Filter Pipeline**: Chart the data flow of search keypresses (`s` -> `Update()`), filtering logic (`applySearchFilter()` -> `t.filteredFiles`), and rendering via `t.visibleFiles()`.
- **Mermaid Block 3: Binary Compilation Layout**: Map the compilation dependency structure.

---

## Verification Plan

### Automated Tests
Not applicable (documentation only).

### Manual Verification
1. Ensure both `.md` files compile markdown cleanly in rendering previews.
2. Verify all path names and file references correspond exactly to the current folder tree.
