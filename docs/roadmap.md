# Development Roadmap

## Phase 1 — Foundations

Goal: app launches, displays a court, loads an exercise from YAML.

1. Go module + Fyne setup, basic window with tab layout
2. Data models in `internal/model/` (Exercise, Sequence, Player, Action, Accessory, Session)
3. YAML store in `internal/store/` (read/write exercises and sessions from `~/.courtdraw/`)
4. Court rendering in `internal/court/` (FIBA + NBA, half + full, all official markings)
5. Element rendering on court canvas (players, actions, accessories)

Deliverable: app opens, reads a YAML exercise, renders it on the court.

## Phase 2 — Exercise Editor

Goal: the coach can create and edit exercises visually.

6. Tool palette (left panel) — player, action, accessory tools
7. Drag & drop on court canvas — create and move elements
8. Properties panel (right panel) — edit selected element
9. Sequence timeline — add/switch/reorder sequences
10. Instructions panel (bottom) — edit per-sequence instructions
11. Exercise metadata editing (name, court type, duration, intensity, category, tags)
12. File operations: new, open, save, duplicate

Deliverable: full exercise editor with Inkscape-style layout.

## Phase 3 — Animation

Goal: exercises animate in the editor.

13. Interpolation engine in `internal/anim/` — compute positions between sequences
14. Playback controls — play/pause, prev/next, speed
15. Arrow/action progressive drawing during animation
16. Player fade-in/fade-out for appearing/disappearing elements

Deliverable: press Play, watch the exercise animate on the court.

## Phase 4 — Session Composer

Goal: build sessions from exercises, generate PDF.

17. Session composer tab — exercise library panel + session list panel
18. Exercise library browser with search/filter by tags/category
19. Add exercises to session (click or drag), reorder via drag & drop
20. Variant support (sub-items under parent exercises)
21. Session metadata editing (title, subtitle, age group, notes, philosophy)
22. Session file operations: new, open, save

Deliverable: compose a session by picking and ordering exercises.

## Phase 5 — PDF Generation

Goal: export session as printable PDF.

23. PDF renderer in `internal/pdf/` — court diagrams, text layout
24. Multi-page layout with header, two columns, summary table
25. Instructions concatenation across sequences
26. PDF save/share via native OS dialog

Deliverable: generate a session sheet PDF.

## Phase 6 — Community Library & Polish

Goal: import from community, cross-platform builds.

27. Library browser — read exercises from `library/` directory
28. Import flow: copy from library to `~/.courtdraw/exercises/`
29. Responsive layout (mobile/tablet/desktop)
30. Cross-platform packaging (APK, IPA, Linux/Windows binaries)
31. Ship initial community exercise collection in `library/`

Deliverable: production-ready app with community exercises.

## Phase 7 — UX Improvements

Goal: polish the user experience based on coach feedback.

32. Localized instructions — exercise names, descriptions, and instructions display in the current app language
33. Recent files — toolbar button shows last 10 opened/saved exercises, persisted in settings
34. Dropdown selectors — role, callout, category, court standard/type use popup lists instead of cycling
35. Community exercises without import — "Open" button on remote-only exercises in the manager
36. Exercise preview — animated court preview in the exercise manager's right panel

Deliverable: smoother, more intuitive UX for coaches.

## Phase 8 — Tab Consolidation

Goal: simplify navigation by merging exercise manager and session composer.

37. Merge Exercise Manager + Session Composer into a single "Session" tab with 3-column layout (library | preview | session)
38. Remove the third tab — app now has 2 tabs: Exercise Editor and Session
39. Unified toolbar with session file operations (New, Open, Save, Refresh, PDF)
40. Library column with search, status filter chips, category dropdown
41. Preview column with "Add to session" button, animated court preview, and contextual management buttons
42. Session column with metadata editors, exercise list, coach notes, philosophy

Deliverable: streamlined 2-tab UX with all exercise management and session composition in one view.

## Phase 9 — Responsive Layout & Zoom

Goal: usable on mobile (Android) with proper touch interactions.

43. `ResponsiveContainer` widget — swaps desktop HSplit vs mobile bottom tabs based on OS and screen width
44. Mobile editor layout — 3 bottom tabs (Court / Tools / Properties) with full-screen panels
45. Mobile session layout — 3 bottom tabs (Library / Preview / Session)
46. Pinch-to-zoom on court widget (1.0x–5.0x) via `fyne.Scrollable` interface
47. Pan when zoomed (drag on empty area), double-tap to reset zoom
48. Zoom indicator overlay + mobile zoom buttons (+/−/1:1) fallback
49. i18n keys for mobile tab labels (EN/FR)

