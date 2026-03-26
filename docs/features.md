# Features

## Court Standards

The app supports two court standards with all official markings.

### FIBA

Source: FIBA Official Basketball Rules — court diagram.

- Dimensions: 28m × 15m
- Three-point arc: 6.75m from basket center
- Free-throw line: 5.80m from backboard
- Restricted area (paint): 4.90m wide (rectangular since 2010)
- Center circle: 3.60m diameter
- No-charge semicircle: 1.25m radius
- Backboard: 1.80m × 1.05m, basket 3.05m high

### NBA

Source: NBA Official Rule Book — court diagram.

- Dimensions: 94ft × 50ft (28.65m × 15.24m)
- Three-point arc: 23ft 9in (7.24m), 22ft (6.71m) at corners
- Free-throw line: 15ft (4.57m) from backboard
- Restricted area (paint): 16ft (4.88m) wide
- Center circle: 12ft (3.66m) diameter
- Restricted area arc: 4ft (1.22m) radius
- Backboard: 6ft × 3.5ft (1.83m × 1.07m), basket 10ft (3.05m) high

### Court Rendering

- All markings drawn with white lines on wood-tone background
- 2m dark-blue apron (run-off area) drawn around the court on all sides (FIBA standard) — visibility toggled by the Show Apron Bands setting
- Half-court: only one basket end
- Elements shared between standards: sidelines, baselines, midcourt line (full), center circle (full), free-throw line, free-throw circle, paint/lane, three-point arc, no-charge zone, basket (backboard + rim)
- Element sizing derived from physical dimensions — player body = 0.90m on the court (2× shoulder width for visibility), via `ElementScaleForCourt()` shared between screen and PDF renderers. Font scales with elements so labels fill the head circle.
- Players are clamped to stay entirely within court boundaries (body radius accounted for, not just center point)

### Court Orientation

Each exercise has an `Orientation` field (`portrait` or `landscape`):

- **Portrait**: court length runs vertically (baskets at top and bottom) — default on mobile
- **Landscape**: court is rotated 90° so the length is horizontal — default on desktop

The orientation can be changed in two places:
- The exercise metadata section of the properties panel
- The **rotate button** (↻) in the sequence timeline bar, next to the zoom buttons — toggles between portrait and landscape immediately
- The exercise settings dialog

Changing orientation updates the court widget immediately. PDF generation is not affected by orientation — PDFs always use the layout best suited to A4.

### Smart Court Type Switching

When switching between half court and full court — either in the properties panel or the exercise settings dialog — all player, accessory, and waypoint positions are automatically remapped so they remain visually in the same location on the court:

- **Half → Full**: Y coordinates are compressed by 0.5 (the half-court occupies the bottom half of a full court)
- **Full → Half**: Y coordinates in the bottom half are expanded by 2.0; if elements span both halves (i.e. some players are in the top half), a dialog blocks the switch and informs the coach

The remapping is applied across all sequences in the exercise atomically.

## Player Roles

| Role | ID | Label | Color |
|---|---|---|---|
| Attacker | `attacker` | A | Red `#e63946` |
| Defender | `defender` | D | Navy `#1d3557` |
| Coach | `coach` | C | Orange `#f4a261` |
| Point Guard | `point_guard` | 1 | Red `#e63946` |
| Shooting Guard | `shooting_guard` | 2 | Red `#e63946` |
| Small Forward | `small_forward` | 3 | Red `#e63946` |
| Power Forward | `power_forward` | 4 | Red `#e63946` |
| Center | `center` | 5 | Red `#e63946` |

Position roles (point_guard through center) display their number as label and use the attack color.

Queued players (waiting in line) render as smaller, greyed-out circles.

## Actions

| Action | ID | Visual Rendering |
|---|---|---|
| Pass | `pass` | Dashed arrow, orange `#f4a261` |
| Dribble | `dribble` | Zigzag line with arrow, orange `#f4a261` |
| Sprint / Run | `sprint` | Solid arrow, red `#e63946` |
| Layup | `shot_layup` | Arrow to basket + layup symbol |
| Push-up shot | `shot_pushup` | Arrow to basket + push shot symbol |
| Jump shot | `shot_jumpshot` | Arrow to basket + jump shot symbol |
| Screen / Pick | `screen` | Thick short horizontal bar at position |
| Cut | `cut` | Curved arrow, red `#e63946` |
| Close-out | `close_out` | Solid arrow, blue `#2a6fdb` |
| Contest | `contest` | Hand-up symbol at position |
| Reverse | `reverse` | U-turn arrow |

### Action Selection

