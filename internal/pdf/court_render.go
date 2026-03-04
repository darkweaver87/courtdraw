package pdf

import (
	"math"

	"github.com/go-pdf/fpdf"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
)

// courtRenderer draws a court diagram into a PDF region.
type courtRenderer struct {
	pdf *fpdf.Fpdf
	// Diagram bounding box in mm.
	x, y, w, h float64
	// Element scale factor (1.0 = reference 60mm diagram, smaller for tiny diagrams).
	es float64
}

// drawCourtDiagram renders the court background, markings, and a single sequence's elements.
func drawCourtDiagram(pdf *fpdf.Fpdf, x, y, w, h float64, ex *model.Exercise, seqIdx int) {
	// Scale decorative elements (players, lines, arrows) based on diagram size.
	ref := math.Min(w, h)
	es := ref / 60.0
	if es > 1.0 {
		es = 1.0
	}
	if es < 0.4 {
		es = 0.4
	}
	cr := &courtRenderer{pdf: pdf, x: x, y: y, w: w, h: h, es: es}
	cr.drawBackground()
	cr.drawMarkings(ex.CourtType)

	if seqIdx >= 0 && seqIdx < len(ex.Sequences) {
		seq := &ex.Sequences[seqIdx]
		cr.drawAccessories(seq)
		cr.drawActions(seq)
		cr.drawPlayers(seq)
	}
}

// roleLabelI18n returns the short translated label for a player role.
func roleLabelI18n(role model.PlayerRole) string {
	key := "role." + string(role)
	label := i18n.T(key)
	if label == key {
		return model.RoleLabel(role)
	}
	return label
}

func (cr *courtRenderer) drawBackground() {
	cr.pdf.SetFillColor(colorCourtBg[0], colorCourtBg[1], colorCourtBg[2])
	cr.pdf.Rect(cr.x, cr.y, cr.w, cr.h, "F")
}

// relToMM converts relative [0,1] position to mm coordinates within the diagram.
// Y is flipped: model [0,0] = bottom-left, PDF [0,0] = top-left.
func (cr *courtRenderer) relToMM(pos model.Position) (float64, float64) {
	mx := cr.x + pos[0]*cr.w
	my := cr.y + (1.0-pos[1])*cr.h
	return mx, my
}

func (cr *courtRenderer) drawMarkings(courtType model.CourtType) {
	cr.pdf.SetDrawColor(colorCourtLine[0], colorCourtLine[1], colorCourtLine[2])
	cr.pdf.SetLineWidth(lineWidthThick * cr.es)

	// Court outline.
	cr.pdf.Rect(cr.x, cr.y, cr.w, cr.h, "D")

	// Half-court line (for full court).
	if courtType == model.FullCourt {
		midY := cr.y + cr.h/2
		cr.pdf.Line(cr.x, midY, cr.x+cr.w, midY)
		// Center circle (FIBA: 1.80m radius on 15m width).
		cr.pdf.Circle(cr.x+cr.w/2, midY, cr.w*(1.80/15.0), "D")
	}

	// Basket end (bottom of diagram = top of court in model).
	cr.drawBasketEnd(false, courtType)
	if courtType == model.FullCourt {
		cr.drawBasketEnd(true, courtType)
	}
}

