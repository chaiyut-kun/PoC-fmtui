# 5. Sequence Diagrams (Conceptual)

This document presents system-level sequence diagrams illustrating the chronological interactions between the user, the file manager application, the host environment, and background tasks.

---

## 1. Shell Environment Synchronization on Exit

Illustrates how the file manager communicates its working context back to the user's parent shell upon termination.

```mermaid
sequenceDiagram
    autonumber
    actor User
    participant Shell as Parent Terminal Shell
    participant App as File Manager Application
    participant Handoff as Session State Handoff
    participant Storage as Host Filesystem

    User->>Shell: Invokes file manager command
    activate Shell
    Shell->>Handoff: Clears any previous session handoff state
    Shell->>App: Boots application process
    activate App

    App->>User: Displays interactive workspace (List, Preview, Status)

    loop Workspace Navigation
        User->>App: Navigates directories and workspaces
        App->>Storage: Queries directory entries & file metadata
        Storage-->>App: Returns file data
        App->>User: Renders updated view
    end

    alt Scenario A: Standard Termination
        User->>App: Requests standard exit
        App-->>Shell: Exits with success status
        deactivate App
        Shell->>Handoff: Checks for directory handoff signal (None found)
        Note over Shell: Remains in original working directory
    else Scenario B: Synchronized Directory Exit
        User->>App: Requests exit with directory synchronization
        App->>Handoff: Writes active working directory path
        App-->>Shell: Exits with success status
        deactivate App
        Shell->>Handoff: Checks for directory handoff signal (Signal Found!)
        Shell->>Handoff: Reads target directory path & clears state
        Shell->>Shell: Updates working directory to target path
    end

    Shell-->>User: Returns prompt in appropriate working directory
    deactivate Shell
```

---

## 2. External Subprocess Delegation & Workspace Resynchronization

Models the lifecycle of handing off terminal control to external tools (such as text editors or interactive shell sessions) and cleanly resuming.

```mermaid
sequenceDiagram
    autonumber
    actor User
    participant App as File Manager Application
    participant TTY as Terminal TTY Environment
    participant Tool as External Tool ($EDITOR / Shell)
    participant Storage as Host Filesystem

    User->>App: Triggers action requiring external tool (Edit file / Open subshell)
    activate App

    App->>TTY: Releases terminal raw mode & restores standard screen
    App->>Tool: Spawns external process attached to terminal I/O
    activate Tool
    deactivate App

    Note over Tool,User: User works directly within external tool
    Tool->>User: Displays editor / interactive shell prompt
    User->>Tool: Performs edits, commands, and saves changes
    Tool->>Storage: Reads and writes modified file data
    User->>Tool: Exits tool session

    Tool-->>App: Process terminates & returns exit code
    deactivate Tool

    activate App
    App->>TTY: Re-acquires terminal raw mode & alternate screen buffer
    App->>Storage: Rescans active directory to detect changes
    Storage-->>App: Returns refreshed directory state
    App->>App: Rebuilds view and updates file preview
    App->>User: Renders fresh workspace with current disk state
    deactivate App
```

---

## 3. Asynchronous Background Task & Real-Time Feedback

Visualizes decoupled background execution (e.g. large file copying or network transfers) with real-time UI telemetry and cancellation.

```mermaid
sequenceDiagram
    autonumber
    actor User
    participant UI as User Interface Controller
    participant Coordinator as Task Coordinator
    participant Worker as Background Task Worker
    participant Storage as Host Filesystem

    User->>UI: Initiates heavy file operation (e.g. Transfer / Paste)
    activate UI

    UI->>Coordinator: Schedules background job
    activate Coordinator
    Coordinator->>Worker: Dispatches worker execution
    activate Worker
    Coordinator-->>UI: Confirms job started (Job ID, initial status)
    deactivate Coordinator

    UI->>User: Displays active progress bar in status area

    loop Periodic Progress Telemetry
        Worker->>Storage: Performs file read/write chunk
        Worker->>Coordinator: Emits progress update (percentage, transfer rate)
        Coordinator->>UI: Delivers progress event
        UI->>User: Updates progress visualization in real time
    end

    opt User Cancels Task
        User->>UI: Requests job cancellation
        UI->>Coordinator: Issues cancellation signal
        Coordinator->>Worker: Signals interruption token
        Worker->>Storage: Cleans up partial/incomplete targets
        Worker-->>Coordinator: Confirms cancellation
    end

    Worker->>Coordinator: Signals completion (Success or Error status)
    deactivate Worker

    activate Coordinator
    Coordinator->>UI: Dispatches task finished event
    deactivate Coordinator

    UI->>Storage: Re-queries directory contents
    Storage-->>UI: Returns updated directory state
    UI->>User: Renders final file list and displays completion notice
    deactivate UI
```
