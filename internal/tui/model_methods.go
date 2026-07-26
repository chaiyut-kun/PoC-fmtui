package tui

import (
	"os"
	"strings"
)

func (m *model) reLoadDir(changeDir string, entries []os.DirEntry) {
	t := m.curTab()
	t.currentDir = changeDir
	t.filesPath = t.filesPath[:0]
	t.filteredFiles = nil
	m.searchActive = false
	m.searchText = ""
	m.searchCursor = 0
	for _, entry := range entries {
		if !m.showHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		t.filesPath = append(t.filesPath, entry.Name())
	}

	t.cursor = 0
	t.scrollOffset = 0
	t.previewScrollOffset = 0
	m.selected = make(map[int]struct{})
}

func (m *model) reloadCurrentDir() {
	m.refreshAllTabs()
	m.selected = make(map[int]struct{})
}

func (m *model) refreshAllTabs() {
	for i := range m.tabs {
		t := &m.tabs[i]
		entries, err := os.ReadDir(t.currentDir)
		if err != nil {
			continue
		}

		var oldCursorFile string
		if t.cursor >= 0 && t.cursor < len(t.filesPath) {
			oldCursorFile = t.filesPath[t.cursor]
		}

		t.filesPath = t.filesPath[:0]
		t.filteredFiles = nil
		m.searchActive = false
		m.searchText = ""
		m.searchCursor = 0
		for _, entry := range entries {
			if !m.showHidden && strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			t.filesPath = append(t.filesPath, entry.Name())
		}

		newCursor := t.cursor
		if oldCursorFile != "" {
			for idx, f := range t.filesPath {
				if f == oldCursorFile {
					newCursor = idx
					break
				}
			}
			// If not found (e.g. file was deleted or moved), newCursor stays at the same index
		}

		t.cursor = newCursor
		if t.cursor >= len(t.filesPath) {
			t.cursor = len(t.filesPath) - 1
		}
		if t.cursor < 0 {
			t.cursor = 0
		}
	}
}

func (m *model) applySearchFilter() {
	t := m.curTab()
	if m.searchText == "" {
		t.filteredFiles = nil
	} else {
		query := strings.ToLower(m.searchText)
		t.filteredFiles = make([]string, 0)
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
