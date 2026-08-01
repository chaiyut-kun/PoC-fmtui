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

type fileStyle struct {
	icon    string
	fgColor func(...any) string
	bgColor func(...any) string
}

var exactFileStyles = map[string]fileStyle{
	"go.mod":              {GoMasCot, Cyan, BgCyan},
	"go.sum":              {GoMasCot, Cyan, BgCyan},
	"go.work":             {GoMasCot, Cyan, BgCyan},
	"Makefile":            {MakeIcon, Yellow, BgYellow},
	".gitignore":          {GitIgnoreIcon, Gray, Plain},
	"Dockerfile":          {DockerFileIcon, Blue, BgBlue},
	"docker-compose.yml":  {DockerFileIcon, Blue, BgBlue},
	"docker-compose.yaml": {DockerFileIcon, Blue, BgBlue},
}

var extStyles = map[string]fileStyle{
	".go":      {GoExt, Cyan, BgCyan},
	".js":      {JsIcon, Yellow, BgYellow},
	".py":      {PyIcon, Cyan, BgCyan},
	".sh":      {BashIcon, Green, BgGreen},
	".bash":    {BashIcon, Green, BgGreen},
	".zsh":     {BashIcon, Green, BgGreen},
	".md":      {MdIcon, Magenta, BgYellow},
	".pdf":     {PdfIcon, Red, BgRed},
	".json":    {JsonIcon, Yellow, BgYellow},
	".ts":      {TsIcon, Blue, BgBlue},
	".tsx":     {TsIcon, Blue, BgBlue},
	".pt":      {PtIcon, Red, BgRed},
	".pth":     {PtIcon, Red, BgRed},
	".java":    {JavaIcon, Red, BgRed},
	".jar":     {JavaIcon, Red, BgRed},
	".css":     {CssIcon, Blue, BgBlue},
	".scss":    {CssIcon, Blue, BgBlue},
	".sass":    {CssIcon, Blue, BgBlue},
	".less":    {CssIcon, Blue, BgBlue},
	".html":    {HtmlIcon, Red, BgRed},
	".htm":     {HtmlIcon, Red, BgRed},
	".vue":     {VueIcon, Green, BgGreen},
	".db":      {SqliteIcon, Yellow, BgYellow},
	".sqlite":  {SqliteIcon, Yellow, BgYellow},
	".sqlite3": {SqliteIcon, Yellow, BgYellow},
	".sql":     {SqliteIcon, Yellow, BgYellow},
	".txt":     {TextIcon, Gray, Plain},
	".log":     {TextIcon, Gray, Plain},
	".env":     {TextIcon, Gray, Plain},
	".xls":     {XlsIcon, Green, BgGreen},
	".xlsx":    {XlsIcon, Green, BgGreen},
	".doc":     {DocsIcon, Blue, BgBlue},
	".docx":    {DocsIcon, Blue, BgBlue},
	".csv":     {CsvIcon, Green, BgGreen},
	".ppt":     {PptxIcon, Red, BgRed},
	".pptx":    {PptxIcon, Red, BgRed},
	".zip":     {CompressedIcon, Yellow, BgYellow},
	".tar":     {CompressedIcon, Yellow, BgYellow},
	".gz":      {CompressedIcon, Yellow, BgYellow},
	".rar":     {CompressedIcon, Yellow, BgYellow},
	".7z":      {CompressedIcon, Yellow, BgYellow},
	".bz2":     {CompressedIcon, Yellow, BgYellow},
	".xz":      {CompressedIcon, Yellow, BgYellow},
	".png":     {ImageIcon, Magenta, BgMagenta},
	".jpg":     {ImageIcon, Magenta, BgMagenta},
	".jpeg":    {ImageIcon, Magenta, BgMagenta},
	".gif":     {ImageIcon, Magenta, BgMagenta},
	".bmp":     {ImageIcon, Magenta, BgMagenta},
	".svg":     {ImageIcon, Magenta, BgMagenta},
	".webp":    {ImageIcon, Magenta, BgMagenta},
	".ico":     {ImageIcon, Magenta, BgMagenta},
	".mp4":     {VdoIcon, Magenta, BgMagenta},
	".avi":     {VdoIcon, Magenta, BgMagenta},
	".mkv":     {VdoIcon, Magenta, BgMagenta},
	".mov":     {VdoIcon, Magenta, BgMagenta},
	".webm":    {VdoIcon, Magenta, BgMagenta},
	".mp3":     {AudioIcon, Cyan, BgCyan},
	".wav":     {AudioIcon, Cyan, BgCyan},
	".flac":    {AudioIcon, Cyan, BgCyan},
	".ogg":     {AudioIcon, Cyan, BgCyan},
	".aac":     {AudioIcon, Cyan, BgCyan},
	".m4a":     {AudioIcon, Cyan, BgCyan},
}

func (m *model) CheckDirFilesExt(list string) (string, func(...any) string, func(...any) string) {
	t := m.curTab()
	if _, isSym, isBroken := getSymlinkTarget(t.currentDir, list); isSym {
		if isBroken {
			return SymlinkIcon, Red, BgRed
		}
		return SymlinkIcon, Cyan, BgCyan
	}

	if isDir(t.currentDir, list) {
		if list == ".git" {
			return GitIcon, White, BgWhite
		}
		return DirIcon, Blue, BgBlue
	}

	if style, ok := exactFileStyles[list]; ok {
		return style.icon, style.fgColor, style.bgColor
	}

	ext := filepath.Ext(list)
	if style, ok := extStyles[ext]; ok {
		return style.icon, style.fgColor, style.bgColor
	}

	return FileIcon, Plain, Plain
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

	// Compensate for extra skipped visual width to prevent right border shifting
	if skippedWidth > fgVisibleWidth {
		for k := 0; k < skippedWidth-fgVisibleWidth; k++ {
			result.WriteRune(' ')
		}
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
