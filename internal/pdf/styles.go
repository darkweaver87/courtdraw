package pdf

import (
	"github.com/go-pdf/fpdf"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// PDF colors (R, G, B values 0–255).
var (
	colorAttack    = [3]int{230, 57, 70}   // #e63946
	colorDefense   = [3]int{29, 53, 87}    // #1d3557
	colorCoach     = [3]int{244, 162, 97}   // #f4a261
	colorCloseOut  = [3]int{42, 111, 219}   // #2a6fdb
	colorNeutral   = [3]int{136, 136, 136}  // #888888
	colorCourtBg   = [3]int{200, 150, 100}   // #c89664 parquet
	colorCourtLine = [3]int{255, 255, 255}  // #ffffff
	colorMaxInt    = [3]int{193, 18, 31}    // #c1121f
	colorWhite     = [3]int{255, 255, 255}
	colorBlack     = [3]int{0, 0, 0}
	colorLightBg   = [3]int{241, 250, 238}  // #f1faee
	colorHeaderBg  = [3]int{29, 53, 87}     // navy
	colorScreen    = [3]int{255, 183, 3}    // #ffb703

	// Intensity dot colors: green, yellow, red.
	colorIntGreen  = [3]int{76, 175, 80}   // #4caf50
	colorIntYellow = [3]int{255, 193, 7}   // #ffc107
	colorIntRed    = [3]int{244, 67, 54}   // #f44336
	colorIntOff    = [3]int{180, 180, 180} // #b4b4b4

	colorFoldLine = [3]int{200, 200, 200} // light gray fold indicator
)

// PageLayout selects the PDF page orientation and column arrangement.
type PageLayout int

const (
	// LayoutPortrait is A4 portrait (210×297mm), single column.
	LayoutPortrait PageLayout = iota
	// LayoutLandscape2Up is A4 landscape (297×210mm), two A5-like columns.
	LayoutLandscape2Up
)

// PDF layout constants in mm.
const (
	pageWidth  = 210.0 // A4 portrait
	pageHeight = 297.0 // A4 portrait
	marginLeft = 10.0
	marginRight = 10.0
	marginTop  = 10.0
	marginBottom = 10.0

	contentWidth = pageWidth - marginLeft - marginRight

	// Landscape 2-up: two A5 columns on one A4 landscape sheet.
	landscapePageW = 297.0 // physical A4 landscape width
	landscapePageH = 210.0 // physical A4 landscape height
	a5PageW        = landscapePageW / 2 // 148.5mm — each column
	a5Margin       = 7.0                // column inner margin
	a5ContentW     = a5PageW - 2*a5Margin // 134.5mm

	headerHeight     = 18.0
	courtDiagramSize = 60.0  // height for court diagram
	exerciseBlockGap = 6.0
	columnGap        = 6.0

	fontSizeTitle     = 14.0
	fontSizeSubtitle  = 9.0
	fontSizeHeader    = 10.0
	fontSizeBody      = 8.0
	fontSizeSmall     = 7.0

	lineWidthThin  = 0.3
	lineWidthThick = 0.6

	intensityDotR      = 1.3 // intensity dot radius (portrait)
	intensityDotR_a5   = 1.3 // intensity dot radius (landscape A5)
)

// layoutContext holds geometry for the current layout mode.
type layoutContext struct {
	mode     PageLayout
	contentW float64 // usable content width within current column
	colW     float64 // = contentW
	colX     float64 // X offset for left edge of content area
	maxY     float64 // maximum Y before page break
	margin   float64 // margin size

	// 2-up state: header is full-width, drawn once per physical page.
	slot    int                    // 0 = left, 1 = right
	session *model.Session
	tr      func(string) string

	// Y position just below the full-width header (start of content area).
	contentStartY float64
}

func (ctx *layoutContext) dotRadius() float64 {
	if ctx.mode == LayoutLandscape2Up {
		return intensityDotR_a5
	}
	return intensityDotR
}

func newLayoutContext(mode PageLayout) *layoutContext {
	switch mode {
	case LayoutLandscape2Up:
		return &layoutContext{
			mode:     LayoutLandscape2Up,
			contentW: a5ContentW,
			colW:     a5ContentW,
			colX:     a5Margin, // left column
			maxY:     landscapePageH - a5Margin,
			margin:   a5Margin,
			slot:     0,
		}
	default:
		return &layoutContext{
			mode:     LayoutPortrait,
			contentW: contentWidth,
			colW:     contentWidth,
			colX:     marginLeft,
			maxY:     pageHeight - marginBottom,
			margin:   marginLeft,
		}
	}
}

// nextPage handles page breaks.
// In portrait mode, adds a physical page.
// In 2-up mode, switches left→right column, or adds a new physical page.
// Returns the starting Y for content.
func (ctx *layoutContext) nextPage(pdf *fpdf.Fpdf) float64 {
	if ctx.mode != LayoutLandscape2Up {
		pdf.AddPage()
		return marginTop
	}

	if ctx.slot == 0 {
		// Switch to right column — same physical page, no new header.
		ctx.slot = 1
		ctx.colX = a5PageW + a5Margin
		return ctx.contentStartY
	}

	// Right column full — new physical page (no header).
	ctx.slot = 0
	ctx.colX = a5Margin
	pdf.AddPage()
	drawFoldLine(pdf)
	ctx.contentStartY = a5Margin
	return ctx.contentStartY
}

// drawFoldLine draws a thin dashed line at the center of the A4 landscape page.
func drawFoldLine(pdf *fpdf.Fpdf) {
	pdf.SetDrawColor(colorFoldLine[0], colorFoldLine[1], colorFoldLine[2])
	pdf.SetLineWidth(0.2)
	pdf.SetDashPattern([]float64{2, 2}, 0)
	pdf.Line(a5PageW, 5, a5PageW, landscapePageH-5)
	pdf.SetDashPattern([]float64{}, 0)
}

// layoutLandscapeHeader draws the session header spanning the full landscape width.
func layoutLandscapeHeader(pdf *fpdf.Fpdf, tr func(string) string, session *model.Session) {
	if session == nil {
		return
	}
	w := landscapePageW - 2*marginTop // full width minus small margins
	x := marginTop

	pdf.SetFillColor(colorHeaderBg[0], colorHeaderBg[1], colorHeaderBg[2])
	pdf.Rect(x, marginTop, w, headerHeight, "F")

	pdf.SetFont("Helvetica", "B", fontSizeTitle)
	pdf.SetTextColor(colorWhite[0], colorWhite[1], colorWhite[2])
	pdf.SetXY(x+4, marginTop+2)
	pdf.CellFormat(w-8, 7, tr(session.Title), "", 0, "L", false, 0, "")

	if session.Subtitle != "" {
		pdf.SetFont("Helvetica", "", fontSizeSubtitle)
		pdf.SetXY(x+4, marginTop+9)
		pdf.CellFormat(w-8, 5, tr(session.Subtitle), "", 0, "L", false, 0, "")
	}

	if session.AgeGroup != "" {
		pdf.SetFont("Helvetica", "B", fontSizeSubtitle)
		pdf.SetXY(x+w-40, marginTop+2)
		pdf.CellFormat(36, 7, tr(session.AgeGroup), "", 0, "R", false, 0, "")
	}

	if session.Date != "" {
		pdf.SetFont("Helvetica", "", fontSizeSubtitle)
		pdf.SetXY(x+w-40, marginTop+9)
		pdf.CellFormat(36, 5, tr(session.Date), "", 0, "R", false, 0, "")
	}
}
