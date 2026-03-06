package pdf

import (
	"math"

	"github.com/go-pdf/fpdf"

	"github.com/darkweaver87/courtdraw/internal/court"
	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
)

// pdfPlayerBodyWidth is the unscaled player body diameter in mm (2 × BodyRX equivalent).
const pdfPlayerBodyWidth = 5.0

// courtRenderer draws a court diagram into a PDF region.
type courtRenderer struct {
	pdf *fpdf.Fpdf
	// Diagram bounding box in mm.
	x, y, w, h float64
	// Element scale factor derived from court physical dimensions.
	es float64
}

// su converts a screen-space constant (from draw_players.go / draw_accessories.go)
// to PDF mm, preserving the same physical proportions via ElementScaleForCourt.
func (cr *courtRenderer) su(c float64) float64 {
	return c * cr.es * pdfPlayerBodyWidth / (2.0 * float64(court.BodyRX))
}

// drawCourtDiagram renders the court background, markings, and a single sequence's elements.
func drawCourtDiagram(pdf *fpdf.Fpdf, x, y, w, h float64, ex *model.Exercise, seqIdx int) {
	geom := court.FIBAGeometry()
	if ex.CourtStandard == model.NBA {
		geom = court.NBAGeometry()
	}

	// Scale elements so a player body represents 0.45m (shoulder width)
	// on the court, using the same function as the on-screen renderer.
	mmPerMeter := w / geom.Width
	es := court.ElementScaleForCourt(mmPerMeter, pdfPlayerBodyWidth)
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

	// Three-point line.
	cornerXL := cr.x + cornerDist
	cornerXR := cr.x + cr.w - cornerDist
	baselineY := fromBaseline(0)

	dx := centerX - cornerXL
	dy2 := tpR*tpR - dx*dx
	if dy2 < 0 {
		dy2 = 0
	}
	arcMeet := math.Sqrt(dy2)

	var arcMeetY float64
	if mirrored {
		arcMeetY = basketY + arcMeet
	} else {
		arcMeetY = basketY - arcMeet
	}

	cr.pdf.Line(cornerXL, baselineY, cornerXL, arcMeetY)
	cr.pdf.Line(cornerXR, baselineY, cornerXR, arcMeetY)

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

// --- Players (B2 style: head + body ellipse + arms) ---

func (cr *courtRenderer) drawPlayers(seq *model.Sequence) {
	for i := range seq.Players {
		p := &seq.Players[i]
		px, py := cr.relToMM(p.Position)

		rad := p.Rotation * math.Pi / 180
		sinR := math.Sin(rad)
		cosR := math.Cos(rad)

		// Queue circles behind the main player (clamped to court bounds).
		if p.Type == "queue" && p.Count > 1 {
			col := roleColorPDF(p.Role)
			qCount := p.Count
			if qCount > 4 {
				qCount = 4
			}
			qr := cr.su(float64(court.QueueRadius))
			qs := cr.su(float64(court.QueueSpacing))
			for qi := qCount - 1; qi >= 1; qi-- {
				off := float64(qi) * qs
				qx := px - sinR*off
				qy := py + cosR*off
				// Skip circles that fall outside the court diagram.
				if qx-qr < cr.x || qx+qr > cr.x+cr.w || qy-qr < cr.y || qy+qr > cr.y+cr.h {
					continue
				}
				cr.pdf.SetFillColor(col[0], col[1], col[2])
				cr.pdf.SetDrawColor(255, 255, 255)
				cr.pdf.SetLineWidth(cr.su(0.5))
				cr.pdf.Circle(qx, qy, qr, "FD")
			}
		}

		// B2 style body.
		col := roleColorPDF(p.Role)
		brx := cr.su(float64(court.BodyRX))
		bry := cr.su(float64(court.BodyRY))
		hr := cr.su(float64(court.HeadRadius))
		gap := cr.su(float64(court.HeadBodyGap))
		headDist := bry*0.5 + gap
		headX := px + sinR*headDist
		headY := py - cosR*headDist

		// Body ellipse (semi-transparent).
		cr.drawRotatedEllipseFill(px, py, brx, bry, p.Rotation, col, 0.45)

		// Head circle (full color).
		cr.pdf.SetFillColor(col[0], col[1], col[2])
		cr.pdf.Circle(headX, headY, hr, "F")

		// Default ball position (right side of body).
		ballPosX := px + cosR*brx
		ballPosY := py + sinR*brx

		// Role-specific extras.
		armLen := cr.su(float64(court.ArmLength))
		armW := cr.su(float64(court.ArmWidth))

		switch p.Role {
		case model.RoleAttacker, model.RolePointGuard, model.RoleShootingGuard,
			model.RoleSmallForward, model.RolePowerForward, model.RoleCenter:
			// Right arm: starts at right side of body, extends forward-right.
			raX := px + cosR*brx
			raY := py + sinR*brx
			reX := raX + sinR*armLen*0.7 + cosR*armLen*0.5
			reY := raY - cosR*armLen*0.7 + sinR*armLen*0.5
			cr.pdf.SetDrawColor(col[0], col[1], col[2])
			cr.pdf.SetLineWidth(armW)
			cr.pdf.Line(raX, raY, reX, reY)
			ballPosX = reX
			ballPosY = reY
		case model.RoleDefender:
			// Arms spread (\O/).
			cr.pdf.SetDrawColor(col[0], col[1], col[2])
			cr.pdf.SetLineWidth(armW)
			lsX := px - cosR*brx
			lsY := py - sinR*brx
			leX := lsX - cosR*armLen + sinR*armLen*0.4
			leY := lsY - sinR*armLen - cosR*armLen*0.4
			rsX := px + cosR*brx
			rsY := py + sinR*brx
			reX := rsX + cosR*armLen + sinR*armLen*0.4
			reY := rsY + sinR*armLen - cosR*armLen*0.4
			cr.pdf.Line(lsX, lsY, leX, leY)
			cr.pdf.Line(rsX, rsY, reX, reY)
			ballPosX = reX
			ballPosY = reY
		case model.RoleCoach:
			// Clipboard beside body.
			cbDist := brx + cr.su(5)
			cbCX := px + cosR*cbDist
			cbCY := py + sinR*cbDist
			cbW := cr.su(3)
			cbH := cr.su(5)
			cr.pdf.SetFillColor(144, 164, 174)
			cr.pdf.SetAlpha(0.7, "Normal")
			cr.pdf.Rect(cbCX-cbW, cbCY-cbH, cbW*2, cbH*2, "F")
			cr.pdf.SetAlpha(1.0, "Normal")
		}

		// Label centered on head.
		label := p.Label
		if label == "" || label == model.RoleLabel(p.Role) {
			label = roleLabelI18n(p.Role)
		}
		fontSize := hr * 1.4
		if fontSize < 2.5 {
			fontSize = 2.5
		}
		cr.pdf.SetFont("Helvetica", "B", fontSize)
		cr.pdf.SetTextColor(255, 255, 255)
		strW := cr.pdf.GetStringWidth(label)
		// fpdf Text() y = baseline (mm). fontSize is in points.
		// Cap-height in mm = fontSize × 0.3528 (pt→mm) × 0.72 (Helvetica cap ratio).
		// Center caps vertically: baseline = centerY + capHeightMM / 2.
		cr.pdf.Text(headX-strW/2, headY+fontSize*0.127, label)

		// Ball indicator.
		if seq.BallCarrier != "" && p.ID == seq.BallCarrier {
			bx := ballPosX + cr.su(float64(court.BallOffsetX))
			by := ballPosY + cr.su(float64(court.BallOffsetY))
			ballR := cr.su(float64(court.BallRadius))
			cr.pdf.SetFillColor(244, 162, 97)
			cr.pdf.Circle(bx, by, ballR, "F")
			cr.pdf.SetDrawColor(0, 0, 0)
			cr.pdf.SetLineWidth(cr.su(float64(court.BallOutlineWidth)))
			cr.pdf.Circle(bx, by, ballR, "D")
		}

		// Callout label above player.
		if p.Callout != "" {
			calloutLabel := i18n.T("callout." + string(p.Callout))
			calloutSize := cr.su(float64(court.HeadRadius)) * 1.0
			if calloutSize < 2.5 {
				calloutSize = 2.5
			}
			cr.pdf.SetFont("Helvetica", "B", calloutSize)
			cr.pdf.SetTextColor(60, 60, 60)
			cstrW := cr.pdf.GetStringWidth(calloutLabel)
			pr := cr.su(float64(court.PlayerRadius))
			cr.pdf.Text(px-cstrW/2, py-pr-cr.su(2), calloutLabel)
		}
	}
}

// drawRotatedEllipseFill draws a filled ellipse approximated as a polygon.
func (cr *courtRenderer) drawRotatedEllipseFill(cx, cy, rx, ry, rotDeg float64, col [3]int, alpha float64) {
	const steps = 24
	rotRad := rotDeg * math.Pi / 180
	cosR := math.Cos(rotRad)
	sinR := math.Sin(rotRad)

	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps) * 2 * math.Pi
		lx := rx * math.Cos(t)
		ly := ry * math.Sin(t)
		ex := cx + lx*cosR - ly*sinR
		ey := cy + lx*sinR + ly*cosR
		if i == 0 {
			cr.pdf.MoveTo(ex, ey)
		} else {
			cr.pdf.LineTo(ex, ey)
		}
	}
	cr.pdf.ClosePath()

	cr.pdf.SetFillColor(col[0], col[1], col[2])
	if alpha < 1.0 {
		cr.pdf.SetAlpha(alpha, "Normal")
	}
	cr.pdf.DrawPath("F")
	if alpha < 1.0 {
		cr.pdf.SetAlpha(1.0, "Normal")
	}
}

