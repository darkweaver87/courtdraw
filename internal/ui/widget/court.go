package widget

import (
	"image"
	"math"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/widget/material"

	"github.com/darkweaver87/courtdraw/internal/anim"
	"github.com/darkweaver87/courtdraw/internal/court"
	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/ui/editor"
)

// CourtWidget renders a basketball court with exercise elements.
type CourtWidget struct {
	exercise *model.Exercise
	seqIndex int
	geom     *court.CourtGeometry
	viewport court.Viewport

	// Pointer interaction — raw pointer events (no gesture.Drag).
	pressed       bool
	pressedPID    pointer.ID
	dragPlayerIdx int
	dragAccIdx    int
	dragActive    bool
}

// SetExercise sets the exercise to display.
func (cw *CourtWidget) SetExercise(ex *model.Exercise) {
	cw.exercise = ex
	cw.seqIndex = 0
	cw.updateGeometry()
}

// SetSequence changes which sequence is displayed.
func (cw *CourtWidget) SetSequence(index int) {
	if cw.exercise != nil && index >= 0 && index < len(cw.exercise.Sequences) {
		cw.seqIndex = index
	}
}

// SeqIndex returns the current sequence index.
func (cw *CourtWidget) SeqIndex() int {
	return cw.seqIndex
}

// Viewport returns a pointer to the current viewport (for external use).
func (cw *CourtWidget) Viewport() *court.Viewport {
	return &cw.viewport
}

func (cw *CourtWidget) updateGeometry() {
	if cw.exercise == nil {
		return
	}
	switch cw.exercise.CourtStandard {
	case model.NBA:
		cw.geom = court.NBAGeometry()
	default:
		cw.geom = court.FIBAGeometry()
	}
}

// currentSequence returns the current sequence or nil.
func (cw *CourtWidget) currentSequence() *model.Sequence {
	if cw.exercise == nil || cw.seqIndex >= len(cw.exercise.Sequences) {
		return nil
	}
	return &cw.exercise.Sequences[cw.seqIndex]
}

// Layout renders the court widget and handles pointer events.
func (cw *CourtWidget) Layout(gtx layout.Context, th *material.Theme, state *editor.EditorState) layout.Dimensions {
	if cw.exercise == nil {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}

	// Refresh geometry each frame in case court standard changed via properties panel.
	cw.updateGeometry()

	size := gtx.Constraints.Max

	// Compute viewport.
	cw.viewport = court.ComputeViewport(
		cw.exercise.CourtType,
		cw.geom,
		image.Pt(size.X, size.Y),
		10,
	)

	// Delete selected element if requested (e.g. from palette Delete button).
	if state.DeleteRequested {
		state.DeleteRequested = false
		cw.deleteSelected(gtx.Source, state)
	}

	// Handle pointer events.
	cw.handlePointer(gtx, state)

	// Handle keyboard events (Delete/Backspace to delete selected element).
	cw.handleKeyboard(gtx, state)

	// Draw court.
	switch cw.exercise.CourtStandard {
	case model.NBA:
		court.DrawNBACourt(gtx.Ops, cw.exercise.CourtType, &cw.viewport, cw.geom)
	default:
		court.DrawFIBACourt(gtx.Ops, cw.exercise.CourtType, &cw.viewport, cw.geom)
	}

	// Draw exercise elements for current sequence.
	seq := cw.currentSequence()
	if seq != nil {
		cw.drawSequence(gtx, th, seq, state)
	}

	// Register pointer and key input area over the court.
	area := clip.Rect{Max: size}.Push(gtx.Ops)
	event.Op(gtx.Ops, cw)
	area.Pop()

	return layout.Dimensions{Size: size}
}

// handlePointer processes pointer events on the court using raw pointer.Filter.
func (cw *CourtWidget) handlePointer(gtx layout.Context, state *editor.EditorState) {
	seq := cw.currentSequence()
	if seq == nil {
		return
	}

	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: cw,
			Kinds:  pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel,
		})
		if !ok {
			break
		}
		e, ok := ev.(pointer.Event)
		if !ok {
			continue
		}
		switch e.Kind {
		case pointer.Press:
			if !(e.Buttons == pointer.ButtonPrimary || e.Source == pointer.Touch) {
					continue
			}
			cw.pressed = true
			cw.pressedPID = e.PointerID
			cw.handlePress(gtx.Source, state, seq, e.Position)
			// Grab the pointer so we receive Drag and Release events.
			gtx.Execute(pointer.GrabCmd{Tag: cw, ID: e.PointerID})
		case pointer.Drag:
			if !cw.pressed || e.PointerID != cw.pressedPID {
				continue
			}
			cw.handleDrag(gtx.Source, state, seq, e.Position)
		case pointer.Release:
			if e.PointerID != cw.pressedPID {
				continue
			}
			cw.pressed = false
			cw.handleRelease(gtx.Source, state)
		case pointer.Cancel:
			cw.pressed = false
			cw.dragActive = false
			cw.dragPlayerIdx = -1
			cw.dragAccIdx = -1
		}
	}
}

