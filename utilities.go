package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/lipgloss"
)

var tempDirFile = filepath.Join(os.TempDir(), "fmtui-dir")

func getCurrentUser() string {
	u := os.Getenv("USER")
	if u != "" {
		return u
	}
	if usr, err := user.Current(); err == nil && usr.Username != "" {
		return usr.Username
	}
	return "user"
}

func openWithVSCode(targetPath string) error {
	cmd := exec.Command("code", targetPath)
	return cmd.Start()
}

func revealInFileExplorer(targetPath string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-R", targetPath)
	case "windows":
		cmd = exec.Command("explorer.exe", "/select,", targetPath)
	default: // linux, etc.
		dir := targetPath
		if fi, err := os.Stat(targetPath); err == nil && !fi.IsDir() {
			dir = filepath.Dir(targetPath)
		}
		cmd = exec.Command("xdg-open", dir)
	}
	return cmd.Start()
}

// copy to clipboard
func copyToClipboard(text string) error {
	return clipboard.WriteAll(text)
}

func generateDirectoryStructure(root string, showHidden bool) string {
	var sb strings.Builder
	rootDirName := filepath.Base(root)
	if rootDirName == "" || rootDirName == "." || rootDirName == "/" {
		rootDirName = root
	}
	sb.WriteString(rootDirName + "/\n")

	var walk func(currentDir string, prefix string, depth int)
	walk = func(currentDir string, prefix string, depth int) {
		if depth > 5 {
			return
		}
		entries, err := os.ReadDir(currentDir)
		if err != nil {
			return
		}

		var filtered []os.DirEntry
		for _, e := range entries {
			if !showHidden && strings.HasPrefix(e.Name(), ".") {
				continue
			}
			filtered = append(filtered, e)
		}

		for i, e := range filtered {
			isLast := i == len(filtered)-1
			connector := "├── "
			childPrefix := "│   "
			if isLast {
				connector = "└── "
				childPrefix = "    "
			}

			if e.IsDir() {
				sb.WriteString(prefix + connector + e.Name() + "/\n")
				walk(filepath.Join(currentDir, e.Name()), prefix+childPrefix, depth+1)
			} else {
				sb.WriteString(prefix + connector + e.Name() + "\n")
			}
		}
	}

	walk(root, "", 1)
	return sb.String()
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
	t := m.curTab()
	t.currentDir = changeDir
	t.filesPath = t.filesPath[:0]
	for _, entry := range entries {
		if !m.showHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		t.filesPath = append(t.filesPath, entry.Name())
	}

	t.cursor = 0
	t.scrollOffset = 0
	t.previewScrollOffset = 0
	m.selected = make(map[int]struct{})
}

func (m *model) reloadCurrentDir() {
	t := m.curTab()
	entries, err := os.ReadDir(t.currentDir)
	if err != nil {
		m.err = err
		return
	}
	m.reLoadDir(t.currentDir, entries)
}

func writeTempDirFile(dir string) error {
	return os.WriteFile(tempDirFile, []byte(dir), 0644)
}

func clearTempDirFile() error {
	return os.Remove(tempDirFile)
}

func getSymlinkTarget(dir, name string) (string, bool) {
	absP := filepath.Join(dir, name)
	info, err := os.Lstat(absP)
	if err != nil {
		return "", false
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(absP)
		if err == nil {
			return target, true
		}
		return "", true
	}
	return "", false
}

func (m *model) CheckDirFilesExt(list string) (string, string, string) {

	var fgColor string
	var bgColor string
	var icon string

	t := m.curTab()
	if _, isSym := getSymlinkTarget(t.currentDir, list); isSym {
		icon = SymlinkIcon
		fgColor = Cyan
		bgColor = BgCyan
		return icon, fgColor, bgColor
	}

	if isDir(t.currentDir, list) {

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
		} else if list == "go.mod" || list == "go.sum" || list == "go.work" {
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
	fgVisibleWidth := lipgloss.Width(fg)

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

		if visibleCol >= startCol {
			break
		}

		rw := lipgloss.Width(string(bgRunes[i]))
		result.WriteRune(bgRunes[i])
		visibleCol += rw
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
		rw := lipgloss.Width(string(bgRunes[i]))
		skippedWidth += rw
		i++
	}

	for i < len(bgRunes) {
		result.WriteRune(bgRunes[i])
		i++
	}

	return result.String()
}

