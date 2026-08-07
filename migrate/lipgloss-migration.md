# Lipgloss v2 Migration - Layout Rendering & Border Conflict Resolution

This document explains how the three main panels (Shortcuts, File Explorer, Previewer) and the two sub-panels (Process, Clipboard) are rendered, how their dimensions and borders are assembled, and how to migrate the codebase cleanly to `charm.land/lipgloss/v2` without double-border conflicts or gaps.

---

## 1. Panel Layout Architecture & Assembly

The screen is split into three main columns horizontally, with the middle column (File Explorer) vertically split to host two sub-panels at the bottom.

### Visual Grid Topology

```text
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                                    Terminal Width (W)                                  │
├───────────────────┬─────────────────────────────────────────────────┬──────────────────┤
│                   │                leftWidth                        │                  │
│  shortcutWidth    ├────────────────────────┬────────────────────────┤   rightWidth     │
│                   │      processWidth      │     clipboardWidth     │                  │
│ ╭───────────────╮ ╭──────────────────────╮ ╭──────────────────────╮ ╭────────────────╮ │
│ │               │ │                      │ │                      │ │                │ │
│ │ 1. Shortcuts  │ │   2. File Explorer   │ │                      │ │  3. Previewer   │ │
│ │    (Pane 0)   │ │       (Pane 1)       │ │                      │ │    (Pane 2)    │ │
│ │               │ │                      │ │                      │ │                │ │
│ │               │ │                      │ │                      │ │                │ │
│ │               │ ├──────────────────────┴────────────────────────┤ │                │ │
│ │               │ │ ╭──────────────────╮   ╭────────────────────╮ │ │                │ │
│ │               │ │ │ 4. Process       │   │ 5. Clipboard       │ │ │                │ │
│ │               │ │ ╰──────────────────╯   ╰────────────────────╯ │ │                │ │
│ ╰───────────────╯ ╰───────────────────────────────────────────────╯ ╰────────────────╯ │
└────────────────────────────────────────────────────────────────────────────────────────┘
```

### Layout Math Calculations

All coordinates and sizes are computed inside `View()` in `internal/tui/view.go`:

1. **Terminal Sizing**:
   - `width = m.terminalWidth`
   - `height = m.terminalHeight`
   - `boxHeight = height - bottomBarLines` (where `bottomBarLines` is reserved for status/error messages).

2. **Column Width Ratios**:
   - `shortcutWidth`: If open, `max(width / 4, 14)`. Otherwise, `0`.
   - `availableWidth = width - shortcutWidth`.
   - `leftWidth` (Explorer):
     - If Shortcuts open: `max(width / 2, 25)`.
     - If Shortcuts closed: `(availableWidth * 3) / 5` (a 60% ratio).
   - `rightWidth` (Previewer):
     - If Shortcuts open: `width - shortcutWidth - leftWidth`.
     - If Shortcuts closed: `availableWidth - leftWidth` (the remaining 40%).

3. **Sub-Panes (Explorer bottom area)**:
   - `bottomSectionHeight = 5` (fixed height).
   - `explorerHeight = boxHeight - bottomSectionHeight - 2` (explorer list height).
   - `processWidth = leftWidth / 3` (a 33% split).
   - `clipboardWidth = leftWidth - processWidth` (the remaining 67%).

4. **Composition (Joining)**:
   - **Horizontal Join** (Process & Clipboard):
     `bottomPanes = lipgloss.JoinHorizontal(lipgloss.Top, processBox, clipboardBox)`
   - **Vertical Join** (Explorer & Bottom panes):
     `leftBox = lipgloss.JoinVertical(lipgloss.Left, leftBox, bottomPanes)`
   - **Main Horizontal Join**:
     `content = lipgloss.JoinHorizontal(lipgloss.Top, shortcutBox, leftBox, rightBox)`

---

## 2. Border Sizing Behavior: Lipgloss v1 vs. v2

The core cause of panel border conflicts/gaps during migration is a change in the Lipgloss layout model:

