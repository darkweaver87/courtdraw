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

// StepDuration is the base duration per action step within a sequence.
const StepDuration = 1 * time.Second

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
	loop     bool // if true, restart from seq 0 after the last sequence

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

// SetLoop enables or disables looping.
func (p *Playback) SetLoop(on bool) {
	p.loop = on
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

// stepFrame creates a frame for intra-sequence step animation.
// t is 0.0–1.0 over the total step animation duration.
func (p *Playback) stepFrame(seq *model.Sequence, t float64) AnimatedFrame {
	frame := AnimatedFrame{}
	// Players/accessories are static — just snapshot them.
	for i := range seq.Players {
		frame.Players = append(frame.Players, AnimatedPlayer{
			Player:  seq.Players[i],
			Opacity: 1.0,
		})
	}
	for i := range seq.Accessories {
		frame.Accessories = append(frame.Accessories, AnimatedAccessory{
			Accessory: seq.Accessories[i],
			Opacity:   1.0,
		})
	}
	// Balls.
	for _, id := range seq.BallCarrier {
		for i := range seq.Players {
			if seq.Players[i].ID == id {
				frame.Balls = append(frame.Balls, AnimatedBall{
					CarrierID: id,
					Pos:       seq.Players[i].Position,
					Opacity:   1.0,
				})
				break
			}
		}
	}
	// Actions: progressive drawing based on step.
	maxStep := model.MaxStep(seq)
	for i := range seq.Actions {
		step := seq.Actions[i].EffectiveStep()
		var progress float64
		if maxStep <= 1 {
			progress = t
		} else {
			stepStart := float64(step-1) / float64(maxStep)
			stepEnd := float64(step) / float64(maxStep)
			if t <= stepStart {
				progress = 0
			} else if t >= stepEnd {
				progress = 1
			} else {
				progress = (t - stepStart) / (stepEnd - stepStart)
			}
		}
		frame.Actions = append(frame.Actions, AnimatedAction{
			Action:   seq.Actions[i],
			Progress: progress,
		})
	}
	return frame
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
	// Cap dt to avoid large jumps when rendering is slow.
	const maxDt = time.Second / 30
	if dt > maxDt {
		dt = maxDt
	}
	p.elapsed += time.Duration(float64(dt) * float64(p.speed))

	switch p.phase {
	case 0: // pausing at keyframe
		if p.elapsed >= PauseDuration {
			p.elapsed -= PauseDuration
			// If this sequence has multiple steps, animate them first.
			maxStep := model.MaxStep(&seq[p.seqIndex])
			if maxStep > 1 {
				p.phase = 2 // intra-sequence step animation
				return p.stepFrame(&seq[p.seqIndex], 0), true
			}
			// No steps to animate — proceed to transition or stop.
			if p.seqIndex >= lastIdx {
				if p.loop {
					p.seqIndex = 0
					p.phase = 0
					return snapshotFrame(&seq[0]), true
				}
				p.state = StateStopped
				return snapshotFrame(&seq[p.seqIndex]), false
			}
			p.phase = 1
		}
		return snapshotFrame(&seq[p.seqIndex]), true

	case 2: // intra-sequence step animation
		maxStep := model.MaxStep(&seq[p.seqIndex])
		stepAnimDuration := time.Duration(maxStep) * StepDuration
		t := float64(p.elapsed) / float64(stepAnimDuration)
		if t >= 1.0 {
			p.elapsed -= stepAnimDuration
			// Steps done — transition to next sequence or stop.
			if p.seqIndex >= lastIdx {
				if p.loop {
					p.seqIndex = 0
					p.phase = 0
					return snapshotFrame(&seq[0]), true
				}
				p.state = StateStopped
				return snapshotFrame(&seq[p.seqIndex]), false
			}
			p.phase = 1
			return snapshotFrame(&seq[p.seqIndex]), true
		}
		return p.stepFrame(&seq[p.seqIndex], t), true

	case 1: // transitioning to next sequence
		t := float64(p.elapsed) / float64(TransitionDuration)
		if t >= 1.0 {
			// Transition complete: advance to next sequence.
			p.seqIndex++
			p.phase = 0
			p.elapsed -= TransitionDuration
			if p.seqIndex >= lastIdx && !p.loop {
				p.state = StateStopped
				return snapshotFrame(&seq[p.seqIndex]), false
			}
			return snapshotFrame(&seq[p.seqIndex]), true
		}
		return InterpolateFrame(&seq[p.seqIndex], &seq[p.seqIndex+1], t), true
	}

	return snapshotFrame(&seq[p.seqIndex]), false
}
