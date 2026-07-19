# Plan: Run App & Project Information Walkthrough

This document outlines the step-by-step plan to inspect, explain, and demonstrate how the **FMTUI** (File Manager TUI) application works and how to run it.

---

## 1. Goal Overview
Provide a clear, structured guide explaining how the FMTUI application functions, its architecture, keybindings, and execution modes (Development, Production Binary, Unit Testing, and Shell Integration).

---

## 2. Execution & Demo Plan

### Phase 1: Application Architecture & Execution Modes
1. **Direct Execution (Development)**:
   - `go run .`
2. **Binary Compilation & Execution**:
   - `go build -o list2`
   - `./list2`
3. **Automated Testing**:
   - `go test ./...`
4. **Shell Auto-CD Integration**:
   - When exiting with key `'e'`, the app writes the final working directory to `/tmp/fmtui-dir`.
   - Shell alias / function: `fmtui() { ./list2; if [ -f /tmp/fmtui-dir ]; then cd "$(cat /tmp/fmtui-dir)" && rm /tmp/fmtui-dir; fi }`

### Phase 2: Key Features & Keyboard Shortcuts Summary
- **Navigation & Focus**: `j`/`k` (Up/Down), `h`/`l` (Parent/Child), `[`/`]` (Switch Pane Focus)
- **File Actions**:
  - `'y'`: Copy Menu (`p` Path, `n` Name, `e` Extension)
  - `'d'`: Delete Menu (`p` Permanent, `n`/`r` Trash)
  - `'r'`: Rename Prompt
  - `'a'`: Create File
  - `'m'`: Create Directory
  - `'u'`: Undo Action
  - `'ctrl+r'` / `'U'`: Redo Action
  - `'o'`: Launch `$EDITOR`
  - `'.'`: Toggle Hidden Files (`ON`/`OFF`)
  - `'e'`: Exit & Change Directory (`cd`)
  - `'q'`: Exit Application

### Phase 3: Project Health Verification
- Execute `go test ./...` to demonstrate unit test health.
- Execute `go build -o list2` to verify compilation.
