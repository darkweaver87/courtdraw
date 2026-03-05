package theme

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Colors from the specification.
var (
	ColorAttack     = color.NRGBA{R: 0xe6, G: 0x39, B: 0x46, A: 0xff}
	ColorDefense    = color.NRGBA{R: 0x1d, G: 0x35, B: 0x57, A: 0xff}
	ColorDefenseArr = color.NRGBA{R: 0x2a, G: 0x6f, B: 0xdb, A: 0xff}
	ColorCoach      = color.NRGBA{R: 0xf4, G: 0xa2, B: 0x61, A: 0xff}
	ColorNeutral    = color.NRGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff}
	ColorCourtBg    = color.NRGBA{R: 0xc8, G: 0x96, B: 0x64, A: 0xff}
	ColorCourtLine  = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	ColorMaxInt     = color.NRGBA{R: 0xc1, G: 0x12, B: 0x1f, A: 0xff}
	ColorSpecial    = color.NRGBA{R: 0xff, G: 0xb7, B: 0x03, A: 0xff}
	ColorLightBg    = color.NRGBA{R: 0xf1, G: 0xfa, B: 0xee, A: 0xff}
	ColorDarkBg     = color.NRGBA{R: 0x26, G: 0x26, B: 0x26, A: 0xff}
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

// CourtDrawTheme implements fyne.Theme with the CourtDraw color palette.
type CourtDrawTheme struct{}

var _ fyne.Theme = (*CourtDrawTheme)(nil)

func (t *CourtDrawTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return ColorDarkBg
	case theme.ColorNameForeground:
		return color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	case theme.ColorNamePrimary:
		return ColorAttack
	case theme.ColorNameButton:
		return color.NRGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xff}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 0x38, G: 0x38, B: 0x38, A: 0xff}
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *CourtDrawTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *CourtDrawTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *CourtDrawTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 13
	case theme.SizeNamePadding:
		return 4
	}
	return theme.DefaultTheme().Size(name)
}
