- [x] Task 1: Border & Title Style (Lipgloss)
  - [x] Add `github.com/charmbracelet/lipgloss` dependency
  - [x] Update `overhead = 5` in scroll logic (to account for borders)
  - [x] Refactor `View()` in `model.go` to render border and center the title "FMTUI"
  - [x] Verify build and tests

- [x] Task 2: Copy Name / Path Menu ("y" key)
  - [x] Define copy menu mode/fields in `model`
  - [x] Intercept `"y"` key to open copy menu, handling `"p"`, `"n"`, `"e"`, `"esc"`
  - [x] Render copy menu popup centered on screen in `View()`
  - [x] Verify build and tests

- [x] Task 3: File / Directory Preview Pane
  - [x] Divide screen horizontally (left: list, right: preview)
  - [x] Implement right-pane view for files (text preview) and directories (children preview)
  - [x] Verify build and tests

- [x] Task 4: Open in Editor ("o" key)
  - [x] Implement `o` key handler using `tea.Exec` or custom process suspension
  - [x] Verify build and tests

- [x] Task 5: Switch active pane with "[" and "]"
  - [x] Add state fields `activePane` and `previewScrollOffset` in `model`
  - [x] Implement key handlers for `[` and `]` to toggle `activePane`
  - [x] Adjust `up`/`down` key handlers to scroll preview pane if `activePane == 1`
  - [x] Reset preview scroll offset when cursor moves in left pane
  - [x] Verify build and tests

- [x] Task 6: Active Pane Styling (Color Blue)
  - [x] Define active border style (Blue foreground `#5555FF`) and inactive border style (Gray foreground `#555555`)
  - [x] Dynamically apply border styles based on `activePane`
  - [x] Verify build and tests

- [x] Task 7: Scrollable Preview Pane
  - [x] Refactor `getPreviewText` in `utilities.go` to `getPreviewLines` returning `[]string`
  - [x] Slice and render lines in `View()` based on `previewScrollOffset`
  - [x] Verify build and tests

- [x] Task 8: Preview Line Truncation (Horizontal Fit)
  - [x] Implement tab expansion and horizontal line truncation in `View()` in `model.go`
  - [x] Add unit test verifying horizontal fit
  - [x] Verify build and tests
