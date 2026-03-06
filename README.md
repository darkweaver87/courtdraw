# CourtDraw

A cross-platform basketball session designer for coaches. Create exercises visually on a court, animate sequences, compose training sessions, and export PDF session sheets.

## Features

- **Visual exercise editor** — drag and drop players, actions (pass, dribble, screen, cut...), and accessories (cones, ladders, chairs) on a basketball court
- **FIBA & NBA courts** — full and half-court with all official markings
- **Player roles** — attackers, defenders, coach, and positional roles (PG, SG, SF, PF, C) with distinct colors
- **Action arrows** — pass, dribble, sprint, shot/layup, screen, cut, close-out, contest, reverse with styled lines
- **Animation** — animate player movements across sequences with interpolation, speed control, and playback
- **Ball possession** — track ball carrier with smooth ball animation between sequences
- **Session composer** — combine exercises into training sessions with drag reorder
- **PDF export** — generate A4 session sheets with court diagrams, summary table, coach notes, and philosophy
- **Community library** — import and share exercises from the built-in YAML collection
- **Localization** — English and French
- **Offline-first** — no account, no cloud, no backend. All data stored locally as YAML files

## Platforms

| Platform | Status | Notes |
|----------|--------|-------|
| Linux (amd64) | Supported | Native OpenGL |
| Linux (arm64) | Supported | Native OpenGL |
| Windows (amd64) | Supported | Includes Mesa software renderer fallback |
| macOS (amd64) | Supported | Native OpenGL |
| macOS (arm64) | Supported | Native OpenGL |
| Android | Supported | Adaptive mobile layout with bottom tabs |

## Screenshots

*Coming soon*

## Installation

### Download

Grab the latest release from the [Releases page](https://github.com/darkweaver87/courtdraw/releases).

- **Linux/macOS**: extract the archive and run `./courtdraw`
- **Windows**: extract the zip and run `courtdraw.exe`. On machines without GPU drivers (VMs, RDP), use `courtdraw-mesa.bat` instead
- **Android**: install the APK

### Build from source

Requires Go 1.23+ and CGO (Fyne uses OpenGL).

```bash
# Linux — install dependencies first
sudo apt-get install libgl1-mesa-dev libegl1-mesa-dev libxkbcommon-x11-dev \
  libwayland-dev libx11-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev

# Build and run
go run ./cmd/courtdraw

# Build binary
go build -o courtdraw ./cmd/courtdraw

# Build for Android (requires fyne-cross)
go install github.com/fyne-io/fyne-cross@latest
fyne-cross android -app-id com.darkweaver87.courtdraw ./cmd/courtdraw
```

## Usage

### Exercise editor

1. Select a player tool from the left palette and tap the court to place players
2. Select an action tool and click two players/positions to create movement arrows
3. Use the timeline at the bottom to add sequences (animation frames)
4. Adjust properties in the right panel (role, label, rotation, ball possession)
5. Hit play to preview the animation

### Session composer

1. Switch to the Session tab
2. Browse the exercise library or open saved exercises
3. Add exercises to the session with drag-and-drop reordering
4. Fill in session metadata (date, coach, theme, philosophy)
5. Export as PDF

## Contributing an exercise

You can share your exercises with the community directly from CourtDraw. Here's how:

### 1. Create a GitHub account

If you don't have one yet, sign up for free at [github.com](https://github.com/join).

### 2. Generate a Personal Access Token (PAT)

1. Go to [github.com/settings/tokens](https://github.com/settings/tokens?type=beta) (Fine-grained tokens)
2. Click **Generate new token**
3. Give it a name (e.g. "CourtDraw")
4. Under **Repository permissions**, set **Contents** to **Read and write** and **Pull requests** to **Read and write**
5. Click **Generate token** and copy the token

### 3. Configure the token in CourtDraw

1. Open CourtDraw
2. Click the gear icon (top-right of the toolbar) to open **Preferences**
3. Paste your token in the **GitHub Token** field
4. Click **Save**

Your token is stored locally in `~/.courtdraw/settings.yaml` (encoded, not in plain text) and never sent anywhere except to the GitHub API.

### 4. Contribute

1. Go to the **Session** tab
2. Find your exercise in the library list
3. Click **Contribute**
4. CourtDraw will automatically fork the repository, upload your exercise, and create a pull request

## Data storage

All data is stored as YAML files in `~/.courtdraw/`:

```
~/.courtdraw/
├── exercises/       # Saved exercises (.yaml)
├── sessions/        # Saved sessions (.yaml)
├── exercises.idx    # Exercise index for fast listing
├── sessions.idx     # Session index for fast listing
└── settings.yaml    # User preferences (language, GitHub token, exercise dirs)
```

## Tech stack

- **Language**: Go
- **UI framework**: [Fyne v2](https://fyne.io/) — single codebase for all platforms
- **Court rendering**: `image.RGBA` via `golang.org/x/image/vector` (framework-agnostic)
- **PDF generation**: `go-pdf/fpdf`
- **Storage**: YAML files (no database)

## License

MIT
