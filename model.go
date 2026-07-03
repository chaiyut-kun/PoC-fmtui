package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
)

type ActionType string

const (
	ActionDelete ActionType = "delete"
	ActionCreate ActionType = "create"
	ActionRename ActionType = "rename"
	ActionChmod  ActionType = "chmod"
)

type ActionRecord struct {
	Type         ActionType
	OriginalPath string
	TrashPath    string
	OldPath      string
	NewPath      string
	OldMode      os.FileMode
	NewMode      os.FileMode
	IsDir        bool
}

type model struct {
	filesPath           []string
	cursor              int
	currentDir          string
	selected            map[int]struct{}
	showHidden          bool
	err                 error
	scrollOffset        int
	terminalHeight      int
	terminalWidth       int
	inputMode           bool
	inputText           string
	isCreatingDir       bool
	isRenaming          bool
	copyMenuOpen        bool
	deleteMenuOpen      bool
	permissionMenuOpen  bool
	helpMenuOpen        bool
	activePane          int // 0 = Left (FM), 1 = Right (Preview)
	previewScrollOffset int // vertical scroll offset for preview
	quitting            bool
	undoStack           []ActionRecord
	redoStack           []ActionRecord
}

type editorFinishedMsg struct {
	err error
}

func InitialModel() model {

	// read and list file from [./files]
	currentDir, _ := os.Getwd()
	entries, err := os.ReadDir(currentDir)
	if err != nil {

		return model{
			filesPath:  []string{},
			currentDir: currentDir,
			selected:   make(map[int]struct{}),
		}
	}

	files := []string{}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		files = append(files, entry.Name())
	}

	return model{
		filesPath:  files,
		currentDir: currentDir,
		selected:   make(map[int]struct{}),
	}

}

func (m model) Init() tea.Cmd {
	return nil
}

