# 02. Plugin System Specification (Proposal)

> [!NOTE]
> This document is an architecture proposal for the future plugin extension system and is not yet implemented in the core app.

## 1. Core Principles

1. **Pluggable Semantics, Universal Infrastructure:** The core handles storage, sync, search, and navigation; plugins define parsing, state transitions, domain schemas, and specialized user interfaces.
2. **Durable Data:** Disabling or uninstalling a plugin **never deletes data**. Entries stored by that plugin remain safely stored in `plugin_entries` and can still be rendered in a generic fallback view. When the plugin is re-enabled, specialized interpretation resumes instantly.
3. **Decoupled Contracts:** Plugins do not access raw internals. They interact with the core through a stable API surface exposed in both Go (backend) and JavaScript (frontend).

---

## 2. Backend Plugin Contract (Go)

Backend plugins provide command-line hooks, entry validation, and optional background routines:

```go
package plugins

import (
	"time"
	"github.com/spf13/cobra"
)

// Plugin represents the server/CLI extension contract.
type Plugin interface {
	ID() string
	Name() string
	Version() string
	Description() string

	// Register binds plugin capabilities to the core framework
	Register(ctx *PluginContext) error
}

type PluginContext struct {
	// CLI Command Tree
	RootCmd *cobra.Command

	// Entry Storage API
	Storage EntryStore

	// Search Registry
	Search SearchRegistry
}

// EntryStore exposes core universal persistence to plugins
type EntryStore interface {
	Create(entry *PluginEntry) error
	GetByID(id string) (*PluginEntry, error)
	ListByDate(pluginID string, date time.Time) ([]PluginEntry, error)
	Query(filter EntryFilter) ([]PluginEntry, error)
	SoftDelete(id string, reason string) error
}
```

---

## 3. Frontend Plugin Contract (Web ES Module)

Frontend plugins are self-contained ES modules that register themselves with the client-side `PluginManager`:

```javascript
/**
 * Frontend Plugin Interface
 */
export const ExamplePlugin = {
  // 1. Identity & Metadata
  id: 'gym',
  name: 'Gym Log',
  version: '1.0.0',
  icon: '🏋️',
  description: 'Fast, touch-first workout and exercise set tracking.',

  // 2. Client-Side Routing
  routes: [
    {
      path: '/gym',
      title: 'Gym — Today',
      component: GymDayViewComponent
    },
    {
      path: '/gym/:date',
      title: 'Gym — Workout Day',
      component: GymDayViewComponent
    }
  ],

  // 3. Parser & Tokenizer (Think -> Type -> DO IT)
  parser: {
    tokenize(rawText) {
      // Returns array of token objects
    },
    parse(tokens) {
      // Returns structured payload ready for DB
    },
    validate(tokens) {
      // Returns { isValid: boolean, error?: string, nextExpected: string }
    }
  },

  // 4. Composer & Predictive Autocomplete
  composer: {
    getGhostContinuation(currentText) {
      // Returns predictive ghost completion (e.g., 'ench press')
    },
    getSuggestions(currentText) {
      // Returns list of quick suggestions
    }
  },

  // 5. ThingsPad Configuration (Mobile Touch Surface)
  thingsPad: {
    initialMode: 'EXERCISE_SELECT',
    getSelectorItems(state) {
      // Returns array of items for the 30% Selector Wheel
      // e.g., ['Bench Press', 'Incline DB', 'Squat', 'Lat Pulldown']
    },
    getCommandPad(state) {
      // Returns layout definition for the 70% Command Pad
      // e.g., 'numeric_set' with keys ['7','8','9','4','5','6','1','2','3','.','0','×']
    },
    handleKeyAction(key, state) {
      // Custom key handler for command pad presses
    }
  },

  // 6. Generic Previews (Desktop Dual-Pane & Global Search)
  renderSummary(entry) {
    // Returns formatted HTML string for the global fuzzy finder
    return `<strong>${entry.data.exerciseName}</strong>: ${entry.data.sets.length} sets`;
  }
};
```

---

## 4. Plugin Discovery & Lifecycle

### 4.1 Configuration (`config.yaml`)
Plugins are managed in the user's local configuration:

```yaml
plugins:
  enabled:
    - gym
    - notes
    - reading
  disabled:
    - finance
```

### 4.2 CLI Commands
```bash
# List all discovered plugins and their status
kb plugin list

# Enable a plugin
kb plugin enable gym

# Disable a plugin (data remains preserved)
kb plugin disable gym
```
