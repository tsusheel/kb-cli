# Proposed Extension Specifications: Plugin Architecture & ThingsPad

This directory contains architectural proposals and design specifications for extensions and ecosystem integrations.

> [!NOTE]
> **Implementation Status Note:**
> - **Proposals 01 (Universal Storage) & 02 (Plugin System Framework)** have been successfully implemented and merged into the core `kb-cli` platform as of v0.1.12. See [09. Multi-Plugin Platform Architecture](../09-plugin-system.md) and [02. Active Database Schema & Models](../02-database-schema.md) for the active production documentation.
> - **Proposal 03 (ThingsPad Touch Console)** is an active UI proposal for upcoming mobile interface enhancements.
> - **Proposal 04 (Gym Logging Plugin)** serves as the canonical domain specification and scaffolding guide for the external gym plugin repository (`kb-plugin-gym`).

---

## Proposal Document Index

| Document | Topic | Status | Description |
| :--- | :--- | :--- | :--- |
| [**01. Universal Storage Schema**](./01-universal-storage.md) | **Database & Scalability** | **Implemented** in Core | The Universal Envelope + Indexed Facets pattern (`plugin_entries`), composite B-Tree indexes, and multi-plugin scale without migrations. |
| [**02. Plugin System Framework**](./02-plugin-system.md) | **Plugin Contracts** | **Implemented** in Core | Out-of-process JSON-RPC lifecycle, Go SDK contracts, Web host integration, dynamic CLI & MCP tools, and permission grants. |
| [**03. ThingsPad Touch Console**](./03-thingspad-touch-console.md) | **Mobile Interaction** | **Proposed** | The 50/50 touch interface, translucent ghost autocomplete, 30% Selector Wheel, 70% Contextual Command Pad, and the "DO IT" commit boundary. |
| [**04. Gym Logging Plugin**](./04-gym-plugin.md) | **Domain Implementation** | **External Repo Guide** | Flagship Gym plugin: tokenized grammar (`bench 60x10 70x8`), structured JSON schema, Single-Day Workout View (`/gym`), and CLI commands. |

---

## Core Vision
Extend `kb-cli` into a **plugin-driven personal data and logging platform** where:
- The core provides infrastructure (storage, sync, search, navigation, mobile command surface).
- Independent plugins provide semantics (parsers, predictive autocomplete, domain views, specialized command pads).
- The primary mobile interaction model is **ThingsPad**: a touch-first command console replacing conventional mobile keyboards.

