# Implementation Plan: Preview Line Truncation (Horizontal Fit)

This plan details how to truncate preview lines horizontally to ensure they fit within the preview pane's boundaries, preventing wrapping and vertical border stretching.

---

## Proposed Changes

### 1. Line Truncation Logic in View

#### [MODIFY] [model.go](file:///data/coding/Go/tui/list2/model.go)
- In `View()`'s right-pane construction logic:
  - Calculate `maxWidth := rightWidth - 2`.
  - For each line in `previewLines[m.previewScrollOffset : endLine]`:
    - Replace tabs `\t` with 4 spaces.
    - Convert to runes and slice up to `maxWidth` characters if the line exceeds it.
  - Join these truncated lines back with newlines to form `rightText`.

```go
		// Inside View() right-pane loop/construction:
		maxWidth := rightWidth - 2
		if maxWidth < 1 {
			maxWidth = 1
		}
		var truncatedLines []string
		for _, line := range previewLines[m.previewScrollOffset:endLine] {
			// Expand tabs to spaces
			expanded := strings.ReplaceAll(line, "\t", "    ")
			runes := []rune(expanded)
			if len(runes) > maxWidth {
				truncatedLines = append(truncatedLines, string(runes[:maxWidth]))
			} else {
				truncatedLines = append(truncatedLines, expanded)
			}
		}
		rightText = strings.Join(truncatedLines, "\n")
```

---

## Verification Plan

### Automated Tests
- Add a new unit test verifying line truncation logic to `main_test.go` and run `go test ./...`.

### Manual Verification
- Compile and run `./list2`.
- Highlight a file containing long lines or tabs.
- Verify that the lines are cut off precisely at the right border, never wrap to the next line, and do not cause any vertical rendering glitches.
