package widget

import (
	"image"
	"image/color"

	"gioui.org/io/event"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/ui/icon"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
)

const maxLibItems = 50

// LibraryOverlay is a modal overlay for browsing and importing community exercises.
type LibraryOverlay struct {
	Visible    bool
	names      []string
	itemClicks [maxLibItems]widget.Clickable
	closeClick widget.Clickable
	scrollList widget.List

	// Selected holds the name selected for import, or "" if none.
	Selected string
}

// NewLibraryOverlay creates an initialized overlay.
func NewLibraryOverlay() *LibraryOverlay {
	lo := &LibraryOverlay{}
	lo.scrollList.Axis = layout.Vertical
	return lo
}

// Show makes the overlay visible with the given exercise names.
func (lo *LibraryOverlay) Show(names []string) {
	lo.names = names
	lo.Visible = true
	lo.Selected = ""
}

// Hide closes the overlay.
func (lo *LibraryOverlay) Hide() {
	lo.Visible = false
}

// Layout renders the overlay and returns the selected exercise name, if any.
func (lo *LibraryOverlay) Layout(gtx layout.Context, th *material.Theme) (layout.Dimensions, string) {
	if !lo.Visible {
		return layout.Dimensions{Size: gtx.Constraints.Max}, ""
	}

	selected := ""
	for i := 0; i < len(lo.names) && i < maxLibItems; i++ {
		if lo.itemClicks[i].Clicked(gtx) {
			selected = lo.names[i]
			lo.Hide()
		}
	}
	if lo.closeClick.Clicked(gtx) {
		lo.Hide()
	}

	// Dim background and block events.
	dimBg := color.NRGBA{A: 0xaa}
	area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	paint.FillShape(gtx.Ops, dimBg, clip.Rect{Max: gtx.Constraints.Max}.Op())
	event.Op(gtx.Ops, lo)
	area.Pop()

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		panelW := gtx.Dp(unit.Dp(320))
		panelH := gtx.Dp(unit.Dp(420))
		gtx.Constraints = layout.Exact(image.Pt(panelW, panelH))

		panelBg := color.NRGBA{R: 0x35, G: 0x35, B: 0x35, A: 0xff}
		paint.FillShape(gtx.Ops, panelBg, clip.Rect{Max: image.Pt(panelW, panelH)}.Op())

		dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(8), Bottom: unit.Dp(4)}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								lbl := material.Label(th, unit.Sp(14), i18n.T("overlay.import_library"))
								lbl.Color = theme.ColorTabActive
								return lbl.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return icon.IconBtn(gtx, &lo.closeClick, icon.Close, color.NRGBA{R: 0xff, G: 0x60, B: 0x60, A: 0xff})
							}),
						)
					},
				)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				count := len(lo.names)
				if count > maxLibItems {
					count = maxLibItems
				}
				if count == 0 {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(12), i18n.T("overlay.no_community"))
						lbl.Color = theme.ColorTabText
						return lbl.Layout(gtx)
					})
				}
				return material.List(th, &lo.scrollList).Layout(gtx, count, func(gtx layout.Context, idx int) layout.Dimensions {
					return material.Clickable(gtx, &lo.itemClicks[idx], func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{
							Top: unit.Dp(4), Bottom: unit.Dp(4),
							Left: unit.Dp(12), Right: unit.Dp(12),
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(th, unit.Sp(13), lo.names[idx])
							lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
							return lbl.Layout(gtx)
						})
					})
				})
			}),
		)
		return dims
	}), selected
}
