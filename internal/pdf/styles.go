package pdf

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
)

// PDF layout constants in mm.
const (
	pageWidth  = 210.0 // A4
	pageHeight = 297.0 // A4
	marginLeft = 10.0
	marginRight = 10.0
	marginTop  = 10.0
	marginBottom = 10.0

	contentWidth = pageWidth - marginLeft - marginRight

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

	intensityDotR = 2.0 // intensity dot radius
)
