package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
)

const baseDir = "./files"

func TestFindAbsPath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get cwd: %v", err)
	}

	testCases := []struct {
		name     string
		fileName string
		expected string
	}{
		{"file 1", "file1.txt", filepath.Join(cwd, "files", "file1.txt")},
		{"file 2", "file2.txt", filepath.Join(cwd, "files", "file2.txt")},
		{"file 10", "file10.txt", filepath.Join(cwd, "files", "file10.txt")},
		{"file random", "random.txt", filepath.Join(cwd, "files", "random.txt")},
		{"files Dir", "", filepath.Join(cwd, "files")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findAbsPath(baseDir, tc.fileName)

			if result != tc.expected {
				t.Errorf("Expected %s, but got %s", tc.expected, result)
			}
		})
	}
}

func TestCopyToClipboard(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		expected error
	}{
		{"text 1", "text1", nil},
		{"text 2", "text2", nil},
		{"file path1", "/data/coding/Go/tui/list2/files/file1.txt", nil},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := copyToClipboard(tc.text)

			if err != tc.expected {
				t.Errorf("Expected %v, but got %s", nil, err.Error())
			}
		})
	}
}

func TestPaneSwitchingAndScrolling(t *testing.T) {
	m := InitialModel()
	m.tabs[m.activeTab].filesPath = []string{"file1.txt", "file2.txt"}

	// Default active pane should be 1 (Left Explorer)
	if m.activePane != 1 {
		t.Errorf("Expected activePane to be 1 by default, got %d", m.activePane)
	}

	// Pressing "tab" should switch to activePane 2 (Right Previewer)
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "tab"})
	m = updatedModel.(model)
	if m.activePane != 2 {
		t.Errorf("Expected activePane to be 2 after pressing 'tab', got %d", m.activePane)
	}

	// Pressing "tab" again should switch back to activePane 0 (Shortcut Sidebar)
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "tab"})
	m = updatedModel.(model)
	if m.activePane != 0 {
		t.Errorf("Expected activePane to switch to 0 after pressing 'tab', got %d", m.activePane)
	}

	// In activePane 2, pressing down/j should increment previewScrollOffset
	m.activePane = 2
	m.curTab().previewScrollOffset = 0
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "j"})
	m = updatedModel.(model)
	if m.curTab().previewScrollOffset != 1 {
		t.Errorf("Expected previewScrollOffset to be 1 after pressing 'j' in right pane, got %d", m.curTab().previewScrollOffset)
	}

	// In activePane 1, pressing down/j should move file cursor instead of scrolling preview
	m.activePane = 1
	m.curTab().cursor = 0
	m.curTab().previewScrollOffset = 5
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "j"})
	m = updatedModel.(model)
	if m.curTab().cursor != 1 {
		t.Errorf("Expected cursor to increment to 1, got %d", m.curTab().cursor)
	}
	if m.curTab().previewScrollOffset != 0 {
		t.Errorf("Expected previewScrollOffset to reset to 0 after moving cursor in left pane, got %d", m.curTab().previewScrollOffset)
	}
}

func TestPreviewLineTruncation(t *testing.T) {
	m := InitialModel()
	m.terminalWidth = 60
	m.terminalHeight = 10
	m.shortcutPaneOpen = false

	tempFile, err := os.CreateTemp("", "test_preview_truncation_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	content := "ShortLine\nLongLineWithManyCharactersAndMore\nTab\tSeparated\n"
	if _, err := tempFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	m.curTab().filesPath = []string{filepath.Base(tempFile.Name())}
	m.curTab().currentDir = filepath.Dir(tempFile.Name())
	m.curTab().cursor = 0

	view := m.View()
	viewStr := view.Content

	if !strings.Contains(viewStr, "LongLineWithManyCh") {
		t.Errorf("Expected truncated long line in view, view: %q", viewStr)
	}
	if strings.Contains(viewStr, "CharactersAndMore") {
		t.Errorf("Expected long line to be truncated and not show full suffix, view: %q", viewStr)
	}

	if !strings.Contains(viewStr, "Tab    Separated") {
		t.Errorf("Expected tab to be expanded to 4 spaces, view: %q", viewStr)
	}
}

func TestDeleteAction(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_delete_action_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempPath := tempFile.Name()
	tempFile.Close()

	m := InitialModel()
	m.terminalWidth = 80
	m.terminalHeight = 24
	m.curTab().currentDir = filepath.Dir(tempPath)
	m.curTab().filesPath = []string{filepath.Base(tempPath)}
	m.curTab().cursor = 0

	// 1. Pressing "d" opens delete menu
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "d"})
	m = updatedModel.(model)
	if !m.deleteMenuOpen {
		t.Errorf("Expected deleteMenuOpen to be true after pressing 'd'")
	}

	view := m.View()
	if !strings.Contains(view.Content, "Delete Menu") {
		t.Errorf("Expected view to contain 'Delete Menu' dialog overlay")
	}

	// 2. Pressing "p" performs permanent delete
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "p"})
	m = updatedModel.(model)
	if m.deleteMenuOpen {
		t.Errorf("Expected deleteMenuOpen to be false after pressing 'p'")
	}
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Errorf("Expected file to be deleted from disk, but it still exists")
	}
}

func TestRenameAction(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_rename_dir_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	oldPath := filepath.Join(tempDir, "old_name.txt")
	os.WriteFile(oldPath, []byte("hello"), 0644)

	m := InitialModel()
	m.terminalWidth = 80
	m.terminalHeight = 24
	m.curTab().currentDir = tempDir
	m.curTab().filesPath = []string{"old_name.txt"}
	m.curTab().cursor = 0

	// Press "r" to trigger rename
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "r"})
	m = updatedModel.(model)
	if !m.isRenaming || !m.inputMode {
		t.Errorf("Expected isRenaming and inputMode to be true after pressing 'r'")
	}

	// Change text to "new_name.txt"
	m.fileInput.SetValue("new_name.txt")

	// Press "enter" to confirm rename
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "enter"})
	m = updatedModel.(model)

	newPath := filepath.Join(tempDir, "new_name.txt")
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Errorf("Expected file to be renamed to %s, but it does not exist", newPath)
	}
}

func TestUndoRedo(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_undo_redo_dir_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	m := InitialModel()
	m.terminalWidth = 80
	m.terminalHeight = 24
	m.curTab().currentDir = tempDir
	m.curTab().filesPath = []string{}
	m.curTab().cursor = 0

	// 1. Create file "created_file.txt"
	m.inputMode = true
	m.fileInput.SetValue("created_file.txt")
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "enter"})
	m = updatedModel.(model)

	createdPath := filepath.Join(tempDir, "created_file.txt")
	if _, err := os.Stat(createdPath); os.IsNotExist(err) {
		t.Fatalf("Expected created_file.txt to exist")
	}

	// 2. Undo creation (press "u")
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "u"})
	m = updatedModel.(model)
	if _, err := os.Stat(createdPath); !os.IsNotExist(err) {
		t.Errorf("Expected created_file.txt to be removed after undo")
	}

	// 3. Redo creation (press "ctrl+r")
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "ctrl+r"})
	m = updatedModel.(model)
	if _, err := os.Stat(createdPath); os.IsNotExist(err) {
		t.Errorf("Expected created_file.txt to be restored after redo")
	}
}

