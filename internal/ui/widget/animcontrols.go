package widget

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/darkweaver87/courtdraw/internal/anim"
	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/ui/icon"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
)

// AnimControls provides playback control buttons for animation.
type AnimControls struct {
	playClick  widget.Clickable
	pauseClick widget.Clickable
	stopClick  widget.Clickable
	prevClick  widget.Clickable
	nextClick  widget.Clickable
	speedClick widget.Clickable
}

// Layout renders the animation controls bar. numSeqs is the total sequence count for display.
func (ac *AnimControls) Layout(gtx layout.Context, th *material.Theme, pb *anim.Playback, numSeqs int) layout.Dimensions {
	if pb == nil {
		return layout.Dimensions{}
	}

	// Handle clicks.
	if ac.playClick.Clicked(gtx) {
		pb.Play()
		gtx.Execute(op.InvalidateCmd{})
	}
	if ac.pauseClick.Clicked(gtx) {
		pb.Pause()
		gtx.Execute(op.InvalidateCmd{})
	}
	if ac.stopClick.Clicked(gtx) {
		pb.Stop()
		gtx.Execute(op.InvalidateCmd{})
	}
	if ac.prevClick.Clicked(gtx) {
		pb.PrevSeq()
		gtx.Execute(op.InvalidateCmd{})
	}
	if ac.nextClick.Clicked(gtx) {
		pb.NextSeq()
		gtx.Execute(op.InvalidateCmd{})
	}
	if ac.speedClick.Clicked(gtx) {
		pb.CycleSpeed()
	}

	barH := gtx.Dp(unit.Dp(32))
	bg := color.NRGBA{R: 0x28, G: 0x28, B: 0x28, A: 0xff}
	paint.FillShape(gtx.Ops, bg, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, barH)}.Op())

	state := pb.State()

	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return icon.IconBtnTooltip(gtx, th, &ac.prevClick, icon.Prev, theme.ColorTabText, i18n.T("tooltip.prev"))
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if state == anim.StatePlaying {
				return icon.IconBtnTooltip(gtx, th, &ac.pauseClick, icon.Pause, theme.ColorCoach, i18n.T("tooltip.pause"))
			}
			return icon.IconBtnTooltip(gtx, th, &ac.playClick, icon.Play, theme.ColorAttack, i18n.T("tooltip.play"))
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return icon.IconBtnTooltip(gtx, th, &ac.stopClick, icon.Stop, theme.ColorTabText, i18n.T("tooltip.stop"))
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return icon.IconBtnTooltip(gtx, th, &ac.nextClick, icon.Next, theme.ColorTabText, i18n.T("tooltip.next"))
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				label := fmt.Sprintf("%.1fx", pb.Speed())
				return ac.btnTooltip(gtx, th, &ac.speedClick, label, i18n.T("tooltip.speed"))
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				label := i18n.Tf("anim.seq_format", pb.SeqIndex()+1, numSeqs)
				lbl := material.Label(th, unit.Sp(11), label)
				lbl.Color = theme.ColorTabText
				return lbl.Layout(gtx)
			})
		}),
	)
}

func (ac *AnimControls) btn(gtx layout.Context, th *material.Theme, click *widget.Clickable, label string) layout.Dimensions {
	return ac.btnColor(gtx, th, click, label, theme.ColorTabText)
}

func (ac *AnimControls) btnColor(gtx layout.Context, th *material.Theme, click *widget.Clickable, label string, col color.NRGBA) layout.Dimensions {
	return material.Clickable(gtx, click, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{
			Top: unit.Dp(4), Bottom: unit.Dp(4),
			Left: unit.Dp(8), Right: unit.Dp(8),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, unit.Sp(12), label)
			lbl.Color = col
			return lbl.Layout(gtx)
		})
	})
}

func (ac *AnimControls) btnTooltip(gtx layout.Context, th *material.Theme, click *widget.Clickable, label string, tooltip string) layout.Dimensions {
	dims := ac.btn(gtx, th, click, label)
	if click.Hovered() && tooltip != "" {
		LayoutTooltip(gtx, th, true, dims, tooltip)
	}
	return dims
}
