package widget

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/ui/editor"
	"github.com/darkweaver87/courtdraw/internal/ui/icon"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
)

const maxSequences = 20

// SeqTimeline is a horizontal row of sequence tabs with an [+] add button.
type SeqTimeline struct {
	tabClicks [maxSequences]widget.Clickable
	addClick  widget.Clickable
	list      widget.List
	listInit  bool
}

// Layout renders the sequence timeline.
func (st *SeqTimeline) Layout(gtx layout.Context, th *material.Theme, exercise *model.Exercise, court *CourtWidget, state *editor.EditorState) layout.Dimensions {
	if exercise == nil {
		return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 0)}
	}

	barHeight := gtx.Dp(unit.Dp(36))

	// Background.
	bg := color.NRGBA{R: 0x2a, G: 0x2a, B: 0x2a, A: 0xff}
	paint.FillShape(gtx.Ops, bg, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, barHeight)}.Op())

	// Handle tab clicks.
	seqIdx := court.SeqIndex()
	for i := 0; i < len(exercise.Sequences) && i < maxSequences; i++ {
		if st.tabClicks[i].Clicked(gtx) {
			court.SetSequence(i)
			state.Deselect()
			state.ActionFrom = nil
		}
	}

	// Handle [+] click.
	if st.addClick.Clicked(gtx) {
		st.addSequence(exercise, court, state)
	}

	// Layout tabs in a horizontal flex.
	numItems := len(exercise.Sequences) + 1 // tabs + add button
	if !st.listInit {
		st.list.Axis = layout.Horizontal
		st.listInit = true
	}
	return layout.Inset{Left: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return material.List(th, &st.list).Layout(gtx, numItems, func(gtx layout.Context, idx int) layout.Dimensions {
			if idx < len(exercise.Sequences) {
				return st.layoutTab(gtx, th, exercise, idx, seqIdx)
			}
			return st.layoutAddButton(gtx, th)
		})
	})
}

func (st *SeqTimeline) layoutTab(gtx layout.Context, th *material.Theme, exercise *model.Exercise, idx, activeIdx int) layout.Dimensions {
	return material.Clickable(gtx, &st.tabClicks[idx], func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{
			Top: unit.Dp(6), Bottom: unit.Dp(6),
			Left: unit.Dp(8), Right: unit.Dp(8),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			// Active tab has colored background.
			if idx == activeIdx {
				sz := gtx.Dp(unit.Dp(24))
				bg := color.NRGBA{R: 0x40, G: 0x40, B: 0x70, A: 0xff}
				paint.FillShape(gtx.Ops, bg,
					clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, sz)}.Op())
			}

			label := exercise.Sequences[idx].Label
			if label == "" {
				label = i18n.Tf("seq.format", idx+1)
			} else {
				label = fmt.Sprintf("%d. %s", idx+1, label)
			}

			col := theme.ColorTabText
			if idx == activeIdx {
				col = theme.ColorTabActive
			}
			lbl := material.Label(th, unit.Sp(12), label)
			lbl.Color = col
			return lbl.Layout(gtx)
		})
	})
}

func (st *SeqTimeline) layoutAddButton(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return icon.IconBtn(gtx, &st.addClick, icon.Add, theme.ColorCoach)
}

// addSequence appends a new sequence by deep-copying player positions and accessories
// from the current sequence, with empty actions and instructions.
func (st *SeqTimeline) addSequence(exercise *model.Exercise, court *CourtWidget, state *editor.EditorState) {
	var newSeq model.Sequence
	newSeq.Label = ""

	currentIdx := court.SeqIndex()
	if currentIdx < len(exercise.Sequences) {
		current := &exercise.Sequences[currentIdx]
		// Deep-copy players.
		newSeq.Players = make([]model.Player, len(current.Players))
		copy(newSeq.Players, current.Players)
		// Deep-copy accessories.
		newSeq.Accessories = make([]model.Accessory, len(current.Accessories))
		copy(newSeq.Accessories, current.Accessories)
	}

	exercise.Sequences = append(exercise.Sequences, newSeq)
	newIdx := len(exercise.Sequences) - 1
	court.SetSequence(newIdx)
	state.Deselect()
	state.MarkModified()
}
