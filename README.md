# File manager TUI (PoC application)

(Proof Of Concept) a Terminal User Interface File manager for managing file / directory use on terminal

# Installation

required - go version >= 1.25.4 - nerdfonts (for icons)

### build from source

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
![navigate](./examples/Navigate.mp4)

**You can navigate through directories with these keys**
- `arrow left`, `h` for left navigate or step back to parents directory
- `arrow up`, `j` for up navigate
- `arrow down`, `h` for down navigate
- `arrow right`, `l` for right navigate or enter to directory

#### Toggle **close/open** `shortcuts pane` 
![shortcut-pane-toggle](./examples/Toggle-shortcuts-pane.mp4)
shortcuts pane for show shortcut directory and bookmark directory
you can toggle close/open pane with `ctrl + b`

#### Switch `pane`
![switch-pane](./examples/switch-pane.mp4)
you can switch across pane with
- `1` for `shortcuts pane`
- `2` for `file explorer pane`
- `3` for `file/directory previewer pane`
or use your pointer click at pane to switch

#### Search file / directory
![search](./examples/Search.mp4)
you can search for file/directory in current directory by press `s` and type for search then `enter`

#### create file
![create-file](./examples/Create-file.mp4)
create file by just press `f` key. then type file name, file extension and `enter`

#### create directory
![create-directory](./examples/Create-directory.mp4)
create file by just press `m` key. then type directory name and `enter`

#### copy file / directory
![copy-action](./examples/copy.mp4)
space to select one file / directory or multiple what you want to copy then `c` copy
navigate to where you want then `v` to paste

#### move file / directory
![move-action](./examples/move.mp4)
space to select one file / directory or multiple what you want to copy then `m` cut
navigate to where you want then `v` to paste

#### delete file / directory
![delete-action](./examples/delete.mp4)
you can delete one file / directory with press `d` 
or delete multiple by `space` to what you want to select then `d` to delete
then choose your choice

#### rename file / directory
![rename-action](./examples/rename.mp4)
you can rename file / directory name with press `r` then rename and `enter`
