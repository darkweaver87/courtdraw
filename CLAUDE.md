# CourtDraw

Cross-platform basketball session designer written in Go with Fyne v2.
Coaches create exercises visually on a court, animate them, compose sessions, and export PDF session sheets.

## Quick Links

- [Architecture](docs/architecture.md) — tech stack, project structure, file storage
- [Data Model](docs/data-model.md) — domain entities (Exercise, Sequence, Session) and YAML format
- [UI Specification](docs/ui.md) — tabs, exercise editor, session composer, animation
- [Features](docs/features.md) — court standards, player roles, actions, accessories, PDF generation
- [Coding Rules](docs/coding-rules.md) — conventions, style, testing, dependencies
- [Roadmap](docs/roadmap.md) — phased development plan

## Constraints

- **Language**: Go — CGO required (Fyne uses OpenGL)
- **UI**: Fyne v2 (fyne.io/fyne/v2) — single codebase for Android, Linux, Windows, macOS
- **Court rendering**: Framework-agnostic `image.RGBA` via `golang.org/x/image/vector`
- **Storage**: YAML files in `~/.courtdraw/` — no database
- **Offline-first**: no backend, no cloud, no accounts — network used only for community library sync and contributions
- **Basketball only**: no multi-sport abstraction
- **Court standards**: FIBA + NBA with all official markings
- **Community exercises**: fetched from `library/` in this repo via GitHub API, cached in `~/.courtdraw/library/`

## Key Commands

```bash
# Run the app
go run ./cmd/courtdraw

# Build for current platform
go build -o courtdraw ./cmd/courtdraw

# Run tests
go test ./...

# Build for Android (requires fyne-cross)
fyne-cross android -app-id com.darkweaver87.courtdraw ./cmd/courtdraw

# CI/CD — trigger manual run
gh workflow run ci.yaml

# CI/CD — create a release (triggers build + release)
git tag v1.0.0
git push origin v1.0.0
```

## Project Layout

```
cmd/courtdraw/           Entry point (Fyne app)
internal/model/          Data models (Exercise, Sequence, Session)
internal/store/          YAML file persistence (~/.courtdraw/)
internal/ui/             Fyne UI panels (app, toolbar, properties, etc.)
internal/ui/editor/      Editor state machine (tool, selection, drag)
internal/ui/fynecourt/   Court widget (canvas.Raster + interaction)
internal/ui/theme/       Fyne theme (dark palette)
internal/ui/icon/        Embedded PNG icons as fyne.Resource
internal/court/          Court rendering into image.RGBA (FIBA, NBA)
internal/anim/           Animation engine (interpolation, playback)
internal/pdf/            PDF session sheet generation
internal/i18n/           Localization (EN/FR)
library/                 Community exercise collection (YAML)
assets/icons/            Accessory and action icons (PNG)
assets/fonts/            Embedded fonts
docs/                    Specifications
.github/workflows/       CI/CD (GitHub Actions)
```
