package widget

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/darkweaver87/courtdraw/internal/ui/theme"
)

// DatePicker is a calendar dropdown for selecting a date.
type DatePicker struct {
	Visible  bool
	Selected time.Time // the selected date (result)
	Result   string    // set to YYYY-MM-DD when a day is picked

	viewing   time.Time // current month being displayed
	dayClicks [42]widget.Clickable
	prevClick widget.Clickable
	nextClick widget.Clickable
}

// Show opens the date picker on the given date.
// If t is zero, defaults to today.
func (dp *DatePicker) Show(t time.Time) {
	if t.IsZero() {
		t = time.Now()
	}
	dp.viewing = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local)
	dp.Selected = time.Time{}
	dp.Result = ""
	dp.Visible = true
}

// Hide closes the picker.
func (dp *DatePicker) Hide() {
	dp.Visible = false
}

// panelW/panelH are the calendar panel dimensions in dp.
const (
	calendarW = 240
	calendarH = 220
)

// LayoutDropdown renders the calendar as a dropdown panel using op.Defer.
// Call this from within the date field layout — the calendar will float below the caller.
// Returns the selected date string (YYYY-MM-DD) if one was picked, empty otherwise.
func (dp *DatePicker) LayoutDropdown(gtx layout.Context, th *material.Theme, parentDims layout.Dimensions) string {
	if !dp.Visible {
		return ""
	}

	// Handle prev/next month.
	if dp.prevClick.Clicked(gtx) {
		dp.viewing = dp.viewing.AddDate(0, -1, 0)
	}
	if dp.nextClick.Clicked(gtx) {
		dp.viewing = dp.viewing.AddDate(0, 1, 0)
	}

	// Check day clicks.
	selected := ""
	year, month, _ := dp.viewing.Date()
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	startOffset := int(firstDay.Weekday()+6) % 7 // Monday=0
	daysInMonth := daysIn(year, month)
	today := time.Now()

	for i := 0; i < 42; i++ {
		if dp.dayClicks[i].Clicked(gtx) {
			day := i - startOffset + 1
			if day >= 1 && day <= daysInMonth {
				t := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
				dp.Selected = t
				selected = t.Format("2006-01-02")
				dp.Result = selected
				dp.Hide()
			}
		}
	}

	// Record the calendar panel in a macro, deferred so it renders on top.
	macro := op.Record(gtx.Ops)

	panelW := gtx.Dp(unit.Dp(calendarW))
	panelH := gtx.Dp(unit.Dp(calendarH))

	// Position below the parent widget.
	offsetY := parentDims.Size.Y + gtx.Dp(unit.Dp(2))
	st := op.Offset(image.Pt(0, offsetY)).Push(gtx.Ops)

	cgtx := gtx
	cgtx.Constraints = layout.Exact(image.Pt(panelW, panelH))

	// Panel background with border.
	panelBg := color.NRGBA{R: 0x30, G: 0x30, B: 0x30, A: 0xff}
	borderCol := color.NRGBA{R: 0x60, G: 0x60, B: 0x60, A: 0xff}
	paint.FillShape(cgtx.Ops, borderCol, clip.Rect{Max: image.Pt(panelW, panelH)}.Op())
	paint.FillShape(cgtx.Ops, panelBg, clip.Rect{
		Min: image.Pt(1, 1),
		Max: image.Pt(panelW-1, panelH-1),
	}.Op())

	layout.Flex{Axis: layout.Vertical}.Layout(cgtx,
		// Month/year header with arrows.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(6), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return material.Clickable(gtx, &dp.prevClick, func(gtx layout.Context) layout.Dimensions {
								lbl := material.Label(th, unit.Sp(14), " < ")
								lbl.Color = theme.ColorTabActive
								return lbl.Layout(gtx)
							})
						}),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							monthStr := dp.viewing.Format("January 2006")
							lbl := material.Label(th, unit.Sp(12), monthStr)
							lbl.Color = theme.ColorTabActive
							lbl.Alignment = 1 // center
							return lbl.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return material.Clickable(gtx, &dp.nextClick, func(gtx layout.Context) layout.Dimensions {
								lbl := material.Label(th, unit.Sp(14), " > ")
								lbl.Color = theme.ColorTabActive
								return lbl.Layout(gtx)
							})
						}),
					)
				},
			)
		}),
		// Day of week headers.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(2), Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					dayNames := [7]string{"Lu", "Ma", "Me", "Je", "Ve", "Sa", "Di"}
					cellW := (panelW - gtx.Dp(unit.Dp(8))) / 7
					rowH := gtx.Dp(unit.Dp(16))
					for i := 0; i < 7; i++ {
						cgtx := gtx
						cgtx.Constraints = layout.Exact(image.Pt(cellW, rowH))
						off := op.Offset(image.Pt(i*cellW, 0)).Push(gtx.Ops)
						lbl := material.Label(th, unit.Sp(9), dayNames[i])
						lbl.Color = theme.ColorTabText
						lbl.Alignment = 1
						lbl.Layout(cgtx)
						off.Pop()
					}
					return layout.Dimensions{Size: image.Pt(panelW-gtx.Dp(unit.Dp(8)), rowH)}
				},
			)
		}),
		// Day grid (6 rows x 7 columns).
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					cellW := (panelW - gtx.Dp(unit.Dp(8))) / 7
					cellH := gtx.Dp(unit.Dp(24))

					for row := 0; row < 6; row++ {
						for col := 0; col < 7; col++ {
							idx := row*7 + col
							day := idx - startOffset + 1

							off := op.Offset(image.Pt(col*cellW, row*cellH)).Push(gtx.Ops)

							dgtx := gtx
							dgtx.Constraints = layout.Exact(image.Pt(cellW, cellH))

							if day >= 1 && day <= daysInMonth {
								isToday := year == today.Year() && month == today.Month() && day == today.Day()
								if isToday {
									todayBg := color.NRGBA{R: 0x50, G: 0x50, B: 0x50, A: 0xff}
									paint.FillShape(dgtx.Ops, todayBg,
										clip.Rect{Max: image.Pt(cellW, cellH)}.Op())
								}

								material.Clickable(dgtx, &dp.dayClicks[idx], func(gtx layout.Context) layout.Dimensions {
									lbl := material.Label(th, unit.Sp(11), fmt.Sprintf("%d", day))
									if isToday {
										lbl.Color = theme.ColorCoach
									} else {
										lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
									}
									lbl.Alignment = 1
									return layout.Center.Layout(gtx, lbl.Layout)
								})
							}

							off.Pop()
						}
					}

					return layout.Dimensions{Size: image.Pt(panelW-gtx.Dp(unit.Dp(8)), 6*cellH)}
				},
			)
		}),
	)

	st.Pop()

	call := macro.Stop()
	op.Defer(gtx.Ops, call)

	return selected
}

// daysIn returns the number of days in the given month.
func daysIn(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local).Day()
}
