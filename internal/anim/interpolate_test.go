package anim

import (
	"math"
	"testing"

	"github.com/darkweaver87/courtdraw/internal/model"
)

func almostEqual(a, b, tol float64) bool {
	return math.Abs(a-b) < tol
}

func TestInterpolatePosition(t *testing.T) {
	from := model.Position{0.0, 0.0}
	to := model.Position{1.0, 1.0}

	mid := InterpolatePosition(from, to, 0.5)
	if !almostEqual(mid[0], 0.5, 0.001) || !almostEqual(mid[1], 0.5, 0.001) {
		t.Errorf("expected (0.5, 0.5), got (%f, %f)", mid[0], mid[1])
	}

	start := InterpolatePosition(from, to, 0.0)
	if !almostEqual(start[0], 0.0, 0.001) || !almostEqual(start[1], 0.0, 0.001) {
		t.Errorf("expected (0.0, 0.0), got (%f, %f)", start[0], start[1])
	}

	end := InterpolatePosition(from, to, 1.0)
	if !almostEqual(end[0], 1.0, 0.001) || !almostEqual(end[1], 1.0, 0.001) {
		t.Errorf("expected (1.0, 1.0), got (%f, %f)", end[0], end[1])
	}
}

func TestInterpolateFrame_BothPresent(t *testing.T) {
	from := &model.Sequence{
		Players: []model.Player{
			{ID: "p1", Label: "A", Role: model.RoleAttacker, Position: model.Position{0.0, 0.0}},
		},
	}
	to := &model.Sequence{
		Players: []model.Player{
			{ID: "p1", Label: "A", Role: model.RoleAttacker, Position: model.Position{1.0, 1.0}},
		},
	}

	frame := InterpolateFrame(from, to, 0.5)
	if len(frame.Players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(frame.Players))
	}
	p := frame.Players[0]
	if !almostEqual(p.Position[0], 0.5, 0.001) || !almostEqual(p.Position[1], 0.5, 0.001) {
		t.Errorf("expected (0.5, 0.5), got (%f, %f)", p.Position[0], p.Position[1])
	}
	if !almostEqual(p.Opacity, 1.0, 0.001) {
		t.Errorf("expected opacity 1.0, got %f", p.Opacity)
	}
}

func TestInterpolateFrame_FadeOut(t *testing.T) {
	from := &model.Sequence{
		Players: []model.Player{
			{ID: "p1", Label: "A", Role: model.RoleAttacker, Position: model.Position{0.5, 0.5}},
		},
	}
	to := &model.Sequence{
		Players: []model.Player{}, // p1 removed
	}

	frame := InterpolateFrame(from, to, 0.75)
	if len(frame.Players) != 1 {
		t.Fatalf("expected 1 player (fading out), got %d", len(frame.Players))
	}
	p := frame.Players[0]
	if !almostEqual(p.Opacity, 0.25, 0.001) {
		t.Errorf("expected opacity 0.25, got %f", p.Opacity)
	}
}

func TestInterpolateFrame_FadeIn(t *testing.T) {
	from := &model.Sequence{
		Players: []model.Player{},
	}
	to := &model.Sequence{
		Players: []model.Player{
			{ID: "p2", Label: "D", Role: model.RoleDefender, Position: model.Position{0.5, 0.5}},
		},
	}

	frame := InterpolateFrame(from, to, 0.3)
	if len(frame.Players) != 1 {
		t.Fatalf("expected 1 player (fading in), got %d", len(frame.Players))
	}
	p := frame.Players[0]
	if !almostEqual(p.Opacity, 0.3, 0.001) {
		t.Errorf("expected opacity 0.3, got %f", p.Opacity)
	}
}

func TestInterpolateFrame_Accessories(t *testing.T) {
	from := &model.Sequence{
		Accessories: []model.Accessory{
			{ID: "acc1", Type: model.AccessoryCone, Position: model.Position{0.2, 0.3}},
		},
	}
	to := &model.Sequence{
		Accessories: []model.Accessory{
			{ID: "acc1", Type: model.AccessoryCone, Position: model.Position{0.2, 0.3}},
			{ID: "acc2", Type: model.AccessoryChair, Position: model.Position{0.8, 0.8}},
		},
	}

	frame := InterpolateFrame(from, to, 0.5)
	if len(frame.Accessories) != 2 {
		t.Fatalf("expected 2 accessories, got %d", len(frame.Accessories))
	}
	// acc1 present in both: opacity 1.0
	if !almostEqual(frame.Accessories[0].Opacity, 1.0, 0.001) {
		t.Errorf("acc1 expected opacity 1.0, got %f", frame.Accessories[0].Opacity)
	}
	// acc2 fading in: opacity 0.5
	if !almostEqual(frame.Accessories[1].Opacity, 0.5, 0.001) {
		t.Errorf("acc2 expected opacity 0.5, got %f", frame.Accessories[1].Opacity)
	}
}

func TestInterpolateFrame_Actions(t *testing.T) {
	from := &model.Sequence{}
	to := &model.Sequence{
		Actions: []model.Action{
			{Type: model.ActionPass, From: model.ActionRef{IsPlayer: true, PlayerID: "p1"}, To: model.ActionRef{IsPlayer: true, PlayerID: "p2"}},
		},
	}

	frame := InterpolateFrame(from, to, 0.6)
	if len(frame.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(frame.Actions))
	}
	if !almostEqual(frame.Actions[0].Progress, 0.6, 0.001) {
		t.Errorf("expected progress 0.6, got %f", frame.Actions[0].Progress)
	}
}

