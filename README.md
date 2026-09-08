# File manager TUI (PoC application)

(Proof Of Concept) a Terminal User Interface File manager for managing file / directory use on terminal

# Installation

required - go version >= 1.25.4 - nerdfonts (for icons)

### build from source
- clone project
```bash
git clone https://github.com/chaiyut-kun/PoC-fmtui
cd PoC-fmtui
```

```bash
go build -o fm
```

**You guys can move to go workspace directory binary**

```bash
mv fm $(go env GOPATH)/bin
```

### Set up wrapper function

this script use for change directory when exit application feature
**you can move script to `~/.bashrc` / `~/.zshrc`**

```bash
fm() {
    local tmp="${TMPDIR:-/tmp}/fmtui-dir"
    command fm "$@"
    if [ -f "$tmp" ]; then
        local dir
        dir=$(cat "$tmp")
        rm -f "$tmp"
        [ -d "$dir" ] && cd "$dir"
    fi
}
```

# Usage

type `fm` and `Enter`

## Examples

#### **Navigate via directories**
https://github.com/user-attachments/assets/619ede32-1c87-42a0-8b46-e140b5677205

**You can navigate through directories with these keys**
- `arrow left`, `h` for left navigate or step back to parents directory
- `arrow up`, `j` for up navigate
- `arrow down`, `h` for down navigate
- `arrow right`, `l` for right navigate or enter to directory

#### Toggle **close/open** `shortcuts pane` 
https://github.com/user-attachments/assets/4f3cb306-0968-4e18-a0aa-8786101d7c0e

shortcuts pane for show shortcut directory and bookmark directory
you can toggle close/open pane with `ctrl + b`

#### Switch `pane`
https://github.com/user-attachments/assets/3564051c-672a-4554-b616-cc6881c0c815

you can switch across pane with
- `1` for `shortcuts pane`
- `2` for `file explorer pane`
- `3` for `file/directory previewer pane`
or use your pointer click at pane to switch

#### Search file / directory
https://github.com/user-attachments/assets/1b5c0610-1892-46fd-a767-4101018ad877

you can search for file/directory in current directory by press `s` and type for search then `enter`

#### create file
https://github.com/user-attachments/assets/0ef862bb-9936-4dbf-86b8-201db5305c2b

create file by just press `f` key. then type file name, file extension and `enter`

#### create directory
https://github.com/user-attachments/assets/a407818a-29d7-407b-b754-307715d5fd19

create file by just press `m` key. then type directory name and `enter`

#### copy file / directory
https://github.com/user-attachments/assets/b4fb4d00-393d-4f3b-8742-d5a747cb912c

space to select one file / directory or multiple what you want to copy then `c` copy
navigate to where you want then `v` to paste

#### move file / directory
https://github.com/user-attachments/assets/cb9a0e4d-0fb7-4306-b71b-cee9e11e1650

space to select one file / directory or multiple what you want to copy then `m` cut
navigate to where you want then `v` to paste

#### delete file / directory
https://github.com/user-attachments/assets/879cc49d-87ca-4e4f-ba05-4de592db9906

you can delete one file / directory with press `d` 
or delete multiple by `space` to what you want to select then `d` to delete
then choose your choice

#### rename file / directory
https://github.com/user-attachments/assets/b5a82c0a-60be-4d57-b88a-c6df74bc3152

you can rename file / directory name with press `r` then rename and `enter`

#### Quick Search
https://github.com/user-attachments/assets/a19c3b3d-4cfd-4952-af14-c8f93220734b

you can use `ctrl + f` to open quick search inpired **Notion quick search**

# Terminal Emulators supports
- Gnome Terminal
- 
