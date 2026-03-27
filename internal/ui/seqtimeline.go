package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
	cdtheme "github.com/darkweaver87/courtdraw/internal/ui/theme"
)

// SeqTimeline is a horizontal bar with prev/next arrows, sequence label, add and delete buttons.
type SeqTimeline struct {
	box       *fyne.Container
	prevBtn   *widget.Button
	nextBtn   *widget.Button
	seqBtn    *widget.Button // current sequence label (tap to rename)
	addBtn    *widget.Button
	deleteBtn *widget.Button
	settingsBtn *TipButton
	activeIdx int
	numSeqs   int

	// Current state (updated on each Update call).
	exercise *model.Exercise
	editLang string

	OnSeqChanged func(int)
	OnAddSeq     func()
	OnDeleteSeq  func(int)
	OnSeqRenamed func(idx int, newLabel string)
	OnSettings   func() // opens exercise settings dialog
	window       fyne.Window
}

// NewSeqTimeline creates a new sequence timeline.
func NewSeqTimeline() *SeqTimeline {
	st := &SeqTimeline{}

	st.prevBtn = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		if st.activeIdx > 0 && st.OnSeqChanged != nil {
			st.OnSeqChanged(st.activeIdx - 1)
		}
	})
	st.prevBtn.Importance = widget.LowImportance

	st.nextBtn = widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
		if st.activeIdx < st.numSeqs-1 && st.OnSeqChanged != nil {
			st.OnSeqChanged(st.activeIdx + 1)
		}
	})
	st.nextBtn.Importance = widget.LowImportance

	st.seqBtn = widget.NewButton("", func() {
		// Tap on label → rename.
		st.showRenameDialog(st.activeIdx, st.exercise)
	})
	st.seqBtn.Importance = widget.HighImportance

	st.addBtn = widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		if st.OnAddSeq != nil {
			st.OnAddSeq()
		}
	})
	st.addBtn.Importance = widget.LowImportance

	st.deleteBtn = widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		if st.OnDeleteSeq != nil {
			st.OnDeleteSeq(st.activeIdx)
		}
	})
	st.deleteBtn.Importance = widget.DangerImportance

	st.settingsBtn = NewTipButton(theme.SettingsIcon(), i18n.T(i18n.KeySettingsExerciseTitle), func() {
		if st.OnSettings != nil {
			st.OnSettings()
		}
	})

	bg := canvas.NewRectangle(color.NRGBA{R: 0x2a, G: 0x2a, B: 0x2a, A: 0xff})
	bar := container.NewHBox(st.prevBtn, st.seqBtn, st.nextBtn, st.addBtn, st.deleteBtn, layout.NewSpacer(), st.settingsBtn)
	st.box = container.NewStack(bg, bar)
	return st
}

// Widget returns the timeline widget.
func (st *SeqTimeline) Widget() fyne.CanvasObject {
	return st.box
}

// SetWindow sets the window reference for dialogs.
func (st *SeqTimeline) SetWindow(w fyne.Window) {
	st.window = w
}

// Update refreshes the timeline for the given exercise and active index.
func (st *SeqTimeline) Update(exercise *model.Exercise, activeIdx int, editLang string) {
	if exercise == nil {
		st.seqBtn.SetText("")
		st.prevBtn.Disable()
		st.nextBtn.Disable()
		st.deleteBtn.Hide()
		return
	}

	st.exercise = exercise
	st.editLang = editLang
	st.activeIdx = activeIdx
	st.numSeqs = len(exercise.Sequences)

	// Update prev/next arrow state.
	if activeIdx <= 0 {
		st.prevBtn.Disable()
	} else {
		st.prevBtn.Enable()
	}
	if activeIdx >= st.numSeqs-1 {
		st.nextBtn.Disable()
	} else {
		st.nextBtn.Enable()
	}

	// Show delete only when 2+ sequences.
	if st.numSeqs > 1 {
		st.deleteBtn.Show()
	} else {
		st.deleteBtn.Hide()
	}

	// Resolve label (without number prefix).
	label := st.resolveLabel(activeIdx)
	st.seqBtn.SetText(label)
}

// resolveLabel returns the display label for the sequence at idx.
func (st *SeqTimeline) resolveLabel(idx int) string {
	if st.exercise == nil || idx >= len(st.exercise.Sequences) {
		return ""
	}
	label := st.exercise.Sequences[idx].Label

	// Use translated label if available.
	if st.editLang != "" && st.editLang != "en" && st.exercise.I18n != nil {
		if tr, ok := st.exercise.I18n[st.editLang]; ok && idx < len(tr.Sequences) && tr.Sequences[idx].Label != "" {
			label = tr.Sequences[idx].Label
		}
	}

	if label == "" {
		label = i18n.Tf(i18n.KeySeqFormat, idx+1)
	}
	return label
}

// showRenameDialog opens an entry dialog to rename a sequence.
func (st *SeqTimeline) showRenameDialog(idx int, exercise *model.Exercise) {
	if st.window == nil || exercise == nil || idx >= len(exercise.Sequences) {
		return
	}

	currentLabel := st.resolveLabel(idx)

	entry := widget.NewEntry()
	entry.SetText(currentLabel)

	dlg := dialog.NewForm(i18n.T(i18n.KeySeqRenameTitle), i18n.T(i18n.KeySeqRenameOk), i18n.T(i18n.KeySeqRenameCancel),
		[]*widget.FormItem{
			widget.NewFormItem(i18n.T(i18n.KeySeqRenameLabel), entry),
		},
		func(ok bool) {
			if ok && st.OnSeqRenamed != nil {
				st.OnSeqRenamed(idx, entry.Text)
			}
		},
		st.window,
	)
	dlg.Show()
}


// unused but needed for theme import
var _ = cdtheme.ColorTabText
