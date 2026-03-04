# Architecture

## Technology Stack

| Component | Technology | Rationale |
|---|---|---|
| Language | **Go** | Performant, cross-compile, no runtime |
| UI Framework | **Gio** (gioui.org) | Native Go UI, supports iOS/Android/Linux/Windows/macOS |
| Court Rendering | Gio canvas | 2D vector drawing via Gio paint/clip ops |
| Animation | Gio frame scheduling | Interpolation between sequence keyframes |
| PDF Generation | **go-pdf/fpdf** (or pure Go equivalent) | PDF generation without CGO |
| Storage | **YAML files** in `~/.courtdraw/` | Human-readable, git-friendly, no database |
| iOS/Android Build | `gogio` (Gio tool) | Native packaging from Go |

## Hard Rules

- **No CGO** — all dependencies must be pure Go for easy cross-compilation
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
│       └── main.go                  # Entry point, window creation
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
│   ├── ui/                          # Gio UI layer
│   │   ├── app.go                   # Root widget, tab navigation
│   │   ├── theme/
│   │   │   └── theme.go             # Colors, fonts, spacing constants
│   │   ├── tab/
│   │   │   ├── exercise_editor.go   # Exercise editor tab
│   │   │   └── session_composer.go  # Session composer tab
│   │   ├── widget/
│   │   │   ├── court.go             # Basketball court canvas (FIBA/NBA, half/full)
│   │   │   ├── player.go            # Player circle + label
│   │   │   ├── arrow.go             # Action arrows (solid/dashed/zigzag)
│   │   │   ├── accessory.go         # Accessory rendering (cones, ladders, chairs)
│   │   │   ├── toolbar.go           # Tool palette (Inkscape-style)
│   │   │   ├── timeline.go          # Sequence timeline bar
│   │   │   ├── dragdrop.go          # Drag & drop logic
│   │   │   ├── properties.go        # Selected element property panel
│   │   │   └── exerciselist.go      # Exercise library browser (for composer)
│   │   └── icon/
│   │       └── icons.go             # Embedded icon assets
│   ├── court/                       # Court geometry & rendering logic
│   │   ├── fiba.go                  # FIBA court dimensions and markings
│   │   ├── nba.go                   # NBA court dimensions and markings
│   │   ├── draw.go                  # Shared drawing primitives
│   │   └── geometry.go              # Coordinate mapping (relative ↔ pixel)
│   ├── anim/                        # Animation engine
│   │   ├── interpolate.go           # Position interpolation between sequences
│   │   └── playback.go              # Play/pause/seek/speed controller
│   └── pdf/                         # PDF generation
│       ├── generator.go             # Session → PDF orchestrator
│       ├── court_render.go          # Render court diagram to PDF
│       ├── layout.go                # Page layout (header, columns, overflow)
│       └── styles.go                # PDF colors, fonts, spacing
├── library/                         # Community exercise YAML collection
├── assets/
│   ├── icons/                       # Accessory/action icons (PNG/SVG)
│   └── fonts/                       # Embedded TTF/OTF fonts
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
                       ├──→ internal/court (renders model on Gio canvas)
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
| `build-android` | ubuntu-latest | `gogio -target android` → APK |
| `release` | ubuntu-latest | `softprops/action-gh-release@v2` with auto release notes |

### Build Targets

| OS | Arch | Runner | Notes |
|----|------|--------|-------|
| Linux | amd64 | ubuntu-latest | Native gcc |
| Linux | arm64 | ubuntu-latest | Cross-compile with `aarch64-linux-gnu-gcc` |
| macOS | arm64 | macos-latest | Native (Apple Silicon runner) |
| macOS | amd64 | macos-latest | Cross-arch via `GOARCH=amd64` |
| Windows | amd64 | windows-latest | Native gcc |
| Android | — | ubuntu-latest | `gogio` → APK |

### Artifacts

Each build produces an artifact (`courtdraw-{os}-{arch}.tar.gz` or `.zip` for Windows, `.apk` for Android) containing the binary and the `library/` directory. On tag push, all artifacts are uploaded to a GitHub Release.
