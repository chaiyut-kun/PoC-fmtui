package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"unicode"

	tea "charm.land/bubbletea/v2"
)

func (m model) handleShortcutKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	t := m.curTab()
	if m.shortcutPaneOpen && m.activePane == 0 {
		switch msg.String() {
		case "1":
			m.activePane = 0
		case "2":
			m.activePane = 1
		case "3":
			m.activePane = 2
		case "ctrl+c", "q", "Q":
			_ = clearTempDirFile()
			m.quitting = true
			return m, tea.Quit
		case "E", "e":
			m.quitting = true
			_ = writeTempDirFile(t.currentDir)
			return m, tea.Quit
		case "`":
			m.err = nil
			m.helpMenuOpen = true
		case "d", "D":
			if m.shortcutTab == 1 && len(m.bookmarks) > 0 {
				m.err = nil
				m.deleteBookmarkConfirmOpen = true
			}
		case "up", "k":
			limit := len(m.shortcuts) - 1
			if m.shortcutTab == 1 {
				limit = len(m.bookmarks) - 1
			}
			if limit > 0 {
				if m.shortcutCursor > 0 {
					m.shortcutCursor--
				} else {
					m.shortcutCursor = limit
				}
			}
		case "down", "j":
			limit := len(m.shortcuts) - 1
			if m.shortcutTab == 1 {
				limit = len(m.bookmarks) - 1
			}
			if limit > 0 {
				if m.shortcutCursor < limit {
					m.shortcutCursor++
				} else {
					m.shortcutCursor = 0
				}
			}
		case "[":
			m.shortcutTab = 0
			m.shortcutCursor = 0
		case "]":
			m.shortcutTab = 1
			m.shortcutCursor = 0
		case "enter", "right", "l":
			var targetDir string
			var targetName string
			if m.shortcutTab == 0 {
				if m.shortcutCursor >= 0 && m.shortcutCursor < len(m.shortcuts) {
					targetDir = m.shortcuts[m.shortcutCursor].Path
					targetName = m.shortcuts[m.shortcutCursor].Name
				}
			} else {
				if m.shortcutCursor >= 0 && m.shortcutCursor < len(m.bookmarks) {
					targetDir = m.bookmarks[m.shortcutCursor].Path
					targetName = m.bookmarks[m.shortcutCursor].Name
				}
			}
			if targetDir != "" {
				entries, err := os.ReadDir(targetDir)
				if err != nil {
					m.err = err
				} else {
					m.reLoadDir(targetDir, entries)
					m.adjustScroll()
					m.activePane = 1
					m.statusMsg = fmt.Sprintf("Jumped to %s", targetName)
				}
			}
		case "esc", "left", "h", "ctrl+b", "\\":
			m.shortcutPaneOpen = false
			m.activePane = 1
		}
		return m, nil
	}

	return m, nil
}

func (m model) handleSearchKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
		default:
			oldText := m.searchText
			m.searchText, m.searchCursor = handleTextEdit(msg.String(), m.searchText, m.searchCursor)
			if m.searchText != oldText {
				m.applySearchFilter()
			}
		}
		return m, nil
	}

	return m, nil
}

func (m model) handleInputKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	t := m.curTab()
	if m.inputMode {
		switch msg.String() {
		case "enter":
			m.inputMode = false
			m.inputCursor = 0
			if m.inputText == "" {
				m.isRenaming = false
				break
			}

			if m.isRenaming {
				if len(t.visibleFiles()) > 0 {
					oldName := t.visibleFiles()[t.cursor]
					oldPath := findAbsPath(t.currentDir, oldName)
					newPath := filepath.Join(t.currentDir, m.inputText)
					if err := os.Rename(oldPath, newPath); err != nil {
						m.err = err
					} else {
						m.pushAction(ActionRecord{
							Type:    ActionRename,
							OldPath: oldPath,
							NewPath: newPath,
						})
						m.reloadCurrentDir()
					}
				}
				m.isRenaming = false
				break
			}

			filePath := filepath.Join(t.currentDir, m.inputText)

			if m.isCreatingDir {
				err := os.Mkdir(filePath, 0755)
				if err != nil {
					m.err = err
					break
				}
				m.pushAction(ActionRecord{
					Type:         ActionCreate,
					OriginalPath: filePath,
					IsDir:        true,
				})
			} else {
				file, err := os.Create(filePath)
				if err != nil {
					m.err = err
					break
				}
				file.Close()
				m.pushAction(ActionRecord{
					Type:         ActionCreate,
					OriginalPath: filePath,
					IsDir:        false,
				})
			}

			m.reloadCurrentDir()
			m.adjustScroll()
		case "esc":
			m.inputMode = false
			m.inputText = ""
			m.inputCursor = 0
			m.isRenaming = false
		default:
			m.inputText, m.inputCursor = handleTextEdit(msg.String(), m.inputText, m.inputCursor)
		}
		return m, nil
	}

	return m, nil
}

