package anim

import (
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

// AnimatedFrame holds the interpolated state between two sequences.
type AnimatedFrame struct {
	Players     []AnimatedPlayer
	Accessories []AnimatedAccessory
	Actions     []AnimatedAction
	BallCarrier string         // player ID who has the ball in this frame
	BallPos     model.Position // interpolated ball position (relative coords)
	BallOpacity float64        // 0.0 = invisible, 1.0 = fully visible
}

// InterpolatePosition linearly interpolates between two positions.
func InterpolatePosition(from, to model.Position, t float64) model.Position {
	return model.Position{
		from[0] + (to[0]-from[0])*t,
		from[1] + (to[1]-from[1])*t,
	}
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

	// Build lookup maps by player ID.
	fromMap := make(map[string]*model.Player, len(fromSeq.Players))
	for i := range fromSeq.Players {
		fromMap[fromSeq.Players[i].ID] = &fromSeq.Players[i]
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
			// Present in both: lerp.
			ap := AnimatedPlayer{
				Player: model.Player{
					ID:       fp.ID,
					Label:    tp.Label,
					Role:     tp.Role,
					Position: InterpolatePosition(fp.Position, tp.Position, t),
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
		if _, ok := toAccMap[fa.ID]; ok {
			frame.Accessories = append(frame.Accessories, AnimatedAccessory{Accessory: *fa, Opacity: 1.0})
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

	// Actions: progressive drawing from the destination sequence.
	for i := range toSeq.Actions {
		frame.Actions = append(frame.Actions, AnimatedAction{
			Action:   toSeq.Actions[i],
			Progress: t,
		})
	}

	// Ball: interpolate position between old carrier and new carrier.
	fromCarrier := fromSeq.BallCarrier
	toCarrier := toSeq.BallCarrier

	switch {
	case fromCarrier != "" && toCarrier != "":
		// Ball moves from one player to another (or stays on same).
		frame.BallCarrier = toCarrier
		frame.BallOpacity = 1.0
		var fromPos, toPos model.Position
		if fp, ok := fromMap[fromCarrier]; ok {
			fromPos = fp.Position
		}
		if tp, ok := toMap[toCarrier]; ok {
			toPos = tp.Position
		}
		frame.BallPos = InterpolatePosition(fromPos, toPos, t)
	case fromCarrier == "" && toCarrier != "":
		// Ball appears: fade in at new carrier position.
		frame.BallCarrier = toCarrier
		frame.BallOpacity = t
		if tp, ok := toMap[toCarrier]; ok {
			frame.BallPos = tp.Position
		}
	case fromCarrier != "" && toCarrier == "":
		// Ball disappears: fade out at old carrier position.
		frame.BallCarrier = fromCarrier
		frame.BallOpacity = 1.0 - t
		if fp, ok := fromMap[fromCarrier]; ok {
			frame.BallPos = fp.Position
		}
	}

	return frame
}

// snapshotFrame creates a fully-visible frame from a single sequence.
func snapshotFrame(seq *model.Sequence) AnimatedFrame {
	frame := AnimatedFrame{
		BallCarrier: seq.BallCarrier,
		BallOpacity: 1.0,
	}
	// Set ball position from carrier.
	if seq.BallCarrier != "" {
		for i := range seq.Players {
			if seq.Players[i].ID == seq.BallCarrier {
				frame.BallPos = seq.Players[i].Position
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
