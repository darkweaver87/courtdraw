package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/ui/editor"
	"github.com/darkweaver87/courtdraw/internal/ui/icon"
)

// ActionTimeline displays an ordered list of actions per sequence
// with drag-and-drop reordering by step.
type ActionTimeline struct {
	list     *DragList
	box      *fyne.Container
	exercise *model.Exercise
	state    *editor.EditorState
	seqIdx   int

	// Collapsible (mobile).
	collapsed  bool
	chevronBtn *TipButton
	listOuter  *fyne.Container

	OnModified func()
}

// NewActionTimeline creates a new action timeline widget.
func NewActionTimeline() *ActionTimeline {
	at := &ActionTimeline{}
	at.list = NewDragList()
	at.list.MinimalMode = true
	at.list.OnReorder = func(from, to int) {
		at.reorderAction(from, to)
	}

	header := canvas.NewText(i18n.T(i18n.KeyPropsStep)+" Timeline", color.NRGBA{R: 0xaa, G: 0xaa, B: 0xaa, A: 0xff})
	header.TextSize = 12
	header.TextStyle.Bold = true

	listMinW := float32(200)
	if isMobile {
		listMinW = 160
	}
	listSpacer := canvas.NewRectangle(color.Transparent)
	listSpacer.SetMinSize(fyne.NewSize(listMinW, 0))
	at.listOuter = container.NewStack(listSpacer, container.NewBorder(header, nil, nil, nil, at.list))

	at.chevronBtn = NewTipButton(icon.ChevronRight, "", func() {
		if at.collapsed {
			at.collapsed = false
			at.listOuter.Show()
			at.chevronBtn.Icon = icon.ChevronRight
		} else {
			at.collapsed = true
			at.listOuter.Hide()
			at.chevronBtn.Icon = icon.ChevronLeft
		}
		at.chevronBtn.Refresh()
		// Force parent layout to resize the court.
		at.box.Refresh()
	})
	chevronSize := fyne.NewSize(24, 24)
	if isMobile {
		chevronSize = fyne.NewSize(40, 40)
	}

	if isMobile {
		at.collapsed = true
		at.listOuter.Hide()
		at.chevronBtn.Icon = icon.ChevronLeft
	}

	at.box = container.NewBorder(nil, nil, container.NewGridWrap(chevronSize, at.chevronBtn), nil, at.listOuter)
	return at
}

// Widget returns the timeline container.
func (at *ActionTimeline) Widget() fyne.CanvasObject {
	return at.box
}

// Update rebuilds the timeline from the current exercise/sequence.
func (at *ActionTimeline) Update(exercise *model.Exercise, state *editor.EditorState, seqIdx int) {
	at.exercise = exercise
	at.state = state
	at.seqIdx = seqIdx
	at.rebuild()
}

func (at *ActionTimeline) rebuild() {
	if at.exercise == nil || at.seqIdx >= len(at.exercise.Sequences) {
		at.list.SetItems(nil)
		return
	}
	seq := &at.exercise.Sequences[at.seqIdx]
	if len(seq.Actions) == 0 {
		at.list.SetItems(nil)
		return
	}

	// Build sorted action indices by step then order.
	sorted := sortedActionIndices(seq)
	actionIssues := model.ValidateActions(seq)

	items := make([]DragListItem, len(sorted))
	for i, idx := range sorted {
		act := &seq.Actions[idx]
		desc := actionTimelineLabel(act, seq)
		indicator := "✅"
		if issues, ok := actionIssues[idx]; ok {
			hasError := false
			for _, issue := range issues {
				if issue.IsError {
					hasError = true
					break
				}
			}
			if hasError {
				indicator = "⛔"
			} else {
				indicator = "⚠️"
			}
		}
		items[i] = DragListItem{
			Text: fmt.Sprintf("%s %d. %s", indicator, act.EffectiveStep(), desc),
		}
	}
	at.list.SetItems(items)
}

func (at *ActionTimeline) reorderAction(from, to int) {
	if at.exercise == nil || at.seqIdx >= len(at.exercise.Sequences) {
		return
	}
	seq := &at.exercise.Sequences[at.seqIdx]
	sorted := sortedActionIndices(seq)
	if from >= len(sorted) || to >= len(sorted) {
		return
	}

	fromIdx := sorted[from]
	toIdx := sorted[to]

	// Swap steps between the two actions.
	fromStep := seq.Actions[fromIdx].EffectiveStep()
	toStep := seq.Actions[toIdx].EffectiveStep()
	seq.Actions[fromIdx].Step = toStep
	seq.Actions[toIdx].Step = fromStep

	at.state.MarkModified()
	at.rebuild()
	if at.OnModified != nil {
		at.OnModified()
	}
}

// sortedActionIndices returns action indices sorted by step then slice order.
func sortedActionIndices(seq *model.Sequence) []int {
	indices := make([]int, len(seq.Actions))
	for i := range indices {
		indices[i] = i
	}
	// Simple insertion sort by step.
	for i := 1; i < len(indices); i++ {
		j := i
		for j > 0 && seq.Actions[indices[j]].EffectiveStep() < seq.Actions[indices[j-1]].EffectiveStep() {
			indices[j], indices[j-1] = indices[j-1], indices[j]
			j--
		}
	}
	return indices
}

// RefreshLanguage updates translatable elements.
func (at *ActionTimeline) RefreshLanguage() {
	at.rebuild()
}

func actionTimelineLabel(act *model.Action, seq *model.Sequence) string {
	typeName := actionDisplayLabel(act.Type)
	fromName := ""
	if act.From.IsPlayer {
		for i := range seq.Players {
			if seq.Players[i].ID == act.From.PlayerID {
				fromName = seq.Players[i].Label
				break
			}
		}
	}
	toName := ""
	if act.To.IsPlayer {
		for i := range seq.Players {
			if seq.Players[i].ID == act.To.PlayerID {
				toName = seq.Players[i].Label
				break
			}
		}
	}

	if fromName != "" && toName != "" {
		return fmt.Sprintf("%s %s → %s", typeName, fromName, toName)
	}
	if fromName != "" {
		return fmt.Sprintf("%s %s", typeName, fromName)
	}
	return typeName
}