func (cr *courtRenderer) drawBasketEnd(mirrored bool, courtType model.CourtType) {
	centerX := cr.x + cr.w/2

	// Real FIBA court dimensions → diagram mm.
	// Court length depends on type: 14m (half) or 28m (full).
	courtLen := 28.0
	if courtType != model.FullCourt {
		courtLen = 14.0
	}
	const courtWid = 15.0

	xMM := func(m float64) float64 { return cr.w * (m / courtWid) }
	yMM := func(m float64) float64 { return cr.h * (m / courtLen) }

	laneW := xMM(4.90)
	laneH := yMM(5.80)
	basketOff := yMM(1.575)
	bbOff := yMM(1.20)
	rimR := xMM(0.225)
	bbW := xMM(1.80)
	tpR := xMM(6.75)
	cornerDist := xMM(0.90)
	ftR := xMM(1.80)

	// fromBaseline converts a distance from the baseline to a PDF Y coordinate.
	fromBaseline := func(dist float64) float64 {
		if mirrored {
			return cr.y + dist
		}
		return cr.y + cr.h - dist
	}

	basketY := fromBaseline(basketOff)

	// Paint/lane.
	if mirrored {
		cr.pdf.Rect(centerX-laneW/2, cr.y, laneW, laneH, "D")
	} else {
		cr.pdf.Rect(centerX-laneW/2, cr.y+cr.h-laneH, laneW, laneH, "D")
	}

	// Free throw circle at end of lane.
	cr.pdf.Circle(centerX, fromBaseline(laneH), ftR, "D")

	// Backboard.
	bbY := fromBaseline(bbOff)
	cr.pdf.Line(centerX-bbW/2, bbY, centerX+bbW/2, bbY)

	// Rim.
	cr.pdf.Circle(centerX, basketY, rimR, "D")

	// Three-point line: corner straight lines + arc.
	cornerXL := cr.x + cornerDist
	cornerXR := cr.x + cr.w - cornerDist
	baselineY := fromBaseline(0)

	// Where the arc meets the corner lines.
	dx := centerX - cornerXL
	dy2 := tpR*tpR - dx*dx
	if dy2 < 0 {
		dy2 = 0
	}
	arcMeet := math.Sqrt(dy2)

	// Arc meet point Y — the arc extends into the court (away from baseline).
	var arcMeetY float64
	if mirrored {
		arcMeetY = basketY + arcMeet
	} else {
		arcMeetY = basketY - arcMeet
	}

	// Corner straight lines from baseline to arc meet point.
	cr.pdf.Line(cornerXL, baselineY, cornerXL, arcMeetY)
	cr.pdf.Line(cornerXR, baselineY, cornerXR, arcMeetY)

	// Three-point arc from left meet point to right meet point.
	leftDy := arcMeetY - basketY
	rightDy := leftDy
	startAngle := math.Atan2(leftDy, cornerXL-centerX) * 180 / math.Pi
	endAngle := math.Atan2(rightDy, cornerXR-centerX) * 180 / math.Pi
	cr.drawArc(centerX, basketY, tpR, startAngle, endAngle)
}

func (cr *courtRenderer) drawArc(cx, cy, r, startDeg, endDeg float64) {
	steps := 30
	startRad := startDeg * math.Pi / 180
	endRad := endDeg * math.Pi / 180
	step := (endRad - startRad) / float64(steps)

	for i := 0; i < steps; i++ {
		a1 := startRad + float64(i)*step
		a2 := a1 + step
		x1 := cx + r*math.Cos(a1)
		y1 := cy + r*math.Sin(a1)
		x2 := cx + r*math.Cos(a2)
		y2 := cy + r*math.Sin(a2)
		cr.pdf.Line(x1, y1, x2, y2)
	}
}