// handleTextEdit applies a single key event to a (text, cursor) pair.
// Handles: printable chars, backspace, left/right/home/end.
// ponytail: pure function — no model mutation, easy to test.
func handleTextEdit(key, text string, cursor int) (string, int) {
	runes := []rune(text)
	switch key {
	case "left":
		if cursor > 0 {
			cursor--
		}
	case "right":
		if cursor < len(runes) {
			cursor++
		}
	case "home":
		cursor = 0
	case "end":
		cursor = len(runes)
	case "backspace":
		if cursor > 0 {
			runes = append(runes[:cursor-1], runes[cursor:]...)
			text = string(runes)
			cursor--
		}
	case "alt+left", "ctrl+left", "alt+b":
		if cursor > 0 {
			newCursor := cursor - 1
			for newCursor >= 0 && unicode.IsSpace(runes[newCursor]) {
				newCursor--
			}
			for newCursor >= 0 && !unicode.IsSpace(runes[newCursor]) {
				newCursor--
			}
			cursor = newCursor + 1
		}
	case "alt+right", "ctrl+right", "alt+f":
		if cursor < len(runes) {
			newCursor := cursor
			for newCursor < len(runes) && unicode.IsSpace(runes[newCursor]) {
				newCursor++
			}
			for newCursor < len(runes) && !unicode.IsSpace(runes[newCursor]) {
				newCursor++
			}
			cursor = newCursor
		}
	case "ctrl+w", "alt+backspace", "ctrl+backspace":
		if cursor > 0 {
			newCursor := cursor - 1
			for newCursor >= 0 && unicode.IsSpace(runes[newCursor]) {
				newCursor--
			}
			for newCursor >= 0 && !unicode.IsSpace(runes[newCursor]) {
				newCursor--
			}
			newCursor++
			runes = append(runes[:newCursor], runes[cursor:]...)
			text = string(runes)
			cursor = newCursor
		}
	case "alt+delete", "ctrl+delete":
		if cursor < len(runes) {
			newCursor := cursor
			for newCursor < len(runes) && unicode.IsSpace(runes[newCursor]) {
				newCursor++
			}
			for newCursor < len(runes) && !unicode.IsSpace(runes[newCursor]) {
				newCursor++
			}
			runes = append(runes[:cursor], runes[newCursor:]...)
			text = string(runes)
		}
	case "ctrl+u":
		runes = runes[cursor:]
		text = string(runes)
		cursor = 0
	case "ctrl+k":
		runes = runes[:cursor]
		text = string(runes)
	default:
		r := []rune(key)
		if len(r) == 1 {
			runes = append(runes[:cursor], append([]rune{r[0]}, runes[cursor:]...)...)
			text = string(runes)
			cursor++
		}
	}
	return text, cursor
}

// quickFindResultsMsg delivers async search results back to the Update loop.
type quickFindResultsMsg struct {
	results []ShortcutEntry
}

// quickFindSearchCmd runs the filesystem walk in a goroutine and returns a tea.Cmd.
// ponytail: one goroutine per keystroke, cancelled naturally when results arrive;
// ceiling: debounce timer or dedicated search goroutine with cancellation channel.
func (m model) quickFindSearchCmd(query string) tea.Cmd {
	return func() tea.Msg {
		return quickFindResultsMsg{results: m.buildQuickFindResults(query)}
	}
}

func (m model) handleQuickFindKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Handle selection navigation keys only if selection mode is active
	if m.quickFindSelecting {
		switch key {
		case "up", "k":
			if m.quickFindSelected > 0 {
				m.quickFindSelected--
			}
			return m, nil
		case "down", "j":
			if m.quickFindSelected < len(m.quickFindResults)-1 {
				m.quickFindSelected++
			}
			return m, nil
		}
	}

	switch key {
	case "esc":
		m.quickFindOpen = false
		m.quickFindText = ""
		m.quickFindResults = nil
		m.quickFindSelecting = false

	case "enter":
		if m.quickFindSelecting {
			if m.quickFindSelected >= 0 && m.quickFindSelected < len(m.quickFindResults) {
				path := m.quickFindResults[m.quickFindSelected].Path
				fi, err := os.Stat(path)
				if err == nil {
					if fi.IsDir() {
						entries, err := os.ReadDir(path)
						if err != nil {
							m.err = err
						} else {
							m.reLoadDir(path, entries)
							m.adjustScroll()
						}
					} else {
						// File jump: load parent directory, then search index to focus it
						dir := filepath.Dir(path)
						entries, err := os.ReadDir(dir)
						if err != nil {
							m.err = err
						} else {
							m.reLoadDir(dir, entries)
							t := m.curTab()
							fileName := fi.Name()
							for idx, f := range t.filesPath {
								if f == fileName {
									t.cursor = idx
									break
								}
							}
							m.adjustScroll()
						}
					}
				}
			}
			m.quickFindOpen = false
			m.quickFindText = ""
			m.quickFindResults = nil
			m.quickFindSelecting = false
		} else {
			if len(m.quickFindResults) > 0 {
				m.quickFindSelecting = true
				m.quickFindSelected = 0
			}
		}

	case "tab":
		if len(m.quickFindResults) > 0 {
			m.quickFindSelecting = !m.quickFindSelecting
		}

	case "up":
		if len(m.quickFindResults) > 0 {
			m.quickFindSelecting = true
			m.quickFindSelected = len(m.quickFindResults) - 1
		}

	case "down":
		if len(m.quickFindResults) > 0 {
			m.quickFindSelecting = true
			m.quickFindSelected = 0
		}

	default:
		if !m.quickFindSelecting {
			m.quickFindText, m.quickFindCursor = handleTextEdit(key, m.quickFindText, m.quickFindCursor)
			m.quickFindSelected = 0
			// Walk runs in a goroutine; results arrive via quickFindResultsMsg.
			return m, m.quickFindSearchCmd(m.quickFindText)
		}
	}
	return m, nil
}
