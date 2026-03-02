package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/darkweaver87/courtdraw/internal/model"
)

func testExercise() *model.Exercise {
	return &model.Exercise{
		Name:          "Test Drill",
		CourtType:     model.HalfCourt,
		CourtStandard: model.FIBA,
		Sequences: []model.Sequence{
			{
				Label: "Setup",
				Players: []model.Player{
					{ID: "a1", Role: model.RoleAttacker, Position: model.Position{0.5, 0.5}},
				},
			},
		},
	}
}

func TestYAMLStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	s, err := NewYAMLStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	ex := testExercise()
	if err := s.SaveExercise(ex); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := s.LoadExercise("test-drill")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Name != "Test Drill" {
		t.Fatalf("name: %s != Test Drill", loaded.Name)
	}
	if len(loaded.Sequences) != 1 {
		t.Fatalf("sequences: %d != 1", len(loaded.Sequences))
	}
}

func TestYAMLStore_ListExercises(t *testing.T) {
	dir := t.TempDir()
	s, err := NewYAMLStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	// empty at first
	names, err := s.ListExercises()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("expected 0 exercises, got %d", len(names))
	}

	// save one
	if err := s.SaveExercise(testExercise()); err != nil {
		t.Fatalf("save: %v", err)
	}

	names, err = s.ListExercises()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(names) != 1 || names[0] != "test-drill" {
		t.Fatalf("expected [test-drill], got %v", names)
	}
}

func TestYAMLStore_LoadNotFound(t *testing.T) {
	dir := t.TempDir()
	s, err := NewYAMLStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	_, err = s.LoadExercise("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing exercise")
	}
}

func TestYAMLStore_Sessions(t *testing.T) {
	dir := t.TempDir()
	s, err := NewYAMLStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	ses := &model.Session{
		Title:    "Test Session",
		AgeGroup: "U15",
		Exercises: []model.ExerciseEntry{
			{Exercise: "test-drill"},
		},
	}

	if err := s.SaveSession(ses); err != nil {
		t.Fatalf("save session: %v", err)
	}

	loaded, err := s.LoadSession("test-session")
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if loaded.Title != "Test Session" {
		t.Fatalf("title: %s != Test Session", loaded.Title)
	}

	names, err := s.ListSessions()
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(names) != 1 || names[0] != "test-session" {
		t.Fatalf("expected [test-session], got %v", names)
	}
}

func TestYAMLStore_CreatesDirs(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "nested", "courtdraw")

	s, err := NewYAMLStore(subDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	// directories should exist
	if _, err := os.Stat(filepath.Join(subDir, "exercises")); err != nil {
		t.Fatalf("exercises dir not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(subDir, "sessions")); err != nil {
		t.Fatalf("sessions dir not created: %v", err)
	}

	// should be able to save
	if err := s.SaveExercise(testExercise()); err != nil {
		t.Fatalf("save after dir creation: %v", err)
	}
}

func TestLibrary_ListAndLoad(t *testing.T) {
	dir := t.TempDir()
	lib := NewLibrary(dir)

	// empty
	names, err := lib.ListExercises()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("expected 0, got %d", len(names))
	}

	// write a file directly
	ex := testExercise()
	data := []byte("name: Test Drill\ncourt_type: half_court\ncourt_standard: fiba\nsequences:\n  - label: Setup\n    players:\n      - id: a1\n        role: attacker\n        position: [0.5, 0.5]\n")
	if err := os.WriteFile(filepath.Join(dir, "test-drill.yaml"), data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = ex

	names, err = lib.ListExercises()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(names) != 1 || names[0] != "test-drill" {
		t.Fatalf("expected [test-drill], got %v", names)
	}

	loaded, err := lib.LoadExercise("test-drill")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Name != "Test Drill" {
		t.Fatalf("name: %s", loaded.Name)
	}
}

func TestLibrary_MissingDir(t *testing.T) {
	lib := NewLibrary("/nonexistent/path")
	names, err := lib.ListExercises()
	if err != nil {
		t.Fatalf("expected nil error for missing dir, got: %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("expected 0 exercises for missing dir, got %d", len(names))
	}
}

func TestToKebab(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Test Drill", "test-drill"},
		{"Double Close-Out", "double-close-out"},
		{"  Spaces  ", "spaces"},
		{"UPPERCASE", "uppercase"},
		{"a--b", "a-b"},
	}
	for _, tt := range tests {
		got := ToKebab(tt.input)
		if got != tt.want {
			t.Errorf("ToKebab(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
