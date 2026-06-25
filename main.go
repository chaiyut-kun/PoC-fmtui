package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

const Reset = "\033[0m"
const Red = "\033[31m"
const Blue = "\033[34m"
const baseDir = "./files"

func main() {
	program := tea.NewProgram(InitialModel())
	m, err := program.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	_ = m
}
