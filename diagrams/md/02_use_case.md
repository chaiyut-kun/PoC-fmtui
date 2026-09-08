# 2. Use Case Diagram (Conceptual)

This diagram models the functional goals and capabilities provided to the user, organized by domain. It focuses purely on user objectives and business capabilities rather than specific keybindings or implementation mechanics.

---

## Conceptual Use Case Model

```mermaid
flowchart LR
    User(["👤 Terminal User"])

    subgraph WorkspaceManagement ["1. Workspace & Navigation"]
        UC_NavigateTree["Explore Directory Hierarchy"]
        UC_MultiTab["Manage Parallel Workspaces (Tabs)"]
        UC_SplitPane["Multi-Pane Focus (List / Preview / Sidebar)"]
        UC_ToggleHidden["Filter System & Hidden Files"]
    end

    subgraph FileOperations ["2. File Manipulation & Safety"]
        UC_Create["Create Files & Directories"]
        UC_Rename["Rename Item"]
        UC_SafeDelete["Delete with Trash Protection"]
        UC_Transfer["Transfer Files (Copy / Cut / Paste)"]
        UC_ResolveConflict["Resolve Overwrite Conflicts"]
        UC_Rollback["Undo / Redo Past Actions"]
    end

    subgraph SearchDiscovery ["3. Search & Discovery"]
        UC_FilterView["Instant In-View Content Filtering"]
        UC_DeepSearch["Deep Path Traversal & Ranked Finding"]
        UC_QuickAccess["Access Bookmarked Locations"]
    end

    subgraph InspectionControl ["4. Inspection & Access Control"]
        UC_PreviewContent["Preview File Content & Syntax"]
        UC_InspectMetadata["Inspect File Metadata & Attributes"]
        UC_ModifyPermissions["Modify Access Permissions (Chmod)"]
    end

    subgraph EnvironmentIntegration ["5. Environment Integration"]
        UC_ExternalEditor["Edit Content with External Editor"]
        UC_ContextShell["Launch Shell at Current Location"]
        UC_SynchronizeDirectory["Preserve Directory on Exit"]
    end

    %% User Connections
    User --> UC_NavigateTree
    User --> UC_MultiTab
    User --> UC_SplitPane
    User --> UC_ToggleHidden

    User --> UC_Create
    User --> UC_Rename
    User --> UC_SafeDelete
    User --> UC_Transfer
    User --> UC_Rollback

    User --> UC_FilterView
    User --> UC_DeepSearch
    User --> UC_QuickAccess

    User --> UC_PreviewContent
    User --> UC_InspectMetadata
    User --> UC_ModifyPermissions

    User --> UC_ExternalEditor
    User --> UC_ContextShell
    User --> UC_SynchronizeDirectory

    %% Relationships
    UC_Transfer -.->|<<extends>> on conflict| UC_ResolveConflict
    UC_SafeDelete -.->|<<includes>> record history| UC_Rollback
    UC_Create -.->|<<includes>> record history| UC_Rollback
    UC_Rename -.->|<<includes>> record history| UC_Rollback
    UC_ModifyPermissions -.->|<<includes>> record history| UC_Rollback
```

---

## Functional Capabilities Summary

| Capability Domain | User Goal | Business Value |
|---|---|---|
| **Workspace & Navigation** | Efficiently browse file hierarchies and manage multiple concurrent work contexts. | Reduces context switching and speeds up developer navigation across large projects. |
| **File Manipulation & Safety** | Perform routine file mutations with an ironclad safety net. | Protects against accidental data loss via Trash integration and multi-step Undo/Redo. |
| **Search & Discovery** | Find items instantly regardless of whether they are in the current directory or buried deep within the tree. | Combines zero-latency in-memory filtering with ranked deep-search algorithms. |
| **Inspection & Control** | Verify file contents and permissions without leaving the file manager. | Eliminates unnecessary tool invocations for read-only inspection. |
| **Environment Integration** | Seamlessly bridge between the TUI, preferred external text editors, and parent shell sessions. | Ensures the tool feels like an integral extension of the terminal environment. |
