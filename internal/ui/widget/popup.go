package widget

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

const maxPopupOptions = 20

// PopupOption is a single item in a PopupSelector.
type PopupOption struct {
	Key   string
	Label string
}

// PopupSelector is a floating dropdown list for selecting from options.
type PopupSelector struct {
	Visible      bool
	Options      []PopupOption
	Selected     string
	optClicks    [maxPopupOptions]widget.Clickable
	dismissClick widget.Clickable
	scrollList   widget.List
}

// Show opens the popup with the given options.
func (ps *PopupSelector) Show(options []PopupOption) {
	ps.Options = options
	ps.Visible = true
	ps.Selected = ""
	ps.scrollList.Axis = layout.Vertical
}

// Update processes popup clicks from the previous frame.
// Call this at the top level of Layout every frame.
// Returns (key, true) when an option was selected, ("", false) otherwise.
func (ps *PopupSelector) Update(gtx layout.Context) (string, bool) {
	if !ps.Visible {
		return "", false
	}
	if ps.dismissClick.Clicked(gtx) {
		ps.Visible = false
		return "", false
	}
	for i := 0; i < len(ps.Options) && i < maxPopupOptions; i++ {
		if ps.optClicks[i].Clicked(gtx) {
			key := ps.Options[i].Key
			ps.Selected = key
			ps.Visible = false
			return key, true
		}
	}
	return "", false
}

// LayoutBelow renders the popup positioned below a trigger element using op.Defer.
// Call this from within the trigger's layout callback, right after laying out the trigger.
// triggerDims is the layout.Dimensions of the trigger element.
func (ps *PopupSelector) LayoutBelow(gtx layout.Context, th *material.Theme, triggerDims layout.Dimensions) {
	if !ps.Visible {
		return
	}

	macro := op.Record(gtx.Ops)

	// Dismiss overlay — large clickable area behind the popup.
	bigSize := 8000
	dismissOff := op.Offset(image.Pt(-bigSize/2, -bigSize/2)).Push(gtx.Ops)
	dgtx := gtx
	dgtx.Constraints = layout.Exact(image.Pt(bigSize, bigSize))
	material.Clickable(dgtx, &ps.dismissClick, func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, color.NRGBA{A: 0x22},
			clip.Rect{Max: image.Pt(bigSize, bigSize)}.Op())
		return layout.Dimensions{Size: image.Pt(bigSize, bigSize)}
	})
	dismissOff.Pop()

	// Position popup below trigger.
	gap := gtx.Dp(unit.Dp(2))
	offsetY := triggerDims.Size.Y + gap
	st := op.Offset(image.Pt(0, offsetY)).Push(gtx.Ops)

	panelW := triggerDims.Size.X
	minW := gtx.Dp(unit.Dp(180))
	if panelW < minW {
		panelW = minW
	}
	count := len(ps.Options)
	if count > maxPopupOptions {
		count = maxPopupOptions
	}
	rowH := gtx.Dp(unit.Dp(28))
	panelH := rowH * count
	maxH := gtx.Dp(unit.Dp(300))
	if panelH > maxH {
		panelH = maxH
	}

	// Panel background.
	panelBg := color.NRGBA{R: 0x38, G: 0x38, B: 0x38, A: 0xff}
	r := gtx.Dp(unit.Dp(4))
	bounds := image.Rect(0, 0, panelW, panelH)
	paint.FillShape(gtx.Ops, panelBg, clip.RRect{Rect: bounds, NE: r, NW: r, SE: r, SW: r}.Op(gtx.Ops))

	cgtx := gtx
	cgtx.Constraints = layout.Exact(image.Pt(panelW, panelH))

	material.List(th, &ps.scrollList).Layout(cgtx, count, func(gtx layout.Context, idx int) layout.Dimensions {
		opt := ps.Options[idx]
		return material.Clickable(gtx, &ps.optClicks[idx], func(gtx layout.Context) layout.Dimensions {
			bgCol := color.NRGBA{R: 0x38, G: 0x38, B: 0x38, A: 0xff}
			if ps.optClicks[idx].Hovered() {
				bgCol = color.NRGBA{R: 0x50, G: 0x50, B: 0x50, A: 0xff}
			}
			h := gtx.Dp(unit.Dp(28))
			paint.FillShape(gtx.Ops, bgCol, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, h)}.Op())
			return layout.Inset{
				Top: unit.Dp(4), Bottom: unit.Dp(4),
				Left: unit.Dp(12), Right: unit.Dp(12),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, unit.Sp(13), opt.Label)
				lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
				return lbl.Layout(gtx)
			})
		})
	})

	st.Pop()

	call := macro.Stop()
	op.Defer(gtx.Ops, call)
}
