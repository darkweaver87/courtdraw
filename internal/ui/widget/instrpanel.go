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
	syncedSeqIdx   int
	syncedCount    int
	syncedEditLang string
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

// ForceResync forces re-syncing editors on the next frame.
func (ip *InstructionsPanel) ForceResync() {
	ip.syncedSeqIdx = -1
	ip.syncedCount = -1
	ip.syncedEditLang = ""
}

// Layout renders the instructions panel.
func (ip *InstructionsPanel) Layout(gtx layout.Context, th *material.Theme, seq *model.Sequence, seqIdx int, state *editor.EditorState, exercise *model.Exercise, editLang string) layout.Dimensions {
	if seq == nil {
		return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 0)}
	}

	panelHeight := gtx.Dp(unit.Dp(120))
	gtx.Constraints.Max.Y = panelHeight
	gtx.Constraints.Min.Y = panelHeight

	// Background.
	bg := color.NRGBA{R: 0x2a, G: 0x2a, B: 0x2a, A: 0xff}
	paint.FillShape(gtx.Ops, bg, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, panelHeight)}.Op())

	// Resolve instructions slice based on editLang.
	instrs := ip.resolveInstructions(exercise, seqIdx, editLang)

	// Sync editors when sequence or language changes.
	needSync := seqIdx != ip.syncedSeqIdx || len(instrs) != ip.syncedCount || editLang != ip.syncedEditLang
	if needSync {
		newCount := len(instrs)
		if newCount > maxInstructions {
			newCount = maxInstructions
		}
		for i := 0; i < newCount; i++ {
			ip.editors[i].SetText(instrs[i])
		}
		// Clear stale editors beyond new count.
		oldCount := ip.syncedCount
		if oldCount < 0 {
			oldCount = maxInstructions
		} else if oldCount > maxInstructions {
			oldCount = maxInstructions
		}
		for i := newCount; i < oldCount; i++ {
			ip.editors[i].SetText("")
		}
		ip.syncedSeqIdx = seqIdx
		ip.syncedCount = len(instrs)
		ip.syncedEditLang = editLang
	}

	// Handle editor changes.
	for i := 0; i < len(instrs) && i < maxInstructions; i++ {
		for {
			evt, ok := ip.editors[i].Update(gtx)
			if !ok {
				break
			}
			if _, isChange := evt.(widget.ChangeEvent); isChange {
				ip.setInstruction(exercise, seqIdx, editLang, i, ip.editors[i].Text())
				instrs = ip.resolveInstructions(exercise, seqIdx, editLang)
				state.MarkModified()
			}
		}
	}

	// Handle delete clicks.
	for i := 0; i < len(instrs) && i < maxInstructions; i++ {
		if ip.delClicks[i].Clicked(gtx) {
			ip.deleteInstruction(exercise, seqIdx, editLang, i)
			ip.syncedCount = -1 // force resync
			state.MarkModified()
			break
		}
	}

	// Handle add click.
	if ip.addClick.Clicked(gtx) {
		instrs = ip.resolveInstructions(exercise, seqIdx, editLang)
		if len(instrs) < maxInstructions {
			ip.addInstruction(exercise, seqIdx, editLang)
			ip.syncedCount = -1 // force resync
			state.MarkModified()
		}
	}

	instrs = ip.resolveInstructions(exercise, seqIdx, editLang)
	numItems := len(instrs) + 2 // header + instructions + add button

	return material.List(th, &ip.scrollList).Layout(gtx, numItems, func(gtx layout.Context, idx int) layout.Dimensions {
		if idx == 0 {
			return ip.layoutHeader(gtx, th)
		}
		if idx <= len(instrs) {
			instrIdx := idx - 1
			return ip.layoutInstruction(gtx, th, instrIdx)
		}
		return ip.layoutAddButton(gtx, th)
	})
}

// resolveInstructions returns the instructions slice for the given language.
func (ip *InstructionsPanel) resolveInstructions(ex *model.Exercise, seqIdx int, editLang string) []string {
	if ex == nil || seqIdx >= len(ex.Sequences) {
		return nil
	}
	if editLang == "en" {
		return ex.Sequences[seqIdx].Instructions
	}
	tr := ex.EnsureI18n(editLang)
	if seqIdx < len(tr.Sequences) {
		return tr.Sequences[seqIdx].Instructions
	}
	return nil
}

// setInstruction writes a single instruction at the given index for the given language.
func (ip *InstructionsPanel) setInstruction(ex *model.Exercise, seqIdx int, editLang string, instrIdx int, text string) {
	if ex == nil || seqIdx >= len(ex.Sequences) {
		return
	}
	if editLang == "en" {
		if instrIdx < len(ex.Sequences[seqIdx].Instructions) {
			ex.Sequences[seqIdx].Instructions[instrIdx] = text
		}
		return
	}
	tr := ex.EnsureI18n(editLang)
	ip.ensureI18nSeq(&tr, seqIdx)
	if instrIdx < len(tr.Sequences[seqIdx].Instructions) {
		tr.Sequences[seqIdx].Instructions[instrIdx] = text
	}
	ex.SetI18n(editLang, tr)
}

// deleteInstruction removes an instruction at the given index for the given language.
func (ip *InstructionsPanel) deleteInstruction(ex *model.Exercise, seqIdx int, editLang string, instrIdx int) {
	if ex == nil || seqIdx >= len(ex.Sequences) {
		return
	}
	if editLang == "en" {
		seq := &ex.Sequences[seqIdx]
		if instrIdx < len(seq.Instructions) {
			seq.Instructions = append(seq.Instructions[:instrIdx], seq.Instructions[instrIdx+1:]...)
		}
		return
	}
	tr := ex.EnsureI18n(editLang)
	ip.ensureI18nSeq(&tr, seqIdx)
	s := &tr.Sequences[seqIdx]
	if instrIdx < len(s.Instructions) {
		s.Instructions = append(s.Instructions[:instrIdx], s.Instructions[instrIdx+1:]...)
	}
	ex.SetI18n(editLang, tr)
}

// addInstruction appends a blank instruction for the given language.
func (ip *InstructionsPanel) addInstruction(ex *model.Exercise, seqIdx int, editLang string) {
	if ex == nil || seqIdx >= len(ex.Sequences) {
		return
	}
	if editLang == "en" {
		ex.Sequences[seqIdx].Instructions = append(ex.Sequences[seqIdx].Instructions, "")
		return
	}
	tr := ex.EnsureI18n(editLang)
	ip.ensureI18nSeq(&tr, seqIdx)
	tr.Sequences[seqIdx].Instructions = append(tr.Sequences[seqIdx].Instructions, "")
	ex.SetI18n(editLang, tr)
}

// ensureI18nSeq ensures the i18n translation has enough sequence entries.
func (ip *InstructionsPanel) ensureI18nSeq(tr *model.ExerciseI18n, seqIdx int) {
	for len(tr.Sequences) <= seqIdx {
		tr.Sequences = append(tr.Sequences, model.SequenceI18n{})
	}
}

func (ip *InstructionsPanel) layoutHeader(gtx layout.Context, th *material.Theme) layout.Dimensions {
	return layoutSectionTitle(gtx, th, i18n.T("instr.header"))
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
