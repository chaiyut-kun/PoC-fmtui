- [x] copy file or dir name with "y" keys  
    - "y" then "p" to copy file or dir path
    - "y" then "n" to copy file or dir name
    - "y" then "e" to copy file with extension
    *copy features will copy to user system clipboard
    *first user will press "y" then show dialog display as overlap layout at center of terminal screen

- [x] pane screen at right to preview data in file or preview child file or dir in dir
- [x] [fixbug pane screen] preview the data by do not make screen has bug if the data in file is more than to show all in terminal screen just make the "scroll" features of preview pane
- [x] [change preview pattern] preivew only data that fits of screen (ignore overflow terminal screen data)

- [x] open Editor (as $EDITOR variable) with "o" keys 

- [x] press "[", "]" to switch active pane (file manager pane, preview pane)

- [x] delete action with "d" keys  and show dialog for option
	- with "p" permanent delete
	- with "n" normal delete
- [x] [fix normal delete] normal delete act like permanent delete (even press "u" error show cannot undo permanent delete)

- [x] undo action with "u" key such as
	- normal delete file or dir
	- rename
	- create

- [x] redo action with "ctrl+r" / "U" keys

- [x] show current path at top of left pane

- [x] manage permission file with chmod with "p" key, (with sub option, dialog layout)
	- "x" as can execute (chmod +x)

- [x] help menu with key "`"
	- open whole help menu page
	- exit with "`" or "esc"

- [x] mouse scroll pane & mouse click focus feature
	- scroll wheel up/down to scroll file cursor (left pane) or preview text (right pane)
	- click left/right pane to switch active pane focus

- [x] multi-tabs of file explorer pane
	- 't' to open new tab with current directory
	- 'w' to close current tab (if >1 tab)
	- '[' / ']' to switch active tab
	- top border header displays tab list `[folder1] | folder2` with active tab highlighted

- [x] preview pane 1/3 of screen size
	- left Explorer pane = 2/3 width
	- right Preview pane = 1/3 width

- [ ] open menu options with "o" key
	- "c" to open with VSCode (code)
	- "r" to reveal in GUI file explorer
	- "e" to open with $EDITOR