func (cr *courtRenderer) drawPlayers(seq *model.Sequence) {
	s := cr.es
	for i := range seq.Players {
		p := &seq.Players[i]
		px, py := cr.relToMM(p.Position)
		r := 2.5 * s

		// Queue circles behind the main player.
		if p.Type == "queue" && p.Count > 1 {
			col := roleColorPDF(p.Role)
			qCount := p.Count
			if qCount > 4 {
				qCount = 4
			}
			for qi := qCount - 1; qi >= 1; qi-- {
				qy := py + float64(qi)*3.5*s
				cr.pdf.SetFillColor(col[0], col[1], col[2])
				cr.pdf.SetDrawColor(255, 255, 255)
				cr.pdf.SetLineWidth(0.2 * s)
				cr.pdf.Circle(px, qy, 1.5*s, "FD")
			}
		}

		// Role color.
		col := roleColorPDF(p.Role)
		cr.pdf.SetFillColor(col[0], col[1], col[2])
		cr.pdf.Circle(px, py, r, "F")

		// White outline.
		cr.pdf.SetDrawColor(255, 255, 255)
		cr.pdf.SetLineWidth(0.3 * s)
		cr.pdf.Circle(px, py, r, "D")

		// Label — translate default role labels.
		label := p.Label
		if label == "" || label == model.RoleLabel(p.Role) {
			label = roleLabelI18n(p.Role)
		}
		fontSize := 5.0 * s
		if fontSize < 3.0 {
			fontSize = 3.0
		}
		cr.pdf.SetFont("Helvetica", "B", fontSize)
		cr.pdf.SetTextColor(255, 255, 255)
		strW := cr.pdf.GetStringWidth(label)
		cr.pdf.Text(px-strW/2, py+1.2*s, label)

		// Ball indicator.
		if seq.BallCarrier != "" && p.ID == seq.BallCarrier {
			ballX := px + 1.8*s
			ballY := py + 1.8*s
			cr.pdf.SetFillColor(244, 162, 97) // #f4a261 orange
			cr.pdf.Circle(ballX, ballY, 0.8*s, "F")
			cr.pdf.SetDrawColor(0, 0, 0)
			cr.pdf.SetLineWidth(0.15 * s)
			cr.pdf.Circle(ballX, ballY, 0.8*s, "D")
		}

		// Callout label above player.
		if p.Callout != "" {
			calloutLabel := i18n.T("callout." + string(p.Callout))
			calloutSize := 4.0 * s
			if calloutSize < 3.0 {
				calloutSize = 3.0
			}
			cr.pdf.SetFont("Helvetica", "B", calloutSize)
			cr.pdf.SetTextColor(60, 60, 60)
			strW := cr.pdf.GetStringWidth(calloutLabel)
			cr.pdf.Text(px-strW/2, py-r-1.0*s, calloutLabel)
		}
	}
}

func (cr *courtRenderer) drawAccessories(seq *model.Sequence) {
	s := cr.es
	for i := range seq.Accessories {
		acc := &seq.Accessories[i]
		ax, ay := cr.relToMM(acc.Position)

		switch acc.Type {
		case model.AccessoryCone:
			// Small triangle.
			cs := 1.5 * s
			cr.pdf.SetFillColor(255, 165, 0) // orange
			cr.pdf.MoveTo(ax, ay-cs)
			cr.pdf.LineTo(ax-cs*0.7, ay+cs*0.5)
			cr.pdf.LineTo(ax+cs*0.7, ay+cs*0.5)
			cr.pdf.ClosePath()
			cr.pdf.DrawPath("F")
		case model.AccessoryAgilityLadder:
			// Small rectangle.
			w, h := 2.0*s, 5.0*s
			cr.pdf.SetFillColor(255, 215, 0) // gold
			cr.pdf.Rect(ax-w/2, ay-h/2, w, h, "F")
		case model.AccessoryChair:
			// Small L-shape.
			cr.pdf.SetDrawColor(128, 128, 128)
			cr.pdf.SetLineWidth(0.5 * s)
			cr.pdf.Line(ax, ay-2*s, ax, ay+1*s)
			cr.pdf.Line(ax, ay+1*s, ax+1.5*s, ay+1*s)
		}
	}
}

func (cr *courtRenderer) drawActions(seq *model.Sequence) {
	s := cr.es
	for i := range seq.Actions {
		act := &seq.Actions[i]
		fromX, fromY := cr.resolveActionRef(act.From, seq.Players)
		toX, toY := cr.resolveActionRef(act.To, seq.Players)

		col := actionColorPDF(act.Type)
		cr.pdf.SetDrawColor(col[0], col[1], col[2])
		cr.pdf.SetLineWidth(0.4 * s)

		switch act.Type {
		case model.ActionPass:
			// Dashed line.
			cr.drawDashed(fromX, fromY, toX, toY, 1.5*s, 1.0*s)
			cr.drawArrowHead(fromX, fromY, toX, toY, col)
		case model.ActionDribble:
			// Zigzag.
			cr.drawZigzag(fromX, fromY, toX, toY)
			cr.drawArrowHead(fromX, fromY, toX, toY, col)
		case model.ActionScreen:
			// Thick bar.
			cr.pdf.SetLineWidth(1.0 * s)
			cr.pdf.Line(fromX, fromY, toX, toY)
		default:
			// Solid line + arrow.
			cr.pdf.Line(fromX, fromY, toX, toY)
			cr.drawArrowHead(fromX, fromY, toX, toY, col)
		}
	}
}

