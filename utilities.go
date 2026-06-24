package main

import (
	"os"
	"path/filepath"

	"github.com/atotto/clipboard"
)

// copy to clipboard
func copyToClipboard(text string) error {
	return clipboard.WriteAll(text)
}

func findAbsPath(dir string, fileName string) string {

	dir, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}

	if fileName == "" {
		return dir
	}

	return filepath.Join(dir, fileName)
}

func isDir(dir, name string) bool {
	info, err := os.Stat(filepath.Join(dir, name))

	if err != nil {
		return false
	}
	return info.IsDir()
}
