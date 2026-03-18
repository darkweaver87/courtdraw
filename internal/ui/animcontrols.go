package ui

import (
	"fmt"
	"image/color"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/anim"
	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/ui/icon"
)

// AnimControls provides playback control buttons.
type AnimControls struct {
	playBtn  *TipButton
	pauseBtn *TipButton
	stopBtn  *TipButton
	prevBtn  *TipButton
	nextBtn  *TipButton
	speedBtn *widget.Button
	seqLabel *canvas.Text     // desktop: "2 / 4"
	seqDots  *fyne.Container  // mobile: dot pills indicator (nil on desktop)
	box      *fyne.Container

	playback *anim.Playback
	numSeqs  int

	OnStateChanged func()
}

// NewAnimControls creates animation controls.
func NewAnimControls() *AnimControls {
	ac := &AnimControls{}

	ac.prevBtn = NewTipButton(icon.Prev(), i18n.T("tooltip.prev"), func() {
		if ac.playback != nil {
			ac.playback.PrevSeq()
			ac.notify()
			ac.Refresh()
		}
	})
	ac.playBtn = NewTipButton(icon.Play(), i18n.T("tooltip.play"), func() {
		if ac.playback != nil {
			ac.playback.Play()
			ac.notify()
			ac.Refresh()
		}
	})
	ac.pauseBtn = NewTipButton(icon.Pause(), i18n.T("tooltip.pause"), func() {
		if ac.playback != nil {
			ac.playback.Pause()
			ac.notify()
			ac.Refresh()
		}
	})
	ac.stopBtn = NewTipButton(icon.Stop(), i18n.T("tooltip.stop"), func() {
		if ac.playback != nil {
			ac.playback.Stop()
			ac.notify()
			ac.Refresh()
		}
	})
	ac.nextBtn = NewTipButton(icon.Next(), i18n.T("tooltip.next"), func() {
		if ac.playback != nil {
			ac.playback.NextSeq()
			ac.notify()
			ac.Refresh()
		}
	})
	// Tooltips above since these controls are at the bottom of the UI.
	ac.prevBtn.TooltipAbove = true
	ac.playBtn.TooltipAbove = true
	ac.pauseBtn.TooltipAbove = true
	ac.stopBtn.TooltipAbove = true
	ac.nextBtn.TooltipAbove = true

	ac.speedBtn = widget.NewButton("1.0x", func() {
		if ac.playback != nil {
			ac.playback.CycleSpeed()
			ac.Refresh()
		}
	})
	ac.speedBtn.Importance = widget.LowImportance

	ac.seqLabel = canvas.NewText("", color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff})
	ac.seqLabel.TextSize = 11

	ac.pauseBtn.Hide()

	bg := canvas.NewRectangle(color.NRGBA{R: 0x28, G: 0x28, B: 0x28, A: 0xff})
	var seqIndicator fyne.CanvasObject
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		ac.seqDots = container.NewHBox()
		seqIndicator = ac.seqDots
	} else {
		seqIndicator = ac.seqLabel
	}
	buttons := container.NewHBox(ac.prevBtn, ac.playBtn, ac.pauseBtn, ac.stopBtn, ac.nextBtn, ac.speedBtn, seqIndicator, layout.NewSpacer())
	ac.box = container.NewStack(bg, buttons)
	ac.box.Hide()
	return ac
}

// Widget returns the animation controls widget.
func (ac *AnimControls) Widget() fyne.CanvasObject {
	return ac.box
}

// SetPlayback sets the playback engine.
func (ac *AnimControls) SetPlayback(pb *anim.Playback, numSeqs int) {
	ac.playback = pb
	ac.numSeqs = numSeqs
	if pb != nil {
		ac.box.Show()
	} else {
		ac.box.Hide()
	}
	ac.Refresh()
}

// Refresh updates button states based on playback state.
func (ac *AnimControls) Refresh() {
	if ac.playback == nil {
		return
	}
	state := ac.playback.State()
	if state == anim.StatePlaying {
		ac.playBtn.Hide()
		ac.pauseBtn.Show()
	} else {
		ac.pauseBtn.Hide()
		ac.playBtn.Show()
	}
	ac.speedBtn.SetText(fmt.Sprintf("%.1fx", ac.playback.Speed()))
	if ac.seqDots != nil {
		ac.refreshSeqDots(ac.playback.SeqIndex(), ac.numSeqs)
	} else {
		ac.seqLabel.Text = fmt.Sprintf("%d / %d", ac.playback.SeqIndex()+1, ac.numSeqs)
		ac.seqLabel.Refresh()
	}
}

// refreshSeqDots rebuilds the sequence dot pills indicator.
func (ac *AnimControls) refreshSeqDots(current, total int) {
	ac.seqDots.RemoveAll()
	dotActive := color.NRGBA{R: 0x29, G: 0x6d, B: 0xd4, A: 0xff}
	dotInactive := color.NRGBA{R: 0x66, G: 0x66, B: 0x66, A: 0xff}
	for i := 0; i < total; i++ {
		c := dotInactive
		if i == current {
			c = dotActive
		}
		dot := canvas.NewCircle(c)
		dot.Resize(fyne.NewSize(8, 8))
		dotWrap := container.NewGridWrap(fyne.NewSize(8, 8), dot)
		ac.seqDots.Add(dotWrap)
	}
	ac.seqDots.Refresh()
}

// RefreshLanguage updates tooltip text for the current language.
func (ac *AnimControls) RefreshLanguage() {
	ac.prevBtn.SetTooltip(i18n.T("tooltip.prev"))
	ac.playBtn.SetTooltip(i18n.T("tooltip.play"))
	ac.pauseBtn.SetTooltip(i18n.T("tooltip.pause"))
	ac.stopBtn.SetTooltip(i18n.T("tooltip.stop"))
	ac.nextBtn.SetTooltip(i18n.T("tooltip.next"))
}

func (ac *AnimControls) notify() {
	if ac.OnStateChanged != nil {
		ac.OnStateChanged()
	}
}
