package anim

import (
	"math"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// AnimatedPlayer is a player with interpolated position and opacity.
type AnimatedPlayer struct {
	model.Player

	Opacity float64 // 0.0 = invisible, 1.0 = fully visible
}

// AnimatedAccessory is an accessory with an opacity for appear/disappear.
type AnimatedAccessory struct {
	model.Accessory

	Opacity float64
}

// AnimatedAction is an action with a progress value for progressive drawing.
type AnimatedAction struct {
	model.Action

	Progress float64 // 0.0 = not drawn, 1.0 = fully drawn
}

// AnimatedBall holds the interpolated state of a single ball.
type AnimatedBall struct {
	CarrierID string
	Pos       model.Position
	Opacity   float64
}

// AnimatedFrame holds the interpolated state between two sequences.
type AnimatedFrame struct {
	Players     []AnimatedPlayer
	Accessories []AnimatedAccessory
	Actions     []AnimatedAction
	Balls       []AnimatedBall // all balls in this frame
}

// InterpolatePosition linearly interpolates between two positions.
func InterpolatePosition(from, to model.Position, t float64) model.Position {
	return model.Position{
		from[0] + (to[0]-from[0])*t,
		from[1] + (to[1]-from[1])*t,
	}
}

// InterpolateRotation interpolates between two rotation angles (degrees) using
// the shortest path around 360°.
func InterpolateRotation(from, to, t float64) float64 {
	// Normalize both to [0, 360).
	from = math.Mod(from, 360)
	if from < 0 {
		from += 360
	}
	to = math.Mod(to, 360)
	if to < 0 {
		to += 360
	}
	diff := to - from
	// Shortest path: if diff > 180, go the other way.
	if diff > 180 {
		diff -= 360
	} else if diff < -180 {
		diff += 360
	}
	result := from + diff*t
	result = math.Mod(result, 360)
	if result < 0 {
		result += 360
	}
	return result
}

// InterpolateFrame computes an interpolated frame between two sequences.
// t is the interpolation factor: 0.0 = fromSeq, 1.0 = toSeq.
func InterpolateFrame(fromSeq, toSeq *model.Sequence, t float64) AnimatedFrame {
	if fromSeq == nil && toSeq == nil {
		return AnimatedFrame{}
	}
	if fromSeq == nil {
		return snapshotFrame(toSeq)
	}
	if toSeq == nil {
		return snapshotFrame(fromSeq)
	}
	if t <= 0 {
		return snapshotFrame(fromSeq)
	}
	if t >= 1 {
		return snapshotFrame(toSeq)
	}

	frame := AnimatedFrame{}

	// Compute final positions after all steps in fromSeq (for step-aware movement).
	finalPositions := ComputeStepPositions(fromSeq, model.MaxStep(fromSeq), 1.0)

	// Build lookup maps by player ID, using final step positions.
	fromMap := make(map[string]*model.Player, len(fromSeq.Players))
	for i := range fromSeq.Players {
		p := fromSeq.Players[i]
		if pos, ok := finalPositions[p.ID]; ok {
			p.Position = pos
		}
		pCopy := p
		fromMap[pCopy.ID] = &pCopy
	}
	toMap := make(map[string]*model.Player, len(toSeq.Players))
	for i := range toSeq.Players {
		toMap[toSeq.Players[i].ID] = &toSeq.Players[i]
	}

	// Players present in both: interpolate position.
	seen := make(map[string]bool)
	for i := range fromSeq.Players {
		fp := &fromSeq.Players[i]
		if tp, ok := toMap[fp.ID]; ok {
			// Present in both: lerp position and rotation.
			ap := AnimatedPlayer{
				Player: model.Player{
					ID:       fp.ID,
					Label:    tp.Label,
					Role:     tp.Role,
					Position: InterpolatePosition(fp.Position, tp.Position, t),
					Rotation: InterpolateRotation(fp.Rotation, tp.Rotation, t),
					Callout:  tp.Callout,
					Type:     fp.Type,
					Count:    fp.Count,
				},
				Opacity: 1.0,
			}
			frame.Players = append(frame.Players, ap)
		} else {
			// Fade out: present in from, absent in to.
			ap := AnimatedPlayer{
				Player:  *fp,
				Opacity: 1.0 - t,
			}
			frame.Players = append(frame.Players, ap)
		}
		seen[fp.ID] = true
	}
	// Fade in: present in to, absent in from.
	for i := range toSeq.Players {
		tp := &toSeq.Players[i]
		if !seen[tp.ID] {
			ap := AnimatedPlayer{
				Player:  *tp,
				Opacity: t,
			}
			frame.Players = append(frame.Players, ap)
		}
	}

	// Accessories: static, appear/disappear based on presence.
	fromAccMap := make(map[string]*model.Accessory, len(fromSeq.Accessories))
	for i := range fromSeq.Accessories {
		fromAccMap[fromSeq.Accessories[i].ID] = &fromSeq.Accessories[i]
	}
	toAccMap := make(map[string]*model.Accessory, len(toSeq.Accessories))
	for i := range toSeq.Accessories {
		toAccMap[toSeq.Accessories[i].ID] = &toSeq.Accessories[i]
	}

	accSeen := make(map[string]bool)
	for i := range fromSeq.Accessories {
		fa := &fromSeq.Accessories[i]
		if ta, ok := toAccMap[fa.ID]; ok {
			// Present in both: interpolate position and rotation.
			interpAcc := model.Accessory{
				Type:     ta.Type,
				ID:       fa.ID,
				Position: InterpolatePosition(fa.Position, ta.Position, t),
				Rotation: InterpolateRotation(fa.Rotation, ta.Rotation, t),
			}
			frame.Accessories = append(frame.Accessories, AnimatedAccessory{Accessory: interpAcc, Opacity: 1.0})
		} else {
			frame.Accessories = append(frame.Accessories, AnimatedAccessory{Accessory: *fa, Opacity: 1.0 - t})
		}
		accSeen[fa.ID] = true
	}
	for i := range toSeq.Accessories {
		ta := &toSeq.Accessories[i]
		if !accSeen[ta.ID] {
			frame.Accessories = append(frame.Accessories, AnimatedAccessory{Accessory: *ta, Opacity: t})
		}
	}

	// Actions: progressive drawing grouped by step.
	maxStep := model.MaxStep(toSeq)
	for i := range toSeq.Actions {
		step := toSeq.Actions[i].EffectiveStep()
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
			Action:   toSeq.Actions[i],
			Progress: progress,
		})
	}

	// Ball interpolation for multiple carriers.
	frame.Balls = interpolateBalls(fromSeq, toSeq, fromMap, toMap, t)

	return frame
}