### Lipgloss v1 (Current)
* **Inner Content Sizing**: `.Width(w)` and `.Height(h)` define only the **inner content area** (excluding borders and padding).
* **Border Extra Space**: Borders and padding add to the outside of this block. A style with `.Border(RoundedBorder())` adds `+2` to width (left/right borders) and `+2` to height (top/bottom borders).
* **Math Requirement**: To make a panel occupy exactly `W` visual columns and `H` visual rows on the terminal, the style must subtract `2` for borders:
  ```go
  shortcutStyle := NewStyle().Width(shortcutWidth - 2).Height(boxHeight - 2) // Outer dimensions are W x H
  ```

### Lipgloss v2
* **Outer Boundary Sizing**: `.Width(w)` and `.Height(h)` define the **total outer boundaries** (including padding and borders).
* **Border Inward Space**: Borders and padding are drawn *inward*, shrinking the content area instead of expanding the outer size.
* **Math Requirement**: To make a panel occupy `W` visual columns and `H` visual rows, you pass `W` and `H` directly:
  ```go
  shortcutStyle := NewStyle().Width(shortcutWidth).Height(boxHeight) // Outer dimensions are W x H
  ```

---

## 3. Resolving Border Conflicts (Conventional vs. Shared Borders)

When adjacent panels both draw rounded borders, they create double lines (e.g. `││` next to each other). We can solve this in Lipgloss v2 using two approaches:

### Option A: Standard Rounded Borders
We keep the default rounded borders for all panels. When we migrate to v2, we remove the `- 2` subtraction from all style width/height arguments so the panels align flush with each other. This results in the standard layout but sized correctly.

### Option B: Custom Shared Grid Borders
Adjacent dividing borders are disabled, and we inject T-junction connection characters (`┬`, `┴`, `├`, `┤`) into the border headers. This merges all 5 panels into a singular, clean, connected grid.

#### Shared Border Configurations:
* **Shortcuts**: Standard border, right border connects with T-junctions (`TopRight: "┬"`, `BottomRight: "┴"`).
* **File Explorer**: Left border disabled (`Left: ""`) when shortcuts are open. Bottom border enabled (`Bottom: "─"`) to act as the horizontal divider for the bottom panes.
* **Process & Clipboard**: Top borders disabled (`Top: ""`), using the File Explorer's bottom border as their top boundary. Left borders disabled to merge with adjacent elements.
* **Previewer**: Left border disabled (`Left: ""`), merging with the File Explorer's right border.

---

## Proposed Changes for migrating to Lipgloss v2 (Option B)

If we decide to migrate to Lipgloss v2 with shared borders:

### `internal/tui/utilities.go`
#### [MODIFY] utilities.go
1. Change import path to `"charm.land/lipgloss/v2"`.
2. Update `createTopBorder` to accept `leftCorner` and `rightCorner` strings to render correct T-junctions:
```go
func createTopBorder(title string, width int, color string, align lipgloss.Position, leftCorner, rightCorner string, paneNum string) string {
	barWidth := width - (len([]rune(leftCorner)) + len([]rune(rightCorner)))
    // Place title and pad with "─"
	middle := lipgloss.PlaceHorizontal(barWidth, align, title, lipgloss.WithWhitespaceChars("─"))
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(leftCorner + prefix + middle + rightCorner)
}
```

### `internal/tui/view.go`
#### [MODIFY] view.go
1. Change import path to `"charm.land/lipgloss/v2"`.
2. Define custom `lipgloss.Border` specifications (`shortcutBorder`, `leftBorderOpen`, etc.).
3. Update style definitions in `View()` to remove `- 2` and `- paneBorderPadding` offsets, and assign custom borders. For example:
```go
	shortcutStyle := lipgloss.NewStyle().
		Border(shortcutBorder).
		Width(shortcutWidth).
		Height(boxHeight)
```

---

## Open Questions
1. Would you like to proceed with migrating the codebase to **Lipgloss v2** now?
2. If we migrate, do you prefer **Option A** (individual panels with separate rounded borders) or **Option B** (the merged shared-border grid)?
