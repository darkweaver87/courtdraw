package ui

import (
	"fmt"
	"image/color"

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
	seqLabel *canvas.Text
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
	buttons := container.NewHBox(ac.prevBtn, ac.playBtn, ac.pauseBtn, ac.stopBtn, ac.nextBtn, ac.speedBtn, ac.seqLabel, layout.NewSpacer())
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
	ac.seqLabel.Text = fmt.Sprintf("%d / %d", ac.playback.SeqIndex()+1, ac.numSeqs)
	ac.seqLabel.Refresh()
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
