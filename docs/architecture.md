# Architecture

## Technology Stack

| Component | Technology | Rationale |
|---|---|---|
| Language | **Go** | Performant, cross-compile, no runtime |
| UI Framework | **Fyne v2** (fyne.io/fyne/v2) | Retained-mode Go UI, supports Android/Linux/Windows/macOS |
| Court Rendering | **image.RGBA** + `golang.org/x/image/vector` | Framework-agnostic 2D rasterization, anti-aliased, 2m apron; landscape mode rotates the court 90° via `ComputeViewportOriented()` |
| Court Widget | **canvas.Raster** (Fyne) | Bitmap rendering bridge between `image.RGBA` and Fyne |
| Animation | goroutine + `time.Ticker` (30fps) | Interpolation between sequence keyframes |
| PDF Generation | **go-pdf/fpdf** v0.9.0 | PDF generation, Helvetica font |
| Storage | **YAML files** in `~/.courtdraw/` | Human-readable, git-friendly, no database |
| Audio (Desktop) | **ebitengine/oto** v3 | Cross-platform audio output (ALSA/CoreAudio/DirectSound) |
| Audio (Android) | **AAudio** via CGO + dlopen | Runtime-loaded, no link-time NDK dependency |
| QR Codes | **skip2/go-qrcode** | Pure Go QR code generation for session sharing |
| Android Build | `fyne-cross` (Docker) | Cross-compilation for Android |

## Hard Rules

- **CGO required** — Fyne uses OpenGL via CGO for rendering
- **No database** — all data is YAML files on disk
- **No backend** — no server, no cloud sync, no accounts
- **Offline-first** — network used only for community library sync (GitHub API) and contributions; always falls back to local cache

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
├── library/            # Community exercise cache (synced from GitHub)
│   ├── .manifest.yaml  # SHA manifest for incremental sync
│   ├── gauntlet.yaml
│   ├── double-close-out.yaml
│   └── ...
└── settings.yaml       # App settings (language, token, exercise dir, PDF export dir) — mode 0600
```

Each directory contains an `index.yaml` that caches metadata (name, category, tags, timestamps) for fast listing without scanning individual files. The index is rebuilt automatically if missing or corrupt.

The community collection is fetched from the `library/` directory of the GitHub repo (`darkweaver87/courtdraw`) and cached locally in `~/.courtdraw/library/`. Sync is incremental (SHA-based manifest) and triggered:
- Automatically on first launch if the cache is empty
- Manually via the Refresh button in the session tab

If the network is unavailable, the local cache is used silently.

Users import exercises from the community cache into `~/.courtdraw/exercises/` via the app. They can also create exercises directly in the app, which saves to `~/.courtdraw/exercises/`.

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
│   │   ├── library.go              # Read-only access to library/ exercises
│   │   ├── library_sync.go         # GitHub fetch + SHA manifest for incremental sync
│   │   └── version.go              # GitHub Releases API version check
│   ├── i18n/                        # Localization (EN/FR)
│   │   └── i18n.go                  # T() translation function
│   ├── ui/                          # Fyne UI layer
│   │   ├── app.go                   # Root app, tab navigation, file operations
│   │   ├── toolbar.go               # File toolbar (new/open/save/duplicate/import/about/prefs)
│   │   ├── about.go                # About dialog (version display)
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
│   │   │   ├── state.go             # Editor state (tool, selection, drag, modified)
│   │   │   └── history.go           # Undo/redo history (YAML snapshot stack, max 50)
│   │   ├── fynecourt/
│   │   │   └── court.go             # Court widget (canvas.Raster + mouse/touch/hover)
│   │   ├── theme/
│   │   │   └── theme.go             # Fyne theme (dark palette)
│   │   └── icon/
│   │       └── icons.go             # Embedded PNG icons as fyne.Resource
│   ├── court/                       # Court rendering (framework-agnostic)
│   │   ├── draw.go                  # Drawing primitives (line, circle, arc, text)
│   │   ├── draw_players.go          # Player rendering (circle, label, ball, queue, selection pulse)
│   │   ├── draw_accessories.go      # Accessory rendering (cone, ladder, chair)
│   │   ├── draw_arrows.go           # Action arrow rendering (solid, dashed, zigzag) + DrawActionPreview()
│   │   ├── render.go                # RenderSequence() — single source of truth for static court+elements rendering (used by PDF, future GIF/MP4)
│   │   ├── fiba.go                  # FIBA court markings
│   │   ├── nba.go                   # NBA court markings
│   │   ├── geometry.go              # Coordinate mapping (relative ↔ pixel), apron, element scaling; ComputeViewportOriented() applies landscape rotation; RelToPixel/PixelToRel handle the axis swap
│   │   └── draw_test.go             # Rendering tests (no UI framework)
│   ├── anim/                        # Animation engine
│   │   ├── interpolate.go           # Position/rotation interpolation
│   │   └── playback.go              # Play/pause/seek/speed controller
│   ├── pdf/                         # PDF generation
│   │   ├── generator.go             # Session → PDF orchestrator
│   │   ├── court_render.go          # Insert court diagram image into PDF (uses court.RenderSequence)
│   │   ├── layout.go                # Page layout (header, columns, overflow)
│   │   └── styles.go                # PDF colors, fonts, spacing
│   └── share/                       # Session sharing (bundle, crypto, upload)
│       ├── bundle.go                # tar.gz bundle creation/extraction
│       ├── crypto.go                # AES-256-GCM encrypt/decrypt
│       └── upload.go                # HTTP upload to tmpfiles.org/file.io
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
                       ├──→ internal/pdf   (renders model to PDF)
                       └──→ internal/share (bundle, encrypt, upload for session sharing)
```

- `model` has **zero** external dependencies — pure data structures and enums
- `store` depends on `model` and a YAML library
- `court` depends on `model` and `i18n`
- `anim` depends on `model` and `court`
- `pdf` depends on `model`, `court`, `anim`, and `i18n`
- `share` depends on `model` only (stdlib crypto + archive + net/http)
- `store` also uses `go-github` for community library sync (incremental fetch from GitHub)
- `ui` orchestrates everything and uses `go-github` for contribution PRs, `go-qrcode` for QR display

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

| Artifact | Format | Contents |
|----------|--------|----------|
| `courtdraw-linux-{arch}` | Plain binary | Executable (chmod +x required) |
| `courtdraw-darwin-{arch}.dmg` | macOS DMG | `.app` bundle with Info.plist |
| `courtdraw-windows-amd64.zip` | Zip | Executable + Mesa DLLs + `courtdraw-mesa.bat` fallback |
| `courtdraw-android.apk` | APK | Android package |

Version is injected at build time via `-ldflags "-X main.version=<tag>"`. On tag push, all artifacts are uploaded to a GitHub Release with auto-generated changelog (mikepenz/release-changelog-builder-action, commit mode).
