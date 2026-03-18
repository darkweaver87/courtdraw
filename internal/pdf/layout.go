package pdf

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-pdf/fpdf"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
)

// wrapText splits a cp1252-encoded string into lines that fit within maxW mm.
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

// layoutHeader draws the header bar for portrait mode.
func layoutHeader(pdf *fpdf.Fpdf, tr func(string) string, session *model.Session, ctx *layoutContext) {
	if session == nil {
		return
	}
	x := ctx.colX
	w := ctx.contentW

	pdf.SetFillColor(colorHeaderBg[0], colorHeaderBg[1], colorHeaderBg[2])
	pdf.Rect(x, ctx.margin, w, headerHeight, "F")

	pdf.SetFont("Helvetica", "B", fontSizeTitle)
	pdf.SetTextColor(colorWhite[0], colorWhite[1], colorWhite[2])
	pdf.SetXY(x+4, ctx.margin+2)
	pdf.CellFormat(w-8, 7, tr(session.Title), "", 0, "L", false, 0, "")

	if session.Subtitle != "" {
		pdf.SetFont("Helvetica", "", fontSizeSubtitle)
		pdf.SetXY(x+4, ctx.margin+9)
		pdf.CellFormat(w-8, 5, tr(session.Subtitle), "", 0, "L", false, 0, "")
	}

	if session.AgeGroup != "" {
		pdf.SetFont("Helvetica", "B", fontSizeSubtitle)
		pdf.SetXY(x+w-40, ctx.margin+2)
		pdf.CellFormat(36, 7, tr(session.AgeGroup), "", 0, "R", false, 0, "")
	}

	if session.Date != "" {
		pdf.SetFont("Helvetica", "", fontSizeSubtitle)
		pdf.SetXY(x+w-40, ctx.margin+9)
		pdf.CellFormat(36, 5, tr(session.Date), "", 0, "R", false, 0, "")
	}
}

