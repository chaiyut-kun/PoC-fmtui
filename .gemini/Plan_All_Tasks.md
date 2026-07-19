# Plan: All TUI Tasks, Multi-Tabs & 1/3 Preview Split Ratio

This document maps out the roadmap for implementing all requested features and designs for **fmtui** as described in `Design.md` and `Features.md`.

---

## Roadmap of Completed Tasks

- [x] **Task 1: Border & Title Style (Lipgloss)** - White rounded borders with centered "FMTUI" title.
- [x] **Task 2: Copy Name / Path Menu ("y" key)** - Pop-up copy menu for path, name, extension.
- [x] **Task 3: File / Directory Preview Pane** - Side-by-side horizontal split.
- [x] **Task 4: Open in Editor ("o" key)** - `$EDITOR` launching via `tea.ExecProcess`.
- [x] **Task 5: Switch Active Pane ("[" and "]" keys)** - Toggle focus between Explorer and Previewer.
- [x] **Task 6: Active Pane Styling (Color Blue)** - Blue active border, Gray inactive border.
- [x] **Task 7: Scrollable Preview Pane** - Vertical preview scrolling with `k`/`j`.
- [x] **Task 8: Preview Line Truncation** - Tab expansion and horizontal line truncation.
- [x] **Task 9: Code Review & Bug Fixes** - Fixed syntax, hidden files, terminal corruption, and safety guards.
- [x] **Task 10: Delete Action Dialog ("d" key)** - Delete menu overlay with permanent (`p`) and normal/trash (`r`) options.
- [x] **Task 11: Rename Feature & Prompt ("r" key)** - Interactive rename prompt.
- [x] **Task 12: Undo & Redo System** - Undo (`u`) and Redo (`ctrl+r`/`U`) stacks.
- [x] **Task 13: Fix Normal Delete & Cross-Device Move to Trash** - Cross-device move with `moveFileOrDir`.
- [x] **Task 14: Show Current Path at Top of Left Pane** - Current directory in top border header.
- [x] **Task 15: Move Help Message to Bottom of App** - Full-width bottom status bar below split panes.
- [x] **Task 16: File Permission Management Dialog ("p" key)** - Permission menu (`x` for chmod +x/-x).
- [x] **Task 17: Full Help Menu Page ("`" key)** - Full help menu page overlay.
- [x] **Task 18: Mouse Scroll Pane & Click Focus** - Mouse wheel scrolling and mouse click pane focus.

---

- [x] **Task 19: Preview Pane 1/3 Screen Size (2/3 Left, 1/3 Right)** - Left Explorer pane occupies 2/3 width, Right Preview pane occupies 1/3 width.
- [x] **Task 20: Multi-Tabs of File Explorer Pane** - Multi-tabs for File Explorer pane with tab bar header overlay.
- [x] **Task 21: Open Menu Dialog ("o" key)** - Pop-up open menu with options:
  - `'c'`: Open file or directory in VSCode (`code`).
  - `'r'`: Reveal in GUI File Explorer (`xdg-open` / `open` / `explorer.exe`).
  - `'e'`: Open file in `$EDITOR`.
