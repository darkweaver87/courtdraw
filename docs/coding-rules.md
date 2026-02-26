# Code Conventions

> Naming conventions, code style, and best practices for the CourtDraw Go codebase.

## Table of Contents

1. [Go Conventions](#go-conventions)
2. [YAML Conventions](#yaml-conventions)
3. [Testing Standards](#testing-standards)
4. [Development Workflow](#development-workflow)
5. [Git Workflow](#git-workflow)
6. [Code Review](#code-review)

---

## Go Conventions

### Naming Conventions

**Packages and files:**
```go
// Package names: singular, lowercase
package model
package store
package court
package anim

// Files: snake_case
exercise.go
session.go
court_render.go
agility_ladder.go
```

**Variables and functions:**
```go
// Variables: camelCase
var exerciseID string
var courtType CourtType

// Exported functions: PascalCase
func LoadExercise(path string) (*Exercise, error)

// Private functions: camelCase
func interpolatePosition(from, to Position, t float64) Position

// Constants: PascalCase for exported, SCREAMING_SNAKE_CASE for config-like
const DefaultIntensity = 0
const MAX_SEQUENCES = 100

// Interfaces: behavior suffix or "er"
type Store interface {}
type Renderer interface {}
```

**Structures:**
```go
// Structs: PascalCase
type Exercise struct {
    Name        string     `yaml:"name"`
    Description string     `yaml:"description"`
    CourtType   CourtType  `yaml:"court_type"`
    Duration    Duration   `yaml:"duration"`
    Intensity   int        `yaml:"intensity"`
    Category    Category   `yaml:"category"`
    Tags        []string   `yaml:"tags"`
    Sequences   []Sequence `yaml:"sequences"`
}

// YAML tags: snake_case (matches YAML field names)
```

### Code Style

**Import organization:**
```go
import (
    // 1. Standard library
    "fmt"
    "os"
    "path/filepath"

    // 2. External packages
    "gioui.org/layout"
    "gopkg.in/yaml.v3"

    // 3. Internal packages
    "github.com/darkweaver87/courtdraw/internal/model"
    "github.com/darkweaver87/courtdraw/internal/store"
)
```

**Error handling:**
```go
// GOOD: Explicit error handling with context
exercise, err := store.LoadExercise(path)
if err != nil {
    return fmt.Errorf("loading exercise %s: %w", path, err)
}

// GOOD: Sentinel errors for expected conditions
var ErrNotFound = errors.New("not found")
var ErrInvalidFormat = errors.New("invalid format")

// BAD: Ignoring errors
exercise, _ := store.LoadExercise(path)

// BAD: Panicking on recoverable errors
exercise, err := store.LoadExercise(path)
if err != nil {
    panic(err)
}
```

**Structured logging:**
```go
// GOOD: Structured logging with context
logger.Info("Loading exercise",
    "path", path,
    "name", exercise.Name)

logger.Error("Failed to save session",
    "error", err,
    "sessionTitle", session.Title)

// BAD: Unstructured logging
log.Printf("Loading exercise %s", path)
```

**Comments:**
```go
// Package documentation
// Package model defines the domain types for exercises, sessions, and court elements.
package model

// Public function: Godoc comments
// LoadExercise reads an exercise YAML file and returns the parsed Exercise.
// It validates the exercise structure and returns an error if the format is invalid.
func LoadExercise(path string) (*Exercise, error) {
    // implementation comments in lowercase without period
    // read file contents
    data, err := os.ReadFile(path)
}
```

### Architecture Rules

- **`model` package has zero dependencies** — no imports from other internal packages, no third-party libs
- All file I/O goes through the `Store` interface — UI never reads/writes files directly
- UI widgets receive data as values, not file handles or store references
- No global state — pass dependencies explicitly (constructor injection)
- Errors are returned, not panicked — `panic` only for truly unrecoverable programmer errors
- No `init()` functions — explicit initialization only

### Coordinate System

```go
// GOOD: All positions use relative coordinates (0.0–1.0)
type Position struct {
    X float64 `yaml:"x"` // 0.0 = left edge, 1.0 = right edge
    Y float64 `yaml:"y"` // 0.0 = bottom edge, 1.0 = top edge
}

// BAD: Using pixels or millimeters in model
type Position struct {
    X float64 // 150px — NO!
    Y float64 // 200mm — NO!
}

// Conversion to pixels/mm happens ONLY at render time
// in court/, pdf/, and ui/ packages
```

### Dependencies

- **Pure Go only** — no CGO. This is non-negotiable for cross-compilation.
- Approved dependencies:
  - `gioui.org` — UI framework
  - `gopkg.in/yaml.v3` — YAML parsing
  - `go-pdf/fpdf` (or pure Go equivalent) — PDF generation
  - `github.com/google/uuid` — UUID generation
- Minimize dependency count. Prefer stdlib when possible.
- Pin dependency versions in `go.sum`

---

## YAML Conventions

### File naming
```
# Exercises: kebab-case
double-close-out.yaml
king-of-the-court.yaml
1v1-grinder.yaml

# Sessions: kebab-case
high-intensity-u13.yaml
shooting-fundamentals.yaml
```

### Field naming
```yaml
# GOOD: snake_case for all YAML fields
court_type: half_court
court_standard: fiba
key_points:
  - "Sprint to close out"

# BAD: camelCase or PascalCase
courtType: halfCourt
CourtStandard: FIBA
```

### Enum values
```yaml
# GOOD: snake_case for enum values
court_type: half_court
category: defense
role: point_guard
action_type: close_out

# BAD: kebab-case or UPPERCASE
court-type: half-court
category: DEFENSE
```

### Duration format
```yaml
# GOOD: Go duration format
duration: 15m
duration: 1h30m

# BAD: Other formats
duration: "15 minutes"
duration: 900
```

---

## Testing Standards

### Coverage Goals

| Component | Target |
|-----------|--------|
| Overall | 80% minimum |
| `model` | 90%+ (domain logic, validation) |
| `store` | 85%+ (YAML read/write) |
| `court` | 85%+ (geometry, coordinate mapping) |
| `anim` | 85%+ (interpolation) |
| `pdf` | 75%+ (layout math) |
| New code | 90%+ |

### TDD Workflow (Red-Green-Refactor)

1. **RED**: Write a failing test that defines the desired behavior
2. **GREEN**: Write minimal code to make the test pass
3. **REFACTOR**: Improve code quality while keeping tests passing

```bash
# Example: Adding exercise validation

# 1. Write test FIRST
# internal/model/exercise_test.go

# 2. Run test (FAILS - validation doesn't exist yet)
go test ./internal/model/...

# 3. Implement minimum code to make test pass

# 4. Run test (PASSES)
go test ./internal/model/...

# 5. Refactor and add more test cases
```

### Test Patterns

**Model test (unit):**
```go
func TestExercise_Validate(t *testing.T) {
    t.Run("valid exercise", func(t *testing.T) {
        ex := &Exercise{
            Name:      "Gauntlet",
            CourtType: FullCourt,
            Sequences: []Sequence{{Label: "Setup"}},
        }
        err := ex.Validate()
        assert.NoError(t, err)
    })

    t.Run("missing name", func(t *testing.T) {
        ex := &Exercise{
            CourtType: FullCourt,
            Sequences: []Sequence{{Label: "Setup"}},
        }
        err := ex.Validate()
        assert.ErrorIs(t, err, ErrMissingName)
    })
}
```

**Store test (integration with temp files):**
```go
func TestYAMLStore_LoadExercise(t *testing.T) {
    // Create temp directory
    dir := t.TempDir()

    // Write test YAML file
    content := []byte(`name: "Test Exercise"\ncourt_type: half_court\n...`)
    os.WriteFile(filepath.Join(dir, "test.yaml"), content, 0644)

    store := NewYAMLStore(dir)
    exercise, err := store.LoadExercise("test")

    assert.NoError(t, err)
    assert.Equal(t, "Test Exercise", exercise.Name)
}
```

**Geometry test (unit):**
```go
func TestFIBACourt_ThreePointArc(t *testing.T) {
    court := NewFIBACourt()
    // Arc at 6.75m from basket center
    assert.InDelta(t, 6.75, court.ThreePointRadius(), 0.01)
}
```

**Interpolation test (unit):**
```go
func TestInterpolate_Position(t *testing.T) {
    from := Position{X: 0.0, Y: 0.0}
    to := Position{X: 1.0, Y: 1.0}

    mid := InterpolatePosition(from, to, 0.5)
    assert.InDelta(t, 0.5, mid.X, 0.001)
    assert.InDelta(t, 0.5, mid.Y, 0.001)
}
```

### Test Best Practices

- Test files next to code: `exercise.go` → `exercise_test.go`
- Use table-driven tests for multiple cases
- Use `t.Run()` for subtests with descriptive names
- Use `testify/assert` for assertions
- Use `t.TempDir()` for file-based tests (auto-cleanup)
- Run tests locally before committing: `go test ./...`

### When to Write Tests

| Scenario | Action |
|----------|--------|
| New feature | ALWAYS write tests first (TDD) |
| Bug fix | Write a test that reproduces the bug, then fix it |
| Refactoring | Ensure existing tests pass after refactoring |
| Code review | No PR should be merged without tests |

### Red Flags

- Writing code first, then retrofitting tests
- Skipping tests because "it's a small change"
- Commenting out failing tests instead of fixing them
- Tests that don't actually test anything meaningful

---

## Development Workflow

### Critical Rule: Test After Code Changes

**IMPORTANT:** Each time you make a code change, you MUST:

1. **Build** to catch compilation errors
2. **Run tests** to ensure nothing is broken

```bash
# After modifying Go code:
go build ./...         # Verify it compiles
go test ./...          # Run all tests

# Or use make:
make build && make test
```

### Make Commands

```bash
# Development
make run              # Run the app
make build            # Build for current platform

# Tests
make test             # All tests
make test-verbose     # Detailed output
make coverage         # Coverage report

# Quality
make lint             # Go linting (golangci-lint)
make vet              # Go vet
make fmt              # Format code (gofmt)

# Build targets
make build-linux      # Linux binary
make build-windows    # Windows binary
make build-android    # Android APK (requires gogio)
make build-ios        # iOS app (requires gogio + macOS)
```

---

## Git Workflow

### Branches

```bash
main              # Stable releases
feature/*         # New features
fix/*             # Bug fixes
```

### Commit Messages

**Format:** `<type>(<scope>): <subject>`

```bash
feat(editor): add drag-and-drop for players
feat(court): implement NBA court markings
fix(anim): correct interpolation at sequence boundaries
fix(pdf): handle exercises with no sequences
refactor(store): simplify YAML loading logic
test(model): add exercise validation tests
docs(specs): update UI specification
chore(deps): update Gio to v0.5.0
```

### Commit Types

| Type | Description |
|------|-------------|
| `feat` | New feature |
| `fix` | Bug fix |
| `refactor` | Refactoring without functional change |
| `test` | Adding/modifying tests |
| `docs` | Documentation |
| `style` | Formatting, typos |
| `perf` | Performance optimization |
| `chore` | Maintenance, dependencies |

---

## Code Review

### Checklist

**Functionality:**
- [ ] Code does what it's supposed to do
- [ ] Error cases handled correctly
- [ ] Edge cases covered

**Architecture:**
- [ ] `model` has zero external dependencies
- [ ] File I/O goes through `Store` interface
- [ ] Dependencies injected, no globals
- [ ] Clear separation of concerns

**Code Quality:**
- [ ] Naming conventions followed
- [ ] No duplicated code (DRY)
- [ ] Short, focused functions (< 50 lines ideally)
- [ ] Useful comments (why, not what)

**Tests:**
- [ ] Unit tests present for new code
- [ ] Coverage ≥ 80% for new code
- [ ] Tests cover success and error cases
- [ ] Tests readable and maintainable

**Coordinates & Rendering:**
- [ ] All positions in model use relative 0.0–1.0 coordinates
- [ ] No pixel/mm values in model or store
- [ ] Coordinate conversion only in court/pdf/ui packages

**Performance:**
- [ ] No unnecessary allocations in render loops
- [ ] Resources freed (defer, Close())
- [ ] YAML files read once and cached, not on every frame

### Review Process

1. **Self-review** before submitting PR
2. **PR description** with clear context
3. **CI checks** pass (tests, linting, coverage)
4. **Peer review** (at least 1 approval)

---

## Related Documentation

- [Architecture](architecture.md)
- [Data Model](data-model.md)
- [Roadmap](roadmap.md)
