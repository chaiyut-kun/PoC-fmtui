package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

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
		m.terminalHeight = msg.Height
		m.terminalWidth = msg.Width
		m.adjustScroll()
		return m, nil

	case tea.MouseMsg:
		mEvt := msg.Mouse()
		t := m.curTab()
		switch mEvt.Button {
		case tea.MouseWheelUp:
			if m.activePane == 0 && m.shortcutPaneOpen {
				if m.shortcutCursor > 0 {
					m.shortcutCursor--
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
			if m.activePane == 0 && m.shortcutPaneOpen {
				limit := len(m.shortcuts) - 1
				if m.shortcutTab == 1 {
					limit = len(m.bookmarks) - 1
				}
				if m.shortcutCursor < limit {
					m.shortcutCursor++
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
				width = 80
			}
			shortcutWidth := 0
			if m.shortcutPaneOpen {
				shortcutWidth = width / 4
				if shortcutWidth < 14 {
					shortcutWidth = 14
				}
			}
			availableWidth := width - shortcutWidth
			leftWidth := (availableWidth * 2) / 3
			if leftWidth < 10 {
				leftWidth = availableWidth / 2
			}

			if m.shortcutPaneOpen {
				if mEvt.X < shortcutWidth {
					m.activePane = 0
				} else if mEvt.X < shortcutWidth+leftWidth {
					m.activePane = 1
				} else {
					m.activePane = 2
				}
			} else {
				if mEvt.X < leftWidth {
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

	return m, nil
}

func (m model) handleMenuKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	t := m.curTab()
	if m.copyMenuOpen {
		switch msg.String() {
		case "p", "P":
			m.copyMenuOpen = false
			if len(t.visibleFiles()) > 0 {
				absPath := findAbsPath(t.currentDir, t.visibleFiles()[t.cursor])
				if err := copyToClipboard(absPath); err != nil {
					m.err = err
				}
			}
		case "n", "N":
			m.copyMenuOpen = false
			if len(t.visibleFiles()) > 0 {
				name := t.visibleFiles()[t.cursor]
				if !isDir(t.currentDir, name) {
					name = strings.TrimSuffix(name, filepath.Ext(name))
				}
				if err := copyToClipboard(name); err != nil {
					m.err = err
				}
			}
		case "e", "E":
			m.copyMenuOpen = false
			if len(t.visibleFiles()) > 0 {
				name := t.visibleFiles()[t.cursor]
				if err := copyToClipboard(name); err != nil {
					m.err = err
				}
			}
		case "s", "S":
			m.copyMenuOpen = false
			treeStr := generateDirectoryStructure(t.currentDir, m.showHidden)
			if err := copyToClipboard(treeStr); err != nil {
				m.err = err
			} else {
				m.statusMsg = "Copied CWD directory structure to clipboard"
			}
		case "c", "C":
			m.copyMenuOpen = false
			if len(t.visibleFiles()) > 0 {
				name := t.visibleFiles()[t.cursor]
				absPath := findAbsPath(t.currentDir, name)
				if isDir(t.currentDir, name) {
					m.err = fmt.Errorf("cannot copy content: %s is a directory", name)
					break
				}
				return m, m.startCopyContentJob(absPath, name)
			}
		default:
			// Any other key closes the menu (including esc)
			m.copyMenuOpen = false
		}
		return m, nil
	}

	if m.bookmarkMenuOpen {
		switch msg.String() {
		case "s", "S":
			m.bookmarkMenuOpen = false
			if len(t.visibleFiles()) > 0 {
				name := t.visibleFiles()[t.cursor]
				absPath := findAbsPath(t.currentDir, name)
				if isDir(t.currentDir, name) {
					baseName := filepath.Base(absPath)
					if baseName == "" || baseName == "." || baseName == "/" {
						baseName = absPath
					}
					m.bookmarks = append(m.bookmarks, ShortcutEntry{
						Name: baseName,
						Path: absPath,
						Icon: "📁",
					})
					if err := saveBookmarks(m.bookmarks); err != nil {
						m.err = err
					} else {
						m.statusMsg = fmt.Sprintf("Bookmarked directory: %s", baseName)
					}
				} else {
					m.err = fmt.Errorf("selected item is not a directory")
				}
			} else {
				m.err = fmt.Errorf("no item selected")
			}
		case "c", "C":
			m.bookmarkMenuOpen = false
			absPath := t.currentDir
			baseName := filepath.Base(absPath)
			if baseName == "" || baseName == "." || baseName == "/" {
				baseName = absPath
			}
			m.bookmarks = append(m.bookmarks, ShortcutEntry{
				Name: baseName,
				Path: absPath,
				Icon: "📁",
			})
			if err := saveBookmarks(m.bookmarks); err != nil {
				m.err = err
			} else {
				m.statusMsg = fmt.Sprintf("Bookmarked current directory: %s", baseName)
			}
		default:
			m.bookmarkMenuOpen = false
		}
		return m, nil
	}

	if m.deleteBookmarkConfirmOpen {
		switch msg.String() {
		case "y", "Y":
			m.deleteBookmarkConfirmOpen = false
			if m.shortcutCursor >= 0 && m.shortcutCursor < len(m.bookmarks) {
				removedName := m.bookmarks[m.shortcutCursor].Name
				m.bookmarks = append(m.bookmarks[:m.shortcutCursor], m.bookmarks[m.shortcutCursor+1:]...)
				if err := saveBookmarks(m.bookmarks); err != nil {
					m.err = err
				} else {
					m.statusMsg = fmt.Sprintf("Removed bookmark: %s", removedName)
				}
				// Clamp cursor
				if m.shortcutCursor >= len(m.bookmarks) {
					m.shortcutCursor = len(m.bookmarks) - 1
				}
				if m.shortcutCursor < 0 {
					m.shortcutCursor = 0
				}
			}
		default:
			m.deleteBookmarkConfirmOpen = false
		}
		return m, nil
	}

	if m.deleteMenuOpen {
		switch msg.String() {
		case "p", "P":
			m.deleteMenuOpen = false
			var targetIndices []int
			if len(m.selected) > 0 {
				for idx := range m.selected {
					targetIndices = append(targetIndices, idx)
				}
			} else if len(t.visibleFiles()) > 0 {
				targetIndices = append(targetIndices, t.cursor)
			}

			deletedCount := 0
			for _, idx := range targetIndices {
				if idx >= 0 && idx < len(t.visibleFiles()) {
					absPath := findAbsPath(t.currentDir, t.visibleFiles()[idx])
					if err := permanentDelete(absPath); err != nil {
						m.err = err
						break
					}
					deletedCount++
				}
			}
			m.selected = make(map[int]struct{})
			if m.err == nil && deletedCount > 0 {
				m.statusMsg = fmt.Sprintf("Permanently deleted %d item(s)", deletedCount)
				m.reloadCurrentDir()
			}

		case "r", "R", "n", "N":
			m.deleteMenuOpen = false
			var targetIndices []int
			if len(m.selected) > 0 {
				for idx := range m.selected {
					targetIndices = append(targetIndices, idx)
				}
			} else if len(t.visibleFiles()) > 0 {
				targetIndices = append(targetIndices, t.cursor)
			}

			deletedCount := 0
			for _, idx := range targetIndices {
				if idx >= 0 && idx < len(t.visibleFiles()) {
					absPath := findAbsPath(t.currentDir, t.visibleFiles()[idx])
					isDirItem := isDir(t.currentDir, t.visibleFiles()[idx])
					trashPath, err := normalDelete(absPath)
					if err != nil {
						m.err = err
						break
					}
					m.pushAction(ActionRecord{
						Type:         ActionDelete,
						OriginalPath: absPath,
						TrashPath:    trashPath,
						IsDir:        isDirItem,
					})
					deletedCount++
				}
			}
			m.selected = make(map[int]struct{})
			if m.err == nil && deletedCount > 0 {
				m.statusMsg = fmt.Sprintf("Moved %d item(s) to trash", deletedCount)
				m.reloadCurrentDir()
			}
		default:
			// Any other key closes the menu (including esc)
			m.deleteMenuOpen = false
		}
		return m, nil
	}

	if m.permissionMenuOpen {
		switch msg.String() {
		case "x", "X":
			m.permissionMenuOpen = false
			if len(t.visibleFiles()) > 0 {
				absPath := findAbsPath(t.currentDir, t.visibleFiles()[t.cursor])
				oldMode, newMode, err := toggleExecutable(absPath)
				if err != nil {
					m.err = err
				} else {
					m.pushAction(ActionRecord{
						Type:         ActionChmod,
						OriginalPath: absPath,
						OldMode:      oldMode,
						NewMode:      newMode,
					})
					m.reloadCurrentDir()
				}
			}
		default:
			// Any other key closes the menu (including esc)
			m.permissionMenuOpen = false
		}
		return m, nil
	}

	if m.openMenuOpen {
		switch msg.String() {
		case "c":
			m.openMenuOpen = false
			if err := openWithVSCode(t.currentDir); err != nil {
				m.err = err
			}
		case "C":
			m.openMenuOpen = false
			if len(t.visibleFiles()) > 0 {
				targetPath := findAbsPath(t.currentDir, t.visibleFiles()[t.cursor])
				if err := openWithVSCode(targetPath); err != nil {
					m.err = err
				}
			}
		case "r", "R":
			m.openMenuOpen = false
			if len(t.visibleFiles()) > 0 {
				targetPath := findAbsPath(t.currentDir, t.visibleFiles()[t.cursor])
				if err := revealInFileExplorer(targetPath); err != nil {
					m.err = err
				}
			}
		case "e", "E":
			m.openMenuOpen = false
			if len(t.visibleFiles()) > 0 {
				name := t.visibleFiles()[t.cursor]
				if isDir(t.currentDir, name) {
					break
				}
				editor := os.Getenv("EDITOR")
				if editor == "" {
					editor = "vi"
				}
				filePath := findAbsPath(t.currentDir, name)
				c := exec.Command(editor, filePath)
				return m, tea.ExecProcess(c, func(err error) tea.Msg {
					return editorFinishedMsg{err: err}
				})
			}
		default:
			// Any other key closes the menu (including esc)
			m.openMenuOpen = false
		}
		return m, nil
	}

	if m.helpMenuOpen {
		m.helpMenuOpen = false
		return m, nil
	}

	if m.metadataInspectorOpen {
		m.metadataInspectorOpen = false
		return m, nil
	}

	return m, nil
}

func (m model) handleShortcutKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	t := m.curTab()
	if m.shortcutPaneOpen && m.activePane == 0 {
		switch msg.String() {
		// ponytail: Allow global exit and exit-with-cd actions while shortcut pane is open
				case "1":
			if m.shortcutPaneOpen {
				m.activePane = 0
			}
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
		case "esc", "left", "h", "ctrl+b":
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

func (m model) handleExplorerKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	t := m.curTab()
	if m.copyJobActive {
		if msg.String() == "esc" {
			if m.copyJobCancel != nil {
				close(m.copyJobCancel)
			}
			m.copyJobActive = false
			m.statusMsg = "Copy cancelled"
		}
		return m, nil
	}

	switch msg.String() {
	case "1":
		if m.shortcutPaneOpen {
			m.activePane = 0
			return m, nil
		}
	case "2":
		m.activePane = 1
		return m, nil
	case "3":
		m.activePane = 2
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "q", "Q":
		_ = clearTempDirFile()
		m.quitting = true
		return m, tea.Quit
	case "esc":
		m.selected = make(map[int]struct{})
		m.stagedPaths = nil
		m.stagedOp = ""
		m.statusMsg = "Cleared all selections"
		m.err = nil

	// Toggle Shortcut Pane
	case "ctrl+b":
		m.shortcutPaneOpen = !m.shortcutPaneOpen
		if !m.shortcutPaneOpen && m.activePane == 0 {
			m.activePane = 1
		}

	case "E", "e":
		m.quitting = true
		_ = writeTempDirFile(t.currentDir)
		return m, tea.Quit
	// switch active pane
	case "tab":
		if m.shortcutPaneOpen {
			m.activePane = (m.activePane + 1) % 3
		} else {
			if m.activePane == 2 {
				m.activePane = 1
			} else {
				m.activePane = 2
			}
		}

	// switch tab previous
	case "[":
		if len(m.tabs) > 1 {
			m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
			m.err = nil
		}

	// switch tab next
	case "]":
		if len(m.tabs) > 1 {
			m.activeTab = (m.activeTab + 1) % len(m.tabs)
			m.err = nil
		}

	// new tab
	case "t", "T":
		entries, err := os.ReadDir(t.currentDir)
		files := []string{}
		if err == nil {
			for _, entry := range entries {
				if !m.showHidden && strings.HasPrefix(entry.Name(), ".") {
					continue
				}
				files = append(files, entry.Name())
			}
		}
		newTab := TabState{
			currentDir: t.currentDir,
			filesPath:  files,
		}
		m.tabs = append(m.tabs, newTab)
		m.activeTab = len(m.tabs) - 1
		m.err = nil

	// close tab
	case "w", "W":
		if len(m.tabs) > 1 {
			m.tabs = append(m.tabs[:m.activeTab], m.tabs[m.activeTab+1:]...)
			if m.activeTab >= len(m.tabs) {
				m.activeTab = len(m.tabs) - 1
			}
			m.err = nil
		}

	case "up", "k":
		if len(t.visibleFiles()) == 0 {
			break
		}
		m.err = nil
		if m.activePane == 1 {
			t.previewScrollOffset = 0
			if t.cursor > 0 {
				t.cursor--
			} else {
				t.cursor = len(t.visibleFiles()) - 1
			}
			m.adjustScroll()
		} else if m.activePane == 2 {
			if t.previewScrollOffset > 0 {
				t.previewScrollOffset--
			}
		}
	case "down", "j":
		if len(t.visibleFiles()) == 0 {
			break
		}
		m.err = nil
		if m.activePane == 1 {
			t.previewScrollOffset = 0
			if t.cursor < len(t.visibleFiles())-1 {
				t.cursor++
			} else {
				t.cursor = 0
			}
			m.adjustScroll()
		} else if m.activePane == 2 {
			t.previewScrollOffset++
		}

	case "space":
		if len(t.visibleFiles()) == 0 {
			break
		}
		_, ok := m.selected[t.cursor]
		if ok {
			delete(m.selected, t.cursor)
		} else {
			m.selected[t.cursor] = struct{}{}
		}

	case "enter":
		if len(t.visibleFiles()) == 0 {
			break
		}
		if isDir(t.currentDir, t.visibleFiles()[t.cursor]) {
			selectedName := t.visibleFiles()[t.cursor]
			nextDir := findAbsPath(t.currentDir, selectedName)
			entries, err := os.ReadDir(nextDir)
			if err != nil {
				m.err = err
				break
			}
			m.reLoadDir(nextDir, entries)
			m.adjustScroll()
		}

	// Rename
	case "r", "R":
		if len(t.visibleFiles()) == 0 {
			break
		}
		m.err = nil
		m.inputMode = true
		m.isRenaming = true
		m.inputText = t.visibleFiles()[t.cursor]
		// ponytail: cursor before ".ext"; dirs and dotfiles go to end
		name := t.visibleFiles()[t.cursor]
		m.inputCursor = len([]rune(name)) // default: end
		if !isDir(t.currentDir, name) {
			if dot := strings.LastIndex(name, "."); dot > 0 {
				m.inputCursor = len([]rune(name[:dot]))
			}
		}

	// Undo
	case "u":
		m.err = m.performUndo()

	// Redo
	case "ctrl+r", "U":
		m.err = m.performRedo()

	// Create file
	case "f", "F":
		m.err = nil
		m.inputMode = true
		m.inputText = ""
		m.isCreatingDir = false
		m.isRenaming = false

	// Toggle Select All ('a' / 'A')
	case "a", "A":
		if len(t.visibleFiles()) == 0 {
			break
		}
		m.err = nil
		if len(m.selected) == len(t.visibleFiles()) {
			m.selected = make(map[int]struct{})
			m.statusMsg = "Deselected all item(s)"
		} else {
			m.selected = make(map[int]struct{})
			for i := range t.visibleFiles() {
				m.selected[i] = struct{}{}
			}
			m.statusMsg = fmt.Sprintf("Selected all %d item(s)", len(t.visibleFiles()))
		}
	case "g", "G":
		if len(t.visibleFiles()) == 0 {
			break
		}
		m.err = nil
		name := t.visibleFiles()[t.cursor]
		target, isSym, isBroken := getSymlinkTarget(t.currentDir, name)
		if !isSym {
			m.statusMsg = "Not a symbolic link"
			break
		}
		if isBroken {
			m.err = fmt.Errorf("Symlink target is broken: %s", target)
			break
		}

		resolvedTarget := target
		if !filepath.IsAbs(resolvedTarget) {
			resolvedTarget = filepath.Join(t.currentDir, target)
		}

		info, err := os.Stat(resolvedTarget)
		if err != nil {
			m.err = err
			break
		}

		if info.IsDir() {
			entries, err := os.ReadDir(resolvedTarget)
			if err != nil {
				m.err = err
				break
			}
			m.reLoadDir(resolvedTarget, entries)
			m.adjustScroll()
			m.statusMsg = fmt.Sprintf("Jumped to symlink target: %s", resolvedTarget)
		} else {
			targetDir := filepath.Dir(resolvedTarget)
			targetBase := filepath.Base(resolvedTarget)
			entries, err := os.ReadDir(targetDir)
			if err != nil {
				m.err = err
				break
			}
			m.reLoadDir(targetDir, entries)
			for idx, fname := range t.visibleFiles() {
				if fname == targetBase {
					t.cursor = idx
					break
				}
			}
			m.adjustScroll()
			m.statusMsg = fmt.Sprintf("Jumped to symlink target file: %s", resolvedTarget)
		}

	// Create Symlink ('K')
	case "K":
		if len(t.visibleFiles()) == 0 {
			break
		}
		m.err = nil
		selectedName := t.visibleFiles()[t.cursor]
		targetPath := findAbsPath(t.currentDir, selectedName)
		linkName := fmt.Sprintf("%s_link", selectedName)
		linkPath := filepath.Join(t.currentDir, linkName)

		if err := os.Symlink(targetPath, linkPath); err != nil {
			m.err = err
		} else {
			m.pushAction(ActionRecord{Type: ActionCreate, OriginalPath: linkPath})
			m.reloadCurrentDir()
			m.statusMsg = fmt.Sprintf("Created symlink: %s -> %s", linkName, selectedName)
		}

	// Move selected / Create Dir
	case "m", "M":
		m.err = nil
		if len(m.selected) > 0 {
			m.stagedPaths = nil
			for idx := range m.selected {
				if idx >= 0 && idx < len(t.visibleFiles()) {
					absP := findAbsPath(t.currentDir, t.visibleFiles()[idx])
					m.stagedPaths = append(m.stagedPaths, absP)
				}
			}
			m.stagedOp = "move"
			m.selected = make(map[int]struct{})
			m.statusMsg = fmt.Sprintf("Staged %d item(s) to move. Press 'v' or 'p' to paste.", len(m.stagedPaths))
		} else {
			m.inputMode = true
			m.inputText = ""
			m.isCreatingDir = true
			m.isRenaming = false
		}

	// Copy selected / copy file path
	case "c", "C":
		m.err = nil
		if len(m.selected) > 0 {
			m.stagedPaths = nil
			for idx := range m.selected {
				if idx >= 0 && idx < len(t.visibleFiles()) {
					absP := findAbsPath(t.currentDir, t.visibleFiles()[idx])
					m.stagedPaths = append(m.stagedPaths, absP)
				}
			}
			m.stagedOp = "copy"
			m.selected = make(map[int]struct{})
			m.statusMsg = fmt.Sprintf("Copied %d item(s). Press 'v' or 'p' to paste.", len(m.stagedPaths))
		} else if len(t.visibleFiles()) > 0 {
			absPath := findAbsPath(t.currentDir, t.visibleFiles()[t.cursor])
			if err := copyToClipboard(absPath); err != nil {
				m.err = err
			} else {
				m.statusMsg = "Copied path to clipboard"
			}
		}

	// Paste staged files
	case "v", "V":
		if len(m.stagedPaths) == 0 {
			break
		}
		m.err = nil
		pastedCount := 0
		for _, srcPath := range m.stagedPaths {
			baseName := filepath.Base(srcPath)
			dstPath := filepath.Join(t.currentDir, baseName)
			if m.stagedOp == "copy" {
				if err := copyFileOrDir(srcPath, dstPath); err != nil {
					m.err = err
					break
				}
				m.pushAction(ActionRecord{Type: ActionCreate, OriginalPath: dstPath})
				pastedCount++
			} else if m.stagedOp == "move" {
				if err := moveFileOrDir(srcPath, dstPath); err != nil {
					m.err = err
					break
				}
				m.pushAction(ActionRecord{Type: ActionRename, OldPath: srcPath, NewPath: dstPath})
				pastedCount++
			}
		}
		if m.err == nil {
			m.statusMsg = fmt.Sprintf("Successfully performed %s for %d item(s)", m.stagedOp, pastedCount)
			m.stagedPaths = nil
			m.stagedOp = ""
			m.reloadCurrentDir()
		}

	// copy menu
	case "y", "Y":
		if len(t.visibleFiles()) == 0 {
			break
		}
		m.err = nil
		m.copyMenuOpen = true

	// bookmark menu
	case "b", "B":
		m.err = nil
		m.bookmarkMenuOpen = true

	// delete menu
	case "d", "D":
		if len(t.visibleFiles()) == 0 {
			break
		}
		m.err = nil
		m.deleteMenuOpen = true

	// permission menu / paste
	case "p", "P":
		if len(m.stagedPaths) > 0 {
			m.err = nil
			pastedCount := 0
			for _, srcPath := range m.stagedPaths {
				baseName := filepath.Base(srcPath)
				dstPath := filepath.Join(t.currentDir, baseName)
				if m.stagedOp == "copy" {
					if err := copyFileOrDir(srcPath, dstPath); err != nil {
						m.err = err
						break
					}
					m.pushAction(ActionRecord{Type: ActionCreate, OriginalPath: dstPath})
					pastedCount++
				} else if m.stagedOp == "move" {
					if err := moveFileOrDir(srcPath, dstPath); err != nil {
						m.err = err
						break
					}
					m.pushAction(ActionRecord{Type: ActionRename, OldPath: srcPath, NewPath: dstPath})
					pastedCount++
				}
			}
			if m.err == nil {
				m.statusMsg = fmt.Sprintf("Successfully performed %s for %d item(s)", m.stagedOp, pastedCount)
				m.stagedPaths = nil
				m.stagedOp = ""
				m.reloadCurrentDir()
			}
		} else if len(t.visibleFiles()) > 0 {
			m.err = nil
			m.permissionMenuOpen = true
		}

	// Metadata Inspector dialog ('i' / 'I')
	case "i", "I":
		if len(t.visibleFiles()) == 0 {
			break
		}
		m.err = nil
		m.metadataInspectorOpen = true

	// help menu page
	case "`":
		m.err = nil
		m.helpMenuOpen = true

	// Temporary switch to shell
	case ":", "!", "S":
		shell := os.Getenv("SHELL")
		if shell == "" {
			if runtime.GOOS == "windows" {
				shell = "cmd.exe"
			} else {
				shell = "/bin/sh"
			}
		}
		c := exec.Command(shell)
		c.Dir = t.currentDir
		return m, tea.ExecProcess(c, func(err error) tea.Msg {
			return subshellFinishedMsg{err: err}
		})

	// open menu
	case "o", "O":
		if len(t.visibleFiles()) == 0 {
			break
		}
		m.err = nil
		m.openMenuOpen = true

	case "right", "l", "L":
		if len(t.visibleFiles()) == 0 {
			break
		}
		m.err = nil
		if !isDir(t.currentDir, t.visibleFiles()[t.cursor]) {
			break
		}

		selectedName := t.visibleFiles()[t.cursor]
		nextDir := findAbsPath(t.currentDir, selectedName)

		entries, err := os.ReadDir(nextDir)
		if err != nil {
			m.err = err
			break
		}

		m.reLoadDir(nextDir, entries)
		m.adjustScroll()

	case "left", "h", "H":
		m.err = nil
		parent := filepath.Dir(t.currentDir)
		entries, err := os.ReadDir(parent)
		if err != nil {
			m.err = err
			break
		}

		m.reLoadDir(parent, entries)
		m.adjustScroll()

	case ".":
		m.showHidden = !m.showHidden
		m.reloadCurrentDir()
		m.adjustScroll()

	case "s":
		if m.activePane == 1 {
			m.searchActive = true
			m.searchText = ""
			m.searchCursor = 0
			return m, nil
		}
	}

	return m, nil
}
