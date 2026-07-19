package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
)

func (m model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

	t := m.curTab()
	renderedText := fmt.Sprintf(" %s%s: %s%s\n", Cyan, getCurrentUser(), t.currentDir, Reset)

	// Search bar with border
	searchIcon := ""
	
	sWidth := m.terminalWidth
	if sWidth < 20 {
		sWidth = 80
	}
	sShortcutWidth := 0
	if m.shortcutPaneOpen {
		sShortcutWidth = sWidth / 4
		if sShortcutWidth < 14 {
			sShortcutWidth = 14
		}
	}
	var sLeftWidth int
	if m.shortcutPaneOpen {
		sLeftWidth = sWidth / 2
		if sLeftWidth < 10 {
			sLeftWidth = 10
		}
	} else {
		sAvailableWidth := sWidth - sShortcutWidth
		sLeftWidth = (sAvailableWidth * 2) / 3
		if sLeftWidth < 10 {
			sLeftWidth = sAvailableWidth / 2
		}
	}
	
	borderColor := "#555555" // Gray
	if m.searchActive {
		borderColor = "#5555FF" // Blue
	}
	
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(0, 1).
		Width(sLeftWidth - 6) // fit inside the left pane
		
	var boxContent string
	if m.searchActive {
		runes := []rune(m.searchText)
		cur := m.searchCursor
		if cur > len(runes) {
			cur = len(runes)
		}
		before := string(runes[:cur])
		after := string(runes[cur:])
		boxContent = fmt.Sprintf("%s %s█%s", searchIcon, before, after)
	} else if m.searchText != "" {
		boxContent = fmt.Sprintf("%s %s", searchIcon, m.searchText)
	} else {
		boxContent = fmt.Sprintf("%s%s press s to search for file or directory%s", Gray, searchIcon, Reset)
	}
	
	renderedText += boxStyle.Render(boxContent) + "\n"

	maxVisible := m.getMaxVisibleFiles()

	files := t.visibleFiles()
	endIdx := t.scrollOffset + maxVisible
	if endIdx > len(files) {
		endIdx = len(files)
	}

	for i := t.scrollOffset; i < endIdx; i++ {
		list := files[i]

		displayName := list
		if target, isSym, isBroken := getSymlinkTarget(t.currentDir, list); isSym {
			if isBroken {
				displayName = fmt.Sprintf("%s -> %s [Broken]", list, target)
			} else {
				displayName = fmt.Sprintf("%s -> %s", list, target)
			}
		}

		icon, fgColor, bgColor := m.CheckDirFilesExt(list)

		cursorSymbol := " "
		if t.cursor == i {
			cursorSymbol = ">"
		}
		selectMarker := " "
		if _, ok := m.selected[i]; ok {
			selectMarker = fmt.Sprintf("%s*%s", Yellow, Reset)
		}

		if t.cursor == i {
			renderedText += fmt.Sprintf("%s%s %s%s %s %s%s%s\n", bgColor, Black, cursorSymbol, selectMarker, icon, displayName, ItemEndSpace, Reset)
		} else {
			if fgColor == Reset {
				renderedText += fmt.Sprintf(" %s%s %s %s%s\n", cursorSymbol, selectMarker, icon, displayName, ItemEndSpace)
			} else {
				renderedText += fmt.Sprintf(" %s%s %s%s %s%s%s \n", cursorSymbol, selectMarker, fgColor, icon, displayName, ItemEndSpace, Reset)
			}
		}
	}

	// Dynamic padding: push metadata to bottom of the box interior
	numRendered := endIdx - t.scrollOffset
	paddingLines := maxVisible - numRendered
	if paddingLines > 0 {
		renderedText += strings.Repeat("\n", paddingLines)
	}

	// Metadata bar pinned at bottom inside the border
	files2 := t.visibleFiles()
	if len(files2) > 0 && t.cursor >= 0 && t.cursor < len(files2) {
		meta := getFileMetadataInfo(t.currentDir, files2[t.cursor])
		renderedText += fmt.Sprintf("%s─%s %s%s | %s | %s | %s%s\n",
			Gray, Reset, Yellow, meta.SizeStr, meta.PermStr, meta.OwnerGroup, meta.ModTimeStr, Reset)
	}

	if m.inputMode {
		prompt := "Create file"
		if m.isRenaming {
			prompt = "Rename"
		} else if m.isCreatingDir {
			prompt = "Create directory"
		}
		// ponytail: split at cursor rune index, no extra lib
		runes := []rune(m.inputText)
		cur := m.inputCursor
		if cur > len(runes) {
			cur = len(runes)
		}
		before := string(runes[:cur])
		after := string(runes[cur:])
		renderedText += fmt.Sprintf("%s: %s█%s\n(Enter to confirm, Esc to cancel)\n", prompt, before, after)
	}

	// Trim trailing newline so lipgloss box height is calculated precisely
	renderedText = strings.TrimSuffix(renderedText, "\n")

	width := m.terminalWidth
	height := m.terminalHeight
	if width < 20 {
		width = 80
	}
	if height < 5 {
		height = 20
	}

	bottomBarLines := 1
	if m.err != nil {
		bottomBarLines = 2
	}
	boxHeight := height - bottomBarLines
	if boxHeight < 3 {
		boxHeight = 3
	}

	// Calculate shortcut pane width
	shortcutWidth := 0
	if m.shortcutPaneOpen {
		shortcutWidth = width / 4
		if shortcutWidth < 14 {
			shortcutWidth = 14
		}
	}

	var leftWidth, rightWidth int
	if m.shortcutPaneOpen {
		leftWidth = width / 2
		if leftWidth < 10 {
			leftWidth = 10
		}
		rightWidth = width - shortcutWidth - leftWidth
		if rightWidth < 10 {
			rightWidth = 10
		}
	} else {
		availableWidth := width - shortcutWidth
		leftWidth = (availableWidth * 2) / 3
		if leftWidth < 10 {
			leftWidth = availableWidth / 2
		}
		rightWidth = availableWidth - leftWidth
	}

	leftBorderColor := "#555555"
	rightBorderColor := "#555555"
	shortcutBorderColor := "#555555"
	if m.shortcutPaneOpen {
		if m.activePane == 0 {
			shortcutBorderColor = "#5555FF"
		} else {
			shortcutBorderColor = "#FFFF55"
		}
	}
	if m.activePane == 1 {
		leftBorderColor = "#5555FF"
	}
	if m.activePane == 2 {
		rightBorderColor = "#5555FF"
	}

	// 0. Create shortcut pane (if open)
	var shortcutBox string
	if m.shortcutPaneOpen {
		var shortcutText string
		var activeShortcuts []ShortcutEntry
		if m.shortcutTab == 0 {
			activeShortcuts = m.shortcuts
		} else {
			activeShortcuts = m.bookmarks
		}

		if len(activeShortcuts) == 0 {
			shortcutText = " (No bookmarks)\n Press 'b' to add"
		} else {
			for i, s := range activeShortcuts {
				cursor := "  "
				if i == m.shortcutCursor {
					cursor = "> "
				}
				label := fmt.Sprintf("%s%s  %s%s", cursor, s.Icon, s.Name, Reset)
				if i == m.shortcutCursor {
					label = fmt.Sprintf("%s%s%s  %s%s", BgCyan, Black, cursor+s.Icon, s.Name, Reset)
				}
				shortcutText += label + "\n"
			}
			shortcutText = strings.TrimSuffix(shortcutText, "\n")

			// ponytail: Append full path of active shortcut at the bottom of the pane
			if m.shortcutCursor >= 0 && m.shortcutCursor < len(activeShortcuts) {
				selectedPath := activeShortcuts[m.shortcutCursor].Path
				maxWidth := shortcutWidth - 4
				if maxWidth < 5 {
					maxWidth = 5
				}
				truncatedPath := truncatePathLeft(selectedPath, maxWidth)
				sep := strings.Repeat("─", shortcutWidth-2)

				linesCount := strings.Count(shortcutText, "\n") + 1
				targetHeight := boxHeight - 2
				paddingNeeded := targetHeight - linesCount - 2 // -2 for sep and path line

				if paddingNeeded > 0 {
					shortcutText += strings.Repeat("\n", paddingNeeded)
				} else {
					shortcutText += "\n"
				}
				shortcutText += fmt.Sprintf("%s%s%s\n %s%s%s", Gray, sep, Reset, Yellow, truncatedPath, Reset)
			}
		}

		shortcutStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(shortcutBorderColor)).
			Width(shortcutWidth - 2).
			Height(boxHeight - 2)

		shortcutBox = shortcutStyle.Render(shortcutText)

		// Overlay title on border depending on tab and width
		scLines := strings.Split(shortcutBox, "\n")
		if len(scLines) > 0 {
			var title string
			if shortcutWidth-2 >= 27 {
				if m.shortcutTab == 0 {
					title = " [Shortcuts] │ Bookmarks "
				} else {
					title = " Shortcuts │ [Bookmarks] "
				}
			} else {
				if m.shortcutTab == 0 {
					title = " [Shortcuts] "
				} else {
					title = " [Bookmarks] "
				}
			}
			scLines[0] = createTopBorder(title, shortcutWidth, shortcutBorderColor, "1")
			shortcutBox = strings.Join(scLines, "\n")
		}
	}

	// 1. Create left box (File Explorer)
	leftHeight := boxHeight - 2
	if m.copyJobActive {
		leftHeight -= 4
		if leftHeight < 3 {
			leftHeight = 3
		}
	}
	leftStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(leftBorderColor)).
		Width(leftWidth - 2).
		Height(leftHeight)

	leftBox := leftStyle.Render(renderedText)

	// Overlay tabs on left border
	leftLines := strings.Split(leftBox, "\n")
	if len(leftLines) > 0 {
		var tabParts []string
		for i, tb := range m.tabs {
			dirName := filepath.Base(tb.currentDir)
			if dirName == "" || dirName == "." || dirName == "/" {
				dirName = tb.currentDir
			}
			if i == m.activeTab {
				tabParts = append(tabParts, fmt.Sprintf("[%s]", dirName))
			} else {
				tabParts = append(tabParts, dirName)
			}
		}
		rawTitle := " " + strings.Join(tabParts, " | ") + " "
		title := []rune(rawTitle)
		maxTitleLen := leftWidth - 4
		if maxTitleLen > 6 && len(title) > maxTitleLen {
			pathTrimmed := " ..." + string(title[len(title)-(maxTitleLen-4):])
			title = []rune(pathTrimmed)
		}
		leftLines[0] = createTopBorder(string(title), leftWidth, leftBorderColor, "2")
		leftBox = strings.Join(leftLines, "\n")
	}

	if m.copyJobActive {
		progressWidth := leftWidth
		if progressWidth < 14 {
			progressWidth = 14
		}

		barWidth := progressWidth - 25 // subtract space for borders and labels
		if barWidth < 5 {
			barWidth = 5
		}
		filledLen := int(m.copyJobProgress * float64(barWidth))
		if filledLen > barWidth {
			filledLen = barWidth
		}
		emptyLen := barWidth - filledLen
		barStr := strings.Repeat("█", filledLen) + strings.Repeat("░", emptyLen)

		percentStr := fmt.Sprintf("%.1f%%", m.copyJobProgress*100)
		progressContent := fmt.Sprintf("Copying: %s\n[%s] %s (Esc to cancel)", m.copyJobFileName, barStr, percentStr)

		progressStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#55FFFF")). // Cyan border
			Padding(0, 1).
			Width(progressWidth - 2).
			Height(2) // 4 lines total including borders

		progressPane := progressStyle.Render(progressContent)
		leftBox = lipgloss.JoinVertical(lipgloss.Left, leftBox, progressPane)
	}

	// 2. Create right box (Previewer)
	var rightText string
	maxVisiblePreviewLines := boxHeight - 2
	if maxVisiblePreviewLines < 1 {
		maxVisiblePreviewLines = 1
	}

	if len(t.visibleFiles()) > 0 {
		selectedName := t.visibleFiles()[t.cursor]
		targetPath := findAbsPath(t.currentDir, selectedName)
		previewLines := getPreviewLines(targetPath, isDir(t.currentDir, selectedName), m.showHidden)

		// Clamp previewScrollOffset based on number of lines and visible height
		maxScroll := len(previewLines) - maxVisiblePreviewLines
		if maxScroll < 0 {
			maxScroll = 0
		}
		if t.previewScrollOffset > maxScroll {
			t.previewScrollOffset = maxScroll
		}
		if t.previewScrollOffset < 0 {
			t.previewScrollOffset = 0
		}

		endLine := t.previewScrollOffset + maxVisiblePreviewLines
		if endLine > len(previewLines) {
			endLine = len(previewLines)
		}

		maxWidth := rightWidth - 2
		if maxWidth < 1 {
			maxWidth = 1
		}
		var truncatedLines []string
		for _, line := range previewLines[t.previewScrollOffset:endLine] {
			expanded := strings.ReplaceAll(line, "\t", "    ")
			runes := []rune(expanded)
			if len(runes) > maxWidth {
				truncatedLines = append(truncatedLines, string(runes[:maxWidth]))
			} else {
				truncatedLines = append(truncatedLines, expanded)
			}
		}
		rightText = strings.Join(truncatedLines, "\n")
	}
	rightText = strings.TrimSuffix(rightText, "\n")

	rightStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(rightBorderColor)).
		Width(rightWidth - 2).
		Height(boxHeight - 2)

	rightBox := rightStyle.Render(rightText)

	// Overlay " Preview " on right border
	rightLines := strings.Split(rightBox, "\n")
	if len(rightLines) > 0 {
		rightLines[0] = createTopBorder(" Preview ", rightWidth, rightBorderColor, "3")
		rightBox = strings.Join(rightLines, "\n")
	}

	// Join all boxes horizontally
	var content string
	if m.shortcutPaneOpen {
		content = lipgloss.JoinHorizontal(lipgloss.Top, shortcutBox, leftBox, rightBox)
	} else {
		content = lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
	}

	// Bottom Help & Error Status Bar
	// hiddenStatus := "OFF"
	if m.showHidden {
		// hiddenStatus = "ON"
	}
	// helpText := fmt.Sprintf(" Press 'q' to quit | ':' shell | 'space' select | 'a' all | 'c' copy | 'm' move | 'v'/'p' paste | 'g' jump link | 'K' mklink | 'e' exit&cd | '.' hidden:%s | '`' help", hiddenStatus)

	if m.err != nil {
		content += fmt.Sprintf("\n%sError: %v%s", Red, m.err, Reset)
	} else if m.statusMsg != "" {
		content += fmt.Sprintf("\n%s%s%s", Green, m.statusMsg, Reset)
	}
	// content += "\n" + helpText
	content += "\n"
	// Render copy menu if open
	if m.copyMenuOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#55FF55")). // Green border
			Padding(1, 2)

		dialogContent := "         Copy Menu\n\n" +
			"[p] Copy absolute path\n" +
			"[n] Copy name only\n" +
			"[e] Copy name with ext\n" +
			"[s] Copy dir structure (tree)\n" +
			"[c] Copy file content\n\n" +
			"Press ESC to cancel"

		dialog := dialogStyle.Render(dialogContent)

		dialogHeight := lipgloss.Height(dialog)
		dialogWidth := lipgloss.Width(dialog)
		startRow := (height - dialogHeight) / 2
		startCol := (width - dialogWidth) / 2

		bgLines := strings.Split(content, "\n")
		fgLines := strings.Split(dialog, "\n")
		for r, fgLine := range fgLines {
			row := startRow + r
			if row >= 0 && row < len(bgLines) {
				bgLines[row] = overlayLine(bgLines[row], fgLine, startCol)
			}
		}
		content = strings.Join(bgLines, "\n")
	}

	// Render bookmark menu if open
	if m.bookmarkMenuOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#FF55FF")). // Purple border
			Padding(1, 2)

		dialogContent := "    Bookmark Options\n\n" +
			"[s] Bookmark Selected Dir\n" +
			"[c] Bookmark Current CWD\n\n" +
			"Press any other key to close"

		dialog := dialogStyle.Render(dialogContent)

		dialogHeight := lipgloss.Height(dialog)
		dialogWidth := lipgloss.Width(dialog)
		startRow := (height - dialogHeight) / 2
		startCol := (width - dialogWidth) / 2

		bgLines := strings.Split(content, "\n")
		fgLines := strings.Split(dialog, "\n")
		for r, fgLine := range fgLines {
			row := startRow + r
			if row >= 0 && row < len(bgLines) {
				bgLines[row] = overlayLine(bgLines[row], fgLine, startCol)
			}
		}
		content = strings.Join(bgLines, "\n")
	}

	// Render delete bookmark confirmation menu if open
	if m.deleteBookmarkConfirmOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#FF5555")). // Red border
			Padding(1, 2)

		var bookmarkName string
		if m.shortcutCursor >= 0 && m.shortcutCursor < len(m.bookmarks) {
			bookmarkName = m.bookmarks[m.shortcutCursor].Name
		}
		dialogContent := fmt.Sprintf("   Remove Bookmark?\n\nRemove bookmark for %q?\n\n   [y] Yes      [n] No", bookmarkName)
		dialog := dialogStyle.Render(dialogContent)

		dialogHeight := lipgloss.Height(dialog)
		dialogWidth := lipgloss.Width(dialog)
		startRow := (height - dialogHeight) / 2
		startCol := (width - dialogWidth) / 2

		bgLines := strings.Split(content, "\n")
		fgLines := strings.Split(dialog, "\n")
		for r, fgLine := range fgLines {
			row := startRow + r
			if row >= 0 && row < len(bgLines) {
				bgLines[row] = overlayLine(bgLines[row], fgLine, startCol)
			}
		}
		content = strings.Join(bgLines, "\n")
	}

	// Render delete menu if open
	if m.deleteMenuOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#FF5555")). // Red border
			Padding(1, 2)

		dialogContent := "Delete Menu\n\n" +
			"[p] Permanent delete\n" +
			"[n] Normal delete (Trash)\n\n" +
			"Press ESC to cancel"

		dialog := dialogStyle.Render(dialogContent)

		dialogHeight := lipgloss.Height(dialog)
		dialogWidth := lipgloss.Width(dialog)
		startRow := (height - dialogHeight) / 2
		startCol := (width - dialogWidth) / 2

		bgLines := strings.Split(content, "\n")
		fgLines := strings.Split(dialog, "\n")
		for r, fgLine := range fgLines {
			row := startRow + r
			if row >= 0 && row < len(bgLines) {
				bgLines[row] = overlayLine(bgLines[row], fgLine, startCol)
			}
		}
		content = strings.Join(bgLines, "\n")
	}

	// Render permission menu if open
	if m.permissionMenuOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#55FFFF")). // Cyan border
			Padding(1, 2)

		dialogContent := "  Permission Menu\n\n" +
			"[x] Toggle executable (chmod +x/-x)\n\n" +
			"Press ESC to cancel"

		dialog := dialogStyle.Render(dialogContent)

		dialogHeight := lipgloss.Height(dialog)
		dialogWidth := lipgloss.Width(dialog)
		startRow := (height - dialogHeight) / 2
		startCol := (width - dialogWidth) / 2

		bgLines := strings.Split(content, "\n")
		fgLines := strings.Split(dialog, "\n")
		for r, fgLine := range fgLines {
			row := startRow + r
			if row >= 0 && row < len(bgLines) {
				bgLines[row] = overlayLine(bgLines[row], fgLine, startCol)
			}
		}
		content = strings.Join(bgLines, "\n")
	}

	// Render open menu if open
	if m.openMenuOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#FF55FF")). // Purple border
			Padding(1, 2)

		dialogContent := "       Open Menu\n\n" +
			"[c] Open CWD with VSCode (code .)\n" +
			"[C] Open selected with VSCode\n" +
			"[r] Reveal in GUI File Explorer\n" +
			"[e] Open with $EDITOR\n\n" +
			"Press ESC to cancel"

		dialog := dialogStyle.Render(dialogContent)

		dialogHeight := lipgloss.Height(dialog)
		dialogWidth := lipgloss.Width(dialog)
		startRow := (height - dialogHeight) / 2
		startCol := (width - dialogWidth) / 2

		bgLines := strings.Split(content, "\n")
		fgLines := strings.Split(dialog, "\n")
		for r, fgLine := range fgLines {
			row := startRow + r
			if row >= 0 && row < len(bgLines) {
				bgLines[row] = overlayLine(bgLines[row], fgLine, startCol)
			}
		}
		content = strings.Join(bgLines, "\n")
	}

	// Render full help menu if open
	if m.helpMenuOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#FFFF55")). // Yellow border
			Padding(1, 4)

		dialogContent := "FMTUI Help Menu\n\n" +
			"Navigation & Panes:\n" +
			"  j / k, Up / Down   Navigate files / scroll preview\n" +
			"  h / l, Left / Right Go to parent dir / enter dir\n" +
			"  tab                Switch active pane\n" +
			"  [ / ]              Switch tab prev / next (or switch shortcuts/bookmarks tab in sidebar)\n" +
			"  t / w              New tab / close tab\n" +
			"  ctrl+b             Toggle shortcut sidebar pane\n" +
			"  b                  Open bookmark menu (add selected directory / CWD)\n" +
			"  d                  Remove bookmark (only when inside Bookmarks tab)\n\n" +
			"File Actions:\n" +
			"  f / m              Create file / directory\n" +
			"  a                  Toggle select all / deselect all\n" +
			"  i                  File metadata inspector\n" +
			"  K                  Create symlink for selected item\n" +
			"  g                  Jump / follow symlink target\n" +
			"  r                  Rename file or directory\n" +
			"  d                  Delete menu (p: permanent, n: trash)\n" +
			"  y                  Copy menu (p: path, n: name, e: ext, s: tree)\n" +
			"  p                  Permission menu (x: chmod +x/-x)\n" +
			"  o                  Open menu [c: VSCode CWD, C: VSCode item, r: GUI, e: $EDITOR]\n" +
			"  .                  Toggle hidden files\n\n" +
			"  :              	  Temporary switch to subshell (type exit or ctrl+d to return)\n" +
			"  e                  Exit & cd to current dir in shell\n" +
			"  q / ctrl+c         Quit application\n\n" +
			"          Press ` or ESC to close"

		dialog := dialogStyle.Render(dialogContent)

		dialogHeight := lipgloss.Height(dialog)
		dialogWidth := lipgloss.Width(dialog)
		startRow := (height - dialogHeight) / 2
		startCol := (width - dialogWidth) / 2

		bgLines := strings.Split(content, "\n")
		fgLines := strings.Split(dialog, "\n")
		for r, fgLine := range fgLines {
			row := startRow + r
			if row >= 0 && row < len(bgLines) {
				bgLines[row] = overlayLine(bgLines[row], fgLine, startCol)
			}
		}
		content = strings.Join(bgLines, "\n")
	}

	// Render Metadata Inspector dialog if open
	if m.metadataInspectorOpen && len(t.visibleFiles()) > 0 && t.cursor >= 0 && t.cursor < len(t.visibleFiles()) {
		meta := getFileMetadataInfo(t.currentDir, t.visibleFiles()[t.cursor])

		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#55FFFF")). // Cyan border
			Padding(1, 3)

		symlinkLine := ""
		if meta.IsSymlink {
			symlinkLine = fmt.Sprintf("Symlink Target: %s\n", meta.SymTarget)
		}

		dialogContent := fmt.Sprintf("File Metadata Inspector\n\n"+
			"Name:        %s\n"+
			"Path:        %s\n"+
			"Size:        %s (%d bytes)\n"+
			"Permissions: %s\n"+
			"Owner/Group: %s\n"+
			"Modified:    %s\n"+
			"Type:        %s\n"+
			"%s\n"+
			"     Press ESC or any key to close",
			meta.Name, meta.AbsPath, meta.SizeStr, meta.SizeBytes,
			meta.PermStr, meta.OwnerGroup, meta.ModTimeStr, meta.MimeOrType, symlinkLine)

		dialog := dialogStyle.Render(dialogContent)

		dialogHeight := lipgloss.Height(dialog)
		dialogWidth := lipgloss.Width(dialog)
		startRow := (height - dialogHeight) / 2
		startCol := (width - dialogWidth) / 2

		bgLines := strings.Split(content, "\n")
		fgLines := strings.Split(dialog, "\n")
		for r, fgLine := range fgLines {
			row := startRow + r
			if row >= 0 && row < len(bgLines) {
				bgLines[row] = overlayLine(bgLines[row], fgLine, startCol)
			}
		}
		content = strings.Join(bgLines, "\n")
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
