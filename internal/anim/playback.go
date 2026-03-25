package anim

import (
	"math"
	"time"

	"github.com/darkweaver87/courtdraw/internal/court"
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
	maxStep := model.MaxStep(seq)

	// Compute cumulative player positions based on completed movement actions.
	positions := ComputeStepPositions(seq, maxStep, t)

	// Players at their computed positions.
	for i := range seq.Players {
		player := seq.Players[i]
		if pos, ok := positions[player.ID]; ok {
			player.Position = pos
		}
		frame.Players = append(frame.Players, AnimatedPlayer{
			Player:  player,
			Opacity: 1.0,
		})
	}

	// Accessories (static).
	for i := range seq.Accessories {
		frame.Accessories = append(frame.Accessories, AnimatedAccessory{
			Accessory: seq.Accessories[i],
			Opacity:   1.0,
		})
	}

	// Ball positions: follow carriers, interpolate during passes.
	frame.Balls = computeStepBalls(seq, maxStep, t, positions)

	// Actions: progressive drawing based on step.
	// Movement actions disappear after their step completes (player has moved).
	for i := range seq.Actions {
		progress := stepProgress(seq.Actions[i].EffectiveStep(), maxStep, t)
		if progress >= 1.0 && model.IsMovementAction(seq.Actions[i].Type) {
			continue // movement done — arrow no longer needed
		}
		frame.Actions = append(frame.Actions, AnimatedAction{
			Action:   seq.Actions[i],
			Progress: progress,
		})
	}
	return frame
}

// stepProgress returns the drawing progress (0–1) for an action at the given step.
func stepProgress(step, maxStep int, t float64) float64 {
	if maxStep <= 1 {
		return t
	}
	stepStart := float64(step-1) / float64(maxStep)
	stepEnd := float64(step) / float64(maxStep)
	if t <= stepStart {
		return 0
	}
	if t >= stepEnd {
		return 1
	}
	return (t - stepStart) / (stepEnd - stepStart)
}

// ComputeStepPositions computes player positions accounting for all movement actions up to time t.
func ComputeStepPositions(seq *model.Sequence, maxStep int, t float64) map[string]model.Position {
	positions := make(map[string]model.Position, len(seq.Players))
	for i := range seq.Players {
		positions[seq.Players[i].ID] = seq.Players[i].Position
	}

	for i := range seq.Actions {
		act := &seq.Actions[i]
		if !model.IsMovementAction(act.Type) || !act.From.IsPlayer {
			continue
		}
		pid := act.From.PlayerID
		progress := stepProgress(act.EffectiveStep(), maxStep, t)
		if progress <= 0 {
			continue
		}
		// Resolve destination position.
		dest := resolveActionDest(act, positions)
		// Avoid overlap with other players at the destination.
		if progress >= 1.0 {
			dest = avoidOverlapPositions(dest, pid, positions)
		}
		// Current position of this player (may have been moved by a prior step).
		from := positions[pid]
		positions[pid] = InterpolatePosition(from, dest, progress)
	}
	return positions
}

// BallState represents a ball's carrier and position after all steps.
type BallState struct {
	CarrierID string         // empty if shot (ball is at ShotPos)
	ShotPos   model.Position // non-zero if ball was shot
	IsShot    bool
}

// ComputeFinalBallState returns ball states after all steps are completed.
func ComputeFinalBallState(seq *model.Sequence) []BallState {
	states := make([]BallState, len(seq.BallCarrier))
	for i, id := range seq.BallCarrier {
		states[i] = BallState{CarrierID: id}
	}
	for i := range seq.Actions {
		act := &seq.Actions[i]
		if act.Type == model.ActionPass && act.From.IsPlayer && act.To.IsPlayer {
			for j := range states {
				if states[j].CarrierID == act.From.PlayerID && !states[j].IsShot {
					states[j].CarrierID = act.To.PlayerID
					break
				}
			}
		} else if model.IsShot(act.Type) && act.From.IsPlayer {
			for j := range states {
				if states[j].CarrierID == act.From.PlayerID && !states[j].IsShot {
					states[j].IsShot = true
					states[j].ShotPos = act.To.Position
					states[j].CarrierID = ""
					break
				}
			}
		}
	}
	return states
}

