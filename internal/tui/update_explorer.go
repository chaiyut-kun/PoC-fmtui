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

func (m *model) pasteStagedFiles() {
	if len(m.stagedPaths) == 0 {
		return
	}
	t := m.curTab()
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

	case "ctrl+c", "q", "Q":
		_ = clearTempDirFile()
		m.quitting = true
		return m, tea.Quit
	case "ctrl+f":
		m.quickFindOpen = true
		m.quickFindInput.SetValue("")
		cmd := m.quickFindInput.Focus()
		m.quickFindSelected = 0
		m.quickFindScrollOffset = 0
		m.quickFindSelecting = false
		m.quickFindResults = nil
		return m, tea.Batch(cmd, m.quickFindSearchCmd(""))
	case "esc":
		m.selected = make(map[int]struct{})
		m.stagedPaths = nil
		m.stagedOp = ""
		m.statusMsg = "Cleared all selections"
		m.err = nil

	// Toggle Shortcut Pane
	case "ctrl+b", "\\":
		m.shortcutPaneOpen = !m.shortcutPaneOpen
		if m.shortcutPaneOpen {
			m.activePane = 0
		} else if m.activePane == 0 {
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
		name := t.visibleFiles()[t.cursor]
		m.fileInput.SetValue(name)
		cmd := m.fileInput.Focus()
		// ponytail: cursor before ".ext"; dirs and dotfiles go to end
		cur := len([]rune(name)) // default: end
		if !isDir(t.currentDir, name) {
			if dot := strings.LastIndex(name, "."); dot > 0 {
				cur = len([]rune(name[:dot]))
			}
		}
		m.fileInput.SetCursor(cur)
		return m, cmd

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
		m.fileInput.SetValue("")
		cmd := m.fileInput.Focus()
		m.isCreatingDir = false
		m.isRenaming = false
		return m, cmd

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
			m.err = fmt.Errorf("symlink target is broken: %s", target)
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
			m.fileInput.SetValue("")
			cmd := m.fileInput.Focus()
			m.isCreatingDir = true
			m.isRenaming = false
			return m, cmd
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
		m.pasteStagedFiles()

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
			m.pasteStagedFiles()
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
			m.searchInput.SetValue("")
			cmd := m.searchInput.Focus()
			return m, cmd
		}
	}

	return m, nil
}
