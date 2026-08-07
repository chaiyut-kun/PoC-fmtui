package tui

type helpItem struct {
	heading bool
	key     string
	desc    string
}

var helpItems = []helpItem{
	{heading: true, desc: "Navigation & Panes"},
	{key: "j / k, Up / Down", desc: "Navigate files / scroll preview"},
	{key: "h / l, Left / Right", desc: "Go to parent dir / enter dir"},
	{key: "tab", desc: "Switch active pane"},
	{key: "[ / ]", desc: "Switch tab prev / next (or switch shortcuts/bookmarks tab in sidebar)"},
	{key: "t / w", desc: "New tab / close tab"},
	{key: "ctrl+b / \\", desc: "Toggle shortcut sidebar pane"},
	{key: "b", desc: "Open bookmark menu (add selected directory / CWD)"},
	{key: "d", desc: "Remove bookmark (only when inside Bookmarks tab)"},

	{heading: true, desc: "File Actions"},
	{key: "n", desc: "New item (name.ext = file, name or name/ = directory)"},
	{key: "a", desc: "Toggle select all / deselect all"},
	{key: "i", desc: "File metadata inspector"},
	{key: "K", desc: "Create symlink for selected item"},
	{key: "g", desc: "Jump / follow symlink target"},
	{key: "r", desc: "Rename file or directory"},
	{key: "d", desc: "Delete menu (p: permanent, n: trash)"},
	{key: "y", desc: "Copy menu (p: path, n: name, e: ext, s: tree)"},
	{key: "p", desc: "Permission menu (x: chmod +x/-x)"},
	{key: "o", desc: "Open menu [c: open code ., C: VSCode item, r: reveal file explorer, e: $EDITOR]"},
	{key: ".", desc: "Toggle hidden files"},

	{heading: true, desc: "Others"},
	{key: ":", desc: "Temporary switch to subshell (type exit or ctrl+d to return)"},
	{key: "e", desc: "Exit & cd to current dir in shell"},
	{key: "q / ctrl+c", desc: "Quit application"},
}
