package anim

import (
	"testing"
	"time"

	"github.com/darkweaver87/courtdraw/internal/model"
)

func TestPlayback_SetSeqIndex_StopsPlayback(t *testing.T) {
	p := NewPlayback(makeExercise())
	p.Play()
	if p.State() != StatePlaying {
		t.Fatal("expected StatePlaying")
	}

	p.SetSeqIndex(1)
	if p.State() != StateStopped {
		t.Errorf("SetSeqIndex should stop playback, got state %d", p.State())
	}
	if p.SeqIndex() != 1 {
		t.Errorf("expected seqIndex 1, got %d", p.SeqIndex())
	}
}

func TestPlayback_SingleSequenceExercise(t *testing.T) {
	ex := &model.Exercise{
		Sequences: []model.Sequence{
			{Label: "Only", Players: []model.Player{{ID: "p1", Position: model.Position{0.5, 0.5}}}},
		},
	}
	p := NewPlayback(ex)
	p.Play()

	// Simulate time passing beyond pause duration.
	p.lastTick = time.Now().Add(-2 * PauseDuration)
	frame, _ := p.Update()

	// With one sequence, playback should stop after the pause.
	if p.State() != StateStopped {
		t.Errorf("expected stopped after single seq, got %d", p.State())
	}
	if len(frame.Players) != 1 {
		t.Errorf("expected 1 player, got %d", len(frame.Players))
	}
}

func TestPlayback_NextSeq_SetsStateStopped(t *testing.T) {
	p := NewPlayback(makeExercise())
	p.Play()

	p.NextSeq()
	if p.State() != StateStopped {
		t.Errorf("NextSeq should set state to Stopped, got %d", p.State())
	}
	if p.SeqIndex() != 1 {
		t.Errorf("expected seqIndex 1, got %d", p.SeqIndex())
	}
}

func TestPlayback_PrevSeq_SetsStateStopped(t *testing.T) {
	p := NewPlayback(makeExercise())
	// Use NextSeq to reach seq 2 (NextSeq sets stopped state).
	p.NextSeq() // → 1
	p.NextSeq() // → 2

	// Now go back.
	p.PrevSeq()
	if p.State() != StateStopped {
		t.Errorf("PrevSeq should set state to Stopped, got %d", p.State())
	}
	if p.SeqIndex() != 1 {
		t.Errorf("expected seqIndex 1, got %d", p.SeqIndex())
	}
}

func TestPlayback_SetExercise_Resets(t *testing.T) {
	p := NewPlayback(makeExercise())
	p.Play()
	p.NextSeq()

	newEx := &model.Exercise{
		Sequences: []model.Sequence{
			{Label: "A"},
			{Label: "B"},
		},
	}
	p.SetExercise(newEx)
	if p.State() != StateStopped {
		t.Errorf("SetExercise should stop, got %d", p.State())
	}
	if p.SeqIndex() != 0 {
		t.Errorf("SetExercise should reset to 0, got %d", p.SeqIndex())
	}
}

func TestPlayback_EmptyExercise(t *testing.T) {
	ex := &model.Exercise{Sequences: []model.Sequence{}}
	p := NewPlayback(ex)

	p.Play()
	if p.State() != StateStopped {
		// Play should be a no-op for empty exercise.
		// (It never transitions to Playing since len(Sequences) == 0.)
		t.Logf("state after Play on empty exercise: %d", p.State())
	}

	frame, needRedraw := p.Update()
	if needRedraw {
		t.Error("empty exercise should not need redraw")
	}
	if len(frame.Players) != 0 {
		t.Errorf("expected 0 players, got %d", len(frame.Players))
	}
}

func TestPlayback_SpeedAffectsNothing_WhenStopped(t *testing.T) {
	p := NewPlayback(makeExercise())
	p.SetSpeed(SpeedDouble)

	frame, needRedraw := p.Update()
	if needRedraw {
		t.Error("stopped playback should not need redraw regardless of speed")
	}
	if len(frame.Players) != 1 {
		t.Errorf("expected 1 player, got %d", len(frame.Players))
	}
}

func TestPlayback_PauseWhileStopped_NoOp(t *testing.T) {
	p := NewPlayback(makeExercise())
	p.Pause() // Should be a no-op since not playing.
	if p.State() != StateStopped {
		t.Errorf("Pause while stopped should remain stopped, got %d", p.State())
	}
}

func TestPlayback_StopFromPaused(t *testing.T) {
	p := NewPlayback(makeExercise())
	p.Play()
	p.NextSeq()
	// Now at seq 1, stopped (NextSeq sets stopped).
	p.Play()
	p.Pause()
	if p.State() != StatePaused {
		t.Fatal("expected paused")
	}

	p.Stop()
	if p.State() != StateStopped {
		t.Errorf("expected stopped after Stop, got %d", p.State())
	}
	if p.SeqIndex() != 0 {
		t.Errorf("Stop should reset to 0, got %d", p.SeqIndex())
	}
}

func TestPlayback_CycleSpeed_FullLoop(t *testing.T) {
	p := NewPlayback(makeExercise())

	// Normal → Double → Half → Normal
	speeds := []Speed{SpeedNormal, SpeedDouble, SpeedHalf, SpeedNormal}
	for i, expected := range speeds {
		if p.Speed() != expected {
			t.Errorf("step %d: got %f, want %f", i, p.Speed(), expected)
		}
		if i < len(speeds)-1 {
			p.CycleSpeed()
		}
	}
}

func TestPlayback_SetSpeed(t *testing.T) {
	p := NewPlayback(makeExercise())

	p.SetSpeed(SpeedHalf)
	if p.Speed() != SpeedHalf {
		t.Errorf("got %f, want %f", p.Speed(), SpeedHalf)
	}

	p.SetSpeed(SpeedDouble)
	if p.Speed() != SpeedDouble {
		t.Errorf("got %f, want %f", p.Speed(), SpeedDouble)
	}
}

func TestPlayback_UpdatePaused_ReturnsStaticFrame(t *testing.T) {
	p := NewPlayback(makeExercise())
	p.Play()
	p.Pause()

	frame, needRedraw := p.Update()
	if needRedraw {
		t.Error("paused playback should not need redraw")
	}
	if len(frame.Players) != 1 {
		t.Errorf("expected 1 player, got %d", len(frame.Players))
	}
}

func TestPlayback_NilExercise_AllOps(t *testing.T) {
	p := NewPlayback(nil)

	// None of these should panic.
	p.Play()
	p.Pause()
	p.Stop()
	p.NextSeq()
	p.PrevSeq()
	p.SetSeqIndex(5)
	p.CycleSpeed()
	frame, _ := p.Update()

	if len(frame.Players) != 0 {
		t.Errorf("nil exercise should give empty frame")
	}
}
