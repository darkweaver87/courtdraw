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
	ArrowLineWidth    = 3.5
	ArrowHeadSize     = 12
	ArrowHeadRatio    = 3.5 // arrowhead size = line width × ratio
	ScreenWidthRatio  = 2.5 // screen line = base width × ratio
	ZigzagAmplitude  = 7
	ZigzagSegmentLen = 10 // fixed segment length for consistent zigzag
	DashLen          = 10
	GapLen           = 6
)

// ActionLineColor is the standard color for all action arrows (black, per basketball convention).
var ActionLineColor = color.NRGBA{R: 0x1a, G: 0x1a, B: 0x1a, A: 0xff}

// ActionErrorColor is used for actions with strong errors (red).
var ActionErrorColor = color.NRGBA{R: 0xcc, G: 0x22, B: 0x22, A: 0xff}

// ActionWarningColor is used for actions with warnings (orange).
var ActionWarningColor = color.NRGBA{R: 0xcc, G: 0x88, B: 0x00, A: 0xff}

// ActionColor returns the color for a given action type.
func ActionColor(_ model.ActionType) color.NRGBA {
	return ActionLineColor
}

// ResolveWaypoints converts model waypoints to pixel-space Points.
func ResolveWaypoints(vp *Viewport, waypoints []model.Position) []Point {
	pts := make([]Point, len(waypoints))
	for i, wp := range waypoints {
		pts[i] = vp.RelToPixel(wp)
	}
	return pts
}

// DrawActionWithColor draws an action with a custom color override.
func DrawActionWithColor(img *image.RGBA, vp *Viewport, action *model.Action, players []model.Player, overrideCol color.NRGBA) {
	drawActionImpl(img, vp, action, players, overrideCol)
}

// DrawAction draws an action (arrow/movement) between elements.
func DrawAction(img *image.RGBA, vp *Viewport, action *model.Action, players []model.Player) {
	drawActionImpl(img, vp, action, players, ActionColor(action.Type))
}

func drawActionImpl(img *image.RGBA, vp *Viewport, action *model.Action, players []model.Player, col color.NRGBA) {
	from := ResolveRef(vp, action.From, players)
	to := ResolveRef(vp, action.To, players)
	lw := vp.S(ArrowLineWidth)
	ah := max(vp.S(ArrowHeadSize), lw*ArrowHeadRatio)
	pr := vp.S(PlayerRadius+8) + ah*1.2 // offset: player body + generous margin + arrowhead
	dotR := lw * 1.5         // endpoint dot radius

	// Shorten endpoints to avoid arrowhead under players.
	drawFrom := from
	drawTo := to
	if action.From.IsPlayer {
		drawFrom = ShortenLine(to, from, pr)
	}
	if action.To.IsPlayer {
		drawTo = ShortenLine(from, to, pr)
	}

	if len(action.Waypoints) > 0 {
		wps := ResolveWaypoints(vp, action.Waypoints)
		pts := BezierPath(drawFrom, drawTo, wps, 16)
		drawActionPath(img, vp, action.Type, pts, lw, ah, col)
		DrawEndpointDot(img, drawFrom, dotR, col)
		return
	}

	za := vp.S(ZigzagAmplitude)
	dl := vp.S(DashLen)
	gl := vp.S(GapLen)

	// Shorten line end by arrowhead size so dashes/zigzags don't overlap the arrowhead.
	lineEnd := ShortenLine(drawFrom, drawTo, ah)
	// For zigzag: extra straight tail so it ends cleanly before the arrowhead.
	zigzagEnd := ShortenLine(drawFrom, drawTo, ah*2.5)

	switch model.NormalizeActionType(action.Type) {
	case model.ActionPass:
		DrawDashedLine(img, drawFrom, lineEnd, lw, dl, gl, col)
		DrawArrowhead(img, drawFrom, drawTo, ah, col)
	case model.ActionDribble:
		DrawZigzag(img, drawFrom, zigzagEnd, lw, za, vp.S(ZigzagSegmentLen), col)
		DrawLine(img, zigzagEnd, lineEnd, lw, col) // straight tail into arrowhead
		DrawArrowhead(img, drawFrom, drawTo, ah, col)
	case model.ActionCut:
		DrawLine(img, drawFrom, lineEnd, lw, col)
		DrawArrowhead(img, drawFrom, drawTo, ah, col)
	case model.ActionScreen:
		DrawLine(img, drawFrom, drawTo, lw, col)
		DrawScreenBar(img, drawTo, drawFrom, vp.S(PlayerRadius*0.8), lw*1.5, col)
	case model.ActionShot:
		DrawDashedLine(img, drawFrom, lineEnd, lw, dl, gl, col)
		DrawArrowhead(img, drawFrom, drawTo, ah, col)
	case model.ActionHandoff:
		DrawLine(img, drawFrom, lineEnd, lw, col)
		DrawHandoffBars(img, drawFrom, drawTo, lw, vp.S(6), col)
		DrawArrowhead(img, drawFrom, drawTo, ah, col)
	default:
		DrawLine(img, drawFrom, lineEnd, lw, col)
		DrawArrowhead(img, drawFrom, drawTo, ah, col)
	}
	// Endpoint dots.
	DrawEndpointDot(img, drawFrom, dotR, col)
}

