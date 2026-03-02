package pdf

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/store"
)

func TestGenerate_BasicSession(t *testing.T) {
	// Create temp store with an exercise.
	dir := t.TempDir()
	st, err := store.NewYAMLStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	ex := &model.Exercise{
		Name:          "Test Exercise",
		CourtType:     model.HalfCourt,
		CourtStandard: model.FIBA,
		Duration:      "15m",
		Intensity:     model.IntensityMax,
		Category:      model.CategoryDefense,
		Sequences: []model.Sequence{
			{
				Label: "Setup",
				Instructions: []string{
					"Player A on the wing",
					"Player D on the baseline",
				},
				Players: []model.Player{
					{ID: "p1", Label: "A", Role: model.RoleAttacker, Position: model.Position{0.2, 0.5}},
					{ID: "p2", Label: "D", Role: model.RoleDefender, Position: model.Position{0.5, 0.1}},
				},
				Actions: []model.Action{
					{
						Type: model.ActionPass,
						From: model.ActionRef{IsPlayer: true, PlayerID: "p1"},
						To:   model.ActionRef{IsPlayer: true, PlayerID: "p2"},
					},
				},
			},
		},
	}
	if err := st.SaveExercise(ex); err != nil {
		t.Fatalf("save exercise: %v", err)
	}

	session := &model.Session{
		Title:    "Test Session",
		Subtitle: "A test session",
		AgeGroup: "U15",
		CoachNotes: []string{
			"Stay hydrated",
			"Focus on technique",
		},
		Philosophy: "Hard work beats talent.",
		Exercises: []model.ExerciseEntry{
			{Exercise: "test-exercise"},
		},
	}

	outputPath := filepath.Join(dir, "test-output.pdf")
	err = Generate(session, st.LoadExercise,outputPath)
	if err != nil {
		t.Fatalf("generate PDF: %v", err)
	}

	// Verify the file exists and has content.
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("stat output: %v", err)
	}
	if info.Size() < 100 {
		t.Errorf("PDF file too small: %d bytes", info.Size())
	}
}

func TestGenerate_EmptySession(t *testing.T) {
	dir := t.TempDir()
	st, err := store.NewYAMLStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	session := &model.Session{
		Title: "Empty Session",
	}

	outputPath := filepath.Join(dir, "empty.pdf")
	err = Generate(session, st.LoadExercise,outputPath)
	if err != nil {
		t.Fatalf("generate PDF: %v", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("stat output: %v", err)
	}
	if info.Size() < 100 {
		t.Errorf("PDF file too small: %d bytes", info.Size())
	}
}

func TestGenerate_NilSession(t *testing.T) {
	dir := t.TempDir()
	st, _ := store.NewYAMLStore(dir)
	err := Generate(nil, st.LoadExercise, filepath.Join(dir, "nil.pdf"))
	if err == nil {
		t.Error("expected error for nil session")
	}
}

func TestParseDurationMins(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"15m", 15},
		{"1h30m", 90},
		{"1h", 60},
		{"2h15m", 135},
		{"", 0},
		{"5m", 5},
	}
	for _, tt := range tests {
		got := parseDurationMins(tt.input)
		if got != tt.expected {
			t.Errorf("parseDurationMins(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestIntensityDotsStr(t *testing.T) {
	tests := []struct {
		n        int
		expected string
	}{
		{0, "[---]"},
		{1, "[*--]"},
		{2, "[**-]"},
		{3, "[***]"},
	}
	for _, tt := range tests {
		got := intensityDotsStr(tt.n)
		if got != tt.expected {
			t.Errorf("intensityDotsStr(%d) = %q, want %q", tt.n, got, tt.expected)
		}
	}
}

func TestGenerateBytes(t *testing.T) {
	dir := t.TempDir()
	st, err := store.NewYAMLStore(dir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	session := &model.Session{
		Title: "Bytes Test",
	}

	data, err := GenerateBytes(session, st.LoadExercise)
	if err != nil {
		t.Fatalf("generate bytes: %v", err)
	}
	if len(data) < 100 {
		t.Errorf("PDF bytes too small: %d", len(data))
	}
}