// --- Accessories (same shapes as editor) ---

func (cr *courtRenderer) drawAccessories(seq *model.Sequence) {
	for i := range seq.Accessories {
		acc := &seq.Accessories[i]
		ax, ay := cr.relToMM(acc.Position)
		rad := acc.Rotation * math.Pi / 180
		cos := math.Cos(rad)
		sin := math.Sin(rad)
		rotate := func(lx, ly float64) (float64, float64) {
			return ax + lx*cos - ly*sin, ay + lx*sin + ly*cos
		}

		switch acc.Type {
		case model.AccessoryCone:
			cs := cr.su(float64(court.AccessoryConeSize))
			cr.pdf.SetFillColor(255, 165, 0) // orange
			tx, ty := rotate(0, -cs)
			lx, ly := rotate(-cs*0.7, cs*0.5)
			rx, ry := rotate(cs*0.7, cs*0.5)
			cr.pdf.MoveTo(tx, ty)
			cr.pdf.LineTo(lx, ly)
			cr.pdf.LineTo(rx, ry)
			cr.pdf.ClosePath()
			cr.pdf.DrawPath("F")

		case model.AccessoryAgilityLadder:
			w := cr.su(float64(court.AccessoryLadderWidth))
			h := cr.su(float64(court.AccessoryLadderLength))
			lw := cr.su(1.5)
			if lw < 0.15 {
				lw = 0.15
			}
			cr.pdf.SetDrawColor(255, 215, 0) // gold
			cr.pdf.SetLineWidth(lw)
			c0x, c0y := rotate(-w/2, -h/2)
			c1x, c1y := rotate(w/2, -h/2)
			c2x, c2y := rotate(w/2, h/2)
			c3x, c3y := rotate(-w/2, h/2)
			cr.pdf.Line(c0x, c0y, c1x, c1y)
			cr.pdf.Line(c1x, c1y, c2x, c2y)
			cr.pdf.Line(c2x, c2y, c3x, c3y)
			cr.pdf.Line(c3x, c3y, c0x, c0y)
			rungs := court.AccessoryLadderRungs
			for ri := 1; ri < rungs; ri++ {
				t := float64(ri) / float64(rungs)
				rlx := c0x + (c3x-c0x)*t
				rly := c0y + (c3y-c0y)*t
				rrx := c1x + (c2x-c1x)*t
				rry := c1y + (c2y-c1y)*t
				cr.pdf.Line(rlx, rly, rrx, rry)
			}

		case model.AccessoryChair:
			// Same shape as editor: seat rectangle + backrest bar + leg circles.
			cs := cr.su(float64(court.AccessoryChairSize))
			half := cs * 0.5
			lw := cr.su(2)
			if lw < 0.15 {
				lw = 0.15
			}

			tlx, tly := rotate(-half, -half)
			trx, trY := rotate(half, -half)
			brx, brY := rotate(half, half)
			blx, blY := rotate(-half, half)

			// Seat fill.
			cr.pdf.SetFillColor(0x90, 0x90, 0x90)
			cr.pdf.SetAlpha(0.67, "Normal")
			cr.pdf.MoveTo(tlx, tly)
			cr.pdf.LineTo(trx, trY)
			cr.pdf.LineTo(brx, brY)
			cr.pdf.LineTo(blx, blY)
			cr.pdf.ClosePath()
			cr.pdf.DrawPath("F")
			cr.pdf.SetAlpha(1.0, "Normal")

			// Seat outline.
			cr.pdf.SetDrawColor(0x80, 0x80, 0x80)
			cr.pdf.SetLineWidth(lw)
			cr.pdf.Line(tlx, tly, trx, trY)
			cr.pdf.Line(trx, trY, brx, brY)
			cr.pdf.Line(brx, brY, blx, blY)
			cr.pdf.Line(blx, blY, tlx, tly)

			// Backrest bar at top.
			backLw := cr.su(4)
			if backLw < 0.2 {
				backLw = 0.2
			}
			btlx, btly := rotate(-half, -half-backLw/2)
			btrx, btrY := rotate(half, -half-backLw/2)
			cr.pdf.SetLineWidth(backLw)
			cr.pdf.Line(btlx, btly, btrx, btrY)

			// Leg circles at corners.
			legR := cr.su(2.5)
			if legR < 0.15 {
				legR = 0.15
			}
			cr.pdf.SetFillColor(0x60, 0x60, 0x60)
			cr.pdf.Circle(tlx, tly, legR, "F")
			cr.pdf.Circle(trx, trY, legR, "F")
			cr.pdf.Circle(brx, brY, legR, "F")
			cr.pdf.Circle(blx, blY, legR, "F")
		}
	}
}

