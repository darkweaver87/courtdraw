package theme

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/darkweaver87/courtdraw/internal/model"
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

	// Category colors for progress bar segments.
	ColorCatWarmup     = color.NRGBA{R: 0xff, G: 0x98, B: 0x00, A: 0xff} // orange
	ColorCatOffense    = color.NRGBA{R: 0xe6, G: 0x39, B: 0x46, A: 0xff} // red
	ColorCatDefense    = color.NRGBA{R: 0x1d, G: 0x35, B: 0x57, A: 0xff} // blue
	ColorCatTransition = color.NRGBA{R: 0x4c, G: 0xaf, B: 0x50, A: 0xff} // green
	ColorCatScrimmage  = color.NRGBA{R: 0x9c, G: 0x27, B: 0xb0, A: 0xff} // purple
	ColorCatCooldown   = color.NRGBA{R: 0x00, G: 0xbc, B: 0xd4, A: 0xff} // cyan
	ColorCatDefault    = color.NRGBA{R: 0x66, G: 0x66, B: 0x66, A: 0xff} // gray

	// Timer colors.
	ColorTimerOK      = color.NRGBA{R: 0x4c, G: 0xaf, B: 0x50, A: 0xff} // green
	ColorTimerExpired  = color.NRGBA{R: 0xe6, G: 0x39, B: 0x46, A: 0xff} // red
)

// CategoryColor returns the display color for an exercise category.
func CategoryColor(cat model.Category) color.NRGBA {
	switch cat {
	case model.CategoryWarmup:
		return ColorCatWarmup
	case model.CategoryOffense:
		return ColorCatOffense
	case model.CategoryDefense:
		return ColorCatDefense
	case model.CategoryTransition:
		return ColorCatTransition
	case model.CategoryScrimmage:
		return ColorCatScrimmage
	case model.CategoryCooldown:
		return ColorCatCooldown
	default:
		return ColorCatDefault
	}
}

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
