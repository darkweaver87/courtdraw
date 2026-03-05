package ui

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

const statusDismissDelay = 3 * time.Second

// StatusBar displays a temporary status message at the bottom of the editor.
type StatusBar struct {
	label *canvas.Text
	bg    *canvas.Rectangle
	box   *fyne.Container
	timer *time.Timer
}

// NewStatusBar creates a new status bar.
func NewStatusBar() *StatusBar {
	sb := &StatusBar{}
	sb.label = canvas.NewText("", color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	sb.label.TextSize = 13
	sb.bg = canvas.NewRectangle(color.NRGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xdd})
	sb.box = container.NewStack(sb.bg, container.NewPadded(sb.label))
	sb.box.Hide()
	return sb
}

// Widget returns the status bar widget.
func (sb *StatusBar) Widget() fyne.CanvasObject {
	return sb.box
}

// SetStatus shows a status message that auto-dismisses after 3 seconds.
func (sb *StatusBar) SetStatus(msg string, level int) {
	if sb.timer != nil {
		sb.timer.Stop()
	}
	if msg == "" {
		sb.box.Hide()
		return
	}
	sb.label.Text = msg
	sb.label.Refresh()
	if level == 1 {
		sb.bg.FillColor = color.NRGBA{R: 0x8b, G: 0x00, B: 0x00, A: 0xdd}
	} else {
		sb.bg.FillColor = color.NRGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xdd}
	}
	sb.bg.Refresh()
	sb.box.Show()

	sb.timer = time.AfterFunc(statusDismissDelay, func() {
		fyne.Do(func() {
			sb.box.Hide()
		})
	})
}