// handleKeyboard processes keyboard events for element deletion.
func (cw *CourtWidget) handleKeyboard(gtx layout.Context, state *editor.EditorState) {
	for {
		ev, ok := gtx.Event(
			key.FocusFilter{Target: cw},
			key.Filter{Focus: cw, Name: key.NameDeleteForward},
			key.Filter{Focus: cw, Name: key.NameDeleteBackward},
		)
		if !ok {
			break
		}
		ke, isKey := ev.(key.Event)
		if !isKey || ke.State != key.Press {
			continue
		}
		if ke.Name == key.NameDeleteForward || ke.Name == key.NameDeleteBackward {
			cw.deleteSelected(gtx.Source, state)
		}
	}
}

// deleteSelected removes the currently selected element from the sequence.
func (cw *CourtWidget) deleteSelected(src input.Source, state *editor.EditorState) {
	sel := state.SelectedElement
	if sel == nil || sel.SeqIndex != cw.seqIndex {
		return
	}
	seq := cw.currentSequence()
	if seq == nil {
		return
	}
	switch sel.Kind {
	case editor.SelectPlayer:
		if sel.Index < len(seq.Players) {
			playerID := seq.Players[sel.Index].ID
			seq.Players = append(seq.Players[:sel.Index], seq.Players[sel.Index+1:]...)
			cw.removeActionsForPlayer(seq, playerID)
			if seq.BallCarrier == playerID {
				seq.BallCarrier = ""
			}
		}
	case editor.SelectAccessory:
		if sel.Index < len(seq.Accessories) {
			seq.Accessories = append(seq.Accessories[:sel.Index], seq.Accessories[sel.Index+1:]...)
		}
	case editor.SelectAction:
		if sel.Index < len(seq.Actions) {
			seq.Actions = append(seq.Actions[:sel.Index], seq.Actions[sel.Index+1:]...)
		}
	default:
		return
	}
	state.Deselect()
	state.MarkModified()
	src.Execute(op.InvalidateCmd{})
}

