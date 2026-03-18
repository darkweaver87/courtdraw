package ui

import (
	"image/color"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// StatusBar displays status messages at the bottom of the editor.
type StatusBar struct {
	label     *canvas.Text
	bg        *canvas.Rectangle
	box       *fyne.Container
	updateBtn *TipButton    // update available button (right side)
	rightBox  *fyne.Container // right-aligned area
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
	sb.rightBox = container.NewHBox()
	bar := container.NewBorder(nil, nil, container.NewPadded(sb.label), sb.rightBox)
	sb.box = container.NewStack(sb.bg, bar)
	return sb
}

// Widget returns the status bar widget.
func (sb *StatusBar) Widget() fyne.CanvasObject {
	return sb.box
}

// ShowUpdateAvailable adds an update warning icon to the right side of the status bar.
func (sb *StatusBar) ShowUpdateAvailable(tooltip string, onTap func()) {
	if sb.updateBtn != nil {
		sb.updateBtn.SetTooltip(tooltip)
		sb.updateBtn.Show()
		return
	}
	sb.updateBtn = NewTipButton(fynetheme.WarningIcon(), tooltip, onTap)
	sb.updateBtn.SetImportance(widget.WarningImportance)
	sb.rightBox.Add(sb.updateBtn)
	sb.rightBox.Refresh()
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