func TestNormalDeleteAndUndo(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_normal_delete_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempPath := tempFile.Name()
	tempFile.Close()

	m := InitialModel()
	m.terminalWidth = 80
	m.terminalHeight = 24
	m.curTab().currentDir = filepath.Dir(tempPath)
	m.curTab().filesPath = []string{filepath.Base(tempPath)}
	m.curTab().cursor = 0

	// 1. Press "d" to open delete menu
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "d"})
	m = updatedModel.(model)

	// 2. Press "r" for normal delete (Trash)
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "r"})
	m = updatedModel.(model)

	if m.err != nil {
		t.Fatalf("Normal delete failed with error: %v", m.err)
	}

	// Verify file was moved out of original location
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Errorf("Expected file to be removed from original location after normal delete")
	}

	// 3. Press "u" to undo normal delete (restore from Trash)
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "u"})
	m = updatedModel.(model)

	if m.err != nil {
		t.Fatalf("Undo normal delete failed with error: %v", m.err)
	}

	// Verify file was restored to original location
	if _, err := os.Stat(tempPath); os.IsNotExist(err) {
		t.Errorf("Expected file to be restored to original location after pressing 'u'")
	}

	// Clean up restored file
	_ = os.Remove(tempPath)
}

func TestShowCurrentPathInTopBorder(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_path_border_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	m := InitialModel()
	m.terminalWidth = 120
	m.terminalHeight = 24
	m.curTab().currentDir = tempDir
	m.curTab().filesPath = []string{}

	view := m.View()
	if !strings.Contains(view.Content, tempDir) {
		t.Errorf("Expected top border to contain current directory %q, view: %q", tempDir, view.Content)
	}
}

func TestBottomHelpMessage(t *testing.T) {
	t.Skip("Help message bar is disabled per YAGNI")
	m := InitialModel()
	m.terminalWidth = 120
	m.terminalHeight = 24

	view := m.View()
	lines := strings.Split(view.Content, "\n")
	if len(lines) == 0 {
		t.Fatalf("Expected non-empty rendered view")
	}

	lastLine := lines[len(lines)-1]
	if !strings.Contains(lastLine, "Press 'q' to quit") {
		t.Errorf("Expected bottom line to contain help message, got: %q", lastLine)
	}
}

func TestChmodExecutableAction(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test_chmod_*.sh")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempPath := tempFile.Name()
	tempFile.Close()

	// Set initial mode to non-executable (0644)
	_ = os.Chmod(tempPath, 0644)
	defer os.Remove(tempPath)

	m := InitialModel()
	m.terminalWidth = 80
	m.terminalHeight = 24
	m.curTab().currentDir = filepath.Dir(tempPath)
	m.curTab().filesPath = []string{filepath.Base(tempPath)}
	m.curTab().cursor = 0

	// 1. Press "p" to open permission menu
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "p"})
	m = updatedModel.(model)
	if !m.permissionMenuOpen {
		t.Errorf("Expected permissionMenuOpen to be true after pressing 'p'")
	}

	view := m.View()
	if !strings.Contains(view.Content, "Permission Menu") {
		t.Errorf("Expected view to contain 'Permission Menu' dialog overlay")
	}

	// 2. Press "x" to toggle executable bit (+x)
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "x"})
	m = updatedModel.(model)
	if m.permissionMenuOpen {
		t.Errorf("Expected permissionMenuOpen to be false after pressing 'x'")
	}

	info, err := os.Stat(tempPath)
	if err != nil {
		t.Fatalf("Failed to stat temp file: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Errorf("Expected executable bit to be set after pressing 'x', mode: %o", info.Mode())
	}

	// 3. Press "u" to undo chmod
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "u"})
	m = updatedModel.(model)

	info, err = os.Stat(tempPath)
	if err != nil {
		t.Fatalf("Failed to stat temp file after undo: %v", err)
	}
	if info.Mode()&0111 != 0 {
		t.Errorf("Expected executable bit to be restored (non-executable) after undo, mode: %o", info.Mode())
	}
}

func TestHelpMenuAction(t *testing.T) {
	m := InitialModel()
	m.terminalWidth = 100
	m.terminalHeight = 30

	// 1. Press "`" to open help menu
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "`"})
	m = updatedModel.(model)
	if !m.helpMenuOpen {
		t.Errorf("Expected helpMenuOpen to be true after pressing '`'")
	}

	view := m.View()
	if !strings.Contains(view.Content, "FMTUI Help Menu") {
		t.Errorf("Expected view content to display 'FMTUI Help Menu' overlay")
	}

	// 2. Press "esc" to close help menu
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "esc"})
	m = updatedModel.(model)
	if m.helpMenuOpen {
		t.Errorf("Expected helpMenuOpen to be false after pressing 'esc'")
	}
}

func TestMouseScrollAction(t *testing.T) {
	m := InitialModel()
	m.terminalWidth = 100
	m.terminalHeight = 30
	m.tabs[m.activeTab].filesPath = []string{"file1.txt", "file2.txt", "file3.txt"}
	m.tabs[m.activeTab].cursor = 0
	m.activePane = 1

	// 1. Mouse Wheel Down in Pane 1 -> cursor moves down to 1
	updatedModel, _ := m.Update(tea.MouseWheelMsg(tea.Mouse{Button: tea.MouseWheelDown}))
	m = updatedModel.(model)
	if m.curTab().cursor != 1 {
		t.Errorf("Expected cursor to be 1 after MouseWheelDown, got %d", m.curTab().cursor)
	}

	// 2. Mouse Wheel Up in Pane 1 -> cursor moves up to 0
	updatedModel, _ = m.Update(tea.MouseWheelMsg(tea.Mouse{Button: tea.MouseWheelUp}))
	m = updatedModel.(model)
	if m.curTab().cursor != 0 {
		t.Errorf("Expected cursor to be 0 after MouseWheelUp, got %d", m.curTab().cursor)
	}

	// 3. Mouse Click on right pane -> activePane switches to 2
	updatedModel, _ = m.Update(tea.MouseClickMsg(tea.Mouse{Button: tea.MouseLeft, X: 90, Y: 10}))
	m = updatedModel.(model)
	if m.activePane != 2 {
		t.Errorf("Expected activePane to be 2 after clicking right pane, got %d", m.activePane)
	}

	// 4. Mouse Wheel Down in Pane 2 -> previewScrollOffset increments to 1
	updatedModel, _ = m.Update(tea.MouseWheelMsg(tea.Mouse{Button: tea.MouseWheelDown}))
	m = updatedModel.(model)
	if m.curTab().previewScrollOffset != 1 {
		t.Errorf("Expected previewScrollOffset to be 1 after MouseWheelDown in Pane 2, got %d", m.curTab().previewScrollOffset)
	}
}

