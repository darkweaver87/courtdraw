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

	"github.com/darkweaver87/courtdraw/internal/anim"
	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
)

// PreviewCourt displays an animated preview of an exercise.
type PreviewCourt struct {
	exercise *model.Exercise
	court    CourtWidget
	playback *anim.Playback
}

// SetExercise sets the exercise to preview and starts playback.
func (pc *PreviewCourt) SetExercise(ex *model.Exercise) {
	pc.exercise = ex
	pc.court.SetExercise(ex)
	if ex != nil && len(ex.Sequences) > 1 {
		pc.playback = anim.NewPlayback(ex)
		pc.playback.Play()
	} else {
		pc.playback = nil
	}
}

// Layout renders the preview court with exercise info.
func (pc *PreviewCourt) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	bg := color.NRGBA{R: 0x2a, G: 0x2a, B: 0x2a, A: 0xff}
	paint.FillShape(gtx.Ops, bg, clip.Rect{Max: gtx.Constraints.Max}.Op())

	if pc.exercise == nil {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, unit.Sp(13), i18n.T("mgr.select_preview"))
			lbl.Color = theme.ColorTabText
			return lbl.Layout(gtx)
		})
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Exercise name.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(8), Left: unit.Dp(8), Right: unit.Dp(8), Bottom: unit.Dp(4)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, unit.Sp(14), pc.exercise.Name)
					lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
					return lbl.Layout(gtx)
				},
			)
		}),
		// Description.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if pc.exercise.Description == "" {
				return layout.Dimensions{}
			}
			return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8), Bottom: unit.Dp(4)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, unit.Sp(11), pc.exercise.Description)
					lbl.Color = theme.ColorTabText
					return lbl.Layout(gtx)
				},
			)
		}),
		// Metadata line (category + duration + intensity).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			info := ""
			if pc.exercise.Category != "" {
				info = i18n.T("category." + string(pc.exercise.Category))
			}
			if pc.exercise.Duration != "" {
				if info != "" {
					info += " · "
				}
				info += pc.exercise.Duration
			}
			info += " " + intensityDots(int(pc.exercise.Intensity))
			if info == " ○○○" {
				return layout.Dimensions{}
			}
			return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8), Bottom: unit.Dp(4)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, unit.Sp(11), info)
					lbl.Color = theme.ColorTabText
					return lbl.Layout(gtx)
				},
			)
		}),
		// Court preview (animated or static).
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(4), Right: unit.Dp(4), Bottom: unit.Dp(4)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return pc.layoutCourt(gtx, th)
				},
			)
		}),
	)
}

func (pc *PreviewCourt) layoutCourt(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if pc.exercise == nil {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}

	// Constrain to maintain aspect ratio within available space.
	size := constrainAspect(gtx.Constraints.Max, pc.exercise.CourtType)
	cgtx := gtx
	cgtx.Constraints = layout.Exact(size)

	// Center the court.
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints = layout.Exact(size)

		if pc.playback != nil {
			frame, needRedraw := pc.playback.Update()
			if needRedraw {
				gtx.Execute(op.InvalidateCmd{})
			}
			// Sync court to playback sequence.
			pc.court.SetSequence(pc.playback.SeqIndex())

			// If playback stopped, restart for looping.
			if pc.playback.State() == anim.StateStopped {
				pc.playback.Play()
			}

			return pc.court.LayoutAnimated(gtx, th, &frame)
		}

		// Static rendering (single sequence).
		return pc.court.LayoutStatic(gtx, th)
	})
}

// constrainAspect computes the largest size that fits within max while
// maintaining the court aspect ratio.
func constrainAspect(max image.Point, ct model.CourtType) image.Point {
	// Basketball court aspect ratio: 28m x 15m (full), 14m x 15m (half).
	var aspectW, aspectH float64
	if ct == model.FullCourt {
		aspectW, aspectH = 15, 28
	} else {
		aspectW, aspectH = 15, 14
	}
	aspect := aspectW / aspectH

	w := float64(max.X)
	h := float64(max.Y)

	if w/h > aspect {
		// Height-constrained.
		w = h * aspect
	} else {
		// Width-constrained.
		h = w / aspect
	}
	return image.Pt(int(w), int(h))
}
