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
			if m.shortcutSection == 1 && len(m.bookmarks) > 0 {
				m.err = nil
				m.deleteBookmarkConfirmOpen = true
			}
		case "tab":
			if m.shortcutSection == 0 {
				m.shortcutSection = 1
			} else {
				m.shortcutSection = 0
			}
			m.shortcutCursor = 0
			m.bookmarkScrollOffset = 0
		case "up", "k":
			if m.shortcutSection == 0 {
				limit := len(m.shortcuts) - 1
				if limit > 0 {
					if m.shortcutCursor > 0 {
						m.shortcutCursor--
					} else {
						m.shortcutCursor = limit
					}
				}
			} else {
				limit := len(m.bookmarks) - 1
				if limit > 0 {
					if m.shortcutCursor > 0 {
						m.shortcutCursor--
					} else {
						m.shortcutCursor = limit
						m.bookmarkScrollOffset = max(limit-1, 0)
					}
					// scroll up if cursor above viewport
					if m.shortcutCursor < m.bookmarkScrollOffset {
						m.bookmarkScrollOffset = m.shortcutCursor
					}
				}
			}
		case "down", "j":
			if m.shortcutSection == 0 {
				limit := len(m.shortcuts) - 1
				if limit > 0 {
					if m.shortcutCursor < limit {
						m.shortcutCursor++
					} else {
						m.shortcutCursor = 0
					}
				}
			} else {
				limit := len(m.bookmarks) - 1
				if limit > 0 {
					if m.shortcutCursor < limit {
						m.shortcutCursor++
					} else {
						m.shortcutCursor = 0
						m.bookmarkScrollOffset = 0
					}
				}
			}
		case "enter", "right", "l":
			var targetDir string
			var targetName string
			if m.shortcutSection == 0 {
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
			m.searchInput.Blur()
			m.searchInput.SetValue("")
			t := m.curTab()
			t.filteredFiles = nil
			t.cursor = 0
			t.scrollOffset = 0
			m.adjustScroll()
			return m, nil
		case "enter":
			m.searchActive = false
			m.searchInput.Blur()
			return m, nil
		default:
			oldText := m.searchInput.Value()
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			if m.searchInput.Value() != oldText {
				m.applySearchFilter()
			}
			return m, cmd
		}
	}

	return m, nil
}

func (m model) handleInputKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	t := m.curTab()
	if m.inputMode {
		switch msg.String() {
		case "enter":
			m.inputMode = false
			m.fileInput.Blur()
			val := m.fileInput.Value()
			m.fileInput.SetValue("")
			if val == "" {
				m.isRenaming = false
				return m, nil
			}

			if m.isRenaming {
				if len(t.visibleFiles()) > 0 {
					oldName := t.visibleFiles()[t.cursor]
					oldPath := findAbsPath(t.currentDir, oldName)
					newPath := filepath.Join(t.currentDir, val)
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
				return m, nil
			}

			filePath := filepath.Join(t.currentDir, val)

			// Auto-detect: no extension = directory, has extension = file
			cleanVal := strings.TrimSuffix(val, "/")
			filePath = filepath.Join(t.currentDir, cleanVal)
			isDir := filepath.Ext(cleanVal) == ""

			if isDir {
				err := os.Mkdir(filePath, 0755)
				if err != nil {
					m.err = err
					return m, nil
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
					return m, nil
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
			return m, nil
		case "esc":
			m.inputMode = false
			m.fileInput.Blur()
			m.fileInput.SetValue("")
			m.isRenaming = false
			return m, nil
		default:
			var cmd tea.Cmd
			m.fileInput, cmd = m.fileInput.Update(msg)
			return m, cmd
		}
	}

	return m, nil
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
	maxResults := max(min(quickSearchMaxResultDefault, m.terminalHeight-8), 5)

	// Handle selection navigation keys only if selection mode is active
	if m.quickFindSelecting {
		switch key {
		case "up", "k":
			if m.quickFindSelected > 0 {
				m.quickFindSelected--
				if m.quickFindSelected < m.quickFindScrollOffset {
					m.quickFindScrollOffset = m.quickFindSelected
				}
			}
			return m, nil
		case "down", "j":
			if m.quickFindSelected < len(m.quickFindResults)-1 {
				m.quickFindSelected++
				if m.quickFindSelected >= m.quickFindScrollOffset+maxResults {
					m.quickFindScrollOffset = m.quickFindSelected - maxResults + 1
				}
			}
			return m, nil
		}
	}

	switch key {
	case "esc":
		m.quickFindOpen = false
		m.quickFindInput.Blur()
		m.quickFindInput.SetValue("")
		m.quickFindResults = nil
		m.quickFindSelecting = false
		m.quickFindScrollOffset = 0

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
			m.quickFindInput.Blur()
			m.quickFindInput.SetValue("")
			m.quickFindResults = nil
			m.quickFindSelecting = false
			m.quickFindScrollOffset = 0
		} else {
			if len(m.quickFindResults) > 0 {
				m.quickFindSelecting = true
				m.quickFindSelected = 0
				m.quickFindScrollOffset = 0
				m.quickFindInput.Blur()
			}
		}

	case "tab":
		if len(m.quickFindResults) > 0 {
			m.quickFindSelecting = !m.quickFindSelecting
			if m.quickFindSelecting {
				m.quickFindInput.Blur()
			} else {
				m.quickFindInput.Focus()
			}
		}

	case "up":
		if len(m.quickFindResults) > 0 {
			m.quickFindSelecting = true
			m.quickFindSelected = len(m.quickFindResults) - 1
			m.quickFindScrollOffset = max(len(m.quickFindResults)-maxResults, 0)
			m.quickFindInput.Blur()
		}

	case "down":
		if len(m.quickFindResults) > 0 {
			m.quickFindSelecting = true
			m.quickFindSelected = 0
			m.quickFindScrollOffset = 0
			m.quickFindInput.Blur()
		}

	default:
		if !m.quickFindSelecting {
			var cmd tea.Cmd
			m.quickFindInput, cmd = m.quickFindInput.Update(msg)
			m.quickFindSelected = 0
			m.quickFindScrollOffset = 0
			// Walk runs in a goroutine; results arrive via quickFindResultsMsg.
			return m, tea.Batch(cmd, m.quickFindSearchCmd(m.quickFindInput.Value()))
		}
	}
	return m, nil
}