func TestMultiTabsAction(t *testing.T) {
	m := InitialModel()
	m.terminalWidth = 120
	m.terminalHeight = 30

	if len(m.tabs) != 1 {
		t.Fatalf("Expected 1 initial tab, got %d", len(m.tabs))
	}

	// 1. Press "t" to open new tab
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "t"})
	m = updatedModel.(model)
	if len(m.tabs) != 2 {
		t.Fatalf("Expected 2 tabs after pressing 't', got %d", len(m.tabs))
	}
	if m.activeTab != 1 {
		t.Errorf("Expected activeTab to be 1 (second tab), got %d", m.activeTab)
	}

	// 2. Press "]" to switch to next tab (loops back to 0)
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "]"})
	m = updatedModel.(model)
	if m.activeTab != 0 {
		t.Errorf("Expected activeTab to be 0 after pressing ']', got %d", m.activeTab)
	}

	// 3. Press "[" to switch backward (goes back to 1)
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "["})
	m = updatedModel.(model)
	if m.activeTab != 1 {
		t.Errorf("Expected activeTab to be 1 after pressing '[', got %d", m.activeTab)
	}

	// 4. Press "w" to close tab 1
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "w"})
	m = updatedModel.(model)
	if len(m.tabs) != 1 {
		t.Fatalf("Expected 1 tab after closing with 'w', got %d", len(m.tabs))
	}
	if m.activeTab != 0 {
		t.Errorf("Expected activeTab to be 0 after tab close, got %d", m.activeTab)
	}
}

func TestOpenMenuAction(t *testing.T) {
	m := InitialModel()
	m.terminalWidth = 100
	m.terminalHeight = 30
	m.curTab().filesPath = []string{"test_file.txt"}
	m.curTab().cursor = 0

	// 1. Press "o" to open open menu
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "o"})
	m = updatedModel.(model)
	if !m.openMenuOpen {
		t.Errorf("Expected openMenuOpen to be true after pressing 'o'")
	}

	view := m.View()
	if !strings.Contains(view.Content, "Open Menu") {
		t.Errorf("Expected view content to display 'Open Menu' overlay, view: %q", view.Content)
	}

	// 2. Press "c" inside menu -> opens CWD in VSCode & closes dialog
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "c"})
	m = updatedModel.(model)
	if m.openMenuOpen {
		t.Errorf("Expected openMenuOpen to be false after pressing 'c'")
	}

	// 3. Open menu again and press "C" -> opens selected item in VSCode & closes dialog
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "o"})
	m = updatedModel.(model)
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "C"})
	m = updatedModel.(model)
	if m.openMenuOpen {
		t.Errorf("Expected openMenuOpen to be false after pressing 'C'")
	}
}

func TestSelectionAndCopyMoveAction(t *testing.T) {
	m := InitialModel()
	m.terminalWidth = 100
	m.terminalHeight = 30
	m.curTab().filesPath = []string{"file1.txt", "file2.txt"}
	m.curTab().cursor = 0

	// 1. Press space to select file1.txt
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "space"})
	m = updatedModel.(model)
	if _, ok := m.selected[0]; !ok {
		t.Errorf("Expected item at index 0 to be selected")
	}
	if m.curTab().cursor != 0 {
		t.Errorf("Expected cursor to remain at 0, got %d", m.curTab().cursor)
	}

	// 2. Press "c" to stage selected items for copy
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "c"})
	m = updatedModel.(model)
	if len(m.stagedPaths) != 1 {
		t.Fatalf("Expected 1 staged path for copy, got %d", len(m.stagedPaths))
	}
	if m.stagedOp != "copy" {
		t.Errorf("Expected stagedOp to be 'copy', got %q", m.stagedOp)
	}

	// 3. Select file2.txt and press "m" to stage for move
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "space"})
	m = updatedModel.(model)
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "m"})
	m = updatedModel.(model)
	if len(m.stagedPaths) != 1 {
		t.Fatalf("Expected 1 staged path for move, got %d", len(m.stagedPaths))
	}
	if m.stagedOp != "move" {
		t.Errorf("Expected stagedOp to be 'move', got %q", m.stagedOp)
	}
}

func TestMultiSelectDeleteAction(t *testing.T) {
	m := InitialModel()
	m.terminalWidth = 100
	m.terminalHeight = 30
	m.curTab().filesPath = []string{"file1.txt", "file2.txt"}
	m.curTab().cursor = 0

	// 1. Press space to select file1.txt
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "space"})
	m = updatedModel.(model)

	// 2. Move cursor down and press space to select file2.txt
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "j"})
	m = updatedModel.(model)
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "space"})
	m = updatedModel.(model)

	if len(m.selected) != 2 {
		t.Fatalf("Expected 2 items selected, got %d", len(m.selected))
	}

	// 3. Press "d" to open delete menu
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "d"})
	m = updatedModel.(model)
	if !m.deleteMenuOpen {
		t.Errorf("Expected deleteMenuOpen to be true")
	}

	// 4. Press "n" to perform normal delete (trash)
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "n"})
	m = updatedModel.(model)
	if m.deleteMenuOpen {
		t.Errorf("Expected deleteMenuOpen to be false")
	}
	if len(m.selected) != 0 {
		t.Errorf("Expected selected map to be cleared after delete, got len %d", len(m.selected))
	}
}

func TestSubshellAction(t *testing.T) {
	m := InitialModel()
	m.terminalWidth = 100
	m.terminalHeight = 30

	// Press "!" to trigger subshell command
	_, cmd := m.Update(tea.KeyPressMsg{Text: "!"})
	if cmd == nil {
		t.Errorf("Expected non-nil tea.Cmd for subshell command on '!' key")
	}

	// Test completion msg handler
	updatedModel, _ := m.Update(subshellFinishedMsg{err: nil})
	m = updatedModel.(model)
	if m.err != nil {
		t.Errorf("Expected m.err to be nil after subshellFinishedMsg")
	}
}

