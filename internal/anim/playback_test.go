package anim

import (
	"testing"

	"github.com/darkweaver87/courtdraw/internal/model"
)

func makeExercise() *model.Exercise {
	return &model.Exercise{
		Sequences: []model.Sequence{
			{
				Label: "Setup",
				Players: []model.Player{
					{ID: "p1", Position: model.Position{0.0, 0.0}},
				},
			},
			{
				Label: "Move",
				Players: []model.Player{
					{ID: "p1", Position: model.Position{1.0, 1.0}},
				},
			},
			{
				Label: "End",
				Players: []model.Player{
					{ID: "p1", Position: model.Position{0.5, 0.5}},
				},
			},
		},
	}
}

func TestPlayback_InitialState(t *testing.T) {
	p := NewPlayback(makeExercise())
	if p.State() != StateStopped {
		t.Errorf("expected StateStopped, got %d", p.State())
	}
	if p.SeqIndex() != 0 {
		t.Errorf("expected seqIndex 0, got %d", p.SeqIndex())
	}
	if p.Speed() != SpeedNormal {
		t.Errorf("expected SpeedNormal, got %f", p.Speed())
	}
}

func TestPlayback_PlayPauseStop(t *testing.T) {
	p := NewPlayback(makeExercise())

	p.Play()
	if p.State() != StatePlaying {
		t.Errorf("expected StatePlaying, got %d", p.State())
	}

	p.Pause()
	if p.State() != StatePaused {
		t.Errorf("expected StatePaused, got %d", p.State())
	}

	p.Play() // resume
	if p.State() != StatePlaying {
		t.Errorf("expected StatePlaying after resume, got %d", p.State())
	}

	p.Stop()
	if p.State() != StateStopped {
		t.Errorf("expected StateStopped, got %d", p.State())
	}
	if p.SeqIndex() != 0 {
		t.Errorf("expected seqIndex 0 after stop, got %d", p.SeqIndex())
	}
}

func TestPlayback_NextPrev(t *testing.T) {
	p := NewPlayback(makeExercise())

	p.NextSeq()
	if p.SeqIndex() != 1 {
		t.Errorf("expected seqIndex 1, got %d", p.SeqIndex())
	}

	p.NextSeq()
	if p.SeqIndex() != 2 {
		t.Errorf("expected seqIndex 2, got %d", p.SeqIndex())
	}

	// Should clamp at max.
	p.NextSeq()
	if p.SeqIndex() != 2 {
		t.Errorf("expected seqIndex 2 (clamped), got %d", p.SeqIndex())
	}

	p.PrevSeq()
	if p.SeqIndex() != 1 {
		t.Errorf("expected seqIndex 1, got %d", p.SeqIndex())
	}

	p.PrevSeq()
	p.PrevSeq()
	if p.SeqIndex() != 0 {
		t.Errorf("expected seqIndex 0 (clamped), got %d", p.SeqIndex())
	}
}

func TestPlayback_CycleSpeed(t *testing.T) {
	p := NewPlayback(makeExercise())

	if p.Speed() != SpeedNormal {
		t.Fatalf("expected SpeedNormal")
	}

	p.CycleSpeed()
	if p.Speed() != SpeedDouble {
		t.Errorf("expected SpeedDouble, got %f", p.Speed())
	}

	p.CycleSpeed()
	if p.Speed() != SpeedHalf {
		t.Errorf("expected SpeedHalf, got %f", p.Speed())
	}

	p.CycleSpeed()
	if p.Speed() != SpeedNormal {
		t.Errorf("expected SpeedNormal, got %f", p.Speed())
	}
}

func TestPlayback_UpdateStopped(t *testing.T) {
	p := NewPlayback(makeExercise())

	frame, needRedraw := p.Update()
	if needRedraw {
		t.Error("stopped playback should not need redraw")
	}
	if len(frame.Players) != 1 {
		t.Errorf("expected 1 player in frame, got %d", len(frame.Players))
	}
}

func TestPlayback_NilExercise(t *testing.T) {
	p := NewPlayback(nil)
	p.Play() // should not panic
	frame, needRedraw := p.Update()
	if needRedraw {
		t.Error("nil exercise should not need redraw")
	}
	if len(frame.Players) != 0 {
		t.Errorf("expected 0 players for nil exercise, got %d", len(frame.Players))
	}
}

func TestPlayback_SetSeqIndex(t *testing.T) {
	p := NewPlayback(makeExercise())
	p.SetSeqIndex(2)
	if p.SeqIndex() != 2 {
		t.Errorf("expected seqIndex 2, got %d", p.SeqIndex())
	}

	// out of bounds: no change
	p.SetSeqIndex(10)
	if p.SeqIndex() != 2 {
		t.Errorf("expected seqIndex 2 (unchanged), got %d", p.SeqIndex())
	}

	p.SetSeqIndex(-1)
	if p.SeqIndex() != 2 {
		t.Errorf("expected seqIndex 2 (unchanged), got %d", p.SeqIndex())
	}
}
