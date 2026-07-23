package tui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
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
	filteredFiles       []string
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
	searchActive              bool
	searchText                string
	searchCursor              int
}

func (t *TabState) visibleFiles() []string {
	if t.filteredFiles != nil {
		return t.filteredFiles
	}
	return t.filesPath
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

func (m model) isAnyMenuOpen() bool {
	return m.copyMenuOpen || m.bookmarkMenuOpen || m.deleteBookmarkConfirmOpen ||
		m.deleteMenuOpen || m.permissionMenuOpen || m.openMenuOpen ||
		m.helpMenuOpen || m.metadataInspectorOpen
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
	//   3 = search bar lines (with border)
	//   1 = metadata bar at bottom
	//   2 = input prompt lines (only in inputMode)
	consumed := 5
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
	if len(t.visibleFiles()) == 0 {
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

	if t.scrollOffset > len(t.visibleFiles())-maxVisible {
		t.scrollOffset = len(t.visibleFiles()) - maxVisible
	}
	if t.scrollOffset < 0 {
		t.scrollOffset = 0
	}
}
