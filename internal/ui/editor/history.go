package editor

import (
	"bytes"

	"github.com/darkweaver87/courtdraw/internal/model"
	"gopkg.in/yaml.v3"
)

// MaxHistorySize is the maximum number of snapshots kept in the undo/redo history.
const MaxHistorySize = 50

// History stores serialized exercise snapshots for undo/redo.
// Uses a "save before mutation" pattern: call SaveState before each mutation,
// then Undo restores the last saved state.
type History struct {
	undoStack [][]byte // states before mutations
	redoStack [][]byte // states after undos (for redo)
}

// NewHistory creates an empty History.
func NewHistory() *History {
	return &History{}
}

// SaveState saves the current exercise state before a mutation.
// Call this BEFORE modifying the exercise.
func (h *History) SaveState(ex *model.Exercise) {
	if ex == nil {
		return
	}
	data, err := yaml.Marshal(ex)
	if err != nil {
		return
	}
	// Skip if identical to the last saved state (e.g., selection without actual change).
	if len(h.undoStack) > 0 && bytes.Equal(h.undoStack[len(h.undoStack)-1], data) {
		return
	}
	h.undoStack = append(h.undoStack, data)
	// New mutation invalidates redo stack.
	h.redoStack = nil
	// Cap size.
	if len(h.undoStack) > MaxHistorySize {
		h.undoStack = h.undoStack[len(h.undoStack)-MaxHistorySize:]
	}
}

// Undo restores the previous state. The last entry in undoStack is the current
// state (saved after the last mutation), so we discard it and return the one before.
// Returns nil if nothing to undo.
func (h *History) Undo(current *model.Exercise) *model.Exercise {
	if len(h.undoStack) < 2 || current == nil {
		return nil
	}
	// Save current state for redo.
	if data, err := yaml.Marshal(current); err == nil {
		h.redoStack = append(h.redoStack, data)
	}
	// Discard the last entry (it's the current state).
	h.undoStack = h.undoStack[:len(h.undoStack)-1]
	// Return the previous state (peek, don't pop — it becomes the new "current").
	prev := h.undoStack[len(h.undoStack)-1]
	var ex model.Exercise
	if err := yaml.Unmarshal(prev, &ex); err != nil {
		return nil
	}
	return &ex
}

// Redo restores the state that was undone. Saves current state to undo stack.
// Returns nil if nothing to redo.
func (h *History) Redo(current *model.Exercise) *model.Exercise {
	if len(h.redoStack) == 0 || current == nil {
		return nil
	}
	// Save current state for undo.
	if data, err := yaml.Marshal(current); err == nil {
		h.undoStack = append(h.undoStack, data)
	}
	// Pop last redo state.
	last := h.redoStack[len(h.redoStack)-1]
	h.redoStack = h.redoStack[:len(h.redoStack)-1]
	var ex model.Exercise
	if err := yaml.Unmarshal(last, &ex); err != nil {
		return nil
	}
	return &ex
}

// CanUndo returns true if an undo operation is possible.
// Need at least 2 entries: the current state and a previous state.
func (h *History) CanUndo() bool {
	return len(h.undoStack) >= 2
}

// CanRedo returns true if a redo operation is possible.
func (h *History) CanRedo() bool {
	return len(h.redoStack) > 0
}

// UndoLen returns the undo stack size (for debugging).
func (h *History) UndoLen() int {
	return len(h.undoStack)
}

// Clear resets the history.
func (h *History) Clear() {
	h.undoStack = nil
	h.redoStack = nil
}
