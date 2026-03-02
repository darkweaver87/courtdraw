package widget

import (
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

const maxInstructions = 30

// InstructionsPanel is the bottom panel for editing sequence instructions.
type InstructionsPanel struct {
	editors    [maxInstructions]widget.Editor
	delClicks  [maxInstructions]widget.Clickable
	addClick   widget.Clickable
	scrollList widget.List

	// Track sync to avoid overwriting user edits.
	syncedSeqIdx int
	syncedCount  int
}

// NewInstructionsPanel creates an initialized instructions panel.
func NewInstructionsPanel() *InstructionsPanel {
	ip := &InstructionsPanel{
		syncedSeqIdx: -1,
	}
	ip.scrollList.Axis = layout.Vertical
	for i := range ip.editors {
		ip.editors[i].SingleLine = true
	}
	return ip
}

// Layout renders the instructions panel.
func (ip *InstructionsPanel) Layout(gtx layout.Context, th *material.Theme, seq *model.Sequence, seqIdx int, state *editor.EditorState) layout.Dimensions {
	if seq == nil {
		return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 0)}
	}

	panelHeight := gtx.Dp(unit.Dp(120))
	gtx.Constraints.Max.Y = panelHeight
	gtx.Constraints.Min.Y = panelHeight

	// Background.
	bg := color.NRGBA{R: 0x2a, G: 0x2a, B: 0x2a, A: 0xff}
	paint.FillShape(gtx.Ops, bg, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, panelHeight)}.Op())

	// Sync editors when sequence changes.
	if seqIdx != ip.syncedSeqIdx || len(seq.Instructions) != ip.syncedCount {
		for i := 0; i < len(seq.Instructions) && i < maxInstructions; i++ {
			ip.editors[i].SetText(seq.Instructions[i])
		}
		ip.syncedSeqIdx = seqIdx
		ip.syncedCount = len(seq.Instructions)
	}

	// Handle editor changes.
	for i := 0; i < len(seq.Instructions) && i < maxInstructions; i++ {
		for {
			evt, ok := ip.editors[i].Update(gtx)
			if !ok {
				break
			}
			if _, isChange := evt.(widget.ChangeEvent); isChange {
				seq.Instructions[i] = ip.editors[i].Text()
				state.MarkModified()
			}
		}
	}

	// Handle delete clicks.
	for i := 0; i < len(seq.Instructions) && i < maxInstructions; i++ {
		if ip.delClicks[i].Clicked(gtx) {
			seq.Instructions = append(seq.Instructions[:i], seq.Instructions[i+1:]...)
			ip.syncedCount = -1 // force resync
			state.MarkModified()
			break
		}
	}

	// Handle add click.
	if ip.addClick.Clicked(gtx) {
		if len(seq.Instructions) < maxInstructions {
			seq.Instructions = append(seq.Instructions, "")
			ip.syncedCount = -1 // force resync
			state.MarkModified()
		}
	}

	numItems := len(seq.Instructions) + 2 // header + instructions + add button

	return material.List(th, &ip.scrollList).Layout(gtx, numItems, func(gtx layout.Context, idx int) layout.Dimensions {
		if idx == 0 {
			return ip.layoutHeader(gtx, th)
		}
		if idx <= len(seq.Instructions) {
			instrIdx := idx - 1
			return ip.layoutInstruction(gtx, th, instrIdx)
		}
		return ip.layoutAddButton(gtx, th)
	})
}

func (ip *InstructionsPanel) layoutHeader(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(4), Left: unit.Dp(8), Bottom: unit.Dp(2)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, unit.Sp(11), i18n.T("instr.header"))
			lbl.Color = theme.ColorTabText
			return lbl.Layout(gtx)
		},
	)
}

func (ip *InstructionsPanel) layoutInstruction(gtx layout.Context, th *material.Theme, idx int) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(1), Bottom: unit.Dp(1), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				// Bullet.
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, unit.Sp(11), "- ")
					lbl.Color = theme.ColorTabText
					return lbl.Layout(gtx)
				}),
				// Editor.
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					edBg := color.NRGBA{R: 0x38, G: 0x38, B: 0x38, A: 0xff}
					return layoutEditorWithBg(gtx, th, &ip.editors[idx], edBg)
				}),
				// Delete button.
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return icon.IconBtn(gtx, &ip.delClicks[idx], icon.Close, color.NRGBA{R: 0xff, G: 0x60, B: 0x60, A: 0xff})
				}),
			)
		},
	)
}

func (ip *InstructionsPanel) layoutAddButton(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(2), Left: unit.Dp(4)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return icon.IconTextBtn(gtx, th, &ip.addClick, icon.Add, i18n.T("instr.add"), theme.ColorCoach)
		},
	)
}
