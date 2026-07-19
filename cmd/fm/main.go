package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"file-list/internal/tui"
)

func main() {
	program := tea.NewProgram(tui.InitialModel())
	m, err := program.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	_ = m
}
