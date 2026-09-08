# fmtui System Architecture & Design Diagrams

This directory contains the complete conceptual architecture, behavioral models, workflows, and pipelines for the **fmtui** file manager. Diagrams are provided in dual formats:
- **`md/`**: Standard Markdown with GitHub Flavored [Mermaid](https://mermaid.js.org/) syntax for native previewing on GitHub, GitLab, and IDEs.
- **`html/`**: Publication-grade, self-contained HTML + inline SVG diagrams created with the [diagram-design](https://github.com/cathrynlavery/diagram-design) editorial grammar, supporting interactive Light and Dark mode toggles.

---

## Master Diagrams Index

| Diagram Topic | Markdown / Mermaid | Interactive Editorial HTML | Core Conceptual Focus |
|---|---|---|---|
| **01. System Architecture** | 📝 [md/01_system_architecture.md](./md/01_system_architecture.md) | 🌐 [html/01_system_architecture.html](./html/01_system_architecture.html) | Four-tier Layered Architecture: Presentation, Application & Orchestration, Domain Services, and Infrastructure Integration. |
| **02. Use Case Model** | 📝 [md/02_use_case.md](./md/02_use_case.md) | 🌐 [html/02_use_case.html](./html/02_use_case.html) | High-level user capabilities, safety boundaries, and workflow goals. |
| **03. Activity Workflows** | 📝 [md/03_activity.md](./md/03_activity.md) | 🌐 [html/03_activity_workflows.html](./html/03_activity_workflows.html) | Decision trees for Async File Transfer, Depth-bounded Quick Find, and Symmetric Undo/Redo rollback. |
| **04. State Machine** | 📝 [md/04_state_machine.md](./md/04_state_machine.md) | 🌐 [html/04_state_machine.html](./html/04_state_machine.html) | User operational modes: Workspace Browsing (Panes), In-View Filter, Deep Search, Naming, Confirmations, and Subprocess Delegation. |
| **05. Sequence Lifecycles** | 📝 [md/05_sequence.md](./md/05_sequence.md) | 🌐 [html/05_sequence.html](./html/05_sequence.html) | System interaction lifecycles: Parent shell directory sync (`d`), Subprocess delegation (`$EDITOR`), and Background telemetry streaming. |
| **06. Data & Rendering Pipeline** | 📝 [md/06_data_flow_pipeline.md](./md/06_data_flow_pipeline.md) | 🌐 [html/06_data_pipeline.html](./html/06_data_pipeline.html) | In-memory directory caching, zero-I/O live query filtering, viewport windowing, and multi-pane responsive composition. |

---

## Directory Organization

```
diagrams/
├── README.md                  # This master directory index
├── html/                      # Self-contained Editorial HTML/SVG diagrams
│   ├── 01_system_architecture.html
│   ├── 02_use_case.html
│   ├── 03_activity_workflows.html
│   ├── 04_state_machine.html
│   ├── 05_sequence.html
│   └── 06_data_pipeline.html
├── md/                        # Conceptual Markdown / Mermaid diagrams
│   ├── 01_system_architecture.md
│   ├── 02_use_case.md
│   ├── 03_activity.md
│   ├── 04_state_machine.md
│   ├── 05_sequence.md
│   ├── 06_data_flow_pipeline.md
│   └── README.md
└── images/                    # Reserved for static PNG/SVG export assets
```

---

## How to View

- **HTML Diagrams**: Open any `.html` file directly in your web browser (e.g. `google-chrome diagrams/html/01_system_architecture.html` or double click in file manager). Use the button in the top-right corner to toggle between **Light** and **Dark** modes.
- **Markdown Diagrams**: View directly on GitHub, VS Code, or any Markdown previewer with Mermaid support.
