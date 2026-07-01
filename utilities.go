package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	m.scrollOffset = 0
	m.previewScrollOffset = 0
	m.selected = make(map[int]struct{})
}

func writeTempDirFile(dir string) error {
	return os.WriteFile(tempDirFile, []byte(dir), 0644)
}

func clearTempDirFile() error {
	return os.Remove(tempDirFile)
}

func (m *model) CheckDirFilesExt(list string) (string, string, string) {

	var fgColor string
	var bgColor string
	var icon string

	if isDir(m.currentDir, list) {

		if list == ".git" {
			icon = GitIcon
			fgColor = White
			bgColor = BgWhite
		} else {
			icon = DirIcon
			fgColor = Blue
			bgColor = BgBlue
		}

	} else {
		ext := filepath.Ext(list)
		if ext == ".go" {
			icon = GoExt
			fgColor = Cyan
			bgColor = BgCyan
		} else if ext == ".js" {
			icon = JsIcon
			fgColor = Yellow
			bgColor = BgYellow
		} else if ext == ".py" {
			icon = PyIcon
			fgColor = Cyan
			bgColor = BgCyan
		} else if ext == ".sh" || ext == ".bash" || ext == ".zsh" {
			icon = BashIcon
			fgColor = Green
			bgColor = BgGreen
		} else if strings.Contains(list, "go") {
			icon = GoMasCot
			fgColor = Cyan
			bgColor = BgCyan
		} else if ext == ".md" {
			icon = MdIcon
			fgColor = Magenta
			bgColor = BgYellow
		} else if ext == ".pdf" {
			icon = PdfIcon
			fgColor = Red
			bgColor = BgRed
		} else if ext == ".json" {
			icon = JsonIcon
			fgColor = Yellow
			bgColor = BgYellow
		} else if list == "Makefile" {
			icon = MakeIcon
			fgColor = Yellow
			bgColor = BgYellow
		} else {
			icon = FileIcon
			fgColor = Reset
			bgColor = GrayHighlight
		}
	}
	return icon, fgColor, bgColor
}

func overlayLine(bg string, fg string, startCol int) string {
	fgVisibleWidth := visibleWidth(fg)

	var result strings.Builder
	bgRunes := []rune(bg)

	visibleCol := 0
	i := 0

	for i < len(bgRunes) {
		if bgRunes[i] == '\033' {
			result.WriteRune(bgRunes[i])
			i++
			for i < len(bgRunes) && bgRunes[i-1] != 'm' {
				result.WriteRune(bgRunes[i])
				i++
			}
			continue
		}

		if visibleCol == startCol {
			break
		}

		result.WriteRune(bgRunes[i])
		visibleCol++
		i++
	}

	for visibleCol < startCol {
		result.WriteRune(' ')
		visibleCol++
	}

	result.WriteString(fg)

	skippedWidth := 0
	for i < len(bgRunes) && skippedWidth < fgVisibleWidth {
		if bgRunes[i] == '\033' {
			result.WriteRune(bgRunes[i])
			i++
			for i < len(bgRunes) && bgRunes[i-1] != 'm' {
				result.WriteRune(bgRunes[i])
				i++
			}
			continue
		}
		i++
		skippedWidth++
	}

	for i < len(bgRunes) {
		result.WriteRune(bgRunes[i])
		i++
	}

	return result.String()
}

func visibleWidth(s string) int {
	w := 0
	runes := []rune(s)
	i := 0
	for i < len(runes) {
		if runes[i] == '\033' {
			i++
			for i < len(runes) && runes[i-1] != 'm' {
				i++
			}
			continue
		}
		w++
		i++
	}
	return w
}

func getPreviewLines(path string, isFolder bool, showHidden bool) []string {
	if isFolder {
		entries, err := os.ReadDir(path)
		if err != nil {
			return []string{fmt.Sprintf("Error reading directory: %v", err)}
		}
		var lines []string
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".") && !showHidden {
				continue
			}
			icon := FileIcon
			if entry.IsDir() {
				icon = DirIcon
			}
			lines = append(lines, icon+" "+entry.Name())
		}
		if len(lines) == 0 {
			return []string{"[Empty directory]"}
		}
		return lines
	}

	// File preview
	file, err := os.Open(path)
	if err != nil {
		return []string{fmt.Sprintf("Error opening file: %v", err)}
	}
	defer file.Close()

	// Read first 64KB for preview capability
	buf := make([]byte, 65536)
	n, err := file.Read(buf)
	if err != nil && n == 0 {
		return []string{"[Empty file]"}
	}

	// Binary check
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return []string{"Binary File [No Preview]"}
		}
	}

	content := string(buf[:n])
	lines := strings.Split(content, "\n")
	return lines
}
