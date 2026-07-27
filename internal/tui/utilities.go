package tui

import (
	"fmt"
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

func writeTempDirFile(dir string) error {
	return os.WriteFile(tempDirFile, []byte(dir), 0644)
}

func clearTempDirFile() error {
	return os.Remove(tempDirFile)
}

func getSymlinkTarget(dir, name string) (target string, isSym bool, isBroken bool) {
	absP := filepath.Join(dir, name)
	info, err := os.Lstat(absP)
	if err != nil {
		return "", false, false
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(absP)
		if err != nil {
			return "", true, true
		}
		resolvedPath := target
		if !filepath.IsAbs(resolvedPath) {
			resolvedPath = filepath.Join(dir, target)
		}
		if _, statErr := os.Stat(resolvedPath); os.IsNotExist(statErr) {
			return target, true, true
		}
		return target, true, false
	}
	return "", false, false
}

func (m *model) CheckDirFilesExt(list string) (string, string, string) {
	var fgColor string
	var bgColor string
	var icon string

	t := m.curTab()
	if _, isSym, isBroken := getSymlinkTarget(t.currentDir, list); isSym {
		icon = SymlinkIcon
		if isBroken {
			fgColor = Red
			bgColor = BgRed
		} else {
			fgColor = Cyan
			bgColor = BgCyan
		}
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
		} else if list == ".gitignore" {
			icon = GitIgnoreIcon
			fgColor = Gray
			bgColor = GrayHighlight
		} else if list == "Dockerfile" || list == "docker-compose.yml" || list == "docker-compose.yaml" {
			icon = DockerFileIcon
			fgColor = Blue
			bgColor = BgBlue
		} else if ext == ".ts" || ext == ".tsx" {
			icon = TsIcon
			fgColor = Blue
			bgColor = BgBlue
		} else if ext == ".pt" || ext == ".pth" {
			icon = PtIcon
			fgColor = Red
			bgColor = BgRed
		} else if ext == ".java" || ext == ".jar" {
			icon = JavaIcon
			fgColor = Red
			bgColor = BgRed
		} else if ext == ".css" || ext == ".scss" || ext == ".sass" || ext == ".less" {
			icon = CssIcon
			fgColor = Blue
			bgColor = BgBlue
		} else if ext == ".html" || ext == ".htm" {
			icon = HtmlIcon
			fgColor = Red
			bgColor = BgRed
		} else if ext == ".vue" {
			icon = VueIcon
			fgColor = Green
			bgColor = BgGreen
		} else if ext == ".db" || ext == ".sqlite" || ext == ".sqlite3" || ext == ".sql" {
			icon = SqliteIcon
			fgColor = Yellow
			bgColor = BgYellow
		} else if ext == ".txt" || ext == ".log" || ext == ".env" {
			icon = TextIcon
			fgColor = Gray
			bgColor = GrayHighlight
		} else if ext == ".xls" || ext == ".xlsx" {
			icon = XlsIcon
			fgColor = Green
			bgColor = BgGreen
		} else if ext == ".doc" || ext == ".docx" {
			icon = DocsIcon
			fgColor = Blue
			bgColor = BgBlue
		} else if ext == ".csv" {
			icon = CsvIcon
			fgColor = Green
			bgColor = BgGreen
		} else if ext == ".ppt" || ext == ".pptx" {
			icon = PptxIcon
			fgColor = Red
			bgColor = BgRed
		} else if ext == ".zip" || ext == ".tar" || ext == ".gz" || ext == ".rar" || ext == ".7z" || ext == ".bz2" || ext == ".xz" {
			icon = CompressedIcon
			fgColor = Yellow
			bgColor = BgYellow
		} else if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".bmp" || ext == ".svg" || ext == ".webp" || ext == ".ico" {
			icon = ImageIcon
			fgColor = Magenta
			bgColor = BgMagenta
		} else if ext == ".mp4" || ext == ".avi" || ext == ".mkv" || ext == ".mov" || ext == ".webm" {
			icon = VdoIcon
			fgColor = Magenta
			bgColor = BgMagenta
		} else if ext == ".mp3" || ext == ".wav" || ext == ".flac" || ext == ".ogg" || ext == ".aac" || ext == ".m4a" {
			icon = AudioIcon
			fgColor = Cyan
			bgColor = BgCyan
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
	for i := range n {
		if buf[i] == 0 {
			return []string{"Binary File [No Preview]"}
		}
	}

	content := string(buf[:n])
	lines := strings.Split(content, "\n")
	return lines
}

type FileMetaInfo struct {
	Name        string
	AbsPath     string
	SizeStr     string
	SizeBytes   int64
	PermStr     string
	OwnerGroup  string
	ModTimeStr  string
	MimeOrType  string
	IsDirectory bool
	IsSymlink   bool
	SymTarget   string
}

func getFileMetadataInfo(dir string, name string) FileMetaInfo {
	meta := FileMetaInfo{
		Name:    name,
		AbsPath: findAbsPath(dir, name),
	}
	if name == "" {
		return meta
	}

	absP := meta.AbsPath
	lInfo, err := os.Lstat(absP)
	if err != nil {
		return meta
	}

	if lInfo.Mode()&os.ModeSymlink != 0 {
		meta.IsSymlink = true
		if target, err := os.Readlink(absP); err == nil {
			meta.SymTarget = target
		}
	}

	info, statErr := os.Stat(absP)
	if statErr != nil {
		info = lInfo
	}

	meta.IsDirectory = info.IsDir()
	meta.SizeBytes = info.Size()
	meta.PermStr = fmt.Sprintf("%s (%o)", info.Mode().String(), info.Mode().Perm())
	meta.ModTimeStr = info.ModTime().Format("2006-01-02 15:04:02")

	if info.IsDir() {
		entries, err := os.ReadDir(absP)
		if err == nil {
			meta.SizeStr = fmt.Sprintf("%d items", len(entries))
		} else {
			meta.SizeStr = "directory"
		}
		meta.MimeOrType = "Directory"
	} else {
		meta.SizeStr = formatFileSize(info.Size())
		ext := filepath.Ext(name)
		if ext != "" {
			meta.MimeOrType = strings.TrimPrefix(ext, ".") + " file"
		} else {
			meta.MimeOrType = "File"
		}
	}

	meta.OwnerGroup = getOwnerGroupStr()
	return meta
}

func formatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024.0)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024.0*1024.0))
	}
	return fmt.Sprintf("%.1f GB", float64(size)/(1024.0*1024.0*1024.0))
}

