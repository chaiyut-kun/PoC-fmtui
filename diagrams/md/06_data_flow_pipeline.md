# 6. Data Flow & Rendering Pipeline (Conceptual)

This diagram visualizes the end-to-end data transformation pipeline: from raw storage entries on disk, through in-memory caching and real-time query filtering, to responsive split-pane layout composition and terminal rendering.

---

## Conceptual Data Transformation Pipeline

```mermaid
flowchart TD
    subgraph StorageTier ["1. Storage & Ingestion Tier"]
        Disk[("Host Filesystem")] -->|Directory Read| RawEntries["Raw Filesystem Entries\n(Names, Types, Permissions)"]
        RawEntries --> FilterAttributes["System Attribute Filtering\n(Hidden & Excluded Items)"]
        FilterAttributes --> SortEntries["Deterministic Sorting\n(Folders First, then Files A-Z)"]
    end

    subgraph MemoryCacheTier ["2. In-Memory State Cache"]
        SortEntries --> MasterState["Master Directory State\n(In-Memory Cached Listing)"]
        MasterState -.-> BypassFilter{Search Active?}
    end

    subgraph FilterEngineTier ["3. Real-Time Query Filter Engine"]
        UserInput["User Types Search Query"] --> ActiveQuery["Active Search Query"]
        ActiveQuery --> MatchAlgorithm["In-Memory Matching Engine\n(Substring & Pattern Matching)"]
        MasterState --> MatchAlgorithm
        MatchAlgorithm --> FilteredSubset["Filtered Items Subset"]
        MatchAlgorithm --> ResetViewport["Reset Viewport & Cursor Indices"]
    end

    subgraph ViewProjectionTier ["4. View Projection & Windowing"]
        FilteredSubset --> ResolveActive{"Has Active Filter?"}
        BypassFilter -->|No| ResolveActive
        ResolveActive -->|Yes| UseFiltered["Project Filtered Subset"]
        ResolveActive -->|No| UseMaster["Project Master State"]
        UseFiltered & UseMaster --> ActiveWindow["Active Viewport Window\n(Visible Rows Based on Screen Height)"]
    end

    subgraph LayoutCompositionTier ["5. Layout Composition & Rendering Engine"]
        ActiveWindow --> LayoutMath["Responsive Geometry Calculator\n(Screen Dimensions & Pane Ratios)"]
        
        LayoutMath --> RenderPanes["Compose Parallel Panes:"]

        subgraph VisualPanes ["Visual Columns"]
            direction LR
            SidebarCol["Sidebar Column\n- Quick access bookmarks\n- Storage shortcuts"]
            ExplorerCol["Explorer Column\n- Highlighted active cursor\n- Type glyphs & file icons\n- Multi-selected markers"]
            PreviewCol["Preview Column\n- Syntax-highlighted code\n- File metadata & attributes\n- Directory hierarchy preview"]
        end

        RenderPanes --> SidebarCol & ExplorerCol & PreviewCol
        SidebarCol & ExplorerCol & PreviewCol --> StatusLine["Status & Telemetry Bar\n- Path breadcrumbs\n- Background progress\n- Error alerts & hints"]

        StatusLine --> ModalCheck{"Modal Overlay Active?\n(Confirmation / Dialog / Inspector)"}
        ModalCheck -->|Yes| LayerModal["Composite Centered Modal Layer"]
        ModalCheck -->|No| CombineFrame["Composite Final Screen Frame"]
        LayerModal --> CombineFrame
    end

    CombineFrame --> DisplayBuffer([Send Frame to Terminal Screen TTY])
```

---

## Core Pipeline Principles

1. **Zero-Latency In-Memory Filtering**: Keystrokes filter the in-memory master state without triggering disk I/O, ensuring instantaneous typing response even on large directories.
2. **Deterministic Layout Math**: All column widths, row heights, and viewport offsets are computed before string rendering, preventing line wrapping glitches or display tearing.
3. **Decoupled Preview Generation**: File preview generation and syntax highlighting occur independently of the main listing window, allowing large files to be inspected smoothly.
