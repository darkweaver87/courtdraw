package model

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Exercise represents a single basketball drill.
type Exercise struct {
	Name          string        `yaml:"name"`
	Description   string        `yaml:"description,omitempty"`
	CourtType     CourtType     `yaml:"court_type"`
	CourtStandard CourtStandard `yaml:"court_standard"`
	Duration      string        `yaml:"duration,omitempty"`
	Intensity     Intensity     `yaml:"intensity,omitempty"`
	Category      Category      `yaml:"category,omitempty"`
	Tags          []string      `yaml:"tags,omitempty"`
	Sequences     []Sequence    `yaml:"sequences"`
}

// Sequence is one chronological step of an exercise.
type Sequence struct {
	Label        string      `yaml:"label,omitempty"`
	Instructions []string    `yaml:"instructions,omitempty"`
	Players      []Player    `yaml:"players,omitempty"`
	Accessories  []Accessory `yaml:"accessories,omitempty"`
	Actions      []Action    `yaml:"actions,omitempty"`
}

// Player is a person on the court.
type Player struct {
	ID        string     `yaml:"id"`
	Label     string     `yaml:"label,omitempty"`
	Role      PlayerRole `yaml:"role"`
	Position  Position   `yaml:"position"`
	Type      string     `yaml:"type,omitempty"`      // "queue" for queued players
	Count     int        `yaml:"count,omitempty"`     // number of players in queue
	Direction string     `yaml:"direction,omitempty"` // queue direction
}

// Action represents a movement or event between elements.
type Action struct {
	Type ActionType `yaml:"type"`
	From ActionRef  `yaml:"from"`
	To   ActionRef  `yaml:"to"`
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

// MarshalYAML implements custom YAML marshalling for ActionRef.
func (r ActionRef) MarshalYAML() (interface{}, error) {
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
