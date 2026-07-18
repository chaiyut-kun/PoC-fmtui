package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
)

type subshellFinishedMsg struct{ err error }

type copyProgressMsg struct {
	progress float64
}

type copyFinishedMsg struct {
	err error
}

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

type TabState struct {
	currentDir          string
	filesPath           []string
	cursor              int
	scrollOffset        int
	previewScrollOffset int
}

type model struct {
	tabs                      []TabState
	activeTab                 int
	selected                  map[int]struct{}
	showHidden                bool
	err                       error
	terminalHeight            int
	terminalWidth             int
	inputMode                 bool
	inputText                 string
	inputCursor               int // rune index into inputText; ponytail: single int, no textarea widget
	isCreatingDir             bool
	isRenaming                bool
	copyMenuOpen              bool
	deleteMenuOpen            bool
	permissionMenuOpen        bool
	openMenuOpen              bool
	helpMenuOpen              bool
	activePane                int // 0 = Left (FM), 1 = Right (Preview)
	stagedPaths               []string
	stagedOp                  string
	statusMsg                 string
	quitting                  bool
	undoStack                 []ActionRecord
	redoStack                 []ActionRecord
	metadataInspectorOpen     bool
	shortcutPaneOpen          bool
	shortcutCursor            int
	shortcuts                 []ShortcutEntry
	bookmarks                 []ShortcutEntry
	shortcutTab               int // 0 = Default, 1 = Bookmarks
	bookmarkMenuOpen          bool
	deleteBookmarkConfirmOpen bool
	copyJobActive             bool
	copyJobProgress           float64
	copyJobFileName           string
	copyProgressChan          chan tea.Msg
	copyJobCancel             chan struct{}
}

func (m *model) curTab() *TabState {
	if len(m.tabs) == 0 {
		dir, _ := os.Getwd()
		m.tabs = []TabState{{currentDir: dir}}
	}
	if m.activeTab < 0 {
		m.activeTab = 0
	}
	if m.activeTab >= len(m.tabs) {
		m.activeTab = len(m.tabs) - 1
	}
	return &m.tabs[m.activeTab]
}

func (m *model) startCopyContentJob(filePath string, fileName string) tea.Cmd {
	m.copyJobActive = true
	m.copyJobProgress = 0.0
	m.copyJobFileName = fileName
	m.copyProgressChan = make(chan tea.Msg, 100)
	m.copyJobCancel = make(chan struct{})

	go func(path string, ch chan tea.Msg, cancel chan struct{}) {
		file, err := os.Open(path)
		if err != nil {
			ch <- copyFinishedMsg{err: err}
			return
		}
		defer file.Close()

		info, err := file.Stat()
		if err != nil {
			ch <- copyFinishedMsg{err: err}
			return
		}
		totalSize := info.Size()

		// Limit file size to 50MB
		if totalSize > 50*1024*1024 {
			ch <- copyFinishedMsg{err: fmt.Errorf("file too large to copy content (max 50MB)")}
			return
		}

		// Read in chunks
		chunkSize := int64(64 * 1024) // 64KB chunks
		buffer := make([]byte, chunkSize)
		var totalRead int64
		var contentBuilder strings.Builder
		contentBuilder.Grow(int(totalSize))

		isBinaryChecked := false

		for {
			select {
			case <-cancel:
				ch <- copyFinishedMsg{err: fmt.Errorf("copy cancelled")}
				return
			default:
			}

			n, err := file.Read(buffer)
			if n > 0 {
				chunk := buffer[:n]
				// Binary detection check on first read
				if !isBinaryChecked {
					isBinaryChecked = true
					checkLen := n
					if checkLen > 1024 {
						checkLen = 1024
					}
					for i := 0; i < checkLen; i++ {
						if chunk[i] == 0x00 {
							ch <- copyFinishedMsg{err: fmt.Errorf("cannot copy content: file is binary")}
							return
						}
					}
				}

				contentBuilder.Write(chunk)
				totalRead += int64(n)

				if totalSize > 0 {
					progress := float64(totalRead) / float64(totalSize)
					ch <- copyProgressMsg{progress: progress}
				}
			}

			if err != nil {
				if err == io.EOF {
					break
				}
				ch <- copyFinishedMsg{err: err}
				return
			}
		}

		// Copy to clipboard
		if err := copyToClipboard(contentBuilder.String()); err != nil {
			ch <- copyFinishedMsg{err: err}
			return
		}

		ch <- copyFinishedMsg{err: nil}
	}(filePath, m.copyProgressChan, m.copyJobCancel)

	return m.listenCopyProgress()
}

