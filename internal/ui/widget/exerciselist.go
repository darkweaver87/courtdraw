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
// In open mode, it allows deleting exercises (with confirmation).
// In recent mode, it allows removing entries from the recent list.
type ExerciseListOverlay struct {
	Visible          bool
	OnDelete         string // set when an exercise is deleted (file deletion)
	OnRemove         string // set when a recent entry is removed (not file deletion)
	names            []string
	recentMode       bool
	confirmDeleteIdx int // -1 = none, >= 0 = row pending confirmation
	itemClicks       [maxExerciseListItems]widget.Clickable
	deleteClicks     [maxExerciseListItems]widget.Clickable
	confirmClicks    [maxExerciseListItems]widget.Clickable
	cancelClicks     [maxExerciseListItems]widget.Clickable
	removeClicks     [maxExerciseListItems]widget.Clickable
	closeClick       widget.Clickable
	scrollList       widget.List
}

// NewExerciseListOverlay creates an initialized overlay.
func NewExerciseListOverlay() *ExerciseListOverlay {
	elo := &ExerciseListOverlay{confirmDeleteIdx: -1}
	elo.scrollList.Axis = layout.Vertical
	return elo
}

// Show makes the overlay visible with the given exercise names (open mode).
func (elo *ExerciseListOverlay) Show(names []string) {
	elo.names = names
	elo.Visible = true
	elo.recentMode = false
	elo.OnRemove = ""
	elo.OnDelete = ""
	elo.confirmDeleteIdx = -1
}

// ShowRecent makes the overlay visible in recent mode.
func (elo *ExerciseListOverlay) ShowRecent(names []string) {
	elo.names = names
	elo.Visible = true
	elo.recentMode = true
	elo.OnRemove = ""
	elo.OnDelete = ""
	elo.confirmDeleteIdx = -1
}

// Hide closes the overlay.
func (elo *ExerciseListOverlay) Hide() {
	elo.Visible = false
	elo.confirmDeleteIdx = -1
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
			if elo.confirmDeleteIdx != i {
				selected = elo.names[i]
				elo.Hide()
			}
		}
		if elo.recentMode {
			if elo.removeClicks[i].Clicked(gtx) {
				elo.OnRemove = elo.names[i]
				elo.names = append(elo.names[:i], elo.names[i+1:]...)
				break
			}
		} else {
			if elo.deleteClicks[i].Clicked(gtx) {
				elo.confirmDeleteIdx = i
			}
			if elo.confirmDeleteIdx == i {
				if elo.confirmClicks[i].Clicked(gtx) {
					elo.OnDelete = elo.names[i]
					elo.names = append(elo.names[:i], elo.names[i+1:]...)
					elo.confirmDeleteIdx = -1
					break
				}
				if elo.cancelClicks[i].Clicked(gtx) {
					elo.confirmDeleteIdx = -1
				}
			}
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

	// Header title.
	headerKey := "overlay.open_exercise"
	if elo.recentMode {
		headerKey = "overlay.recent_exercises"
	}

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
								lbl := material.Label(th, unit.Sp(14), i18n.T(headerKey))
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
					if elo.recentMode {
						return elo.layoutRecentRow(gtx, th, idx)
					}
					if elo.confirmDeleteIdx == idx {
						return elo.layoutConfirmRow(gtx, th, idx)
					}
					return elo.layoutNormalRow(gtx, th, idx)
				})
			}),
		)
		return dims
	}), selected
}

// layoutNormalRow renders a row in open mode with name + delete button.
func (elo *ExerciseListOverlay) layoutNormalRow(gtx layout.Context, th *material.Theme, idx int) layout.Dimensions {
	return layout.Inset{
		Top: unit.Dp(2), Bottom: unit.Dp(2),
		Left: unit.Dp(12), Right: unit.Dp(8),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return material.Clickable(gtx, &elo.itemClicks[idx], func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(13), elo.names[idx])
						lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
						return lbl.Layout(gtx)
					})
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon.IconBtn(gtx, &elo.deleteClicks[idx], icon.Delete, color.NRGBA{R: 0xff, G: 0x60, B: 0x60, A: 0xff})
			}),
		)
	})
}

// layoutConfirmRow renders a confirmation row: "Confirm?" + confirm/cancel buttons.
func (elo *ExerciseListOverlay) layoutConfirmRow(gtx layout.Context, th *material.Theme, idx int) layout.Dimensions {
	confirmBg := color.NRGBA{R: 0x50, G: 0x20, B: 0x20, A: 0xff}
	return layout.Inset{
		Top: unit.Dp(2), Bottom: unit.Dp(2),
		Left: unit.Dp(12), Right: unit.Dp(8),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		paint.FillShape(gtx.Ops, confirmBg, clip.Rect{Max: gtx.Constraints.Max}.Op())
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, unit.Sp(12), i18n.T("overlay.confirm_delete"))
					lbl.Color = color.NRGBA{R: 0xff, G: 0x80, B: 0x80, A: 0xff}
					return lbl.Layout(gtx)
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon.IconBtn(gtx, &elo.confirmClicks[idx], icon.Delete, color.NRGBA{R: 0xff, G: 0x40, B: 0x40, A: 0xff})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon.IconBtn(gtx, &elo.cancelClicks[idx], icon.Close, theme.ColorTabText)
			}),
		)
	})
}

// layoutRecentRow renders a row in recent mode with a remove button.
func (elo *ExerciseListOverlay) layoutRecentRow(gtx layout.Context, th *material.Theme, idx int) layout.Dimensions {
	return layout.Inset{
		Top: unit.Dp(2), Bottom: unit.Dp(2),
		Left: unit.Dp(12), Right: unit.Dp(8),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return material.Clickable(gtx, &elo.itemClicks[idx], func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(13), elo.names[idx])
						lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
						return lbl.Layout(gtx)
					})
				})
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon.IconBtn(gtx, &elo.removeClicks[idx], icon.Close, theme.ColorTabText)
			}),
		)
	})
}
