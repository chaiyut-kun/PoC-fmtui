package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
)

const (
	colorCyan   = "#55FFFF"
	colorGreen  = "#55FF55"
	colorPurple = "#FF55FF"
	colorRed    = "#FF5555"
	colorYellow = "#FFFF55"
	colorGray   = "#555555"
	colorBlue   = "#5555FF"

	minScreenWidth        = 20
	minScreenHeight       = 5
	defaultScreenHeight   = 20
	minBoxHeight          = 3
	minLeftPaneWidth      = 10
	minShortcutWidth      = 14
	shortcutWidthRatio    = 4
	openShortcutPaneRatio = 2

	leftPaneRatioNumerator   = 1
	leftPaneRatioDenominator = 2

	minPathLen          = 15
	pathHeaderPadding   = 2
	searchBoxPadding    = 6
	tabBorderOffset     = 4
	paneBorderPadding   = 2
	minWidthForFullTabs = 27

	shortcutPathPadding = 4
	copyJobHeightOffset = 4
	minProgressWidth    = 14
	progressLabelOffset = 25
	minBarWidth         = 5
	progressPaneHeight  = 2

	dialogWidthPadding     = 8
	shortcutNameFieldWidth = 25
	layoutMarginOffset     = 6
	pathPrefixRatio        = 3
	pathTruncationOffset   = 4
)