// drawActionPath draws an action along a curved polyline path.
func drawActionPath(img *image.RGBA, vp *Viewport, actionType model.ActionType, pts []Point, lw, ah float32, col color.NRGBA) {
	za := vp.S(ZigzagAmplitude)
	dl := vp.S(DashLen)
	gl := vp.S(GapLen)
	zsl := vp.S(12) // zigzag segment length

	// Shorten the line path so dashes/zigzags don't overlap the arrowhead.
	linePts := ShortenPolyline(pts, ah)
	zigzagPts := ShortenPolyline(pts, ah*2.5) // extra space for straight tail

	switch model.NormalizeActionType(actionType) {
	case model.ActionPass, model.ActionShot:
		DrawDashedPolyline(img, linePts, lw, dl, gl, col)
		DrawArrowheadAtEnd(img, pts, ah, col)
	case model.ActionDribble:
		DrawZigzagPolyline(img, zigzagPts, lw, za, zsl, col)
		// Straight tail from zigzag end to arrowhead base.
		if len(zigzagPts) > 0 && len(linePts) > 0 {
			DrawLine(img, zigzagPts[len(zigzagPts)-1], linePts[len(linePts)-1], lw, col)
		}
		DrawArrowheadAtEnd(img, pts, ah, col)
	case model.ActionScreen:
		DrawPolyline(img, pts, lw, col)
		if len(pts) >= 2 {
			DrawScreenBar(img, pts[len(pts)-1], pts[len(pts)-2], lw*4, lw*1.5, col)
		}
	default:
		DrawPolyline(img, linePts, lw, col)
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
	ah := max(vp.S(ArrowHeadSize), lw*ArrowHeadRatio)

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

	switch model.NormalizeActionType(action.Type) {
	case model.ActionPass, model.ActionShot:
		DrawDashedLine(img, from, partialTo, lw, dl, gl, col)
		if progress > 0.3 {
			DrawArrowhead(img, from, partialTo, ah, col)
		}
	case model.ActionDribble:
		DrawZigzag(img, from, partialTo, lw, za, vp.S(ZigzagSegmentLen), col)
		if progress > 0.3 {
			DrawArrowhead(img, from, partialTo, ah, col)
		}
	case model.ActionScreen:
		DrawLine(img, from, partialTo, lw, col)
		if progress >= 1.0 {
			DrawScreenBar(img, partialTo, from, lw*4, lw*1.5, col)
		}
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
	ah := max(vp.S(ArrowHeadSize), lw*ArrowHeadRatio)
	za := vp.S(ZigzagAmplitude)
	dl := vp.S(DashLen)
	gl := vp.S(GapLen)

	switch actionType {
	case model.ActionPass, model.ActionContest:
		DrawDashedLine(img, from, to, lw, dl, gl, col)
		DrawArrowhead(img, from, to, ah, col)
	case model.ActionDribble:
		DrawZigzag(img, from, to, lw, za, vp.S(ZigzagSegmentLen), col)
		DrawArrowhead(img, from, to, ah, col)
	case model.ActionScreen:
		DrawLine(img, from, to, lw*ScreenWidthRatio, col)
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

// ShortenLine offsets an endpoint inward by `offset` pixels along the from→to direction.
func ShortenLine(from, to Point, offset float32) Point {
	dx := to.X - from.X
	dy := to.Y - from.Y
	dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if dist < offset*2 {
		return to
	}
	return Pt(to.X-dx/dist*offset, to.Y-dy/dist*offset)
}

// ShortenPolylineEnd shortens a polyline by offset pixels from the end.
func ShortenPolylineEnd(pts []Point, offset float32) []Point {
	if len(pts) < 2 {
		return pts
	}
	last := pts[len(pts)-1]
	prev := pts[len(pts)-2]
	shortened := ShortenLine(prev, last, offset)
	result := make([]Point, len(pts))
	copy(result, pts)
	result[len(result)-1] = shortened
	return result
}

// DrawEndpointDot draws a small filled circle at a point.
func DrawEndpointDot(img *image.RGBA, center Point, radius float32, col color.NRGBA) {
	DrawCircleFill(img, center, radius, col)
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
