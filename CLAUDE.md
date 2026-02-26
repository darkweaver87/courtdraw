# CourtDraw

Cross-platform basketball session designer written in Go with Gio UI.
Coaches create exercises visually on a court, animate them, compose sessions, and export PDF session sheets.

## Quick Links

- [Architecture](docs/architecture.md) — tech stack, project structure, file storage
- [Data Model](docs/data-model.md) — domain entities (Exercise, Sequence, Session) and YAML format
- [UI Specification](docs/ui.md) — tabs, exercise editor, session composer, animation
- [Features](docs/features.md) — court standards, player roles, actions, accessories, PDF generation
- [Coding Rules](docs/coding-rules.md) — conventions, style, testing, dependencies
- [Roadmap](docs/roadmap.md) — phased development plan

## Constraints

- **Language**: Go — no CGO, pure Go dependencies only
- **UI**: Gio (gioui.org) — single codebase for iOS, Android, Linux, Windows
- **Storage**: YAML files in `~/.courtdraw/` — no database
- **Offline-first**: no backend, no cloud, no accounts
- **Basketball only**: no multi-sport abstraction
- **Court standards**: FIBA + NBA with all official markings
- **Community exercises**: YAML collection in `library/` directory of this repo

## Key Commands

```bash
# Run the app
go run ./cmd/courtdraw

# Build for current platform
go build -o courtdraw ./cmd/courtdraw

# Run tests
go test ./...

# Build for Android/iOS (requires gogio)
gogio -target android ./cmd/courtdraw
gogio -target ios ./cmd/courtdraw
```

## Project Layout

```
cmd/courtdraw/       Entry point
internal/model/      Data models (Exercise, Sequence, Session)
internal/store/      YAML file persistence (~/.courtdraw/)
internal/ui/         Gio UI (screens, widgets, theme)
internal/court/      Court rendering (FIBA, NBA)
internal/anim/       Animation engine (interpolation, playback)
internal/pdf/        PDF session sheet generation
library/             Community exercise collection (YAML)
assets/icons/        Accessory and action icons (PNG/SVG)
assets/fonts/        Embedded fonts
docs/                Specifications
```