func getOwnerGroupStr() string {
	u := os.Getenv("USER")
	if u == "" {
		u = os.Getenv("USERNAME")
	}
	if u == "" {
		u = "user"
	}
	return u
}

// ponytail: Helper to construct clean top border lines with titles and optional pane numbers without breaking ANSI escape codes or alignment
func createTopBorder(title string, width int, color string, paneNum string) string {
	if width < 2 {
		return ""
	}
	barWidth := width - 2

	var prefix string
	if paneNum != "" {
		prefix = paneNum + "─"
	}
	prefixLen := len([]rune(prefix))
	if barWidth < prefixLen {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("╭" + strings.Repeat("─", barWidth) + "╮")
	}

	barWidth -= prefixLen
	titleRunes := []rune(title)
	if len(titleRunes) > barWidth {
		titleRunes = titleRunes[:barWidth]
	}
	startIdx := (barWidth - len(titleRunes)) / 2
	endIdx := barWidth - startIdx - len(titleRunes)

	topStr := "╭" + prefix + strings.Repeat("─", startIdx) + string(titleRunes) + strings.Repeat("─", endIdx) + "╮"
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(topStr)
}

// ponytail: Helper to truncate paths from the left (showing the most specific subfolders) if they exceed maximum width
func truncatePathLeft(path string, maxWidth int) string {
	if len(path) <= maxWidth {
		return path
	}
	if maxWidth <= 3 {
		return "..."
	}
	return "..." + path[len(path)-(maxWidth-3):]
}