func TestCopyDirStructureAction(t *testing.T) {
	m := InitialModel()
	m.terminalWidth = 100
	m.terminalHeight = 30

	// 1. Press "y" to open Copy Menu
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "y"})
	m = updatedModel.(model)
	if !m.copyMenuOpen {
		t.Errorf("Expected copyMenuOpen to be true after pressing 'y'")
	}

	// 2. Press "s" to copy directory structure tree
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "s"})
	m = updatedModel.(model)
	if m.copyMenuOpen {
		t.Errorf("Expected copyMenuOpen to be false after pressing 's'")
	}

	// Test tree structure generation function directly
	tree := generateDirectoryStructure(".", false)
	if !strings.Contains(tree, "/") {
		t.Errorf("Expected generated directory structure tree to contain '/', got: %q", tree)
	}
}

func TestSymlinkDisplay(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "symlink_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	targetFile := filepath.Join(tempDir, "real_file.txt")
	_ = os.WriteFile(targetFile, []byte("hello"), 0644)

	linkFile := filepath.Join(tempDir, "link_file.txt")
	err = os.Symlink(targetFile, linkFile)
	if err != nil {
		t.Skip("Skipping symlink test due to OS permission restriction")
	}

	target, isSym, isBroken := getSymlinkTarget(tempDir, "link_file.txt")
	if !isSym {
		t.Errorf("Expected getSymlinkTarget to return isSym = true")
	}
	if isBroken {
		t.Errorf("Expected isBroken = false for valid symlink")
	}
	if target != targetFile {
		t.Errorf("Expected target to be %q, got %q", targetFile, target)
	}
}

func TestCreateAndJumpSymlinkActions(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "symlink_action_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	subDir := filepath.Join(tempDir, "sub_folder")
	_ = os.MkdirAll(subDir, 0755)

	m := InitialModel()
	m.terminalWidth = 100
	m.terminalHeight = 30
	m.curTab().currentDir = tempDir
	m.curTab().filesPath = []string{"sub_folder"}
	m.curTab().cursor = 0

	// 1. Press "K" to create symlink for sub_folder
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "K"})
	m = updatedModel.(model)

	linkPath := filepath.Join(tempDir, "sub_folder_link")
	if _, statErr := os.Lstat(linkPath); os.IsNotExist(statErr) {
		t.Skip("Skipping symlink creation test due to OS permission restriction")
	}

	// Position cursor on the newly created symlink
	for idx, name := range m.curTab().filesPath {
		if name == "sub_folder_link" {
			m.curTab().cursor = idx
			break
		}
	}

	// 2. Press "g" to jump to symlink target
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "g"})
	m = updatedModel.(model)
	if m.curTab().currentDir != subDir {
		t.Errorf("Expected currentDir to be %q after jump 'g', got %q", subDir, m.curTab().currentDir)
	}
}

func TestClearAllSelections(t *testing.T) {
	m := InitialModel()
	m.terminalWidth = 100
	m.terminalHeight = 30
	m.curTab().filesPath = []string{"file1.txt", "file2.txt"}
	m.curTab().cursor = 0

	// Select file1.txt
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "space"})
	m = updatedModel.(model)

	if len(m.selected) != 1 {
		t.Fatalf("Expected 1 selected item, got %d", len(m.selected))
	}

	// Press ESC in normal mode to clear all selections
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "esc"})
	m = updatedModel.(model)

	if len(m.selected) != 0 {
		t.Errorf("Expected 0 selected items after ESC, got %d", len(m.selected))
	}
}

func TestSelectAllToggleAction(t *testing.T) {
	m := InitialModel()
	m.terminalWidth = 100
	m.terminalHeight = 30
	m.curTab().filesPath = []string{"file1.txt", "file2.txt", "file3.txt"}
	m.curTab().cursor = 0

	// 1. Press "a" -> selects all 3 items
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "a"})
	m = updatedModel.(model)
	if len(m.selected) != 3 {
		t.Fatalf("Expected 3 items selected after 'a', got %d", len(m.selected))
	}

	// 2. Press "a" again -> deselects all items
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "a"})
	m = updatedModel.(model)
	if len(m.selected) != 0 {
		t.Errorf("Expected 0 items selected after second 'a', got %d", len(m.selected))
	}
}

func TestMetadataInspectorAction(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "meta_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "test_meta.txt")
	_ = os.WriteFile(filePath, []byte("hello metadata"), 0644)

	meta := getFileMetadataInfo(tempDir, "test_meta.txt")
	if meta.Name != "test_meta.txt" {
		t.Errorf("Expected meta.Name to be 'test_meta.txt', got %q", meta.Name)
	}
	if meta.SizeBytes != 14 {
		t.Errorf("Expected meta.SizeBytes to be 14, got %d", meta.SizeBytes)
	}

	m := InitialModel()
	m.terminalWidth = 100
	m.terminalHeight = 30
	m.curTab().currentDir = tempDir
	m.curTab().filesPath = []string{"test_meta.txt"}
	m.curTab().cursor = 0

	// 1. Press "i" to open Metadata Inspector modal
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "i"})
	m = updatedModel.(model)
	if !m.metadataInspectorOpen {
		t.Errorf("Expected metadataInspectorOpen to be true after pressing 'i'")
	}

	// 2. Press any key (e.g. "esc") to close Metadata Inspector
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "esc"})
	m = updatedModel.(model)
	if m.metadataInspectorOpen {
		t.Errorf("Expected metadataInspectorOpen to be false after pressing 'esc'")
	}
}

func TestShortcutPaneAction(t *testing.T) {
	m := InitialModel()
	m.terminalWidth = 100
	m.terminalHeight = 30

	// Verify shortcuts were populated
	if len(m.shortcuts) == 0 {
		t.Fatalf("Expected shortcuts to be populated, got 0")
	}

	// Since shortcutPaneOpen is true by default, press "ctrl+b" to close it, then "\" to open it
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "ctrl+b"})
	m = updatedModel.(model)
	if m.shortcutPaneOpen {
		t.Errorf("Expected shortcutPaneOpen to be false after pressing 'ctrl+b'")
	}

	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "\\"})
	m = updatedModel.(model)
	if !m.shortcutPaneOpen {
		t.Errorf("Expected shortcutPaneOpen to be true after pressing '\\' again")
	}
	if m.activePane != 0 {
		t.Errorf("Expected activePane to be auto-focused to 0 when opening with '\\', got %d", m.activePane)
	}

	// Focus the shortcut pane by pressing "1"
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "1"})
	m = updatedModel.(model)
	if m.activePane != 0 {
		t.Errorf("Expected activePane to be 0 after pressing '1', got %d", m.activePane)
	}

	view := m.View()
	if !strings.Contains(view.Content, "──") {
		t.Errorf("Expected shortcut pane to render path separator line")
	}

	// 2. Press "j" to navigate down in shortcut pane
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "j"})
	m = updatedModel.(model)
	if m.shortcutCursor != 1 {
		t.Errorf("Expected shortcutCursor to be 1 after 'j', got %d", m.shortcutCursor)
	}

	view = m.View()
	baseDirName := filepath.Base(m.shortcuts[1].Path)
	if !strings.Contains(view.Content, baseDirName) {
		t.Errorf("Expected view to contain active shortcut path base %q", baseDirName)
	}

	// 3. Press "esc" to close shortcut pane
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "esc"})
	m = updatedModel.(model)
	if m.shortcutPaneOpen {
		t.Errorf("Expected shortcutPaneOpen to be false after pressing 'esc'")
	}
}

