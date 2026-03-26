# Data Model

## Exercise

The core building block. Represents a single basketball drill.

### YAML Format

```yaml
name: "Double Close-Out"
description: "Sprint + 2 close-outs + 1v1 live"
court_type: half_court       # half_court | full_court
court_standard: fiba         # fiba | nba
duration: 15m
intensity: 3                 # 0=rest, 1=low, 2=medium, 3=max
category: defense            # warmup | offense | defense | transition | scrimmage | cooldown
tags:
  - close-out
  - 1v1
  - pressing
  - U13
  - U15

sequences:
  - label: "Setup"
    instructions:
      - "D starts on the baseline"
      - "Coach at top of the key with the ball"
      - "A1 on the left wing, A2 on the right wing"
    players:
      - id: d1
        label: "D"
        role: defender
        position: [0.5, 0.06]
      - id: a1
        label: "A1"
        role: attacker
        position: [0.13, 0.50]
      - id: a2
        label: "A2"
        role: attacker
        position: [0.87, 0.50]
      - id: coach
        label: "C"
        role: coach
        position: [0.5, 0.90]
    accessories: []
    actions: []

  - label: "Close-out on A1"
    instructions:
      - "Coach passes to A1"
      - "D sprints to close out on A1"
      - "Explode toward the ball handler, small steps on arrival"
    players:
      - id: d1
        position: [0.15, 0.48]   # D has moved to A1
      - id: a1
        position: [0.13, 0.50]
      - id: a2
        position: [0.87, 0.50]
      - id: coach
        position: [0.5, 0.90]
    accessories: []
    actions:
      - type: pass
        from: coach
        to: a1
      - type: sprint
        from: d1
        to: [0.15, 0.48]

  - label: "Recovery + 1v1 on A2"
    instructions:
      - "Coach passes directly to A2"
      - "D RE-sprints to close out on A2"
      - "1v1 LIVE on A2"
      - "Recovery must be a SPRINT, not a jog"
    players:
      - id: d1
        position: [0.85, 0.48]   # D has moved to A2
      - id: a1
        position: [0.13, 0.50]
      - id: a2
        position: [0.87, 0.50]
      - id: coach
        position: [0.5, 0.90]
    accessories: []
    actions:
      - type: pass
        from: coach
        to: a2
      - type: sprint
        from: d1
        to: [0.85, 0.48]
      - type: close_out
        from: d1
        to: a2
```

### Fields Reference

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Exercise name |
| `description` | string | no | Short description |
| `court_type` | enum | yes | `half_court` or `full_court` |
| `court_standard` | enum | yes | `fiba` or `nba` |
| `orientation` | enum | no | `portrait` or `landscape` — court display orientation; defaults to `landscape` on desktop, `portrait` on mobile (omitempty) |
| `duration` | duration | no | Estimated duration (e.g. `15m`, `1h30m`) |
| `intensity` | int 0–3 | no | 0=rest, 1=low, 2=medium, 3=max |
| `category` | enum | no | `warmup`, `offense`, `defense`, `transition`, `scrimmage`, `cooldown` |
| `tags` | []string | no | Free tags for filtering |
| `age_group` | enum | no | Target age group (`u9`, `u11`, `u13`, `u15`, `u17`, `u19`, `senior`) |
| `sequences` | []Sequence | yes | Ordered list of sequences (at least 1) |
| `i18n` | map[string]ExerciseI18n | no | Translations keyed by language code (e.g. `fr`) |

## Sequence

A sequence is one chronological step of an exercise. It captures the state of the court at that point in time.

| Field | Type | Required | Description |
|---|---|---|---|
| `label` | string | no | Phase name (e.g. "Close-out on A1") |
| `instructions` | []string | no | Coaching instructions for this phase |
| `players` | []Player | no | All players on the court at this step |
| `accessories` | []Accessory | no | Equipment on the court |
| `actions` | []Action | no | Movements/actions happening in this step |
| `ball_carrier` | string or []string | no | Player ID(s) holding a ball — single string or list for multiple balls |

## Player

| Field | Type | Required | Description |
|---|---|---|---|
| `id` | string | yes | Unique ID within the exercise (e.g. "d1", "a1") |
| `label` | string | no | Display label (e.g. "D", "A1", "1") |
| `role` | enum | no | See player roles below |
| `position` | [float, float] | yes | Relative position on court [x, y], range 0.0–1.0 |
| `rotation` | float | no | Rotation in degrees (0 = facing basket) |
| `callout` | enum | no | Predefined shout (`block`, `shoot`, `here`, `screen`, `switch`, `help`, `ball`, `go`) |
| `type` | string | no | `"queue"` for queued players (omit for individual) |
| `count` | int | no | Number of players in queue (only when type=queue) |
| `direction` | string | no | Queue extension direction (only when type=queue) |

