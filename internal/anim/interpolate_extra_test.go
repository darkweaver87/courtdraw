package anim

import (
	"testing"

	"github.com/darkweaver87/courtdraw/internal/model"
)

func TestInterpolateFrame_NilNil(t *testing.T) {
	frame := InterpolateFrame(nil, nil, 0.5)
	if len(frame.Players) != 0 || len(frame.Accessories) != 0 || len(frame.Actions) != 0 {
		t.Error("nil+nil should return empty frame")
	}
}

func TestInterpolateFrame_NilFrom(t *testing.T) {
	to := &model.Sequence{
		Players: []model.Player{
			{ID: "p1", Position: model.Position{0.5, 0.5}},
		},
	}
	frame := InterpolateFrame(nil, to, 0.5)
	if len(frame.Players) != 1 {
		t.Fatalf("nil from: expected 1 player, got %d", len(frame.Players))
	}
	if !almostEqual(frame.Players[0].Opacity, 1.0, 0.001) {
		t.Errorf("nil from: expected opacity 1.0, got %f", frame.Players[0].Opacity)
	}
}

func TestInterpolateFrame_NilTo(t *testing.T) {
	from := &model.Sequence{
		Players: []model.Player{
			{ID: "p1", Position: model.Position{0.5, 0.5}},
		},
	}
	frame := InterpolateFrame(from, nil, 0.5)
	if len(frame.Players) != 1 {
		t.Fatalf("nil to: expected 1 player, got %d", len(frame.Players))
	}
	if !almostEqual(frame.Players[0].Opacity, 1.0, 0.001) {
		t.Errorf("nil to: expected opacity 1.0, got %f", frame.Players[0].Opacity)
	}
}

func TestInterpolateFrame_NegativeT(t *testing.T) {
	from := &model.Sequence{
		Players: []model.Player{
			{ID: "p1", Position: model.Position{0.1, 0.2}},
		},
	}
	to := &model.Sequence{
		Players: []model.Player{
			{ID: "p1", Position: model.Position{0.9, 0.8}},
		},
	}
	frame := InterpolateFrame(from, to, -0.5)
	if len(frame.Players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(frame.Players))
	}
	// t<0 should clamp to from snapshot.
	if !almostEqual(frame.Players[0].Position[0], 0.1, 0.001) {
		t.Errorf("expected from position x=0.1, got %f", frame.Players[0].Position[0])
	}
}

func TestInterpolateFrame_TGreaterThan1(t *testing.T) {
	from := &model.Sequence{
		Players: []model.Player{
			{ID: "p1", Position: model.Position{0.1, 0.2}},
		},
	}
	to := &model.Sequence{
		Players: []model.Player{
			{ID: "p1", Position: model.Position{0.9, 0.8}},
		},
	}
	frame := InterpolateFrame(from, to, 1.5)
	if len(frame.Players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(frame.Players))
	}
	// t>1 should clamp to snapshot.
	if !almostEqual(frame.Players[0].Position[0], 0.9, 0.001) {
		t.Errorf("expected to position x=0.9, got %f", frame.Players[0].Position[0])
	}
}

func TestInterpolateFrame_MultiplePlayersComplex(t *testing.T) {
	from := &model.Sequence{
		Players: []model.Player{
			{ID: "p1", Position: model.Position{0.0, 0.0}},
			{ID: "p2", Position: model.Position{1.0, 0.0}},
			{ID: "p3", Position: model.Position{0.5, 0.5}}, // only in from
		},
	}
	to := &model.Sequence{
		Players: []model.Player{
			{ID: "p1", Position: model.Position{1.0, 1.0}},
			{ID: "p2", Position: model.Position{0.0, 1.0}},
			{ID: "p4", Position: model.Position{0.5, 0.5}}, // only in to
		},
	}

	frame := InterpolateFrame(from, to, 0.5)
	if len(frame.Players) != 4 {
		t.Fatalf("expected 4 players, got %d", len(frame.Players))
	}

	// Build map by ID.
	m := make(map[string]AnimatedPlayer)
	for _, p := range frame.Players {
		m[p.ID] = p
	}

	// p1: both present → lerp.
	if !almostEqual(m["p1"].Position[0], 0.5, 0.001) {
		t.Errorf("p1 x: got %f, want 0.5", m["p1"].Position[0])
	}
	if !almostEqual(m["p1"].Opacity, 1.0, 0.001) {
		t.Errorf("p1 opacity: got %f, want 1.0", m["p1"].Opacity)
	}

	// p2: both present → lerp.
	if !almostEqual(m["p2"].Position[0], 0.5, 0.001) {
		t.Errorf("p2 x: got %f, want 0.5", m["p2"].Position[0])
	}

	// p3: fade out.
	if !almostEqual(m["p3"].Opacity, 0.5, 0.001) {
		t.Errorf("p3 opacity: got %f, want 0.5", m["p3"].Opacity)
	}

	// p4: fade in.
	if !almostEqual(m["p4"].Opacity, 0.5, 0.001) {
		t.Errorf("p4 opacity: got %f, want 0.5", m["p4"].Opacity)
	}
}

