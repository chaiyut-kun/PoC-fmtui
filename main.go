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

	if _, err := program.Run(); err != nil {
		err = fmt.Errorf("Sorry Amanai something is wrong %v", err.Error())
		os.Exit(1)
	}
}
