package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

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
	
	// Default active pane should be 0 (Left Explorer)
	if m.activePane != 0 {
		t.Errorf("Expected activePane to be 0 by default, got %d", m.activePane)
	}

	// Pressing "tab" should switch to activePane 1 (Right Previewer)
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "tab"})
	m = updatedModel.(model)
	if m.activePane != 1 {
		t.Errorf("Expected activePane to be 1 after pressing 'tab', got %d", m.activePane)
	}

	// Pressing "tab" again should switch back to activePane 0
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "tab"})
	m = updatedModel.(model)
	if m.activePane != 0 {
		t.Errorf("Expected activePane to switch back to 0 after pressing 'tab', got %d", m.activePane)
	}

	// In activePane 1, pressing down/j should increment previewScrollOffset
	m.activePane = 1
	m.curTab().previewScrollOffset = 0
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "j"})
	m = updatedModel.(model)
	if m.curTab().previewScrollOffset != 1 {
		t.Errorf("Expected previewScrollOffset to be 1 after pressing 'j' in right pane, got %d", m.curTab().previewScrollOffset)
	}

	// In activePane 0, pressing down/j should move file cursor instead of scrolling preview
	m.activePane = 0
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
	m.inputText = "new_name.txt"

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
	m.inputText = "created_file.txt"
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
	m.activePane = 0

	// 1. Mouse Wheel Down in Pane 0 -> cursor moves down to 1
	updatedModel, _ := m.Update(tea.MouseWheelMsg(tea.Mouse{Button: tea.MouseWheelDown}))
	m = updatedModel.(model)
	if m.curTab().cursor != 1 {
		t.Errorf("Expected cursor to be 1 after MouseWheelDown, got %d", m.curTab().cursor)
	}

	// 2. Mouse Wheel Up in Pane 0 -> cursor moves up to 0
	updatedModel, _ = m.Update(tea.MouseWheelMsg(tea.Mouse{Button: tea.MouseWheelUp}))
	m = updatedModel.(model)
	if m.curTab().cursor != 0 {
		t.Errorf("Expected cursor to be 0 after MouseWheelUp, got %d", m.curTab().cursor)
	}

	// 3. Mouse Click on right pane -> activePane switches to 1
	updatedModel, _ = m.Update(tea.MouseClickMsg(tea.Mouse{Button: tea.MouseLeft, X: 80, Y: 10}))
	m = updatedModel.(model)
	if m.activePane != 1 {
		t.Errorf("Expected activePane to be 1 after clicking right pane, got %d", m.activePane)
	}

	// 4. Mouse Wheel Down in Pane 1 -> previewScrollOffset increments to 1
	updatedModel, _ = m.Update(tea.MouseWheelMsg(tea.Mouse{Button: tea.MouseWheelDown}))
	m = updatedModel.(model)
	if m.curTab().previewScrollOffset != 1 {
		t.Errorf("Expected previewScrollOffset to be 1 after MouseWheelDown in Pane 1, got %d", m.curTab().previewScrollOffset)
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
	if m.curTab().cursor != 1 {
		t.Errorf("Expected cursor to advance to 1, got %d", m.curTab().cursor)
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
	// 2. Press space to select file2.txt
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

