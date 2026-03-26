package model

import (
	"errors"
	"fmt"
	"slices"
	"sort"

	"gopkg.in/yaml.v3"
)

// Exercise represents a single basketball drill.
type Exercise struct {
	Name          string        `yaml:"name"`
	Description   string        `yaml:"description,omitempty"`
	CourtType     CourtType     `yaml:"court_type"`
	CourtStandard CourtStandard `yaml:"court_standard"`
	Orientation   Orientation   `yaml:"orientation,omitempty"`
	Duration      string        `yaml:"duration,omitempty"`
	Intensity     Intensity     `yaml:"intensity,omitempty"`
	Category      Category      `yaml:"category,omitempty"`
	AgeGroup      AgeGroup      `yaml:"age_group,omitempty"`
	Tags          []string      `yaml:"tags,omitempty"`
	Sequences     []Sequence    `yaml:"sequences"`
	I18n          map[string]ExerciseI18n `yaml:"i18n,omitempty"`
}

// ExerciseI18n holds translated text fields for an exercise.
type ExerciseI18n struct {
	Name        string         `yaml:"name,omitempty"`
	Description string         `yaml:"description,omitempty"`
	Tags        []string       `yaml:"tags,omitempty"`
	Sequences   []SequenceI18n `yaml:"sequences,omitempty"`
}

// SequenceI18n holds translated text fields for a sequence.
type SequenceI18n struct {
	Label        string   `yaml:"label,omitempty"`
	Instructions []string `yaml:"instructions,omitempty"`
}

// EnsureI18n returns the ExerciseI18n entry for the given language,
// creating it if necessary. Callers should use SetI18n to write back changes
// since Go maps return copies for struct values.
func (e *Exercise) EnsureI18n(lang string) ExerciseI18n {
	if e.I18n == nil {
		e.I18n = make(map[string]ExerciseI18n)
	}
	return e.I18n[lang]
}

// SetI18n sets the ExerciseI18n entry for the given language.
func (e *Exercise) SetI18n(lang string, tr ExerciseI18n) {
	if e.I18n == nil {
		e.I18n = make(map[string]ExerciseI18n)
	}
	e.I18n[lang] = tr
}

// Localized returns a shallow copy of the exercise with translated text fields
// applied for the given language. Falls back to the original if no translation
// exists. Non-text fields (players, actions, positions) are shared, not copied.
func (e *Exercise) Localized(lang string) *Exercise {
	if lang == "" || lang == "en" || e.I18n == nil {
		return e
	}
	tr, ok := e.I18n[lang]
	if !ok {
		return e
	}
	cp := *e
	if tr.Name != "" {
		cp.Name = tr.Name
	}
	if tr.Description != "" {
		cp.Description = tr.Description
	}
	if len(tr.Tags) > 0 {
		cp.Tags = tr.Tags
	}
	if len(tr.Sequences) > 0 {
		cp.Sequences = make([]Sequence, len(e.Sequences))
		copy(cp.Sequences, e.Sequences)
		for i := range cp.Sequences {
			if i >= len(tr.Sequences) {
				break
			}
			if tr.Sequences[i].Label != "" {
				cp.Sequences[i].Label = tr.Sequences[i].Label
			}
			if len(tr.Sequences[i].Instructions) > 0 {
				cp.Sequences[i].Instructions = tr.Sequences[i].Instructions
			}
		}
	}
	return &cp
}

// Sequence is one chronological step of an exercise.
type Sequence struct {
	Label        string       `yaml:"label,omitempty"`
	Instructions []string     `yaml:"instructions,omitempty"`
	Players      []Player     `yaml:"players,omitempty"`
	Accessories  []Accessory  `yaml:"accessories,omitempty"`
	Actions      []Action     `yaml:"actions,omitempty"`
	BallCarrier  BallCarriers `yaml:"ball_carrier,omitempty"`
}

// BallCarriers holds zero or more player IDs that currently have a ball.
// YAML: accepts a single string ("p1") or a list (["p1","p2"]).
type BallCarriers []string

// HasBall returns true if the given player ID is a ball carrier.
func (bc BallCarriers) HasBall(id string) bool {
	return slices.Contains(bc, id)
}

// AddBall adds a player ID as a ball carrier (no duplicates).
func (bc *BallCarriers) AddBall(id string) {
	if !bc.HasBall(id) {
		*bc = append(*bc, id)
	}
}

// RemoveBall removes a player ID from the ball carriers.
func (bc *BallCarriers) RemoveBall(id string) {
	for i, c := range *bc {
		if c == id {
			*bc = append((*bc)[:i], (*bc)[i+1:]...)
			return
		}
	}
}

// Any returns true if there is at least one ball carrier.
func (bc BallCarriers) Any() bool { return len(bc) > 0 }

