# Architecture Diagrams (Mermaid)

Below are the system architecture diagrams representing the modular packaging, layout math, and search filtering pipelines currently implemented in **fmtui**.

---

## 1. Modular Package Architecture

This diagram visualizes how the Bubble Tea runtime, state containers, and execution systems are split across modular files within the same `tui` package.

```mermaid
flowchart TD
    terminal["Terminal Window (I/O)"] <--> tea["Bubble Tea Program (tea.Program)"]
    
    subgraph CMD ["Entrypoint Package (main)"]
        main["cmd/fm/main.go"]
    end
    
    subgraph TUI ["Application UI Package (tui)"]
        model["model.go\n- model struct\n- TabState struct\n- InitialModel()"]
        update["update.go\n- Update(msg)\n- Key Bindings\n- Subprocess Exec"]
        view["view.go\n- View() tea.View\n- Pane Layouts\n- Lipgloss Borders"]
        utils["utilities.go\n- reLoadDir()\n- applySearchFilter()\n- File Operations\n- Undo/Redo stack"]
        consts["constant.go\n- Icons\n- Colors"]
        bookmarks["bookmarks.go\n- bookmarks.json IO"]
    end

    main -->|imports| TUI
    main -->|instantiates| model
    tea <-->|invokes| update
    tea -->|renders via| view
    update <-->|reads/writes| model
    view <-->|reads| model
    update & view & model -->|calls| utils
    utils -->|reads icons/colors| consts
    model -->|loads bookmarks| bookmarks
    
    subgraph OS ["Operating System & System State"]
        utils <--> FS["Filesystem\n(os.ReadDir, Stat, Create, Rename, Delete)"]
        utils <--> Clip["Clipboard\n(atotto/clipboard)"]
    end
```

---

## 2. Search and Filter Pipeline

This flowchart outlines the real-time search filtering loop. The application avoids re-reading the filesystem directory on every keystroke by applying filters against the cached `t.filesPath` into a dynamic `t.filteredFiles` slice.

```mermaid
flowchart TD
    User["User Types Char in Search Mode"] -->|triggers Update()| KeyMsg["tea.KeyMsg"]
    KeyMsg -->|appends char| SearchText["m.searchText"]
    SearchText -->|calls| Filter["applySearchFilter()"]
    
    subgraph Cache ["TabState Filtering Engine"]
        Filter -->|checks query against| FilesPath["t.filesPath (Full Directory Cache)"]
        FilesPath -->|matches contain query| Filtered["t.filteredFiles (Muted Search Results)"]
    end
    
    Filter -->|resets cursor and scroll| ResetScroll["t.cursor = 0\nt.scrollOffset = 0"]
    Filtered -->|returns| VisibleFiles["t.visibleFiles() helper"]
    
    VisibleFiles -->|determines active listing| View["view.go / View() Renders Filtered List"]
    VisibleFiles -->|targets file for action| Ops["update.go / File Operations (Delete, Rename, Open)"]
```

---

## 3. Shell Integration & Exit Directory Handoff

This sequence diagram details the interaction between the shell wrapper function `fm()`, the handoff file `/tmp/fmtui-dir`, and the **fmtui** Go program.

```mermaid
sequenceDiagram
    actor User
    participant Shell as User's Shell (bash/zsh)
    participant HandoffFile as Temp File (/tmp/fmtui-dir)
    participant App as fmtui App (Binary compiled as fm)

    User->>Shell: Runs fm() shell wrapper function
    activate Shell
    Shell->>App: Launches 'command fm "$@"'
    activate App
    App->>User: Renders TUI interface
    
    alt User quits normally ('q')
        User->>App: Press 'q'
        App->>HandoffFile: Clears temp file (if any exists)
        App-->>Shell: Exits
    else User exits and changes shell directory ('d')
        User->>App: Press 'd'
        App->>HandoffFile: Writes currentDir path
        App-->>Shell: Exits
    end
    deactivate App

    Shell->>HandoffFile: Checks and reads target directory path
    Note over Shell: If target directory path exists in file:
    Shell->>Shell: cd to target directory
    Shell->>HandoffFile: Deletes temp file
    Shell-->>User: Returns control in new directory
    deactivate Shell
```