- **Select mode**: click the midpoint of an action arrow to select it (yellow highlight)
- **Delete mode**: click the midpoint of an action arrow to delete it directly
- **Keyboard**: press Delete/Backspace to remove the selected action
- **Properties panel**: shows Type / From / To for the selected action

### Ball Validation

Actions that require ball possession (`pass`, `dribble`, `shot_layup`, `shot_pushup`, `shot_jumpshot`) are validated at creation:
- The "from" player must be the current ball carrier, otherwise the action is rejected
- A pass must target a player (not an empty position), otherwise the action is rejected
- Sprint, cut, screen, close-out, contest, reverse do **not** require ball possession

Validation errors are displayed in the **status bar** below the court.

## Status Bar

- Displays temporary messages (errors, info, success, warning) below the court canvas
- Auto-dismisses after 3 seconds
- Message levels with color coding:
  - **Error** (level 1): dark red background
  - **Success** (level 2): green background
  - **Warning** (level 3): orange background
  - **Info** (level 0): dark grey background

## Accessories (MVP)

| Accessory | ID | Icon file |
|---|---|---|
| Cone / Plot | `cone` | `assets/icons/cone.svg` |
| Agility Ladder | `agility_ladder` | `assets/icons/agility-ladder.svg` |
| Chair | `chair` | `assets/icons/chair.svg` |

Icons are PNG or SVG files in `assets/icons/`. They can be replaced for custom styling without changing code.

## Animation

### How It Works

1. An exercise has N sequences (keyframes)
2. Each sequence defines positions for all elements
3. The animation engine interpolates positions between sequence N and N+1
4. Actions (arrows, zigzags) are drawn progressively during transitions

### Playback Controls

- **Play / Pause**: start or stop the animation
- **Previous / Next**: step one sequence at a time
- **Speed**: 0.5x, 1x, 2x

### Interpolation Rules

