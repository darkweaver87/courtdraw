# Architecture

## Technology Stack

| Component | Technology | Rationale |
|---|---|---|
| Language | **Go** | Performant, cross-compile, no runtime |
| UI Framework | **Fyne v2** (fyne.io/fyne/v2) | Retained-mode Go UI, supports Android/Linux/Windows/macOS |
| Court Rendering | **image.RGBA** + `golang.org/x/image/vector` | Framework-agnostic 2D rasterization, anti-aliased |
| Court Widget | **canvas.Raster** (Fyne) | Bitmap rendering bridge between `image.RGBA` and Fyne |
| Animation | goroutine + `time.Ticker` (30fps) | Interpolation between sequence keyframes |
| PDF Generation | **go-pdf/fpdf** v0.9.0 | PDF generation, Helvetica font |
| Storage | **YAML files** in `~/.courtdraw/` | Human-readable, git-friendly, no database |
| Android Build | `fyne-cross` (Docker) | Cross-compilation for Android |

## Hard Rules

- **CGO required** — Fyne uses OpenGL via CGO for rendering
- **No database** — all data is YAML files on disk
- **No network** — the app is 100% offline
- **No backend** — no server, no cloud sync, no accounts

## File Storage

All user data lives in `~/.courtdraw/`:

```
~/.courtdraw/
├── exercises/          # Exercise YAML files (flat)
│   ├── index.yaml      # Exercise metadata index (auto-generated)
│   ├── gauntlet.yaml
│   ├── double-close-out.yaml
│   └── king-of-the-court.yaml
├── sessions/           # Session YAML files (flat)
│   ├── index.yaml      # Session metadata index (auto-generated)
│   ├── high-intensity-u13.yaml
│   └── shooting-fundamentals.yaml
└── settings.yaml       # App settings (language, PDF export dir)
```

Each directory contains an `index.yaml` that caches metadata (name, category, tags, timestamps) for fast listing without scanning individual files. The index is rebuilt automatically if missing or corrupt.

The community collection lives in the repo under `library/`:

```
courtdraw/
├── library/            # Community exercises (shipped with repo, importable)
│   ├── gauntlet.yaml
│   ├── double-close-out.yaml
│   └── ...
└── ...
```

Users import exercises from `library/` into `~/.courtdraw/exercises/` via the app. They can also create exercises directly in the app, which saves to `~/.courtdraw/exercises/`.

## Project Structure