// handlePress handles a pointer press on the court canvas.
func (cw *CourtWidget) handlePress(src input.Source, state *editor.EditorState, seq *model.Sequence, pos f32.Point) {
	cw.dragActive = false
	cw.dragPlayerIdx = -1
	cw.dragAccIdx = -1

	// Request keyboard focus so Delete/Backspace work.
	src.Execute(key.FocusCmd{Tag: cw})

	switch state.ActiveTool {
	case editor.ToolNone, editor.ToolSelect:
		// Hit test: try to select an element.
		if pi := cw.hitTestPlayer(seq, pos); pi >= 0 {
			state.Select(editor.SelectPlayer, pi, cw.seqIndex)
			cw.dragPlayerIdx = pi
			cw.dragActive = true
			state.IsDragging = true
			src.Execute(op.InvalidateCmd{})
			return
		}
		if ai := cw.hitTestAccessory(seq, pos); ai >= 0 {
			state.Select(editor.SelectAccessory, ai, cw.seqIndex)
			cw.dragAccIdx = ai
			cw.dragActive = true
			state.IsDragging = true
			src.Execute(op.InvalidateCmd{})
			return
		}
		// Clicked empty area: deselect.
		state.Deselect()
		src.Execute(op.InvalidateCmd{})

	case editor.ToolPlayer:
		relPos := cw.viewport.PixelToRel(pos)
		relPos = clampPosition(relPos)
		p := model.Player{
			ID:       editor.NextPlayerID(seq),
			Label:    model.RoleLabel(state.ToolRole),
			Role:     state.ToolRole,
			Position: relPos,
		}
		if state.ToolQueue {
			p.Type = "queue"
			p.Count = 3
		}
		seq.Players = append(seq.Players, p)
		idx := len(seq.Players) - 1
		state.Select(editor.SelectPlayer, idx, cw.seqIndex)
		state.MarkModified()
		src.Execute(op.InvalidateCmd{})

	case editor.ToolAction:
		// Two-click flow: first click picks "from" player, second picks "to".
		if state.ActionFrom == nil {
			// First click: must be a player.
			if pi := cw.hitTestPlayer(seq, pos); pi >= 0 {
				id := seq.Players[pi].ID
				state.ActionFrom = &id
			}
		} else {
			// Second click: can be a player or a position.
			toRef := model.ActionRef{}
			if pi := cw.hitTestPlayer(seq, pos); pi >= 0 {
				toRef.IsPlayer = true
				toRef.PlayerID = seq.Players[pi].ID
			} else {
				toRef.Position = clampPosition(cw.viewport.PixelToRel(pos))
			}
			action := model.Action{
				Type: state.ToolActionType,
				From: model.ActionRef{IsPlayer: true, PlayerID: *state.ActionFrom},
				To:   toRef,
			}
			seq.Actions = append(seq.Actions, action)
			// Auto-transfer ball carrier on pass to a player.
			if state.ToolActionType == model.ActionPass && toRef.IsPlayer {
				seq.BallCarrier = toRef.PlayerID
			}
			state.ActionFrom = nil
			state.MarkModified()
			src.Execute(op.InvalidateCmd{})
		}

	case editor.ToolAccessory:
		relPos := cw.viewport.PixelToRel(pos)
		relPos = clampPosition(relPos)
		acc := model.Accessory{
			Type:     state.ToolAccessoryType,
			ID:       editor.NextAccessoryID(seq),
			Position: relPos,
		}
		seq.Accessories = append(seq.Accessories, acc)
		idx := len(seq.Accessories) - 1
		state.Select(editor.SelectAccessory, idx, cw.seqIndex)
		state.MarkModified()
		src.Execute(op.InvalidateCmd{})

	case editor.ToolDelete:
		if pi := cw.hitTestPlayer(seq, pos); pi >= 0 {
			playerID := seq.Players[pi].ID
			seq.Players = append(seq.Players[:pi], seq.Players[pi+1:]...)
			// Also remove actions referencing this player.
			cw.removeActionsForPlayer(seq, playerID)
			// Clear ball carrier if deleted player was the carrier.
			if seq.BallCarrier == playerID {
				seq.BallCarrier = ""
			}
			state.Deselect()
			state.MarkModified()
			src.Execute(op.InvalidateCmd{})
			return
		}
		if ai := cw.hitTestAccessory(seq, pos); ai >= 0 {
			seq.Accessories = append(seq.Accessories[:ai], seq.Accessories[ai+1:]...)
			state.Deselect()
			state.MarkModified()
			src.Execute(op.InvalidateCmd{})
			return
		}
		// Try to hit-test an action (by checking if click is near the midpoint).
		if actIdx := cw.hitTestAction(seq, pos); actIdx >= 0 {
			seq.Actions = append(seq.Actions[:actIdx], seq.Actions[actIdx+1:]...)
			state.Deselect()
			state.MarkModified()
			src.Execute(op.InvalidateCmd{})
			return
		}
		// Fallback: if nothing was hit but an element is selected, delete it.
		cw.deleteSelected(src, state)
	}
}

// handleDrag moves the currently dragged element.
func (cw *CourtWidget) handleDrag(src input.Source, state *editor.EditorState, seq *model.Sequence, pos f32.Point) {
	if !cw.dragActive {
		return
	}
	relPos := clampPosition(cw.viewport.PixelToRel(pos))

	if cw.dragPlayerIdx >= 0 && cw.dragPlayerIdx < len(seq.Players) {
		seq.Players[cw.dragPlayerIdx].Position = relPos
		state.MarkModified()
		src.Execute(op.InvalidateCmd{})
	}
	if cw.dragAccIdx >= 0 && cw.dragAccIdx < len(seq.Accessories) {
		seq.Accessories[cw.dragAccIdx].Position = relPos
		state.MarkModified()
		src.Execute(op.InvalidateCmd{})
	}
}

// handleRelease ends the current drag.
func (cw *CourtWidget) handleRelease(src input.Source, state *editor.EditorState) {
	cw.dragActive = false
	cw.dragPlayerIdx = -1
	cw.dragAccIdx = -1
	state.IsDragging = false
	src.Execute(op.InvalidateCmd{})
}

