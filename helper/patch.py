import sys

def patch_model():
    with open('model.go', 'r') as f:
        content = f.read()

    # 1. Add filteredFiles to TabState
    content = content.replace(
        "filesPath           []string\n",
        "filesPath           []string\n\tfilteredFiles       []string\n"
    )

    # 2. Add search fields to model
    content = content.replace(
        "copyJobCancel             chan struct{}\n}",
        "copyJobCancel             chan struct{}\n\tsearchActive              bool\n\tsearchText                string\n\tsearchCursor              int\n}"
    )

    # 3. Add visibleFiles helper
    helper = """
func (t *TabState) visibleFiles() []string {
	if t.filteredFiles != nil {
		return t.filteredFiles
	}
	return t.filesPath
}
"""
    content = content.replace(
        "func (m *model) curTab() *TabState {",
        helper + "\nfunc (m *model) curTab() *TabState {"
    )

    # 4. Update getMaxVisibleFiles
    content = content.replace(
        "consumed := 2\n\tif m.inputMode {",
        "consumed := 3\n\tif m.inputMode {"
    )

    # 5. Add 's' key to activate search (around line 466)
    s_key = """
		case "s":
			if m.activePane == 1 {
				m.searchActive = true
				m.searchText = ""
				m.searchCursor = 0
				return m, nil
			}
		}
"""
    content = content.replace(
        "m.adjustScroll()\n\t\t}",
        "m.adjustScroll()\n" + s_key,
        1 # Only the first occurrence which is in the main switch
    )

    # 6. Add search input handler before inputMode
    search_handler = """
		if m.searchActive {
			switch msg.String() {
			case "esc":
				m.searchActive = false
				m.searchText = ""
				m.searchCursor = 0
				t := m.curTab()
				t.filteredFiles = nil
				t.cursor = 0
				t.scrollOffset = 0
				m.adjustScroll()
			case "enter":
				m.searchActive = false
			case "backspace":
				runes := []rune(m.searchText)
				if m.searchCursor > 0 {
					runes = append(runes[:m.searchCursor-1], runes[m.searchCursor:]...)
					m.searchCursor--
					m.searchText = string(runes)
					m.applySearchFilter()
				}
			case "left":
				if m.searchCursor > 0 {
					m.searchCursor--
				}
			case "right":
				if m.searchCursor < len([]rune(m.searchText)) {
					m.searchCursor++
				}
			case "home":
				m.searchCursor = 0
			case "end":
				m.searchCursor = len([]rune(m.searchText))
			default:
				char := msg.String()
				if len(char) == 1 || (len(char) > 1 && !strings.HasPrefix(char, "ctrl+")) {
					runes := []rune(m.searchText)
					runes = append(runes[:m.searchCursor], append([]rune(char), runes[m.searchCursor:]...)...)
					m.searchCursor += len([]rune(char))
					m.searchText = string(runes)
					m.applySearchFilter()
				}
			}
			return m, nil
		}

		if m.inputMode {"""
    content = content.replace("\n\t\tif m.inputMode {", "\n" + search_handler)

    # 7. Add search bar rendering to View
    search_bar = """
	// Search bar
	searchIcon := "\ue68f"
	if m.searchActive {
		runes := []rune(m.searchText)
		cur := m.searchCursor
		if cur > len(runes) {
			cur = len(runes)
		}
		before := string(runes[:cur])
		after := string(runes[cur:])
		renderedText += fmt.Sprintf(" %s%s %s█%s%s\\n", Blue, searchIcon, before, after, Reset)
	} else if m.searchText != "" {
		renderedText += fmt.Sprintf(" %s%s %s%s\\n", Gray, searchIcon, m.searchText, Reset)
	} else {
		renderedText += fmt.Sprintf(" %s%s press s to search for file or directory%s\\n", Gray, searchIcon, Reset)
	}
"""
    content = content.replace(
        "renderedText := fmt.Sprintf(\" %s%s: %s%s\\n\", Cyan, getCurrentUser(), t.currentDir, Reset)\n\n\tmaxVisible := m.getMaxVisibleFiles()",
        "renderedText := fmt.Sprintf(\" %s%s: %s%s\\n\", Cyan, getCurrentUser(), t.currentDir, Reset)\n" + search_bar + "\n\tmaxVisible := m.getMaxVisibleFiles()"
    )

    # 8. Replace t.filesPath with t.visibleFiles() in View loop
    view_loop_orig = """	endIdx := t.scrollOffset + maxVisible
	if endIdx > len(t.filesPath) {
		endIdx = len(t.filesPath)
	}

	for i := t.scrollOffset; i < endIdx; i++ {
		list := t.filesPath[i]"""
    view_loop_new = """	files := t.visibleFiles()
	endIdx := t.scrollOffset + maxVisible
	if endIdx > len(files) {
		endIdx = len(files)
	}

	for i := t.scrollOffset; i < endIdx; i++ {
		list := files[i]"""
    content = content.replace(view_loop_orig, view_loop_new)

    # 9. Replace t.filesPath with t.visibleFiles() in Metadata rendering
    meta_orig = """	// Metadata bar pinned at bottom inside the border
	if len(t.filesPath) > 0 && t.cursor >= 0 && t.cursor < len(t.filesPath) {
		meta := getFileMetadataInfo(t.currentDir, t.filesPath[t.cursor])"""
    meta_new = """	// Metadata bar pinned at bottom inside the border
	files2 := t.visibleFiles()
	if len(files2) > 0 && t.cursor >= 0 && t.cursor < len(files2) {
		meta := getFileMetadataInfo(t.currentDir, files2[t.cursor])"""
    content = content.replace(meta_orig, meta_new)

    with open('model.go', 'w') as f:
        f.write(content)

def patch_utilities():
    with open('utilities.go', 'r') as f:
        content = f.read()

    # 1. Add applySearchFilter
    search_filter = """
func (m *model) applySearchFilter() {
	t := m.curTab()
	if m.searchText == "" {
		t.filteredFiles = nil
	} else {
		query := strings.ToLower(m.searchText)
		t.filteredFiles = t.filteredFiles[:0]
		for _, name := range t.filesPath {
			if strings.Contains(strings.ToLower(name), query) {
				t.filteredFiles = append(t.filteredFiles, name)
			}
		}
	}
	t.cursor = 0
	t.scrollOffset = 0
	m.adjustScroll()
}
"""
    content = content + "\n" + search_filter

    # 2. Clear filter in reLoadDir
    content = content.replace(
        "t.filesPath = t.filesPath[:0]",
        "t.filesPath = t.filesPath[:0]\n\tt.filteredFiles = nil\n\tm.searchActive = false\n\tm.searchText = \"\"\n\tm.searchCursor = 0"
    )

    # 3. Clear filter in refreshAllTabs
    content = content.replace(
        "t.filesPath = t.filesPath[:0]\n\t\tfor _, entry",
        "t.filesPath = t.filesPath[:0]\n\t\tt.filteredFiles = nil\n\t\tfor _, entry"
    )

    with open('utilities.go', 'w') as f:
        f.write(content)

if __name__ == '__main__':
    patch_model()
    patch_utilities()
