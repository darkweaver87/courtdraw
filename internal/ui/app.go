package ui

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/font"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/store"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
	uiwidget "github.com/darkweaver87/courtdraw/internal/ui/widget"
)

// App is the main application state.
type App struct {
	theme   *material.Theme
	store   store.Store
	court   uiwidget.CourtWidget
	exercise *model.Exercise

	activeTab     int
	tabClickables [2]widget.Clickable
}

// NewApp creates a new App instance.
func NewApp(th *material.Theme, st store.Store) *App {
	return &App{
		theme: th,
		store: st,
	}
}

// SetExercise sets the current exercise.
func (a *App) SetExercise(ex *model.Exercise) {
	a.exercise = ex
	a.court.SetExercise(ex)
}

// LoadFirstExercise attempts to load the first available exercise.
func (a *App) LoadFirstExercise() error {
	names, err := a.store.ListExercises()
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return nil
	}
	ex, err := a.store.LoadExercise(names[0])
	if err != nil {
		return err
	}
	a.SetExercise(ex)
	return nil
}

// Layout renders the full application UI.
func (a *App) Layout(gtx layout.Context) layout.Dimensions {
	// handle tab clicks
	for i := range a.tabClickables {
		if a.tabClickables[i].Clicked(gtx) {
			a.activeTab = i
		}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.layoutTabBar(gtx)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.layoutContent(gtx)
		}),
	)
}

func (a *App) layoutTabBar(gtx layout.Context) layout.Dimensions {
	barHeight := gtx.Dp(unit.Dp(theme.TabBarHeight))

	// background
	rect := clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, barHeight)}.Op()
	paint.FillShape(gtx.Ops, theme.ColorDarkBg, rect)

	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		// app name
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(12), Right: unit.Dp(20)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(a.theme, unit.Sp(16), "CourtDraw")
					lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
					lbl.Font.Weight = font.Bold
					return lbl.Layout(gtx)
				},
			)
		}),
		// exercise editor tab
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.layoutTab(gtx, 0, "Exercise Editor")
		}),
		// session composer tab
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.layoutTab(gtx, 1, "Session Composer")
		}),
	)
}

func (a *App) layoutTab(gtx layout.Context, index int, title string) layout.Dimensions {
	return material.Clickable(gtx, &a.tabClickables[index], func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{
			Top: unit.Dp(8), Bottom: unit.Dp(8),
			Left: unit.Dp(16), Right: unit.Dp(16),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			col := theme.ColorTabText
			if a.activeTab == index {
				col = theme.ColorTabActive
			}
			lbl := material.Label(a.theme, unit.Sp(14), title)
			lbl.Color = col
			return lbl.Layout(gtx)
		})
	})
}

func (a *App) layoutContent(gtx layout.Context) layout.Dimensions {
	switch a.activeTab {
	case 0:
		return a.layoutExerciseEditor(gtx)
	case 1:
		return a.layoutSessionComposer(gtx)
	default:
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}

func (a *App) layoutExerciseEditor(gtx layout.Context) layout.Dimensions {
	if a.exercise == nil {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(a.theme, unit.Sp(18), "No exercise loaded")
			lbl.Color = theme.ColorTabText
			return lbl.Layout(gtx)
		})
	}
	return a.court.Layout(gtx, a.theme)
}

func (a *App) layoutSessionComposer(gtx layout.Context) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		lbl := material.Label(a.theme, unit.Sp(18), "Session Composer (coming soon)")
		lbl.Color = theme.ColorTabText
		return lbl.Layout(gtx)
	})
}