// ComputeFinalBallCarriers returns ball carrier IDs after all steps are completed.
func ComputeFinalBallCarriers(seq *model.Sequence) []string {
	states := ComputeFinalBallState(seq)
	var carriers []string
	for _, s := range states {
		if !s.IsShot && s.CarrierID != "" {
			carriers = append(carriers, s.CarrierID)
		}
	}
	return carriers
}

// computeStepBalls computes ball positions during step animation, interpolating during passes.
func computeStepBalls(seq *model.Sequence, maxStep int, t float64, positions map[string]model.Position) []AnimatedBall {
	// Start with initial carriers.
	balls := make([]ballState, 0, len(seq.BallCarrier))
	for _, id := range seq.BallCarrier {
		if pos, ok := positions[id]; ok {
			balls = append(balls, ballState{carrierID: id, pos: pos})
		}
	}

	// Apply passes and shots step by step.
	for i := range seq.Actions {
		act := &seq.Actions[i]
		progress := stepProgress(act.EffectiveStep(), maxStep, t)
		if progress <= 0 {
			continue
		}

		switch {
		case act.Type == model.ActionPass && act.From.IsPlayer && act.To.IsPlayer:
			applyPassToBalls(balls, act, progress, positions)
		case model.IsShot(act.Type) && act.From.IsPlayer:
			applyShotToBalls(balls, act, progress, positions)
		}
	}

	result := make([]AnimatedBall, len(balls))
	for i, b := range balls {
		result[i] = AnimatedBall{CarrierID: b.carrierID, Pos: b.pos, Opacity: 1.0}
	}
	return result
}

type ballState struct {
	carrierID string
	pos       model.Position
}

func applyPassToBalls(balls []ballState, act *model.Action, progress float64, positions map[string]model.Position) {
	fromPos := positions[act.From.PlayerID]
	toPos := positions[act.To.PlayerID]
	for j := range balls {
		if balls[j].carrierID == act.From.PlayerID {
			if progress >= 1.0 {
				balls[j].carrierID = act.To.PlayerID
				balls[j].pos = toPos
			} else {
				balls[j].pos = InterpolatePosition(fromPos, toPos, progress)
			}
			break
		}
	}
}

func applyShotToBalls(balls []ballState, act *model.Action, progress float64, positions map[string]model.Position) {
	fromPos := positions[act.From.PlayerID]
	for j := range balls {
		if balls[j].carrierID == act.From.PlayerID {
			balls[j].pos = InterpolatePosition(fromPos, act.To.Position, progress)
			if progress >= 1.0 {
				balls[j].carrierID = ""
			}
			break
		}
	}
}

// avoidOverlapPositions adjusts a position so it doesn't overlap with other players.
func avoidOverlapPositions(pos model.Position, selfID string, positions map[string]model.Position) model.Position {
	for id, p := range positions {
		if id == selfID {
			continue
		}
		dx := pos[0] - p[0]
		dy := pos[1] - p[1]
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < court.MinPlayerSpacing && dist > 0.001 {
			scale := court.MinPlayerSpacing / dist
			pos[0] = p[0] + dx*scale
			pos[1] = p[1] + dy*scale
		} else if dist <= 0.001 {
			pos[0] += court.MinPlayerSpacing
		}
	}
	return pos
}

// resolveActionDest returns the destination position for an action,
// using the player position map for player-targeted actions.
func resolveActionDest(act *model.Action, positions map[string]model.Position) model.Position {
	if act.To.IsPlayer {
		if pos, ok := positions[act.To.PlayerID]; ok {
			return pos
		}
	}
	return act.To.Position
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
