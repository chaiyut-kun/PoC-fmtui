# fmtui Conceptual Diagrams (Markdown & Mermaid)

This folder contains the conceptual architectural and design diagrams for **fmtui**, authored in GitHub Flavored Markdown and Mermaid.

---

## Diagrams Directory Index

| Document | Category | Conceptual Focus |
|---|---|---|
| 🏛️ [01_system_architecture.md](./01_system_architecture.md) | **System Architecture** | 4-tier Layered Architecture: Presentation, Application, Domain Services, and Infrastructure/OS Integration. |
| 🎯 [02_use_case.md](./02_use_case.md) | **Use Case Model** | User functional capabilities across Workspace Navigation, Safe File Manipulation, Search, Inspection, and Tools. |
| ⚙️ [03_activity.md](./03_activity.md) | **Activity Workflows** | Decision logic: Async transfer with conflict resolution, Depth-bounded Quick Find, and Action rollback journal. |
| 🔄 [04_state_machine.md](./04_state_machine.md) | **State Machine** | Operational user mental states: Workspace Browsing (Panes), In-View Filter, Deep Search, Naming, and Subprocesses. |
| ⏱️ [05_sequence.md](./05_sequence.md) | **Sequence Lifecycles** | Chronological lifecycles: Shell directory handoff, External tool delegation ($EDITOR), and Background task progress. |
| 🌊 [06_data_flow_pipeline.md](./06_data_flow_pipeline.md) | **Data Flow & Rendering** | 5-tier pipeline: Storage Ingestion $\rightarrow$ In-Memory Cache $\rightarrow$ Real-Time Filter $\rightarrow$ View Projection $\rightarrow$ Multi-Pane Layout. |

---

> [!TIP]
> Self-contained, publication-grade HTML + SVG versions of these diagrams with Light/Dark mode toggling are available in [diagrams/html/](../html/).