func (cr *courtRenderer) resolveActionRef(ref model.ActionRef, players []model.Player) (float64, float64) {
	if ref.IsPlayer {
		for i := range players {
			if players[i].ID == ref.PlayerID {
				return cr.relToMM(players[i].Position)
			}
		}
		return cr.relToMM(model.Position{0.5, 0.5})
	}
	return cr.relToMM(ref.Position)
}

func (cr *courtRenderer) drawDashed(x1, y1, x2, y2, dashLen, gapLen float64) {
	dx := x2 - x1
	dy := y2 - y1
	length := math.Sqrt(dx*dx + dy*dy)
	if length == 0 {
		return
	}
	ux := dx / length
	uy := dy / length

	pos := 0.0
	drawing := true
	for pos < length {
		segLen := dashLen
		if !drawing {
			segLen = gapLen
		}
		end := pos + segLen
		if end > length {
			end = length
		}
		if drawing {
			cr.pdf.Line(x1+ux*pos, y1+uy*pos, x1+ux*end, y1+uy*end)
		}
		pos = end
		drawing = !drawing
	}
}

func (cr *courtRenderer) drawZigzag(x1, y1, x2, y2 float64) {
	dx := x2 - x1
	dy := y2 - y1
	length := math.Sqrt(dx*dx + dy*dy)
	if length == 0 {
		return
	}
	segments := 6
	amp := 1.0 * cr.es // mm amplitude
	ux := dx / length
	uy := dy / length
	// perpendicular
	px := -uy
	py := ux

	prevX, prevY := x1, y1
	for i := 1; i <= segments; i++ {
		t := float64(i) / float64(segments)
		mx := x1 + dx*t
		my := y1 + dy*t
		side := 1.0
		if i%2 == 0 {
			side = -1.0
		}
		if i == segments {
			side = 0 // end at target
		}
		zx := mx + px*amp*side
		zy := my + py*amp*side
		cr.pdf.Line(prevX, prevY, zx, zy)
		prevX, prevY = zx, zy
	}
}

func (cr *courtRenderer) drawArrowHead(fromX, fromY, toX, toY float64, col [3]int) {
	dx := toX - fromX
	dy := toY - fromY
	length := math.Sqrt(dx*dx + dy*dy)
	if length < 1 {
		return
	}
	ux := dx / length
	uy := dy / length

	size := 1.5 * cr.es // arrow head size in mm
	// Arrow tip is at (toX, toY), two sides.
	lx := toX - ux*size + uy*size*0.5
	ly := toY - uy*size - ux*size*0.5
	rx := toX - ux*size - uy*size*0.5
	ry := toY - uy*size + ux*size*0.5

	cr.pdf.SetFillColor(col[0], col[1], col[2])
	cr.pdf.MoveTo(toX, toY)
	cr.pdf.LineTo(lx, ly)
	cr.pdf.LineTo(rx, ry)
	cr.pdf.ClosePath()
	cr.pdf.DrawPath("F")
}

func roleColorPDF(role model.PlayerRole) [3]int {
	switch role {
	case model.RoleDefender:
		return colorDefense
	case model.RoleCoach:
		return colorCoach
	case model.RoleAttacker, model.RolePointGuard, model.RoleShootingGuard,
		model.RoleSmallForward, model.RolePowerForward, model.RoleCenter:
		return colorAttack
	default:
		return colorNeutral
	}
}

func actionColorPDF(at model.ActionType) [3]int {
	switch at {
	case model.ActionPass, model.ActionDribble:
		return colorCoach
	case model.ActionSprint, model.ActionCut, model.ActionShotLayup,
		model.ActionShotPushup, model.ActionShotJump, model.ActionReverse:
		return colorAttack
	case model.ActionCloseOut, model.ActionContest:
		return colorCloseOut
	case model.ActionScreen:
		return colorScreen
	default:
		return colorWhite
	}
}
