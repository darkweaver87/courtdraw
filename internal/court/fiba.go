package court

import (
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/op"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// FIBA court dimensions in meters.
func FIBAGeometry() *CourtGeometry {
	return &CourtGeometry{
		Width:                15.0,
		Length:               28.0,
		BasketOffset:         1.575,
		LaneWidth:            4.90,
		LaneLength:           5.80,
		ThreePointRadius:     6.75,
		ThreePointCornerDist: 0.90,
		FreeThrowRadius:      1.80,
		CenterCircleRadius:   1.80,
		RestrictedAreaRadius: 1.25,
		BackboardWidth:       1.80,
		RimDiameter:          0.45,
		LineWidth:            0.05,
	}
}

// DrawFIBACourt draws a FIBA basketball court.
func DrawFIBACourt(ops *op.Ops, courtType model.CourtType, vp *Viewport, geom *CourtGeometry) {
	drawCourt(ops, courtType, vp, geom)
}

// drawCourt draws the court for any standard using the geometry.
func drawCourt(ops *op.Ops, courtType model.CourtType, vp *Viewport, geom *CourtGeometry) {
	lineCol := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	bgCol := color.NRGBA{R: 0xc8, G: 0x96, B: 0x64, A: 0xff}

	lineW := float32(vp.MeterToPixel(geom.LineWidth, geom, courtType))
	if lineW < 1.5 {
		lineW = 1.5
	}

	// court dimensions in this view
	courtW, courtH := courtDimensions(geom, courtType)

	// helper: meters from bottom-left to pixel coords
	m2p := func(mx, my float64) f32.Point {
		rx := mx / courtW
		ry := my / courtH
		return vp.RelToPixel(model.Position{rx, ry})
	}

	// green background
	topLeft := m2p(0, courtH)
	botRight := m2p(courtW, 0)
	DrawRectFill(ops, topLeft, botRight, bgCol)

	// court outline
	DrawRect(ops, topLeft, botRight, lineW, lineCol)

	isFullCourt := courtType == model.FullCourt

	// near basket (bottom)
	drawBasketEnd(ops, vp, geom, courtType, lineW, lineCol, courtW, courtH, false)

	if isFullCourt {
		// half-court line
		hlY := courtH / 2
		DrawLine(ops, m2p(0, hlY), m2p(courtW, hlY), lineW, lineCol)

		// center circle
		centerPt := m2p(courtW/2, hlY)
		centerR := float32(vp.MeterToPixel(geom.CenterCircleRadius, geom, courtType))
		DrawCircleOutline(ops, centerPt, centerR, lineW, lineCol)

		// far basket (top) — mirrored
		drawBasketEnd(ops, vp, geom, courtType, lineW, lineCol, courtW, courtH, true)
	}
}

// drawBasketEnd draws all elements for one end of the court.
// If mirrored=true, draws the far end (top of the court).
func drawBasketEnd(ops *op.Ops, vp *Viewport, geom *CourtGeometry, courtType model.CourtType, lineW float32, lineCol color.NRGBA, courtW, courtH float64, mirrored bool) {
	// helper: y-coordinate accounting for mirroring
	yCoord := func(y float64) float64 {
		if mirrored {
			return courtH - y
		}
		return y
	}

	m2p := func(mx, my float64) f32.Point {
		rx := mx / courtW
		ry := my / courtH
		return vp.RelToPixel(model.Position{rx, ry})
	}

	basketX := courtW / 2
	basketY := yCoord(geom.BasketOffset)

	// lane (paint)
	laneLeft := (courtW - geom.LaneWidth) / 2
	laneRight := (courtW + geom.LaneWidth) / 2
	laneTop := yCoord(geom.LaneLength)

	if mirrored {
		DrawRect(ops, m2p(laneLeft, laneTop), m2p(laneRight, yCoord(0)), lineW, lineCol)
	} else {
		DrawRect(ops, m2p(laneLeft, laneTop), m2p(laneRight, yCoord(0)), lineW, lineCol)
	}

	// free-throw line is at lane top, already part of lane rect

	// free-throw semicircle (facing away from basket)
	ftCenter := m2p(basketX, laneTop)
	ftR := float32(vp.MeterToPixel(geom.FreeThrowRadius, geom, courtType))
	if mirrored {
		// semicircle opens downward (toward baseline at top)
		DrawArc(ops, ftCenter, ftR, math.Pi, 2*math.Pi, lineW, lineCol)
	} else {
		// semicircle opens upward (toward center)
		DrawArc(ops, ftCenter, ftR, 0, math.Pi, lineW, lineCol)
	}

	// three-point line
	draw3ptLine(ops, vp, geom, courtType, lineW, lineCol, courtW, courtH, mirrored)

	// restricted area (no-charge semicircle)
	raCenter := m2p(basketX, basketY)
	raR := float32(vp.MeterToPixel(geom.RestrictedAreaRadius, geom, courtType))
	if mirrored {
		DrawArc(ops, raCenter, raR, math.Pi, 2*math.Pi, lineW, lineCol)
	} else {
		DrawArc(ops, raCenter, raR, 0, math.Pi, lineW, lineCol)
	}

	// backboard
	bbHalf := geom.BackboardWidth / 2
	bbY := yCoord(geom.BasketOffset - 0.15) // backboard slightly behind basket
	DrawLine(ops, m2p(basketX-bbHalf, bbY), m2p(basketX+bbHalf, bbY), lineW*1.5, lineCol)

	// rim
	rimCenter := m2p(basketX, basketY)
	rimR := float32(vp.MeterToPixel(geom.RimDiameter/2, geom, courtType))
	DrawCircleOutline(ops, rimCenter, rimR, lineW, color.NRGBA{R: 0xff, G: 0x66, B: 0x00, A: 0xff})
}

// draw3ptLine draws the three-point line for one end.
func draw3ptLine(ops *op.Ops, vp *Viewport, geom *CourtGeometry, courtType model.CourtType, lineW float32, lineCol color.NRGBA, courtW, courtH float64, mirrored bool) {
	yCoord := func(y float64) float64 {
		if mirrored {
			return courtH - y
		}
		return y
	}

	m2p := func(mx, my float64) f32.Point {
		rx := mx / courtW
		ry := my / courtH
		return vp.RelToPixel(model.Position{rx, ry})
	}

	basketX := courtW / 2
	basketY := geom.BasketOffset // physical distance from baseline

	// corner 3pt straight lines
	cornerDist := geom.ThreePointCornerDist

	// the arc starts where the straight line meets the arc
	// arc intersect Y: y where distance from basket = 3pt radius
	// basket is at (basketX, basketY), corner line at x = cornerDist or courtW-cornerDist
	// distance = sqrt((basketX - cornerX)^2 + (y - basketY)^2) = ThreePointRadius
	cornerX := cornerDist
	dx := basketX - cornerX
	dy2 := geom.ThreePointRadius*geom.ThreePointRadius - dx*dx
	arcStartY := basketY + math.Sqrt(dy2)

	// left corner
	DrawLine(ops, m2p(cornerX, yCoord(0)), m2p(cornerX, yCoord(arcStartY)), lineW, lineCol)
	// right corner
	rightCornerX := courtW - cornerDist
	DrawLine(ops, m2p(rightCornerX, yCoord(0)), m2p(rightCornerX, yCoord(arcStartY)), lineW, lineCol)

	// arc from left corner to right corner
	arcCenter := m2p(basketX, yCoord(basketY))
	arcR := float32(vp.MeterToPixel(geom.ThreePointRadius, geom, courtType))

	// calculate angles for the arc endpoints
	startAngleRaw := math.Atan2(arcStartY-basketY, cornerX-basketX)
	endAngleRaw := math.Atan2(arcStartY-basketY, rightCornerX-basketX)

	if mirrored {
		// screen Y is flipped, so angles need adjustment
		// in screen coords the arc goes from right side to left side
		DrawArc(ops, arcCenter, arcR, -startAngleRaw, -endAngleRaw, lineW, lineCol)
	} else {
		DrawArc(ops, arcCenter, arcR, startAngleRaw, endAngleRaw, lineW, lineCol)
	}
}