// hitTestPlayer returns the index of the player under pos, or -1.
func (cw *CourtWidget) hitTestPlayer(seq *model.Sequence, pos f32.Point) int {
	// Check in reverse order (top-most drawn last).
	for i := len(seq.Players) - 1; i >= 0; i-- {
		center := cw.viewport.RelToPixel(seq.Players[i].Position)
		dx := pos.X - center.X
		dy := pos.Y - center.Y
		dist := math.Sqrt(float64(dx*dx + dy*dy))
		if dist <= playerRadius+4 { // slight tolerance
			return i
		}
	}
	return -1
}

// hitTestAccessory returns the index of the accessory under pos, or -1.
func (cw *CourtWidget) hitTestAccessory(seq *model.Sequence, pos f32.Point) int {
	hitRadius := float64(coneSize + 6) // generous hit area for small shapes
	for i := len(seq.Accessories) - 1; i >= 0; i-- {
		center := cw.viewport.RelToPixel(seq.Accessories[i].Position)
		dx := pos.X - center.X
		dy := pos.Y - center.Y
		dist := math.Sqrt(float64(dx*dx + dy*dy))
		if dist <= hitRadius {
			return i
		}
	}
	return -1
}

// hitTestAction checks if pos is near the midpoint of an action line.
func (cw *CourtWidget) hitTestAction(seq *model.Sequence, pos f32.Point) int {
	const hitThreshold = 12.0
	for i := len(seq.Actions) - 1; i >= 0; i-- {
		from := resolveRef(&cw.viewport, seq.Actions[i].From, seq.Players)
		to := resolveRef(&cw.viewport, seq.Actions[i].To, seq.Players)
		mid := f32.Point{X: (from.X + to.X) / 2, Y: (from.Y + to.Y) / 2}
		dx := pos.X - mid.X
		dy := pos.Y - mid.Y
		dist := math.Sqrt(float64(dx*dx + dy*dy))
		if dist <= hitThreshold {
			return i
		}
	}
	return -1
}

// removeActionsForPlayer removes all actions referencing the given player ID.
func (cw *CourtWidget) removeActionsForPlayer(seq *model.Sequence, playerID string) {
	filtered := seq.Actions[:0]
	for _, a := range seq.Actions {
		if (a.From.IsPlayer && a.From.PlayerID == playerID) ||
			(a.To.IsPlayer && a.To.PlayerID == playerID) {
			continue
		}
		filtered = append(filtered, a)
	}
	seq.Actions = filtered
}

func (cw *CourtWidget) drawSequence(gtx layout.Context, th *material.Theme, seq *model.Sequence, state *editor.EditorState) {
	sel := state.SelectedElement

	// Draw accessories first (below everything).
	for i := range seq.Accessories {
		selected := sel != nil && sel.Kind == editor.SelectAccessory && sel.Index == i && sel.SeqIndex == cw.seqIndex
		DrawAccessory(gtx.Ops, &cw.viewport, &seq.Accessories[i], selected)
	}

	// Draw actions (arrows).
	for i := range seq.Actions {
		DrawAction(gtx.Ops, &cw.viewport, &seq.Actions[i], seq.Players)
	}

	// Draw players on top with labels.
	for i := range seq.Players {
		selected := sel != nil && sel.Kind == editor.SelectPlayer && sel.Index == i && sel.SeqIndex == cw.seqIndex
		hasBall := seq.BallCarrier != "" && seq.Players[i].ID == seq.BallCarrier
		DrawPlayerWithLabel(gtx, th, &cw.viewport, &seq.Players[i], selected, hasBall)
	}

	// Draw callouts above players.
	for i := range seq.Players {
		DrawCallout(gtx, th, &cw.viewport, &seq.Players[i])
	}

	// Draw "action from" indicator: highlight the first-click player.
	if state.ActionFrom != nil {
		for i := range seq.Players {
			if seq.Players[i].ID == *state.ActionFrom {
				center := cw.viewport.RelToPixel(seq.Players[i].Position)
				court.DrawCircleOutline(gtx.Ops, center, playerRadius+6, 2,
					colorPass)
				break
			}
		}
	}
}

