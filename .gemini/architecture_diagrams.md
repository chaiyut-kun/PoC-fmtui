# Architecture Diagrams (Mermaid)

Below are the system architecture diagrams derived from [Plan.MD](file:///data/coding/Go/tui/list2/Plan.MD) represented in Mermaid markdown.

---

## 1. Current Architecture

This diagram visualizes the existing codebase design where bubbletea components directly interact with filesystem helpers and external dependencies.

```mermaid
flowchart TD
    terminal["Terminal (I/O)"] <--> program["Bubble Tea Program (tea.Program)"]
    
    subgraph App ["Application Engine (main.go & model.go)"]
        program <--> update["Update(msg)"]
        program --> view["View()"]
        update --> state["Model State\n- currentDir\n- filesPath\n- cursor\n- selected\n- showHidden\n- inputMode / inputText"]
        state --> adjustScroll["adjustScroll()"]
    end
    
    subgraph Utils ["Helpers (utilities.go)"]
        update & state <--> utils["reLoadDir()\nreloadCurrentDir()\nisDir()\nfindAbsPath()\nCheckDirFilesExt()\ncopyToClipboard()"]
    end
    
    utils <--> FS["Filesystem\n(os.ReadDir, os.Stat, os.Create, os.Mkdir)"]
    utils <--> Clip["Clipboard / Temp File\n(atotto/clipboard, /tmp/fmtui-dir)"]
```

---

## 2. Target Layered Architecture

This diagram shows the proposed package and layered design following a clean architecture style (Presentation -> Application -> Domain -> Infrastructure).

```mermaid
flowchart TD
    subgraph TerminalUI ["Terminal UI (Bubble Tea)"]
        direction TB
        View["View()"]
        Update["Update()"]
    end

    subgraph Presentation ["Presentation Layer"]
        direction TB
        TUIModel["tui/model\n(input mode, cursor, scroll, view state)"]
    end

    subgraph Application ["Application Layer"]
        direction TB
        AppService["app/service\n(LoadDirectory, ToggleHidden, CreateFile, CreateDir, CopySelectedPath)"]
    end

    subgraph Domain ["Domain Layer"]
        direction TB
        DomainModel["domain/model\n(CurrentDir, Entries, Cursor, Selection, ShowHidden, Error)"]
        EntryStruct["Entry struct\n(Name, Path, IsDir, IsHidden, Kind)"]
    end

    subgraph Infrastructure ["Infrastructure Layer"]
        direction TB
        FSInfra["infra/filesystem\n(ReadDir, Stat, CreateFile, CreateDir)"]
        ClipInfra["infra/clipboard\n(Copy)"]
        HandoffInfra["infra/handoff\n(WriteChosenDir, ClearChosenDir)"]
    end

    TerminalUI --> Presentation
    Presentation --> Application
    Application --> Domain
    Domain --> Infrastructure
    Infrastructure --> OS["Operating System\n(Filesystem, Clipboard, Processes)"]
```

---

## 3. Shell Integration & Handoff Sequence

This sequence diagram details the interaction between the shell function wrappers, the temporary file directory handoff (`/tmp/fmtui-dir`), and the TUI application when directory switching (`d`) or normal exits (`q`) occur.

```mermaid
sequenceDiagram
    actor User
    participant Shell as User's Shell (bash/zsh)
    participant HandoffFile as Temp File (/tmp/fmtui-dir)
    participant App as fmtui App

    User->>Shell: Runs fm() shortcut
    activate Shell
    Shell->>App: Launches fmtui
    activate App
    App->>User: Renders TUI list

    alt User quits normally ('q')
        User->>App: Press 'q'
        App->>HandoffFile: Clears temp file
        App-->>Shell: Exits
    else User quits and changes directory ('d')
        User->>App: Press 'd'
        App->>HandoffFile: Writes currentDir path
        App-->>Shell: Exits
    end
    deactivate App

    Shell->>HandoffFile: Reads target directory
    Shell->>Shell: cd to target directory
    Shell-->>User: Returns prompt in new directory
    deactivate Shell
```