// ponytail: View is a single massive function to avoid premature abstraction.
// Ceiling: When layout gets too complex to reason about, split into renderLeftPane, renderRightPane.
func (m model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

	t := m.curTab()
	var sb strings.Builder
	sb.Grow(m.terminalWidth * m.terminalHeight) // ponytail: pre-allocate to prevent reallocation

	// screen width
	sWidth := m.terminalWidth
	if sWidth < minScreenWidth {
		sWidth = DefaultScreenWidth
	}

	sShortcutWidth := 0
	if m.shortcutPaneOpen {
		sShortcutWidth = max(sWidth/shortcutWidthRatio, minShortcutWidth)
	}
	var sLeftWidth int
	if m.shortcutPaneOpen {
		sLeftWidth = max(sWidth/openShortcutPaneRatio, minLeftPaneWidth)
	} else {
		sAvailableWidth := sWidth - sShortcutWidth
		sLeftWidth = (sAvailableWidth * leftPaneRatioNumerator) / leftPaneRatioDenominator
		if sLeftWidth < minLeftPaneWidth {
			sLeftWidth = sAvailableWidth / openShortcutPaneRatio
		}
	}

	userPart := fmt.Sprintf(" %s: ", getCurrentUser())
	maxPathLen := max(sLeftWidth-pathHeaderPadding-len(userPart), minPathLen)
	displayDir := shrinkPath(t.currentDir, maxPathLen)
	sb.WriteString(fmt.Sprintf(" %s: %s\n", Cyan(getCurrentUser()), Cyan(displayDir)))

	borderColor := colorGray
	if m.searchActive {
		borderColor = colorBlue
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Width(sLeftWidth - searchBoxPadding) // fit inside the left pane

	var boxContent string
	if m.searchActive {
		runes := []rune(m.searchText)
		cur := min(m.searchCursor, len(runes))
		before := string(runes[:cur])
		after := string(runes[cur:])
		boxContent = fmt.Sprintf("%s %s█%s", SearchIcon, before, after)
	} else if m.searchText != "" {
		boxContent = fmt.Sprintf("%s %s", SearchIcon, m.searchText)
	} else {
		boxContent = Gray(fmt.Sprintf("%s press s to search", SearchIcon))
	}

	sb.WriteString(boxStyle.Render(boxContent))
	sb.WriteString("\n")

	maxVisible := m.getMaxVisibleFiles()

	files := t.visibleFiles()
	endIdx := min(t.scrollOffset+maxVisible, len(files))

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

		icon, fg, bg := m.CheckDirFilesExt(list)

		cursorSymbol := " "
		if t.cursor == i {
			cursorSymbol = CursorIcon
		}
		selectMarker := " "
		if _, ok := m.selected[i]; ok {
			selectMarker = Yellow("*")
		}

		if t.cursor == i {
			rowText := fmt.Sprintf("%s %s %s %s%s", cursorSymbol, selectMarker, icon, displayName, ItemEndSpace)
			sb.WriteString(bg(Black(rowText)) + "\n")
		} else {
			rowText := fmt.Sprintf("%s%s %s %s%s", cursorSymbol, selectMarker, icon, displayName, ItemEndSpace)
			sb.WriteString(" " + bg(fg(rowText)) + "\n")
		}
	}

	// Dynamic padding: push metadata to bottom of the box interior
	numRendered := endIdx - t.scrollOffset
	paddingLines := maxVisible - numRendered
	if paddingLines > 0 {
		sb.WriteString(strings.Repeat("\n", paddingLines))
	}

	// Metadata bar pinned at bottom inside the border
	files2 := t.visibleFiles()
	if len(files2) > 0 && t.cursor >= 0 && t.cursor < len(files2) {
		meta := getFileMetadataInfo(t.currentDir, files2[t.cursor])
		sb.WriteString(fmt.Sprintf("%s %s | %s | %s | %s\n",
			Gray("─"), Yellow(meta.SizeStr), meta.PermStr, meta.OwnerGroup, meta.ModTimeStr))
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
		cur := min(m.inputCursor, len(runes))
		before := string(runes[:cur])
		after := string(runes[cur:])
		sb.WriteString(fmt.Sprintf("%s: %s█%s\n(Enter to confirm, Esc to cancel)\n", prompt, before, after))
	}

	// Trim trailing newline so lipgloss box height is calculated precisely
	renderedText := strings.TrimSuffix(sb.String(), "\n")

	width := m.terminalWidth
	height := m.terminalHeight
	if width < minScreenWidth {
		width = DefaultScreenWidth
	}
	if height < minScreenHeight {
		height = defaultScreenHeight
	}

	bottomBarLines := 1
	if m.err != nil {
		bottomBarLines = 2
	}
	boxHeight := max(height-bottomBarLines, minBoxHeight)

	// Calculate shortcut pane width
	shortcutWidth := 0
	if m.shortcutPaneOpen {
		shortcutWidth = max(width/shortcutWidthRatio, minShortcutWidth)
	}

	var leftWidth, rightWidth int
	if m.shortcutPaneOpen {
		leftWidth = max(width/openShortcutPaneRatio, minLeftPaneWidth)
		rightWidth = max(width-shortcutWidth-leftWidth, minLeftPaneWidth)
	} else {
		availableWidth := width - shortcutWidth
		leftWidth = (availableWidth * leftPaneRatioNumerator) / leftPaneRatioDenominator
		if leftWidth < minLeftPaneWidth {
			leftWidth = availableWidth / openShortcutPaneRatio
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
					// use space for icon gap with cursor
					cursor = CursorIcon + " "
				}
				label := fmt.Sprintf("%s%s  %s", cursor, s.Icon, s.Name)
				if i == m.shortcutCursor {
					label = BgCyan(Black(fmt.Sprintf("%s%s  %s", cursor, s.Icon, s.Name)))
				}
				shortcutText += label + "\n"
			}
			shortcutText = strings.TrimSuffix(shortcutText, "\n")

			// ponytail: Append full path of active shortcut at the bottom of the pane
			if m.shortcutCursor >= 0 && m.shortcutCursor < len(activeShortcuts) {
				selectedPath := activeShortcuts[m.shortcutCursor].Path
				maxWidth := max(shortcutWidth-shortcutPathPadding, minBarWidth)
				truncatedPath := truncatePathLeft(selectedPath, maxWidth)
				sep := strings.Repeat("─", shortcutWidth-paneBorderPadding)

				linesCount := strings.Count(shortcutText, "\n") + 1
				targetHeight := boxHeight - paneBorderPadding
				paddingNeeded := targetHeight - linesCount - paneBorderPadding // subtract for sep and path line

				if paddingNeeded > 0 {
					shortcutText += strings.Repeat("\n", paddingNeeded)
				} else {
					shortcutText += "\n"
				}
				shortcutText += fmt.Sprintf("%s\n %s", Gray(sep), Yellow(truncatedPath))
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
			if shortcutWidth-paneBorderPadding >= minWidthForFullTabs {
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
	bottomSectionHeight := 5
	explorerHeight := boxHeight - bottomSectionHeight - paneBorderPadding
	if explorerHeight < minBoxHeight-paneBorderPadding {
		explorerHeight = minBoxHeight - paneBorderPadding
	}

	leftStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(leftBorderColor)).
		Width(leftWidth - paneBorderPadding).
		Height(explorerHeight)

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
		maxTitleLen := leftWidth - tabBorderOffset
		if maxTitleLen > searchBoxPadding && len(title) > maxTitleLen {
			pathTrimmed := " ..." + string(title[len(title)-(maxTitleLen-tabBorderOffset):])
			title = []rune(pathTrimmed)
		}
		leftLines[0] = createTopBorder(string(title), leftWidth, leftBorderColor, "2")
		leftBox = strings.Join(leftLines, "\n")
	}

	// Render process status and clipboard panes at bottom
	processWidth := leftWidth / 3
	clipboardWidth := leftWidth - processWidth

	var processContent string
	if m.copyJobActive {
		innerW := max(processWidth-2, 1)
		percentStr := fmt.Sprintf("%.0f%%", m.copyJobProgress*100)
		labelAndBracketsLen := 5 + len(percentStr)
		barW := innerW - labelAndBracketsLen
		var barStr string
		if barW > 0 {
			filledLen := min(int(m.copyJobProgress*float64(barW)), barW)
			emptyLen := barW - filledLen
			barStr = fmt.Sprintf(" [%s%s] %s", strings.Repeat("█", filledLen), strings.Repeat("░", emptyLen), percentStr)
		} else {
			barStr = fmt.Sprintf(" %s", percentStr)
		}

		fileName := m.copyJobFileName
		if len(fileName) > innerW-2 {
			fileName = fileName[:max(0, innerW-5)] + "..."
		}
		processContent = fmt.Sprintf("Copying...\n%s\n  %s", barStr, fileName)
	} else {
		processContent = "\n  Idle"
	}

	var clipboardContent string
	if len(m.stagedPaths) == 0 {
		clipboardContent = "\n  Empty"
	} else {
		innerW := max(clipboardWidth-2, 1)
		var itemLines []string
		for i, path := range m.stagedPaths {
			if i >= 2 {
				itemLines = append(itemLines, fmt.Sprintf("  ... and %d more", len(m.stagedPaths)-2))
				break
			}
			baseName := filepath.Base(path)
			if len(baseName) > innerW-4 {
				baseName = baseName[:max(0, innerW-7)] + "..."
			}
			itemLines = append(itemLines, "  - "+baseName)
		}
		clipboardContent = strings.Join(itemLines, "\n")
	}

	processStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(leftBorderColor)).
		Width(processWidth - paneBorderPadding).
		Height(bottomSectionHeight - paneBorderPadding)

	clipboardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(leftBorderColor)).
		Width(clipboardWidth - paneBorderPadding).
		Height(bottomSectionHeight - paneBorderPadding)

	processBox := processStyle.Render(processContent)
	clipboardBox := clipboardStyle.Render(clipboardContent)

	// Add top border labels to the sub-panes
	processLines := strings.Split(processBox, "\n")
	if len(processLines) > 0 {
		processLines[0] = createTopBorder(" Process ", processWidth, leftBorderColor, "")
		processBox = strings.Join(processLines, "\n")
	}

	var clipTitle string
	if len(m.stagedPaths) > 0 {
		opLabel := "Copy"
		if m.stagedOp == "move" {
			opLabel = "Cut"
		}
		clipTitle = fmt.Sprintf(" Clipboard (%s) ", opLabel)
	} else {
		clipTitle = " Clipboard "
	}
	clipboardLines := strings.Split(clipboardBox, "\n")
	if len(clipboardLines) > 0 {
		clipboardLines[0] = createTopBorder(clipTitle, clipboardWidth, leftBorderColor, "")
		clipboardBox = strings.Join(clipboardLines, "\n")
	}

	bottomPanes := lipgloss.JoinHorizontal(lipgloss.Top, processBox, clipboardBox)
	leftBox = lipgloss.JoinVertical(lipgloss.Left, leftBox, bottomPanes)

	// 2. Create right box (Previewer)
	var rightText string
	maxVisiblePreviewLines := max(boxHeight-paneBorderPadding, 1)

	if len(t.visibleFiles()) > 0 {
		selectedName := t.visibleFiles()[t.cursor]
		targetPath := findAbsPath(t.currentDir, selectedName)
		previewLines := getPreviewLines(targetPath, isDir(t.currentDir, selectedName), m.showHidden)

		// Clamp previewScrollOffset based on number of lines and visible height
		maxScroll := max(len(previewLines)-maxVisiblePreviewLines, 0)
		if t.previewScrollOffset > maxScroll {
			t.previewScrollOffset = maxScroll
		}
		if t.previewScrollOffset < 0 {
			t.previewScrollOffset = 0
		}

		endLine := min(t.previewScrollOffset+maxVisiblePreviewLines, len(previewLines))

		maxWidth := max(rightWidth-paneBorderPadding, 1)
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
		Width(rightWidth - paneBorderPadding).
		Height(boxHeight - paneBorderPadding)

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
	if m.err != nil {
		content += "\n" + Red(fmt.Sprintf("Error: %v", m.err))
	} else if m.statusMsg != "" {
		content += "\n" + Green(m.statusMsg)
	}
	// content += "\n" + helpText
	content += "\n"
	// Render copy menu if open
	if m.copyMenuOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorGreen)). // Green border
			Padding(1, 2)

		dialogContent := "         Copy Menu\n\n" +
			"[p] Copy absolute path\n" +
			"[n] Copy name only\n" +
			"[e] Copy name with ext\n" +
			"[s] Copy dir structure (tree)\n" +
			"[c] Copy file content\n\n" +
			"Press ESC to cancel"

		dialog := dialogStyle.Render(dialogContent)
		content = overlayDialog(content, dialog, width, height, DefaultYDivisor)
	}

	// Render bookmark menu if open
	if m.bookmarkMenuOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorPurple)). // Purple border
			Padding(1, 2)

		dialogContent := "    Bookmark Options\n\n" +
			"[s] Bookmark Selected Dir\n" +
			"[c] Bookmark Current CWD\n\n" +
			"Press any other key to close"

		dialog := dialogStyle.Render(dialogContent)
		content = overlayDialog(content, dialog, width, height, DefaultYDivisor)
	}

	// Render delete bookmark confirmation menu if open
	if m.deleteBookmarkConfirmOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorRed)). // Red border
			Padding(1, 2)

		var bookmarkName string
		if m.shortcutCursor >= 0 && m.shortcutCursor < len(m.bookmarks) {
			bookmarkName = m.bookmarks[m.shortcutCursor].Name
		}
		dialogContent := fmt.Sprintf("   Remove Bookmark?\n\nRemove bookmark for %q?\n\n   [y] Yes      [n] No", bookmarkName)
		dialog := dialogStyle.Render(dialogContent)
		content = overlayDialog(content, dialog, width, height, DefaultYDivisor)
	}

	// Render delete menu if open
	if m.deleteMenuOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorRed)). // Red border
			Padding(1, 2)

		dialogContent := "Delete Menu\n\n" +
			"[p] Permanent delete\n" +
			"[n] Normal delete (Trash)\n\n" +
			"Press ESC to cancel"

		dialog := dialogStyle.Render(dialogContent)
		content = overlayDialog(content, dialog, width, height, DefaultYDivisor)
	}

	// Render permission menu if open
	if m.permissionMenuOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorCyan)). // Cyan border
			Padding(1, 2)

		dialogContent := "  Permission Menu\n\n" +
			"[x] Toggle executable (chmod +x/-x)\n\n" +
			"Press ESC to cancel"

		dialog := dialogStyle.Render(dialogContent)
		content = overlayDialog(content, dialog, width, height, DefaultYDivisor)
	}

	// Render open menu if open
	if m.openMenuOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorPurple)). // Purple border
			Padding(1, 2)

		dialogContent := "       Open Menu\n\n" +
			"[c] Open CWD with VSCode (code .)\n" +
			"[C] Open selected with VSCode\n" +
			"[r] Reveal in GUI File Explorer\n" +
			"[e] Open with $EDITOR\n\n" +
			"Press ESC to cancel"

		dialog := dialogStyle.Render(dialogContent)
		content = overlayDialog(content, dialog, width, height, DefaultYDivisor)
	}

	// Render full help menu if open
	if m.helpMenuOpen {
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorYellow)). // Yellow border
			Padding(1, 4)

		dialogContent := "FMTUI Help Menu\n\n" +
			"Navigation & Panes:\n" +
			"  j / k, Up / Down   Navigate files / scroll preview\n" +
			"  h / l, Left / Right Go to parent dir / enter dir\n" +
			"  tab                Switch active pane\n" +
			"  [ / ]              Switch tab prev / next (or switch shortcuts/bookmarks tab in sidebar)\n" +
			"  t / w              New tab / close tab\n" +
			"  ctrl+b / \\          Toggle shortcut sidebar pane\n" +
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
		content = overlayDialog(content, dialog, width, height, DefaultYDivisor)
	}

	// Render Metadata Inspector dialog if open
	if m.metadataInspectorOpen && len(t.visibleFiles()) > 0 && t.cursor >= 0 && t.cursor < len(t.visibleFiles()) {
		meta := getFileMetadataInfo(t.currentDir, t.visibleFiles()[t.cursor])

		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorGray)). // Cyan border
			Padding(1, 4)

		symlinkLine := ""
		if meta.IsSymlink {
			symlinkLine = fmt.Sprintf("Symlink Target: %s\n", meta.SymTarget)
		}

		dialogContent := fmt.Sprintf("File Metadata Inspector\n\n"+
			"Name:        %s\n"+
			"Size:        %s\n"+
			"Permissions: %s\n"+
			"Owner/Group: %s\n"+
			"Modified:    %s\n"+
			"Type:        %s\n"+
			"%s\n"+
			color.YellowString("Press ESC or any key to close"),
			meta.Name, meta.SizeStr,
			meta.PermStr, meta.OwnerGroup, meta.ModTimeStr, meta.MimeOrType, symlinkLine)

		dialog := dialogStyle.Render(dialogContent)
		content = overlayDialog(content, dialog, width, height, DefaultYDivisor)
	}

	// Render Quick Find overlay
	if m.quickFindOpen {
		dialogW := min(quickSearchWidthDefault, width-4)
		if dialogW < 40 {
			dialogW = quickSearchHalfWidth
		}

		maxResults := max(min(quickSearchMaxResultDefault, height-8), 5)

		// Build input line with cursor
		runes := []rune(m.quickFindText)
		cur := min(m.quickFindCursor, len(runes))
		var inputLine string

		if m.quickFindSelecting {
			inputLine = fmt.Sprintf("  %s   %s", SearchIcon, Gray(m.quickFindText))
		} else {
			before := string(runes[:cur])
			after := string(runes[cur:])
			inputLine = fmt.Sprintf("  %s  %s█%s", SearchIcon, before, after)
		}

		// Horizontal divider (dialogW - 4 accounts for Padding(1, 2) which takes 4 columns)
		divider := lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorCyan)).
			Render(strings.Repeat("─", dialogW-4))

		// Build results list
		var resultLines []string
		if len(m.quickFindResults) == 0 && m.quickFindText == "" {
			resultLines = append(resultLines, fmt.Sprintf("  %s", Gray("Bookmarks are shown here…")))
		} else if len(m.quickFindResults) == 0 {
			resultLines = append(resultLines, fmt.Sprintf("  %s", Gray("No results")))
		} else {
			shown := m.quickFindResults
			if len(shown) > maxResults {
				shown = shown[:maxResults]
			}

			// Compute available space for paths
			pathMaxLen := max(dialogW-dialogWidthPadding-shortcutNameFieldWidth-layoutMarginOffset, minPathLen)

			for i, r := range shown {
				icon := r.Icon
				if icon == "" {
					icon = DirIcon
				}
				namePart := fmt.Sprintf("%-25s", r.Name)
				if len(r.Name) > shortcutNameFieldWidth {
					namePart = r.Name[:shortcutNameFieldWidth-3] + "..."
				}

				beautifiedPath := shrinkPath(r.Path, pathMaxLen)

				line := fmt.Sprintf("  %s %s   %s", icon, namePart, Gray(beautifiedPath))
				if m.quickFindSelecting && i == m.quickFindSelected {
					line = fmt.Sprintf("%s %s %s   %s", Cyan(CursorIcon), icon, namePart, Gray(beautifiedPath))
				}
				resultLines = append(resultLines, line)
			}
		}

		// Assemble dialog content
		modeHint := "  tab: toggle input - select • Enter: select mode • Esc: close"
		if m.quickFindSelecting {
			modeHint = "  tab: toggle input - select • ↑↓/jk: navigate • Enter: jump • Esc: close"
		}
		dialogContent := inputLine + "\n" + divider + "\n" + strings.Join(resultLines, "\n") + "\n\n" + Gray(modeHint)

		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorCyan)).
			Padding(1, 2).
			Width(dialogW)

		dialog := dialogStyle.Render(dialogContent)
		content = overlayDialog(content, dialog, width, height, 3)
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// shrinkPath replaces user home path with ~/ and truncates in the middle if it exceeds maxLen.
func shrinkPath(path string, maxLen int) string {
	home, err := os.UserHomeDir()
	if err == nil {
		if path == home {
			return "~"
		}
		if strings.HasPrefix(path, home+string(filepath.Separator)) {
			path = "~" + path[len(home):]
		}
	}

	if len(path) <= maxLen {
		return path
	}

	prefixLen := maxLen / pathPrefixRatio
	suffixLen := maxLen - prefixLen - pathTruncationOffset
	if prefixLen < 3 {
		prefixLen = 3
	}
	if suffixLen < 5 {
		suffixLen = 5
	}
	if prefixLen+suffixLen+4 >= len(path) {
		return path
	}
	return path[:prefixLen] + "..." + path[len(path)-suffixLen:]
}

// overlayDialog centers a foreground dialog string and overlays it onto the background string.
func overlayDialog(bg string, fg string, width int, height int, yDivisor int) string {
	const centerDivisor = 2
	dialogHeight := lipgloss.Height(fg)
	dialogWidth := lipgloss.Width(fg)
	startRow := (height - dialogHeight) / yDivisor
	startCol := (width - dialogWidth) / centerDivisor

	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")
	for r, fgLine := range fgLines {
		row := startRow + r
		if row >= 0 && row < len(bgLines) {
			bgLines[row] = overlayLine(bgLines[row], fgLine, startCol)
		}
	}
	return strings.Join(bgLines, "\n")
}