func visibleWidth(s string) int {
	return lipgloss.Width(s)
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

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}

	info, err := os.Stat(src)
	if err == nil {
		_ = os.Chmod(dst, info.Mode())
	}
	return nil
}

func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFileOrDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

func moveFileOrDir(src, dst string) error {
	// Try atomic rename first
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	info, statErr := os.Stat(src)
	if statErr != nil {
		return statErr
	}

	if info.IsDir() {
		if copyErr := copyDir(src, dst); copyErr != nil {
			return copyErr
		}
		return os.RemoveAll(src)
	}

	if copyErr := copyFile(src, dst); copyErr != nil {
		return copyErr
	}
	return os.Remove(src)
}

func permanentDelete(path string) error {
	return os.RemoveAll(path)
}

func normalDelete(path string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	trashDir := filepath.Join(homeDir, ".local", "share", "Trash", "files")
	if err := os.MkdirAll(trashDir, 0755); err != nil {
		return "", err
	}

	baseName := filepath.Base(path)
	destination := filepath.Join(trashDir, baseName)
	counter := 1
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	for {
		if _, statErr := os.Stat(destination); os.IsNotExist(statErr) {
			break
		}
		destination = filepath.Join(trashDir, fmt.Sprintf("%s_%d%s", nameWithoutExt, counter, ext))
		counter++
	}

	if err := moveFileOrDir(path, destination); err != nil {
		return "", err
	}

	return destination, nil
}

// ponytail: stdlib os.Stat & os.Chmod to toggle executable mode directly
func toggleExecutable(path string) (oldMode, newMode os.FileMode, err error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, 0, err
	}
	oldMode = info.Mode()
	if oldMode&0111 != 0 {
		newMode = oldMode &^ 0111
	} else {
		newMode = oldMode | 0111
	}
	if err := os.Chmod(path, newMode); err != nil {
		return 0, 0, err
	}
	return oldMode, newMode, nil
}

func (m *model) pushAction(action ActionRecord) {
	m.undoStack = append(m.undoStack, action)
	m.redoStack = nil
}

func (m *model) performUndo() error {
	if len(m.undoStack) == 0 {
		return fmt.Errorf("Nothing to undo")
	}

	lastIdx := len(m.undoStack) - 1
	action := m.undoStack[lastIdx]
	m.undoStack = m.undoStack[:lastIdx]

	switch action.Type {
	case ActionDelete:
		if action.TrashPath == "" {
			return fmt.Errorf("Cannot undo permanent delete")
		}
		if err := moveFileOrDir(action.TrashPath, action.OriginalPath); err != nil {
			return err
		}
	case ActionCreate:
		trashPath, err := normalDelete(action.OriginalPath)
		if err != nil {
			return err
		}
		action.TrashPath = trashPath
	case ActionRename:
		if err := moveFileOrDir(action.NewPath, action.OldPath); err != nil {
			return err
		}
	case ActionChmod:
		if err := os.Chmod(action.OriginalPath, action.OldMode); err != nil {
			return err
		}
	}

	m.redoStack = append(m.redoStack, action)
	m.reloadCurrentDir()
	return nil
}

func (m *model) performRedo() error {
	if len(m.redoStack) == 0 {
		return fmt.Errorf("Nothing to redo")
	}

	lastIdx := len(m.redoStack) - 1
	action := m.redoStack[lastIdx]
	m.redoStack = m.redoStack[:lastIdx]

	switch action.Type {
	case ActionDelete:
		if err := moveFileOrDir(action.OriginalPath, action.TrashPath); err != nil {
			return err
		}
	case ActionCreate:
		if action.TrashPath != "" {
			if err := moveFileOrDir(action.TrashPath, action.OriginalPath); err != nil {
				return err
			}
		} else {
			if action.IsDir {
				if err := os.MkdirAll(action.OriginalPath, 0755); err != nil {
					return err
				}
			} else {
				file, err := os.Create(action.OriginalPath)
				if err != nil {
					return err
				}
				file.Close()
			}
		}
	case ActionRename:
		if err := moveFileOrDir(action.OldPath, action.NewPath); err != nil {
			return err
		}
	case ActionChmod:
		if err := os.Chmod(action.OriginalPath, action.NewMode); err != nil {
			return err
		}
	}

	m.undoStack = append(m.undoStack, action)
	m.reloadCurrentDir()
	return nil
}
