package court

import (
	"image"
	"image/color"
	"math"
	"strconv"

	"golang.org/x/image/font"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// StepBadgeRadius is the base radius of the step number circle.
const StepBadgeRadius = 8

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

// ResolveWaypoints converts model waypoints to pixel-space Points.
func ResolveWaypoints(vp *Viewport, waypoints []model.Position) []Point {
	pts := make([]Point, len(waypoints))
	for i, wp := range waypoints {
		pts[i] = vp.RelToPixel(wp)
	}
	return pts
}

// DrawAction draws an action (arrow/movement) between elements.
func DrawAction(img *image.RGBA, vp *Viewport, action *model.Action, players []model.Player) {
	from := ResolveRef(vp, action.From, players)
	to := ResolveRef(vp, action.To, players)
	col := ActionColor(action.Type)
	lw := vp.S(ArrowLineWidth)
	ah := vp.S(ArrowHeadSize)

	if len(action.Waypoints) > 0 {
		wps := ResolveWaypoints(vp, action.Waypoints)
		pts := BezierPath(from, to, wps, 16)
		drawActionPath(img, vp, action.Type, pts, lw, ah, col)
		return
	}

	za := vp.S(ZigzagAmplitude)
	dl := vp.S(DashLen)
	gl := vp.S(GapLen)

	switch action.Type {
	case model.ActionPass:
		DrawDashedLine(img, from, to, lw, dl, gl, col)
		DrawArrowhead(img, from, to, ah, col)
	case model.ActionDribble:
		DrawZigzag(img, from, to, lw, za, ZigzagSegments, col)
		DrawArrowhead(img, from, to, ah, col)
	case model.ActionScreen:
		DrawLine(img, from, to, lw*3, col)
	case model.ActionContest:
		DrawDashedLine(img, from, to, lw, dl, gl, col)
		DrawArrowhead(img, from, to, ah, col)
	default:
		DrawLine(img, from, to, lw, col)
		DrawArrowhead(img, from, to, ah, col)
	}
}

// drawActionPath draws an action along a curved polyline path.
func drawActionPath(img *image.RGBA, vp *Viewport, actionType model.ActionType, pts []Point, lw, ah float32, col color.NRGBA) {
	za := vp.S(ZigzagAmplitude)
	dl := vp.S(DashLen)
	gl := vp.S(GapLen)
	zsl := vp.S(12) // zigzag segment length

	switch actionType {
	case model.ActionPass, model.ActionContest:
		DrawDashedPolyline(img, pts, lw, dl, gl, col)
		DrawArrowheadAtEnd(img, pts, ah, col)
	case model.ActionDribble:
		DrawZigzagPolyline(img, pts, lw, za, zsl, col)
		DrawArrowheadAtEnd(img, pts, ah, col)
	case model.ActionScreen:
		DrawPolyline(img, pts, lw*3, col)
	default:
		DrawPolyline(img, pts, lw, col)
		DrawArrowheadAtEnd(img, pts, ah, col)
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
	col := ActionColor(action.Type)
	lw := vp.S(ArrowLineWidth)
	ah := vp.S(ArrowHeadSize)

	if len(action.Waypoints) > 0 {
		wps := ResolveWaypoints(vp, action.Waypoints)
		fullPts := BezierPath(from, to, wps, 16)
		// Truncate polyline to progress fraction.
		pts := truncatePolyline(fullPts, progress)
		drawActionPath(img, vp, action.Type, pts, lw, ah, col)
		return
	}

	partialTo := Pt(from.X+(to.X-from.X)*float32(progress), from.Y+(to.Y-from.Y)*float32(progress))
	za := vp.S(ZigzagAmplitude)
	dl := vp.S(DashLen)
	gl := vp.S(GapLen)

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

// truncatePolyline returns points up to a given fraction (0–1) of the total length.
func truncatePolyline(pts []Point, frac float64) []Point {
	if frac >= 1 || len(pts) < 2 {
		return pts
	}
	total := PolylineLength(pts)
	target := total * frac
	walked := 0.0
	result := []Point{pts[0]}
	for i := 1; i < len(pts); i++ {
		dx := float64(pts[i].X - pts[i-1].X)
		dy := float64(pts[i].Y - pts[i-1].Y)
		segLen := math.Sqrt(dx*dx + dy*dy)
		if walked+segLen >= target {
			t := (target - walked) / segLen
			result = append(result, Pt(
				pts[i-1].X+float32(t*dx),
				pts[i-1].Y+float32(t*dy),
			))
			return result
		}
		result = append(result, pts[i])
		walked += segLen
	}
	return result
}

// DrawActionHighlight draws a wide semi-transparent yellow line behind an action.
func DrawActionHighlight(img *image.RGBA, vp *Viewport, action *model.Action, players []model.Player) {
	from := ResolveRef(vp, action.From, players)
	to := ResolveRef(vp, action.To, players)
	col := color.NRGBA{R: 0xff, G: 0xff, B: 0x00, A: 0x66}
	DrawLine(img, from, to, vp.S(ArrowLineWidth+6), col)
}

// DrawActionPreview draws a semi-transparent ghost arrow from a source point to a cursor position.
func DrawActionPreview(img *image.RGBA, vp *Viewport, from, to Point, actionType model.ActionType, alpha float64) {
	if alpha <= 0 {
		return
	}
	a := uint8(alpha * 255)
	col := ActionColor(actionType)
	col.A = a

	lw := vp.S(ArrowLineWidth)
	ah := vp.S(ArrowHeadSize)
	za := vp.S(ZigzagAmplitude)
	dl := vp.S(DashLen)
	gl := vp.S(GapLen)

	switch actionType {
	case model.ActionPass, model.ActionContest:
		DrawDashedLine(img, from, to, lw, dl, gl, col)
		DrawArrowhead(img, from, to, ah, col)
	case model.ActionDribble:
		DrawZigzag(img, from, to, lw, za, ZigzagSegments, col)
		DrawArrowhead(img, from, to, ah, col)
	case model.ActionScreen:
		DrawLine(img, from, to, lw*3, col)
	default:
		DrawLine(img, from, to, lw, col)
		DrawArrowhead(img, from, to, ah, col)
	}
}

// DrawStepBadge draws a circled step number at the midpoint of an action arrow.
func DrawStepBadge(img *image.RGBA, vp *Viewport, action *model.Action, players []model.Player, face font.Face) {
	step := action.EffectiveStep()
	from := ResolveRef(vp, action.From, players)
	to := ResolveRef(vp, action.To, players)
	mid := Pt((from.X+to.X)/2, (from.Y+to.Y)/2)
	r := vp.S(StepBadgeRadius)
	// White filled circle with action-colored text.
	DrawCircleFill(img, mid, r, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xdd})
	DrawCircleOutline(img, mid, r, vp.S(1.0), color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x66})
	if face != nil {
		col := ActionColor(action.Type)
		DrawText(img, strconv.Itoa(step), mid, face, col)
	}
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
