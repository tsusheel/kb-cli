# Knowledge Base Philosophy & Principles

`kb` is built on the premise that personal knowledge management should be friction-free, progressive, interconnected, and natively accessible to both humans and AI agents.

This document articulates the core philosophy, mental models, and architectural principles governing how notes are captured, organized, and evolved.

---

## 1. Core Principles

### 1.1 Atomic Yet Substantive (`Note` + `Note Flesh`)
- **The Summary (`note`)**: Every note begins with a clear, concise headline or thesis statement. It answers *"What is this thought?"* at a glance.
- **The Substance (`note_flesh`)**: The body contains the context, code snippets, reasoning, references, and nuance.
- **Why this separation matters**: In listing, fuzzy-finding, and AI tool selection, the headline provides immediate signal without cognitive overload, while the flesh preserves depth.

### 1.2 Intent-Driven Typing over Hierarchical Folders
Folders create rigid silos that decay over time. `kb` replaces folders with **Typing by Intent**:
- **`note`**: General atomic knowledge, thoughts, and insights.
- **`todo`**: Actionable tasks with optional target dates and completion statuses.
- **`project`**: Ongoing initiatives or milestones that aggregate related notes and tasks.
- **`idea`**: Early-stage sparks, hypotheses, and unexplored possibilities.
- **`concept`**: Mental models, definitions, and systemic abstractions.
- **`decision`**: Architecture decision records (ADRs), rationales, and evaluated trade-offs.
- **`til`**: "Today I Learned" — discrete discoveries and practical learnings.
- **`question`**: Open inquiries and hypotheses driving future research.
- **`resource`**: Curated references, articles, tools, and documentation.
- **`experiment`**: Empirical tests, logs, and findings.
- **`person`**: Collaborators, mentors, and subject-matter experts.

### 1.3 Progressive Refinement & Lifecycle
Knowledge is not static; it matures over time.
- **`raw`**: Fast, unedited capture to avoid losing fleeting ideas.
- **`refined`**: Structured, clear, and tagged for long-term retrieval.
- **`in-progress`**: Active projects or tasks currently being worked on.
- **`completed`**: Accomplished tasks or validated experiments.
- **`archived`**: Retired knowledge retained for historical context.

Notes can also track **`importance`** (1–5) and **`clarity`** (1–5), guiding both user review sessions and AI agents on what needs elaboration or immediate focus.

### 1.4 Clean Text & Relational Metadata (No Syntax Pollution)
- **Pure Markdown Content**: Note summaries and note flesh remain clean, human-readable prose without embedded `#tag` or `[[wiki-link]]` markup clutter.
- **Metadata in the Graph**: Tags, relationships, and links live in the relational database layer.
- **Asynchronous Structuring**: Rapid capture happens without tagging or linking. Linking and tagging are deliberate curation steps performed afterward—either by the user during triage or autonomously by AI agents via MCP.

### 1.5 Two-Speed Capture (Instant Jot vs. Deep Fleshing)
Human thought operates in two distinct modes:
1. **Speed 1 (Instant Fleeting Jot)**: Rapidly jotting down a 1-line thought or task in milliseconds without editor interruption (`kb jot "headline"` or `kb "headline"`).
2. **Speed 2 (Substantive Deep Dive)**: Opening the editor (`$EDITOR`) to elaborate, structure, and write detailed `note_flesh` on new or existing notes (`kb add`, `kb edit <id>`, `kb flesh <id>`).

### 1.6 Temporal Stream & Promotion (Logs to Notes)
- **Daily Stream (`kb log`)**: Micro-thoughts, work logs, and status updates are recorded in a lightweight chronological timeline throughout the day.
- **Log-to-Note Promotion**: Not every thought starts as a full note. When a micro-log develops substance, it can be promoted (`kb promote <log_id>`) into a standalone, typed `models.Note` with its own lifecycle and flesh.

### 1.7 Human + AI Symbiosis (Chaotic Capture -> Structured Knowledge)
`kb` is designed from the ground up as a shared workspace between the human and AI assistants:
- **Chaotic Human Capture, Structured AI Synthesis**: The human captures freely and chaotically in `raw` state. AI agents (via MCP) analyze, link related concepts, suggest tags, and assist in triaging the inbox.
- **Model Context Protocol (MCP)**: AI agents connect natively via standard protocol tools (`list_notes`, `get_note`, `search_notes`, `create_note`, `update_note`, `delete_note`).
- **Auditability & Safe Deletion**: Deletions are strictly non-destructive soft deletes with timestamps (`deleted_at`) and attribution reasons (`deleted_note` e.g., *"deleted by AI: task completed"*).

### 1.8 Local-First, Fast & Autonomous
- **Self-Contained**: Powered by embedded SQLite schemas with zero external server dependencies.
- **Ergonomic Ergonomics**: 7-character short IDs, editor integration (`$EDITOR`), fuzzy-finding (`kb open`), and flexible date parsing (`today`, `tomorrow`, `+3d`, `monday`).

---

## 2. Note-Taking & Triage Guidelines

1. **Jot First, Flesh Out Later**: When in the flow state, jot down thoughts instantly without opening an editor. Elaborate `note_flesh` when in reflective mode.
2. **Keep the Prose Clean**: Avoid embedding custom markup inside the note body. Use relational links and tags.
3. **Log the Stream, Promote What Matters**: Use daily logs for micro-context. Promote impactful logs into permanent notes.
4. **Regular Triage**: Keep `raw` capture friction-free by using inbox review and AI synthesis to move notes into `refined`, `in-progress`, or `completed`.
5. **Let the Graph Connect Ideas**: Use semantic link types (`related_to`, `depends_on`, `part_of`, `inspired_by`, `supports`, `contradicts`) to preserve context across projects.

---

## 3. Guiding Architectural Decisions

Code additions and refactorings in `kb` should align with these guidelines:
- **Never perform destructive deletes**: All entity removals must preserve an audit trail via soft deletion columns (`deleted_at`, `deleted_note`).
- **Maintain clean content boundaries**: Never inject custom bracketed link syntax or tags into note markdown text; keep metadata relational.
- **Maintain CLI and MCP parity**: Any feature available via the CLI should be intuitively accessible to AI agents via MCP tools, and vice versa.
- **Preserve zero-configuration reliability**: Database schemas and configurations initialize automatically on first run without friction.
- **Optimize for readability and speed**: Queries must remain fast and lean, favoring lightweight structured JSON and efficient SQLite indexing.