func (m *model) adjustScroll() {
	if len(m.filesPath) == 0 {
		m.scrollOffset = 0
		return
	}

	overhead := 3
	if m.err != nil || m.inputMode {
		overhead = 4
	}
	maxVisible := m.terminalHeight - overhead
	if maxVisible < 1 {
		maxVisible = 1
	}

	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}

	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}

	if m.cursor >= m.scrollOffset+maxVisible {
		m.scrollOffset = m.cursor - maxVisible + 1
	}

	if m.scrollOffset > len(m.filesPath)-maxVisible {
		m.scrollOffset = len(m.filesPath) - maxVisible
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case editorFinishedMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.reloadCurrentDir()
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.terminalHeight = msg.Height
		m.terminalWidth = msg.Width
		m.adjustScroll()
		return m, nil

	case tea.MouseMsg:
		mEvt := msg.Mouse()
		switch mEvt.Button {
		case tea.MouseWheelUp:
			if m.activePane == 0 {
				if m.cursor > 0 {
					m.cursor--
					m.adjustScroll()
				}
			} else {
				if m.previewScrollOffset > 0 {
					m.previewScrollOffset--
				}
			}
			return m, nil

		case tea.MouseWheelDown:
			if m.activePane == 0 {
				if m.cursor < len(m.filesPath)-1 {
					m.cursor++
					m.adjustScroll()
				}
			} else {
				m.previewScrollOffset++
			}
			return m, nil

		case tea.MouseLeft:
			width := m.terminalWidth
			if width < 20 {
				width = 80
			}
			if mEvt.X < width/2 {
				m.activePane = 0
			} else {
				m.activePane = 1
			}
			return m, nil
		}

	case tea.KeyPressMsg:
		if m.copyMenuOpen {
			switch msg.String() {
			case "p", "P":
				m.copyMenuOpen = false
				if len(m.filesPath) > 0 {
					absPath := findAbsPath(m.currentDir, m.filesPath[m.cursor])
					if err := copyToClipboard(absPath); err != nil {
						m.err = err
					}
				}
			case "n", "N":
				m.copyMenuOpen = false
				if len(m.filesPath) > 0 {
					name := m.filesPath[m.cursor]
					if !isDir(m.currentDir, name) {
						name = strings.TrimSuffix(name, filepath.Ext(name))
					}
					if err := copyToClipboard(name); err != nil {
						m.err = err
					}
				}
			case "e", "E":
				m.copyMenuOpen = false
				if len(m.filesPath) > 0 {
					name := m.filesPath[m.cursor]
					if err := copyToClipboard(name); err != nil {
						m.err = err
					}
				}
			default:
				// Any other key closes the menu (including esc)
				m.copyMenuOpen = false
			}
			return m, nil
		}

		if m.deleteMenuOpen {
			switch msg.String() {
			case "p", "P":
				m.deleteMenuOpen = false
				if len(m.filesPath) > 0 {
					absPath := findAbsPath(m.currentDir, m.filesPath[m.cursor])
					if err := permanentDelete(absPath); err != nil {
						m.err = err
					} else {
						m.reloadCurrentDir()
					}
				}
			case "r", "R", "n", "N":
				m.deleteMenuOpen = false
				if len(m.filesPath) > 0 {
					absPath := findAbsPath(m.currentDir, m.filesPath[m.cursor])
					isDirItem := isDir(m.currentDir, m.filesPath[m.cursor])
					trashPath, err := normalDelete(absPath)
					if err != nil {
						m.err = err
					} else {
						m.pushAction(ActionRecord{
							Type:         ActionDelete,
							OriginalPath: absPath,
							TrashPath:    trashPath,
							IsDir:        isDirItem,
						})
						m.reloadCurrentDir()
					}
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
				if len(m.filesPath) > 0 {
					absPath := findAbsPath(m.currentDir, m.filesPath[m.cursor])
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

		if m.helpMenuOpen {
			m.helpMenuOpen = false
			return m, nil
		}

		if m.inputMode {
			switch msg.String() {
			case "enter":
				m.inputMode = false
				if m.inputText == "" {
					m.isRenaming = false
					break
				}

				if m.isRenaming {
					if len(m.filesPath) > 0 {
						oldName := m.filesPath[m.cursor]
						oldPath := findAbsPath(m.currentDir, oldName)
						newPath := filepath.Join(m.currentDir, m.inputText)
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

				filePath := filepath.Join(m.currentDir, m.inputText)

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
				m.isRenaming = false
			case "backspace":
				if len(m.inputText) > 0 {
					m.inputText = m.inputText[:len(m.inputText)-1]
				}
			default:
				runes := []rune(msg.String())
				if len(runes) == 1 {
					m.inputText += string(runes)
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q", "Q":
			_ = clearTempDirFile()
			m.quitting = true
			return m, tea.Quit
		case "E", "e":
			m.quitting = true
			_ = writeTempDirFile(m.currentDir)
			return m, tea.Quit
		case "[", "]":
			if m.activePane == 0 {
				m.activePane = 1
			} else {
				m.activePane = 0
			}

		case "up", "k":
			if len(m.filesPath) == 0 {
				break
			}
			m.err = nil
			if m.activePane == 0 {
				m.previewScrollOffset = 0
				if m.cursor > 0 {
					m.cursor--
				} else {
					m.cursor = len(m.filesPath) - 1
				}
				m.adjustScroll()
			} else {
				if m.previewScrollOffset > 0 {
					m.previewScrollOffset--
				}
			}
		case "down", "j":
			if len(m.filesPath) == 0 {
				break
			}
			m.err = nil
			if m.activePane == 0 {
				m.previewScrollOffset = 0
				if m.cursor < len(m.filesPath)-1 {
					m.cursor++
				} else {
					m.cursor = 0
				}
				m.adjustScroll()
			} else {
				m.previewScrollOffset++
			}

		case "enter", "space":
			if len(m.filesPath) == 0 {
				break
			}
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}

		// Rename
		case "r", "R":
			if len(m.filesPath) == 0 {
				break
			}
			m.err = nil
			m.inputMode = true
			m.isRenaming = true
			m.inputText = m.filesPath[m.cursor]

		// Undo
		case "u":
			m.err = m.performUndo()

		// Redo
		case "ctrl+r", "U":
			m.err = m.performRedo()

		// Create file
		case "a", "A":
			m.err = nil
			m.inputMode = true
			m.inputText = ""
			m.isCreatingDir = false
			m.isRenaming = false

		// Create Dir
		case "m", "M":
			m.err = nil
			m.inputMode = true
			m.inputText = ""
			m.isCreatingDir = true
			m.isRenaming = false

		// copy file path
		case "c", "C":
			if len(m.filesPath) == 0 {
				break
			}
			m.err = nil
			absPath := findAbsPath(m.currentDir, m.filesPath[m.cursor])
			// copy to clipboard
			if err := copyToClipboard(absPath); err != nil {
				m.err = err
			}

		// copy menu
		case "y", "Y":
			if len(m.filesPath) == 0 {
				break
			}
			m.err = nil
			m.copyMenuOpen = true

		// delete menu
		case "d", "D":
			if len(m.filesPath) == 0 {
				break
			}
			m.err = nil
			m.deleteMenuOpen = true

		// permission menu
		case "p", "P":
			if len(m.filesPath) == 0 {
				break
			}
			m.err = nil
			m.permissionMenuOpen = true

		// help menu page
		case "`":
			m.err = nil
			m.helpMenuOpen = true

		// open editor
		case "o", "O":
			if len(m.filesPath) == 0 {
				break
			}
			m.err = nil
			name := m.filesPath[m.cursor]
			if isDir(m.currentDir, name) {
				break
			}

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vi"
			}

			filePath := findAbsPath(m.currentDir, name)
			c := exec.Command(editor, filePath)
			return m, tea.ExecProcess(c, func(err error) tea.Msg {
				return editorFinishedMsg{err: err}
			})

		case "right", "l", "L":
			if len(m.filesPath) == 0 {
				break
			}
			m.err = nil
			if !isDir(m.currentDir, m.filesPath[m.cursor]) {
				break
			}

			selectedName := m.filesPath[m.cursor]
			nextDir := findAbsPath(m.currentDir, selectedName)

			entries, err := os.ReadDir(nextDir)
			if err != nil {
				m.err = err
				break
			}

			m.reLoadDir(nextDir, entries)
			m.adjustScroll()

		case "left", "h", "H":
			m.err = nil
			parent := filepath.Dir(m.currentDir)
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
		}
	}

	return m, nil
}

func (m model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

	renderedText := ""

	overhead := 5
	maxVisible := m.terminalHeight - overhead
	if maxVisible < 1 {
		maxVisible = 1
	}

	endIdx := m.scrollOffset + maxVisible
	if endIdx > len(m.filesPath) {
		endIdx = len(m.filesPath)
	}

	for i := m.scrollOffset; i < endIdx; i++ {
		list := m.filesPath[i]

		icon, fgColor, bgColor := m.CheckDirFilesExt(list)

		cursorSymbol := " "
		if m.cursor == i {
			cursorSymbol = ">"
			renderedText += fmt.Sprintf("%s%s %s %s %s%s%s\n", bgColor, Black, cursorSymbol, icon, list, ItemEndSpace, Reset)
		} else {
			if fgColor == Reset {
				renderedText += fmt.Sprintf(" %s %s %s%s\n", cursorSymbol, icon, list, ItemEndSpace)
			} else {
				renderedText += fmt.Sprintf(" %s %s%s %s%s%s \n", cursorSymbol, fgColor, icon, list, ItemEndSpace, Reset)
			}
		}
	}

	if m.inputMode {
		prompt := "Create file"
		if m.isRenaming {
			prompt = "Rename"
		} else if m.isCreatingDir {
			prompt = "Create directory"
		}
		renderedText += fmt.Sprintf("\n%s: %s█\n(Enter to confirm, Esc to cancel)", prompt, m.inputText)
	}

	// Trim trailing newline so lipgloss box height is calculated precisely
	renderedText = strings.TrimSuffix(renderedText, "\n")

	width := m.terminalWidth
	height := m.terminalHeight
	if width < 20 {
		width = 80
	}
	if height < 5 {
		height = 20
	}

	bottomBarLines := 1
	if m.err != nil {
		bottomBarLines = 2
	}
	boxHeight := height - bottomBarLines
	if boxHeight < 3 {
		boxHeight = 3
	}

	leftWidth := width / 2
	rightWidth := width - leftWidth

	leftBorderColor := "#555555"
	rightBorderColor := "#555555"
	if m.activePane == 0 {
		leftBorderColor = "#5555FF"
	} else {
		rightBorderColor = "#5555FF"
	}

	// 1. Create left box (File Explorer)
	leftStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(leftBorderColor)).
		Width(leftWidth - 2).
		Height(boxHeight - 2)

	leftBox := leftStyle.Render(renderedText)

	// Overlay " FMTUI: <currentDir> " on left border
	leftLines := strings.Split(leftBox, "\n")
	if len(leftLines) > 0 {
		topLine := []rune(leftLines[0])
		rawTitle := fmt.Sprintf(" FMTUI: %s ", m.currentDir)
		title := []rune(rawTitle)
		maxTitleLen := len(topLine) - 4
		if maxTitleLen > 6 && len(title) > maxTitleLen {
			pathTrimmed := " FMTUI: ..." + string(title[len(title)-(maxTitleLen-8):])
			title = []rune(pathTrimmed)
		}
		startIdx := (len(topLine) - len(title)) / 2
		if startIdx > 0 && startIdx+len(title) < len(topLine) {
			copy(topLine[startIdx:], title)
			leftLines[0] = string(topLine)
		}
		leftBox = strings.Join(leftLines, "\n")
	}

	// 2. Create right box (Previewer)
	var rightText string
	maxVisiblePreviewLines := boxHeight - 2
	if maxVisiblePreviewLines < 1 {
		maxVisiblePreviewLines = 1
	}

	if len(m.filesPath) > 0 {
		selectedName := m.filesPath[m.cursor]
		targetPath := findAbsPath(m.currentDir, selectedName)
		previewLines := getPreviewLines(targetPath, isDir(m.currentDir, selectedName), m.showHidden)

		// Clamp previewScrollOffset based on number of lines and visible height
		maxScroll := len(previewLines) - maxVisiblePreviewLines
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.previewScrollOffset > maxScroll {
			m.previewScrollOffset = maxScroll
		}
		if m.previewScrollOffset < 0 {
			m.previewScrollOffset = 0
		}

		endLine := m.previewScrollOffset + maxVisiblePreviewLines
		if endLine > len(previewLines) {
			endLine = len(previewLines)
		}

		maxWidth := rightWidth - 2
		if maxWidth < 1 {
			maxWidth = 1
		}
		var truncatedLines []string
		for _, line := range previewLines[m.previewScrollOffset:endLine] {
			expanded := strings.ReplaceAll(line, "\t", "    ")
			runes := []rune(expanded)
			if len(runes) > maxWidth {
				truncatedLines = append(truncatedLines, string(runes[:maxWidth]))
			} else {
				truncatedLines = append(truncatedLines, expanded)
			}
		}
		rightText = strings.Join(truncatedLines, "\n")
	}
	rightText = strings.TrimSuffix(rightText, "\n")

	rightStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(rightBorderColor)).
		Width(rightWidth - 2).
		Height(boxHeight - 2)

	rightBox := rightStyle.Render(rightText)

	// Overlay " Preview " on right border
	rightLines := strings.Split(rightBox, "\n")
	if len(rightLines) > 0 {
		topLine := []rune(rightLines[0])
		title := []rune(" Preview ")
		startIdx := (len(topLine) - len(title)) / 2
		if startIdx > 0 && startIdx+len(title) < len(topLine) {
			copy(topLine[startIdx:], title)
			rightLines[0] = string(topLine)
		}
		rightBox = strings.Join(rightLines, "\n")
	}

	// Join both boxes horizontally
	content := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)

	// Bottom Help & Error Status Bar
	hiddenStatus := "OFF"
	if m.showHidden {
		hiddenStatus = "ON"
	}
	helpText := fmt.Sprintf(" Press 'q' to quit | 'e' exit&cd | '.' hidden:%s | '[' ']' switch pane | 'y' copy | 'd' delete | 'r' rename | 'p' chmod | '`' help", hiddenStatus)

	if m.err != nil {
		content += fmt.Sprintf("\n%sError: %v%s", Red, m.err, Reset)
	}
	content += "\n" + helpText

	// Render copy menu if open
	if m.copyMenuOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#55FF55")). // Green border
			Padding(1, 2)

		dialogContent := "    Copy Menu\n\n" +
			"[p] Copy absolute path\n" +
			"[n] Copy name only\n" +
			"[e] Copy name with ext\n\n" +
			"Press ESC to cancel"

		dialog := dialogStyle.Render(dialogContent)

		dialogHeight := lipgloss.Height(dialog)
		dialogWidth := lipgloss.Width(dialog)
		startRow := (height - dialogHeight) / 2
		startCol := (width - dialogWidth) / 2

		bgLines := strings.Split(content, "\n")
		fgLines := strings.Split(dialog, "\n")
		for r, fgLine := range fgLines {
			row := startRow + r
			if row >= 0 && row < len(bgLines) {
				bgLines[row] = overlayLine(bgLines[row], fgLine, startCol)
			}
		}
		content = strings.Join(bgLines, "\n")
	}

	// Render delete menu if open
	if m.deleteMenuOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#FF5555")). // Red border
			Padding(1, 2)

		dialogContent := "Delete Menu\n\n" +
			"[p] Permanent delete\n" +
			"[n] Normal delete (Trash)\n\n" +
			"Press ESC to cancel"

		dialog := dialogStyle.Render(dialogContent)

		dialogHeight := lipgloss.Height(dialog)
		dialogWidth := lipgloss.Width(dialog)
		startRow := (height - dialogHeight) / 2
		startCol := (width - dialogWidth) / 2

		bgLines := strings.Split(content, "\n")
		fgLines := strings.Split(dialog, "\n")
		for r, fgLine := range fgLines {
			row := startRow + r
			if row >= 0 && row < len(bgLines) {
				bgLines[row] = overlayLine(bgLines[row], fgLine, startCol)
			}
		}
		content = strings.Join(bgLines, "\n")
	}

	// Render permission menu if open
	if m.permissionMenuOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#55FFFF")). // Cyan border
			Padding(1, 2)

		dialogContent := "  Permission Menu\n\n" +
			"[x] Toggle executable (chmod +x/-x)\n\n" +
			"Press ESC to cancel"

		dialog := dialogStyle.Render(dialogContent)

		dialogHeight := lipgloss.Height(dialog)
		dialogWidth := lipgloss.Width(dialog)
		startRow := (height - dialogHeight) / 2
		startCol := (width - dialogWidth) / 2

		bgLines := strings.Split(content, "\n")
		fgLines := strings.Split(dialog, "\n")
		for r, fgLine := range fgLines {
			row := startRow + r
			if row >= 0 && row < len(bgLines) {
				bgLines[row] = overlayLine(bgLines[row], fgLine, startCol)
			}
		}
		content = strings.Join(bgLines, "\n")
	}

	// Render full help menu if open
	if m.helpMenuOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#FFFF55")). // Yellow border
			Padding(1, 4)

		dialogContent := "FMTUI Help Menu\n\n" +
			"Navigation & Panes:\n" +
			"  j / k, Up / Down   Navigate files / scroll preview\n" +
			"  h / l, Left / Right Go to parent dir / enter dir\n" +
			"  [ / ]              Switch active pane\n\n" +
			"File Actions:\n" +
			"  a / m              Create file / directory\n" +
			"  r                  Rename file or directory\n" +
			"  d                  Delete menu (p: permanent, n: trash)\n" +
			"  y                  Copy menu (p: path, n: name, e: ext)\n" +
			"  p                  Permission menu (x: chmod +x/-x)\n" +
			"  o                  Open file in $EDITOR\n" +
			"  .                  Toggle hidden files\n\n" +
			"Undo / Redo & Exit:\n" +
			"  u                  Undo action\n" +
			"  ctrl+r / U         Redo action\n" +
			"  e                  Exit & cd to current dir in shell\n" +
			"  q / ctrl+c         Quit application\n\n" +
			"          Press ` or ESC to close"

		dialog := dialogStyle.Render(dialogContent)

		dialogHeight := lipgloss.Height(dialog)
		dialogWidth := lipgloss.Width(dialog)
		startRow := (height - dialogHeight) / 2
		startCol := (width - dialogWidth) / 2

		bgLines := strings.Split(content, "\n")
		fgLines := strings.Split(dialog, "\n")
		for r, fgLine := range fgLines {
			row := startRow + r
			if row >= 0 && row < len(bgLines) {
				bgLines[row] = overlayLine(bgLines[row], fgLine, startCol)
			}
		}
		content = strings.Join(bgLines, "\n")
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
