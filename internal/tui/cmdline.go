package tui

import (
	"strings"
)

type cmdEntry struct {
	name string
	desc string
}

var cmdRegistry = []cmdEntry{
	{name: "bookmark", desc: "Open bookmark menu"},
	{name: "new_item", desc: "Open create new item input"},
	{name: "symlink", desc: "Create symlink for selected item"},
	{name: "jump_symlink", desc: "Jump to symlink target"},
	{name: "copy", desc: "Open copy menu"},
	{name: "file_permission", desc: "Open file permission menu"},
	{name: "open", desc: "Open with menu"},
	{name: "quit", desc: "Quit without change directory"},
	{name: "exit", desc: "Quit with change directory"},
	{name: "delete", desc: "Open delete menu"},
	{name: "newtab", desc: "Create new file explorer tab"},
	{name: "toggle_shortcut", desc: "Toggle open/close shortcut pane"},
	{name: "metadata", desc: "Open file metadata inspector"},
	{name: "rename", desc: "Open rename input"},
	{name: "hidden_file", desc: "Toggle show/off hidden file"},
}

// findCmdSuggestion returns the first command matching the prefix, or empty.
func findCmdSuggestion(input string) (cmdEntry, bool) {
	if input == "" {
		return cmdEntry{}, false
	}
	input = strings.ToLower(input)
	for _, cmd := range cmdRegistry {
		if strings.HasPrefix(cmd.name, input) {
			return cmd, true
		}
	}
	return cmdEntry{}, false
}

// findExactCmd returns the command matching exactly.
func findExactCmd(input string) (cmdEntry, bool) {
	input = strings.ToLower(input)
	for _, cmd := range cmdRegistry {
		if cmd.name == input {
			return cmd, true
		}
	}
	return cmdEntry{}, false
}