// layoutExerciseBlock draws one exercise block.
// IMPORTANT: always reads ctx.colX (not a local copy) so that mid-block
// page breaks in 2-up mode correctly shift rendering to the right column.
func layoutExerciseBlock(pdf *fpdf.Fpdf, tr func(string) string, y float64, block exerciseBlock, ctx *layoutContext) float64 {
	ex := block.exercise
	if ex == nil {
		return y
	}

	colW := ctx.colW

	// Check if we need a new page for the header.
	if y+20 > ctx.maxY {
		y = ctx.nextPage(pdf)
	}

	// Exercise header line.
	pdf.SetFillColor(colorHeaderBg[0], colorHeaderBg[1], colorHeaderBg[2])
	pdf.Rect(ctx.colX, y, colW, 6, "F")

	pdf.SetFont("Helvetica", "B", fontSizeHeader)
	pdf.SetTextColor(colorWhite[0], colorWhite[1], colorWhite[2])
	pdf.SetXY(ctx.colX, y)

	headerText := fmt.Sprintf("%d. %s", block.index+1, ex.Name)
	pdf.CellFormat(colW*0.6, 6, tr(headerText), "", 0, "L", false, 0, "")

	// Duration.
	pdf.SetFont("Helvetica", "", fontSizeSmall)
	pdf.SetTextColor(colorWhite[0], colorWhite[1], colorWhite[2])
	if ex.Duration != "" {
		pdf.SetXY(ctx.colX+colW*0.6, y)
		pdf.CellFormat(colW*0.4-20, 6, ex.Duration, "", 0, "R", false, 0, "")
	}
	drawIntensityDots(pdf, ctx.colX+colW-16, y+3, int(ex.Intensity), ctx.dotRadius())

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

	seqCols := 4
	if colW < 150 {
		seqCols = 2
	}

	gap := columnGap * 0.6
	cellW := (colW - gap*float64(seqCols-1)) / float64(seqCols)

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
		rowEnd = min(rowEnd, actualSeqs)

		rowEstimate := seqDiagramH + 25
		if y+rowEstimate > ctx.maxY {
			y = ctx.nextPage(pdf)
		}

		// Sequence labels — use ctx.colX (may have changed after nextPage).
		for ci := range rowEnd - si {
			seq := &ex.Sequences[si+ci]
			cx := ctx.colX + float64(ci)*(cellW+gap)
			if seq.Label != "" {
				pdf.SetFont("Helvetica", "B", fontSizeSmall)
				pdf.SetTextColor(colorHeaderBg[0], colorHeaderBg[1], colorHeaderBg[2])
				pdf.SetXY(cx, y)
				pdf.CellFormat(cellW, 3.5, tr(seq.Label), "", 0, "L", false, 0, "")
			}
		}
		y += 4

		for ci := range rowEnd - si {
			cx := ctx.colX + float64(ci)*(cellW+gap)
			drawCourtDiagram(pdf, cx, y, courtActualW, seqDiagramH, ex, si+ci)
		}
		y += seqDiagramH + 1

		instrYs := make([]float64, rowEnd-si)
		for ci := range rowEnd - si {
			instrYs[ci] = y
		}

		pdf.SetFont("Helvetica", "", fontSizeSmall)
		pdf.SetTextColor(colorBlack[0], colorBlack[1], colorBlack[2])
		for ci := range rowEnd - si {
			cx := ctx.colX + float64(ci)*(cellW+gap)
			seq := &ex.Sequences[si+ci]
			iy := instrYs[ci]
			for _, instr := range seq.Instructions {
				lines := wrapText(pdf, tr(instr), cellW-4)
				for li, line := range lines {
					pdf.SetXY(cx+1, iy)
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

		localMaxY := y
		for _, iy := range instrYs {
			if iy > localMaxY {
				localMaxY = iy
			}
		}
		y = localMaxY + 2
	}

	// Variants.
	if len(block.entry.Variants) > 0 {
		y += 1
		pdf.SetFont("Helvetica", "I", fontSizeSmall)
		pdf.SetTextColor(colorNeutral[0], colorNeutral[1], colorNeutral[2])
		for _, v := range block.entry.Variants {
			pdf.SetXY(ctx.colX+4, y)
			pdf.CellFormat(colW, 3.5, tr(i18n.Tf(i18n.KeyPdfVariant, v.Exercise)), "", 0, "L", false, 0, "")
			y += 4
		}
	}

	return y + exerciseBlockGap
}

// layoutSummaryTable draws the summary table.
func layoutSummaryTable(pdf *fpdf.Fpdf, tr func(string) string, y float64, blocks []exerciseBlock, ctx *layoutContext) float64 {
	colW := ctx.colW

	tableHeight := float64(len(blocks)+1)*6 + 10
	if y+tableHeight > ctx.maxY {
		y = ctx.nextPage(pdf)
	}

	pdf.SetFont("Helvetica", "B", fontSizeHeader)
	pdf.SetTextColor(colorHeaderBg[0], colorHeaderBg[1], colorHeaderBg[2])
	pdf.SetXY(ctx.colX, y)
	pdf.CellFormat(colW, 6, tr(i18n.T(i18n.KeyPdfSummary)), "", 0, "L", false, 0, "")
	y += 8

	nameW := colW*0.5 - 10
	if nameW < 30 {
		nameW = 30
	}
	catW := colW - 10 - nameW - 30 - 20
	if catW < 10 {
		catW = 10
	}
	colWidths := []float64{10, nameW, 30, 20, catW}
	headers := []string{tr(i18n.T(i18n.KeyPdfColNum)), tr(i18n.T(i18n.KeyPdfColExercise)), tr(i18n.T(i18n.KeyPdfColDuration)), tr(i18n.T(i18n.KeyPdfColIntensity)), tr(i18n.T(i18n.KeyPdfColCategory))}
	pdf.SetFont("Helvetica", "B", fontSizeSmall)
	pdf.SetFillColor(colorLightBg[0], colorLightBg[1], colorLightBg[2])
	pdf.SetDrawColor(colorBlack[0], colorBlack[1], colorBlack[2])
	pdf.SetTextColor(colorBlack[0], colorBlack[1], colorBlack[2])
	x := ctx.colX
	for i, h := range headers {
		pdf.SetXY(x, y)
		pdf.CellFormat(colWidths[i], 5, h, "1", 0, "C", true, 0, "")
		x += colWidths[i]
	}
	y += 5

	pdf.SetFont("Helvetica", "", fontSizeSmall)
	totalMinutes := 0
	lineH := 3.6
	for _, b := range blocks {
		if b.exercise == nil {
			continue
		}
		cells := []string{
			strconv.Itoa(b.index+1),
			tr(b.exercise.Name),
			tr(b.exercise.Duration),
			"",
			tr(i18n.T(categoryI18nKey(b.exercise.Category))),
		}
		aligns := []string{"C", "L", "C", "C", "C"}

		// Compute wrapped lines per cell and row height.
		wrappedCells := make([][]string, len(cells))
		maxLines := 1
		for i, cell := range cells {
			wrappedCells[i] = wrapText(pdf, cell, colWidths[i]-2)
			if len(wrappedCells[i]) > maxLines {
				maxLines = len(wrappedCells[i])
			}
		}
		rowH := float64(maxLines) * lineH

		// Draw bordered cells with wrapped text.
		x = ctx.colX
		for i := range cells {
			// Draw the cell border/background.
			pdf.Rect(x, y, colWidths[i], rowH, "D")
			// Draw each line of text inside the cell.
			for li, line := range wrappedCells[i] {
				pdf.SetXY(x+1, y+float64(li)*lineH)
				pdf.CellFormat(colWidths[i]-2, lineH, line, "", 0, aligns[i], false, 0, "")
			}
			x += colWidths[i]
		}
		intColX := ctx.colX + colWidths[0] + colWidths[1] + colWidths[2]
		drawIntensityDots(pdf, intColX+colWidths[3]/2-5, y+rowH/2, int(b.exercise.Intensity), ctx.dotRadius())
		y += rowH
		totalMinutes += parseDurationMins(b.exercise.Duration)
	}

	y += 2
	pdf.SetFont("Helvetica", "B", fontSizeBody)
	pdf.SetXY(ctx.colX, y)
	totalStr := i18n.Tf(i18n.KeyPdfTotalFormat, i18n.Tf(i18n.KeySessionDurationM, totalMinutes))
	if totalMinutes >= 60 {
		totalStr = i18n.Tf(i18n.KeyPdfTotalFormat, i18n.Tf(i18n.KeySessionDurationHm, totalMinutes/60, totalMinutes%60))
	}
	pdf.CellFormat(colW, 5, tr(totalStr), "", 0, "R", false, 0, "")
	y += 7

	return y
}

// layoutCoachNotes draws coach notes and philosophy sections.
func layoutCoachNotes(pdf *fpdf.Fpdf, tr func(string) string, y float64, session *model.Session, ctx *layoutContext) float64 {
	if len(session.CoachNotes) == 0 && session.Philosophy == "" {
		return y
	}

	colW := ctx.colW

	if y+30 > ctx.maxY {
		y = ctx.nextPage(pdf)
	}

	if len(session.CoachNotes) > 0 {
		pdf.SetFont("Helvetica", "B", fontSizeHeader)
		pdf.SetTextColor(colorHeaderBg[0], colorHeaderBg[1], colorHeaderBg[2])
		pdf.SetXY(ctx.colX, y)
		pdf.CellFormat(colW, 6, tr(i18n.T(i18n.KeyPdfCoachNotes)), "", 0, "L", false, 0, "")
		y += 7

		pdf.SetFont("Helvetica", "", fontSizeBody)
		pdf.SetTextColor(colorBlack[0], colorBlack[1], colorBlack[2])
		for _, note := range session.CoachNotes {
			pdf.SetXY(ctx.colX+2, y)
			lines := wrapText(pdf, tr(note), colW-4)
			for _, line := range lines {
				pdf.SetXY(ctx.colX+2, y)
				pdf.CellFormat(colW-4, 4, "- "+line, "", 0, "L", false, 0, "")
				y += 4.2
			}
		}
		y += 3
	}

	if session.Philosophy != "" {
		if y+20 > ctx.maxY {
			y = ctx.nextPage(pdf)
		}

		pdf.SetFont("Helvetica", "B", fontSizeHeader)
		pdf.SetTextColor(colorHeaderBg[0], colorHeaderBg[1], colorHeaderBg[2])
		pdf.SetXY(ctx.colX, y)
		pdf.CellFormat(colW, 6, tr(i18n.T(i18n.KeyPdfPhilosophy)), "", 0, "L", false, 0, "")
		y += 7

		pdf.SetFillColor(colorLightBg[0], colorLightBg[1], colorLightBg[2])
		pdf.SetFont("Helvetica", "I", fontSizeBody)
		pdf.SetTextColor(colorBlack[0], colorBlack[1], colorBlack[2])

		lines := strings.Split(tr(session.Philosophy), "\n")
		boxY := y
		for _, line := range lines {
			wrapped := wrapText(pdf, line, colW-8)
			for range wrapped {
				y += 4
			}
		}
		boxH := y - boxY + 4
		pdf.Rect(ctx.colX, boxY-1, colW, boxH, "F")

		y = boxY
		for _, line := range lines {
			wrapped := wrapText(pdf, line, colW-8)
			for _, wl := range wrapped {
				pdf.SetXY(ctx.colX+4, y)
				pdf.CellFormat(colW-8, 4, wl, "", 0, "L", false, 0, "")
				y += 4
			}
		}
		y += 4
	}

	return y
}

// drawIntensityDots draws 3 colored circles (green/yellow/red) at the given position.
func drawIntensityDots(pdf *fpdf.Fpdf, x, y float64, level int, r float64) {
	colors := [3][3]int{colorIntGreen, colorIntYellow, colorIntRed}
	for i := range 3 {
		cx := x + float64(i)*(r*2+1.5)
		if i < level {
			pdf.SetFillColor(colors[i][0], colors[i][1], colors[i][2])
		} else {
			pdf.SetFillColor(colorIntOff[0], colorIntOff[1], colorIntOff[2])
		}
		pdf.Circle(cx, y, r, "F")
	}
}

// categoryI18nKey returns the i18n constant key for a known category.
func categoryI18nKey(cat model.Category) string {
	switch cat {
	case model.CategoryWarmup:
		return i18n.KeyCategoryWarmup
	case model.CategoryOffense:
		return i18n.KeyCategoryOffense
	case model.CategoryDefense:
		return i18n.KeyCategoryDefense
	case model.CategoryTransition:
		return i18n.KeyCategoryTransition
	case model.CategoryScrimmage:
		return i18n.KeyCategoryScrimmage
	case model.CategoryCooldown:
		return i18n.KeyCategoryCooldown
	default:
		return "category." + string(cat)
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
