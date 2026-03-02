package anim

import (
	"time"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// Speed represents an animation speed multiplier.
type Speed float64

const (
	SpeedHalf   Speed = 0.5
	SpeedNormal Speed = 1.0
	SpeedDouble Speed = 2.0
)

// TransitionDuration is the base duration for animating between two sequences.
const TransitionDuration = 2 * time.Second

// PauseDuration is the pause at each keyframe before transitioning.
const PauseDuration = 1 * time.Second

// PlaybackState indicates whether the animation is playing.
type PlaybackState int

const (
	StateStopped PlaybackState = iota
	StatePlaying
	StatePaused
)

// Playback controls animation playback through exercise sequences.
type Playback struct {
	exercise *model.Exercise
	state    PlaybackState
	speed    Speed

	// Current position in the animation.
	// seqIndex is the "from" sequence index.
	seqIndex int

	// phase: 0 = pausing at keyframe, 1 = transitioning to next.
	phase int

	// elapsed time within the current phase.
	elapsed time.Duration

	// lastTick is the time of the last Update call.
	lastTick time.Time
}

// NewPlayback creates a Playback for the given exercise.
func NewPlayback(exercise *model.Exercise) *Playback {
	return &Playback{
		exercise: exercise,
		state:    StateStopped,
		speed:    SpeedNormal,
		seqIndex: 0,
	}
}

// SetExercise changes the exercise and resets playback.
func (p *Playback) SetExercise(ex *model.Exercise) {
	p.exercise = ex
	p.Stop()
}

// Play starts or resumes animation.
func (p *Playback) Play() {
	if p.exercise == nil || len(p.exercise.Sequences) == 0 {
		return
	}
	if p.state == StatePaused {
		p.state = StatePlaying
		p.lastTick = time.Now()
		return
	}
	p.state = StatePlaying
	p.seqIndex = 0
	p.phase = 0
	p.elapsed = 0
	p.lastTick = time.Now()
}

// Pause pauses animation.
func (p *Playback) Pause() {
	if p.state == StatePlaying {
		p.state = StatePaused
	}
}

// Stop stops animation and resets to the first sequence.
func (p *Playback) Stop() {
	p.state = StateStopped
	p.seqIndex = 0
	p.phase = 0
	p.elapsed = 0
}

// State returns the current playback state.
func (p *Playback) State() PlaybackState {
	return p.state
}

// Speed returns the current speed.
func (p *Playback) Speed() Speed {
	return p.speed
}

// SetSpeed changes the playback speed.
func (p *Playback) SetSpeed(s Speed) {
	p.speed = s
}

// CycleSpeed cycles through available speeds.
func (p *Playback) CycleSpeed() {
	switch p.speed {
	case SpeedHalf:
		p.speed = SpeedNormal
	case SpeedNormal:
		p.speed = SpeedDouble
	case SpeedDouble:
		p.speed = SpeedHalf
	default:
		p.speed = SpeedNormal
	}
}

// SeqIndex returns the primary sequence index (the "from" sequence).
func (p *Playback) SeqIndex() int {
	return p.seqIndex
}

// SetSeqIndex jumps to a specific sequence and stops playback.
func (p *Playback) SetSeqIndex(idx int) {
	if p.exercise == nil {
		return
	}
	if idx >= 0 && idx < len(p.exercise.Sequences) {
		p.state = StateStopped
		p.seqIndex = idx
		p.phase = 0
		p.elapsed = 0
	}
}

// NextSeq jumps to the next sequence (step mode).
func (p *Playback) NextSeq() {
	if p.exercise == nil {
		return
	}
	p.state = StateStopped
	max := len(p.exercise.Sequences) - 1
	if p.seqIndex < max {
		p.seqIndex++
	}
	p.phase = 0
	p.elapsed = 0
}

// PrevSeq jumps to the previous sequence (step mode).
func (p *Playback) PrevSeq() {
	if p.exercise == nil {
		return
	}
	p.state = StateStopped
	if p.seqIndex > 0 {
		p.seqIndex--
	}
	p.phase = 0
	p.elapsed = 0
}

// IsAnimating returns true if the playback is currently in a transition
// (not paused at a keyframe).
func (p *Playback) IsAnimating() bool {
	return p.state == StatePlaying
}

// Update advances the animation by the time since the last call.
// Returns the current AnimatedFrame and whether a redraw is needed.
func (p *Playback) Update() (AnimatedFrame, bool) {
	if p.exercise == nil || len(p.exercise.Sequences) == 0 {
		return AnimatedFrame{}, false
	}

	seq := p.exercise.Sequences
	lastIdx := len(seq) - 1

	if p.state != StatePlaying {
		// Return static frame of the current sequence.
		return snapshotFrame(&seq[p.seqIndex]), false
	}

	now := time.Now()
	dt := now.Sub(p.lastTick)
	p.lastTick = now
	p.elapsed += time.Duration(float64(dt) * float64(p.speed))

	switch p.phase {
	case 0: // pausing at keyframe
		if p.elapsed >= PauseDuration {
			if p.seqIndex >= lastIdx {
				// Reached the end: stop.
				p.state = StateStopped
				return snapshotFrame(&seq[p.seqIndex]), false
			}
			p.phase = 1
			p.elapsed -= PauseDuration
		}
		return snapshotFrame(&seq[p.seqIndex]), true

	case 1: // transitioning
		t := float64(p.elapsed) / float64(TransitionDuration)
		if t >= 1.0 {
			// Transition complete: advance to next sequence.
			p.seqIndex++
			p.phase = 0
			p.elapsed -= TransitionDuration
			if p.seqIndex >= lastIdx {
				p.state = StateStopped
				return snapshotFrame(&seq[p.seqIndex]), false
			}
			return snapshotFrame(&seq[p.seqIndex]), true
		}
		return InterpolateFrame(&seq[p.seqIndex], &seq[p.seqIndex+1], t), true
	}

	return snapshotFrame(&seq[p.seqIndex]), false
}
