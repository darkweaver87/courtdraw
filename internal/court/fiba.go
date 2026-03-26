package court

import (
	"image"
	"image/color"
	"math"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// FIBAGeometry returns FIBA court dimensions in meters.
//
// Sources:
//   - FFBB Annuaire Officiel 2020-2021, Annexe 1 (terrain 28m×15m), p.226
//   - FFBB Annuaire Officiel 2020-2021, Annexe 3 (zone restrictive), p.228
//   - FFBB Annuaire Officiel 2020-2021, Annexe 4 (zone 3 points, 28m×15m), p.229
//   - https://files.ffbb.com/sites/default/files/reglement_salles_et_terrains_2020-2021_vdef.pdf
func FIBAGeometry() *CourtGeometry {
	return &CourtGeometry{
		Width:                15.0,   // Annexe 1: 1500 cm
		Length:               28.0,   // Annexe 1: 2800 cm
		BasketOffset:         1.575,  // Annexe 3: 120 cm (panneau) + 37.5 cm (projection)
		LaneWidth:            4.90,   // Annexe 3: 490 cm
		LaneLength:           5.80,   // Annexe 3: 580 cm
		ThreePointRadius:     6.75,   // Annexe 4: 675 cm
		ThreePointCornerDist: 0.90,   // Annexe 4: 90 cm
		FreeThrowRadius:      1.80,   // Annexe 3: 180 cm
		CenterCircleRadius:   1.80,   // Annexe 1: 180 cm (rayon)
		RestrictedAreaRadius: 1.25,   // Annexe 3: 125 cm
		BackboardWidth:       1.80,   // FIBA rules: 180 cm
		RimDiameter:          0.45,   // FIBA rules: 45 cm inner diameter
		LineWidth:            0.05,   // FIBA rules: 5 cm
	}
}

// DrawFIBACourt draws a FIBA basketball court on the given image.
func DrawFIBACourt(img *image.RGBA, courtType model.CourtType, vp *Viewport, geom *CourtGeometry) {
	drawCourt(img, courtType, vp, geom)
}

// drawCourt draws the court for any standard using the geometry.
func drawCourt(img *image.RGBA, courtType model.CourtType, vp *Viewport, geom *CourtGeometry) {
	lineCol := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	bgCol := color.NRGBA{R: 0xc8, G: 0x96, B: 0x64, A: 0xff}    // fallback if texture not loaded
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
	if !vp.HideApron {
		apronTL := m2p(-ApronMeters, courtH+ApronMeters)
		apronBR := m2p(courtW+ApronMeters, -ApronMeters)
		DrawRectFill(img, apronTL, apronBR, apronCol)
	}

	// Court floor — flat color slightly oversized as safety under lines, texture on exact court area.
	pad := lineW + 1
	paddedTL := Pt(m2p(0, courtH).X-pad, m2p(0, courtH).Y-pad)
	paddedBR := Pt(m2p(courtW, 0).X+pad, m2p(courtW, 0).Y+pad)
	DrawRectFill(img, paddedTL, paddedBR, bgCol)
	topLeft := m2p(0, courtH)
	botRight := m2p(courtW, 0)
	if woodTile := WoodFloorTexture(); woodTile != nil {
		TileRectScaled(img, topLeft, botRight, woodTile, vp, geom, courtType)
	}

	isFullCourt := courtType == model.FullCourt

	// Near basket (bottom).
	drawBasketEnd(img, vp, geom, courtType, lineW, lineCol, courtW, courtH, false)

	if isFullCourt {
		hlY := courtH / 2

		// Center circle — same fill as the paint.
		paintCol := color.NRGBA{R: 0x0c, G: 0x8f, B: 0xbf, A: 0xaa}
		centerPt := m2p(courtW/2, hlY)
		centerR := float32(vp.MeterToPixel(geom.CenterCircleRadius, geom, courtType))
		DrawCircleFill(img, centerPt, centerR, paintCol)
		DrawCircleOutline(img, centerPt, centerR, lineW, lineCol)

		// Half-court line drawn after circle fill so it stays on top.
		DrawLine(img, m2p(0, hlY), m2p(courtW, hlY), lineW, lineCol)

		// Far basket (top) — mirrored.
		drawBasketEnd(img, vp, geom, courtType, lineW, lineCol, courtW, courtH, true)
	}

	// Court outline redrawn last so fills don't cover it.
	DrawRect(img, topLeft, botRight, lineW, lineCol)
}

// courtEnd holds shared context for drawing one end of the court.
type courtEnd struct {
	img      *image.RGBA
	vp       *Viewport
	geom     *CourtGeometry
	ct       model.CourtType
	lineW    float32
	lineCol  color.NRGBA
	courtW   float64
	courtH   float64
	mirrored bool
}

func (ce *courtEnd) yCoord(y float64) float64 {
	if ce.mirrored {
		return ce.courtH - y
	}
	return y
}

func (ce *courtEnd) m2p(mx, my float64) Point {
	return ce.vp.RelToPixel(model.Position{mx / ce.courtW, my / ce.courtH})
}

// drawBasketEnd draws all elements for one end of the court.
func drawBasketEnd(img *image.RGBA, vp *Viewport, geom *CourtGeometry, courtType model.CourtType, lineW float32, lineCol color.NRGBA, courtW, courtH float64, mirrored bool) {
	ce := &courtEnd{img, vp, geom, courtType, lineW, lineCol, courtW, courtH, mirrored}
	ce.drawPaint()
	ce.drawLaneSideBands()
	ce.drawReboundSlots()
	ce.drawFreeThrowCircle()
	ce.draw3ptLine()
	ce.drawRestrictedArea()
	ce.drawBackboard()
	ce.drawRim()
}

func (ce *courtEnd) drawPaint() {
	laneLeft := (ce.courtW - ce.geom.LaneWidth) / 2
	laneRight := (ce.courtW + ce.geom.LaneWidth) / 2
	laneTop := ce.yCoord(ce.geom.LaneLength)
	paintCol := color.NRGBA{R: 0x0c, G: 0x8f, B: 0xbf, A: 0xaa}
	DrawRectFill(ce.img, ce.m2p(laneLeft, laneTop), ce.m2p(laneRight, ce.yCoord(0)), paintCol)
	DrawRect(ce.img, ce.m2p(laneLeft, laneTop), ce.m2p(laneRight, ce.yCoord(0)), ce.lineW, ce.lineCol)
}

func (ce *courtEnd) drawLaneSideBands() {
	laneLeft := (ce.courtW - ce.geom.LaneWidth) / 2
	laneRight := (ce.courtW + ce.geom.LaneWidth) / 2
	laneTop := ce.yCoord(ce.geom.LaneLength)
	bandWidth := 0.40
	bandCol := color.NRGBA{R: 0x1a, G: 0x3c, B: 0x6e, A: 0xcc}
	DrawRectFill(ce.img, ce.m2p(laneLeft-bandWidth, laneTop), ce.m2p(laneLeft, ce.yCoord(0)), bandCol)
	DrawRectFill(ce.img, ce.m2p(laneRight, laneTop), ce.m2p(laneRight+bandWidth, ce.yCoord(0)), bandCol)
}

func (ce *courtEnd) drawReboundSlots() {
	laneLeft := (ce.courtW - ce.geom.LaneWidth) / 2
	laneRight := (ce.courtW + ce.geom.LaneWidth) / 2
	tickLen := 0.10 // 10cm tick marks (Annexe 3, p.228)

	// FIBA rebound slot positions from baseline (Annexe 3, p.228):
	// 175, +85=260, +40=300, +85=385, +85=470 (cm).
	tickPositions := []float64{1.75, 2.60, 3.00, 3.85, 4.70}
	for _, slotY := range tickPositions {
		// Left side.
		DrawLine(ce.img, ce.m2p(laneLeft-tickLen, ce.yCoord(slotY)), ce.m2p(laneLeft, ce.yCoord(slotY)), ce.lineW, ce.lineCol)
		// Right side.
		DrawLine(ce.img, ce.m2p(laneRight, ce.yCoord(slotY)), ce.m2p(laneRight+tickLen, ce.yCoord(slotY)), ce.lineW, ce.lineCol)
	}

	// Neutral zone block (Annexe 3, p.228): 0.40m deep × 0.10m wide between 2.60 and 3.00.
	nzY := 2.60
	nzHeight := 0.40
	nzWidth := 0.10
	DrawRectFill(ce.img, ce.m2p(laneLeft-nzWidth, ce.yCoord(nzY+nzHeight)), ce.m2p(laneLeft, ce.yCoord(nzY)), ce.lineCol)
	DrawRectFill(ce.img, ce.m2p(laneRight, ce.yCoord(nzY+nzHeight)), ce.m2p(laneRight+nzWidth, ce.yCoord(nzY)), ce.lineCol)
}

func (ce *courtEnd) drawFreeThrowCircle() {
	basketX := ce.courtW / 2
	laneTop := ce.yCoord(ce.geom.LaneLength)
	ftCenter := ce.m2p(basketX, laneTop)
	ftR := float32(ce.vp.MeterToPixel(ce.geom.FreeThrowRadius, ce.geom, ce.ct))

	// Solid half toward basket.
	if ce.mirrored {
		DrawArc(ce.img, ftCenter, ftR, math.Pi, 2*math.Pi, ce.lineW, ce.lineCol)
	} else {
		DrawArc(ce.img, ftCenter, ftR, 0, math.Pi, ce.lineW, ce.lineCol)
	}

	// Dashed half away from basket.
	segments := 24
	arcStart := math.Pi
	arcEnd := 2 * math.Pi
	if ce.mirrored {
		arcStart = 0
		arcEnd = math.Pi
	}
	span := arcEnd - arcStart
	for i := range segments {
		if i%2 != 0 {
			continue
		}
		a1 := arcStart + span*float64(i)/float64(segments)
		a2 := arcStart + span*float64(i+1)/float64(segments)
		DrawArc(ce.img, ftCenter, ftR, a1, a2, ce.lineW, ce.lineCol)
	}
}

func (ce *courtEnd) draw3ptLine() {
	draw3ptLineImpl(ce.img, ce.vp, ce.geom, ce.ct, ce.lineW, ce.lineCol, ce.courtW, ce.courtH, ce.mirrored)
}

func (ce *courtEnd) drawRestrictedArea() {
	basketX := ce.courtW / 2
	basketY := ce.yCoord(ce.geom.BasketOffset)
	raCenter := ce.m2p(basketX, basketY)
	raR := float32(ce.vp.MeterToPixel(ce.geom.RestrictedAreaRadius, ce.geom, ce.ct))
	if ce.mirrored {
		DrawArc(ce.img, raCenter, raR, math.Pi, 2*math.Pi, ce.lineW, ce.lineCol)
	} else {
		DrawArc(ce.img, raCenter, raR, 0, math.Pi, ce.lineW, ce.lineCol)
	}
}

func (ce *courtEnd) drawBackboard() {
	basketX := ce.courtW / 2
	bbHalf := ce.geom.BackboardWidth / 2
	bbY := ce.yCoord(ce.geom.BasketOffset - 0.15)
	DrawLine(ce.img, ce.m2p(basketX-bbHalf, bbY), ce.m2p(basketX+bbHalf, bbY), ce.lineW*1.5, ce.lineCol)
}

func (ce *courtEnd) drawRim() {
	basketX := ce.courtW / 2
	basketY := ce.yCoord(ce.geom.BasketOffset)
	rimCenter := ce.m2p(basketX, basketY)
	rimR := float32(ce.vp.MeterToPixel(ce.geom.RimDiameter/2, ce.geom, ce.ct))
	DrawCircleOutline(ce.img, rimCenter, rimR, ce.lineW, color.NRGBA{R: 0xff, G: 0x66, B: 0x00, A: 0xff})
}

// draw3ptLineImpl draws the three-point line for one end.
func draw3ptLineImpl(img *image.RGBA, vp *Viewport, geom *CourtGeometry, courtType model.CourtType, lineW float32, lineCol color.NRGBA, courtW, courtH float64, mirrored bool) {
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
