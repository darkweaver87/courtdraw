package model

// Session represents an ordered collection of exercises.
type Session struct {
	Title      string          `yaml:"title"`
	Subtitle   string          `yaml:"subtitle,omitempty"`
	AgeGroup   string          `yaml:"age_group,omitempty"`
	CoachNotes []string        `yaml:"coach_notes,omitempty"`
	Philosophy string          `yaml:"philosophy,omitempty"`
	Exercises  []ExerciseEntry `yaml:"exercises"`
}

// ExerciseEntry references an exercise file with optional variants.
type ExerciseEntry struct {
	Exercise string          `yaml:"exercise"`
	Variants []ExerciseEntry `yaml:"variants,omitempty"`
}
