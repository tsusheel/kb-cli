# 04. Gym Logging Plugin Specification (Proposal)

> [!NOTE]
> This document specifies the future Gym Logging plugin domain, grammar, and single-day view, planned for implementation on top of the plugin architecture.

## 1. Motivation & User Experience

The Gym plugin is the flagship planned use case of the plugin architecture. Its goal is simple:

> **Log a workout as quickly as possible from a phone while actively training at the gym.**

The user should never have to navigate tedious multi-step forms. The interaction is:
1. Select exercise (swipe the 30% Selector Wheel or type its alias).
2. Tap weight and reps on the 70% Numeric Pad (`70 × 8`).
3. Tap **[ DO IT ]** to persist.

---

## 2. Command Language & Parser Grammar

The Gym parser converts quick single-line syntax into structured workout data:

### Grammar Patterns
```text
<exercise-alias> <set-expression> [<set-expression> ...]
```

### Supported Syntax Examples
- **Standard Multi-Set:**
  ```text
  bench 60x10 70x8 70x7
  ```
  *(Parses to 3 sets of Bench Press: 60kg for 10 reps, 70kg for 8 reps, 70kg for 7 reps)*

- **Repeated Sets Multiplier (`<weight>x<reps>x<sets>`):**
  ```text
  squat 100x5x3
  ```
  *(Parses to 3 identical working sets of Squat: 100kg for 5 reps each)*

- **Decimal Weights & Dumbbells:**
  ```text
  incline db 22.5x10 22.5x8
  ```

- **Bodyweight & Plus Notation:**
  ```text
  pullups bw+10x6 bwx8
  ```

---

## 3. Structured Data Model

When saved to `plugin_entries`, each exercise entry uses the universal envelope:

```json
{
  "id": "01JABC1234567890",
  "plugin_id": "gym",
  "entity_type": "workout_exercise",
  "entry_date": "2026-10-18",
  "parent_id": "session_01JXYZ...",
  "status": "active",
  "num_val": 70.0,
  "search_text": "bench press chest 60x10 70x8 70x7",
  "data": {
    "exerciseId": "bench_press",
    "exerciseName": "Bench Press",
    "order": 1,
    "sets": [
      { "setNumber": 1, "weight": 60, "reps": 10, "type": "warmup" },
      { "setNumber": 2, "weight": 70, "reps": 8, "type": "normal" },
      { "setNumber": 3, "weight": 70, "reps": 7, "type": "normal" }
    ],
    "notes": "Felt light on warmup"
  },
  "created_at": "2026-10-18T18:20:00Z",
  "updated_at": "2026-10-18T18:20:00Z"
}
```

---

## 4. Single-Day Workout View (`/gym` & `/gym/:date`)

The primary visualization is the **Single-Day View**. Users think in terms of *"What did I do today?"*, not database IDs:

```text
┌─────────────────────────────────────────────────────────┐
│ ←  Prev Day          23 September 2026        Next Day →│
│                                                         │
│ CHEST & BACK                                            │
│ Total Volume: 4,820 kg  •  9 Sets  •  3 Exercises       │
├─────────────────────────────────────────────────────────┤
│                                                         │
│ 1. Bench Press                                   3 sets │
│ ─────────────────────────────────────────────────────── │
│ Set 1:  60.0 kg  ×  10 reps                   (Warmup)  │
│ Set 2:  70.0 kg  ×   8 reps                             │
│ Set 3:  70.0 kg  ×   7 reps                             │
│                                                         │
│ 2. Incline DB Press                              3 sets │
│ ─────────────────────────────────────────────────────── │
│ Set 1:  22.5 kg  ×  10 reps                             │
│ Set 2:  22.5 kg  ×   9 reps                             │
│ Set 3:  22.5 kg  ×   8 reps                             │
│                                                         │
│ 3. Lat Pulldown                                  3 sets │
│ ─────────────────────────────────────────────────────── │
│ Set 1:  50.0 kg  ×  12 reps                             │
│ Set 2:  55.0 kg  ×  10 reps                             │
│ Set 3:  55.0 kg  ×   8 reps                             │
│                                                         │
│ Session Notes                                           │
│ Felt strong today. Rested 2 mins between heavy sets.   │
└─────────────────────────────────────────────────────────┘
```
