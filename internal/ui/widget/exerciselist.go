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

const maxExerciseListItems = 50

// ExerciseListOverlay is a modal overlay for picking an exercise to open.
type ExerciseListOverlay struct {
	Visible    bool
	names      []string
	itemClicks [maxExerciseListItems]widget.Clickable
	closeClick widget.Clickable
	scrollList widget.List
}

// NewExerciseListOverlay creates an initialized overlay.
func NewExerciseListOverlay() *ExerciseListOverlay {
	elo := &ExerciseListOverlay{}
	elo.scrollList.Axis = layout.Vertical
	return elo
}

// Show makes the overlay visible with the given exercise names.
func (elo *ExerciseListOverlay) Show(names []string) {
	elo.names = names
	elo.Visible = true
}

// Hide closes the overlay.
func (elo *ExerciseListOverlay) Hide() {
	elo.Visible = false
}

// Layout renders the overlay and returns the selected exercise name, if any.
func (elo *ExerciseListOverlay) Layout(gtx layout.Context, th *material.Theme) (layout.Dimensions, string) {
	if !elo.Visible {
		return layout.Dimensions{Size: gtx.Constraints.Max}, ""
	}

	selected := ""

	// Handle clicks.
	for i := 0; i < len(elo.names) && i < maxExerciseListItems; i++ {
		if elo.itemClicks[i].Clicked(gtx) {
			selected = elo.names[i]
			elo.Hide()
		}
	}
	if elo.closeClick.Clicked(gtx) {
		elo.Hide()
	}

	// Dim background and block pointer events from passing through.
	dimBg := color.NRGBA{A: 0xaa}
	area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	paint.FillShape(gtx.Ops, dimBg, clip.Rect{Max: gtx.Constraints.Max}.Op())
	event.Op(gtx.Ops, elo)
	area.Pop()

	// Centered panel.
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		panelW := gtx.Dp(unit.Dp(300))
		panelH := gtx.Dp(unit.Dp(400))
		gtx.Constraints = layout.Exact(image.Pt(panelW, panelH))

		// Panel background.
		panelBg := color.NRGBA{R: 0x35, G: 0x35, B: 0x35, A: 0xff}
		paint.FillShape(gtx.Ops, panelBg, clip.Rect{Max: image.Pt(panelW, panelH)}.Op())

		dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			// Header.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(8), Bottom: unit.Dp(4)}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								lbl := material.Label(th, unit.Sp(14), i18n.T("overlay.open_exercise"))
								lbl.Color = theme.ColorTabActive
								return lbl.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return icon.IconBtn(gtx, &elo.closeClick, icon.Close, color.NRGBA{R: 0xff, G: 0x60, B: 0x60, A: 0xff})
							}),
						)
					},
				)
			}),
			// List.
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				count := len(elo.names)
				if count > maxExerciseListItems {
					count = maxExerciseListItems
				}
				if count == 0 {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(12), i18n.T("overlay.no_exercises"))
						lbl.Color = theme.ColorTabText
						return lbl.Layout(gtx)
					})
				}
				return material.List(th, &elo.scrollList).Layout(gtx, count, func(gtx layout.Context, idx int) layout.Dimensions {
					return material.Clickable(gtx, &elo.itemClicks[idx], func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{
							Top: unit.Dp(4), Bottom: unit.Dp(4),
							Left: unit.Dp(12), Right: unit.Dp(12),
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(th, unit.Sp(13), elo.names[idx])
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
