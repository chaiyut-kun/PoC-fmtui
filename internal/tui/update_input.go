package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
			if m.shortcutCursor > 0 {
				m.shortcutCursor--
			}
		case "down", "j":
			limit := len(m.shortcuts) - 1
			if m.shortcutTab == 1 {
				limit = len(m.bookmarks) - 1
			}
			if m.shortcutCursor < limit {
				m.shortcutCursor++
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
					m.shortcutPaneOpen = false
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
		case "left":
			if m.inputCursor > 0 {
				m.inputCursor--
			}
		case "right":
			if m.inputCursor < len([]rune(m.inputText)) {
				m.inputCursor++
			}
		case "home":
			m.inputCursor = 0
		case "end":
			m.inputCursor = len([]rune(m.inputText))
		case "backspace":
			runes := []rune(m.inputText)
			if m.inputCursor > 0 {
				runes = append(runes[:m.inputCursor-1], runes[m.inputCursor:]...)
				m.inputText = string(runes)
				m.inputCursor--
			}
		default:
			r := []rune(msg.String())
			if len(r) == 1 {
				runes := []rune(m.inputText)
				runes = append(runes[:m.inputCursor], append([]rune{r[0]}, runes[m.inputCursor:]...)...)
				m.inputText = string(runes)
				m.inputCursor++
			}
		}
		return m, nil
	}

	return m, nil
}