func TestInterpolateFrame_BoundaryT0(t *testing.T) {
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

	frame := InterpolateFrame(from, to, 0.0)
	if len(frame.Players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(frame.Players))
	}
	p := frame.Players[0]
	if !almostEqual(p.Position[0], 0.1, 0.001) || !almostEqual(p.Position[1], 0.2, 0.001) {
		t.Errorf("expected from position, got (%f, %f)", p.Position[0], p.Position[1])
	}
}

func TestInterpolateFrame_BoundaryT1(t *testing.T) {
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

	frame := InterpolateFrame(from, to, 1.0)
	if len(frame.Players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(frame.Players))
	}
	p := frame.Players[0]
	if !almostEqual(p.Position[0], 0.9, 0.001) || !almostEqual(p.Position[1], 0.8, 0.001) {
		t.Errorf("expected to position, got (%f, %f)", p.Position[0], p.Position[1])
	}
}

func TestInterpolateRotation(t *testing.T) {
	tests := []struct {
		name     string
		from, to float64
		t        float64
		expected float64
	}{
		{"0 to 90 at 0.5", 0, 90, 0.5, 45},
		{"350 to 10 wrap forward at 0.5", 350, 10, 0.5, 0},
		{"10 to 350 wrap backward at 0.5", 10, 350, 0.5, 0},
		{"same angle", 45, 45, 0.5, 45},
		{"0 to 180 at 0.5", 0, 180, 0.5, 90},
		{"full range at t=0", 0, 90, 0.0, 0},
		{"full range at t=1", 0, 90, 1.0, 90},
		{"270 to 90 at 0.5", 270, 90, 0.5, 180}, // 180° diff: both directions equal, algo goes clockwise
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := InterpolateRotation(tc.from, tc.to, tc.t)
			if !almostEqual(result, tc.expected, 0.01) {
				t.Errorf("InterpolateRotation(%v, %v, %v) = %v, want %v",
					tc.from, tc.to, tc.t, result, tc.expected)
			}
		})
	}
}

func TestInterpolateFrame_PlayerRotation(t *testing.T) {
	from := &model.Sequence{
		Players: []model.Player{
			{ID: "p1", Role: model.RoleAttacker, Position: model.Position{0.5, 0.5}, Rotation: 0},
		},
	}
	to := &model.Sequence{
		Players: []model.Player{
			{ID: "p1", Role: model.RoleAttacker, Position: model.Position{0.5, 0.5}, Rotation: 90},
		},
	}

	frame := InterpolateFrame(from, to, 0.5)
	if len(frame.Players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(frame.Players))
	}
	if !almostEqual(frame.Players[0].Rotation, 45, 0.01) {
		t.Errorf("expected rotation 45, got %f", frame.Players[0].Rotation)
	}
}

func TestInterpolateFrame_AccessoryPositionAndRotation(t *testing.T) {
	from := &model.Sequence{
		Accessories: []model.Accessory{
			{ID: "acc1", Type: model.AccessoryCone, Position: model.Position{0.2, 0.3}, Rotation: 0},
		},
	}
	to := &model.Sequence{
		Accessories: []model.Accessory{
			{ID: "acc1", Type: model.AccessoryCone, Position: model.Position{0.8, 0.7}, Rotation: 90},
		},
	}

	frame := InterpolateFrame(from, to, 0.5)
	if len(frame.Accessories) != 1 {
		t.Fatalf("expected 1 accessory, got %d", len(frame.Accessories))
	}
	a := frame.Accessories[0]
	if !almostEqual(a.Position[0], 0.5, 0.01) || !almostEqual(a.Position[1], 0.5, 0.01) {
		t.Errorf("expected position (0.5, 0.5), got (%f, %f)", a.Position[0], a.Position[1])
	}
	if !almostEqual(a.Rotation, 45, 0.01) {
		t.Errorf("expected rotation 45, got %f", a.Rotation)
	}
}

func TestSnapshotFrame(t *testing.T) {
	seq := &model.Sequence{
		Players: []model.Player{
			{ID: "p1", Position: model.Position{0.5, 0.5}},
		},
		Accessories: []model.Accessory{
			{ID: "acc1", Type: model.AccessoryCone, Position: model.Position{0.3, 0.3}},
		},
		Actions: []model.Action{
			{Type: model.ActionPass},
		},
	}

	frame := snapshotFrame(seq)
	if len(frame.Players) != 1 {
		t.Errorf("expected 1 player, got %d", len(frame.Players))
	}
	if len(frame.Accessories) != 1 {
		t.Errorf("expected 1 accessory, got %d", len(frame.Accessories))
	}
	if len(frame.Actions) != 1 {
		t.Errorf("expected 1 action, got %d", len(frame.Actions))
	}
	if !almostEqual(frame.Players[0].Opacity, 1.0, 0.001) {
		t.Errorf("expected opacity 1.0, got %f", frame.Players[0].Opacity)
	}
	if !almostEqual(frame.Actions[0].Progress, 1.0, 0.001) {
		t.Errorf("expected progress 1.0, got %f", frame.Actions[0].Progress)
	}
}
