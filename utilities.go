package main

import (
	"os"
	"path/filepath"

	"github.com/atotto/clipboard"
)

const tempDirFile = "/tmp/fmtui-dir"
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

func (m *model) reLoadDir(changeDir string, entries []os.DirEntry) {
	m.currentDir = changeDir
	m.filesPath = m.filesPath[:0]
	for _, entry := range entries {
		m.filesPath = append(m.filesPath, entry.Name())
	}

	m.cursor = 0
	m.selected = make(map[int]struct{})
}


func writeTempDirFile(dir string) error {
	return os.WriteFile(tempDirFile, []byte(dir), 0644)
}

func clearTempDirFile() error {
	return os.Remove(tempDirFile)
}