func (m *model) listenCopyProgress() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-m.copyProgressChan
		if !ok {
			return nil
		}
		return msg
	}
}

type ShortcutEntry struct {
	Name string
	Path string
	Icon string
}

func buildShortcuts() []ShortcutEntry {
	home, _ := os.UserHomeDir()
	if home == "" {
		home = "/"
	}
	candidates := []ShortcutEntry{
		{Name: "Home", Path: home, Icon: "\uf015"},
		{Name: "Desktop", Path: filepath.Join(home, "Desktop"), Icon: "\uf108"},
		{Name: "Documents", Path: filepath.Join(home, "Documents"), Icon: "\ue6ad"},
		{Name: "Downloads", Path: filepath.Join(home, "Downloads"), Icon: "\uf019"},
		{Name: "Music", Path: filepath.Join(home, "Music"), Icon: "\uf001"},
		{Name: "Pictures", Path: filepath.Join(home, "Pictures"), Icon: "\uf03e"},
		{Name: "Videos", Path: filepath.Join(home, "Videos"), Icon: "\uf52c"},
		// {Name: "/", Path: "/", Icon: "\uf013"},
	}
	var existing []ShortcutEntry
	for _, c := range candidates {
		if _, err := os.Stat(c.Path); err == nil {
			existing = append(existing, c)
		}
	}
	return existing
}

type editorFinishedMsg struct {
	err error
}

