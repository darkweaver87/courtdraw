package pdf

import (
	"math"

	"github.com/go-pdf/fpdf"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// courtRenderer draws a court diagram into a PDF region.
type courtRenderer struct {
	pdf *fpdf.Fpdf
	// Diagram bounding box in mm.
	x, y, w, h float64
}

// drawCourtDiagram renders the court background, markings, and a single sequence's elements.
func drawCourtDiagram(pdf *fpdf.Fpdf, x, y, w, h float64, ex *model.Exercise, seqIdx int) {
	cr := &courtRenderer{pdf: pdf, x: x, y: y, w: w, h: h}
	cr.drawBackground()
	cr.drawMarkings(ex.CourtType)

	if seqIdx >= 0 && seqIdx < len(ex.Sequences) {
		seq := &ex.Sequences[seqIdx]
		cr.drawAccessories(seq)
		cr.drawActions(seq)
		cr.drawPlayers(seq)
	}
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
	cr.pdf.SetLineWidth(lineWidthThick)

	// Court outline.
	cr.pdf.Rect(cr.x, cr.y, cr.w, cr.h, "D")

	// Half-court line (for full court).
	if courtType == model.FullCourt {
		midY := cr.y + cr.h/2
		cr.pdf.Line(cr.x, midY, cr.x+cr.w, midY)
		// Center circle.
		cr.pdf.Circle(cr.x+cr.w/2, midY, cr.w*0.12, "D")
	}

	// Basket end (bottom of diagram = top of court in model).
	cr.drawBasketEnd(false)
	if courtType == model.FullCourt {
		cr.drawBasketEnd(true)
	}
}

func (cr *courtRenderer) drawBasketEnd(mirrored bool) {
	centerX := cr.x + cr.w/2

	// Paint/lane: ~32% of court width, ~20% of half-length.
	laneW := cr.w * 0.32
	laneH := cr.h * 0.20
	if mirrored {
		// Paint at top.
		cr.pdf.Rect(centerX-laneW/2, cr.y, laneW, laneH, "D")
	} else {
		// Paint at bottom.
		cr.pdf.Rect(centerX-laneW/2, cr.y+cr.h-laneH, laneW, laneH, "D")
	}

	// Free throw circle.
	ftRadius := laneW / 2
	if mirrored {
		cr.pdf.Circle(centerX, cr.y+laneH, ftRadius, "D")
	} else {
		cr.pdf.Circle(centerX, cr.y+cr.h-laneH, ftRadius, "D")
	}

	// Three-point arc (simplified as an arc).
	arcRadius := cr.w * 0.44
	if mirrored {
		basketY := cr.y + cr.h*0.055
		cr.drawArc(centerX, basketY, arcRadius, 200, 340)
	} else {
		basketY := cr.y + cr.h - cr.h*0.055
		cr.drawArc(centerX, basketY, arcRadius, 20, 160)
	}

	// Basket (small circle + rectangle).
	rimR := cr.w * 0.02
	if mirrored {
		basketY := cr.y + cr.h*0.055
		cr.pdf.Circle(centerX, basketY, rimR, "D")
		bbW := cr.w * 0.12
		cr.pdf.Line(centerX-bbW/2, cr.y+cr.h*0.03, centerX+bbW/2, cr.y+cr.h*0.03)
	} else {
		basketY := cr.y + cr.h - cr.h*0.055
		cr.pdf.Circle(centerX, basketY, rimR, "D")
		bbW := cr.w * 0.12
		cr.pdf.Line(centerX-bbW/2, cr.y+cr.h-cr.h*0.03, centerX+bbW/2, cr.y+cr.h-cr.h*0.03)
	}
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
	for i := range seq.Players {
		p := &seq.Players[i]
		px, py := cr.relToMM(p.Position)
		r := 2.5 // player radius in mm

		// Role color.
		col := roleColorPDF(p.Role)
		cr.pdf.SetFillColor(col[0], col[1], col[2])
		cr.pdf.Circle(px, py, r, "F")

		// White outline.
		cr.pdf.SetDrawColor(255, 255, 255)
		cr.pdf.SetLineWidth(0.3)
		cr.pdf.Circle(px, py, r, "D")

		// Label.
		label := p.Label
		if label == "" {
			label = model.RoleLabel(p.Role)
		}
		cr.pdf.SetFont("Helvetica", "B", 5)
		cr.pdf.SetTextColor(255, 255, 255)
		strW := cr.pdf.GetStringWidth(label)
		cr.pdf.Text(px-strW/2, py+1.5, label)

		// Ball indicator.
		if seq.BallCarrier != "" && p.ID == seq.BallCarrier {
			ballX := px + 1.8
			ballY := py + 1.8
			cr.pdf.SetFillColor(244, 162, 97) // #f4a261 orange
			cr.pdf.Circle(ballX, ballY, 0.8, "F")
			cr.pdf.SetDrawColor(0, 0, 0)
			cr.pdf.SetLineWidth(0.15)
			cr.pdf.Circle(ballX, ballY, 0.8, "D")
		}
	}
}

func (cr *courtRenderer) drawAccessories(seq *model.Sequence) {
	for i := range seq.Accessories {
		acc := &seq.Accessories[i]
		ax, ay := cr.relToMM(acc.Position)

		switch acc.Type {
		case model.AccessoryCone:
			// Small triangle.
			s := 1.5
			cr.pdf.SetFillColor(255, 165, 0) // orange
			cr.pdf.MoveTo(ax, ay-s)
			cr.pdf.LineTo(ax-s*0.7, ay+s*0.5)
			cr.pdf.LineTo(ax+s*0.7, ay+s*0.5)
			cr.pdf.ClosePath()
			cr.pdf.DrawPath("F")
		case model.AccessoryAgilityLadder:
			// Small rectangle.
			w, h := 2.0, 5.0
			cr.pdf.SetFillColor(255, 215, 0) // gold
			cr.pdf.Rect(ax-w/2, ay-h/2, w, h, "F")
		case model.AccessoryChair:
			// Small L-shape.
			cr.pdf.SetDrawColor(128, 128, 128)
			cr.pdf.SetLineWidth(0.5)
			cr.pdf.Line(ax, ay-2, ax, ay+1)
			cr.pdf.Line(ax, ay+1, ax+1.5, ay+1)
		}
	}
}

func (cr *courtRenderer) drawActions(seq *model.Sequence) {
	for i := range seq.Actions {
		act := &seq.Actions[i]
		fromX, fromY := cr.resolveActionRef(act.From, seq.Players)
		toX, toY := cr.resolveActionRef(act.To, seq.Players)

		col := actionColorPDF(act.Type)
		cr.pdf.SetDrawColor(col[0], col[1], col[2])
		cr.pdf.SetLineWidth(0.4)

		switch act.Type {
		case model.ActionPass:
			// Dashed line.
			cr.drawDashed(fromX, fromY, toX, toY, 1.5, 1.0)
			cr.drawArrowHead(fromX, fromY, toX, toY, col)
		case model.ActionDribble:
			// Zigzag.
			cr.drawZigzag(fromX, fromY, toX, toY)
			cr.drawArrowHead(fromX, fromY, toX, toY, col)
		case model.ActionScreen:
			// Thick bar.
			cr.pdf.SetLineWidth(1.0)
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
	amp := 1.0 // mm amplitude
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

	size := 1.5 // arrow head size in mm
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