Deliverable: fully usable mobile experience with zoom/pan for precise element placement.

## Phase 10 — Court Polish & Preferences

Goal: professional court rendering and in-app settings.

50. 2m dark-blue apron (run-off area) around the court (FIBA standard)
51. Element scaling from physical dimensions — player body = 2× shoulder width (0.90m) for visibility, unified `ElementScaleForCourt()` for screen and PDF
52. Body-aware clamping — `ClampPosition` accounts for player body radius, not just center point
53. Preferences dialog — GitHub token, language, exercise directory, PDF export directory with folder pickers
54. Contribute via `go-github` — replace `gh` CLI with `go-github/v74` library for exercise PRs

Deliverable: polished court rendering with apron, proportional elements, and in-app preferences.

## Phase 11 — Community Library Remote Sync

Goal: fetch community exercises from GitHub instead of shipping them with the binary.

55. GitHub library sync — fetch `library/` from GitHub API, cache locally in `~/.courtdraw/library/` with SHA manifest for incremental sync
56. Auto-sync on first launch if cache is empty
57. Manual sync via Refresh button in session tab
58. Offline-first — fallback to local cache if network unavailable
59. Optional GitHub token for higher rate limit (60 req/h without, 5000 req/h with)

Deliverable: community exercises are always up-to-date without recompiling.

## Phase 12 — Version Display & Update Check

Goal: show app version and notify users of new releases.

60. Build-time version injection via `ldflags` (`-X main.version=vX.Y.Z`)
61. About dialog accessible from toolbar info icon — displays current version
62. Startup version check — queries GitHub Releases API for latest release, shows status bar notification if newer version available
63. CI/CD injects version tag automatically for all platforms (desktop + Android)

Deliverable: users see their current version and get notified when updates are available.

## Phase 13 — Training Mode

Goal: a dedicated view for coaches to run sessions on the court, with timing tools.

64. Training mode entry — button in session tab, opens full-screen read-only view
65. Exercise navigation — Prev/Next buttons, progress indicator ("3 / 7"), progress bar with category-colored segments
66. Exercise display — read-only `CourtWidget`, sequence navigation (prev/next + "Seq 2/4"), scrollable instructions, metadata (category, intensity, duration)
67. Exercise timer (timebox) — auto-starts from `exercise.Duration`, turns red at 0:00, continues in negative, vibration + sound alert
68. Coach tools — manual countdown timer (configurable duration), stopwatch (start/stop/reset), Luc Léger test (progressive beep intervals by stage)
69. Responsive layout — desktop: court left + instructions/tools right; mobile: court top (2/3), instructions bottom, metadata bar
70. Wake lock — screen stays on while training mode is active

Deliverable: coaches run their sessions from the app with built-in timing tools.

## Phase 14 — Session Sharing (PC → Mobile)

Goal: transfer sessions from desktop to mobile without technical knowledge.

71. Bundle format — `.courtdraw` file: gzip archive containing session YAML + all referenced exercise YAMLs
72. Export — "Share" button on session, generates `.courtdraw` bundle for manual sharing (email, WhatsApp, USB)
73. Import — "Import" button on mobile, opens file picker for `.courtdraw` files, extracts session + exercises into local store
74. Cloud transfer — upload encrypted bundle to tmpfiles.org (60 min auto-delete, no account), display QR code with URL + AES-256-GCM decryption key in fragment
75. QR scan import — mobile scans QR code, downloads blob, decrypts with key from fragment, imports session + exercises
76. Fallback — file.io as alternative upload service if tmpfiles.org is unavailable

Deliverable: coaches share sessions from PC to phone by scanning a QR code or sending a file.

## Phase 15 — Mobile & Desktop UX Overhaul ⚡ P0

Goal: make the mobile experience feel native and the desktop editor cleaner. Court always visible, tools accessible without switching tabs.

