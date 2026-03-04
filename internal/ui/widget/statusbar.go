package widget

import (
	"image"
	"image/color"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/darkweaver87/courtdraw/internal/ui/editor"
)

const statusDismissDelay = 3 * time.Second

// StatusBar displays a temporary status message at the bottom of the editor.
type StatusBar struct{}

// Layout renders the status bar. It auto-dismisses after 3 seconds.
func (sb *StatusBar) Layout(gtx layout.Context, th *material.Theme, state *editor.EditorState) layout.Dimensions {
	if state.StatusMsg == "" {
		return layout.Dimensions{}
	}
	elapsed := time.Since(state.StatusAt)
	if elapsed >= statusDismissDelay {
		return layout.Dimensions{}
	}

	// Schedule a redraw at dismiss time.
	dismissAt := state.StatusAt.Add(statusDismissDelay)
	gtx.Execute(op.InvalidateCmd{At: dismissAt})

	height := gtx.Dp(unit.Dp(28))
	width := gtx.Constraints.Max.X

	// Background color: dark red for error, dark grey for info.
	var bg color.NRGBA
	if state.StatusLevel == 1 {
		bg = color.NRGBA{R: 0x8b, G: 0x00, B: 0x00, A: 0xdd}
	} else {
		bg = color.NRGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xdd}
	}

	// Draw background.
	rect := clip.Rect{Max: image.Pt(width, height)}.Push(gtx.Ops)
	paint.ColorOp{Color: bg}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	rect.Pop()

	// Draw text.
	return layout.Stack{Alignment: layout.W}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(width, height)}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(8), Top: unit.Dp(4)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, unit.Sp(13), state.StatusMsg)
					lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
					return lbl.Layout(gtx)
				},
			)
		}),
	)
}
