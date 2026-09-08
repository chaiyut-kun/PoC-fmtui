# 1. System Architecture Diagram (Conceptual)

This diagram illustrates the conceptual architecture of the file manager, organized into distinct architectural layers and functional subsystems. It highlights the separation of concerns between presentation, orchestration, domain capabilities, and host platform integrations.

---

## Conceptual Layered Architecture

```mermaid
flowchart TD
    User(["👤 User (Keyboard / Mouse)"]) <--> Terminal["Terminal Display & TTY Environment"]

    subgraph PresentationLayer ["1. Presentation Layer"]
        Terminal <--> LayoutManager["Split-Pane Layout Manager\n(Workspace, Sidebar, Preview)"]
        LayoutManager <--> ViewEngine["UI Rendering Engine\n(Component Composition & Styling)"]
        ViewEngine <--> ModalManager["Modal Overlay System\n(Dialogues, Inspectors, Prompts)"]
    end

    subgraph ApplicationLayer ["2. Application & Orchestration Layer"]
        EventCoordinator["Event & Input Coordinator\n(Key Mapping, Mouse Events, Window Resizing)"]
        WorkspaceManager["Workspace & Tab Controller\n(Active Session, Cursor & Viewport State)"]
        TaskCoordinator["Background Task Coordinator\n(Asynchronous Job Lifecycle & Progress)"]
    end

    subgraph DomainLayer ["3. Domain Services Layer"]
        FileOpsService["File Operations Service\n(Safe Deletion, Staged Transfer, Metadata)"]
        SearchService["Search & Discovery Engine\n(Real-Time In-View Filter & Deep Path Traversal)"]
        HistoryService["History & Rollback Engine\n(Atomic Undo / Redo Journal)"]
        BookmarkService["Bookmark & Quick Access Service\n(Pinned Locations & Favorites)"]
    end

    subgraph InfrastructureLayer ["4. Infrastructure & Platform Integration"]
        FSBridge["Host Filesystem Gateway\n(POSIX File I/O, OS Trash System)"]
        ProcessBridge["Subprocess Execution Bridge\n(External Editor & Interactive Shell)"]
        ClipboardBridge["System Clipboard Service\n(Path & Content Buffer)"]
        ShellHandoff["Shell Session Synchronizer\n(Directory Preservation on Exit)"]
    end

    %% Presentation to Application
    Terminal <--> EventCoordinator
    EventCoordinator --> WorkspaceManager
    EventCoordinator --> TaskCoordinator
    WorkspaceManager <--> ViewEngine
    TaskCoordinator --> ViewEngine

    %% Application to Domain
    EventCoordinator --> FileOpsService
    EventCoordinator --> SearchService
    EventCoordinator --> HistoryService
    EventCoordinator --> BookmarkService

    %% Domain to Infrastructure
    FileOpsService <--> FSBridge
    FileOpsService <--> ClipboardBridge
    FileOpsService --> HistoryService
    SearchService <--> FSBridge
    BookmarkService <--> FSBridge

    %% Application to Infrastructure
    EventCoordinator --> ProcessBridge
    EventCoordinator --> ShellHandoff
    ProcessBridge <--> Terminal
```

---

## Architectural Principles

1. **Strict Layer Separation**: The Presentation Layer renders views strictly based on domain and application state; it never performs direct filesystem mutations.
2. **Safe Mutation & Recoverability**: Modifying operations are funneled through the File Operations Service, which logs actions to the History & Rollback Engine and routes deletions to the OS Trash.
3. **Decoupled Asynchronous Tasks**: Heavy operations (such as large file transfers or deep filesystem walks) are coordinated asynchronously to ensure terminal responsiveness at all times.
4. **Transparent Environment Integration**: The application acts as a good UNIX citizen, providing native handoffs to the user's shell and delegating complex editing to external tools.
