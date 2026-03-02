package widget

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// tooltipBg is the background color for tooltip labels.
var tooltipBg = color.NRGBA{R: 0x1a, G: 0x1a, B: 0x1a, A: 0xee}

// tooltipFg is the foreground (text) color for tooltip labels.
var tooltipFg = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}

// LayoutTooltip draws a tooltip label below the given dimensions when hovered
// is true. It uses layout.Stack so the tooltip floats above surrounding content.
// The caller should pass the hovered state from widget.Clickable.Hovered().
//
// Usage:
//
//	dims := icon.IconBtn(gtx, &click, ic, col)
//	LayoutTooltip(gtx, th, click.Hovered(), dims, "My tooltip")
//	return dims
func LayoutTooltip(gtx layout.Context, th *material.Theme, hovered bool, parent layout.Dimensions, text string) {
	if !hovered || text == "" {
		return
	}

	lbl := material.Label(th, unit.Sp(11), text)
	lbl.Color = tooltipFg

	padH := gtx.Dp(unit.Dp(6))
	padV := gtx.Dp(unit.Dp(3))

	// Measure the label in a discarded macro.
	measure := op.Record(gtx.Ops)
	cgtx := gtx
	cgtx.Constraints = layout.Constraints{
		Min: image.Point{},
		Max: image.Pt(gtx.Dp(unit.Dp(200)), gtx.Dp(unit.Dp(40))),
	}
	lblDims := lbl.Layout(cgtx)
	measure.Stop()

	tipW := lblDims.Size.X + 2*padH
	tipH := lblDims.Size.Y + 2*padV

	// Position: centered below parent, offset downward by parent height + small gap.
	offsetX := (parent.Size.X - tipW) / 2
	offsetY := parent.Size.Y + gtx.Dp(unit.Dp(4))

	// Record tooltip drawing in a macro, then defer it to render on top.
	macro := op.Record(gtx.Ops)

	st1 := op.Offset(image.Pt(offsetX, offsetY)).Push(gtx.Ops)

	// Draw background rounded rect.
	r := gtx.Dp(unit.Dp(3))
	bounds := image.Rect(0, 0, tipW, tipH)
	paint.FillShape(gtx.Ops, tooltipBg,
		clip.RRect{Rect: bounds, NE: r, NW: r, SE: r, SW: r}.Op(gtx.Ops))

	// Draw label text centered in the tooltip.
	st2 := op.Offset(image.Pt(padH, padV)).Push(gtx.Ops)
	lbl.Layout(cgtx)
	st2.Pop()

	st1.Pop()

	call := macro.Stop()
	op.Defer(gtx.Ops, call)
}
