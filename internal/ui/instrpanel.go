package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/ui/editor"
)

const maxInstructions = 30

// InstructionsPanel is the bottom panel for editing sequence instructions.
type InstructionsPanel struct {
	box     *fyne.Container
	entries []*widget.Entry
	vbox    *fyne.Container
	addBtn  *widget.Button
	header  *canvas.Text

	syncedSeqIdx   int
	syncedCount    int
	syncedEditLang string

	exercise *model.Exercise
	state    *editor.EditorState
	seqIdx   int
	editLang string

	OnModified func()
}

// NewInstructionsPanel creates a new instructions panel.
func NewInstructionsPanel() *InstructionsPanel {
	ip := &InstructionsPanel{
		syncedSeqIdx: -1,
	}
	ip.vbox = container.NewVBox()
	ip.addBtn = widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		ip.addInstruction()
	})
	ip.addBtn.Importance = widget.LowImportance

	ip.header = canvas.NewText(i18n.T(i18n.KeyInstrHeader), color.NRGBA{R: 0xf4, G: 0xa2, B: 0x61, A: 0xff})
	ip.header.TextSize = 11
	ip.header.TextStyle.Bold = true

	content := container.NewVBox(container.NewPadded(ip.header), ip.vbox, ip.addBtn)
	bg := canvas.NewRectangle(color.NRGBA{R: 0x2a, G: 0x2a, B: 0x2a, A: 0xff})
	scroll := container.NewVScroll(content)
	scroll.SetMinSize(fyne.NewSize(0, 120))
	ip.box = container.NewStack(bg, scroll)
	return ip
}

// Widget returns the instructions panel widget.
func (ip *InstructionsPanel) Widget() fyne.CanvasObject {
	return ip.box
}

// Update syncs the panel with the current exercise/sequence state.
func (ip *InstructionsPanel) Update(exercise *model.Exercise, state *editor.EditorState, seqIdx int, editLang string) {
	ip.exercise = exercise
	ip.state = state
	ip.seqIdx = seqIdx
	ip.editLang = editLang

	if exercise == nil || seqIdx >= len(exercise.Sequences) {
		ip.vbox.RemoveAll()
		ip.entries = nil
		ip.syncedSeqIdx = -1
		ip.syncedCount = 0
		return
	}

	instrs := ip.resolveInstructions()
	needSync := seqIdx != ip.syncedSeqIdx || len(instrs) != ip.syncedCount || editLang != ip.syncedEditLang

	if needSync {
		ip.rebuildEntries(instrs)
		ip.syncedSeqIdx = seqIdx
		ip.syncedCount = len(instrs)
		ip.syncedEditLang = editLang
	}
}

// RefreshLanguage updates the header text for the current language.
func (ip *InstructionsPanel) RefreshLanguage() {
	ip.header.Text = i18n.T(i18n.KeyInstrHeader)
	ip.header.Refresh()
}

// ForceResync forces rebuilding entries on the next Update.
func (ip *InstructionsPanel) ForceResync() {
	ip.syncedSeqIdx = -1
	ip.syncedCount = -1
	ip.syncedEditLang = ""
}

func (ip *InstructionsPanel) rebuildEntries(instrs []string) {
	ip.vbox.RemoveAll()
	ip.entries = make([]*widget.Entry, len(instrs))
	for i, text := range instrs {
		idx := i
		entry := widget.NewEntry()
		entry.SetText(text)
		entry.OnChanged = func(s string) {
			ip.setInstruction(idx, s)
		}

		delBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
			ip.deleteInstruction(idx)
		})
		delBtn.Importance = widget.LowImportance

		row := container.NewBorder(nil, nil, nil, delBtn, entry)
		ip.entries[i] = entry
		ip.vbox.Add(row)
	}
}

func (ip *InstructionsPanel) resolveInstructions() []string {
	if ip.exercise == nil || ip.seqIdx >= len(ip.exercise.Sequences) {
		return nil
	}
	if ip.editLang == "en" {
		return ip.exercise.Sequences[ip.seqIdx].Instructions
	}
	tr := ip.exercise.EnsureI18n(ip.editLang)
	if ip.seqIdx < len(tr.Sequences) {
		return tr.Sequences[ip.seqIdx].Instructions
	}
	return nil
}

func (ip *InstructionsPanel) setInstruction(idx int, text string) {
	if ip.exercise == nil || ip.seqIdx >= len(ip.exercise.Sequences) {
		return
	}
	if ip.editLang == "en" {
		if idx < len(ip.exercise.Sequences[ip.seqIdx].Instructions) {
			ip.exercise.Sequences[ip.seqIdx].Instructions[idx] = text
		}
	} else {
		tr := ip.exercise.EnsureI18n(ip.editLang)
		ip.ensureI18nSeq(&tr, ip.seqIdx)
		if idx < len(tr.Sequences[ip.seqIdx].Instructions) {
			tr.Sequences[ip.seqIdx].Instructions[idx] = text
		}
		ip.exercise.SetI18n(ip.editLang, tr)
	}
	if ip.state != nil {
		ip.state.MarkModified()
	}
	if ip.OnModified != nil {
		ip.OnModified()
	}
}

func (ip *InstructionsPanel) deleteInstruction(idx int) {
	if ip.exercise == nil || ip.seqIdx >= len(ip.exercise.Sequences) {
		return
	}
	if ip.editLang == "en" {
		seq := &ip.exercise.Sequences[ip.seqIdx]
		if idx < len(seq.Instructions) {
			seq.Instructions = append(seq.Instructions[:idx], seq.Instructions[idx+1:]...)
		}
	} else {
		tr := ip.exercise.EnsureI18n(ip.editLang)
		ip.ensureI18nSeq(&tr, ip.seqIdx)
		s := &tr.Sequences[ip.seqIdx]
		if idx < len(s.Instructions) {
			s.Instructions = append(s.Instructions[:idx], s.Instructions[idx+1:]...)
		}
		ip.exercise.SetI18n(ip.editLang, tr)
	}
	if ip.state != nil {
		ip.state.MarkModified()
	}
	ip.syncedCount = -1
	ip.Update(ip.exercise, ip.state, ip.seqIdx, ip.editLang)
	if ip.OnModified != nil {
		ip.OnModified()
	}
}

func (ip *InstructionsPanel) addInstruction() {
	if ip.exercise == nil || ip.seqIdx >= len(ip.exercise.Sequences) {
		return
	}
	instrs := ip.resolveInstructions()
	if len(instrs) >= maxInstructions {
		return
	}
	if ip.editLang == "en" {
		ip.exercise.Sequences[ip.seqIdx].Instructions = append(ip.exercise.Sequences[ip.seqIdx].Instructions, "")
	} else {
		tr := ip.exercise.EnsureI18n(ip.editLang)
		ip.ensureI18nSeq(&tr, ip.seqIdx)
		tr.Sequences[ip.seqIdx].Instructions = append(tr.Sequences[ip.seqIdx].Instructions, "")
		ip.exercise.SetI18n(ip.editLang, tr)
	}
	if ip.state != nil {
		ip.state.MarkModified()
	}
	ip.syncedCount = -1
	ip.Update(ip.exercise, ip.state, ip.seqIdx, ip.editLang)
	if ip.OnModified != nil {
		ip.OnModified()
	}
}

func (ip *InstructionsPanel) ensureI18nSeq(tr *model.ExerciseI18n, seqIdx int) {
	for len(tr.Sequences) <= seqIdx {
		tr.Sequences = append(tr.Sequences, model.SequenceI18n{})
	}
}
