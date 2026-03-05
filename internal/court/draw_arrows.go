package court

import (
	"image"
	"image/color"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// Action visual constants (base sizes at 1x zoom).
const (
	ArrowLineWidth  = 2.5
	ArrowHeadSize   = 10
	ZigzagAmplitude = 5
	ZigzagSegments  = 8
	DashLen         = 8
	GapLen          = 5
)

// Action colors.
var (
	ColorSprint   = color.NRGBA{R: 0xe6, G: 0x39, B: 0x46, A: 0xff}
	ColorPass     = color.NRGBA{R: 0xf4, G: 0xa2, B: 0x61, A: 0xff}
	ColorDribble  = color.NRGBA{R: 0xf4, G: 0xa2, B: 0x61, A: 0xff}
	ColorCloseOut = color.NRGBA{R: 0x2a, G: 0x6f, B: 0xdb, A: 0xff}
	ColorCut      = color.NRGBA{R: 0xe6, G: 0x39, B: 0x46, A: 0xff}
	ColorScreen   = color.NRGBA{R: 0xff, G: 0xb7, B: 0x03, A: 0xff}
	ColorDefault  = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
)

// ActionColor returns the color for a given action type.
func ActionColor(at model.ActionType) color.NRGBA {
	switch at {
	case model.ActionPass:
		return ColorPass
	case model.ActionDribble:
		return ColorDribble
	case model.ActionSprint, model.ActionCut, model.ActionShotLayup,
		model.ActionShotPushup, model.ActionShotJump, model.ActionReverse:
		return ColorSprint
	case model.ActionCloseOut, model.ActionContest:
		return ColorCloseOut
	case model.ActionScreen:
		return ColorScreen
	default:
		return ColorDefault
	}
}

// DrawAction draws an action (arrow/movement) between elements.
func DrawAction(img *image.RGBA, vp *Viewport, action *model.Action, players []model.Player) {
	from := ResolveRef(vp, action.From, players)
	to := ResolveRef(vp, action.To, players)

	lw := vp.S(ArrowLineWidth)
	ah := vp.S(ArrowHeadSize)
	za := vp.S(ZigzagAmplitude)
	dl := vp.S(DashLen)
	gl := vp.S(GapLen)

	switch action.Type {
	case model.ActionPass:
		DrawDashedLine(img, from, to, lw, dl, gl, ColorPass)
		DrawArrowhead(img, from, to, ah, ColorPass)

	case model.ActionDribble:
		DrawZigzag(img, from, to, lw, za, ZigzagSegments, ColorDribble)
		DrawArrowhead(img, from, to, ah, ColorDribble)

	case model.ActionSprint:
		DrawLine(img, from, to, lw, ColorSprint)
		DrawArrowhead(img, from, to, ah, ColorSprint)

	case model.ActionCloseOut:
		DrawLine(img, from, to, lw, ColorCloseOut)
		DrawArrowhead(img, from, to, ah, ColorCloseOut)

	case model.ActionCut:
		DrawLine(img, from, to, lw, ColorCut)
		DrawArrowhead(img, from, to, ah, ColorCut)

	case model.ActionScreen:
		DrawLine(img, from, to, lw*3, ColorScreen)

	case model.ActionShotLayup, model.ActionShotPushup, model.ActionShotJump:
		DrawLine(img, from, to, lw, ColorSprint)
		DrawArrowhead(img, from, to, ah, ColorSprint)

	case model.ActionContest:
		DrawDashedLine(img, from, to, lw, dl, gl, ColorCloseOut)
		DrawArrowhead(img, from, to, ah, ColorCloseOut)

	case model.ActionReverse:
		DrawLine(img, from, to, lw, ColorSprint)
		DrawArrowhead(img, from, to, ah, ColorSprint)

	default:
		DrawLine(img, from, to, lw, ColorDefault)
		DrawArrowhead(img, from, to, ah, ColorDefault)
	}
}

// DrawActionWithProgress draws an action with progressive stroke (0.0–1.0).
func DrawActionWithProgress(img *image.RGBA, vp *Viewport, action *model.Action, players []model.Player, progress float64) {
	if progress <= 0 {
		return
	}
	if progress >= 1.0 {
		DrawAction(img, vp, action, players)
		return
	}

	from := ResolveRef(vp, action.From, players)
	to := ResolveRef(vp, action.To, players)
	partialTo := Pt(
		from.X+(to.X-from.X)*float32(progress),
		from.Y+(to.Y-from.Y)*float32(progress),
	)

	lw := vp.S(ArrowLineWidth)
	ah := vp.S(ArrowHeadSize)
	za := vp.S(ZigzagAmplitude)
	dl := vp.S(DashLen)
	gl := vp.S(GapLen)

	col := ActionColor(action.Type)
	switch action.Type {
	case model.ActionPass, model.ActionContest:
		DrawDashedLine(img, from, partialTo, lw, dl, gl, col)
		if progress > 0.3 {
			DrawArrowhead(img, from, partialTo, ah, col)
		}
	case model.ActionDribble:
		DrawZigzag(img, from, partialTo, lw, za, ZigzagSegments, col)
		if progress > 0.3 {
			DrawArrowhead(img, from, partialTo, ah, col)
		}
	case model.ActionScreen:
		DrawLine(img, from, partialTo, lw*3, col)
	default:
		DrawLine(img, from, partialTo, lw, col)
		if progress > 0.3 {
			DrawArrowhead(img, from, partialTo, ah, col)
		}
	}
}

// DrawActionHighlight draws a wide semi-transparent yellow line behind an action.
func DrawActionHighlight(img *image.RGBA, vp *Viewport, action *model.Action, players []model.Player) {
	from := ResolveRef(vp, action.From, players)
	to := ResolveRef(vp, action.To, players)
	col := color.NRGBA{R: 0xff, G: 0xff, B: 0x00, A: 0x66}
	DrawLine(img, from, to, vp.S(ArrowLineWidth+6), col)
}

// ResolveRef resolves an ActionRef to a pixel position.
func ResolveRef(vp *Viewport, ref model.ActionRef, players []model.Player) Point {
	if ref.IsPlayer {
		for i := range players {
			if players[i].ID == ref.PlayerID {
				return vp.RelToPixel(players[i].Position)
			}
		}
		return vp.RelToPixel(model.Position{0.5, 0.5})
	}
	return vp.RelToPixel(ref.Position)
}
