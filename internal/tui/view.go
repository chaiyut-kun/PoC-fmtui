package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	minShortcutWidth      = 10
	shortcutWidthRatio    = 4
	openShortcutPaneRatio = 2

	leftPaneRatioNumerator   = 1
	leftPaneRatioDenominator = 2

	minPathLen        = 15
	pathHeaderPadding = 2
	searchBoxPadding  = 2
	tabBorderOffset   = 4
	paneBorderPadding = 2

	// shortcutPathPadding = 4
	minBarWidth = 5

	dialogWidthPadding     = 8
	shortcutNameFieldWidth = 25
	layoutMarginOffset     = 6
	pathPrefixRatio        = 3
	pathTruncationOffset   = 4
	counterSlotWidth       = 6

	// bottom bar (status line) frame widths — used by view.go & utilities.go
	bottomBarLeftFrameW = 2 // "╰ "
	bottomBarMinGap     = 1
	bottomBarMinWidth   = 12
	bottomBarMinDateW   = 10 // below this a cut date is noise, so hide it
)

// ponytail: View is a single massive function to avoid premature abstraction.
// Ceiling: When layout gets too complex to reason about, split into renderLeftPane, renderRightPane.
func (m model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

	if m.helpMenuOpen {
		width := m.terminalWidth
		height := m.terminalHeight

		helpStyle := lipgloss.NewStyle().
			Width(width).
			Height(height).
			Padding(2, 4)

		var sb strings.Builder
		sb.WriteString(Orange("FMTUI Help Menu") + "\n\n")

		// Render help search bar
		if m.helpSearchActive {
			sb.WriteString(fmt.Sprintf("%s %s\n\n", SearchIcon, m.searchInput.View()))
		} else if m.searchInput.Value() != "" {
			sb.WriteString(fmt.Sprintf("%s %s\n\n", SearchIcon, m.searchInput.Value()))
		} else {
			sb.WriteString(Gray(fmt.Sprintf("%s press s to search help", SearchIcon)) + "\n\n")
		}

		query := strings.ToLower(m.searchInput.Value())
		hasFilter := query != ""

		var renderedLines []string
		if hasFilter {
			for _, item := range helpItems {
				if item.heading {
					continue
				}
				if strings.Contains(strings.ToLower(item.key), query) || strings.Contains(strings.ToLower(item.desc), query) {
					renderedLines = append(renderedLines, fmt.Sprintf("  %-22s  %s", Green(item.key), item.desc))
				}
			}
			if len(renderedLines) == 0 {
				renderedLines = append(renderedLines, "  No matching commands found")
			}
		} else {
			for _, item := range helpItems {
				if item.heading {
					renderedLines = append(renderedLines, "\n"+Orange(item.desc))
				} else {
					renderedLines = append(renderedLines, fmt.Sprintf("  %-22s  %s", Green(item.key), item.desc))
				}
			}
		}

		// Available height for lines: terminal height minus header, search bar, instructions, and padding.
		// Height - (2 top padding + 1 header + 2 search bar + 2 bottom spacing + 2 padding)
		availableHeight := max(height-9, 5)

		// Clamp scroll offset
		maxScroll := max(len(renderedLines)-availableHeight, 0)
		if m.helpScrollOffset > maxScroll {
			m.helpScrollOffset = maxScroll
		}
		if m.helpScrollOffset < 0 {
			m.helpScrollOffset = 0
		}

		endLine := min(m.helpScrollOffset+availableHeight, len(renderedLines))
		visibleLines := renderedLines[m.helpScrollOffset:endLine]

		sb.WriteString(strings.Join(visibleLines, "\n") + "\n\n")
		sb.WriteString(Gray("          Press ESC to exit help / search • j/k/↑↓ to scroll"))

		content := helpStyle.Render(sb.String())
		v := tea.NewView(content)
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
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
		sShortcutWidth = max(sWidth/shortcutWidthRatio-4, minShortcutWidth)
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
		boxContent = fmt.Sprintf("%s %s", SearchIcon, m.searchInput.View())
	} else if m.searchInput.Value() != "" {
		boxContent = fmt.Sprintf("%s %s", SearchIcon, m.searchInput.Value())
	} else {
		boxContent = Gray(fmt.Sprintf("%s press s to search", SearchIcon))
	}

	sb.WriteString(boxStyle.Render(boxContent))
	sb.WriteString("\n")

	maxVisible := m.getMaxVisibleFiles()

	files := t.visibleFiles()
	endIdx := min(t.scrollOffset+maxVisible, len(files))

	// Max width for displayName = leftWidth minus prefix overhead minus border
	// Prefix: cursor(1) + space(1) + marker(1) + space(1) + icon(~2) + space(1) + ItemEndSpace(1) + border(2)
	maxDisplayWidth := max(sLeftWidth-10, 10)

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

		// Truncate long names to prevent line wrapping
		displayName = truncateFileName(displayName, maxDisplayWidth)

		icon, fg, bg := m.CheckDirFilesExt(list)

		cursorSymbol := " "
		if t.cursor == i {
			cursorSymbol = CursorIcon
		}
		selectMarker := ""
		if _, ok := m.selected[i]; ok {
			if t.cursor == i {
				selectMarker = Marker
			} else {
				selectMarker = Green(Marker)
			}
		}

		if t.cursor == i {
			rowText := fmt.Sprintf("%s %s %s %s", cursorSymbol, selectMarker, icon, displayName)
			sb.WriteString(bg(Black(rowText)) + "\n")
		} else {
			sb.WriteString(fmt.Sprintf("%s%s %s\n", cursorSymbol, selectMarker, fg(icon+ItemEndSpace+displayName)))
		}
	}

	// Dynamic padding: push metadata to bottom of the box interior
	numRendered := endIdx - t.scrollOffset
	paddingLines := maxVisible - numRendered
	if paddingLines > 0 {
		sb.WriteString(strings.Repeat("\n", paddingLines))
	}

	// Metadata bar rides the bottom border of the pane.
	metaFiles := t.visibleFiles()
	total := len(metaFiles)
	pos := min(t.cursor+1, total)
	// ponytail: fixed-width slot so the counter never shifts; a scrollbar would
	// need fixed-width (truncated) file rows — add when names are truncated.
	inner := sLeftWidth - paneBorderPadding
	borderText := inner - 2
	slot := min(counterSlotWidth, max(borderText-4, 3))
	raw := fmt.Sprintf("%d/%d", pos, total)
	if len(raw) > slot {
		slot = len(raw) // 10k+ items: let the number grow, metadata shifts
	}
	counter := raw + strings.Repeat(" ", max(0, slot-len(raw)))
	budget := sLeftWidth - bottomBarLeftFrameW - slot - 1 - bottomBarMinGap
	var metaText string
	switch {
	case len(m.selected) > 0:
		plain := buildSelectionSummary(t.currentDir, metaFiles, m.selected)
		metaText = Yellow(truncateMeta(plain, budget))
	case len(metaFiles) > 0 && t.cursor >= 0 && t.cursor < len(metaFiles):
		meta := getFileMetadataInfo(t.currentDir, metaFiles[t.cursor])
		leftPlain := meta.SizeStr + " • " + meta.PermStrShort
		room := budget - len([]rune(leftPlain)) - 3 // 3 = " • " before the date
		switch {
		case room < 0: // tiny pane: drop size, keep perms whole if at all possible
			metaText = truncateMeta(meta.PermStrShort, budget)
		case room < len([]rune(meta.ModTimeStr)):
			if room >= bottomBarMinDateW {
				metaText = Yellow(meta.SizeStr) + " • " + meta.PermStrShort +
					" • " + truncateMeta(meta.ModTimeStr, room)
			} else {
				metaText = Yellow(meta.SizeStr) + " • " + meta.PermStrShort
			}
		default:
			metaText = Yellow(meta.SizeStr) + " • " + meta.PermStrShort +
				" • " + meta.ModTimeStr
		}
	default:
		metaText = Gray(truncateMeta("No file / directory found", budget))
	}
	bottomBody := Cyan(counter) + metaText

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

	//prepare for if any error
	bottomBarLines := 0
	if m.err != nil {
		bottomBarLines = 2
	}
	boxHeight := max(height-bottomBarLines, minBoxHeight)

	// Calculate shortcut pane width
	shortcutWidth := 0
	if m.shortcutPaneOpen {
		// 8 is reduce unit
		shortcutWidth = max(width/shortcutWidthRatio-8, minShortcutWidth)
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

	leftBorderColor := colorGray
	rightBorderColor := colorGray
	shortcutBorderColor := colorGray
	if m.shortcutPaneOpen {
		if m.activePane == 0 {
			shortcutBorderColor = colorBlue
		} else {
			shortcutBorderColor = colorYellow
		}
	}
	if m.activePane == 1 {
		leftBorderColor = colorBlue
	}
	if m.activePane == 2 {
		rightBorderColor = colorBlue
	}

	// 0. Create shortcut pane (if open)
	var shortcutBox string
	if m.shortcutPaneOpen {
		var lines []string

		// --- Shortcuts section ---
		for i, s := range m.shortcuts {
			cursor := "  "
			label := fmt.Sprintf("%s%s %s", cursor, s.Icon, s.Name)
			if m.shortcutSection == 0 && i == m.shortcutCursor {
				label = BgCyan(Black(fmt.Sprintf("%s %s %s", CursorIcon, s.Icon, s.Name)))
			}
			lines = append(lines, label)
		}

		// --- Divider ---
		sepWidth := max(shortcutWidth-paneBorderPadding, minBarWidth)
		lines = append(lines, Gray(strings.Repeat("─", sepWidth)))

		// --- Bookmarks section ---
		// available lines for bookmarks = total height - shortcuts - 1 divider - 2 border
		availBmLines := max(boxHeight-len(m.shortcuts)-1-paneBorderPadding, 0)

		// Clamp scroll
		maxBmScroll := max(len(m.bookmarks)-availBmLines, 0)
		bmScroll := min(m.bookmarkScrollOffset, maxBmScroll)

		visibleBm := m.bookmarks
		if bmScroll < len(m.bookmarks) {
			visibleBm = m.bookmarks[bmScroll:]
		} else {
			visibleBm = nil
		}
		if len(visibleBm) > availBmLines {
			visibleBm = visibleBm[:availBmLines]
		}

		if len(m.bookmarks) == 0 {
			lines = append(lines, Gray(" (No bookmarks)"))
			lines = append(lines, Gray(" Press 'b' to add"))
		} else {
			for i, bm := range visibleBm {
				idx := i + bmScroll
				label := fmt.Sprintf("  %s", bm.Name)
				if m.shortcutSection == 1 && idx == m.shortcutCursor {
					label = BgCyan(Black(fmt.Sprintf("%s%s", CursorIcon+ItemEndSpace, bm.Name)))
				}
				lines = append(lines, label)
			}
		}

		shortcutText := strings.Join(lines, "\n")

		shortcutStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(shortcutBorderColor)).
			Width(shortcutWidth).
			Height(boxHeight)

		shortcutBox = shortcutStyle.Render(shortcutText)

		// Overlay title on border
		scLines := strings.Split(shortcutBox, "\n")
		if len(scLines) > 0 {
			scLines[0] = createTopBorder(" Shortcuts ", shortcutWidth, shortcutBorderColor, "1")
			shortcutBox = strings.Join(scLines, "\n")
		}
	}

	// 1. Create left box (File Explorer)
	bottomSectionHeight := 5
	explorerHeight := max(boxHeight-bottomSectionHeight-paneBorderPadding, minBoxHeight-paneBorderPadding)

	leftStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(leftBorderColor)).
		Width(leftWidth).
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
		rawTitle := ItemEndSpace + strings.Join(tabParts, " | ") + ItemEndSpace
		title := []rune(rawTitle)
		maxTitleLen := leftWidth - tabBorderOffset
		if maxTitleLen > searchBoxPadding && len(title) > maxTitleLen {
			pathTrimmed := " ..." + string(title[len(title)-(maxTitleLen-tabBorderOffset):])
			title = []rune(pathTrimmed)
		}
		leftLines[0] = createTopBorder(string(title), leftWidth, leftBorderColor, "2")
		// Metadata bar rides the bottom border of the explorer pane.
		leftLines[len(leftLines)-1] = createBottomBorder(bottomBody, leftWidth, leftBorderColor)
		leftBox = strings.Join(leftLines, "\n")
	}

	// Render process status and clipboard panes at bottom
	processWidth := (leftWidth / 3) + 10
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
			barStr = fmt.Sprintf("[%s%s] %s", strings.Repeat("█", filledLen), strings.Repeat("░", emptyLen), percentStr)
		} else {
			barStr = fmt.Sprintf(" %s", percentStr)
		}

		// fileName := m.copyJobFileName
		// if len(fileName) > innerW-2 {
		// 	fileName = fileName[:max(0, innerW-5)] + "..."
		// }
		processContent = fmt.Sprintf("\n%s\n", barStr)
	}

	var clipboardContent string
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

	processStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(leftBorderColor)).
		Width(processWidth).
		Height(bottomSectionHeight)

	clipboardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(leftBorderColor)).
		Width(clipboardWidth).
		Height(bottomSectionHeight)

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
		Width(rightWidth).
		Height(boxHeight)

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

		// need to split as function
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

		// need to split as function
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
		// need to split as function
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
		// need to split as function
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

		// need to split as function
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

		// need to split as function
		dialogContent := "       Open Menu\n\n" +
			"[c] Open CWD with VSCode (code .)\n" +
			"[C] Open selected with VSCode\n" +
			"[r] Reveal in GUI File Explorer\n" +
			"[e] Open with $EDITOR\n\n" +
			"Press ESC to cancel"

		dialog := dialogStyle.Render(dialogContent)
		content = overlayDialog(content, dialog, width, height, DefaultYDivisor)
	}

	// Render Command Line overlay
	if m.cmdLineOpen {
		var suggestionLine string
		if suggestion, ok := findCmdSuggestion(m.fileInput.Value()); ok {
			suggestionLine = Gray(fmt.Sprintf("%s [%s]", suggestion.name, suggestion.desc))
		}

		dialogContent := m.fileInput.View()
		if suggestionLine != "" {
			dialogContent += "\n" + suggestionLine
		}

		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorGreen)).
			Padding(0, 1).
			Width(dialogWidth)

		dialog := dialogStyle.Render(dialogContent)

		// Inject left-aligned title "Command:" on top border of dialog
		dialogLines := strings.Split(dialog, "\n")
		if len(dialogLines) > 0 {
			dw := lipgloss.Width(dialog)
			barWidth := dw - 2
			titleStr := " Command: "
			middle := lipgloss.PlaceHorizontal(
				barWidth,
				lipgloss.Left,
				titleStr,
				lipgloss.WithWhitespaceChars("─"),
			)
			topStr := "╭" + middle + "╮"
			dialogLines[0] = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGreen)).Render(topStr)
			dialog = strings.Join(dialogLines, "\n")
		}

		content = overlayDialog(content, dialog, width, height, DefaultYDivisor)
	}

	// Render create / rename input overlay
	if m.inputMode {
		titleText := "New item"
		if m.isRenaming {
			titleText = "Rename"
		}

		// renameHelp = "• name.ext = file, name = dir"

		dialogContent := m.fileInput.View() + "\n" + Gray("Enter to confirm • Esc to cancel")

		// Width is exactly 60 characters outer width (content width is 56)
		dialogWidth := 60
		dialogStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(pink)).
			Padding(0, 1).
			Width(dialogWidth - 4)

		dialog := dialogStyle.Render(dialogContent)

		// Inject left-aligned title on top border of dialog using PlaceHorizontal
		dialogLines := strings.Split(dialog, "\n")
		if len(dialogLines) > 0 {
			dialogWidth := lipgloss.Width(dialog)
			barWidth := dialogWidth - 2
			titleStr := ItemEndSpace + titleText + ItemEndSpace
			runes := []rune(titleStr)
			if len(runes) > barWidth {
				titleStr = string(runes[:barWidth])
			}
			middle := lipgloss.PlaceHorizontal(
				barWidth,
				lipgloss.Left,
				titleStr,
				lipgloss.WithWhitespaceChars("─"),
			)
			topStr := "╭" + middle + "╮"
			dialogLines[0] = lipgloss.NewStyle().Foreground(lipgloss.Color(pink)).Render(topStr)
			dialog = strings.Join(dialogLines, "\n")
		}

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

		// need to split as function
		dialogContent := fmt.Sprintf("File Metadata Inspector\n\n"+
			"Name:        %s\n"+
			"Size:        %s\n"+
			"Permissions: %s\n"+
			"Owner/Group: %s\n"+
			"Modified:    %s\n"+
			"Type:        %s\n"+
			"%s\n"+
			Yellow("Press ESC or any key to close"),
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
		var inputLine string

		if m.quickFindSelecting {
			inputLine = fmt.Sprintf(" %s %s", SearchIcon, Gray(m.quickFindInput.Value()))
		} else {
			inputLine = fmt.Sprintf(" %s %s", SearchIcon, m.quickFindInput.View())
		}

		// Build results list
		var resultLines []string
		if len(m.quickFindResults) == 0 && m.quickFindInput.Value() == "" {
			resultLines = append(resultLines, fmt.Sprintf(" %s", Gray("Bookmarks are shown here…")))
		} else if len(m.quickFindResults) == 0 {
			resultLines = append(resultLines, fmt.Sprintf(" %s", Gray("No results")))
		} else {
			// Clamp scroll offset
			maxScroll := max(len(m.quickFindResults)-maxResults, 0)
			if m.quickFindScrollOffset > maxScroll {
				m.quickFindScrollOffset = maxScroll
			}
			if m.quickFindScrollOffset < 0 {
				m.quickFindScrollOffset = 0
			}

			endIdx := min(m.quickFindScrollOffset+maxResults, len(m.quickFindResults))
			shown := m.quickFindResults[m.quickFindScrollOffset:endIdx]

			// Compute available space for paths
			pathMaxLen := max(dialogW-dialogWidthPadding-shortcutNameFieldWidth-layoutMarginOffset, minPathLen)

			for i, r := range shown {
				actualIdx := m.quickFindScrollOffset + i
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
				if m.quickFindSelecting && actualIdx == m.quickFindSelected {
					line = fmt.Sprintf("%s %s %s   %s", Cyan(CursorIcon), icon, namePart, Gray(beautifiedPath))
				}
				resultLines = append(resultLines, line)
			}
		}

		// Pad resultLines to maxResults so height doesn't change
		for len(resultLines) < maxResults {
			resultLines = append(resultLines, "")
		}

		// Assemble dialog content
		modeHint := "  tab: toggle input - select • Enter: select mode • Esc: close"
		if m.quickFindSelecting {
			modeHint = "  tab: toggle input - select • ↑↓/jk: navigate • Enter: jump • Esc: close"
		}
		dialogContent := inputLine + "\n" + "\n" + strings.Join(resultLines, "\n") + "\n\n" + Gray(modeHint)

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

// need to move to utilities
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