func InitialModel() model {
	currentDir, _ := os.Getwd()
	entries, err := os.ReadDir(currentDir)
	files := []string{}
	if err == nil {
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			files = append(files, entry.Name())
		}
	}

	initialTab := TabState{
		currentDir: currentDir,
		filesPath:  files,
	}

	return model{
		tabs:             []TabState{initialTab},
		activeTab:        0,
		selected:         make(map[int]struct{}),
		shortcuts:        buildShortcuts(),
		bookmarks:        loadBookmarks(),
		activePane:       1,
		shortcutPaneOpen: true,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

// getMaxVisibleFiles returns the exact number of file rows that fit inside
// the File Explorer pane. Both adjustScroll and View must use this value so
// they always agree on the scroll window. // ponytail: single source of truth
func (m *model) getMaxVisibleFiles() int {
	bottomBarLines := 1
	if m.err != nil {
		bottomBarLines = 2
	}
	boxHeight := m.terminalHeight - bottomBarLines
	if boxHeight < 3 {
		boxHeight = 3
	}
	leftHeight := boxHeight - 2 // subtract top+bottom border
	if m.copyJobActive {
		leftHeight -= 4 // progress pane height
		if leftHeight < 3 {
			leftHeight = 3
		}
	}
	// rows consumed inside the box beyond the file list:
	//   1 = path header line
	//   1 = metadata bar at bottom
	//   2 = input prompt lines (only in inputMode)
	consumed := 2
	if m.inputMode {
		consumed += 2
	}
	maxVisible := leftHeight - consumed
	if maxVisible < 1 {
		maxVisible = 1
	}
	return maxVisible
}

func (m *model) adjustScroll() {
	t := m.curTab()
	if len(t.filesPath) == 0 {
		t.scrollOffset = 0
		return
	}

	maxVisible := m.getMaxVisibleFiles()

	if t.scrollOffset < 0 {
		t.scrollOffset = 0
	}

	if t.cursor < t.scrollOffset {
		t.scrollOffset = t.cursor
	}

	if t.cursor >= t.scrollOffset+maxVisible {
		t.scrollOffset = t.cursor - maxVisible + 1
	}

	if t.scrollOffset > len(t.filesPath)-maxVisible {
		t.scrollOffset = len(t.filesPath) - maxVisible
	}
	if t.scrollOffset < 0 {
		t.scrollOffset = 0
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
				if t.cursor < len(t.filesPath)-1 {
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
		if !m.inputMode && !m.copyMenuOpen && !m.bookmarkMenuOpen && !m.deleteBookmarkConfirmOpen && !m.deleteMenuOpen && !m.permissionMenuOpen && !m.openMenuOpen && !m.helpMenuOpen && !m.metadataInspectorOpen {
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
		}

		if m.copyMenuOpen {
			switch msg.String() {
			case "p", "P":
				m.copyMenuOpen = false
				if len(t.filesPath) > 0 {
					absPath := findAbsPath(t.currentDir, t.filesPath[t.cursor])
					if err := copyToClipboard(absPath); err != nil {
						m.err = err
					}
				}
			case "n", "N":
				m.copyMenuOpen = false
				if len(t.filesPath) > 0 {
					name := t.filesPath[t.cursor]
					if !isDir(t.currentDir, name) {
						name = strings.TrimSuffix(name, filepath.Ext(name))
					}
					if err := copyToClipboard(name); err != nil {
						m.err = err
					}
				}
			case "e", "E":
				m.copyMenuOpen = false
				if len(t.filesPath) > 0 {
					name := t.filesPath[t.cursor]
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
				if len(t.filesPath) > 0 {
					name := t.filesPath[t.cursor]
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
				if len(t.filesPath) > 0 {
					name := t.filesPath[t.cursor]
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
				} else if len(t.filesPath) > 0 {
					targetIndices = append(targetIndices, t.cursor)
				}

				deletedCount := 0
				for _, idx := range targetIndices {
					if idx >= 0 && idx < len(t.filesPath) {
						absPath := findAbsPath(t.currentDir, t.filesPath[idx])
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
				} else if len(t.filesPath) > 0 {
					targetIndices = append(targetIndices, t.cursor)
				}

				deletedCount := 0
				for _, idx := range targetIndices {
					if idx >= 0 && idx < len(t.filesPath) {
						absPath := findAbsPath(t.currentDir, t.filesPath[idx])
						isDirItem := isDir(t.currentDir, t.filesPath[idx])
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
				if len(t.filesPath) > 0 {
					absPath := findAbsPath(t.currentDir, t.filesPath[t.cursor])
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
				if len(t.filesPath) > 0 {
					targetPath := findAbsPath(t.currentDir, t.filesPath[t.cursor])
					if err := openWithVSCode(targetPath); err != nil {
						m.err = err
					}
				}
			case "r", "R":
				m.openMenuOpen = false
				if len(t.filesPath) > 0 {
					targetPath := findAbsPath(t.currentDir, t.filesPath[t.cursor])
					if err := revealInFileExplorer(targetPath); err != nil {
						m.err = err
					}
				}
			case "e", "E":
				m.openMenuOpen = false
				if len(t.filesPath) > 0 {
					name := t.filesPath[t.cursor]
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

		// Shortcut pane navigation
		if m.shortcutPaneOpen && m.activePane == 0 {
			switch msg.String() {
			// ponytail: Allow global exit and exit-with-cd actions while shortcut pane is open
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
					if len(t.filesPath) > 0 {
						oldName := t.filesPath[t.cursor]
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
			if len(t.filesPath) == 0 {
				break
			}
			m.err = nil
			if m.activePane == 1 {
				t.previewScrollOffset = 0
				if t.cursor > 0 {
					t.cursor--
				} else {
					t.cursor = len(t.filesPath) - 1
				}
				m.adjustScroll()
			} else if m.activePane == 2 {
				if t.previewScrollOffset > 0 {
					t.previewScrollOffset--
				}
			}
		case "down", "j":
			if len(t.filesPath) == 0 {
				break
			}
			m.err = nil
			if m.activePane == 1 {
				t.previewScrollOffset = 0
				if t.cursor < len(t.filesPath)-1 {
					t.cursor++
				} else {
					t.cursor = 0
				}
				m.adjustScroll()
			} else if m.activePane == 2 {
				t.previewScrollOffset++
			}

		case "space":
			if len(t.filesPath) == 0 {
				break
			}
			_, ok := m.selected[t.cursor]
			if ok {
				delete(m.selected, t.cursor)
			} else {
				m.selected[t.cursor] = struct{}{}
			}

		case "enter":
			if len(t.filesPath) == 0 {
				break
			}
			if isDir(t.currentDir, t.filesPath[t.cursor]) {
				selectedName := t.filesPath[t.cursor]
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
			if len(t.filesPath) == 0 {
				break
			}
			m.err = nil
			m.inputMode = true
			m.isRenaming = true
			m.inputText = t.filesPath[t.cursor]
			// ponytail: cursor before ".ext"; dirs and dotfiles go to end
			name := t.filesPath[t.cursor]
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
			if len(t.filesPath) == 0 {
				break
			}
			m.err = nil
			if len(m.selected) == len(t.filesPath) {
				m.selected = make(map[int]struct{})
				m.statusMsg = "Deselected all item(s)"
			} else {
				m.selected = make(map[int]struct{})
				for i := range t.filesPath {
					m.selected[i] = struct{}{}
				}
				m.statusMsg = fmt.Sprintf("Selected all %d item(s)", len(t.filesPath))
			}
		case "g", "G":
			if len(t.filesPath) == 0 {
				break
			}
			m.err = nil
			name := t.filesPath[t.cursor]
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
				for idx, fname := range t.filesPath {
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
			if len(t.filesPath) == 0 {
				break
			}
			m.err = nil
			selectedName := t.filesPath[t.cursor]
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
					if idx >= 0 && idx < len(t.filesPath) {
						absP := findAbsPath(t.currentDir, t.filesPath[idx])
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
					if idx >= 0 && idx < len(t.filesPath) {
						absP := findAbsPath(t.currentDir, t.filesPath[idx])
						m.stagedPaths = append(m.stagedPaths, absP)
					}
				}
				m.stagedOp = "copy"
				m.selected = make(map[int]struct{})
				m.statusMsg = fmt.Sprintf("Copied %d item(s). Press 'v' or 'p' to paste.", len(m.stagedPaths))
			} else if len(t.filesPath) > 0 {
				absPath := findAbsPath(t.currentDir, t.filesPath[t.cursor])
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
			if len(t.filesPath) == 0 {
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
			if len(t.filesPath) == 0 {
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
			} else if len(t.filesPath) > 0 {
				m.err = nil
				m.permissionMenuOpen = true
			}

		// Metadata Inspector dialog ('i' / 'I')
		case "i", "I":
			if len(t.filesPath) == 0 {
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
			if len(t.filesPath) == 0 {
				break
			}
			m.err = nil
			m.openMenuOpen = true

		case "right", "l", "L":
			if len(t.filesPath) == 0 {
				break
			}
			m.err = nil
			if !isDir(t.currentDir, t.filesPath[t.cursor]) {
				break
			}

			selectedName := t.filesPath[t.cursor]
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
		}
	}

	return m, nil
}

func (m model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

	t := m.curTab()
	renderedText := fmt.Sprintf(" %s%s: %s%s\n", Cyan, getCurrentUser(), t.currentDir, Reset)

	maxVisible := m.getMaxVisibleFiles()

	endIdx := t.scrollOffset + maxVisible
	if endIdx > len(t.filesPath) {
		endIdx = len(t.filesPath)
	}

	for i := t.scrollOffset; i < endIdx; i++ {
		list := t.filesPath[i]

		displayName := list
		if target, isSym, isBroken := getSymlinkTarget(t.currentDir, list); isSym {
			if isBroken {
				displayName = fmt.Sprintf("%s -> %s [Broken]", list, target)
			} else {
				displayName = fmt.Sprintf("%s -> %s", list, target)
			}
		}

		icon, fgColor, bgColor := m.CheckDirFilesExt(list)

		cursorSymbol := " "
		if t.cursor == i {
			cursorSymbol = ">"
		}
		selectMarker := " "
		if _, ok := m.selected[i]; ok {
			selectMarker = fmt.Sprintf("%s*%s", Yellow, Reset)
		}

		if t.cursor == i {
			renderedText += fmt.Sprintf("%s%s %s%s %s %s%s%s\n", bgColor, Black, cursorSymbol, selectMarker, icon, displayName, ItemEndSpace, Reset)
		} else {
			if fgColor == Reset {
				renderedText += fmt.Sprintf(" %s%s %s %s%s\n", cursorSymbol, selectMarker, icon, displayName, ItemEndSpace)
			} else {
				renderedText += fmt.Sprintf(" %s%s %s%s %s%s%s \n", cursorSymbol, selectMarker, fgColor, icon, displayName, ItemEndSpace, Reset)
			}
		}
	}

	// Dynamic padding: push metadata to bottom of the box interior
	numRendered := endIdx - t.scrollOffset
	paddingLines := maxVisible - numRendered
	if paddingLines > 0 {
		renderedText += strings.Repeat("\n", paddingLines)
	}

	// Metadata bar pinned at bottom inside the border
	if len(t.filesPath) > 0 && t.cursor >= 0 && t.cursor < len(t.filesPath) {
		meta := getFileMetadataInfo(t.currentDir, t.filesPath[t.cursor])
		renderedText += fmt.Sprintf("%s─%s %s%s | %s | %s | %s%s\n",
			Gray, Reset, Yellow, meta.SizeStr, meta.PermStr, meta.OwnerGroup, meta.ModTimeStr, Reset)
	}

	if m.inputMode {
		prompt := "Create file"
		if m.isRenaming {
			prompt = "Rename"
		} else if m.isCreatingDir {
			prompt = "Create directory"
		}
		// ponytail: split at cursor rune index, no extra lib
		runes := []rune(m.inputText)
		cur := m.inputCursor
		if cur > len(runes) {
			cur = len(runes)
		}
		before := string(runes[:cur])
		after := string(runes[cur:])
		renderedText += fmt.Sprintf("%s: %s█%s\n(Enter to confirm, Esc to cancel)\n", prompt, before, after)
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

	// Calculate shortcut pane width
	shortcutWidth := 0
	if m.shortcutPaneOpen {
		shortcutWidth = width / 4
		if shortcutWidth < 14 {
			shortcutWidth = 14
		}
	}

	var leftWidth, rightWidth int
	if m.shortcutPaneOpen {
		leftWidth = width / 2
		if leftWidth < 10 {
			leftWidth = 10
		}
		rightWidth = width - shortcutWidth - leftWidth
		if rightWidth < 10 {
			rightWidth = 10
		}
	} else {
		availableWidth := width - shortcutWidth
		leftWidth = (availableWidth * 2) / 3
		if leftWidth < 10 {
			leftWidth = availableWidth / 2
		}
		rightWidth = availableWidth - leftWidth
	}

	leftBorderColor := "#555555"
	rightBorderColor := "#555555"
	shortcutBorderColor := "#555555"
	if m.shortcutPaneOpen {
		if m.activePane == 0 {
			shortcutBorderColor = "#5555FF"
		} else {
			shortcutBorderColor = "#FFFF55"
		}
	}
	if m.activePane == 1 {
		leftBorderColor = "#5555FF"
	}
	if m.activePane == 2 {
		rightBorderColor = "#5555FF"
	}

	// 0. Create shortcut pane (if open)
	var shortcutBox string
	if m.shortcutPaneOpen {
		var shortcutText string
		var activeShortcuts []ShortcutEntry
		if m.shortcutTab == 0 {
			activeShortcuts = m.shortcuts
		} else {
			activeShortcuts = m.bookmarks
		}

		if len(activeShortcuts) == 0 {
			shortcutText = " (No bookmarks)\n Press 'b' to add"
		} else {
			for i, s := range activeShortcuts {
				cursor := "  "
				if i == m.shortcutCursor {
					cursor = "> "
				}
				label := fmt.Sprintf("%s%s  %s%s", cursor, s.Icon, s.Name, Reset)
				if i == m.shortcutCursor {
					label = fmt.Sprintf("%s%s%s  %s%s", BgCyan, Black, cursor+s.Icon, s.Name, Reset)
				}
				shortcutText += label + "\n"
			}
			shortcutText = strings.TrimSuffix(shortcutText, "\n")

			// ponytail: Append full path of active shortcut at the bottom of the pane
			if m.shortcutCursor >= 0 && m.shortcutCursor < len(activeShortcuts) {
				selectedPath := activeShortcuts[m.shortcutCursor].Path
				maxWidth := shortcutWidth - 4
				if maxWidth < 5 {
					maxWidth = 5
				}
				truncatedPath := truncatePathLeft(selectedPath, maxWidth)
				sep := strings.Repeat("─", shortcutWidth-2)

				linesCount := strings.Count(shortcutText, "\n") + 1
				targetHeight := boxHeight - 2
				paddingNeeded := targetHeight - linesCount - 2 // -2 for sep and path line

				if paddingNeeded > 0 {
					shortcutText += strings.Repeat("\n", paddingNeeded)
				} else {
					shortcutText += "\n"
				}
				shortcutText += fmt.Sprintf("%s%s%s\n %s%s%s", Gray, sep, Reset, Yellow, truncatedPath, Reset)
			}
		}

		shortcutStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(shortcutBorderColor)).
			Width(shortcutWidth - 2).
			Height(boxHeight - 2)

		shortcutBox = shortcutStyle.Render(shortcutText)

		// Overlay title on border depending on tab and width
		scLines := strings.Split(shortcutBox, "\n")
		if len(scLines) > 0 {
			var title string
			if shortcutWidth-2 >= 27 {
				if m.shortcutTab == 0 {
					title = " [Shortcuts] │ Bookmarks "
				} else {
					title = " Shortcuts │ [Bookmarks] "
				}
			} else {
				if m.shortcutTab == 0 {
					title = " [Shortcuts] "
				} else {
					title = " [Bookmarks] "
				}
			}
			scLines[0] = createTopBorder(title, shortcutWidth, shortcutBorderColor, "1")
			shortcutBox = strings.Join(scLines, "\n")
		}
	}

	// 1. Create left box (File Explorer)
	leftHeight := boxHeight - 2
	if m.copyJobActive {
		leftHeight -= 4
		if leftHeight < 3 {
			leftHeight = 3
		}
	}
	leftStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(leftBorderColor)).
		Width(leftWidth - 2).
		Height(leftHeight)

	leftBox := leftStyle.Render(renderedText)

	// Overlay tabs on left border
	leftLines := strings.Split(leftBox, "\n")
	if len(leftLines) > 0 {
		var tabParts []string
		for i, tb := range m.tabs {
			dirName := filepath.Base(tb.currentDir)
			if dirName == "" || dirName == "." || dirName == "/" {
				dirName = tb.currentDir
			}
			if i == m.activeTab {
				tabParts = append(tabParts, fmt.Sprintf("[%s]", dirName))
			} else {
				tabParts = append(tabParts, dirName)
			}
		}
		rawTitle := " " + strings.Join(tabParts, " | ") + " "
		title := []rune(rawTitle)
		maxTitleLen := leftWidth - 4
		if maxTitleLen > 6 && len(title) > maxTitleLen {
			pathTrimmed := " ..." + string(title[len(title)-(maxTitleLen-4):])
			title = []rune(pathTrimmed)
		}
		leftLines[0] = createTopBorder(string(title), leftWidth, leftBorderColor, "2")
		leftBox = strings.Join(leftLines, "\n")
	}

	if m.copyJobActive {
		progressWidth := leftWidth
		if progressWidth < 14 {
			progressWidth = 14
		}

		barWidth := progressWidth - 25 // subtract space for borders and labels
		if barWidth < 5 {
			barWidth = 5
		}
		filledLen := int(m.copyJobProgress * float64(barWidth))
		if filledLen > barWidth {
			filledLen = barWidth
		}
		emptyLen := barWidth - filledLen
		barStr := strings.Repeat("█", filledLen) + strings.Repeat("░", emptyLen)

		percentStr := fmt.Sprintf("%.1f%%", m.copyJobProgress*100)
		progressContent := fmt.Sprintf("Copying: %s\n[%s] %s (Esc to cancel)", m.copyJobFileName, barStr, percentStr)

		progressStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#55FFFF")). // Cyan border
			Padding(0, 1).
			Width(progressWidth - 2).
			Height(2) // 4 lines total including borders

		progressPane := progressStyle.Render(progressContent)
		leftBox = lipgloss.JoinVertical(lipgloss.Left, leftBox, progressPane)
	}

	// 2. Create right box (Previewer)
	var rightText string
	maxVisiblePreviewLines := boxHeight - 2
	if maxVisiblePreviewLines < 1 {
		maxVisiblePreviewLines = 1
	}

	if len(t.filesPath) > 0 {
		selectedName := t.filesPath[t.cursor]
		targetPath := findAbsPath(t.currentDir, selectedName)
		previewLines := getPreviewLines(targetPath, isDir(t.currentDir, selectedName), m.showHidden)

		// Clamp previewScrollOffset based on number of lines and visible height
		maxScroll := len(previewLines) - maxVisiblePreviewLines
		if maxScroll < 0 {
			maxScroll = 0
		}
		if t.previewScrollOffset > maxScroll {
			t.previewScrollOffset = maxScroll
		}
		if t.previewScrollOffset < 0 {
			t.previewScrollOffset = 0
		}

		endLine := t.previewScrollOffset + maxVisiblePreviewLines
		if endLine > len(previewLines) {
			endLine = len(previewLines)
		}

		maxWidth := rightWidth - 2
		if maxWidth < 1 {
			maxWidth = 1
		}
		var truncatedLines []string
		for _, line := range previewLines[t.previewScrollOffset:endLine] {
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
		rightLines[0] = createTopBorder(" Preview ", rightWidth, rightBorderColor, "3")
		rightBox = strings.Join(rightLines, "\n")
	}

	// Join all boxes horizontally
	var content string
	if m.shortcutPaneOpen {
		content = lipgloss.JoinHorizontal(lipgloss.Top, shortcutBox, leftBox, rightBox)
	} else {
		content = lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
	}

	// Bottom Help & Error Status Bar
	// hiddenStatus := "OFF"
	if m.showHidden {
		// hiddenStatus = "ON"
	}
	// helpText := fmt.Sprintf(" Press 'q' to quit | ':' shell | 'space' select | 'a' all | 'c' copy | 'm' move | 'v'/'p' paste | 'g' jump link | 'K' mklink | 'e' exit&cd | '.' hidden:%s | '`' help", hiddenStatus)

	if m.err != nil {
		content += fmt.Sprintf("\n%sError: %v%s", Red, m.err, Reset)
	} else if m.statusMsg != "" {
		content += fmt.Sprintf("\n%s%s%s", Green, m.statusMsg, Reset)
	}
	// content += "\n" + helpText
	content += "\n"
	// Render copy menu if open
	if m.copyMenuOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#55FF55")). // Green border
			Padding(1, 2)

		dialogContent := "         Copy Menu\n\n" +
			"[p] Copy absolute path\n" +
			"[n] Copy name only\n" +
			"[e] Copy name with ext\n" +
			"[s] Copy dir structure (tree)\n" +
			"[c] Copy file content\n\n" +
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

	// Render bookmark menu if open
	if m.bookmarkMenuOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#FF55FF")). // Purple border
			Padding(1, 2)

		dialogContent := "    Bookmark Options\n\n" +
			"[s] Bookmark Selected Dir\n" +
			"[c] Bookmark Current CWD\n\n" +
			"Press any other key to close"

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

	// Render delete bookmark confirmation menu if open
	if m.deleteBookmarkConfirmOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#FF5555")). // Red border
			Padding(1, 2)

		var bookmarkName string
		if m.shortcutCursor >= 0 && m.shortcutCursor < len(m.bookmarks) {
			bookmarkName = m.bookmarks[m.shortcutCursor].Name
		}
		dialogContent := fmt.Sprintf("   Remove Bookmark?\n\nRemove bookmark for %q?\n\n   [y] Yes      [n] No", bookmarkName)
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

	// Render open menu if open
	if m.openMenuOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#FF55FF")). // Purple border
			Padding(1, 2)

		dialogContent := "       Open Menu\n\n" +
			"[c] Open CWD with VSCode (code .)\n" +
			"[C] Open selected with VSCode\n" +
			"[r] Reveal in GUI File Explorer\n" +
			"[e] Open with $EDITOR\n\n" +
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
			"  tab                Switch active pane\n" +
			"  [ / ]              Switch tab prev / next (or switch shortcuts/bookmarks tab in sidebar)\n" +
			"  t / w              New tab / close tab\n" +
			"  ctrl+b             Toggle shortcut sidebar pane\n" +
			"  b                  Open bookmark menu (add selected directory / CWD)\n" +
			"  d                  Remove bookmark (only when inside Bookmarks tab)\n\n" +
			"File Actions:\n" +
			"  f / m              Create file / directory\n" +
			"  a                  Toggle select all / deselect all\n" +
			"  i                  File metadata inspector\n" +
			"  K                  Create symlink for selected item\n" +
			"  g                  Jump / follow symlink target\n" +
			"  r                  Rename file or directory\n" +
			"  d                  Delete menu (p: permanent, n: trash)\n" +
			"  y                  Copy menu (p: path, n: name, e: ext, s: tree)\n" +
			"  p                  Permission menu (x: chmod +x/-x)\n" +
			"  o                  Open menu [c: VSCode CWD, C: VSCode item, r: GUI, e: $EDITOR]\n" +
			"  .                  Toggle hidden files\n\n" +
			"  :              	  Temporary switch to subshell (type exit or ctrl+d to return)\n" +
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

	// Render Metadata Inspector dialog if open
	if m.metadataInspectorOpen && len(t.filesPath) > 0 && t.cursor >= 0 && t.cursor < len(t.filesPath) {
		meta := getFileMetadataInfo(t.currentDir, t.filesPath[t.cursor])

		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#55FFFF")). // Cyan border
			Padding(1, 3)

		symlinkLine := ""
		if meta.IsSymlink {
			symlinkLine = fmt.Sprintf("Symlink Target: %s\n", meta.SymTarget)
		}

		dialogContent := fmt.Sprintf("File Metadata Inspector\n\n"+
			"Name:        %s\n"+
			"Path:        %s\n"+
			"Size:        %s (%d bytes)\n"+
			"Permissions: %s\n"+
			"Owner/Group: %s\n"+
			"Modified:    %s\n"+
			"Type:        %s\n"+
			"%s\n"+
			"     Press ESC or any key to close",
			meta.Name, meta.AbsPath, meta.SizeStr, meta.SizeBytes,
			meta.PermStr, meta.OwnerGroup, meta.ModTimeStr, meta.MimeOrType, symlinkLine)

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