// findShotTarget checks if the given player has a shot action in the sequence
// and returns the shot's target position. Shot actions are: ShotLayup, ShotPushup, ShotJump.
func findShotTarget(seq *model.Sequence, playerID string) (model.Position, bool) {
	if playerID == "" {
		return model.Position{}, false
	}
	for i := range seq.Actions {
		a := &seq.Actions[i]
		switch a.Type {
		case model.ActionShotLayup, model.ActionShotPushup, model.ActionShotJump:
			if a.From.IsPlayer && a.From.PlayerID == playerID {
				if a.To.IsPlayer {
					// Target is a player — find their position.
					for j := range seq.Players {
						if seq.Players[j].ID == a.To.PlayerID {
							return seq.Players[j].Position, true
						}
					}
				}
				return a.To.Position, true
			}
		}
	}
	return model.Position{}, false
}

// interpolateBalls computes animated balls between two sequences.
func interpolateBalls(fromSeq, toSeq *model.Sequence, fromMap, toMap map[string]*model.Player, t float64) []AnimatedBall {
	balls := make([]AnimatedBall, 0, len(fromSeq.BallCarrier)+len(toSeq.BallCarrier))
	handled := make(map[string]bool) // to-carriers already accounted for

	playerPos := func(m map[string]*model.Player, id string) model.Position {
		if p, ok := m[id]; ok {
			return p.Position
		}
		return model.Position{}
	}

	for _, id := range fromSeq.BallCarrier {
		// Check for shot or pass action first — even if the player is still
		// a carrier in the next sequence (supports ball exchanges where both
		// players keep a ball but swap them via simultaneous passes).
		if target, hasShot := findShotTarget(toSeq, id); hasShot {
			balls = append(balls, AnimatedBall{
				CarrierID: id,
				Pos:       InterpolatePosition(playerPos(fromMap, id), target, t),
				Opacity:   1.0,
			})
			continue
		}
		if targetID, hasPass := findPassTargetPlayer(toSeq, id); hasPass {
			toPos := playerPos(toMap, targetID)
			balls = append(balls, AnimatedBall{
				CarrierID: id,
				Pos:       InterpolatePosition(playerPos(fromMap, id), toPos, t),
				Opacity:   1.0,
			})
			handled[targetID] = true
			continue
		}
		if toSeq.BallCarrier.HasBall(id) {
			// No action and still a carrier — ball stays with the player.
			balls = append(balls, AnimatedBall{
				CarrierID: id,
				Pos:       InterpolatePosition(playerPos(fromMap, id), playerPos(toMap, id), t),
				Opacity:   1.0,
			})
			handled[id] = true
			continue
		}
		// No action — fade out.
		balls = append(balls, AnimatedBall{
			CarrierID: id,
			Pos:       playerPos(fromMap, id),
			Opacity:   1.0 - t,
		})
	}

	// New carriers that fade in.
	for _, id := range toSeq.BallCarrier {
		if handled[id] || fromSeq.BallCarrier.HasBall(id) {
			continue
		}
		balls = append(balls, AnimatedBall{
			CarrierID: id,
			Pos:       playerPos(toMap, id),
			Opacity:   t,
		})
	}

	return balls
}

// findPassTargetPlayer checks if the given player has a pass action in the
// sequence and returns the target player ID.
func findPassTargetPlayer(seq *model.Sequence, playerID string) (string, bool) {
	for i := range seq.Actions {
		a := &seq.Actions[i]
		if a.Type == model.ActionPass && a.From.IsPlayer && a.From.PlayerID == playerID && a.To.IsPlayer {
			return a.To.PlayerID, true
		}
	}
	return "", false
}

// snapshotFrame creates a fully-visible frame from a single sequence.
func snapshotFrame(seq *model.Sequence) AnimatedFrame {
	frame := AnimatedFrame{}
	// Create one ball per carrier.
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
	for i := range seq.Actions {
		frame.Actions = append(frame.Actions, AnimatedAction{
			Action:   seq.Actions[i],
			Progress: 1.0,
		})
	}
	return frame
}