// First returns the first ball carrier, or "" if empty.
func (bc BallCarriers) First() string {
	if len(bc) > 0 {
		return bc[0]
	}
	return ""
}

// UnmarshalYAML accepts either a scalar string or a sequence of strings.
func (bc *BallCarriers) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		if value.Value == "" {
			*bc = nil
			return nil
		}
		*bc = BallCarriers{value.Value}
		return nil
	case yaml.SequenceNode:
		var list []string
		if err := value.Decode(&list); err != nil {
			return fmt.Errorf("ball_carrier list: %w", err)
		}
		*bc = list
		return nil
	default:
		return fmt.Errorf("ball_carrier must be a string or list, got %v", value.Kind)
	}
}

// MarshalYAML outputs a single string when len==1, a list otherwise.
// Returns (nil, nil) for empty carriers, which is the standard YAML omitempty pattern.
func (bc BallCarriers) MarshalYAML() (any, error) {
	switch len(bc) {
	case 0:
		return nil, nil //nolint:nilnil // standard MarshalYAML pattern to omit empty field
	case 1:
		return bc[0], nil
	default:
		return []string(bc), nil
	}
}

// Player is a person on the court.
type Player struct {
	ID        string      `yaml:"id"`
	Label     string      `yaml:"label,omitempty"`
	Role      PlayerRole  `yaml:"role"`
	Position  Position    `yaml:"position"`
	Rotation  float64     `yaml:"rotation,omitempty"` // degrees, 0 = facing basket
	Callout   CalloutType `yaml:"callout,omitempty"`
	Type      string      `yaml:"type,omitempty"`      // "queue" for queued players
	Count     int         `yaml:"count,omitempty"`     // number of players in queue
	Direction string      `yaml:"direction,omitempty"` // queue direction
}

// Action is a movement or interaction between elements.
type Action struct {
	Type      ActionType `yaml:"type"`
	From      ActionRef  `yaml:"from"`
	To        ActionRef  `yaml:"to"`
	Step      int        `yaml:"step,omitempty"`      // step order within sequence (1-based, 0 treated as 1)
	Waypoints []Position `yaml:"waypoints,omitempty"` // intermediate curve control points
}

// EffectiveStep returns the step number, treating 0 as 1.
func (a *Action) EffectiveStep() int {
	if a.Step <= 0 {
		return 1
	}
	return a.Step
}

// ReorderSteps compacts step numbers to remove gaps after deletion.
// e.g., steps [1, 3, 4] become [1, 2, 3].
func ReorderSteps(seq *Sequence) {
	if len(seq.Actions) == 0 {
		return
	}
	// Collect unique steps in order.
	seen := make(map[int]bool)
	var steps []int
	for i := range seq.Actions {
		s := seq.Actions[i].EffectiveStep()
		if !seen[s] {
			seen[s] = true
			steps = append(steps, s)
		}
	}
	sort.Ints(steps)
	// Build mapping old → new.
	mapping := make(map[int]int, len(steps))
	for i, s := range steps {
		mapping[s] = i + 1
	}
	// Apply.
	for i := range seq.Actions {
		seq.Actions[i].Step = mapping[seq.Actions[i].EffectiveStep()]
	}
}

// MaxStep returns the highest step number among actions in a sequence.
func MaxStep(seq *Sequence) int {
	m := 0
	for i := range seq.Actions {
		if s := seq.Actions[i].EffectiveStep(); s > m {
			m = s
		}
	}
	return m
}

// ActionRef can be either a player ID (string) or a position ([x,y]).
// In YAML: string → player ID, array of 2 floats → position.
type ActionRef struct {
	PlayerID string
	Position Position
	IsPlayer bool
}

// UnmarshalYAML implements custom YAML unmarshalling for ActionRef.
// Accepts either a string (player ID) or a [float, float] (position).
func (r *ActionRef) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		r.PlayerID = value.Value
		r.IsPlayer = true
		return nil
	case yaml.SequenceNode:
		if len(value.Content) != 2 {
			return fmt.Errorf("action ref position must have exactly 2 elements, got %d", len(value.Content))
		}
		var pos [2]float64
		for i, node := range value.Content {
			var v float64
			if err := node.Decode(&v); err != nil {
				return fmt.Errorf("action ref position[%d]: %w", i, err)
			}
			pos[i] = v
		}
		r.Position = Position(pos)
		r.IsPlayer = false
		return nil
	default:
		return fmt.Errorf("action ref must be a string or [x, y] array, got %v", value.Kind)
	}
}

// MarshalYAML implements custom YAML marshaling for ActionRef.
func (r ActionRef) MarshalYAML() (any, error) {
	if r.IsPlayer {
		return r.PlayerID, nil
	}
	return []float64{r.Position[0], r.Position[1]}, nil
}

// Accessory is a court accessory (cone, ladder, chair).
type Accessory struct {
	Type     AccessoryType `yaml:"type"`
	ID       string        `yaml:"id"`
	Position Position      `yaml:"position"`
	Rotation float64       `yaml:"rotation,omitempty"`
}

