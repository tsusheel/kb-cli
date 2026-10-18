# Proposed Extension Specifications: Plugin Architecture & ThingsPad

This directory contains architectural proposals and design specifications for upcoming extensions that are **planned for future releases and not yet implemented in the core app**.

---

## Proposal Document Index

| Document | Topic | Description |
| :--- | :--- | :--- |
| [**01. Universal Storage Schema**](./01-universal-storage.md) | **Database & Scalability** | The Universal Envelope + Indexed Facets pattern (`plugin_entries` table), composite B-Tree indexes, and multi-plugin scale without migrations. |
| [**02. Plugin System Framework**](./02-plugin-system.md) | **Plugin Contracts** | Lifecycle, Go backend interface, Web ES module contracts, dynamic route registration, and non-destructive data preservation. |
| [**03. ThingsPad Touch Console**](./03-thingspad-touch-console.md) | **Mobile Interaction** | The 50/50 touch interface, translucent ghost autocomplete, 30% Selector Wheel, 70% Contextual Command Pad, and the "DO IT" commit boundary. |
| [**04. Gym Logging Plugin**](./04-gym-plugin.md) | **Domain Implementation** | Flagship Gym plugin: tokenized grammar (`bench 60x10 70x8`), structured JSON schema, Single-Day Workout View (`/gym`), and CLI commands. |

---

## Core Vision
Extend `kb-cli` into a **plugin-driven personal data and logging platform** where:
- The core provides infrastructure (storage, sync, search, navigation, mobile command surface).
- Independent plugins provide semantics (parsers, predictive autocomplete, domain views, specialized command pads).
- The primary mobile interaction model is **ThingsPad**: a touch-first command console replacing conventional mobile keyboards.
