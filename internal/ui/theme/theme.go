package theme

import (
	"image/color"

	"gioui.org/font/gofont"
	"gioui.org/text"
	"gioui.org/widget/material"
)

// Colors from the specification.
var (
	ColorAttack     = color.NRGBA{R: 0xe6, G: 0x39, B: 0x46, A: 0xff} // #e63946
	ColorDefense    = color.NRGBA{R: 0x1d, G: 0x35, B: 0x57, A: 0xff} // #1d3557
	ColorDefenseArr = color.NRGBA{R: 0x2a, G: 0x6f, B: 0xdb, A: 0xff} // #2a6fdb
	ColorCoach      = color.NRGBA{R: 0xf4, G: 0xa2, B: 0x61, A: 0xff} // #f4a261
	ColorNeutral    = color.NRGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff} // #888888
	ColorCourtBg    = color.NRGBA{R: 0x3a, G: 0x7d, B: 0x3a, A: 0xff} // #3a7d3a
	ColorCourtLine  = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff} // #ffffff
	ColorMaxInt     = color.NRGBA{R: 0xc1, G: 0x12, B: 0x1f, A: 0xff} // #c1121f
	ColorSpecial    = color.NRGBA{R: 0xff, G: 0xb7, B: 0x03, A: 0xff} // #ffb703
	ColorLightBg    = color.NRGBA{R: 0xf1, G: 0xfa, B: 0xee, A: 0xff} // #f1faee
	ColorDarkBg     = color.NRGBA{R: 0x26, G: 0x26, B: 0x26, A: 0xff} // #262626
	ColorTabActive  = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	ColorTabText    = color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff}
)

// Spacing constants in dp.
const (
	SpacingSmall  = 4
	SpacingMedium = 8
	SpacingLarge  = 16
	TabBarHeight  = 40
)

// NewTheme creates the app theme.
func NewTheme() *material.Theme {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	th.Palette.Bg = ColorDarkBg
	th.Palette.Fg = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	th.Palette.ContrastBg = ColorAttack
	return th
}
