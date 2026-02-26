# UI Specification

## Overall Layout

The app uses a **two-tab** interface:

```
┌─────────────────────────────────────────────────────────┐
│  CourtDraw     [ Exercise Editor | Session Composer ]   │
├─────────────────────────────────────────────────────────┤
│                                                         │
│                   (active tab content)                   │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

## Tab 1: Exercise Editor

Inkscape/GIMP-style layout: court canvas in the center, tool panels docked around it.

```
┌─────────────────────────────────────────────────────────────┐
│  CourtDraw     [ ● Exercise Editor | Session Composer ]     │
├───────┬─────────────────────────────────────────┬───────────┤
│       │  Sequence: [1. Setup][2. Close-out][+]  │           │
│ TOOLS │                                         │ PROPS     │
│       │         ┌───────────────────┐           │           │
│ ──────│         │                   │           │ Label:    │
│ Players         │                   │           │ [A1    ]  │
│ [Atk] │         │   COURT CANVAS    │           │           │
│ [Def] │         │                   │           │ Role:     │
│ [Coach]         │   (FIBA / NBA)    │           │ [Attack▼] │
│ [PG]  │         │                   │           │           │
│ [SG]  │         │   (half / full)   │           │ Group:    │
│ [SF]  │         │                   │           │ [      ]  │
│ [PF]  │         └───────────────────┘           │           │
│ [C]   │                                         │ ────────  │
│ ──────│  [▶ Play] [⏸] [◀ Prev] [Next ▶]        │ Exercise  │
│ Actions         [Speed: 1x ▼]                   │ metadata  │
│ [Pass]│                                         │           │
│ [Drib]│                                         │ Name:     │
│ [Sprt]│                                         │ [Closeout]│
│ [Shot]│                                         │           │
│ [Scrn]│                                         │ Court:    │
│ [Cut] │                                         │ [FIBA  ▼] │
│ [C-O] │                                         │ [Half  ▼] │
│ [Cntst]                                         │           │
│ [Rev] │                                         │ Duration: │
│ ──────│                                         │ [15min  ] │
│ Access.                                         │           │
│ [Cone]│                                         │ Intensity:│
│ [Ladr]│                                         │ [●●●○   ] │
│ [Chair]                                         │           │
│ ──────│                                         │ Category: │
│ [🗑]  │                                         │ [Defense▼]│
│       │                                         │           │
│       │                                         │ Tags:     │
│       │                                         │ [close-out│
│       │                                         │  1v1, U13]│
├───────┴─────────────────────────────────────────┴───────────┤
│ Instructions (current sequence):                             │
│ • Coach passes to A1                                         │
│ • D sprints to close out on A1                               │
│ • Explode toward the ball handler, small steps on arrival    │
│ [+ Add instruction]                                          │
└──────────────────────────────────────────────────────────────┘
```

### Left Panel: Tool Palette

Grouped into sections:

**Players**
- Attacker, Defender, Coach
- Point Guard (1), Shooting Guard (2), Small Forward (3), Power Forward (4), Center (5)
- Queue (creates a line of waiting players)

**Actions**
- Pass, Dribble, Sprint
- Shot (layup / push-up / jump shot — sub-menu)
- Screen, Cut, Close-out, Contest, Reverse

**Accessories**
- Cone, Agility Ladder, Chair

**Delete tool** (trash icon)

### Center: Court Canvas

- Displays the court for the **current sequence**
- Court type set in exercise metadata (FIBA or NBA, half or full)
- All players, accessories, and actions for the current sequence are rendered
- Supports zoom (pinch/scroll wheel) and pan (two-finger/middle-click)

### Top Bar: Sequence Timeline

- Horizontal row of sequence tabs: `[1. Setup] [2. Close-out] [3. Recovery] [+]`
- Click a tab to switch to that sequence (court updates)
- **[+]** adds a new sequence (copies current element positions as starting point)
- Drag tabs to reorder sequences
- Right-click / long-press a tab: rename, duplicate, delete

### Right Panel: Properties

Two sections:

1. **Element properties** (top) — shown when an element is selected on the court:
   - Player: label, role, group
   - Action: type, from, to
   - Accessory: type, rotation

2. **Exercise metadata** (bottom) — always visible:
   - Name, court standard, court type, duration, intensity, category, tags

### Bottom Panel: Instructions

- List of text instructions for the **current sequence**
- Editable inline (click to edit)
- Add/remove/reorder instructions
- These are concatenated across all sequences when generating the PDF

### Animation Controls

Below the court canvas:
- **[▶ Play]** — animates through sequences (interpolates element positions)
- **[⏸ Pause]**
- **[◀ Prev] / [Next ▶]** — step through sequences one at a time
- **Speed** dropdown: 0.5x, 1x, 2x

Animation plays directly on the court canvas. Elements smoothly move from their position in sequence N to their position in sequence N+1.

### Court Canvas Interactions

| Gesture | Action |
|---|---|
| Tap/click element | Select → show properties in right panel |
| Drag element | Move to new position on court |
| Drag from tool palette to court | Create new element at drop position |
| Select action tool, then click player A then player B | Create action between A and B |
| Long press / right-click element | Context menu: delete, duplicate, change role |
| Pinch / scroll wheel | Zoom |
| Two-finger pan / middle-click drag | Pan the view |
| Double-tap empty area | Deselect all |

### Exercise File Operations

- **New**: create blank exercise (select court type/standard first)
- **Open**: pick from `~/.courtdraw/exercises/`
- **Import from library**: browse community exercises from `library/`, copies to `~/.courtdraw/exercises/`
- **Save**: write to `~/.courtdraw/exercises/<name>.yaml`
- **Duplicate**: save as new exercise with different name

## Tab 2: Session Composer

```
┌──────────────────────────────────────────────────────────────┐
│  CourtDraw     [ Exercise Editor | ● Session Composer ]      │
├────────────────────────────┬─────────────────────────────────┤
│                            │                                 │
│  EXERCISE LIBRARY          │  SESSION                        │
│                            │                                 │
│  Search: [____________]    │  Title: [High Intensity U13  ]  │
│  Category: [All      ▼]   │  Subtitle: [1v1 aggression.. ]  │
│  Tags: [defense] [1v1]    │  Age group: [U13/U15         ]  │
│                            │                                 │
│  ┌──────────────────────┐  │  ──────────────────────────     │
│  │ 🏀 Gauntlet         │  │                                 │
│  │   warmup · 12min ●●○│  │  1. [≡] 🏀 Gauntlet    12min   │
│  ├──────────────────────┤  │  2. [≡] 🏀 Double C-O  15min   │
│  │ 🏀 Double Close-Out │  │  3. [≡] 🏀 1v1 Grinder 15min   │
│  │   defense · 15min●●●│  │     ↳ 🏀 1v1 Grinder 10s       │
│  ├──────────────────────┤  │  4. [≡] 🏀 King o/t C  15min   │
│  │ 🏀 1v1 Grinder      │  │  5. [≡] 🏀 2v1 Waves   13min   │
│  │   defense · 15min●●●│  │  6. [≡] 🏀 5v5 Match   20min   │
│  ├──────────────────────┤  │  7. [≡] 🏀 Cool Down    5min   │
│  │ 🏀 King of the Court│  │                                 │
│  │   scrimmage·15min●●●│  │         Total: ~1h35            │
│  ├──────────────────────┤  │                                 │
│  │ 🏀 2v1 Waves        │  │  [+ Add Exercise]               │
│  │   transition·13m ●●○│  │                                 │
│  └──────────────────────┘  │  ──────────────────────────     │
│                            │  Coach notes:                   │
│  [+ Add] or drag ──────>  │  • Hydration: 1min water break  │
│                            │  • Adapt if fatigue             │
│                            │  [+ Add note]                   │
│                            │                                 │
│                            │  Philosophy:                    │
│                            │  [Intensity doesn't come from.. │
│                            │                              ]  │
│                            │                                 │
│                            │  [📄 Generate PDF]              │
│                            │                                 │
├────────────────────────────┴─────────────────────────────────┤
│  Preview: (court thumbnail of selected exercise)             │
└──────────────────────────────────────────────────────────────┘
```

### Left Panel: Exercise Library

- Lists all exercises from `~/.courtdraw/exercises/`
- Each entry shows: name, category, duration, intensity dots
- **Filter bar**: text search, category dropdown, tag chips
- **Add**: click button or drag exercise to the session list

### Right Panel: Session

- **Metadata**: title, subtitle, age group (text inputs)
- **Exercise list**: ordered list of exercise references
  - **[≡]** drag handle for reordering
  - Variants appear indented with `↳` prefix
  - Right-click / long-press: add variant, remove from session
  - Click: select (shows preview in bottom panel)
- **Total duration**: auto-calculated from exercise durations
- **[+ Add Exercise]**: opens exercise picker or allows drag from library
- **Coach notes**: editable list of notes
- **Philosophy**: multiline text field
- **[Generate PDF]**: generates and saves/shares the session sheet PDF

### Adding Exercises to Session

1. **Click [+ Add]** on a library exercise → appends to session list
2. **Drag** from library panel → drop in session list at desired position
3. **[+ Add Exercise]** button in session panel → shows exercise picker overlay

### Reordering

- Drag **[≡]** handles to reorder exercises in the session
- Variants stay attached to their parent exercise when reordering

### Variants

- Right-click an exercise in the session list → "Add variant"
- Pick another exercise from the library
- It appears indented under the parent: `↳ variant-name`
- In the PDF, variants are displayed as sub-items of the parent exercise

### Bottom Panel: Preview

- When an exercise is selected in the session list, shows its court thumbnail (sequence 1)
- Click the preview to open the exercise in the Exercise Editor tab

### Session File Operations

- **New**: blank session
- **Open**: pick from `~/.courtdraw/sessions/`
- **Save**: write to `~/.courtdraw/sessions/<title>.yaml`
- **Generate PDF**: render session to PDF, save or share

## Responsive Behavior

- **Mobile** (< 600dp): panels become bottom sheets or tabs instead of side-by-side. Tool palette becomes a bottom toolbar.
- **Tablet** (600–1000dp): layout as described above.
- **Desktop** (> 1000dp): wider canvas, panels can be resized.
