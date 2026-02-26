package widget

import (
	"image"

	"gioui.org/layout"
	"gioui.org/widget/material"

	"github.com/darkweaver87/courtdraw/internal/court"
	"github.com/darkweaver87/courtdraw/internal/model"
)

// CourtWidget renders a basketball court with exercise elements.
type CourtWidget struct {
	exercise *model.Exercise
	seqIndex int
	geom     *court.CourtGeometry
	viewport court.Viewport
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

// Layout renders the court widget.
func (cw *CourtWidget) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if cw.exercise == nil || cw.geom == nil {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}

	size := gtx.Constraints.Max

	// compute viewport
	cw.viewport = court.ComputeViewport(
		cw.exercise.CourtType,
		cw.geom,
		image.Pt(size.X, size.Y),
		10,
	)

	// draw court
	switch cw.exercise.CourtStandard {
	case model.NBA:
		court.DrawNBACourt(gtx.Ops, cw.exercise.CourtType, &cw.viewport, cw.geom)
	default:
		court.DrawFIBACourt(gtx.Ops, cw.exercise.CourtType, &cw.viewport, cw.geom)
	}

	// draw exercise elements for current sequence
	if cw.seqIndex < len(cw.exercise.Sequences) {
		seq := &cw.exercise.Sequences[cw.seqIndex]
		cw.drawSequence(gtx, th, seq)
	}

	return layout.Dimensions{Size: size}
}

func (cw *CourtWidget) drawSequence(gtx layout.Context, th *material.Theme, seq *model.Sequence) {
	// draw accessories first (below everything)
	for i := range seq.Accessories {
		DrawAccessory(gtx.Ops, &cw.viewport, &seq.Accessories[i])
	}

	// draw actions (arrows)
	for i := range seq.Actions {
		DrawAction(gtx.Ops, &cw.viewport, &seq.Actions[i], seq.Players)
	}

	// draw players on top with labels
	for i := range seq.Players {
		DrawPlayerWithLabel(gtx, th, &cw.viewport, &seq.Players[i])
	}
}
