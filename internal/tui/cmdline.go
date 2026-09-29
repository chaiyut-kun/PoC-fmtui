package tui

import (
	"strings"
)

type cmdEntry struct {
	name string
	desc string
}

var cmdRegistry = []cmdEntry{
	{name: "bookmark", desc: "Bookmark menu"},
	{name: "new_item", desc: "Create new item input"},
	{name: "symlink", desc: "Create symlink for selected item"},
	{name: "jump_symlink", desc: "Jump to symlink target"},
	{name: "copy", desc: "Copy menu"},
	{name: "fileperm", desc: "File permission menu"},
	{name: "open", desc: "'Open with...' menu"},
	{name: "quit", desc: "Quit without change directory"},
	{name: "exit", desc: "Quit with change directory"},
	{name: "delete", desc: "Delete menu"},
	{name: "newtab", desc: "Create new file explorer tab"},
	{name: "shortcut_toggle", desc: "Toggle on/off shortcut panel"},
	{name: "metadata", desc: "File metadata inspector"},
	{name: "rename", desc: "Rename input"},
	{name: "hidden_toggle", desc: "Toggle on/off hidden file"},
	{name: "whoami", desc: "whoami"},
	{name: "pwd", desc: "pwd"},
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
