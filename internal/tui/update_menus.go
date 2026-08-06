package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
)

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
						Icon: DirIcon,
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
				Icon: DirIcon,
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
		if m.helpSearchActive {
			switch msg.String() {
			case "esc":
				m.helpSearchActive = false
				m.searchInput.Blur()
				m.searchInput.SetValue("")
				m.searchInput.Placeholder = "search file / directory"
			case "enter":
				m.helpSearchActive = false
				m.searchInput.Blur()
			case "up":
				if m.helpScrollOffset > 0 {
					m.helpScrollOffset--
				}
			case "down":
				m.helpScrollOffset++
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				return m, cmd
			}
			return m, nil
		}

		switch msg.String() {
		case "s", "S":
			m.helpSearchActive = true
			m.searchInput.Placeholder = "Search help..."
			m.searchInput.SetValue("")
			m.helpScrollOffset = 0
			return m, m.searchInput.Focus()
		case "j", "down":
			m.helpScrollOffset++
		case "k", "up":
			if m.helpScrollOffset > 0 {
				m.helpScrollOffset--
			}
		case "esc", "`":
			m.helpMenuOpen = false
			m.searchInput.SetValue("")
			m.searchInput.Blur()
			m.searchInput.Placeholder = "search file / directory"
			m.helpScrollOffset = 0
		}
		return m, nil
	}

	if m.metadataInspectorOpen {
		m.metadataInspectorOpen = false
		return m, nil
	}

	return m, nil
}
