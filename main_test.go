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
	m.filesPath = []string{"file1.txt", "file2.txt"}
	
	// Default active pane should be 0 (Left Explorer)
	if m.activePane != 0 {
		t.Errorf("Expected activePane to be 0 by default, got %d", m.activePane)
	}

	// Pressing "[" should switch to activePane 1 (Right Previewer)
	updatedModel, _ := m.Update(tea.KeyPressMsg{Text: "["})
	m = updatedModel.(model)
	if m.activePane != 1 {
		t.Errorf("Expected activePane to be 1 after pressing '[', got %d", m.activePane)
	}

	// Pressing "]" should switch back to activePane 0
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "]"})
	m = updatedModel.(model)
	if m.activePane != 0 {
		t.Errorf("Expected activePane to switch back to 0 after pressing ']', got %d", m.activePane)
	}

	// In activePane 1, pressing down/j should increment previewScrollOffset
	m.activePane = 1
	m.previewScrollOffset = 0
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "j"})
	m = updatedModel.(model)
	if m.previewScrollOffset != 1 {
		t.Errorf("Expected previewScrollOffset to be 1 after pressing 'j' in right pane, got %d", m.previewScrollOffset)
	}

	// In activePane 0, pressing down/j should move file cursor instead of scrolling preview
	m.activePane = 0
	m.cursor = 0
	m.previewScrollOffset = 5
	updatedModel, _ = m.Update(tea.KeyPressMsg{Text: "j"})
	m = updatedModel.(model)
	if m.cursor != 1 {
		t.Errorf("Expected cursor to increment to 1, got %d", m.cursor)
	}
	if m.previewScrollOffset != 0 {
		t.Errorf("Expected previewScrollOffset to reset to 0 after moving cursor in left pane, got %d", m.previewScrollOffset)
	}
}

func TestPreviewLineTruncation(t *testing.T) {
	m := InitialModel()
	m.terminalWidth = 40
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
	
	m.filesPath = []string{filepath.Base(tempFile.Name())}
	m.currentDir = filepath.Dir(tempFile.Name())
	m.cursor = 0
	
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

