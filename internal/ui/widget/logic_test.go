package widget

import (
	"fmt"
	"testing"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
)

func init() {
	i18n.Load()
	i18n.SetLang(i18n.EN)
}

func TestParseDurationMinutes(t *testing.T) {
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
		{"0m", 0},
		{"10m", 10},
		{"3h", 180},
		{"1h1m", 61},
	}
	for _, tt := range tests {
		got := parseDurationMinutes(tt.input)
		if got != tt.expected {
			t.Errorf("parseDurationMinutes(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestIntensityDots(t *testing.T) {
	tests := []struct {
		n        int
		expected string
	}{
		{0, ""},
		{1, "●"},
		{2, "●●"},
		{3, "●●●"},
	}
	for _, tt := range tests {
		got := intensityDots(tt.n)
		// intensityDots always returns 3 characters (filled + empty).
		if len([]rune(got)) != 3 {
			t.Errorf("intensityDots(%d) = %q, expected 3 runes", tt.n, got)
		}
	}

	// Verify filled/empty counts.
	for n := 0; n <= 3; n++ {
		got := intensityDots(n)
		runes := []rune(got)
		filled := 0
		for _, r := range runes {
			if r == '●' {
				filled++
			}
		}
		if filled != n {
			t.Errorf("intensityDots(%d): %d filled dots, want %d", n, filled, n)
		}
	}
}

func TestNextCategoryWithAll(t *testing.T) {
	// Should cycle: "" → warmup → offense → defense → transition → scrimmage → cooldown → ""
	expected := []model.Category{
		"",
		model.CategoryWarmup,
		model.CategoryOffense,
		model.CategoryDefense,
		model.CategoryTransition,
		model.CategoryScrimmage,
		model.CategoryCooldown,
	}

	current := model.Category("")
	for _, exp := range expected[1:] {
		next := nextCategoryWithAll(current)
		if next != exp {
			t.Errorf("nextCategoryWithAll(%q) = %q, want %q", current, next, exp)
		}
		current = next
	}

	// After cooldown, should wrap back to "".
	final := nextCategoryWithAll(model.CategoryCooldown)
	if final != "" {
		t.Errorf("nextCategoryWithAll(cooldown) = %q, want \"\"", final)
	}
}

func TestNextCategoryWithAll_UnknownCategory(t *testing.T) {
	got := nextCategoryWithAll(model.Category("unknown"))
	if got != "" {
		t.Errorf("nextCategoryWithAll(unknown) = %q, want \"\"", got)
	}
}

func TestNextCategory(t *testing.T) {
	// nextCategory in propspanel.go cycles through non-empty categories.
	current := model.CategoryWarmup
	seen := make(map[model.Category]bool)
	for i := 0; i < 10; i++ {
		next := nextCategory(current)
		if next == "" {
			t.Errorf("nextCategory should never return empty, got empty from %q", current)
		}
		seen[next] = true
		current = next
	}
	// Should have seen all 6 categories.
	if len(seen) < 6 {
		t.Errorf("expected to see all 6 categories, only saw %d: %v", len(seen), seen)
	}
}

func TestActionColor(t *testing.T) {
	// Verify actionColor returns non-zero alpha for known types.
	types := []model.ActionType{
		model.ActionPass,
		model.ActionDribble,
		model.ActionSprint,
		model.ActionShotLayup,
		model.ActionScreen,
		model.ActionCut,
		model.ActionCloseOut,
		model.ActionContest,
		model.ActionReverse,
	}
	for _, at := range types {
		col := actionColor(at)
		if col.A == 0 {
			t.Errorf("actionColor(%q) returned zero alpha", at)
		}
	}

	// Unknown type should still return something usable.
	col := actionColor(model.ActionType("unknown"))
	if col.A == 0 {
		t.Error("actionColor(unknown) returned zero alpha")
	}
}

func TestRefString(t *testing.T) {
	// Player reference.
	ref := model.ActionRef{IsPlayer: true, PlayerID: "p1"}
	got := refString(ref)
	if got != "p1" {
		t.Errorf("refString(player) = %q, want \"p1\"", got)
	}

	// Position reference.
	ref2 := model.ActionRef{IsPlayer: false, Position: model.Position{0.5, 0.75}}
	got2 := refString(ref2)
	if got2 == "" {
		t.Error("refString(position) returned empty string")
	}
}

func TestSessionTab_ComputeTotalDuration(t *testing.T) {
	st := NewSessionTab()
	st.SetSession(&model.Session{
		Title: "Test",
		Exercises: []model.ExerciseEntry{
			{Exercise: "ex1"},
			{Exercise: "ex2"},
		},
	})
	st.SetResolvedExercises(map[string]*model.Exercise{
		"ex1": {Name: "Ex1", Duration: "15m"},
		"ex2": {Name: "Ex2", Duration: "1h"},
	})

	got := st.computeTotalDuration()
	if got != "1h15m" {
		t.Errorf("computeTotalDuration() = %q, want \"1h15m\"", got)
	}
}

func TestSessionTab_ComputeTotalDuration_NoExercises(t *testing.T) {
	st := NewSessionTab()
	st.SetSession(&model.Session{Title: "Empty"})

	got := st.computeTotalDuration()
	if got != "N/A" {
		t.Errorf("computeTotalDuration() = %q, want \"N/A\"", got)
	}
}

func TestSessionTab_ComputeTotalDuration_Short(t *testing.T) {
	st := NewSessionTab()
	st.SetSession(&model.Session{
		Exercises: []model.ExerciseEntry{{Exercise: "ex1"}},
	})
	st.SetResolvedExercises(map[string]*model.Exercise{
		"ex1": {Name: "Ex1", Duration: "10m"},
	})

	got := st.computeTotalDuration()
	if got != "10m" {
		t.Errorf("computeTotalDuration() = %q, want \"10m\"", got)
	}
}

func TestSessionTab_AddExercise_BoundsCheck(t *testing.T) {
	st := NewSessionTab()
	st.SetSession(&model.Session{Title: "Test"})

	// Add maxSessionItems exercises (unique names).
	for i := 0; i < maxSessionItems; i++ {
		st.addExerciseByRef(fmt.Sprintf("test-%d", i))
	}
	if len(st.session.Exercises) != maxSessionItems {
		t.Fatalf("expected %d exercises, got %d", maxSessionItems, len(st.session.Exercises))
	}

	// Adding one more should be a no-op.
	st.addExerciseByRef("overflow")
	if len(st.session.Exercises) != maxSessionItems {
		t.Errorf("expected %d exercises after overflow, got %d", maxSessionItems, len(st.session.Exercises))
	}
}

func TestSessionTab_FilteredExercises(t *testing.T) {
	st := NewSessionTab()

	items := []ManagedExercise{
		{Name: "fast-break", DisplayName: "Fast Break", Category: string(model.CategoryTransition), Status: StatusLocalOnly},
		{Name: "shell-drill", DisplayName: "Shell Drill", Category: string(model.CategoryDefense), Status: StatusRemoteOnly},
		{Name: "layup-lines", DisplayName: "Layup Lines", Category: string(model.CategoryWarmup), Status: StatusSynced},
	}
	st.SetExercises(items)

	// No filter: all exercises.
	st.filterIndex = 0
	st.filterCategory = ""
	st.searchEditor.SetText("")
	got := st.filteredExercises()
	if len(got) != 3 {
		t.Errorf("unfiltered: expected 3, got %d", len(got))
	}

	// Category filter.
	st.filterCategory = model.CategoryDefense
	got = st.filteredExercises()
	if len(got) != 1 || got[0].DisplayName != "Shell Drill" {
		t.Errorf("defense filter: expected 1 Shell Drill, got %d", len(got))
	}

	// Search filter.
	st.filterCategory = ""
	st.searchEditor.SetText("layup")
	got = st.filteredExercises()
	if len(got) != 1 || got[0].DisplayName != "Layup Lines" {
		t.Errorf("search 'layup': expected 1 Layup Lines, got %d", len(got))
	}

	// Status filter.
	st.searchEditor.SetText("")
	st.filterIndex = 2 // Community
	got = st.filteredExercises()
	if len(got) != 1 || got[0].DisplayName != "Shell Drill" {
		t.Errorf("community filter: expected 1 Shell Drill, got %d", len(got))
	}

	// Combined: category + search with no match.
	st.filterIndex = 0
	st.filterCategory = model.CategoryWarmup
	st.searchEditor.SetText("defense")
	got = st.filteredExercises()
	if len(got) != 0 {
		t.Errorf("warmup+defense: expected 0, got %d", len(got))
	}
}

func TestSessionTab_SetSession_Resets(t *testing.T) {
	st := NewSessionTab()
	st.modified = true

	st.SetSession(&model.Session{Title: "New"})
	if st.modified {
		t.Error("SetSession should clear modified flag")
	}
}

func TestSessionListOverlay_ShowHide(t *testing.T) {
	slo := NewSessionListOverlay()
	if slo.Visible {
		t.Error("should start hidden")
	}

	slo.Show([]string{"session1", "session2"})
	if !slo.Visible {
		t.Error("should be visible after Show")
	}
	if len(slo.names) != 2 {
		t.Errorf("expected 2 names, got %d", len(slo.names))
	}

	slo.Hide()
	if slo.Visible {
		t.Error("should be hidden after Hide")
	}
}
