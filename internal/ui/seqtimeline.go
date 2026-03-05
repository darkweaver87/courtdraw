package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
	cdtheme "github.com/darkweaver87/courtdraw/internal/ui/theme"
)

// SeqTimeline is a horizontal row of sequence tabs with an [+] add button.
type SeqTimeline struct {
	box       *fyne.Container
	tabBox    *fyne.Container
	addBtn    *widget.Button
	buttons   []*widget.Button
	activeIdx int

	OnSeqChanged func(int)
	OnAddSeq     func()
}

// NewSeqTimeline creates a new sequence timeline.
func NewSeqTimeline() *SeqTimeline {
	st := &SeqTimeline{}
	st.tabBox = container.NewHBox()
	st.addBtn = widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		if st.OnAddSeq != nil {
			st.OnAddSeq()
		}
	})
	st.addBtn.Importance = widget.LowImportance

	bg := canvas.NewRectangle(color.NRGBA{R: 0x2a, G: 0x2a, B: 0x2a, A: 0xff})
	scroll := container.NewHScroll(container.NewHBox(st.tabBox, st.addBtn, layout.NewSpacer()))
	st.box = container.NewStack(bg, scroll)
	return st
}

// Widget returns the timeline widget.
func (st *SeqTimeline) Widget() fyne.CanvasObject {
	return st.box
}

// Update rebuilds the timeline tabs for the given exercise and active index.
// editLang selects translated sequence labels when available.
func (st *SeqTimeline) Update(exercise *model.Exercise, activeIdx int, editLang string) {
	if exercise == nil {
		st.tabBox.RemoveAll()
		st.buttons = nil
		return
	}

	st.activeIdx = activeIdx

	// Rebuild buttons if count changed.
	if len(st.buttons) != len(exercise.Sequences) {
		st.tabBox.RemoveAll()
		st.buttons = make([]*widget.Button, len(exercise.Sequences))
		for i := range exercise.Sequences {
			idx := i
			btn := widget.NewButton("", func() {
				if st.OnSeqChanged != nil {
					st.OnSeqChanged(idx)
				}
			})
			btn.Importance = widget.LowImportance
			st.buttons[i] = btn
			st.tabBox.Add(btn)
		}
	}

	// Resolve i18n sequence labels.
	var trSeqs []model.SequenceI18n
	if editLang != "" && editLang != "en" && exercise.I18n != nil {
		if tr, ok := exercise.I18n[editLang]; ok {
			trSeqs = tr.Sequences
		}
	}

	// Update labels and colors.
	for i, seq := range exercise.Sequences {
		label := seq.Label
		// Use translated label if available.
		if i < len(trSeqs) && trSeqs[i].Label != "" {
			label = trSeqs[i].Label
		}
		if label == "" {
			label = i18n.Tf("seq.format", i+1)
		} else {
			label = fmt.Sprintf("%d. %s", i+1, label)
		}
		st.buttons[i].SetText(label)
		if i == activeIdx {
			st.buttons[i].Importance = widget.HighImportance
		} else {
			st.buttons[i].Importance = widget.LowImportance
		}
		st.buttons[i].Refresh()
	}
}

// unused but needed for theme import
var _ = cdtheme.ColorTabText
