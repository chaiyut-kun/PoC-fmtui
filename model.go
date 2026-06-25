package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	tea "charm.land/bubbletea/v2"
)

type model struct {
	filesPath  []string
	cursor     int
	currentDir string
	selected   map[int]struct{}
}

func InitialModel() model {

	// read and list file from [./files]
	currentDir, _ := os.Getwd()
	entries, err := os.ReadDir(currentDir)
	if err != nil {

		return model{
			filesPath:  []string{},
			currentDir: currentDir,
			selected:   make(map[int]struct{}),
		}
	}

	files := []string{}
	for _, entry := range entries {
		files = append(files, entry.Name())
	}

	return model{
		filesPath:  files,
		currentDir: currentDir,
		selected:   make(map[int]struct{}),
	}

}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			_ = clearTempDirFile()
			return m, tea.Quit
		case "d", "D":
			_ = writeTempDirFile(m.currentDir)
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.filesPath) - 1
			}
		case "down", "j":
			if m.cursor < len(m.filesPath)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}

		case "enter", "space":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}

		case "a", "A":
			if err := m.AddTask("Some task"); err != nil {
				fmt.Println(err.Error())
			}

		// copy file path
		case "c", "C":
			absPath := findAbsPath(m.currentDir, m.filesPath[m.cursor])
			// copy to clipboard
			if err := copyToClipboard(absPath); err != nil {
				fmt.Println(err.Error())
			}

		case "right", "l", "L":
			if !isDir(m.currentDir, m.filesPath[m.cursor]) {
				return m, nil
			}

			selectedName := m.filesPath[m.cursor]
			nextDir := findAbsPath(m.currentDir, selectedName)

			entries, err := os.ReadDir(nextDir)
			if err != nil {
				fmt.Println(err.Error())
			}

			m.reLoadDir(nextDir, entries)

		case "left", "h", "H":

			parent := filepath.Dir(m.currentDir)
			entries, err := os.ReadDir(parent)
			if err != nil {
				fmt.Println(err.Error())
				return m, nil
			}

			m.reLoadDir(parent, entries)
		}
	}

	return m, nil
}

func (m model) View() tea.View {
	renderedText := ""

	for i, list := range m.filesPath {

		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		if isDir(m.currentDir, list) {
			list = fmt.Sprintf("%s%s%s", Blue, list, Reset)
		}

		renderedText += fmt.Sprintf("%s %s\n", cursor, list)
	}
	renderedText += "\nPress 'q' to quit.	| Press 'd' to quit with cd to selected directory.\n"

	return tea.NewView(renderedText)
}

func (m *model) AddTask(task string) error {
	m.filesPath = append(m.filesPath, task)
	if !slices.Contains(m.filesPath, task) {
		return fmt.Errorf("%sHas an error to create task%s\n", Red, Reset)
	}

	return nil
}
