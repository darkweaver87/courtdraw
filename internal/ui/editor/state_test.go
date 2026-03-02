package editor

import (
	"testing"

	"github.com/darkweaver87/courtdraw/internal/model"
)

func TestNextPlayerID(t *testing.T) {
	seq := &model.Sequence{
		Players: []model.Player{
			{ID: "p1"},
			{ID: "p3"},
			{ID: "p2"},
		},
	}
	got := NextPlayerID(seq)
	if got != "p4" {
		t.Errorf("NextPlayerID = %q, want %q", got, "p4")
	}
}

func TestNextPlayerID_Empty(t *testing.T) {
	seq := &model.Sequence{}
	got := NextPlayerID(seq)
	if got != "p1" {
		t.Errorf("NextPlayerID = %q, want %q", got, "p1")
	}
}

func TestNextAccessoryID(t *testing.T) {
	seq := &model.Sequence{
		Accessories: []model.Accessory{
			{ID: "acc1"},
			{ID: "acc2"},
		},
	}
	got := NextAccessoryID(seq)
	if got != "acc3" {
		t.Errorf("NextAccessoryID = %q, want %q", got, "acc3")
	}
}

func TestEditorState_Select(t *testing.T) {
	s := EditorState{}
	s.Select(SelectPlayer, 2, 0)
	if s.SelectedElement == nil {
		t.Fatal("SelectedElement should not be nil")
	}
	if s.SelectedElement.Kind != SelectPlayer || s.SelectedElement.Index != 2 {
		t.Errorf("selection = %+v, want player index 2", s.SelectedElement)
	}
}

func TestEditorState_Deselect(t *testing.T) {
	s := EditorState{}
	s.Select(SelectPlayer, 0, 0)
	s.Deselect()
	if s.SelectedElement != nil {
		t.Error("SelectedElement should be nil after Deselect")
	}
}

func TestEditorState_SetTool(t *testing.T) {
	s := EditorState{}
	id := "p1"
	s.ActionFrom = &id
	s.SetTool(ToolDelete)
	if s.ActiveTool != ToolDelete {
		t.Errorf("ActiveTool = %d, want ToolDelete", s.ActiveTool)
	}
	if s.ActionFrom != nil {
		t.Error("ActionFrom should be nil after SetTool")
	}
}

func TestEditorState_SetPlayerTool(t *testing.T) {
	s := EditorState{}
	s.SetPlayerTool(model.RoleDefender)
	if s.ActiveTool != ToolPlayer {
		t.Errorf("ActiveTool = %d, want ToolPlayer", s.ActiveTool)
	}
	if s.ToolRole != model.RoleDefender {
		t.Errorf("ToolRole = %s, want defender", s.ToolRole)
	}
}

func TestEditorState_Modified(t *testing.T) {
	s := EditorState{}
	if s.Modified {
		t.Error("should not be modified initially")
	}
	s.MarkModified()
	if !s.Modified {
		t.Error("should be modified after MarkModified")
	}
	s.ClearModified()
	if s.Modified {
		t.Error("should not be modified after ClearModified")
	}
}