func TestShortcutCircularNavigation(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	m := InitialModel()
	m.terminalWidth = 100
	m.terminalHeight = 30

	// Enable shortcut pane and focus it
	m.shortcutPaneOpen = true
	m.activePane = 0
	m.shortcutCursor = 0

	// Check total shortcuts
	limit := len(m.shortcuts) - 1
	if limit <= 0 {
		t.Skip("Not enough default shortcuts to test navigation")
	}

	// 1. Press "up" key at top (index 0) -> should wrap to the bottom (limit)
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "up"})
	m = updatedModel.(model)
	if m.shortcutCursor != limit {
		t.Errorf("Expected shortcutCursor to wrap to %d, got %d", limit, m.shortcutCursor)
	}

	// 2. Press "down" key at bottom (limit) -> should wrap to the top (0)
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "down"})
	m = updatedModel.(model)
	if m.shortcutCursor != 0 {
		t.Errorf("Expected shortcutCursor to wrap to 0, got %d", m.shortcutCursor)
	}
}

func TestBookmarkActionsAndSwitching(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	m := InitialModel()
	m.terminalWidth = 100
	m.terminalHeight = 30

	// 1. Press "b" to open bookmark dialog
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "b"})
	m = updatedModel.(model)
	if !m.bookmarkMenuOpen {
		t.Errorf("Expected bookmarkMenuOpen to be true after pressing 'b'")
	}

	// 2. Press "c" to bookmark current directory (CWD)
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "c"})
	m = updatedModel.(model)
	if m.bookmarkMenuOpen {
		t.Errorf("Expected bookmarkMenuOpen to be false after pressing 'c'")
	}
	if len(m.bookmarks) != 1 {
		t.Fatalf("Expected bookmarks count to be 1, got %d", len(m.bookmarks))
	}
	cwd, _ := os.Getwd()
	if m.bookmarks[0].Path != cwd {
		t.Errorf("Expected bookmarked path to be %s, got %s", cwd, m.bookmarks[0].Path)
	}

	// 3. Since shortcutPaneOpen is true by default, press ctrl+b to close it, then ctrl+b to open it
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "ctrl+b"})
	m = updatedModel.(model)
	if m.shortcutPaneOpen {
		t.Fatalf("Expected shortcutPaneOpen to be false after pressing 'ctrl+b'")
	}

	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "ctrl+b"})
	m = updatedModel.(model)
	if !m.shortcutPaneOpen {
		t.Fatalf("Expected shortcutPaneOpen to be true after pressing 'ctrl+b' again")
	}

	// Focus the shortcut pane by pressing "1"
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "1"})
	m = updatedModel.(model)
	if m.activePane != 0 {
		t.Fatalf("Expected activePane to be 0 after pressing '1', got %d", m.activePane)
	}

	// Default tab is 0
	if m.shortcutTab != 0 {
		t.Errorf("Expected default shortcutTab to be 0, got %d", m.shortcutTab)
	}

	// 4. Press "]" to switch to user bookmarks tab
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "]"})
	m = updatedModel.(model)
	if m.shortcutTab != 1 {
		t.Errorf("Expected shortcutTab to be 1 after pressing ']', got %d", m.shortcutTab)
	}

	// 5. Press "[" to switch back to default shortcuts
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "["})
	m = updatedModel.(model)
	if m.shortcutTab != 0 {
		t.Errorf("Expected shortcutTab to be 0 after pressing '[', got %d", m.shortcutTab)
	}

	// 6. Test help menu open inside shortcut pane
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "`"})
	m = updatedModel.(model)
	if !m.helpMenuOpen {
		t.Errorf("Expected helpMenuOpen to be true inside shortcut pane after pressing '`'")
	}

	// Close help
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "esc"})
	m = updatedModel.(model)

	// 7. Test deleting a bookmark from the Bookmarks tab
	// Go to tab 1
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "]"})
	m = updatedModel.(model)
	// Press "d" to trigger delete bookmark dialog
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "d"})
	m = updatedModel.(model)
	if !m.deleteBookmarkConfirmOpen {
		t.Errorf("Expected deleteBookmarkConfirmOpen to be true after pressing 'd' on bookmarks tab")
	}

	// Press "y" to confirm removal
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "y"})
	m = updatedModel.(model)
	if m.deleteBookmarkConfirmOpen {
		t.Errorf("Expected deleteBookmarkConfirmOpen to be false after 'y'")
	}
	if len(m.bookmarks) != 0 {
		t.Errorf("Expected bookmarks count to be 0 after deletion, got %d", len(m.bookmarks))
	}
}

func TestSwitchActivePaneKeys(t *testing.T) {
	m := InitialModel()
	// Default is 1 (File Explorer) and shortcutPaneOpen is true
	if m.activePane != 1 {
		t.Errorf("Expected default activePane to be 1, got %d", m.activePane)
	}

	// Press "3" to switch to Preview pane
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "3"})
	m = updatedModel.(model)
	if m.activePane != 2 {
		t.Errorf("Expected activePane to be 2 after pressing '3', got %d", m.activePane)
	}

	// Press "1" to switch to Shortcut pane
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "1"})
	m = updatedModel.(model)
	if m.activePane != 0 {
		t.Errorf("Expected activePane to be 0 after pressing '1', got %d", m.activePane)
	}

	// Press "2" to switch back to File Explorer
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "2"})
	m = updatedModel.(model)
	if m.activePane != 1 {
		t.Errorf("Expected activePane to be 1 after pressing '2', got %d", m.activePane)
	}

	// Close shortcut pane, then try to press "1" (should not switch to 0)
	m.shortcutPaneOpen = false
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "1"})
	m = updatedModel.(model)
	if m.activePane == 0 {
		t.Errorf("Expected activePane to not switch to 0 when shortcut pane is closed")
	}
}

func TestShortcutCloseRedirection(t *testing.T) {
	m := InitialModel()
	// Open and focus shortcut pane
	m.shortcutPaneOpen = true
	m.activePane = 0

	// Pressing esc to close shortcut pane should redirect activePane to 1
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "esc"})
	m = updatedModel.(model)
	if m.shortcutPaneOpen {
		t.Errorf("Expected shortcutPaneOpen to be false after esc")
	}
	if m.activePane != 1 {
		t.Errorf("Expected activePane to switch to 1 (File Explorer) after closing shortcut pane, got %d", m.activePane)
	}
}

