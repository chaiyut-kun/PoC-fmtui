package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case editorFinishedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.reloadCurrentDir()
		}
		return m, nil

	case subshellFinishedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.reloadCurrentDir()
		}
		return m, nil

	case quickFindResultsMsg:
		// Discard stale results if the overlay was closed while walk was running.
		if m.quickFindOpen {
			m.quickFindResults = msg.results
		}
		return m, nil

	case copyProgressMsg:
		m.copyJobProgress = msg.progress
		return m, m.listenCopyProgress()

	case copyFinishedMsg:
		m.copyJobActive = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.statusMsg = fmt.Sprintf("Copied %s content to clipboard", m.copyJobFileName)
		}
		return m, nil

	case tea.WindowSizeMsg:
		// + 1 for fits with terminal height (reduce padding at bottom)
		m.terminalHeight = msg.Height
		m.terminalWidth = msg.Width
		m.adjustScroll()
		return m, nil

	case tea.MouseMsg:
		mouseEvent := msg.Mouse()
		t := m.curTab()
		switch mouseEvent.Button {
		case tea.MouseWheelUp:
			if m.helpMenuOpen {
				if m.helpScrollOffset > 0 {
					m.helpScrollOffset--
				}
			} else if m.activePane == 0 && m.shortcutPaneOpen {
				if m.shortcutSection == 0 {
					if m.shortcutCursor > 0 {
						m.shortcutCursor--
					}
				} else {
					if m.shortcutCursor > 0 {
						m.shortcutCursor--
						if m.shortcutCursor < m.bookmarkScrollOffset {
							m.bookmarkScrollOffset = m.shortcutCursor
						}
					}
				}
			} else if m.activePane == 1 {
				if t.cursor > 0 {
					t.cursor--
					m.adjustScroll()
				}
			} else if m.activePane == 2 {
				if t.previewScrollOffset > 0 {
					t.previewScrollOffset--
				}
			}
			return m, nil

		case tea.MouseWheelDown:
			if m.helpMenuOpen {
				m.helpScrollOffset++
			} else if m.activePane == 0 && m.shortcutPaneOpen {
				if m.shortcutSection == 0 {
					limit := len(m.shortcuts) - 1
					if m.shortcutCursor < limit {
						m.shortcutCursor++
					}
				} else {
					limit := len(m.bookmarks) - 1
					if m.shortcutCursor < limit {
						m.shortcutCursor++
					}
				}
			} else if m.activePane == 1 {
				if t.cursor < len(t.visibleFiles())-1 {
					t.cursor++
					m.adjustScroll()
				}
			} else if m.activePane == 2 {
				t.previewScrollOffset++
			}
			return m, nil

		case tea.MouseLeft:
			width := m.terminalWidth
			if width < 20 {
				width = DefaultScreenWidth
			}
			shortcutWidth := 0
			if m.shortcutPaneOpen {
				// 8 is reduce unit 
				shortcutWidth = max(width/shortcutWidthRatio-8, minShortcutWidth)
			}
			availableWidth := width - shortcutWidth
			leftWidth := (availableWidth * leftPaneRatioNumerator) / leftPaneRatioDenominator
			if leftWidth < minLeftPaneWidth {
				leftWidth = availableWidth / openShortcutPaneRatio
			}

			if m.shortcutPaneOpen {
				if mouseEvent.X < shortcutWidth {
					m.activePane = 0
				} else if mouseEvent.X < shortcutWidth+leftWidth {
					m.activePane = 1
				} else {
					m.activePane = 2
				}
			} else {
				if mouseEvent.X < leftWidth {
					m.activePane = 1
				} else {
					m.activePane = 2
				}
			}
			return m, nil
		}

	case tea.KeyPressMsg:
		// Global overrides
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

		// Route based on active UI state
		if m.quickFindOpen {
			return m.handleQuickFindKeys(msg)
		}
		if m.isAnyMenuOpen() {
			return m.handleMenuKeys(msg)
		}
		if m.shortcutPaneOpen && m.activePane == 0 {
			return m.handleShortcutKeys(msg)
		}
		if m.searchActive {
			return m.handleSearchKeys(msg)
		}
		if m.inputMode {
			return m.handleInputKeys(msg)
		}

		// Default Explorer Mode
		return m.handleExplorerKeys(msg)
	}

	var cmd tea.Cmd
	var cmds []tea.Cmd

	if m.inputMode {
		m.fileInput, cmd = m.fileInput.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.searchActive {
		m.searchInput, cmd = m.searchInput.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.quickFindOpen {
		m.quickFindInput, cmd = m.quickFindInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}
