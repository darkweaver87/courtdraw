package court

import (
	"image"
	"image/color"
	"math"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// FIBAGeometry returns FIBA court dimensions in meters.
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

// DrawFIBACourt draws a FIBA basketball court on the given image.
func DrawFIBACourt(img *image.RGBA, courtType model.CourtType, vp *Viewport, geom *CourtGeometry) {
	drawCourt(img, courtType, vp, geom)
}

// drawCourt draws the court for any standard using the geometry.
func drawCourt(img *image.RGBA, courtType model.CourtType, vp *Viewport, geom *CourtGeometry) {
	lineCol := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	bgCol := color.NRGBA{R: 0xc8, G: 0x96, B: 0x64, A: 0xff} // fallback if texture not loaded
	apronCol := color.NRGBA{R: 0x1a, G: 0x3c, B: 0x6e, A: 0xff} // dark blue apron

	lineW := float32(vp.MeterToPixel(geom.LineWidth, geom, courtType))
	if lineW < 1.5 {
		lineW = 1.5
	}

	courtW, courtH := courtDimensions(geom, courtType)

	m2p := func(mx, my float64) Point {
		rx := mx / courtW
		ry := my / courtH
		return vp.RelToPixel(model.Position{rx, ry})
	}

	// Apron (2m run-off area around the court).
	apronTL := m2p(-ApronMeters, courtH+ApronMeters)
	apronBR := m2p(courtW+ApronMeters, -ApronMeters)
	DrawRectFill(img, apronTL, apronBR, apronCol)

	// Court floor.
	topLeft := m2p(0, courtH)
	botRight := m2p(courtW, 0)
	if woodTile := WoodFloorTexture(); woodTile != nil {
		TileRectScaled(img, topLeft, botRight, woodTile, vp, geom, courtType)
	} else {
		DrawRectFill(img, topLeft, botRight, bgCol)
	}

	// Court outline.
	DrawRect(img, topLeft, botRight, lineW, lineCol)

	isFullCourt := courtType == model.FullCourt

	// Near basket (bottom).
	drawBasketEnd(img, vp, geom, courtType, lineW, lineCol, courtW, courtH, false)

	if isFullCourt {
		// Half-court line.
		hlY := courtH / 2
		DrawLine(img, m2p(0, hlY), m2p(courtW, hlY), lineW, lineCol)

		// Center circle.
		centerPt := m2p(courtW/2, hlY)
		centerR := float32(vp.MeterToPixel(geom.CenterCircleRadius, geom, courtType))
		DrawCircleOutline(img, centerPt, centerR, lineW, lineCol)

		// Far basket (top) — mirrored.
		drawBasketEnd(img, vp, geom, courtType, lineW, lineCol, courtW, courtH, true)
	}
}

// drawBasketEnd draws all elements for one end of the court.
func drawBasketEnd(img *image.RGBA, vp *Viewport, geom *CourtGeometry, courtType model.CourtType, lineW float32, lineCol color.NRGBA, courtW, courtH float64, mirrored bool) {
	yCoord := func(y float64) float64 {
		if mirrored {
			return courtH - y
		}
		return y
	}

	m2p := func(mx, my float64) Point {
		rx := mx / courtW
		ry := my / courtH
		return vp.RelToPixel(model.Position{rx, ry})
	}

	basketX := courtW / 2
	basketY := yCoord(geom.BasketOffset)

	// Lane (paint).
	laneLeft := (courtW - geom.LaneWidth) / 2
	laneRight := (courtW + geom.LaneWidth) / 2
	laneTop := yCoord(geom.LaneLength)

	DrawRect(img, m2p(laneLeft, laneTop), m2p(laneRight, yCoord(0)), lineW, lineCol)

	// Free-throw semicircle.
	ftCenter := m2p(basketX, laneTop)
	ftR := float32(vp.MeterToPixel(geom.FreeThrowRadius, geom, courtType))
	if mirrored {
		DrawArc(img, ftCenter, ftR, math.Pi, 2*math.Pi, lineW, lineCol)
	} else {
		DrawArc(img, ftCenter, ftR, 0, math.Pi, lineW, lineCol)
	}

	// Three-point line.
	draw3ptLine(img, vp, geom, courtType, lineW, lineCol, courtW, courtH, mirrored)

	// Restricted area (no-charge semicircle).
	raCenter := m2p(basketX, basketY)
	raR := float32(vp.MeterToPixel(geom.RestrictedAreaRadius, geom, courtType))
	if mirrored {
		DrawArc(img, raCenter, raR, math.Pi, 2*math.Pi, lineW, lineCol)
	} else {
		DrawArc(img, raCenter, raR, 0, math.Pi, lineW, lineCol)
	}

	// Backboard.
	bbHalf := geom.BackboardWidth / 2
	bbY := yCoord(geom.BasketOffset - 0.15)
	DrawLine(img, m2p(basketX-bbHalf, bbY), m2p(basketX+bbHalf, bbY), lineW*1.5, lineCol)

	// Rim.
	rimCenter := m2p(basketX, basketY)
	rimR := float32(vp.MeterToPixel(geom.RimDiameter/2, geom, courtType))
	DrawCircleOutline(img, rimCenter, rimR, lineW, color.NRGBA{R: 0xff, G: 0x66, B: 0x00, A: 0xff})
}

// draw3ptLine draws the three-point line for one end.
func draw3ptLine(img *image.RGBA, vp *Viewport, geom *CourtGeometry, courtType model.CourtType, lineW float32, lineCol color.NRGBA, courtW, courtH float64, mirrored bool) {
	yCoord := func(y float64) float64 {
		if mirrored {
			return courtH - y
		}
		return y
	}

	m2p := func(mx, my float64) Point {
		rx := mx / courtW
		ry := my / courtH
		return vp.RelToPixel(model.Position{rx, ry})
	}

	basketX := courtW / 2
	basketY := geom.BasketOffset

	cornerDist := geom.ThreePointCornerDist
	cornerX := cornerDist
	dx := basketX - cornerX
	dy2 := geom.ThreePointRadius*geom.ThreePointRadius - dx*dx
	arcStartY := basketY + math.Sqrt(dy2)

	// Left corner.
	DrawLine(img, m2p(cornerX, yCoord(0)), m2p(cornerX, yCoord(arcStartY)), lineW, lineCol)
	// Right corner.
	rightCornerX := courtW - cornerDist
	DrawLine(img, m2p(rightCornerX, yCoord(0)), m2p(rightCornerX, yCoord(arcStartY)), lineW, lineCol)

	// Arc from left corner to right corner.
	arcCenter := m2p(basketX, yCoord(basketY))
	arcR := float32(vp.MeterToPixel(geom.ThreePointRadius, geom, courtType))

	startAngleRaw := math.Atan2(arcStartY-basketY, cornerX-basketX)
	endAngleRaw := math.Atan2(arcStartY-basketY, rightCornerX-basketX)

	if mirrored {
		DrawArc(img, arcCenter, arcR, -startAngleRaw, -endAngleRaw, lineW, lineCol)
	} else {
		DrawArc(img, arcCenter, arcR, startAngleRaw, endAngleRaw, lineW, lineCol)
	}
}
