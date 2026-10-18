# 03. ThingsPad & Mobile Touch Command Console (Proposal)

> [!NOTE]
> This document is an interaction design proposal for the upcoming mobile command surface (ThingsPad) and is not yet part of the active UI.

## 1. Concept & Philosophy

Traditional mobile forms are cumbersome and frustrating in active environments like the gym. Navigating nested menus (`Gym → Chest → Bench Press → Add Set → Weight → Reps → Save`) breaks focus. On the other hand, squeezing an actual Linux terminal keyboard onto a phone screen results in tiny keys and frequent typos.

**ThingsPad** is a **touch-first command console**. It is designed around the philosophy:

> **Think → Tap / Type → DO IT**

The user never deals with an ordinary phone keyboard. The application controls the entire touch surface, dynamically guiding the user into valid input states while keeping entry speed near instantaneous.

---

## 2. Visual Architecture: The 50/50 Screen Split

The viewport is divided horizontally into two equal halves:

```ui
┌─────────────────────────────────────────────────────────┐
│ UPPER 50%: COMMAND STATE & PREDICTIVE OUTPUT            │
│                                                         │
│  [GYM] 2026-10-18                                       │
│                                                         │
│  Bench Press                                            │
│  70 × 8   70 × 8   [75 × 6]                             │
│                    ↳ Translucent ghost continuation     │
│                                                         │
│  Status: Valid set expression ready                     │
├─────────────────────────────────────────────────────────┤
│ LOWER 50%: CONTEXTUAL TOUCH SURFACE (THINGSPAD)         │
│ ┌──────────────────────┬──────────────────────────────┐ │
│ │ 30% SELECTOR WHEEL   │ 70% CONTEXTUAL COMMAND PAD   │ │
│ │                      │                              │ │
│ │    Incline DB Press  │   [ 7 ]   [ 8 ]   [ 9 ]      │ │
│ │  → Bench Press   ←   │   [ 4 ]   [ 5 ]   [ 6 ]      │ │
│ │    Cable Fly         │   [ 1 ]   [ 2 ]   [ 3 ]      │ │
│ │    Chest Press       │   [ . ]   [ 0 ]   [ × ]      │ │
│ │                      │                              │ │
│ │                      │  [ESC]   [CLEAR]  [ DO IT ]  │ │
│ └──────────────────────┴──────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

---

## 3. Upper Half: State, Output & Ghost Autocomplete

The upper pane is a visual projection of the active command state:
1. **Mode & Context Indicator:** Displays active plugin (`GYM`), target date (`2026-10-18`), and active exercise.
2. **Current Composition Stream:** Shows the sets or command tokens entered so far.
3. **Translucent Ghost Continuation:** Predicts what the user will type next based on historical workouts.
   - Example: If the user typed `7`, the screen shows `7[0 × 8]` in a dimmed, translucent style. A single tap accepts the prediction.
4. **Validation Feedback:** Highlights syntax errors in real-time before the user can commit.

---

## 4. Lower Half: ThingsPad Dual Surface

The lower half is an application-controlled touch surface divided into a **30% Selector Wheel** and a **70% Contextual Command Pad**.

### 4.1 The 30% Selector Wheel
- A vertical-scrolling carousel with smooth snap-to-center physics.
- The item in the exact vertical center is the active selection.
- **Dynamic Content:** The wheel adapts to the current plugin mode:
  - In `GYM`: Shows exercises (filtered by recent usage or muscle group).
  - In `COMMAND`: Shows available actions (`sync`, `today`, `log`, `search`).
  - In `FINANCE`: Shows expense categories (`food`, `transit`, `gear`).
- Swiping the wheel updates the upper pane in real time without requiring a tap.

### 4.2 The 70% Contextual Command Pad
The right-hand pad changes its layout and keybindings depending on what input is expected next:

#### Layout A: Numeric Set Pad (Mode: `LOG_SET`)
Used when entering weight and reps:
```text
┌───────────┬───────────┬───────────┐
│     7     │     8     │     9     │
├───────────┼───────────┼───────────┤
│     4     │     5     │     6     │
├───────────┼───────────┼───────────┤
│     1     │     2     │     3     │
├───────────┼───────────┼───────────┤
│     .     │     0     │     ×     │
├───────────┴───────────┴───────────┤
│   [ESC]       [SPACE]     [ENTER] │
└───────────────────────────────────┘
```

#### Layout B: Command / Action Pad (Mode: `ROOT`)
Used for quick navigation and global actions:
```text
┌───────────┬───────────┬───────────┐
│   TODAY   │   YEST    │   TOMOR   │
├───────────┼───────────┼───────────┤
│   NOTE    │   TODO    │   LOG     │
├───────────┼───────────┼───────────┤
│   FIND    │   STATS   │   SYNC    │
├───────────┴───────────┴───────────┤
│   [ESC]       [CLEAR]     [DO IT] │
└───────────────────────────────────┘
```

---

## 5. First-Class Control Boundary: "DO IT" vs. "SYNC"

ThingsPad enforces two distinct boundaries to eliminate cognitive overhead:

### The "DO IT" Action
- **Primary Commit Boundary:** When the user taps `[ DO IT ]`, the current composition is validated, converted into a structured entry, and written directly to local storage (SQLite or local IndexedDB cache).
- **Zero Network Delay:** Execution completes in under 5ms with haptic confirmation and a momentary green flash.
- **Immediate Reset:** The input resets and prepares for the next set immediately.

### The "SYNC" Action
- **Completely Decoupled:** Sync never blocks logging. If there is no cellular connection in the gym, the user continues logging sets offline.
- Tapping `[ SYNC ]` pushes all unsynced local rows to the cloud PostgreSQL database when connectivity is restored.