// LayoutAnimated renders the court with an animated frame (during playback).
func (cw *CourtWidget) LayoutAnimated(gtx layout.Context, th *material.Theme, frame *anim.AnimatedFrame) layout.Dimensions {
	if cw.exercise == nil {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}

	cw.updateGeometry()
	size := gtx.Constraints.Max

	cw.viewport = court.ComputeViewport(
		cw.exercise.CourtType,
		cw.geom,
		image.Pt(size.X, size.Y),
		10,
	)

	// Draw court.
	switch cw.exercise.CourtStandard {
	case model.NBA:
		court.DrawNBACourt(gtx.Ops, cw.exercise.CourtType, &cw.viewport, cw.geom)
	default:
		court.DrawFIBACourt(gtx.Ops, cw.exercise.CourtType, &cw.viewport, cw.geom)
	}

	// Draw animated elements.
	cw.drawAnimatedFrame(gtx, th, frame)

	return layout.Dimensions{Size: size}
}

// LayoutStatic renders the court and current sequence elements without
// any pointer interaction or selection — suitable for preview panels.
func (cw *CourtWidget) LayoutStatic(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if cw.exercise == nil {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}

	cw.updateGeometry()
	size := gtx.Constraints.Max

	cw.viewport = court.ComputeViewport(
		cw.exercise.CourtType,
		cw.geom,
		image.Pt(size.X, size.Y),
		10,
	)

	// Draw court.
	switch cw.exercise.CourtStandard {
	case model.NBA:
		court.DrawNBACourt(gtx.Ops, cw.exercise.CourtType, &cw.viewport, cw.geom)
	default:
		court.DrawFIBACourt(gtx.Ops, cw.exercise.CourtType, &cw.viewport, cw.geom)
	}

	// Draw elements of current sequence.
	seq := cw.currentSequence()
	if seq != nil {
		cw.drawStaticSequence(gtx, th, seq)
	}

	return layout.Dimensions{Size: size}
}

// drawStaticSequence renders sequence elements without selection highlighting.
func (cw *CourtWidget) drawStaticSequence(gtx layout.Context, th *material.Theme, seq *model.Sequence) {
	for i := range seq.Accessories {
		DrawAccessory(gtx.Ops, &cw.viewport, &seq.Accessories[i], false)
	}
	for i := range seq.Actions {
		DrawAction(gtx.Ops, &cw.viewport, &seq.Actions[i], seq.Players)
	}
	for i := range seq.Players {
		hasBall := seq.BallCarrier != "" && seq.Players[i].ID == seq.BallCarrier
		DrawPlayerWithLabel(gtx, th, &cw.viewport, &seq.Players[i], false, hasBall)
	}
	for i := range seq.Players {
		DrawCallout(gtx, th, &cw.viewport, &seq.Players[i])
	}
}

// drawAnimatedFrame renders an interpolated frame on the court.
func (cw *CourtWidget) drawAnimatedFrame(gtx layout.Context, th *material.Theme, frame *anim.AnimatedFrame) {
	// Accessories (bottom layer).
	for i := range frame.Accessories {
		DrawAccessoryWithOpacity(gtx.Ops, &cw.viewport, &frame.Accessories[i].Accessory, frame.Accessories[i].Opacity)
	}

	// Actions with progressive drawing.
	// Build a plain player list for resolveRef.
	players := make([]model.Player, len(frame.Players))
	for i := range frame.Players {
		players[i] = frame.Players[i].Player
	}
	for i := range frame.Actions {
		DrawActionWithProgress(gtx.Ops, &cw.viewport, &frame.Actions[i].Action, players, frame.Actions[i].Progress)
	}

	// Players (top layer) with opacity — ball drawn separately during animation.
	for i := range frame.Players {
		DrawPlayerWithOpacity(gtx, th, &cw.viewport, &frame.Players[i].Player, frame.Players[i].Opacity, false)
	}

	// Draw callouts above animated players.
	for i := range frame.Players {
		DrawCalloutWithOpacity(gtx, th, &cw.viewport, &frame.Players[i].Player, frame.Players[i].Opacity)
	}

	// Draw ball at interpolated position.
	if frame.BallCarrier != "" && frame.BallOpacity > 0 {
		ballPixel := cw.viewport.RelToPixel(frame.BallPos)
		DrawBallWithOpacity(gtx.Ops, ballPixel, frame.BallOpacity)
	}
}

// clampPosition clamps a relative position to [0,1].
func clampPosition(p model.Position) model.Position {
	if p[0] < 0 {
		p[0] = 0
	}
	if p[0] > 1 {
		p[0] = 1
	}
	if p[1] < 0 {
		p[1] = 0
	}
	if p[1] > 1 {
		p[1] = 1
	}
	return p
}