### Player Roles

| Role | Description | Default Color |
|---|---|---|
| `attacker` | Generic attacker | Red `#e63946` |
| `defender` | Generic defender | Navy `#1d3557` |
| `coach` | Coach | Orange `#f4a261` |
| `point_guard` | Meneur (1) | Red `#e63946` |
| `shooting_guard` | Arrière (2) | Red `#e63946` |
| `small_forward` | Ailier (3) | Red `#e63946` |
| `power_forward` | Ailier fort (4) | Red `#e63946` |
| `center` | Pivot (5) | Red `#e63946` |

Position roles (point_guard through center) use the attack color by default but display their position number.

### Player Groups and Queues

Players can be represented as:

- **Individual**: a single player with id, label, role, position
- **Queue**: a line of waiting players

```yaml
players:
  # Individual player
  - id: a1
    label: "A1"
    role: attacker
    position: [0.13, 0.50]

  # Queue of waiting players
  - id: queue1
    type: queue
    count: 4
    position: [0.15, 0.85]    # Position of first player in queue
    direction: [1.0, 0.0]     # Direction the queue extends (horizontal right)
    label: ""                  # Optional label per queued player
```

Named groups are defined by sharing a `group` field:

```yaml
players:
  - id: a1
    label: "1"
    role: point_guard
    group: "team_a"
    position: [0.3, 0.4]
  - id: a2
    label: "2"
    role: shooting_guard
    group: "team_a"
    position: [0.7, 0.4]
```

## Accessory

| Field | Type | Required | Description |
|---|---|---|---|
| `type` | enum | yes | `cone`, `agility_ladder`, `chair` |
| `id` | string | yes | Unique ID within the exercise |
| `position` | [float, float] | yes | Relative position on court |
| `rotation` | float | no | Rotation in degrees (default 0) |

Accessory icons are PNG/SVG files in `assets/icons/` — replaceable for custom styling.

### Accessory Types (MVP)

| Type | Icon | Description |
|---|---|---|
| `cone` | Triangle | Training cone / plot |
| `agility_ladder` | Ladder | Agility / rhythm ladder |
| `chair` | Chair | Defensive drill chair |

## Action

An action represents a movement or event between elements in a sequence.

| Field | Type | Required | Description |
|---|---|---|---|
| `type` | enum | yes | See action types below |
| `from` | string or [float,float] | yes | Player ID or court position |
| `to` | string or [float,float] | yes | Player ID or court position |

### Action Types

| Type | Visual | Description |
|---|---|---|
| `pass` | Dashed arrow (orange) | Ball pass between players |
| `dribble` | Zigzag line (orange) | Dribbling with the ball |
| `sprint` | Solid arrow (red) | Sprint / run without ball |
| `shot_layup` | Arrow to basket + symbol | Layup attempt |
| `shot_pushup` | Arrow to basket + symbol | Push-up shot attempt |
| `shot_jumpshot` | Arrow to basket + symbol | Jump shot attempt |
| `screen` | Thick short bar | Screen / pick |
| `cut` | Curved arrow | Cut to the basket |
| `close_out` | Solid arrow (blue) | Defensive close-out |
| `contest` | Hand-up symbol | Shot contest |
| `reverse` | U-turn arrow | Reverse / change direction |

## Session

An ordered collection of exercises forming a training session.

### YAML Format

```yaml
title: "High Intensity Session U13/U15"
subtitle: "1v1 aggression · Full-court defense · 2v1 · Cardio"
age_group: "U13/U15"
coach_notes:
  - "HYDRATION: 1min water break between each block"
  - "Adapt if fatigue (reduce sets, not exercises)"
  - "Encourage those who fight. Address slacking through talk, not punishment."
philosophy: |
  Intensity doesn't come from punishment.
  It comes from the FORMAT of the exercises:
    → Time limit = natural urgency
    → No breaks between drills = built-in cardio
    → 1v1 competition = maximum engagement
    → In-game consequences = motivation

exercises:
  - exercise: gauntlet
    # References gauntlet.yaml by filename (without extension)

  - exercise: double-close-out

  - exercise: 1v1-grinder
    variants:
      - exercise: 1v1-grinder-10sec
        # Displayed as sub-item under 1v1-grinder

  - exercise: king-of-the-court

  - exercise: 2v1-continuous-waves

  - exercise: 5v5-scrimmage

  - exercise: cool-down
```

### Fields Reference

