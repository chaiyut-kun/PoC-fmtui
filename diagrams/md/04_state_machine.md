# 4. State Machine Diagram (Conceptual)

This diagram models the file manager's operational modes, user mental states, and interaction transitions. It abstracts away low-level flags in favor of clear functional states.

---

## Conceptual State Machine

```mermaid
stateDiagram-v2
    [*] --> WorkspaceBrowsing: Launch Application

    state WorkspaceBrowsing {
        [*] --> FileListingPane

        state FileListingPane {
            [*] --> NavigatingItems
            NavigatingItems --> NavigatingItems: Move selection / Change directory
            NavigatingItems --> NavigatingItems: Switch / Close workspace tabs
            NavigatingItems --> NavigatingItems: Toggle hidden items
        }

        state ContentPreviewPane {
            [*] --> ViewingContent
            ViewingContent --> ViewingContent: Scroll preview document
        }

        state QuickAccessSidebar {
            [*] --> BrowsingBookmarks
            BrowsingBookmarks --> FileListingPane: Jump to selected location
        }

        FileListingPane --> ContentPreviewPane: Switch focus to preview
        ContentPreviewPane --> FileListingPane: Switch focus to file list
        FileListingPane --> QuickAccessSidebar: Open sidebar
        QuickAccessSidebar --> FileListingPane: Close sidebar / Dismiss
    }

    %% Search & Filter Modes
    WorkspaceBrowsing --> InViewFilterMode: Trigger instant filter
    state InViewFilterMode {
        [*] --> FilteringEntries
        FilteringEntries --> FilteringEntries: Type query (live update)
        FilteringEntries --> WorkspaceBrowsing: Accept selection or clear
    }

    WorkspaceBrowsing --> DeepDiscoveryMode: Trigger deep search
    state DeepDiscoveryMode {
        [*] --> ComposingQuery
        ComposingQuery --> SelectingResults: Navigate into matches
        SelectingResults --> ComposingQuery: Resume query editing
        SelectingResults --> WorkspaceBrowsing: Confirm path selection (jump)
        ComposingQuery --> WorkspaceBrowsing: Cancel search
    }

    %% Creation & Renaming
    WorkspaceBrowsing --> ItemNamingMode: Trigger create or rename
    state ItemNamingMode {
        [*] --> EditingName
        EditingName --> WorkspaceBrowsing: Confirm name (commit change)
        EditingName --> WorkspaceBrowsing: Cancel editing
    }

    %% Safety & Attributes
    WorkspaceBrowsing --> SafetyConfirmation: Request destructive action (delete / overwrite)
    state SafetyConfirmation {
        [*] --> AwaitingUserDecision
        AwaitingUserDecision --> WorkspaceBrowsing: Confirm action (execute & record history)
        AwaitingUserDecision --> WorkspaceBrowsing: Reject / Cancel
    }

    WorkspaceBrowsing --> AttributeInspection: Open metadata or permission editor
    state AttributeInspection {
        [*] --> InspectingProperties
        InspectingProperties --> WorkspaceBrowsing: Apply changes or dismiss
    }

    WorkspaceBrowsing --> HelpReference: Request help guide
    state HelpReference {
        [*] --> BrowsingHelpTopics
        BrowsingHelpTopics --> WorkspaceBrowsing: Dismiss help
    }

    %% External Delegation
    WorkspaceBrowsing --> DelegatedExternalSession: Launch editor or shell session
    state DelegatedExternalSession {
        [*] --> ExternalProcessActive: Terminal control transferred
        ExternalProcessActive --> WorkspaceBrowsing: External process exits (refresh workspace)
    }

    %% Terminations
    WorkspaceBrowsing --> CleanTermination: Exit application
    WorkspaceBrowsing --> SynchronizedShellExit: Exit with directory handoff
    CleanTermination --> [*]
    SynchronizedShellExit --> [*]
```

---

## Operational Modes Summary

| Mode | Role & User Intent | Typical Transitions |
|---|---|---|
| **Workspace Browsing** | The primary exploration hub. Users navigate folders, inspect live file previews, manage tabs, and browse bookmarks. | Transitions freely between panes; acts as the launchpad for all other modes. |
| **In-View Filter Mode** | Zero-latency content discovery. Narrows the active folder's file list as characters are typed. | Returns to Browsing on confirmation or escape. |
| **Deep Discovery Mode** | Global search across project scopes. Provides two-stage interaction: query composition and ranked result selection. | Jumps directly to the selected item upon confirmation. |
| **Item Naming Mode** | Captures input for creating new files, new directories, or renaming items. | Commits filesystem changes or aborts cleanly. |
| **Safety Confirmation** | Prevents unintended data loss by confirming destructive deletions or file overwrite collisions. | Commits verified action to trash/disk or aborts. |
| **Attribute Inspection** | Allows deep examination of file properties, sizes, MIME types, and editing access permission flags. | Applies updated permissions or closes. |
| **Delegated External Session** | Temporarily pauses the file manager to give full terminal control to external tools ($EDITOR, interactive subshell). | Automatically resumes the file manager and syncs file changes upon process exit. |
| **Synchronized Shell Exit** | Exits the TUI while signaling the parent shell to update its current working directory to match the TUI's location. | Terminates the application session. |