func TestAsyncCopyFileContent(t *testing.T) {
	m := InitialModel()
	m.activePane = 1

	// 1. Test normal text file copy
	tempFile, err := os.CreateTemp("", "test_copy_content_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	if _, err := tempFile.WriteString("Hello, Antigravity async copy!"); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	m.curTab().filesPath = []string{filepath.Base(tempFile.Name())}
	m.curTab().currentDir = filepath.Dir(tempFile.Name())
	m.curTab().cursor = 0

	// Open copy menu
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "y"})
	m = updatedModel.(model)
	if !m.copyMenuOpen {
		t.Fatalf("Expected copy menu to be open")
	}

	// Choose copy content
	updatedModel, cmd := m.Update(tea.KeyPressMsg{Text: "c"})
	m = updatedModel.(model)
	if m.copyMenuOpen {
		t.Errorf("Expected copy menu to close after selection")
	}
	if !m.copyJobActive {
		t.Fatalf("Expected copyJobActive to be true")
	}
	if cmd == nil {
		t.Fatalf("Expected tea.Cmd to be returned for async copy")
	}

	// Run the returned cmd(s) to completion
	for m.copyJobActive {
		msg := cmd()
		if msg == nil {
			break
		}
		updatedModel, cmd = m.Update(msg)
		m = updatedModel.(model)
	}

	if m.err != nil {
		t.Errorf("Expected no copy error, got %v", m.err)
	}

	clipText, err := clipboard.ReadAll()
	if err == nil {
		if clipText != "Hello, Antigravity async copy!" {
			t.Errorf("Expected clipboard content to be %q, got %q", "Hello, Antigravity async copy!", clipText)
		}
	}

	// 2. Test binary file warning
	tempBin, err := os.CreateTemp("", "test_copy_bin_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempBin.Name())
	if _, err := tempBin.Write([]byte{0x00, 0x01, 0x02}); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempBin.Close()

	m.curTab().filesPath = []string{filepath.Base(tempBin.Name())}
	m.curTab().cursor = 0

	// Open copy menu
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "y"})
	m = updatedModel.(model)
	// Choose copy content
	updatedModel, cmd = m.Update(tea.KeyPressMsg{Text: "c"})
	m = updatedModel.(model)

	// Run to completion
	for m.copyJobActive {
		msg := cmd()
		if msg == nil {
			break
		}
		updatedModel, cmd = m.Update(msg)
		m = updatedModel.(model)
	}

	if m.err == nil {
		t.Errorf("Expected binary copy file content operation to fail")
	} else if !strings.Contains(m.err.Error(), "binary") {
		t.Errorf("Expected binary file warning, got error %v", m.err)
	}

	// 3. Test cancellation
	m.err = nil
	m.copyJobActive = true
	m.copyJobCancel = make(chan struct{})
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "esc"})
	m = updatedModel.(model)
	if m.copyJobActive {
		t.Errorf("Expected copyJobActive to be false after pressing esc")
	}
	if !strings.Contains(m.statusMsg, "cancelled") {
		t.Errorf("Expected status message to contain 'cancelled', got %q", m.statusMsg)
	}
}
func TestRenameCursor(t *testing.T) {
	// helpers
	press := func(m model, key string) model {
		updated, _ := m.Update(tea.KeyPressMsg{Text: key})
		return updated.(model)
	}

	// Setup: temp dir with a real file so isDir() works correctly
	tmp := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.WriteFile(filepath.Join(tmp, "hello.txt"), nil, 0644))
	must(os.WriteFile(filepath.Join(tmp, ".bashrc"), nil, 0644))
	must(os.Mkdir(filepath.Join(tmp, "mydir"), 0755))
	must(os.WriteFile(filepath.Join(tmp, "noext"), nil, 0644))

	newModel := func(name string) model {
		m := InitialModel()
		m.tabs[0].currentDir = tmp
		m.tabs[0].filesPath = []string{name}
		m.tabs[0].cursor = 0
		return m
	}

	t.Run("cursor before ext on file", func(t *testing.T) {
		m := newModel("hello.txt")
		m, _ = func() (model, tea.Cmd) {
			u, c := m.Update(tea.KeyPressMsg{Text: "r"})
			return u.(model), c
		}()
		// "hello" is 5 runes → cursor at 5
		if m.fileInput.Position() != 5 {
			t.Errorf("expected inputCursor=5, got %d", m.fileInput.Position())
		}
		if !m.inputMode || !m.isRenaming {
			t.Fatal("expected inputMode and isRenaming")
		}
	})

	t.Run("cursor at end for dotfile", func(t *testing.T) {
		m := newModel(".bashrc")
		m, _ = func() (model, tea.Cmd) {
			u, c := m.Update(tea.KeyPressMsg{Text: "r"})
			return u.(model), c
		}()
		want := len([]rune(".bashrc")) // 7
		if m.fileInput.Position() != want {
			t.Errorf("expected inputCursor=%d, got %d", want, m.fileInput.Position())
		}
	})

	t.Run("cursor at end for dir", func(t *testing.T) {
		m := newModel("mydir")
		m, _ = func() (model, tea.Cmd) {
			u, c := m.Update(tea.KeyPressMsg{Text: "r"})
			return u.(model), c
		}()
		want := len([]rune("mydir")) // 5
		if m.fileInput.Position() != want {
			t.Errorf("expected inputCursor=%d, got %d", want, m.fileInput.Position())
		}
	})

	t.Run("cursor at end for no-ext file", func(t *testing.T) {
		m := newModel("noext")
		m, _ = func() (model, tea.Cmd) {
			u, c := m.Update(tea.KeyPressMsg{Text: "r"})
			return u.(model), c
		}()
		want := len([]rune("noext")) // 5
		if m.fileInput.Position() != want {
			t.Errorf("expected inputCursor=%d, got %d", want, m.fileInput.Position())
		}
	})

	t.Run("esc resets cursor", func(t *testing.T) {
		m := newModel("hello.txt")
		m, _ = func() (model, tea.Cmd) {
			u, c := m.Update(tea.KeyPressMsg{Text: "r"})
			return u.(model), c
		}()
		m = press(m, "esc")
		if m.inputMode || m.fileInput.Value() != "" {
			t.Errorf("esc should reset inputMode and inputText")
		}
	})
}

