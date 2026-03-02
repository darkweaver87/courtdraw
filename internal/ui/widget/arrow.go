package widget

import (
	"image/color"

	"gioui.org/f32"
	"gioui.org/op"

	"github.com/darkweaver87/courtdraw/internal/court"
	"github.com/darkweaver87/courtdraw/internal/model"
)

// Action visual constants.
const (
	arrowLineWidth  = 2.5
	arrowHeadSize   = 10
	zigzagAmplitude = 5
	zigzagSegments  = 8
	dashLen         = 8
	gapLen          = 5
)

// Action colors from spec.
var (
	colorSprint   = color.NRGBA{R: 0xe6, G: 0x39, B: 0x46, A: 0xff} // #e63946 red
	colorPass     = color.NRGBA{R: 0xf4, G: 0xa2, B: 0x61, A: 0xff} // #f4a261 orange
	colorDribble  = color.NRGBA{R: 0xf4, G: 0xa2, B: 0x61, A: 0xff} // #f4a261 orange
	colorCloseOut = color.NRGBA{R: 0x2a, G: 0x6f, B: 0xdb, A: 0xff} // #2a6fdb blue
	colorCut      = color.NRGBA{R: 0xe6, G: 0x39, B: 0x46, A: 0xff} // #e63946 red
	colorScreen   = color.NRGBA{R: 0xff, G: 0xb7, B: 0x03, A: 0xff} // #ffb703 yellow
	colorDefault  = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff} // white
)

// DrawActionWithProgress draws an action with progressive stroke (0.0–1.0).
func DrawActionWithProgress(ops *op.Ops, vp *court.Viewport, action *model.Action, players []model.Player, progress float64) {
	if progress <= 0 {
		return
	}
	if progress >= 1.0 {
		DrawAction(ops, vp, action, players)
		return
	}
	from := resolveRef(vp, action.From, players)
	to := resolveRef(vp, action.To, players)
	// Compute partial endpoint.
	partialTo := f32.Point{
		X: from.X + (to.X-from.X)*float32(progress),
		Y: from.Y + (to.Y-from.Y)*float32(progress),
	}

	col := actionColor(action.Type)
	switch action.Type {
	case model.ActionPass, model.ActionContest:
		court.DrawDashedLine(ops, from, partialTo, arrowLineWidth, dashLen, gapLen, col)
		if progress > 0.3 {
			court.DrawArrowhead(ops, from, partialTo, arrowHeadSize, col)
		}
	case model.ActionDribble:
		court.DrawZigzag(ops, from, partialTo, arrowLineWidth, zigzagAmplitude, zigzagSegments, col)
		if progress > 0.3 {
			court.DrawArrowhead(ops, from, partialTo, arrowHeadSize, col)
		}
	case model.ActionScreen:
		court.DrawLine(ops, from, partialTo, arrowLineWidth*3, col)
	default:
		court.DrawLine(ops, from, partialTo, arrowLineWidth, col)
		if progress > 0.3 {
			court.DrawArrowhead(ops, from, partialTo, arrowHeadSize, col)
		}
	}
}

// actionColor returns the color for a given action type.
func actionColor(at model.ActionType) color.NRGBA {
	switch at {
	case model.ActionPass:
		return colorPass
	case model.ActionDribble:
		return colorDribble
	case model.ActionSprint, model.ActionCut, model.ActionShotLayup,
		model.ActionShotPushup, model.ActionShotJump, model.ActionReverse:
		return colorSprint
	case model.ActionCloseOut, model.ActionContest:
		return colorCloseOut
	case model.ActionScreen:
		return colorScreen
	default:
		return colorDefault
	}
}

// DrawAction draws an action (arrow/movement) between elements.
func DrawAction(ops *op.Ops, vp *court.Viewport, action *model.Action, players []model.Player) {
	from := resolveRef(vp, action.From, players)
	to := resolveRef(vp, action.To, players)

	switch action.Type {
	case model.ActionPass:
		court.DrawDashedLine(ops, from, to, arrowLineWidth, dashLen, gapLen, colorPass)
		court.DrawArrowhead(ops, from, to, arrowHeadSize, colorPass)

	case model.ActionDribble:
		court.DrawZigzag(ops, from, to, arrowLineWidth, zigzagAmplitude, zigzagSegments, colorDribble)
		court.DrawArrowhead(ops, from, to, arrowHeadSize, colorDribble)

	case model.ActionSprint:
		court.DrawLine(ops, from, to, arrowLineWidth, colorSprint)
		court.DrawArrowhead(ops, from, to, arrowHeadSize, colorSprint)

	case model.ActionCloseOut:
		court.DrawLine(ops, from, to, arrowLineWidth, colorCloseOut)
		court.DrawArrowhead(ops, from, to, arrowHeadSize, colorCloseOut)

	case model.ActionCut:
		court.DrawLine(ops, from, to, arrowLineWidth, colorCut)
		court.DrawArrowhead(ops, from, to, arrowHeadSize, colorCut)

	case model.ActionScreen:
		// thick short bar perpendicular to direction
		court.DrawLine(ops, from, to, arrowLineWidth*3, colorScreen)

	case model.ActionShotLayup, model.ActionShotPushup, model.ActionShotJump:
		court.DrawLine(ops, from, to, arrowLineWidth, colorSprint)
		court.DrawArrowhead(ops, from, to, arrowHeadSize, colorSprint)

	case model.ActionContest:
		court.DrawDashedLine(ops, from, to, arrowLineWidth, dashLen, gapLen, colorCloseOut)
		court.DrawArrowhead(ops, from, to, arrowHeadSize, colorCloseOut)

	case model.ActionReverse:
		court.DrawLine(ops, from, to, arrowLineWidth, colorSprint)
		court.DrawArrowhead(ops, from, to, arrowHeadSize, colorSprint)

	default:
		court.DrawLine(ops, from, to, arrowLineWidth, colorDefault)
		court.DrawArrowhead(ops, from, to, arrowHeadSize, colorDefault)
	}
}

// resolveRef resolves an ActionRef to a pixel position.
func resolveRef(vp *court.Viewport, ref model.ActionRef, players []model.Player) f32.Point {
	if ref.IsPlayer {
		for i := range players {
			if players[i].ID == ref.PlayerID {
				return vp.RelToPixel(players[i].Position)
			}
		}
		// fallback: center of court
		return vp.RelToPixel(model.Position{0.5, 0.5})
	}
	return vp.RelToPixel(ref.Position)
}
