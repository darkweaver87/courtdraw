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
- 2m dark-blue apron (run-off area) rendered around the court — visibility controlled by the Show Apron Bands preference (`CourtWidget.ShowApron`)
- All players, accessories, and actions for the current sequence are rendered
- Element sizes scale proportionally with court type (smaller on full court, larger on half court)
- Players are clamped within court boundaries (body radius accounted for)
- Supports zoom (pinch/scroll wheel) and pan (two-finger/middle-click)

### Top Bar: Sequence Timeline

- Horizontal row of sequence tabs: `[1. Setup] [2. Close-out] [3. Recovery] [+]`
- Click a tab to switch to that sequence (court updates)
- **[+]** adds a new sequence (copies current element positions as starting point)
- Drag tabs to reorder sequences
- Right-click / long-press a tab: rename, duplicate, delete
- **Rotate button** (↻) next to the zoom buttons — toggles court orientation between portrait and landscape; fires `SeqTimeline.OnRotate`, wired to `app.toggleOrientation()`

### Right Panel: Properties

Two sections:

1. **Element properties** (top) — shown when an element is selected on the court:
   - Player: label, role, group
   - Action: type, from, to
   - Accessory: type, rotation

2. **Exercise metadata** (bottom) — always visible:
   - Name, court standard, court type, orientation (portrait/landscape), duration, intensity, category, tags
   - **Court type switching**: changing between half court and full court triggers automatic position remapping across all sequences (Half→Full compresses Y by 0.5; Full→Half expands Y by 2.0). If elements span both halves, a blocking dialog is shown instead of remapping. The same smart switching logic is also available in the **exercise settings dialog** (`app.applyCourtTypeSwitch(wantFull bool)`).
   - **Exercise settings dialog**: provides an orientation selector (portrait/landscape) and court type switcher with the same smart remapping behaviour.

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
| Tap/click element | Select → show properties in right panel (pulsing ring on selected element) |
| Hover over element | Blue highlight outline on hovered element |
| Drag element | Move to new position on court |
| Drag from tool palette to court | Create new element at drop position |
| Select action tool, then click player A then player B | Create action between A and B |
| Action tool active, hover over player | Green glow on valid action targets |
| Action tool active, move mouse after selecting source | Ghost arrow follows cursor from source player, snaps to nearest player within 30dp |
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
- **[PDF]**: generate session sheet PDF — file dialog opens in configured PDF export dir (or home)
- **[About]**: info icon, right-aligned — shows version and app info
- **[Preferences]**: gear icon, right-aligned — opens preferences dialog (GitHub token, language, exercise directory, PDF export directory, default court type, default orientation, show apron bands)
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

## Responsive Layout

The app uses a `ResponsiveContainer` (`internal/ui/responsive.go`) that swaps between desktop and mobile layouts.

### Detection Rules

- **Android / iOS**: always mobile layout, regardless of screen size or orientation
- **Desktop**: mobile layout if window width < 600dp, desktop layout otherwise
- Rotation on mobile triggers a `Layout()` call with the new size — the container rebuilds automatically

### Desktop Layout

As described above: HSplit panels (palette | court | properties), resizable.

### Mobile Layout — Exercise Editor

Three bottom tabs replace the side panels:

```
┌─────────────────┐
│ Toolbar + Lang   │
├─────────────────┤
│  [−] [+] [1:1]  │  ← zoom buttons
├─────────────────┤
│                  │
│  COURT (zoomed)  │
│                  │
├─────────────────┤
│ Timeline + Anim  │
├─────────────────┤
│[Court][Tools][Props]│  ← bottom tabs
└─────────────────┘
```

- **Court tab**: toolbar + language bar + zoom buttons + court (full screen) + sequence timeline + animation controls + status bar
- **Tools tab**: tool palette (scrollable, full screen)
- **Properties tab**: properties panel + instructions editor (VSplit 60/40)

### Mobile Layout — Session Tab

Three bottom tabs replace the 3-column split:

```
┌─────────────────┐
│                  │
│  (active panel)  │
│                  │
├─────────────────┤
│[Library][Preview][Session]│  ← bottom tabs
└─────────────────┘
```

- **Library tab**: search + filters + exercise list
- **Preview tab**: court preview + add/open/delete buttons
- **Session tab**: toolbar + metadata + exercise list + philosophy

## Zoom and Pan

The court widget supports zoom and pan for precise element placement:

| Gesture | Action |
|---|---|
| Mouse wheel / pinch | Zoom in/out (1.0x – 5.0x) |
| Double-tap | Reset zoom to 1.0x |
| Drag on empty area (when zoomed) | Pan the view |
| `+` / `−` / `1:1` buttons (mobile) | Zoom in / zoom out / reset |

A "2.0x" overlay indicator appears in the top-right corner when zoomed.
Pan is clamped so the court stays within view.
Zoom level resets when loading a new exercise.

## Training Mode

View for running a session during practice. The training view replaces the app content in the same window (no OS-level fullscreen).

### Desktop Layout

