package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ponytail: loadBookmarks retrieves user bookmarked directories from ~/.config/fmtui/bookmarks.json
func loadBookmarks() []ShortcutEntry {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	configPath := filepath.Join(home, ".config", "fmtui", "bookmarks.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}
	var bookmarks []ShortcutEntry
	if err := json.Unmarshal(data, &bookmarks); err != nil {
		return nil
	}
	return bookmarks
}

// ponytail: saveBookmarks writes user bookmarked directories to ~/.config/fmtui/bookmarks.json
func saveBookmarks(bookmarks []ShortcutEntry) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dirPath := filepath.Join(home, ".config", "fmtui")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}
	configPath := filepath.Join(dirPath, "bookmarks.json")
	data, err := json.MarshalIndent(bookmarks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}