```
courtdraw/
├── cmd/
│   └── courtdraw/
│       └── main.go                  # Entry point, Fyne app + window
├── internal/
│   ├── model/                       # Domain models (zero external dependencies)
│   │   ├── exercise.go              # Exercise, Sequence, Element
│   │   ├── session.go               # Session, ExerciseRef
│   │   └── types.go                 # Enums: roles, actions, court types, accessories
│   ├── store/                       # YAML file persistence
│   │   ├── store.go                 # Store interface
│   │   ├── yaml.go                  # YAML read/write implementation
│   │   ├── index.go                 # Index structs, load/save/rebuild
│   │   ├── settings.go              # App settings persistence
│   │   └── library.go              # Read-only access to library/ exercises
│   ├── i18n/                        # Localization (EN/FR)
│   │   └── i18n.go                  # T() translation function
│   ├── ui/                          # Fyne UI layer
│   │   ├── app.go                   # Root app, tab navigation, file operations
│   │   ├── toolbar.go               # File toolbar (new/open/save/duplicate/import)
│   │   ├── toolpalette.go           # Tool palette (players/actions/accessories)
│   │   ├── propspanel.go            # Properties panel (element + exercise metadata)
│   │   ├── seqtimeline.go           # Sequence timeline tabs
│   │   ├── animcontrols.go          # Animation playback controls
│   │   ├── instrpanel.go            # Instruction editing panel
│   │   ├── sessiontab.go            # Session tab (library + preview + session)
│   │   ├── responsive.go            # ResponsiveContainer (desktop/mobile layout swap)
│   │   ├── statusbar.go             # Status bar with auto-dismiss
│   │   ├── browser.go               # URL opener utility
│   │   ├── editor/
│   │   │   └── state.go             # Editor state (tool, selection, drag, modified)
│   │   ├── fynecourt/
│   │   │   └── court.go             # Court widget (canvas.Raster + mouse/touch)
│   │   ├── theme/
│   │   │   └── theme.go             # Fyne theme (dark palette)
│   │   └── icon/
│   │       └── icons.go             # Embedded PNG icons as fyne.Resource
│   ├── court/                       # Court rendering (framework-agnostic)
│   │   ├── draw.go                  # Drawing primitives (line, circle, arc, text)
│   │   ├── draw_players.go          # Player rendering (circle, label, ball, queue)
│   │   ├── draw_accessories.go      # Accessory rendering (cone, ladder, chair)
│   │   ├── draw_arrows.go           # Action arrow rendering (solid, dashed, zigzag)
│   │   ├── fiba.go                  # FIBA court markings
│   │   ├── nba.go                   # NBA court markings
│   │   ├── geometry.go              # Coordinate mapping (relative ↔ pixel)
│   │   └── draw_test.go             # Rendering tests (no UI framework)
│   ├── anim/                        # Animation engine
│   │   ├── interpolate.go           # Position/rotation interpolation
│   │   └── playback.go              # Play/pause/seek/speed controller
│   └── pdf/                         # PDF generation
│       ├── generator.go             # Session → PDF orchestrator
│       ├── court_render.go          # Render court diagram to PDF
│       ├── layout.go                # Page layout (header, columns, overflow)
│       └── styles.go                # PDF colors, fonts, spacing
├── library/                         # Community exercise YAML collection
├── assets/
│   ├── icons/                       # Player/action/accessory icons (PNG)
│   │   └── embed.go                 # go:embed FS
│   └── fonts/                       # Embedded TTF fonts
├── docs/                            # Specifications
├── go.mod
├── go.sum
└── CLAUDE.md
```

## Layer Dependencies

```
cmd/courtdraw  →  internal/ui  →  internal/model
                       │              ↑
                       ├──→ internal/store (reads/writes YAML for model types)
                       ├──→ internal/court (renders model into image.RGBA)
                       ├──→ internal/anim  (animates model sequences)
                       └──→ internal/pdf   (renders model to PDF)
```

- `model` has **zero** external dependencies — pure data structures and enums
- `store` depends on `model` and a YAML library
- `court`, `anim`, `pdf` depend on `model` only
- `ui` orchestrates everything

## CI/CD

GitHub Actions pipeline in `.github/workflows/ci.yaml`.

**Triggers**: `workflow_dispatch` (manual) and `push tags: v*` (release).

### Pipeline

```
test → build-desktop (matrix 5 targets) ─┐
     → build-android                     ─┤→ release (if tag v*)
```

### Jobs

| Job | Runner | Description |
|-----|--------|-------------|
| `test` | ubuntu-latest | `go test ./...` + `go vet ./...` |
| `build-desktop` | matrix | Cross-compile for 5 desktop targets (CGO_ENABLED=1) |
| `build-android` | ubuntu-latest | `fyne-cross android` → APK |
| `release` | ubuntu-latest | `softprops/action-gh-release@v2` with auto release notes |

### Build Targets

| OS | Arch | Runner | Notes |
|----|------|--------|-------|
| Linux | amd64 | ubuntu-latest | Native gcc |
| Linux | arm64 | ubuntu-latest | Cross-compile with `aarch64-linux-gnu-gcc` |
| macOS | arm64 | macos-latest | Native (Apple Silicon runner) |
| macOS | amd64 | macos-latest | Cross-arch via `GOARCH=amd64` |
| Windows | amd64 | windows-latest | Native gcc |
| Android | — | ubuntu-latest | `fyne-cross` → APK |

### Artifacts

Each build produces an artifact (`courtdraw-{os}-{arch}.tar.gz` or `.zip` for Windows, `.apk` for Android) containing the binary and the `library/` directory. On tag push, all artifacts are uploaded to a GitHub Release.
