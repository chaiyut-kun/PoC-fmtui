# 3. Activity Diagrams (Conceptual)

This document contains conceptual activity diagrams modeling the business processes, decision logic, and operational workflows for key file manager capabilities.

---

## 1. Asynchronous File Transfer & Conflict Resolution

Illustrates the end-to-end process of transferring files, handling destination collisions, reporting background progress, and cleaning up source items.

```mermaid
flowchart TD
    Start([User Initiates Transfer]) --> CheckStaged{Are items staged in clipboard?}
    CheckStaged -->|No| NotifyEmpty([Notify User: Nothing to Transfer])
    CheckStaged -->|Yes| CheckConflict{Does destination item already exist?}

    CheckConflict -->|Yes| PromptConflict["Prompt User for Resolution Strategy"]
    PromptConflict --> UserChoice{User Resolution}
    UserChoice -->|Abort / Cancel| AbortTransfer([Cancel Transfer Operation])
    UserChoice -->|Overwrite| ProceedTransfer["Confirm Overwrite"]

    CheckConflict -->|No| ProceedTransfer

    ProceedTransfer --> InitTransfer["Initialize Background Transfer Task\n- Set initial progress\n- Arm cancellation token"]

    InitTransfer --> ForkTransfer["Execute Transfer Asynchronously"]

    subgraph ExecutionPhase ["Asynchronous Execution Phase"]
        ForkTransfer --> TransferLoop{More Data to Copy?}
        TransferLoop -->|Check Interrupt| CheckCanceled{Cancellation Requested?}
        CheckCanceled -->|Yes| RollbackPartial["Halt & Clean Partial Files"] --> TransferTerminated([Transfer Aborted])
        CheckCanceled -->|No| StreamData["Stream Data Chunk / Transfer File"]
        StreamData --> UpdateMetrics["Update Cumulative Progress Metric"]
        UpdateMetrics --> NotifyProgress["Emit Progress Telemetry"]
        NotifyProgress --> TransferLoop

        TransferLoop -->|Complete| CheckOpType{Transfer Mode}
        CheckOpType -->|Cut / Move| CleanSource["Remove Original Source Items"] --> NotifySuccess["Emit Completion Signal"]
        CheckOpType -->|Copy| NotifySuccess
        NotifySuccess --> TransferDone([Transfer Complete])
    end

    subgraph UIReaction ["User Interface Updates"]
        NotifyProgress -.-> RenderProgress["Render Dynamic Progress Indicator"]
        NotifySuccess -.-> RefreshView["Reload Active Directory Listing\nClear Staged Buffer\nDisplay Success Summary"]
        TransferTerminated -.-> ShowCanceled["Display Cancellation Notice"]
    end
```

---

## 2. Intelligent Deep Search & Relevance Ranking (Quick Find)

Details the algorithm for discovering candidate paths, pruning noisy branches, scoring matches, and ranking results.

```mermaid
flowchart TD
    StartSearch([User Requests Quick Find]) --> IdentifyRoots["Resolve Search Scopes\n(User Home, Active Project Root, System Data Stores)"]
    IdentifyRoots --> CheckQuery{Is search query empty?}
    CheckQuery -->|Yes| ShowBookmarks["Display Pinned Bookmarks Only"] --> PresentUI([Render Search Overlay])

    CheckQuery -->|No| InitTraverse["Initialize Depth-Bounded Tree Traversal"]

    subgraph TraversalAndScoring ["Tree Traversal & Scoring Engine"]
        InitTraverse --> DepthGuard{Current Depth > Threshold?}
        DepthGuard -->|Yes (Exceeds Limit)| PruneBranch[Prune Branch / Return]
        DepthGuard -->|No| InspectEntries["Inspect Current Directory Entries"]

        InspectEntries --> EvaluateEntry{For Each Item}
        
        EvaluateEntry --> IsIgnored{Is System Noise or Excluded?\ne.g. Version Control, Build Output, Caches}
        IsIgnored -->|Yes| SkipEntry[Skip Branch / Item]
        IsIgnored -->|No| CheckDirectory{Is Directory?}

        CheckDirectory -->|Yes| Descend["Descend to Subdirectory (Depth + 1)"]
        CheckDirectory -->|No| ScoreItem["Calculate Relevance Score"]

        Descend --> DepthGuard
        ScoreItem --> MatchQuality{Match Tier}
        MatchQuality -->|Exact Basename Match| Tier1["Assign High Priority Score (Tier 1)"]
        MatchQuality -->|Prefix Basename Match| Tier2["Assign Medium Priority Score (Tier 2)"]
        MatchQuality -->|Partial Basename Match| Tier3["Assign Standard Priority Score (Tier 3)"]
        MatchQuality -->|No Match| DiscardItem[Discard]

        Tier1 & Tier2 & Tier3 --> CollectCandidate["Register Candidate Result"]
        SkipEntry & DiscardItem & CollectCandidate --> NextEntry{More Entries?}
        NextEntry -->|Yes| EvaluateEntry
        NextEntry -->|No| CompleteTraversal[Finish Traversal]
    end

    CompleteTraversal --> RankResults["Stable Sort Candidates by Relevance Score"]
    RankResults --> LimitTop["Merge Bookmarks & Truncate to Top Matches"]
    LimitTop --> PresentUI
```

---

## 3. Action Rollback & History Management (Undo / Redo)

Models the atomic inversion of past actions and reciprocal journal tracking.

```mermaid
flowchart TD
    StartRollback([User Requests Undo or Redo]) --> CheckHistory{Is Action Journal Empty?}
    CheckHistory -->|Yes| NotifyNoAction([Notify User: Nothing to Rollback])

    CheckHistory -->|No| FetchRecord["Retrieve Latest Action from Active Journal"]
    FetchRecord --> ActionType{Classify Action Type}

    %% Delete Rollback
    ActionType -->|Item Deletion| RestoreTrash["Restore Item from Trash Back to Original Location"]
    RestoreTrash --> FormCreateRecord["Generate Reciprocal Creation Record"]

    %% Create Rollback
    ActionType -->|Item Creation| RemoveCreated["Safely Remove Created Item"]
    RemoveCreated --> FormDeleteRecord["Generate Reciprocal Deletion Record"]

    %% Rename Rollback
    ActionType -->|Item Rename| RevertName["Revert Item Path Back to Original Name"]
    RevertName --> FormRenameRecord["Generate Reciprocal Rename Record"]

    %% Permission Rollback
    ActionType -->|Permission Change| RevertMode["Restore Original Access Permissions"]
    RevertMode --> FormChmodRecord["Generate Reciprocal Permission Record"]

    %% Commit and Sync
    FormCreateRecord & FormDeleteRecord & FormRenameRecord & FormChmodRecord --> CommitRollback["Push Reciprocal Record to Opposing Journal\n(Undo Journal 🔁 Redo Journal)"]

    CommitRollback --> SynchronizeWorkspace["Synchronize Workspace State\n- Refresh Directory View\n- Recalculate Selections\n- Display Outcome Summary"]

    SynchronizeWorkspace --> EndRollback([Rollback Complete])
```