// --- Actions (same proportions as editor) ---

func (cr *courtRenderer) drawActions(seq *model.Sequence) {
	lw := cr.su(float64(court.ArrowLineWidth))
	for i := range seq.Actions {
		act := &seq.Actions[i]
		fromX, fromY := cr.resolveActionRef(act.From, seq.Players)
		toX, toY := cr.resolveActionRef(act.To, seq.Players)

		col := actionColorPDF(act.Type)
		cr.pdf.SetDrawColor(col[0], col[1], col[2])
		cr.pdf.SetLineWidth(lw)

		switch act.Type {
		case model.ActionPass:
			dl := cr.su(float64(court.DashLen))
			gl := cr.su(float64(court.GapLen))
			cr.drawDashed(fromX, fromY, toX, toY, dl, gl)
			cr.drawArrowHead(fromX, fromY, toX, toY, col)
		case model.ActionDribble:
			cr.drawZigzag(fromX, fromY, toX, toY)
			cr.drawArrowHead(fromX, fromY, toX, toY, col)
		case model.ActionScreen:
			cr.pdf.SetLineWidth(lw * 3)
			cr.pdf.Line(fromX, fromY, toX, toY)
		case model.ActionContest:
			dl := cr.su(float64(court.DashLen))
			gl := cr.su(float64(court.GapLen))
			cr.drawDashed(fromX, fromY, toX, toY, dl, gl)
			cr.drawArrowHead(fromX, fromY, toX, toY, col)
		default:
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
	segments := court.ZigzagSegments
	amp := cr.su(float64(court.ZigzagAmplitude))
	ux := dx / length
	uy := dy / length
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
			side = 0
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

	size := cr.su(float64(court.ArrowHeadSize))
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
