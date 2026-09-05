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

### 1.4 Networked Thought (Graph over Tree)
Insights rarely exist in isolation. `kb` treats knowledge as a directed semantic graph:
- **Explicit Relationships**: Notes link to other notes with clear semantic types:
  - `related_to`: Conceptual association.
  - `depends_on`: Prerequisites and blockers.
  - `part_of`: Component of a larger project or concept.
  - `inspired_by`: Intellectual genealogy or source ideation.
  - `supports` / `contradicts`: Dialectical reasoning and arguments.
  - `about` / `created_by`: Contextual attribution.
- **Cross-Cutting Tags**: Multi-dimensional tagging (`ai/mcp`, `golang/sqlite`, `reading/2026`) provides orthogonal discovery across types and areas.

### 1.5 Human + AI Symbiosis (Agent-First Architecture)
`kb` is designed from the ground up as a shared workspace between the user and AI assistants:
- **Model Context Protocol (MCP)**: AI agents connect natively via standard protocol tools (`list_notes`, `get_note`, `search_notes`, `create_note`, `update_note`, `delete_note`).
- **Auditability & Safe Deletion**: Deletions are strictly non-destructive soft deletes with timestamps (`deleted_at`) and attribution reasons (`deleted_note` e.g., *"deleted by AI: task completed"*).
- **Fast Search Retrieval**: SQLite FTS5 full-text indexing allows agents and humans to locate relevant context in milliseconds without scanning whole files.

### 1.6 Local-First, Fast & Autonomous
- **Self-Contained**: Powered by embedded SQLite schemas with zero external server dependencies.
- **Ergonomic Ergonomics**: 7-character short IDs, editor integration (`$EDITOR`), fuzzy-finding (`kb open`), and flexible date parsing (`today`, `tomorrow`, `+3d`, `monday`).

---

## 2. Note-Taking Guidelines

1. **Capture First, Organize Progressively**: Capture thoughts immediately in `raw` state. Structure and links can be added during review or with agent assistance.
2. **One Primary Idea Per Note**: If a note covers multiple unrelated topics, split them and connect them with `related_to` or `part_of` links.
3. **Make Titles Actionable or Descriptive**: Prefer `"Implement SQLite FTS5 for fuzzy search"` over `"Search feature"`.
4. **Link Decisions to Context**: When making architectural choices, link the `decision` note to the relevant `project` and `concept` notes.
5. **Use Target Dates Purposefully**: Set target dates on `todo` and `project` notes to maintain momentum across daily and weekly workflows.

---

## 3. Guiding Architectural Decisions

Code additions and refactorings in `kb` should align with these guidelines:
- **Never perform destructive deletes**: All entity removals must preserve an audit trail via soft deletion columns (`deleted_at`, `deleted_note`).
- **Maintain CLI and MCP parity**: Any feature available via the CLI should be intuitively accessible to AI agents via MCP tools, and vice versa.
- **Preserve zero-configuration reliability**: New database migrations, schema updates, or configurations must run automatically without breaking the user's workflow.
- **Optimize for readability and speed**: Queries must remain fast and lean, favoring lightweight structured JSON and efficient SQLite indexing.