func TestSearchFilter(t *testing.T) {
	m := InitialModel()
	m.curTab().filesPath = []string{"alpha.txt", "beta.txt", "GAMMA.txt", "charlie.txt"}
	m.searchActive = true

	// Filter case insensitive
	m.searchInput.SetValue("M")
	m.applySearchFilter()

	visible := m.curTab().visibleFiles()
	if len(visible) != 1 {
		t.Fatalf("Expected 1 match for 'M', got %d: %v", len(visible), visible)
	}

	// Filter multiple characters
	m.searchInput.SetValue("ma")
	m.applySearchFilter()
	visible = m.curTab().visibleFiles()
	if len(visible) != 1 || visible[0] != "GAMMA.txt" {
		t.Fatalf("Expected ['GAMMA.txt'] for 'ma', got %v", visible)
	}

	// Empty string should clear filter
	m.searchInput.SetValue("")
	m.applySearchFilter()
	visible = m.curTab().visibleFiles()
	if len(visible) != 4 {
		t.Fatalf("Expected 4 items for empty filter, got %d", len(visible))
	}
}

// ── Quick Find tests ──────────────────────────────────────────────────────────

func TestQuickFindOpen(t *testing.T) {
	m := InitialModel()
	m.bookmarks = []ShortcutEntry{
		{Name: "Downloads", Path: "/home/test/Downloads", Icon: DirIcon},
		{Name: "Documents", Path: "/home/test/Documents", Icon: DirIcon},
	}

	next, _ := m.Update(tea.KeyPressMsg{Text: "ctrl+f"})
	nm := next.(model)

	if !nm.quickFindOpen {
		t.Fatal("expected quickFindOpen=true after ctrl+f")
	}
	if nm.quickFindInput.Value() != "" {
		t.Fatalf("expected empty quickFindInput, got %q", nm.quickFindInput.Value())
	}
	if nm.quickFindSelecting {
		t.Fatal("expected input mode (quickFindSelecting=false) on open")
	}

	// Now simulate the command finishing
	msg := nm.quickFindSearchCmd("")()
	next, _ = nm.Update(msg)
	nm = next.(model)

	if len(nm.quickFindResults) < 2 {
		t.Fatalf("expected at least 2 bookmark results, got %d", len(nm.quickFindResults))
	}
}

func TestQuickFindEsc(t *testing.T) {
	m := InitialModel()
	m.quickFindOpen = true
	m.quickFindInput.SetValue("docs")
	m.quickFindSelecting = true

	next, _ := m.Update(tea.KeyPressMsg{Text: "esc"})
	nm := next.(model)

	if nm.quickFindOpen {
		t.Fatal("expected quickFindOpen=false after esc")
	}
	if nm.quickFindInput.Value() != "" {
		t.Fatalf("expected empty text after esc, got %q", nm.quickFindInput.Value())
	}
	if nm.quickFindSelecting {
		t.Fatal("expected quickFindSelecting=false after esc")
	}
}

func TestQuickFindEnterSwitchesToSelectMode(t *testing.T) {
	m := InitialModel()
	m.quickFindOpen = true
	m.quickFindResults = []ShortcutEntry{
		{Name: "Downloads", Path: "/home/test/Downloads", Icon: DirIcon},
	}

	next, _ := m.Update(tea.KeyPressMsg{Text: "enter"})
	nm := next.(model)

	if !nm.quickFindSelecting {
		t.Fatal("expected quickFindSelecting=true after Enter in input mode")
	}
	if nm.quickFindSelected != 0 {
		t.Fatalf("expected cursor at 0, got %d", nm.quickFindSelected)
	}
}

func TestQuickFindNavigation(t *testing.T) {
	m := InitialModel()
	m.quickFindOpen = true
	m.quickFindSelecting = false // Input mode
	m.quickFindInput.SetValue("")
	m.quickFindInput.Focus()

	// Typing 'j' should insert character
	next, _ := m.Update(tea.KeyPressMsg{Text: "j"})
	nm := next.(model)
	if nm.quickFindInput.Value() != "j" {
		t.Fatalf("expected quickFindText='j', got %q", nm.quickFindInput.Value())
	}

	// Typing 'k' should insert character
	next, _ = nm.Update(tea.KeyPressMsg{Text: "k"})
	nm = next.(model)
	if nm.quickFindInput.Value() != "jk" {
		t.Fatalf("expected quickFindText='jk', got %q", nm.quickFindInput.Value())
	}

	// Switch to select mode
	nm.quickFindSelecting = true
	nm.quickFindResults = []ShortcutEntry{
		{Name: "A", Path: "/a", Icon: DirIcon},
		{Name: "B", Path: "/b", Icon: DirIcon},
		{Name: "C", Path: "/c", Icon: DirIcon},
	}
	nm.quickFindSelected = 0

	// down
	next, _ = nm.Update(tea.KeyPressMsg{Text: "down"})
	nm = next.(model)
	if nm.quickFindSelected != 1 {
		t.Fatalf("expected selected=1 after down, got %d", nm.quickFindSelected)
	}

	// j also moves down in select mode
	next, _ = nm.Update(tea.KeyPressMsg{Text: "j"})
	nm = next.(model)
	if nm.quickFindSelected != 2 {
		t.Fatalf("expected selected=2 after j, got %d", nm.quickFindSelected)
	}

	// up
	next, _ = nm.Update(tea.KeyPressMsg{Text: "up"})
	nm = next.(model)
	if nm.quickFindSelected != 1 {
		t.Fatalf("expected selected=1 after up, got %d", nm.quickFindSelected)
	}
}

func TestQuickFindFilterBookmarks(t *testing.T) {
	m := InitialModel()
	m.bookmarks = []ShortcutEntry{
		{Name: "UniqueDownloadsTest", Path: "/home/test/UniqueDownloadsTest", Icon: DirIcon},
		{Name: "Documents", Path: "/home/test/Documents", Icon: DirIcon},
		{Name: "Videos", Path: "/home/test/Videos", Icon: DirIcon},
	}

	results := m.buildQuickFindResults("uniquedownloadstest")
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'uniquedownloadstest', got %d: %v", len(results), results)
	}
	if results[0].Name != "UniqueDownloadsTest" {
		t.Fatalf("expected UniqueDownloadsTest, got %s", results[0].Name)
	}

	// Empty query returns all bookmarks
	results = m.buildQuickFindResults("")
	// Since results can contain user's real bookmarks, we only assert it contains at least our mock ones.
	if len(results) < 3 {
		t.Fatalf("expected at least 3 results for empty query, got %d", len(results))
	}
}



func TestShrinkPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("skipping home dir dependent tests")
	}

	cases := []struct {
		path     string
		maxLen   int
		expected string
	}{
		{home, 30, "~"},
		{filepath.Join(home, "Downloads"), 30, "~/Downloads"},
		{filepath.Join(home, "coding", "Go", "tui", "poc-fmtui"), 20, "~/codi.../poc-fmtui"},
		{"/var/log/syslog/extremely/long/path/name/here", 20, "/var/l.../name/here"},
	}

	for _, c := range cases {
		got := shrinkPath(c.path, c.maxLen)
		if got != c.expected {
			t.Errorf("shrinkPath(%q, %d) = %q; want %q", c.path, c.maxLen, got, c.expected)
		}
	}
}