### Touch targets & widget sizing (done)
77. ✅ Enlarge touch targets — TipButton mobile: 56dp (icon 36dp + padding 10dp), tool palette grid cells: 64×64dp, shelf cells: 64×64dp
78. ✅ Icon-first buttons — timer adjustments use −/+ icons on mobile, sequence indicator uses dot pills on mobile
79. ✅ Custom bottom tab bar — `MobileTabBar` widget: 52dp height, 24dp icons, 10pt labels (replaces Fyne's `AppTabs` in all mobile layouts)
80. ✅ Properties panel mobile — `makeField` returns 56dp-height rows with 13pt labels on mobile

### Mobile layout: shelf architecture (done)
81. ✅ Court always visible — the court is never hidden behind a tab. Tools live in a collapsible shelf below the court
82. ✅ Tool shelf with category tabs — bottom tab bar with 4 categories: Tools | Players | Actions | Accessories. Each category shows a grid of icon buttons
83. ✅ Collapsible shelf — chevron icon collapses/expands the shelf. Shelf auto-collapses after selecting a tool to maximize court space
84. ✅ Top bar — compact row above the court: [mode selector] [file icons: new/open/save/recent] [language flag] [settings gear] [about]
85. ✅ Sequence bar — below top bar, above court: [← prev] [label (tap to rename)] [next →] [+ add] [delete] [settings] [instructions]

### Library & file management
86. ✅ Sort by date — library sort selector (A→Z / Recent first) using file ModTime from exercise index
87. ✅ Date of creation — `Created` timestamp in exercise index entries, preserved across saves, set to Modified on first index

### Desktop improvements
88. ✅ Tool palette icon-only with tooltip — keep icons without text labels (tooltip on hover suffices), palette stays as left sidebar
89. ✅ Sequence dots on desktop — dot pills indicator on both mobile and desktop for visual consistency

### Mode system (done, was Phase 15b)
90. ✅ Mode toggle — mode selector in top bar (mobile) or toolbar (desktop) switches between Edition, Animation, Notes, Session, MyFiles, Training modes
91. ✅ Mode persistence — switching mode does not lose editor state (selected element, active tool)
92. ✅ Desktop integration — same mode switch on desktop and mobile for consistency

Deliverable: court-centric layout on mobile with instant tool access, cleaner desktop editor.

## Phase 15b — Action Timeline & Arrow Rework ⚡ P1

Goal: define the order of actions within a single sequence with step-aware player movement, and rework arrow rendering for consistency and expressiveness. Inspired by HoopsGeek's interaction model — actions drive player movement, not just visual arrows.

### A — Action Steps (done)

93. ✅ `step` field on Action — `Step int`, YAML `step,omitempty`, default 1, backward-compatible. `EffectiveStep()` + `MaxStep(seq)` helpers
94. ✅ Animation engine — intra-sequence step animation (phase 2 in playback). Each step gets equal share of duration
95. ✅ Step badges — circled numbers at action midpoints when maxStep > 1
96. ✅ Step editing — +/− buttons in shelf props. Default new actions get step = max + 1
97. ✅ Action endpoint drag — drag action destination to reposition

### B — Step-Aware Player Movement

98. `IsMovementAction()` helper — dribble, sprint, cut, reverse, close-out return true. Pass, screen, shots return false
99. `ComputePlayerPositions(seq, step, progress)` — cumulative player positions: base position + all completed movement actions from prior steps + current step interpolation
100. `stepFrame()` update — players move to their action `To` destination during their step's window. Ball follows passes. Inter-sequence transition uses final cumulative positions as starting point
101. Ball carrier per step — pass at step N transfers ball to receiver after step N completes

### C — Action Buttons from Player

102. Contextual action buttons — when a player is selected in the shelf, show Dribble/Pass/Cut/Screen/Shot buttons directly in the props area. One tap selects action type with ActionFrom pre-filled

### D — Action Timeline Panel

103. Action Timeline widget — in Animation mode, reorderable list of actions grouped by step: "Écran par AF", "Dribble par MJ"... Tap to select, drag or +/− to reorder steps
104. Mode integration — Animation mode shelf shows timeline instead of editing tools

### E — Curved Paths / Waypoints

105. `waypoints` field on Action — `[]Position`, YAML `waypoints,omitempty`. Single waypoint = quadratic Bézier. Multiple = chained curves
106. Curved drawing — `PathPoints()`, `DrawCurvedLine`, `DrawCurvedDashedLine`, `DrawCurvedZigzag`. Arrowhead follows curve tangent
107. Waypoint interaction — drag action midpoint to create waypoint. Drag existing waypoints to adjust. Circle handles on selection
108. Progressive curved drawing + hit testing on curved paths
109. PDF curved paths + movement along curves in animation
110. Branching / aiguillage — multiple actions from same player at same step = simultaneous forked arrows

### F — Arrow Visual Polish

111. ✅ Zigzag consistency — fixed segment length instead of fixed count. Long arrows get more segments
112. ✅ Arrowhead proportional to line width, endpoint dots, arrowheads offset from player body
113. ✅ Waypoints as pass-through points (not Bézier control points). Blue handles on curve
114. PDF step badges + visual parity with screen renderer

### G — Arrow & Action Style Rework ⚡ P2

115. ✅ Action types simplified — 6 canonical types (Dribble, Pass, Cut, Screen, Shot, Handoff). Legacy types normalized automatically
116. ✅ Action icons — generated programmatically with line style (zigzag, dashes, solid, T-bar, target, double bars)
117. ✅ Screen T-bar — perpendicular bar at screen endpoint (convention standard)
118. ✅ Single color palette — near-black for all actions (convention standard), differentiation by line style only
119. Endpoint dots as drag targets — blue dot at arrow endpoint allows dragging to change pass/action target visually
120. Ball carrier validation — prevent changing pass target if receiver uses ball in subsequent steps
121. Arrow color customization — allow coaches to override action color per exercise or globally in preferences
122. Player with ball shortcut — add "Attacker + Ball" player type in shelf to avoid switching between Players and Tools tabs just to assign the ball
123. Light/dark theme toggle — add theme preference (dark default, light option). Fyne auto-tints icons. Consider HoopsGeek-style light theme with accent colors

Deliverable: coaches build a full pick-and-roll in a single sequence — players move along their action paths step by step, with curved arrows and consistent visuals.

## Phase 16 — Visual Polish & Feedback ⚡ P1

Goal: match modern app standards with smooth interactions and clear visual feedback.

97. ✅ Hover/tap feedback — highlight player circle (glow outline) when pointer/finger is over a valid drop target during action creation, pulsing ring on selected element. Court widget implements `desktop.Hoverable`, `HoveredElement` in EditorState, blue highlight on hover, green glow on action targets
98. ✅ Action snap preview — when creating an action (e.g., pass), show a ghost arrow following the cursor/finger from source player to current position via `DrawActionPreview()`, snap to nearest player with magnetic effect (within 30dp). `PreviewMousePos` in EditorState
99. Transition animations — smooth fade when switching sequences (cross-dissolve on court canvas), slide-up for properties panel on mobile
100. Court theme refinement — subtle wood grain texture on court background (embedded PNG tile), anti-aliased court lines, shadow under players for depth
101. ✅ Empty state — centered text overlay when no exercise loaded (`empty.title`/`empty.subtitle`), session empty state (`empty.session`), i18n-ready
102. ✅ Status bar improvements — auto-dismiss after 3s, color-coded levels: success (green, level=2), warning (orange, level=3), error (red, level=1), info (grey, level=0)
103. ✅ Court widget refactoring — `drawSequence` split into `drawHoverHighlights`, `drawActionPreview`, `magneticSnap`, `drawSelectionOverlays` to keep cyclomatic complexity under 30

Deliverable: the app feels responsive and polished, with clear visual cues at every interaction.

## Phase 16c — Shelf & Tools UX Simplification ⚡ P1 (done)

Goal: streamline the editing interface — fewer tabs, contextual properties, lateral view tools.

### ViewTools lateral panel (done)
- ✅ Collapsible panel on the left side of the court (chevron ‹/› to fold/unfold, collapsed by default on mobile)
- ✅ Contains: Select, Eraser, Apron toggle, Rotate 90°, Zoom +/−/reset
- ✅ Select/Eraser highlight syncs across all tool sources (palette, shelf, ViewTools)

### Shelf simplification (done)
- ✅ Remove "Outils" tab — 3 tabs: **Joueurs | Actions | Accessoires**
- ✅ Shelf starts collapsed on launch

### Contextual properties in shelf tabs (done)
- ✅ **Joueurs tab** + player selected → player properties: label, role, ball, callout, RotKnob, position + d-pad, delete
- ✅ **Actions tab** + player selected → list of actions involving that player with delete per action
- ✅ **Actions tab** + action selected → action properties (type, step ±, validation, delete)
- ✅ **Accessoires tab** + accessory selected → properties with RotKnob, position + d-pad, delete
- ✅ Auto-switch to relevant tab on element selection

### Eraser tool (done)
- ✅ Eraser mode — clicking any element deletes it immediately
- ✅ Step-aware hit testing for actions (curves, waypoints, step positions)
- ✅ Red highlight ring on hover (players, accessories, actions)

Deliverable: simpler shelf with 3 tabs, contextual properties, and always-accessible view tools on the side.

## Phase 16b — Undo/Redo ⚡ P1

Goal: coaches can undo mistakes without losing work.

103. Undo/redo engine — command pattern in `internal/ui/editor/`: each mutation (move player, add action, change property, add/remove element) is wrapped in a `Command` with `Do()` and `Undo()` methods, stored in a history stack (max 50 entries)
104. Keyboard shortcuts — Ctrl+Z / Ctrl+Shift+Z (desktop), undo/redo icons in top bar (mobile) and toolbar (desktop)
105. Scope — undo/redo operates per exercise, history cleared on exercise load/new

Deliverable: Ctrl+Z works, coaches can experiment freely.

## Phase 17 — Export GIF/MP4 ⚡ P1

Goal: coaches can share animated exercises on social media and messaging apps.

89. GIF export — render animation frames to `image.RGBA` sequence (reuse existing court renderer), encode with Go `image/gif` package, configurable frame rate (10/15/20 fps) and resolution (480p/720p)
90. Export dialog — choose format (GIF/MP4), resolution, speed, include/exclude watermark, output file picker
91. MP4 export — encode frames via `ffmpeg` (bundled or system) or pure-Go encoder (`github.com/gen2brain/x264-go`), H.264 baseline profile for maximum compatibility
92. Progress indicator — show export progress bar with frame count ("Rendering 45/120…")
93. Quick share — after export, offer direct share via OS share sheet (Android Intent, desktop file manager)
94. Social-ready defaults — square 1:1 aspect ratio option (crop court to fit), loop count for GIF (infinite), 5-second minimum duration with intro/outro hold frames

Deliverable: export any exercise as a GIF or MP4 video for sharing on WhatsApp, Instagram, or team chats.

## Phase 18 — Onboarding & Tutorials ⚡ P1

Goal: new users understand the app within 2 minutes without external help.

95. First-launch wizard — 3-screen overlay: (1) "Create exercises on the court" with court screenshot, (2) "Build sessions from your library" with session screenshot, (3) "Run training on the field" with training mode screenshot — skip button always visible
96. Interactive tooltips — on first use of each major feature, show a floating tooltip pointing to the relevant UI element: "Tap here to add a player", "Drag to create a pass", "Tap Play to animate" — dismiss on interaction, don't repeat
97. Sample exercise — ship a pre-loaded "Welcome" exercise that demonstrates all element types (players, actions, accessories, 3 sequences) — auto-opens on first launch if no exercises exist
98. Help overlay — accessible from toolbar "?" icon, shows translucent overlay with labeled arrows pointing to each panel/zone, tap anywhere to dismiss

Deliverable: coaches discover the app's capabilities without reading a manual.

## Phase 19 — Team & Roster Management ⚡ P1

Goal: coaches manage their team roster — foundation for match mode and season stats.

99. Team model — new entity in `internal/model/`: `Team` (name, club, season, logo) and `Member` (first name, last name, number, license number, birth year, role [player/coach/assistant], position, photo, email, phone) — stored as YAML in `~/.courtdraw/teams/`. Coaches and assistants are team members with role=coach/assistant (license number mandatory for all — required by federation)
100. Team tab — new third app tab "Team" with member list (sortable by number/name/role), add/edit/remove members, team photo grid view. Coaches/assistants displayed in a separate "Staff" section at the top, players below sorted by jersey number
101. Member card — tap a member to see/edit their profile: photo (camera or gallery pick), role, position (players only), jersey number, license number, birth year, contact info. Birth year displayed as age category auto-computed (e.g., "2012 → U14")
102. Season concept — a team belongs to a season (e.g., "2025-2026"), coaches can archive past seasons and start new ones with roster carry-over (select which players return)
103. Player availability — per-session presence tracking: present / absent / excused / injured — simple toggle per player before or during a session
104. Jersey duty roster — assign "jersey wash" duty to a player per match/week, rotating schedule with history ("last washed by: Lucas, 2026-03-12")
105. Team export — export roster as PDF (list format: headshots + numbers + names + license numbers + birth years) for federation paperwork or parent communication
106. Trombinoscope export — generate a printable PDF photo board: grid of player photos (4×3 per page) with jersey number, first name, and position under each photo. Staff section on first page. Suitable for posting in the gym or sending to parents. Option to include or exclude contact info

Deliverable: coaches have a digital roster with contact info, availability tracking, and jersey duty management.

## Phase 20 — Match Mode ⚡ P1

Goal: live game management with substitution tracking and playing time — an e-Marque-like experience focused on the coach's needs.

106. Match model — `Match` entity: date, opponent, location, competition, home/away, roster (subset of team), quarters/periods config (4×8min, 4×10min, 2×20min — configurable), score, result — stored in `~/.courtdraw/matches/`
107. Match creation — select team, pick opponent (free text or from past opponents), set date/time/location, select available players from roster, configure period format
108. Live match view — full-screen interface optimized for quick taps during a game:
    - **Scoreboard header**: Home score | Period clock | Away score — large font, always visible
    - **On-court lineup**: 5 player slots showing jersey number + first name, large tap targets (64dp+)
    - **Bench**: horizontal scrollable row of bench players with jersey numbers
    - **Sub button**: tap bench player → tap on-court player to swap — records timestamp, highlights 5-foul players in red
109. Playing time tracking — automatic timer per player: starts when subbed in, pauses when subbed out. Real-time display of each player's cumulative time in the current game. Visual indicator when a player hasn't played yet or has significantly less time than others
110. Period management — start/stop period clock, advance to next period, handle overtime. Period transitions auto-pause all player timers. Halftime summary popup showing playing time per player
111. Quick score — +2 / +3 / +1 buttons for each team (no need to track who scored — keep it simple, this isn't e-Marque). Running score displayed prominently
112. Foul tracking — tap player circle to increment foul count (displayed as dots on player card), visual warning at 4 fouls, auto-highlight at 5 fouls with "fouled out" status
113. Match summary — end-of-game screen: final score, playing time per player (bar chart), fouls per player, substitution timeline (horizontal swim-lane chart showing when each player was on/off court)

Deliverable: coaches manage substitutions and playing time live during games, with automatic time tracking.

## Phase 21 — Season Stats & Dashboard ⚡ P2

Goal: aggregate data across the season — playing time fairness, attendance, and team management insights.

114. Season dashboard — new view accessible from Team tab: summary cards showing season overview at a glance
115. Playing time stats — per player across all matches: total minutes, average minutes/game, percentage of total available time, games played/missed. Bar chart comparison across the roster. Fairness indicator (standard deviation of playing time — helps coaches ensure equitable distribution for youth categories)
116. Attendance tracking — per player across all training sessions: present/absent/excused/injured count, attendance rate percentage, streak tracking (consecutive presences/absences). Calendar heatmap view showing team attendance over the season
117. Match history — chronological list of all matches: date, opponent, score, result (W/L). Win/loss record, points scored/conceded trend chart
118. Jersey duty history — log of who washed jerseys and when, auto-suggest next player in rotation based on fairness (longest since last wash)
119. Player season card — individual player view aggregating: games played, total playing time, attendance rate, jersey washes done, fouls total. Exportable as PDF for parent/player meetings
120. Data export — export full season stats as CSV (for spreadsheet analysis) or PDF report (for club AGM or federation reporting)

Deliverable: coaches have a clear picture of their season — who plays how much, who shows up, and whose turn it is to wash the jerseys.

## Phase 22 — Federation Integration (FFBB) ⚡ P2

Goal: import match schedules and results from the French Basketball Federation (FFBB/FBI) to avoid double data entry.

121. FFBB connector — `internal/federation/ffbb/` package: authenticate with club federal number, fetch team calendar and results from `competitions.ffbb.com` (HTML scraping as no public REST API exists — parse match tables, dates, opponents, scores)
122. Match import — sync FFBB calendar into CourtDraw match list: create match stubs with date, opponent, location pre-filled. Coach only needs to add lineup and track subs/time live
123. Result sync — after a match, fetch the official score from FFBB and reconcile with locally tracked score (flag discrepancies)
124. Competition context — display league/cup name, current standings, and upcoming matches in a "Competition" section of the Team tab
125. Multi-federation architecture — `internal/federation/` interface with `Connector` abstraction (methods: `FetchCalendar`, `FetchResults`, `FetchStandings`), FFBB as first implementation. Future connectors: FIBA Europe, NBA-style recreational leagues, Spanish FEB, etc.
126. Offline resilience — cache all fetched data locally, sync only when network available, never block the app on network failure
127. Settings — federation config in Preferences: federation type (FFBB/none), club number, team identifier, sync frequency (manual/daily/weekly)

Deliverable: match schedules appear automatically from the federation — coaches just show up and track the game.

## Phase 23 — Smart Action Creation ⚡ P2

Goal: streamline action creation from click-click-click to drag-and-drop.

128. Drag-to-create actions — drag FROM a player to initiate action creation: a ghost arrow follows the finger/cursor, drop ON another player or court position to complete. Action type inferred from context:
    - Player → Player: defaults to Pass
    - Player → Basket area: defaults to Shot (layup/jumpshot submenu)
    - Player → Empty court: defaults to Sprint/Dribble (toggle based on ball carrier)
129. Action type picker — after drop, show a compact radial/popup menu near the endpoint to override the default type (pass/dribble/cut/screen…) — disappears after selection or 3 seconds
130. Quick-delete gesture — swipe action arrow sideways (mobile) or press Delete key (desktop) to remove
131. Multi-action chains — tap a player, then tap multiple destinations to chain actions in sequence (sprint → screen → roll) without reselecting the tool each time

Deliverable: creating a 5-action exercise takes 30 seconds instead of 2 minutes.

## Phase 24 — Playbook & Exercise Organization ⚡ P2

Goal: coaches organize exercises beyond sessions, with thematic grouping.

132. Playbook model — new entity: `Playbook` with name, description, category (offense/defense/transition/specials), and ordered list of exercise references — stored as YAML in `~/.courtdraw/playbooks/`
133. Playbook tab — add a "Playbooks" section in the library/My Files area, list playbooks with exercise count and preview thumbnails
134. Drag exercises into playbooks — from library or session, drag an exercise into a playbook group
135. Playbook PDF export — generate a multi-page PDF with all exercises in the playbook, table of contents, one exercise per page with diagram + instructions

Deliverable: coaches maintain a permanent "offensive playbook" or "defensive sets" collection independent of sessions.

## Phase 25 — Web Sharing & Links ⚡ P3

Goal: share animated exercises via a simple URL, viewable in any browser.

136. Static web viewer — generate a self-contained HTML page with embedded court SVG + CSS keyframe animation (no JS dependency), exercise name, instructions, and metadata
137. Share to GitHub Pages — upload generated HTML to a `gh-pages` branch in the user's fork or a shared hosting space, return a stable URL
138. Share via link — generate a temporary link (like current QR sharing) that serves the HTML viewer with the animated exercise, auto-expires after 7 days
139. Embed code — provide an `<iframe>` snippet for coaches who want to embed exercises on their club website or blog

Deliverable: coaches share an exercise as a clickable link that plays in any browser.

## Phase 26 — Import & Interoperability ⚡ P3

Goal: coaches can migrate from other tools without recreating everything.

140. FastDraw import — parse `.fdb` files (XML-based) and convert to CourtDraw YAML: map player positions, action types, court dimensions
141. Basketball Playbook import — parse common `.bpz` format used by Basketball Playbook app
142. Image-to-exercise (AI-assisted) — given a screenshot of a play diagram, use vision model to identify player positions, actions, and generate a CourtDraw YAML (experimental, requires API key)

Deliverable: coaches migrating from FastDraw or other tools can import their existing library.

## Phase 27 — Advanced Animation ⚡ P3

Goal: richer, more realistic animations that match HoopsGeek quality.

143. Optional/conditional actions — mark actions as "option A" / "option B" with visual toggle, allowing a single exercise to show multiple reads (e.g., pick-and-roll: option A = pop, option B = roll)
144. Ball physics — ball follows realistic arc on passes (parabolic), bounces on dribble, spin on shots — visual only, no simulation
145. Trail effect — fading trail behind moving players during animation (last 0.5s of path visible)
146. Fine-grained timing — extend Phase 15c step system with relative delay values (e.g., "step 2 starts 0.5s after step 1 ends") for precise choreography

Deliverable: animations are fluid, realistic, and can express complex plays with timing and options.
