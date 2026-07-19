import sys

with open('model.go', 'r') as f:
    content = f.read()

# 1. Update getMaxVisibleFiles to consume 5 (path + 3 for search + 1 metadata)
content = content.replace(
    "consumed := 3 // path + search bar + metadata",
    "consumed := 5 // path + 3 for search box + metadata"
)

# 2. Extract width calculation and move to top of View()
width_calc = """	width := m.terminalWidth
	height := m.terminalHeight
	if width < 20 {
		width = 80
	}
	if height < 5 {
		height = 20
	}

	shortcutWidth := 0
	if m.shortcutPaneOpen {
		shortcutWidth = width / 4
		if shortcutWidth < 14 {
			shortcutWidth = 14
		}
	}
	var leftWidth int
	if m.shortcutPaneOpen {
		leftWidth = width / 2
		if leftWidth < 10 {
			leftWidth = 10
		}
	} else {
		availableWidth := width - shortcutWidth
		leftWidth = (availableWidth * 2) / 3
		if leftWidth < 10 {
			leftWidth = availableWidth / 2
		}
	}
"""

# Find where width is currently calculated (around line 1584)
# Remove the duplicated shortcutWidth and leftWidth calculation lower down.
# Actually, it's easier to just do it via string replacement.

# Let's write a targeted replace.
