package ui

import (
	"image/color"
	"sync"
	"time"

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
	updateBtn *TipButton     // update available button (right side)
	rightBox  *fyne.Container // right-aligned area

	dismissMu    sync.Mutex
	dismissTimer *time.Timer
}

var statusBarHeight float32 = 22

func init() {
	if isMobile {
		statusBarHeight = 28
	}
}

// NewStatusBar creates a new status bar.
func NewStatusBar() *StatusBar {
	sb := &StatusBar{}
	sb.label = canvas.NewText("CourtDraw", color.NRGBA{R: 0xaa, G: 0xaa, B: 0xaa, A: 0xff})
	if isMobile {
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

// Status levels.
const (
	StatusInfo    = 0
	StatusError   = 1
	StatusSuccess = 2
	StatusWarning = 3
)

// SetStatus shows a status message.
// level: 0=info (gray), 1=error (red), 2=success (green), 3=warning (orange).
func (sb *StatusBar) SetStatus(msg string, level int) {
	if msg == "" {
		msg = "CourtDraw"
	}
	sb.label.Text = msg
	switch level {
	case StatusError:
		sb.label.Color = color.NRGBA{R: 0xff, G: 0x66, B: 0x66, A: 0xff}
	case StatusSuccess:
		sb.label.Color = color.NRGBA{R: 0x66, G: 0xff, B: 0x66, A: 0xff}
	case StatusWarning:
		sb.label.Color = color.NRGBA{R: 0xff, G: 0xaa, B: 0x33, A: 0xff}
	default:
		sb.label.Color = color.NRGBA{R: 0xaa, G: 0xaa, B: 0xaa, A: 0xff}
	}
	sb.label.Refresh()

	// Auto-dismiss after 3 seconds (reset timer if already running).
	sb.dismissMu.Lock()
	if sb.dismissTimer != nil {
		sb.dismissTimer.Stop()
	}
	sb.dismissTimer = time.AfterFunc(3*time.Second, func() {
		fyne.Do(func() {
			sb.label.Text = "CourtDraw"
			sb.label.Color = color.NRGBA{R: 0xaa, G: 0xaa, B: 0xaa, A: 0xff}
			sb.label.Refresh()
		})
	})
	sb.dismissMu.Unlock()
}