```
┌──────────────────────────────────────────────────────────────┐
│ [Quit]  [◀]     3 / 7 — Transition Drill     [▶]   04:32   │
│ ████████████████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  │
├────────────────────────────────┬─────────────────────────────┤
│                                │ Warmup  ●●○  10m            │
│                                │ ◀ Seq 2 / 4 ▶               │
│       COURT (read-only)        │ ⚠ Time expired               │
│                                │                             │
│                                │ • Instruction 1             │
│                                │ • Instruction 2             │
│                                ├─────────────────────────────┤
│                                │ [Countdown][Stopwatch][Léger]│
│                                │        00:30                │
│                                │    [Start] [Reset]          │
└────────────────────────────────┴─────────────────────────────┘
```

### Mobile Layout

```
┌──────────────────────────────────┐
│ [Quit] 3/7     04:32    [◀] [▶] │
│ ████████████░░░░░░░░░░░░░░░░░░░  │
│ Transition Drill                 │
│ ⚠ Time expired                   │
│ ┌──────────────────────────────┐ │
│ │                              │ │
│ │      COURT (read-only)       │ │
│ │                              │ │
│ └──────────────────────────────┘ │
│ Warmup ●●○ 10m  ◀ Seq 2/4 ▶     │
├──[Court]──[Instructions]──[Tools]┤
└──────────────────────────────────┘
```

### Components

- **Progress bar**: one segment per exercise — green (completed), white (current), dark grey (upcoming)
- **Exercise timer**: auto-counts down from exercise duration, turns red at 0:00, shows negative time
- **Sequence nav**: Prev/Next buttons + "Seq 2/4" label (hidden for single-sequence exercises)
- **Coach tools**: manual tab buttons (not AppTabs) switching between Countdown, Stopwatch, and Luc Léger panels; fixed-height content area prevents layout shifts; tools run independently from the exercise timer
  - **Countdown**: -1m / -10s / +10s / +1m buttons adjust duration; Start/Pause/Reset buttons; beep at zero
  - **Stopwatch**: displays mm:ss.SSS; Start/Pause/Reset buttons
  - **Luc Léger**: shows current stage, shuttle count, and speed; beeps on each shuttle interval
- **Quit**: returns to normal session tab

## Match Mode

Live match management view for tracking substitutions, playing time, fouls, and score during a game.

### Match Tab (ModeMatch)

Two views in a content stack:
- **Match list**: scrollable list of matches showing opponent, date, home/away, status badge (gray=planned, green=live, blue=finished). Open/delete buttons per row. "New Match" button at the bottom.
- **Match creation form**: team dropdown (from store), opponent entry, date/time/location/competition fields, home/away radio, period format selector (4x8, 4x10, 2x20), player selection checkboxes from team roster with starting five checkboxes. "Create Match" button.

### Live Match View (full-screen)

Full-screen takeover following the same pattern as Training Mode (`normalContent` save/restore):

```
+------------------------------------------------------------------+
|   Team A          P2  05:32          Opponent                     |
|     42                                 38                         |
+------------------------------------------------------------------+
| [+1] [+2] [+3]  Home    | [+1] [+2] [+3]  Away                  |
| [Start Period] [Timeout]                                          |
+------------------------------------------------------------------+
| ON COURT                                                          |
| #7 Jean  ●●●○○  12:30  [Foul] [Sub]                             |
| #11 Marc ●○○○○  08:45  [Foul] [Sub]                             |
| ...                                                               |
+------------------------------------------------------------------+
| BENCH                                                             |
| #23 Lucas  00:00  [Select In]                                    |
| #5 Pierre  04:20  [Select In]                                    |
+------------------------------------------------------------------+
|                              [Quit] [End Match]                   |
+------------------------------------------------------------------+
```

### Components

- **Scoreboard header**: dark background, home/away team names, large score numbers, period indicator, countdown clock
- **Period clock**: countdown from period duration, 100ms ticker with `sync.Mutex`, `fyne.Do()` for UI updates
- **Player cards**: jersey number (large), first name, foul dots (filled red / empty gray), playing time (MM:SS)
  - Foul background: 4 fouls = orange, 5 fouls = red
  - Bench players with zero playing time get yellow name color
  - Selected bench player highlighted with blue background
- **Substitution flow**: tap bench player "Select In" -> player highlighted -> tap on-court player "Sub" -> swap executed with sub_in/sub_out events
- **Score buttons**: +1/+2/+3 for each team, creates score event
- **Foul button**: per on-court player, creates foul event, shows warning dialog at 4 and fouled-out dialog at 5
- **Timeout**: pauses clock, creates timeout event
- **Auto-save**: `store.SaveMatch()` after every event

### Match Summary

Shown when match ends or when opening a finished match from the list:
- Final score header (large text)
- Playing time per player: proportional horizontal bars
- Fouls per player
- Close button to return

## Session Sharing

### Share Dialog

Triggered by the Share button in the session toolbar (visible when session has exercises).

Two options via `ShowCustomConfirm`:
1. **Share via QR** (confirm button): creates bundle → encrypts (AES-256-GCM) → uploads to tmpfiles.org (fallback: file.io) → shows QR code dialog with copyable URL
2. **Save File** (dismiss button): file save dialog for `.courtdraw` bundle

### QR Code Dialog

- QR code image (256×256) rendered via `canvas.NewImageFromImage`
- Instructional text
- Read-only entry with the full share URL
- "Copy Link" button → copies to clipboard

### Import Dialog

Triggered by the Import button in the session toolbar.

- **From File** button: opens file picker filtered to `.courtdraw` extension
- **From Link** section: text entry for pasting share URL + "Download" button
- On import: saves all exercises + session to local store, loads the session
