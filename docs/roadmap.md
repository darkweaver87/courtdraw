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
