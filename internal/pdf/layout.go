package pdf

import (
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
)

// wrapText splits a cp1252-encoded string into lines that fit within maxW mm.
// Unlike fpdf.SplitText, this works safely with cp1252 byte strings because
// GetStringWidth uses byte-level width lookup for standard fonts.
func wrapText(p *fpdf.Fpdf, txt string, maxW float64) []string {
	if p.GetStringWidth(txt) <= maxW {
		return []string{txt}
	}
	words := strings.Split(txt, " ")
	var lines []string
	cur := ""
	for _, w := range words {
		if cur == "" {
			cur = w
			continue
		}
		test := cur + " " + w
		if p.GetStringWidth(test) <= maxW {
			cur = test
		} else {
			lines = append(lines, cur)
			cur = w
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

// exerciseBlock holds resolved data for rendering one exercise in the PDF.
type exerciseBlock struct {
	entry    model.ExerciseEntry
	exercise *model.Exercise
	index    int
}

// layoutHeader draws the header bar with session title and metadata.
func layoutHeader(pdf *fpdf.Fpdf, tr func(string) string, session *model.Session) {
	pdf.SetFillColor(colorHeaderBg[0], colorHeaderBg[1], colorHeaderBg[2])
	pdf.Rect(marginLeft, marginTop, contentWidth, headerHeight, "F")

	pdf.SetFont("Helvetica", "B", fontSizeTitle)
	pdf.SetTextColor(colorWhite[0], colorWhite[1], colorWhite[2])
	pdf.SetXY(marginLeft+4, marginTop+2)
	pdf.CellFormat(contentWidth-8, 7, tr(session.Title), "", 0, "L", false, 0, "")

	if session.Subtitle != "" {
		pdf.SetFont("Helvetica", "", fontSizeSubtitle)
		pdf.SetXY(marginLeft+4, marginTop+9)
		pdf.CellFormat(contentWidth-8, 5, tr(session.Subtitle), "", 0, "L", false, 0, "")
	}

	if session.AgeGroup != "" {
		pdf.SetFont("Helvetica", "B", fontSizeSubtitle)
		pdf.SetXY(marginLeft+contentWidth-40, marginTop+2)
		pdf.CellFormat(36, 7, tr(session.AgeGroup), "", 0, "R", false, 0, "")
	}

	if session.Date != "" {
		pdf.SetFont("Helvetica", "", fontSizeSubtitle)
		pdf.SetXY(marginLeft+contentWidth-40, marginTop+9)
		pdf.CellFormat(36, 5, tr(session.Date), "", 0, "R", false, 0, "")
	}
}

// layoutExerciseBlock draws one exercise block (header + per-sequence court diagrams + instructions).
// Returns the Y position after the block.
func layoutExerciseBlock(pdf *fpdf.Fpdf, tr func(string) string, y float64, block exerciseBlock) float64 {
	ex := block.exercise
	if ex == nil {
		return y
	}

	numSeqs := len(ex.Sequences)
	if numSeqs == 0 {
		numSeqs = 1
	}

	// Check if we need a new page for the header.
	if y+20 > pageHeight-marginBottom {
		pdf.AddPage()
		y = marginTop
	}

	// Exercise header line — white text on blue background.
	pdf.SetFillColor(colorHeaderBg[0], colorHeaderBg[1], colorHeaderBg[2])
	pdf.Rect(marginLeft, y, contentWidth, 6, "F")

	pdf.SetFont("Helvetica", "B", fontSizeHeader)
	pdf.SetTextColor(colorWhite[0], colorWhite[1], colorWhite[2])
	pdf.SetXY(marginLeft, y)

	headerText := fmt.Sprintf("%d. %s", block.index+1, ex.Name)
	pdf.CellFormat(contentWidth*0.6, 6, tr(headerText), "", 0, "L", false, 0, "")

	// Duration.
	pdf.SetFont("Helvetica", "", fontSizeSmall)
	pdf.SetTextColor(colorWhite[0], colorWhite[1], colorWhite[2])
	if ex.Duration != "" {
		pdf.SetXY(marginLeft+contentWidth*0.6, y)
		pdf.CellFormat(contentWidth*0.4-20, 6, ex.Duration, "", 0, "R", false, 0, "")
	}
	// Intensity dots (green/yellow/red).
	drawIntensityDots(pdf, marginLeft+contentWidth-16, y+3, int(ex.Intensity))

	y += 7

	// Court aspect ratio.
	var aspectR float64
	if ex.CourtType == model.FullCourt {
		aspectR = 15.0 / 28.0
	} else {
		aspectR = 15.0 / 14.0
	}
	if aspectR <= 0 {
		aspectR = 15.0 / 14.0
	}

	// Uniform grid layout: up to 4 diagrams per row, instructions below each.
	const seqCols = 4
	gap := columnGap * 0.6
	cellW := (contentWidth - gap*float64(seqCols-1)) / float64(seqCols)

	seqDiagramH := courtDiagramSize * 0.45
	courtActualW := seqDiagramH * aspectR
	if courtActualW > cellW {
		courtActualW = cellW
		seqDiagramH = courtActualW / aspectR
	}

	actualSeqs := len(ex.Sequences)
	if actualSeqs == 0 {
		actualSeqs = 1
	}

	for si := 0; si < actualSeqs; si += seqCols {
		rowEnd := si + seqCols
		if rowEnd > actualSeqs {
			rowEnd = actualSeqs
		}

		// Check page space.
		rowEstimate := seqDiagramH + 25
		if y+rowEstimate > pageHeight-marginBottom {
			pdf.AddPage()
			y = marginTop
		}

		// Sequence labels.
		for ci := 0; ci < rowEnd-si; ci++ {
			seq := &ex.Sequences[si+ci]
			colX := marginLeft + float64(ci)*(cellW+gap)
			if seq.Label != "" {
				pdf.SetFont("Helvetica", "B", fontSizeSmall)
				pdf.SetTextColor(colorHeaderBg[0], colorHeaderBg[1], colorHeaderBg[2])
				pdf.SetXY(colX, y)
				pdf.CellFormat(cellW, 3.5, tr(seq.Label), "", 0, "L", false, 0, "")
			}
		}
		y += 4

		// Court diagrams side by side.
		for ci := 0; ci < rowEnd-si; ci++ {
			colX := marginLeft + float64(ci)*(cellW+gap)
			drawCourtDiagram(pdf, colX, y, courtActualW, seqDiagramH, ex, si+ci)
		}
		y += seqDiagramH + 1

		// Instructions below each diagram.
		instrYs := make([]float64, rowEnd-si)
		for ci := 0; ci < rowEnd-si; ci++ {
			instrYs[ci] = y
		}

		pdf.SetFont("Helvetica", "", fontSizeSmall)
		pdf.SetTextColor(colorBlack[0], colorBlack[1], colorBlack[2])
		for ci := 0; ci < rowEnd-si; ci++ {
			colX := marginLeft + float64(ci)*(cellW+gap)
			seq := &ex.Sequences[si+ci]
			iy := instrYs[ci]
			for _, instr := range seq.Instructions {
				lines := wrapText(pdf, tr(instr), cellW-4)
				for li, line := range lines {
					pdf.SetXY(colX+1, iy)
					if li == 0 {
						pdf.CellFormat(cellW-2, 3.2, "- "+line, "", 0, "L", false, 0, "")
					} else {
						pdf.CellFormat(cellW-2, 3.2, "  "+line, "", 0, "L", false, 0, "")
					}
					iy += 3.4
				}
			}
			instrYs[ci] = iy
		}

		maxY := y
		for _, iy := range instrYs {
			if iy > maxY {
				maxY = iy
			}
		}
		y = maxY + 2
	}

	// Variants.
	if len(block.entry.Variants) > 0 {
		y += 1
		pdf.SetFont("Helvetica", "I", fontSizeSmall)
		pdf.SetTextColor(colorNeutral[0], colorNeutral[1], colorNeutral[2])
		for _, v := range block.entry.Variants {
			pdf.SetXY(marginLeft+4, y)
			pdf.CellFormat(contentWidth, 3.5, tr(i18n.Tf("pdf.variant", v.Exercise)), "", 0, "L", false, 0, "")
			y += 4
		}
	}

	return y + exerciseBlockGap
}

// layoutSummaryTable draws the summary table at the end.
func layoutSummaryTable(pdf *fpdf.Fpdf, tr func(string) string, y float64, blocks []exerciseBlock) float64 {
	// Check page space.
	tableHeight := float64(len(blocks)+1)*6 + 10
	if y+tableHeight > pageHeight-marginBottom {
		pdf.AddPage()
		y = marginTop
	}

	// Header.
	pdf.SetFont("Helvetica", "B", fontSizeHeader)
	pdf.SetTextColor(colorHeaderBg[0], colorHeaderBg[1], colorHeaderBg[2])
	pdf.SetXY(marginLeft, y)
	pdf.CellFormat(contentWidth, 6, tr(i18n.T("pdf.summary")), "", 0, "L", false, 0, "")
	y += 8

	// Table header.
	colWidths := []float64{10, contentWidth * 0.5, 30, 20, contentWidth*0.5 - 50}
	headers := []string{tr(i18n.T("pdf.col_num")), tr(i18n.T("pdf.col_exercise")), tr(i18n.T("pdf.col_duration")), tr(i18n.T("pdf.col_intensity")), tr(i18n.T("pdf.col_category"))}
	pdf.SetFont("Helvetica", "B", fontSizeSmall)
	pdf.SetFillColor(colorLightBg[0], colorLightBg[1], colorLightBg[2])
	pdf.SetTextColor(colorBlack[0], colorBlack[1], colorBlack[2])
	x := marginLeft
	for i, h := range headers {
		pdf.SetXY(x, y)
		pdf.CellFormat(colWidths[i], 5, h, "1", 0, "C", true, 0, "")
		x += colWidths[i]
	}
	y += 5

	// Rows.
	pdf.SetFont("Helvetica", "", fontSizeSmall)
	totalMinutes := 0
	for _, b := range blocks {
		if b.exercise == nil {
			continue
		}
		x = marginLeft
		cells := []string{
			fmt.Sprintf("%d", b.index+1),
			tr(b.exercise.Name),
			tr(b.exercise.Duration),
			"", // intensity drawn as colored dots below
			tr(i18n.T("category." + string(b.exercise.Category))),
		}
		for i, cell := range cells {
			pdf.SetXY(x, y)
			pdf.CellFormat(colWidths[i], 5, cell, "1", 0, "C", false, 0, "")
			x += colWidths[i]
		}
		// Draw intensity dots centered in the intensity column.
		intColX := marginLeft + colWidths[0] + colWidths[1] + colWidths[2]
		drawIntensityDots(pdf, intColX+colWidths[3]/2-5, y+2.5, int(b.exercise.Intensity))
		y += 5
		totalMinutes += parseDurationMins(b.exercise.Duration)
	}

	// Total row.
	y += 2
	pdf.SetFont("Helvetica", "B", fontSizeBody)
	pdf.SetXY(marginLeft, y)
	totalStr := i18n.Tf("pdf.total_format", i18n.Tf("session.duration_m", totalMinutes))
	if totalMinutes >= 60 {
		totalStr = i18n.Tf("pdf.total_format", i18n.Tf("session.duration_hm", totalMinutes/60, totalMinutes%60))
	}
	pdf.CellFormat(contentWidth, 5, tr(totalStr), "", 0, "R", false, 0, "")
	y += 7

	return y
}

// layoutCoachNotes draws coach notes and philosophy sections.
func layoutCoachNotes(pdf *fpdf.Fpdf, tr func(string) string, y float64, session *model.Session) float64 {
	if len(session.CoachNotes) == 0 && session.Philosophy == "" {
		return y
	}

	if y+30 > pageHeight-marginBottom {
		pdf.AddPage()
		y = marginTop
	}

	// Coach notes.
	if len(session.CoachNotes) > 0 {
		pdf.SetFont("Helvetica", "B", fontSizeHeader)
		pdf.SetTextColor(colorHeaderBg[0], colorHeaderBg[1], colorHeaderBg[2])
		pdf.SetXY(marginLeft, y)
		pdf.CellFormat(contentWidth, 6, tr(i18n.T("pdf.coach_notes")), "", 0, "L", false, 0, "")
		y += 7

		pdf.SetFont("Helvetica", "", fontSizeBody)
		pdf.SetTextColor(colorBlack[0], colorBlack[1], colorBlack[2])
		for _, note := range session.CoachNotes {
			pdf.SetXY(marginLeft+2, y)
			lines := wrapText(pdf, tr(note), contentWidth-4)
			for _, line := range lines {
				pdf.SetXY(marginLeft+2, y)
				pdf.CellFormat(contentWidth-4, 4, "- "+line, "", 0, "L", false, 0, "")
				y += 4.2
			}
		}
		y += 3
	}

	// Philosophy.
	if session.Philosophy != "" {
		if y+20 > pageHeight-marginBottom {
			pdf.AddPage()
			y = marginTop
		}

		pdf.SetFont("Helvetica", "B", fontSizeHeader)
		pdf.SetTextColor(colorHeaderBg[0], colorHeaderBg[1], colorHeaderBg[2])
		pdf.SetXY(marginLeft, y)
		pdf.CellFormat(contentWidth, 6, tr(i18n.T("pdf.philosophy")), "", 0, "L", false, 0, "")
		y += 7

		// Philosophy box.
		pdf.SetFillColor(colorLightBg[0], colorLightBg[1], colorLightBg[2])
		pdf.SetFont("Helvetica", "I", fontSizeBody)
		pdf.SetTextColor(colorBlack[0], colorBlack[1], colorBlack[2])

		lines := strings.Split(tr(session.Philosophy), "\n")
		boxY := y
		for _, line := range lines {
			wrapped := wrapText(pdf, line, contentWidth-8)
			for range wrapped {
				y += 4
			}
		}
		boxH := y - boxY + 4
		pdf.Rect(marginLeft, boxY-1, contentWidth, boxH, "F")

		y = boxY
		for _, line := range lines {
			wrapped := wrapText(pdf, line, contentWidth-8)
			for _, wl := range wrapped {
				pdf.SetXY(marginLeft+4, y)
				pdf.CellFormat(contentWidth-8, 4, wl, "", 0, "L", false, 0, "")
				y += 4
			}
		}
		y += 4
	}

	return y
}

// drawIntensityDots draws 3 colored circles (green/yellow/red) at the given position.
// Active dots use their color; inactive dots are gray.
func drawIntensityDots(pdf *fpdf.Fpdf, x, y float64, level int) {
	colors := [3][3]int{colorIntGreen, colorIntYellow, colorIntRed}
	r := intensityDotR
	for i := 0; i < 3; i++ {
		cx := x + float64(i)*(r*2+1.5)
		if i < level {
			pdf.SetFillColor(colors[i][0], colors[i][1], colors[i][2])
		} else {
			pdf.SetFillColor(colorIntOff[0], colorIntOff[1], colorIntOff[2])
		}
		pdf.Circle(cx, y, r, "F")
	}
}

func parseDurationMins(d string) int {
	total := 0
	num := 0
	for _, c := range d {
		if c >= '0' && c <= '9' {
			num = num*10 + int(c-'0')
		} else if c == 'h' {
			total += num * 60
			num = 0
		} else if c == 'm' {
			total += num
			num = 0
		}
	}
	return total
}