func TestInterpolateFrame_AccessoryFadeOut(t *testing.T) {
	from := &model.Sequence{
		Accessories: []model.Accessory{
			{ID: "acc1", Type: model.AccessoryCone, Position: model.Position{0.3, 0.3}},
		},
	}
	to := &model.Sequence{
		Accessories: []model.Accessory{}, // acc1 removed
	}

	frame := InterpolateFrame(from, to, 0.8)
	if len(frame.Accessories) != 1 {
		t.Fatalf("expected 1 accessory, got %d", len(frame.Accessories))
	}
	if !almostEqual(frame.Accessories[0].Opacity, 0.2, 0.001) {
		t.Errorf("accessory fade-out opacity: got %f, want 0.2", frame.Accessories[0].Opacity)
	}
}

func TestInterpolateFrame_EmptySequences(t *testing.T) {
	from := &model.Sequence{}
	to := &model.Sequence{}

	frame := InterpolateFrame(from, to, 0.5)
	if len(frame.Players) != 0 {
		t.Errorf("expected 0 players, got %d", len(frame.Players))
	}
	if len(frame.Accessories) != 0 {
		t.Errorf("expected 0 accessories, got %d", len(frame.Accessories))
	}
	if len(frame.Actions) != 0 {
		t.Errorf("expected 0 actions, got %d", len(frame.Actions))
	}
}

func TestInterpolateFrame_MultipleActions(t *testing.T) {
	from := &model.Sequence{}
	to := &model.Sequence{
		Actions: []model.Action{
			{Type: model.ActionPass},
			{Type: model.ActionDribble},
			{Type: model.ActionSprint},
		},
	}

	frame := InterpolateFrame(from, to, 0.4)
	if len(frame.Actions) != 3 {
		t.Fatalf("expected 3 actions, got %d", len(frame.Actions))
	}
	for i, a := range frame.Actions {
		if !almostEqual(a.Progress, 0.4, 0.001) {
			t.Errorf("action %d progress: got %f, want 0.4", i, a.Progress)
		}
	}
}

func TestInterpolateFrame_LabelTakenFromTo(t *testing.T) {
	from := &model.Sequence{
		Players: []model.Player{
			{ID: "p1", Label: "Old", Role: model.RoleAttacker, Position: model.Position{0, 0}},
		},
	}
	to := &model.Sequence{
		Players: []model.Player{
			{ID: "p1", Label: "New", Role: model.RoleDefender, Position: model.Position{1, 1}},
		},
	}

	frame := InterpolateFrame(from, to, 0.5)
	if frame.Players[0].Label != "New" {
		t.Errorf("expected label from 'to' sequence, got %q", frame.Players[0].Label)
	}
	if frame.Players[0].Role != model.RoleDefender {
		t.Errorf("expected role from 'to' sequence, got %q", frame.Players[0].Role)
	}
}

func TestInterpolatePosition_Midpoint(t *testing.T) {
	cases := []struct {
		from, to model.Position
		t        float64
		wantX    float64
		wantY    float64
	}{
		{model.Position{0, 0}, model.Position{10, 10}, 0.5, 5, 5},
		{model.Position{0, 0}, model.Position{10, 10}, 0.0, 0, 0},
		{model.Position{0, 0}, model.Position{10, 10}, 1.0, 10, 10},
		{model.Position{0.2, 0.3}, model.Position{0.8, 0.9}, 0.25, 0.35, 0.45},
		{model.Position{1, 1}, model.Position{1, 1}, 0.5, 1, 1}, // same position
	}
	for _, c := range cases {
		got := InterpolatePosition(c.from, c.to, c.t)
		if !almostEqual(got[0], c.wantX, 0.001) || !almostEqual(got[1], c.wantY, 0.001) {
			t.Errorf("InterpolatePosition(%v, %v, %.1f) = %v, want (%.2f, %.2f)",
				c.from, c.to, c.t, got, c.wantX, c.wantY)
		}
	}
}