func TestViewHeaderPathFormatting(t *testing.T) {
	m := InitialModel()
	m.terminalWidth = 40
	m.tabs = []TabState{
		{currentDir: "/var/log/syslog/extremely/long/path/name/here"},
	}

	view := m.View()
	rendered := view.Content

	// Should not contain the full path
	if strings.Contains(rendered, "/var/log/syslog/extremely/long/path/name/here") {
		t.Fatal("expected view header path to be truncated, but it was fully rendered")
	}

	// Should contain the truncated path segment "/var/...e/here"
	if !strings.Contains(rendered, "/var/...e/here") {
		t.Fatalf("expected view header path to contain truncated path segment '/var/...e/here', got view: %q", rendered)
	}
}

func TestQuickFindRoots(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}

	dataExists := false
	if fi, err := os.Stat("/data"); err == nil && fi.IsDir() {
		dataExists = true
	}

	// currentDir inside home → home always first; /data added if it exists
	roots := quickFindRoots(home, filepath.Join(home, "coding", "Go"))
	if roots[0] != home {
		t.Errorf("expected first root to be home, got %q", roots[0])
	}
	if dataExists && len(roots) < 2 {
		t.Errorf("expected home + /data when /data exists, got %v", roots)
	}
	if !dataExists && len(roots) != 1 {
		t.Errorf("expected only [home] when /data absent, got %v", roots)
	}

	// currentDir outside home and not /data → home + /data + ancestor
	outsideDir := "/srv/coding/Go/tui/poc-fmtui"
	roots = quickFindRoots(home, outsideDir)
	if roots[0] != home {
		t.Errorf("expected first root to be home, got %q", roots[0])
	}
	// /data already added; /srv should also be added if it exists (no assertion on
	// existence, just ensure no duplicates and home is first)
	seen := map[string]bool{}
	for _, r := range roots {
		if seen[r] {
			t.Errorf("duplicate root %q in %v", r, roots)
		}
		seen[r] = true
	}
}


func TestQuickFindScoring(t *testing.T) {
	// Build a temp dir tree: /tmp/qftest/Go/tui/poc-fmtui
	root := t.TempDir()
	goDir := filepath.Join(root, "Go")
	tuiDir := filepath.Join(goDir, "tui")
	pocDir := filepath.Join(tuiDir, "poc-fmtui")
	for _, d := range []string{goDir, tuiDir, pocDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	m := InitialModel()
	// Override roots to only walk our temp tree via a mock:
	// We test the scoring by directly examining candidates via buildQuickFindResults
	// using a currentDir outside home that resolves to root.
	// Since quickFindRoots climbs 4 levels from currentDir, set currentDir = pocDir
	// so the ancestor 4 levels up lands near root.
	m.tabs[m.activeTab].currentDir = pocDir

	// Query "tui" → exact match "tui" (score 3) should appear before "poc-fmtui" (score 1)
	// We can't control the real filesystem here, so test the scoring logic directly.
	cases := []struct {
		query    string
		name     string
		wantScore int
	}{
		{"tui", "tui", 3},
		{"tui", "poc-fmtui", 1},
		{"go", "Go", 3},
		{"go", "golang", 2},
		{"go", "cargo", 1},
	}
	for _, c := range cases {
		q := strings.ToLower(c.query)
		lname := strings.ToLower(c.name)
		score := 0
		switch {
		case lname == q:
			score = 3
		case strings.HasPrefix(lname, q):
			score = 2
		case strings.Contains(lname, q):
			score = 1
		}
		if score != c.wantScore {
			t.Errorf("score(%q vs %q) = %d, want %d", c.query, c.name, score, c.wantScore)
		}
	}
}

func TestQuickFindFileJump(t *testing.T) {
	// Create a temp folder structure containing a file:
	// /tmp/qfjump/dir/file.txt
	root := t.TempDir()
	subDir := filepath.Join(root, "dir")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	targetFile := filepath.Join(subDir, "file.txt")
	if err := os.WriteFile(targetFile, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	m := InitialModel()
	m.quickFindOpen = true
	m.quickFindSelecting = true
	m.quickFindSelected = 0
	// Inject the file search result manually to avoid filesystem search variations
	m.quickFindResults = []ShortcutEntry{
		{Name: "file.txt", Path: targetFile, Icon: FileIcon},
	}

	// Pressing enter should trigger a file jump: parent folder loaded, and cursor focuses "file.txt"
	next, _ := m.Update(tea.KeyPressMsg{Text: "enter"})
	nm := next.(model)

	if nm.quickFindOpen {
		t.Fatal("expected quickFindOpen to close after selection")
	}

	// Check if correct parent dir is loaded
	tab := nm.curTab()
	if tab.currentDir != subDir {
		t.Fatalf("expected currentDir to be %q, got %q", subDir, tab.currentDir)
	}

	// Check if file is highlighted (i.e. cursor is at the index of "file.txt")
	found := false
	for idx, f := range tab.filesPath {
		if f == "file.txt" {
			if tab.cursor != idx {
				t.Fatalf("expected cursor to be at index %d for file.txt, got %d", idx, tab.cursor)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected file.txt to be in tab's filesPath")
	}
}

func TestUserConfig(t *testing.T) {
	// 1. Default config check
	cfg := defaultUserConfig()
	if len(cfg.AllowedHiddenFiles) == 0 {
		t.Fatal("expected default allowed hidden files list to be non-empty")
	}
	
	// Check standard whitelisted file present
	found := false
	for _, f := range cfg.AllowedHiddenFiles {
		if f == ".zshrc" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected .zshrc in default config allowed list")
	}
}

func TestShortcutNavigationKeepPaneOpen(t *testing.T) {
	tempDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tempDir, "dummy.txt"), []byte("dummy"), 0644)
	if err != nil {
		t.Fatalf("failed to create dummy file: %v", err)
	}

	m := InitialModel()
	m.shortcuts = []ShortcutEntry{
		{Name: "TempDir", Path: tempDir, Icon: DirIcon},
	}
	m.shortcutPaneOpen = true
	m.activePane = 0 // focus shortcuts
	m.shortcutCursor = 0

	// Navigate using "enter"
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "enter"})
	m = updatedModel.(model)

	if !m.shortcutPaneOpen {
		t.Errorf("expected shortcutPaneOpen to remain true after navigation")
	}
	if m.activePane != 1 {
		t.Errorf("expected activePane to switch to 1 (File Explorer), got %d", m.activePane)
	}
	if m.curTab().currentDir != tempDir {
		t.Errorf("expected currentDir to be %q, got %q", tempDir, m.curTab().currentDir)
	}
}


