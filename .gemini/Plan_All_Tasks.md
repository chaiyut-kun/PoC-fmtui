# Plan: Active Pane Switching, Borders, and Preview Scrolling

This plan details how to implement active pane toggling (using `[` and `]`), dynamic border coloring (Blue for active, Gray for inactive), and vertical scrolling inside the preview pane.

---

## 1. State Additions

#### [model.go](file:///data/coding/Go/tui/list2/model.go)
Add state tracking variables to `model`:
- `activePane int` (0 = Left Explorer, 1 = Right Previewer)
- `previewScrollOffset int` (offset of vertical scrolling in the preview pane)

---

## 2. Event Handling Updates

- **Active Pane Switch**:
  - Intercept `[` and `]` keypresses in normal mode to toggle `activePane` between `0` (Left) and `1` (Right).
- **Context-Aware Navigation**:
  - In `Update()`'s normal mode key handlers for `up`/`k` and `down`/`j`:
    - If `activePane == 0` (Left Explorer): move the cursor and adjust left scroll offset (`m.adjustScroll()`).
    - If `activePane == 1` (Right Previewer): increment/decrement `previewScrollOffset` and adjust right scroll bounds.
  - Reset `previewScrollOffset = 0` whenever the cursor moves on the left pane (so highlighting a new file resets the preview scroll position).

---

## 3. View & Layout Updates

- **Dynamic Border Styles**:
  - Define an active border style (using Blue foreground `#5555FF` or ANSI Blue `4`) and an inactive border style (using Gray foreground `#555555` or White `#FFFFFF`).
  - Assign the active border style to the pane matching `activePane`, and the inactive style to the other.
- **Scrollable Preview Pane**:
  - Refactored `getPreviewText` to return all lines as a slice of strings `[]string`.
  - Slice the preview lines using `[previewScrollOffset : previewScrollOffset + maxVisible]` before rendering, ensuring no text overflows the border.

---

## 4. Horizontal Fit & Line Truncation (Preview Pattern Update)

- **Ignore Overflow Data**:
  - To prevent line wrapping from breaking the Lipgloss box layout, each line in the preview pane is processed before rendering:
    - Tab characters (`\t`) are expanded to 4 spaces.
    - The line is truncated to fit precisely within the inner width of the preview pane (`rightWidth - 2` characters).