// RemapPositionsHalfToFull remaps all position Y values from half court [0,1]
// to the bottom half of full court [0,0.5]. X is unchanged.
func (e *Exercise) RemapPositionsHalfToFull() {
	for si := range e.Sequences {
		seq := &e.Sequences[si]
		for pi := range seq.Players {
			seq.Players[pi].Position[1] *= 0.5
		}
		for ai := range seq.Accessories {
			seq.Accessories[ai].Position[1] *= 0.5
		}
		for ai := range seq.Actions {
			for wi := range seq.Actions[ai].Waypoints {
				seq.Actions[ai].Waypoints[wi][1] *= 0.5
			}
			if !seq.Actions[ai].From.IsPlayer {
				seq.Actions[ai].From.Position[1] *= 0.5
			}
			if !seq.Actions[ai].To.IsPlayer {
				seq.Actions[ai].To.Position[1] *= 0.5
			}
		}
	}
}

// RemapPositionsFullToHalf remaps all position Y values from full court to half court.
// If bottom is true, maps [0,0.5] → [0,1] (bottom half).
// If bottom is false, maps [0.5,1] → [0,1] with mirroring (top half).
func (e *Exercise) RemapPositionsFullToHalf(bottom bool) {
	for si := range e.Sequences {
		seq := &e.Sequences[si]
		for pi := range seq.Players {
			seq.Players[pi].Position[1] = remapFullToHalfY(seq.Players[pi].Position[1], bottom)
		}
		for ai := range seq.Accessories {
			seq.Accessories[ai].Position[1] = remapFullToHalfY(seq.Accessories[ai].Position[1], bottom)
		}
		for ai := range seq.Actions {
			for wi := range seq.Actions[ai].Waypoints {
				seq.Actions[ai].Waypoints[wi][1] = remapFullToHalfY(seq.Actions[ai].Waypoints[wi][1], bottom)
			}
			if !seq.Actions[ai].From.IsPlayer {
				seq.Actions[ai].From.Position[1] = remapFullToHalfY(seq.Actions[ai].From.Position[1], bottom)
			}
			if !seq.Actions[ai].To.IsPlayer {
				seq.Actions[ai].To.Position[1] = remapFullToHalfY(seq.Actions[ai].To.Position[1], bottom)
			}
		}
	}
}

func remapFullToHalfY(y float64, bottom bool) float64 {
	if bottom {
		return y * 2.0
	}
	return (1.0 - y) * 2.0
}

// FullCourtPlayerHalf scans all position Y values across all sequences and returns:
//   - "bottom" if all Y ≤ 0.5
//   - "top" if all Y ≥ 0.5
//   - "mixed" if elements span both halves
//   - "bottom" if there are no positioned elements (default)
func (e *Exercise) FullCourtPlayerHalf() string {
	hasBottom := false // Y < 0.5
	hasTop := false    // Y > 0.5

	check := func(y float64) {
		if y < 0.5 {
			hasBottom = true
		}
		if y > 0.5 {
			hasTop = true
		}
		// Y == 0.5 is compatible with both halves.
	}

	for _, seq := range e.Sequences {
		for _, p := range seq.Players {
			check(p.Position[1])
		}
		for _, a := range seq.Accessories {
			check(a.Position[1])
		}
		for _, act := range seq.Actions {
			for _, wp := range act.Waypoints {
				check(wp[1])
			}
			if !act.From.IsPlayer {
				check(act.From.Position[1])
			}
			if !act.To.IsPlayer {
				check(act.To.Position[1])
			}
		}
	}

	if hasBottom && hasTop {
		return "mixed"
	}
	if hasTop {
		return "top"
	}
	return "bottom"
}

// Sentinel errors for validation.
var (
	ErrNoName      = errors.New("exercise name is required")
	ErrNoSequences = errors.New("exercise must have at least one sequence")
	ErrNoCourtType = errors.New("court type is required")
	ErrNoStandard  = errors.New("court standard is required")
	ErrNoPlayerID  = errors.New("player id is required")
)

// Validate checks the exercise for required fields and consistency.
func (e *Exercise) Validate() error {
	if e.Name == "" {
		return ErrNoName
	}
	if e.CourtType != HalfCourt && e.CourtType != FullCourt {
		return ErrNoCourtType
	}
	if e.CourtStandard != FIBA && e.CourtStandard != NBA {
		return ErrNoStandard
	}
	if len(e.Sequences) == 0 {
		return ErrNoSequences
	}
	for si, seq := range e.Sequences {
		for pi, p := range seq.Players {
			if p.ID == "" {
				return fmt.Errorf("sequence %d, player %d: %w", si, pi, ErrNoPlayerID)
			}
		}
	}
	return nil
}