| Field | Type | Required | Description |
|---|---|---|---|
| `title` | string | yes | Session title |
| `date` | string | no | Session date (e.g. `"2026-03-03"`) |
| `subtitle` | string | no | Themes / subtitle |
| `age_group` | string | no | Target age group |
| `coach_notes` | []string | no | Coach notes |
| `philosophy` | string | no | Session philosophy text |
| `exercises` | []ExerciseEntry | yes | Ordered list of exercises |

### ExerciseEntry

| Field | Type | Required | Description |
|---|---|---|---|
| `exercise` | string | yes | Exercise filename (without `.yaml` extension) |
| `variants` | []ExerciseEntry | no | Variant exercises (displayed as sub-items) |

Exercise references are resolved by looking up `<name>.yaml` in `~/.courtdraw/exercises/`.

## Exercise Index

Auto-generated metadata cache stored as `~/.courtdraw/exercises/index.yaml`:

```yaml
version: 1
entries:
  - file: "3v2-jeu-a-trois"
    name: "3v2 Jeu à trois"
    category: "offense"
    age_group: "u13"
    court_type: "half_court"
    duration: "15m"
    tags: ["3v2", "passing"]
    modified: "2026-03-04T10:30:00Z"
    last_opened: "2026-03-04T14:22:00Z"
```

| Field | Type | Description |
|---|---|---|
| `file` | string | Kebab-case filename without extension |
| `name` | string | Exercise display name |
| `category` | string | Exercise category |
| `age_group` | string | Target age group |
| `court_type` | string | `half_court` or `full_court` |
| `duration` | string | Estimated duration |
| `tags` | []string | Free tags |
| `modified` | timestamp | Last save time |
| `last_opened` | timestamp | Last time exercise was opened (for recent files) |

## Session Index

Auto-generated metadata cache stored as `~/.courtdraw/sessions/index.yaml`:

```yaml
version: 1
entries:
  - file: "seance-2026-03-03"
    title: "Séance"
    date: "2026-03-03"
    modified: "2026-03-03T18:30:00Z"
    last_opened: "2026-03-03T20:00:00Z"
```

| Field | Type | Description |
|---|---|---|
| `file` | string | Kebab-case filename without extension |
| `title` | string | Session title |
| `date` | string | Session date |
| `modified` | timestamp | Last save time |
| `last_opened` | timestamp | Last time session was opened (for recent sessions) |

## Coordinate System

All positions use **relative coordinates**:
- `[0.0, 0.0]` = bottom-left corner of the court
- `[1.0, 1.0]` = top-right corner of the court
- `[0.5, 0.5]` = center of the court

Conversion to pixels/mm happens only at render time (in `court`, `pdf`, and `ui` packages).

## Settings

Stored in `~/.courtdraw/settings.yaml` with file permissions `0600`.

```yaml
language: en
github_token: "base64-encoded-token"
exercise_dir: /home/user/.courtdraw/exercises
pdf_export_dir: /home/user/Documents
default_court_type: half_court
default_orientation: landscape
show_apron: true
```

| Field | Type | Description |
|---|---|---|
| `language` | string | UI language (`en` or `fr`) |
| `pdf_export_dir` | string | Default directory for PDF exports |
| `github_token` | string | GitHub PAT, base64-encoded in YAML, decoded at load |
| `exercise_dir` | string | Exercises storage directory (defaults to `~/.courtdraw/exercises/`) |
| `default_court_type` | string | Default court type for new exercises: `half_court` or `full_court` |
| `default_orientation` | string | Default orientation for new exercises: `portrait` or `landscape` |
| `show_apron` | *bool | Whether to render the 2m apron band around the court (defaults to `true`); accessed via `ApronVisible()` helper |
| `recent_files` | []string | Recently opened exercise names (deprecated — migrated to index `last_opened`) |

## Share Bundle

A `.courtdraw` file is a gzip-compressed tar archive used for session sharing between devices.

```
session.courtdraw (tar.gz)
├── session.yaml          # Session YAML (same format as ~/.courtdraw/sessions/)
└── exercises/
    ├── drill-a.yaml      # Each referenced exercise (same format as ~/.courtdraw/exercises/)
    └── drill-b.yaml
```

Exercise names are collected from `session.Exercises` including nested `Variants`. All exercises are resolved and bundled.

### Encrypted Bundle

For cloud sharing, the bundle is encrypted before upload:

1. A random 32-byte AES-256 key is generated
2. The bundle is encrypted with AES-256-GCM (12-byte nonce prepended)
3. The encrypted blob is uploaded to tmpfiles.org (fallback: file.io)
4. The share URL includes the key in the fragment: `https://tmpfiles.org/dl/<id>/session.courtdraw.enc#k=<hex-key>`
5. The URL fragment (containing the key) is never sent to the server