- Player positions: linear interpolation of (x, y) between sequences
- Players not present in the next sequence: fade out
- Players new in the next sequence: fade in
- Actions: arrow stroke draws progressively from `from` to `to`
- Accessories: static (don't interpolate, appear/disappear)

## Import / Export

### Community Library (GitHub Sync)

- Community exercises are fetched from the `library/` directory of the GitHub repo and cached locally in `~/.courtdraw/library/`
- Sync is **incremental**: a `.manifest.yaml` tracks each file's git blob SHA, only changed files are downloaded
- **Auto-sync on first launch** if the cache is empty
- **Manual sync** via the Refresh button in the session tab
- **Offline-first**: if the network is unavailable, the local cache is used silently
- Token is **optional** — the GitHub API allows 60 req/h without authentication (sufficient for occasional syncs); if a token is configured (Preferences or `GITHUB_TOKEN` env), it is used for a higher rate limit
- Status bar shows progress: syncing, up-to-date, or summary of changes (added/updated/removed)
- Importing copies the YAML file from the cache to `~/.courtdraw/exercises/`
- The user can then modify their local copy
- Community exercises can also be **opened directly** without importing (read-only from cache)

### Export / Contribute

- Exercises can be contributed to the community library directly from the app using the **Contribute** button
- Uses the GitHub API (`go-github`) to fork the repo, push the exercise YAML, and create a pull request
- Requires a GitHub Personal Access Token configured in **Preferences** (or via `GITHUB_TOKEN` env)
- If no token is configured, the status bar shows a message directing the user to Preferences

### About / Version

- Accessible via the info icon (right side of the toolbar, next to Preferences)
- Displays the application version in a dialog
- Version is injected at build time via `-ldflags "-X main.version=vX.Y.Z"` (defaults to `dev` for local builds)
- At startup, checks GitHub Releases API for a newer version and shows a notification in the status bar

### Preferences

- Accessible via the gear icon (right side of the toolbar)
- **GitHub Token**: stored base64-encoded in `settings.yaml` (mode 0600)
- **Language**: switch between EN/FR (applies immediately)
- **Exercise Directory**: exercises storage path with folder picker (defaults to `~/.courtdraw/exercises/`)
- **Default Court Type**: half or full — applied when creating a new exercise
- **Default Orientation**: portrait or landscape — applied when creating a new exercise
- **Show Apron Bands**: checkbox (default: on) — controls visibility of the 2m dark-blue apron band around the court

## Localization

- Exercise instructions, names, descriptions, and tags are localized based on the current app language
- The `i18n` field in YAML exercise files provides translations per language
- When opening, importing, or resolving exercises, the `Localized(lang)` method is applied
- Community exercises with French translations display correctly when the app is set to French

## Visual Feedback

### Selection Pulse

- Selected elements display a pulsing ring animation
- Uses a wall-clock sine wave cycling alpha between 0.3 and 1.0
- Provides clear visual indication of the currently selected element

### Hover Highlights

- Court widget implements `desktop.Hoverable` for hover detection
- `HoveredElement` tracked in EditorState
- Hovered elements show a blue highlight outline
- During action creation, valid target players show a green glow to indicate they can receive the action

### Ghost Arrow (Action Preview)

- When creating an action (e.g., pass, sprint), a semi-transparent arrow is drawn from the source player to the current mouse position via `DrawActionPreview()`
- `PreviewMousePos` in EditorState tracks the current pointer position
- The ghost arrow provides immediate visual feedback about the action being created

### Magnetic Snap

- During action creation, the ghost arrow snaps to the nearest player within 30dp
- Reduces the precision needed to target a player, especially useful on mobile

### Empty State

- When no exercise is loaded, the court area shows a centered text overlay with a title and subtitle
- When the session list is empty, a similar placeholder is displayed
- i18n keys: `empty.title`, `empty.subtitle`, `empty.session`

## Dropdown Selectors

- Properties panel fields (role, callout, court standard, court type, category) use `widget.Select` dropdowns
- The session tab category filter also uses a `widget.Select` dropdown

## Exercise & Session Management (Overlays)

Both the Open Exercise and Open Session overlays follow the same pattern:
- Each row shows the file name with a **delete button** (red trash icon)
- Clicking delete shows a **confirmation row** ("Confirm?" with confirm/cancel) before deleting the file
- Clicking the name opens the file

### Recent Files

Both exercises and sessions have a **Recent** button (clock icon) in their toolbar:
- Clicking Recent shows the overlay in **recent mode** with the 10 most recently opened/saved items
- Recent items are tracked via `last_opened` timestamps in the index (`index.yaml`)
- In recent mode, each row has a grey X button to **remove from recents** (does **not** delete the file)
- The file remains accessible via the Open button (full list)
- Legacy `recent_files` entries in `settings.yaml` are automatically migrated to the exercise index on startup

## Index Files

- Each directory (`exercises/`, `sessions/`) contains an `index.yaml` that caches metadata
- The index is loaded on startup and updated on every save/delete operation
- If `index.yaml` is missing or corrupt, it is rebuilt automatically by scanning all YAML files
- Atomic writes (`.tmp` + rename) prevent corruption
- The index enables fast listing without reading every exercise/session file

## Exercise Deletion

- From the Open Exercise overlay or the Session tab library column
- Confirming removes the exercise YAML file and its index entry
- If the deleted exercise is currently open in the editor, the editor is cleared

## Session Deletion

- From the Open Session overlay
- Confirming removes the session YAML file and its index entry
- If the deleted session is currently open, a blank session is created

## Exercise Preview

- The Session tab includes a central preview column (~35%) between the library and session panels
- Clicking an exercise in the library selects it and shows an animated preview in the center column
- Multi-sequence exercises play in a continuous loop; single-sequence exercises show a static court rendering
- The preview displays exercise name, description, category, duration, and intensity
- Below the preview, contextual management buttons appear based on the exercise's sync status

## PDF Generation

### Layout (starting point, to be refined)

**Page 1:**
- Header bar: session title + subtitle + age group
- Legend: player/arrow symbol explanations
- Two-column layout with exercises

**Per exercise block:**
- Section header: exercise name, intensity dots, duration
- Court diagram: rendering of sequence 1 (starting positions)
- Instructions: concatenated from all sequences
- Variants: listed as sub-items

**Final page(s):**
- Summary table (block number, exercise name, intensity, duration)
- Coach notes
- Philosophy section
- Total duration

Layout overflows to additional pages automatically.

## Color Palette

| Element | Hex | Usage |
|---|---|---|
| Attack / sprint | `#e63946` | Attacker players, sprint arrows, intensity labels |
| Defense | `#1d3557` | Defender players, section headers |
| Defense arrow | `#2a6fdb` | Close-out, defensive movement arrows |
| Coach / dribble / pass | `#f4a261` | Coach, dribble zigzag, pass arrows |
| Neutral / queue | `#888888` | Queued players, low-intensity |
| Court background | `#3a7d3a` | Court surface |
| Court lines | `#ffffff` | All court markings |
| Max intensity | `#c1121f` | Header bar, max intensity indicators |
| Special (King, etc.) | `#ffb703` | Special roles in specific drills |
| Light background | `#f1faee` | Cards, philosophy box |

## Tags & Filtering

- Tags are free-text strings assigned to exercises
- The exercise library panel in the session tab supports:
  - Text search (matches name, description, tags)
  - Category dropdown filter
  - Clickable tag chips to filter by tag
  - Filters combine with AND logic

## Responsive Layout

- `ResponsiveContainer` widget dynamically swaps between desktop (HSplit) and mobile (bottom tabs) layouts
- **Android / iOS**: always uses mobile layout regardless of screen size or orientation
- **Desktop**: switches to mobile layout when window width < 600dp
- Mobile editor: 3 bottom tabs — Court, Tools, Properties
- Mobile session: 3 bottom tabs — Library, Preview, Session
- Layout rebuilds on mode change; same widgets are reused across layouts

## Zoom and Pan

- Court widget supports zoom (1.0x – 5.0x) via mouse wheel scroll or pinch-to-zoom (Fyne maps pinch to scroll events)
- Double-tap resets zoom to 1.0x
- When zoomed in, dragging on an empty area pans the view (no element hit = pan mode)
- Zoom indicator overlay ("2.0x") displayed in top-right corner when zoomed
- Mobile fallback: `+` / `−` / `1:1` buttons in the toolbar
- Pan is clamped to keep the court visible
- Court background cache includes zoom/pan in its cache key
- Zoom resets when loading a new exercise

## Training Mode

View for coaches to run sessions during practice from their phone or desktop. Activated from the session tab; the training view replaces the app content in the same window (no OS-level fullscreen).

### Access

- **Training button** (play icon) in the session tab toolbar, active when session has exercises
- Replaces the normal app content in the same window
- **Quit button** returns to the session tab

### Exercise Navigation

- **Prev / Next buttons** to move between exercises
- Progress indicator: "3 / 7" with exercise name
- Progress bar — one segment per exercise: green (completed), white (current), dark grey (upcoming)

### Exercise Display

- Read-only court widget (zoomable, no editing)
- Sequence navigation (Prev/Next + "Seq 2/4") for multi-sequence exercises
- Scrollable instructions showing all sequences, with the current sequence highlighted in gold
- Metadata bar: category, intensity dots, duration

### Exercise Timer (Timebox)

- Auto-starts from `exercise.Duration` when navigating to an exercise
- Counts down automatically — always visible
- At 0:00: turns red, continues in negative (-0:15, -0:30…), shows "Time expired" banner
- The coach is never blocked — can continue or move to the next exercise

### Coach Tools

Independent tools accessible via manual tab buttons (not Fyne AppTabs) with a fixed-height content area so the layout does not shift between tools:

| Tool | Description |
|------|-------------|
| **Countdown** | Configurable timer with -1m / -10s / +10s / +1m buttons to adjust duration; start/pause/reset; alert beep at zero |
| **Stopwatch** | Counts up in mm:ss.SSS format; start/pause/reset |
| **Luc Léger** | 20m shuttle run beep test — progressive stages with automatic beep intervals |

Tools run independently from the exercise timer.

### Audio Alert

- On desktop (Linux, Windows, macOS): a 200ms sine tone at 880 Hz is generated via the `oto` library and played through the system audio output (ALSA on Linux)
- On Android: a 200ms sine tone at 880 Hz via AAudio (CGO + dlopen, no link-time dependency)
- On iOS: audio alert is a no-op (silent); native audio is not yet implemented

### Responsive Layout

- Desktop: court left (60%), instructions + tools right
- Mobile: 3 bottom tabs — Court, Instructions, Tools

## Session Sharing

Transfer sessions between devices (PC → mobile or between coaches).

### Bundle Format

A `.courtdraw` file is a gzip-compressed tar archive:

```
session.courtdraw
├── session.yaml
└── exercises/
    ├── drill-a.yaml
    └── drill-b.yaml
```

### Export Options

1. **Share via QR** — bundle is encrypted (AES-256-GCM), uploaded to tmpfiles.org (60 min auto-delete, fallback: file.io). A QR code is displayed containing the download URL with the decryption key in the URL fragment (never sent to the server). The recipient scans the QR code or copies the link.
2. **Save File** — bundle saved locally as `.courtdraw` file for manual transfer (email, USB, messaging apps).

### Import Options

1. **From File** — file picker for `.courtdraw` files, extracts session + exercises into local store.
2. **From Link** — paste a share link, app downloads the encrypted bundle, decrypts with the key from the URL fragment, and imports.

### Security

- AES-256-GCM encryption with a random 32-byte key
- Key is transmitted only via URL fragment (client-side, never sent to the server)
- Uploaded files auto-expire (60 minutes on tmpfiles.org)
