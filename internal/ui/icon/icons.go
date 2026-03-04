package icon

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
	icons "golang.org/x/exp/shiny/materialdesign/icons"
)

// Material Design icons used across the UI.
var (
	New       = mustIcon(icons.ActionNoteAdd)
	Open      = mustIcon(icons.FileFolderOpen)
	Save      = mustIcon(icons.ContentSave)
	Duplicate = mustIcon(icons.ContentContentCopy)
	Import    = mustIcon(icons.FileFileDownload)
	Play      = mustIcon(icons.AVPlayArrow)
	Pause     = mustIcon(icons.AVPause)
	Stop      = mustIcon(icons.AVStop)
	Prev      = mustIcon(icons.AVSkipPrevious)
	Next      = mustIcon(icons.AVSkipNext)
	Select    = mustIcon(icons.ActionTouchApp)
	Delete    = mustIcon(icons.ActionDelete)
	Close     = mustIcon(icons.NavigationClose)
	Add       = mustIcon(icons.ContentAdd)
	PDF       = mustIcon(icons.ActionDescription)
	Language  = mustIcon(icons.ActionLanguage)
	Refresh   = mustIcon(icons.NavigationRefresh)
	Upload    = mustIcon(icons.FileFileUpload)
	Sync      = mustIcon(icons.NotificationSync)
	Today     = mustIcon(icons.ActionToday)
	Calendar  = mustIcon(icons.ActionDateRange)
	Recent   = mustIcon(icons.ActionHistory)
	MoveUp     = mustIcon(icons.NavigationArrowUpward)
	MoveDown   = mustIcon(icons.NavigationArrowDownward)
	DragHandle = mustIcon(icons.EditorDragHandle)
)

func mustIcon(data []byte) *widget.Icon {
	ic, err := widget.NewIcon(data)
	if err != nil {
		panic(err)
	}
	return ic
}

// IconSize is the default icon size in dp.
const IconSize = unit.Dp(18)

// LayoutIcon renders an icon at IconSize with the given color.
func LayoutIcon(gtx layout.Context, ic *widget.Icon, col color.NRGBA) layout.Dimensions {
	sz := gtx.Dp(IconSize)
	gtx.Constraints = layout.Exact(image.Pt(sz, sz))
	return ic.Layout(gtx, col)
}

// IconBtn renders a clickable icon button with padding.
func IconBtn(gtx layout.Context, click *widget.Clickable, ic *widget.Icon, col color.NRGBA) layout.Dimensions {
	return material.Clickable(gtx, click, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{
			Top: unit.Dp(4), Bottom: unit.Dp(4),
			Left: unit.Dp(6), Right: unit.Dp(6),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return LayoutIcon(gtx, ic, col)
		})
	})
}

// tooltipBg is the background color for tooltip labels.
var tooltipBg = color.NRGBA{R: 0x1a, G: 0x1a, B: 0x1a, A: 0xee}

// tooltipFg is the foreground (text) color for tooltip labels.
var tooltipFg = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}

// IconBtnTooltip renders a clickable icon button with a tooltip shown on hover.
func IconBtnTooltip(gtx layout.Context, th *material.Theme, click *widget.Clickable, ic *widget.Icon, col color.NRGBA, tooltip string) layout.Dimensions {
	dims := IconBtn(gtx, click, ic, col)
	if click.Hovered() && tooltip != "" {
		drawTooltip(gtx, th, dims, tooltip)
	}
	return dims
}

// drawTooltip draws a small tooltip label below the given parent dimensions.
// Uses op.Defer so the tooltip renders on top of all other content.
func drawTooltip(gtx layout.Context, th *material.Theme, parent layout.Dimensions, text string) {
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

	// Center the tooltip below the parent widget.
	offsetX := (parent.Size.X - tipW) / 2
	offsetY := parent.Size.Y + gtx.Dp(unit.Dp(4))

	// Record tooltip drawing in a macro, then defer it to render on top.
	macro := op.Record(gtx.Ops)

	st1 := op.Offset(image.Pt(offsetX, offsetY)).Push(gtx.Ops)

	r := gtx.Dp(unit.Dp(3))
	bounds := image.Rect(0, 0, tipW, tipH)
	paint.FillShape(gtx.Ops, tooltipBg,
		clip.RRect{Rect: bounds, NE: r, NW: r, SE: r, SW: r}.Op(gtx.Ops))

	st2 := op.Offset(image.Pt(padH, padV)).Push(gtx.Ops)
	lbl.Layout(cgtx)
	st2.Pop()

	st1.Pop()

	call := macro.Stop()
	op.Defer(gtx.Ops, call)
}

// IconTextBtn renders a clickable button with icon + text label.
func IconTextBtn(gtx layout.Context, th *material.Theme, click *widget.Clickable, ic *widget.Icon, label string, col color.NRGBA) layout.Dimensions {
	return material.Clickable(gtx, click, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{
			Top: unit.Dp(4), Bottom: unit.Dp(4),
			Left: unit.Dp(6), Right: unit.Dp(6),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return LayoutIcon(gtx, ic, col)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Left: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(12), label)
						lbl.Color = col
						return lbl.Layout(gtx)
					})
				}),
			)
		})
	})
}
