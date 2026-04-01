package editor

import (
	"testing"

	"github.com/darkweaver87/courtdraw/internal/model"
)

func makeExercise(name string) *model.Exercise {
	return &model.Exercise{
		Name:          name,
		CourtType:     model.HalfCourt,
		CourtStandard: model.FIBA,
		Sequences:     []model.Sequence{{Label: "Setup"}},
	}
}

func TestHistory_SaveAndUndo(t *testing.T) {
	h := NewHistory()

	if h.CanUndo() {
		t.Fatal("should not be able to undo with empty history")
	}

	// Initial state + mutation state.
	h.SaveState(makeExercise("initial"))

	if h.CanUndo() {
		t.Fatal("should not be able to undo with only one entry")
	}

	h.SaveState(makeExercise("after-mutation"))

	if !h.CanUndo() {
		t.Fatal("should be able to undo with 2 entries")
	}

	// Undo: pass current, get back the previous state.
	current := makeExercise("after-mutation")
	restored := h.Undo(current)
	if restored == nil {
		t.Fatal("undo returned nil")
	}
	if restored.Name != "initial" {
		t.Fatalf("expected 'initial', got %s", restored.Name)
	}
}

func TestHistory_Redo(t *testing.T) {
	h := NewHistory()
	h.SaveState(makeExercise("v0")) // initial
	h.SaveState(makeExercise("v1")) // after mutation 1
	h.SaveState(makeExercise("v2")) // after mutation 2

	// Stack = [v0, v1, v2]. Current = v2.
	current := makeExercise("v2")
	r := h.Undo(current) // discard v2, return v1. Stack = [v0, v1].
	if r == nil || r.Name != "v1" {
		t.Fatalf("expected v1, got %v", r)
	}

	r = h.Undo(r) // discard v1, return v0. Stack = [v0].
	if r == nil || r.Name != "v0" {
		t.Fatalf("expected v0, got %v", r)
	}

	if !h.CanRedo() {
		t.Fatal("should be able to redo")
	}

	r = h.Redo(r) // redo pops from redo stack (LIFO: v1 first, then v2)
	if r == nil || r.Name != "v1" {
		t.Fatalf("expected v1 on redo, got %v", r)
	}

	r = h.Redo(r)
	if r == nil || r.Name != "v2" {
		t.Fatalf("expected v2 on second redo, got %v", r)
	}

	if h.CanRedo() {
		t.Fatal("should not be able to redo past the end")
	}
}

func TestHistory_NewMutationClearsRedo(t *testing.T) {
	h := NewHistory()
	h.SaveState(makeExercise("v0"))
	h.SaveState(makeExercise("v1"))
	h.SaveState(makeExercise("v2"))

	current := makeExercise("v2")
	h.Undo(current) // back to v1

	// New mutation — should discard redo.
	h.SaveState(makeExercise("v1-modified"))

	if h.CanRedo() {
		t.Fatal("redo should be cleared after new SaveState")
	}
}

func TestHistory_Clear(t *testing.T) {
	h := NewHistory()
	h.SaveState(makeExercise("v0"))

	h.Clear()

	if h.CanUndo() || h.CanRedo() {
		t.Fatal("clear should reset all history")
	}
}

func TestHistory_MaxSize(t *testing.T) {
	h := NewHistory()
	for i := range MaxHistorySize + 10 {
		h.SaveState(makeExercise("v" + string(rune('0'+i%10))))
	}

	if len(h.undoStack) != MaxHistorySize {
		t.Fatalf("expected %d entries, got %d", MaxHistorySize, len(h.undoStack))
	}
}

func TestHistory_NilExercise(t *testing.T) {
	h := NewHistory()
	h.SaveState(nil)
	if h.CanUndo() {
		t.Fatal("saving nil should not create entry")
	}
}

func TestHistory_DeepCopy(t *testing.T) {
	h := NewHistory()
	h.SaveState(makeExercise("initial"))
	ex := makeExercise("original")
	h.SaveState(ex)

	// Mutate the original after saving.
	ex.Name = "mutated"

	restored := h.Undo(ex)
	if restored.Name != "initial" {
		t.Fatalf("snapshot should be a deep copy, got %s", restored.Name)
	}
}
