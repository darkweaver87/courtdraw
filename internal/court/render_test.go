package court_test

import (
	"testing"

	"github.com/darkweaver87/courtdraw/internal/court"
	"github.com/darkweaver87/courtdraw/internal/model"
)

// stubStepPlayers is a simple step-players function that returns base positions.
func stubStepPlayers(seq *model.Sequence, _, _ int) []model.Player {
	players := make([]model.Player, len(seq.Players))
	copy(players, seq.Players)
	return players
}

// stubFinalBallState returns ball states based on the sequence's BallCarrier.
func stubFinalBallState(seq *model.Sequence) []court.BallState {
	states := make([]court.BallState, len(seq.BallCarrier))
	for i, id := range seq.BallCarrier {
		states[i] = court.BallState{CarrierID: id}
	}
	return states
}

func TestRenderSequence_NilExercise(t *testing.T) {
	img := court.RenderSequence(nil, 0, 400, 600, stubStepPlayers, stubFinalBallState)
	if img == nil {
		t.Fatal("expected non-nil image for nil exercise")
	}
	b := img.Bounds()
	if b.Dx() != 400 || b.Dy() != 600 {
		t.Errorf("expected 400x600, got %dx%d", b.Dx(), b.Dy())
	}
}

func TestRenderSequence_EmptyExercise(t *testing.T) {
	ex := &model.Exercise{
		Name:      "empty",
		CourtType: model.HalfCourt,
	}
	img := court.RenderSequence(ex, 0, 300, 500, stubStepPlayers, stubFinalBallState)
	if img == nil {
		t.Fatal("expected non-nil image for empty exercise")
	}
	b := img.Bounds()
	if b.Dx() != 300 || b.Dy() != 500 {
		t.Errorf("expected 300x500, got %dx%d", b.Dx(), b.Dy())
	}
}

func TestRenderSequence_OutOfBoundsSeqIndex(t *testing.T) {
	ex := &model.Exercise{
		Name:      "test",
		CourtType: model.HalfCourt,
		Sequences: []model.Sequence{
			{Label: "S1"},
		},
	}
	// Negative index.
	img := court.RenderSequence(ex, -1, 200, 300, stubStepPlayers, stubFinalBallState)
	if img == nil {
		t.Fatal("expected non-nil image for negative seqIndex")
	}
	// Out of range.
	img = court.RenderSequence(ex, 5, 200, 300, stubStepPlayers, stubFinalBallState)
	if img == nil {
		t.Fatal("expected non-nil image for out-of-range seqIndex")
	}
}

func TestRenderSequence_ZeroDimensions(t *testing.T) {
	ex := &model.Exercise{
		Name:      "test",
		CourtType: model.HalfCourt,
	}
	img := court.RenderSequence(ex, 0, 0, 0, stubStepPlayers, stubFinalBallState)
	if img == nil {
		t.Fatal("expected non-nil image for zero dimensions")
	}
}

func TestRenderSequence_WithPlayers(t *testing.T) {
	ex := &model.Exercise{
		Name:          "test",
		CourtType:     model.HalfCourt,
		CourtStandard: model.FIBA,
		Sequences: []model.Sequence{
			{
				Label: "S1",
				Players: []model.Player{
					{ID: "p1", Role: model.RoleAttacker, Position: model.Position{0.5, 0.5}},
					{ID: "p2", Role: model.RoleDefender, Position: model.Position{0.3, 0.7}},
				},
				BallCarrier: model.BallCarriers{"p1"},
			},
		},
	}
	img := court.RenderSequence(ex, 0, 800, 1200, stubStepPlayers, stubFinalBallState)
	if img == nil {
		t.Fatal("expected non-nil image")
	}
	b := img.Bounds()
	if b.Dx() != 800 || b.Dy() != 1200 {
		t.Errorf("expected 800x1200, got %dx%d", b.Dx(), b.Dy())
	}
}

func TestRenderSequence_WithActionsAndAccessories(t *testing.T) {
	ex := &model.Exercise{
		Name:          "full",
		CourtType:     model.FullCourt,
		CourtStandard: model.NBA,
		Sequences: []model.Sequence{
			{
				Label: "S1",
				Players: []model.Player{
					{ID: "p1", Role: model.RolePointGuard, Position: model.Position{0.5, 0.3}},
					{ID: "p2", Role: model.RoleCenter, Position: model.Position{0.5, 0.7}},
				},
				Actions: []model.Action{
					{
						Type: model.ActionPass,
						From: model.ActionRef{IsPlayer: true, PlayerID: "p1"},
						To:   model.ActionRef{IsPlayer: true, PlayerID: "p2"},
					},
				},
				Accessories: []model.Accessory{
					{Type: model.AccessoryCone, Position: model.Position{0.4, 0.5}},
					{Type: model.AccessoryChair, Position: model.Position{0.6, 0.5}},
				},
				BallCarrier: model.BallCarriers{"p1"},
			},
		},
	}
	img := court.RenderSequence(ex, 0, 600, 1000, stubStepPlayers, stubFinalBallState)
	if img == nil {
		t.Fatal("expected non-nil image")
	}
	b := img.Bounds()
	if b.Dx() != 600 || b.Dy() != 1000 {
		t.Errorf("expected 600x1000, got %dx%d", b.Dx(), b.Dy())
	}
}

func TestScaledFace(t *testing.T) {
	face := court.ScaledFace(1.0)
	if face == nil {
		t.Fatal("expected non-nil face at zoom 1.0")
	}
	// Same zoom should return cached face.
	face2 := court.ScaledFace(1.0)
	if face2 != face {
		t.Error("expected cached face to be returned")
	}
}
