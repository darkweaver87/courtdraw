package ui

import (
	"image/color"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// StatusBar displays status messages at the bottom of the editor.
// On desktop it is always visible; on mobile it auto-hides.
type StatusBar struct {
	label *canvas.Text
	bg    *canvas.Rectangle
	box   *fyne.Container
}

var statusBarHeight float32 = 22

func init() {
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		statusBarHeight = 28
	}
}

// NewStatusBar creates a new status bar.
func NewStatusBar() *StatusBar {
	sb := &StatusBar{}
	sb.label = canvas.NewText("CourtDraw", color.NRGBA{R: 0xaa, G: 0xaa, B: 0xaa, A: 0xff})
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		sb.label.TextSize = 13
	} else {
		sb.label.TextSize = 11
	}
	sb.bg = canvas.NewRectangle(color.NRGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xff})
	sb.bg.SetMinSize(fyne.NewSize(0, statusBarHeight))
	padded := container.NewPadded(sb.label)
	sb.box = container.NewStack(sb.bg, padded)
	return sb
}

// Widget returns the status bar widget.
func (sb *StatusBar) Widget() fyne.CanvasObject {
	return sb.box
}

// SetStatus shows a status message. level 0 = info, 1 = error.
func (sb *StatusBar) SetStatus(msg string, level int) {
	if msg == "" {
		msg = "CourtDraw"
	}
	sb.label.Text = msg
	if level == 1 {
		sb.label.Color = color.NRGBA{R: 0xff, G: 0x66, B: 0x66, A: 0xff}
	} else {
		sb.label.Color = color.NRGBA{R: 0xaa, G: 0xaa, B: 0xaa, A: 0xff}
	}
	sb.label.Refresh()
}
