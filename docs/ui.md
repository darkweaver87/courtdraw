# UI Specification

## Overall Layout

The app uses a **two-tab** interface:

```
┌───────────────────────────────────────────────────────────────────────┐
│  CourtDraw                  [ Exercise Editor | Session ]             │
├───────────────────────────────────────────────────────────────────────┤
│                                                                       │
│                        (active tab content)                           │
│                                                                       │
└───────────────────────────────────────────────────────────────────────┘
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
- **Recent**: open from a list of the 10 most recently opened/saved exercises
- **Import from library**: browse community exercises from `library/`, copies to `~/.courtdraw/exercises/`
- **Save**: write to `~/.courtdraw/exercises/<name>.yaml`
- **Duplicate**: save as new exercise with different name

## Tab 2: Session

Three-column layout merging the exercise library, preview, and session composition into a single tab.

```
┌──────────────────────────────────────────────────────────────────────┐
│ [New] [Open] [Save] [Refresh]                               [PDF]   │
├────────────────────┬─────────────────────┬───────────────────────────┤
│  Library           │  Preview            │  Session                  │
│  (~30%)            │  (~35%)             │  (~35%)                  │
│  [Search..]        │  [+ Add to session] │  Title: ___              │
│  [All|Local|...]   │                     │  Date: ___               │
│  [Category ▼]      │  ┌───────────────┐  │  Subtitle: ___           │
│                    │  │   court       │  │  Age Group: ___          │
│  Ex1   Local       │  │   preview     │  │                          │
│  Ex2   Community   │  │   (animated)  │  │  Exercises:              │
│ >Ex3   Modified    │  └───────────────┘  │  1. Ex1    x             │
│  Ex4   Synced      │  "Exercise Name"    │  2. Ex3    x             │
│  ...               │  Cat · 15m ●●○      │  Total: 30m              │
│                    │                     │                          │
│                    │  [Open] [Import]    │  Coach Notes              │
│                    │  [Contribute] [Del] │  Philosophy               │
└────────────────────┴─────────────────────┴───────────────────────────┘
```

### Toolbar

- **[New]**: create a blank session
- **[Open]**: pick from `~/.courtdraw/sessions/`
- **[Save]**: write to `~/.courtdraw/sessions/<title>.yaml`
- **[Refresh]**: reload exercise library
- **[PDF]**: generate session sheet PDF (right-aligned)
- Save icon highlights when session is modified

### Left Column: Library (~30%)

- Lists all exercises (local + community) merged and sorted
- Search bar with text filter
- Status filter chips: All, Local, Community, Synced, Modified
- Category dropdown filter
- Each row shows: display name + sync status badge (compact)
- Clicking a row selects it and loads the preview in the center column

### Center Column: Preview (~35%)

- **[+ Add to session]** button at top — adds selected exercise to the session list
- **Animated court preview** in the middle — loops through sequences, static for single-sequence exercises
- Shows exercise metadata (name, description, category, duration, intensity)
- **Management buttons** at bottom, contextual based on sync status:
  - **Always**: Open (in editor)
  - **Remote only**: Import
  - **Local only**: Contribute, Delete
  - **Synced**: Delete
  - **Modified**: Update, Contribute, Delete

### Right Column: Session (~35%)

- **Metadata editors**: title, date (with today/calendar buttons), subtitle, age group
- **Exercise list**: ordered entries with remove buttons, total duration
- **Coach notes**: editable list with add/remove
- **Philosophy**: multiline text field
- "No session loaded" placeholder when empty

### Session List Overlay

- Modal overlay for picking a session to open (triggered by Open button)
- Shows list of saved session names from `~/.courtdraw/sessions/`

## Responsive Behavior

- **Mobile** (< 600dp): panels become bottom sheets or tabs instead of side-by-side. Tool palette becomes a bottom toolbar.
- **Tablet** (600–1000dp): layout as described above.
- **Desktop** (> 1000dp): wider canvas, panels can be resized.
