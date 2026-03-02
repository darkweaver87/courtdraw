package widget

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"time"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/ui/icon"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
)

const maxLibraryItems = 100
const maxSessionItems = 50
const maxCoachNotes = 20

// SessionComposer is the full session composer widget.
type SessionComposer struct {
	session *model.Session
	// Resolved exercises from store (keyed by entry name).
	resolvedExercises map[string]*model.Exercise

	// Library panel.
	libraryNames    []string
	libraryExercises []*model.Exercise
	libraryClicks   [maxLibraryItems]widget.Clickable
	libraryList     widget.List
	searchEditor    widget.Editor
	categoryFilter  widget.Clickable
	filterCategory  model.Category

	// Session panel.
	sessionList   widget.List
	sessionClicks [maxSessionItems]widget.Clickable
	removeClicks  [maxSessionItems]widget.Clickable

	// Session metadata editors.
	titleEditor    widget.Editor
	dateEditor     widget.Editor
	todayClick     widget.Clickable
	calendarClick  widget.Clickable
	datePicker     DatePicker
	subtitleEditor widget.Editor
	ageGroupEditor widget.Editor
	philEditor     widget.Editor

	// Coach notes.
	noteEditors  [maxCoachNotes]widget.Editor
	addNoteClick widget.Clickable
	delNoteClicks [maxCoachNotes]widget.Clickable

	// (addExerciseClick removed — exercises are added from library panel)

	// File operations.
	newClick  widget.Clickable
	openClick widget.Clickable
	saveClick widget.Clickable

	// Session list overlay (for Open).
	sessionListOverlay *SessionListOverlay

	// Generate PDF button.
	generateClick widget.Clickable

	// Sync flags.
	metaSynced bool
	modified   bool

	// Scroll for session panel.
	sessionScrollList widget.List

	// Selected exercise in session list (for preview).
	selectedIdx int

	initialized bool
}

// NewSessionComposer creates a new SessionComposer.
func NewSessionComposer() *SessionComposer {
	sc := &SessionComposer{
		resolvedExercises: make(map[string]*model.Exercise),
		selectedIdx:       -1,
		sessionListOverlay: NewSessionListOverlay(),
	}
	sc.searchEditor.SingleLine = true
	sc.titleEditor.SingleLine = true
	sc.dateEditor.SingleLine = true
	sc.subtitleEditor.SingleLine = true
	sc.ageGroupEditor.SingleLine = true
	return sc
}

// SetSession sets the current session.
func (sc *SessionComposer) SetSession(s *model.Session) {
	sc.session = s
	sc.metaSynced = false
	sc.modified = false
	sc.selectedIdx = -1
}

// Session returns the current session.
func (sc *SessionComposer) Session() *model.Session {
	return sc.session
}

// Modified returns whether the session has unsaved changes.
func (sc *SessionComposer) Modified() bool {
	return sc.modified
}

// ClearModified clears the modified flag.
func (sc *SessionComposer) ClearModified() {
	sc.modified = false
}

// SetLibrary sets the available exercises for the library panel.
func (sc *SessionComposer) SetLibrary(names []string, exercises []*model.Exercise) {
	sc.libraryNames = names
	sc.libraryExercises = exercises
}

// SetResolvedExercises sets resolved exercises for display.
func (sc *SessionComposer) SetResolvedExercises(resolved map[string]*model.Exercise) {
	sc.resolvedExercises = resolved
}

// SessionListOverlay returns the session list overlay for external control.
func (sc *SessionComposer) SessionListOverlay() *SessionListOverlay {
	return sc.sessionListOverlay
}

// Layout renders the session composer.
func (sc *SessionComposer) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if !sc.initialized {
		sc.libraryList.Axis = layout.Vertical
		sc.sessionList.Axis = layout.Vertical
		sc.sessionScrollList.Axis = layout.Vertical
		sc.initialized = true
	}

	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				// File toolbar.
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return sc.layoutSessionToolbar(gtx, th)
				}),
				// Main content: library | session.
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						// Left: exercise library.
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return sc.layoutLibrary(gtx, th)
						}),
						// Right: session panel.
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return sc.layoutSessionPanel(gtx, th)
						}),
					)
				}),
			)
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			if sc.sessionListOverlay == nil || !sc.sessionListOverlay.Visible {
				return layout.Dimensions{}
			}
			dims, selected := sc.sessionListOverlay.Layout(gtx, th)
			if selected != "" {
				sc.sessionListOverlay.OnSelect = selected
			}
			return dims
		}),
	)
}

// SessionAction represents a session file action.
type SessionAction int

const (
	SessionActionNone SessionAction = iota
	SessionActionNew
	SessionActionOpen
	SessionActionSave
	SessionActionGenerate
)

// HandleActions checks for pending actions and returns the action type.
func (sc *SessionComposer) HandleActions(gtx layout.Context) SessionAction {
	if sc.newClick.Clicked(gtx) {
		return SessionActionNew
	}
	if sc.openClick.Clicked(gtx) {
		return SessionActionOpen
	}
	if sc.saveClick.Clicked(gtx) {
		return SessionActionSave
	}
	if sc.generateClick.Clicked(gtx) {
		return SessionActionGenerate
	}
	return SessionActionNone
}

func (sc *SessionComposer) layoutSessionToolbar(gtx layout.Context, th *material.Theme) layout.Dimensions {
	// Handle file actions.
	barH := gtx.Dp(unit.Dp(28))
	bg := color.NRGBA{R: 0x2e, G: 0x2e, B: 0x2e, A: 0xff}
	paint.FillShape(gtx.Ops, bg, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, barH)}.Op())

	saveColor := theme.ColorTabText
	if sc.modified {
		saveColor = theme.ColorCoach
	}

	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return icon.IconBtnTooltip(gtx, th, &sc.newClick, icon.New, theme.ColorTabText, i18n.T("tooltip.new"))
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return icon.IconBtnTooltip(gtx, th, &sc.openClick, icon.Open, theme.ColorTabText, i18n.T("tooltip.open"))
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return icon.IconBtnTooltip(gtx, th, &sc.saveClick, icon.Save, saveColor, i18n.T("tooltip.save"))
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return icon.IconBtnTooltip(gtx, th, &sc.generateClick, icon.PDF, theme.ColorCoach, i18n.T("tooltip.generate_pdf"))
			})
		}),
	)
}


func (sc *SessionComposer) layoutLibrary(gtx layout.Context, th *material.Theme) layout.Dimensions {
	bg := color.NRGBA{R: 0x2c, G: 0x2c, B: 0x2c, A: 0xff}
	paint.FillShape(gtx.Ops, bg, clip.Rect{Max: gtx.Constraints.Max}.Op())

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Header.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(8), Left: unit.Dp(8), Bottom: unit.Dp(4)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, unit.Sp(13), i18n.T("session.exercise_library"))
					lbl.Color = theme.ColorTabActive
					return lbl.Layout(gtx)
				},
			)
		}),
		// Search bar.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8), Bottom: unit.Dp(4)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					edBg := color.NRGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xff}
					return layoutEditorWithBg(gtx, th, &sc.searchEditor, edBg)
				},
			)
		}),
		// Category filter.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if sc.categoryFilter.Clicked(gtx) {
				sc.filterCategory = nextCategoryWithAll(sc.filterCategory)
			}
			label := i18n.T("session.category_all")
			if sc.filterCategory != "" {
				label = i18n.T("category." + string(sc.filterCategory))
			}
			return layout.Inset{Left: unit.Dp(8), Bottom: unit.Dp(4)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return material.Clickable(gtx, &sc.categoryFilter, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(11), i18n.Tf("session.category_label", label))
						lbl.Color = theme.ColorTabText
						return lbl.Layout(gtx)
					})
				},
			)
		}),
		// Exercise list.
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			filtered := sc.filteredLibrary()
			count := len(filtered)
			if count > maxLibraryItems {
				count = maxLibraryItems
			}
			if count == 0 {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, unit.Sp(12), i18n.T("session.no_exercises"))
					lbl.Color = theme.ColorTabText
					return lbl.Layout(gtx)
				})
			}
			return material.List(th, &sc.libraryList).Layout(gtx, count, func(gtx layout.Context, idx int) layout.Dimensions {
				item := filtered[idx]
				if sc.libraryClicks[idx].Clicked(gtx) && sc.session != nil {
					sc.AddExerciseByRef(item.name)
				}
				return material.Clickable(gtx, &sc.libraryClicks[idx], func(gtx layout.Context) layout.Dimensions {
					return sc.layoutLibraryItem(gtx, th, item.exercise)
				})
			})
		}),
	)
}

type libraryItem struct {
	name     string          // store key (kebab)
	exercise *model.Exercise // resolved exercise
}

func (sc *SessionComposer) filteredLibrary() []libraryItem {
	search := strings.ToLower(sc.searchEditor.Text())
	var result []libraryItem
	for i, ex := range sc.libraryExercises {
		if ex == nil {
			continue
		}
		// Category filter.
		if sc.filterCategory != "" && ex.Category != sc.filterCategory {
			continue
		}
		// Text search.
		if search != "" {
			match := strings.Contains(strings.ToLower(ex.Name), search)
			if !match {
				for _, tag := range ex.Tags {
					if strings.Contains(strings.ToLower(tag), search) {
						match = true
						break
					}
				}
			}
			if !match {
				continue
			}
		}
		name := ""
		if i < len(sc.libraryNames) {
			name = sc.libraryNames[i]
		}
		result = append(result, libraryItem{name: name, exercise: ex})
	}
	return result
}

func (sc *SessionComposer) layoutLibraryItem(gtx layout.Context, th *material.Theme, ex *model.Exercise) layout.Dimensions {
	return layout.Inset{
		Top: unit.Dp(3), Bottom: unit.Dp(3),
		Left: unit.Dp(8), Right: unit.Dp(8),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, unit.Sp(12), ex.Name)
				lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
				return lbl.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				info := i18n.T("category." + string(ex.Category))
				if ex.Category == "" {
					info = ""
				}
				if ex.Duration != "" {
					if info != "" {
						info += " · "
					}
					info += ex.Duration
				}
				info += " " + intensityDots(int(ex.Intensity))
				lbl := material.Label(th, unit.Sp(10), info)
				lbl.Color = theme.ColorTabText
				return lbl.Layout(gtx)
			}),
		)
	})
}

func intensityDots(n int) string {
	dots := ""
	for i := 0; i < 3; i++ {
		if i < n {
			dots += "●"
		} else {
			dots += "○"
		}
	}
	return dots
}

// AddExerciseByRef adds an exercise entry by its store reference (kebab name).
// Does nothing if the exercise is already in the session.
func (sc *SessionComposer) AddExerciseByRef(ref string) {
	if sc.session == nil {
		return
	}
	if len(sc.session.Exercises) >= maxSessionItems {
		return
	}
	// Prevent duplicates.
	for _, entry := range sc.session.Exercises {
		if entry.Exercise == ref {
			return
		}
	}
	sc.session.Exercises = append(sc.session.Exercises, model.ExerciseEntry{Exercise: ref})
	sc.modified = true
}

func (sc *SessionComposer) layoutSessionPanel(gtx layout.Context, th *material.Theme) layout.Dimensions {
	bg := color.NRGBA{R: 0x30, G: 0x30, B: 0x30, A: 0xff}
	paint.FillShape(gtx.Ops, bg, clip.Rect{Max: gtx.Constraints.Max}.Op())

	if sc.session == nil {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, unit.Sp(14), i18n.T("tab.no_session"))
			lbl.Color = theme.ColorTabText
			return lbl.Layout(gtx)
		})
	}

	// Sync editors.
	if !sc.metaSynced {
		sc.titleEditor.SetText(sc.session.Title)
		sc.dateEditor.SetText(sc.session.Date)
		sc.subtitleEditor.SetText(sc.session.Subtitle)
		sc.ageGroupEditor.SetText(sc.session.AgeGroup)
		sc.philEditor.SetText(sc.session.Philosophy)
		for i := 0; i < len(sc.session.CoachNotes) && i < maxCoachNotes; i++ {
			sc.noteEditors[i].SetText(sc.session.CoachNotes[i])
		}
		sc.metaSynced = true
	}

	// Process text changes.
	sc.processEditorChanges(gtx)

	// Build items list.
	var items []func(gtx layout.Context) layout.Dimensions

	// Header.
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(8), Left: unit.Dp(8), Bottom: unit.Dp(4)}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, unit.Sp(13), i18n.T("session.header"))
				lbl.Color = theme.ColorTabActive
				return lbl.Layout(gtx)
			},
		)
	})

	// Metadata fields.
	items = append(items, sc.metaField(th, i18n.T("session.title"), &sc.titleEditor)...)
	items = append(items, sc.dateField(gtx, th)...)
	items = append(items, sc.metaField(th, i18n.T("session.subtitle"), &sc.subtitleEditor)...)
	items = append(items, sc.metaField(th, i18n.T("session.age_group"), &sc.ageGroupEditor)...)

	// Exercise list header.
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(8), Left: unit.Dp(8), Bottom: unit.Dp(2)}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, unit.Sp(11), i18n.T("session.exercises"))
				lbl.Color = theme.ColorTabActive
				return lbl.Layout(gtx)
			},
		)
	})

	// Exercise entries.
	totalDuration := ""
	exCount := len(sc.session.Exercises)
	if exCount > maxSessionItems {
		exCount = maxSessionItems
	}
	for i := 0; i < exCount; i++ {
		idx := i
		entry := &sc.session.Exercises[idx]
		if sc.removeClicks[idx].Clicked(gtx) {
			sc.session.Exercises = append(sc.session.Exercises[:idx], sc.session.Exercises[idx+1:]...)
			sc.modified = true
			break // indices shifted
		}
		items = append(items, func(gtx layout.Context) layout.Dimensions {
			return sc.layoutSessionExercise(gtx, th, idx, entry)
		})
		// Variants.
		for vi := range entry.Variants {
			vIdx := vi
			variant := &entry.Variants[vIdx]
			items = append(items, func(gtx layout.Context) layout.Dimensions {
				return sc.layoutVariant(gtx, th, variant)
			})
		}
	}

	// Total duration.
	totalDuration = sc.computeTotalDuration()
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(4), Left: unit.Dp(8)}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, unit.Sp(11), i18n.Tf("session.total_format", totalDuration))
				lbl.Color = theme.ColorCoach
				return lbl.Layout(gtx)
			},
		)
	})

	// Hint to add exercises from library.
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(4), Left: unit.Dp(8)}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, unit.Sp(10), i18n.T("session.add_from_library"))
				lbl.Color = theme.ColorTabText
				return lbl.Layout(gtx)
			},
		)
	})

	// Coach notes header.
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(8), Left: unit.Dp(8), Bottom: unit.Dp(2)}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, unit.Sp(11), i18n.T("session.coach_notes"))
				lbl.Color = theme.ColorTabActive
				return lbl.Layout(gtx)
			},
		)
	})

	// Coach notes.
	for i := 0; i < len(sc.session.CoachNotes) && i < maxCoachNotes; i++ {
		idx := i
		items = append(items, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8), Top: unit.Dp(1)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							edBg := color.NRGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xff}
							return layoutEditorWithBg(gtx, th, &sc.noteEditors[idx], edBg)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if sc.delNoteClicks[idx].Clicked(gtx) {
								sc.session.CoachNotes = append(sc.session.CoachNotes[:idx], sc.session.CoachNotes[idx+1:]...)
								sc.modified = true
								sc.metaSynced = false
							}
							return icon.IconBtn(gtx, &sc.delNoteClicks[idx], icon.Close, color.NRGBA{R: 0xff, G: 0x60, B: 0x60, A: 0xff})
						}),
					)
				},
			)
		})
	}

	// Add note button.
	if sc.addNoteClick.Clicked(gtx) && sc.session != nil && len(sc.session.CoachNotes) < maxCoachNotes {
		sc.session.CoachNotes = append(sc.session.CoachNotes, "")
		sc.metaSynced = false
		sc.modified = true
	}
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: unit.Dp(4), Top: unit.Dp(2)}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				return icon.IconTextBtn(gtx, th, &sc.addNoteClick, icon.Add, i18n.T("session.add_note"), theme.ColorCoach)
			},
		)
	})

	// Philosophy.
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(8), Left: unit.Dp(8), Bottom: unit.Dp(2)}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, unit.Sp(11), i18n.T("session.philosophy"))
				lbl.Color = theme.ColorTabActive
				return lbl.Layout(gtx)
			},
		)
	})
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				edBg := color.NRGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xff}
				return layoutEditorWithBg(gtx, th, &sc.philEditor, edBg)
			},
		)
	})

	return material.List(th, &sc.sessionScrollList).Layout(gtx, len(items), func(gtx layout.Context, idx int) layout.Dimensions {
		return items[idx](gtx)
	})
}

func (sc *SessionComposer) processEditorChanges(gtx layout.Context) {
	if sc.session == nil {
		return
	}
	for {
		evt, ok := sc.titleEditor.Update(gtx)
		if !ok {
			break
		}
		if _, isChange := evt.(widget.ChangeEvent); isChange {
			sc.session.Title = sc.titleEditor.Text()
			sc.modified = true
		}
	}
	for {
		evt, ok := sc.dateEditor.Update(gtx)
		if !ok {
			break
		}
		if _, isChange := evt.(widget.ChangeEvent); isChange {
			sc.session.Date = sc.dateEditor.Text()
			sc.modified = true
		}
	}
	for {
		evt, ok := sc.subtitleEditor.Update(gtx)
		if !ok {
			break
		}
		if _, isChange := evt.(widget.ChangeEvent); isChange {
			sc.session.Subtitle = sc.subtitleEditor.Text()
			sc.modified = true
		}
	}
	for {
		evt, ok := sc.ageGroupEditor.Update(gtx)
		if !ok {
			break
		}
		if _, isChange := evt.(widget.ChangeEvent); isChange {
			sc.session.AgeGroup = sc.ageGroupEditor.Text()
			sc.modified = true
		}
	}
	for {
		evt, ok := sc.philEditor.Update(gtx)
		if !ok {
			break
		}
		if _, isChange := evt.(widget.ChangeEvent); isChange {
			sc.session.Philosophy = sc.philEditor.Text()
			sc.modified = true
		}
	}
	// Coach notes.
	for i := 0; i < len(sc.session.CoachNotes) && i < maxCoachNotes; i++ {
		for {
			evt, ok := sc.noteEditors[i].Update(gtx)
			if !ok {
				break
			}
			if _, isChange := evt.(widget.ChangeEvent); isChange {
				sc.session.CoachNotes[i] = sc.noteEditors[i].Text()
				sc.modified = true
			}
		}
	}
}

func (sc *SessionComposer) metaField(th *material.Theme, label string, ed *widget.Editor) []func(layout.Context) layout.Dimensions {
	return []func(layout.Context) layout.Dimensions{
		func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(2), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(th, unit.Sp(10), label)
							lbl.Color = theme.ColorTabText
							return lbl.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							edBg := color.NRGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xff}
							return layoutEditorWithBg(gtx, th, ed, edBg)
						}),
					)
				},
			)
		},
	}
}

func (sc *SessionComposer) dateField(gtx layout.Context, th *material.Theme) []func(layout.Context) layout.Dimensions {
	// Handle today button.
	if sc.todayClick.Clicked(gtx) && sc.session != nil {
		today := time.Now().Format("2006-01-02")
		sc.dateEditor.SetText(today)
		sc.session.Date = today
		sc.modified = true
	}

	// Handle calendar button — open date picker.
	if sc.calendarClick.Clicked(gtx) && sc.session != nil {
		t, err := time.Parse("2006-01-02", sc.dateEditor.Text())
		if err != nil {
			t = time.Now()
		}
		sc.datePicker.Show(t)
	}

	// Handle date picker result.
	if sc.datePicker.Result != "" && sc.session != nil {
		sc.dateEditor.SetText(sc.datePicker.Result)
		sc.session.Date = sc.datePicker.Result
		sc.modified = true
		sc.datePicker.Result = ""
	}

	return []func(layout.Context) layout.Dimensions{
		func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(2), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(th, unit.Sp(10), i18n.T("session.date"))
							lbl.Color = theme.ColorTabText
							return lbl.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									edBg := color.NRGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xff}
									txt := sc.dateEditor.Text()
									if txt != "" && !isValidDate(txt) {
										edBg = color.NRGBA{R: 0x60, G: 0x30, B: 0x30, A: 0xff}
									}
									return layoutEditorWithBg(gtx, th, &sc.dateEditor, edBg)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return icon.IconBtnTooltip(gtx, th, &sc.calendarClick, icon.Calendar, theme.ColorCoach, i18n.T("session.calendar"))
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return icon.IconBtnTooltip(gtx, th, &sc.todayClick, icon.Today, theme.ColorCoach, i18n.T("session.today"))
								}),
							)
						}),
					)

					// Render calendar dropdown below this field.
					if sc.datePicker.Visible {
						sc.datePicker.LayoutDropdown(gtx, th, dims)
					}

					return dims
				},
			)
		},
	}
}

// isValidDate checks if the string is a valid YYYY-MM-DD date.
func isValidDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

func (sc *SessionComposer) layoutSessionExercise(gtx layout.Context, th *material.Theme, idx int, entry *model.ExerciseEntry) layout.Dimensions {
	return layout.Inset{
		Top: unit.Dp(2), Bottom: unit.Dp(2),
		Left: unit.Dp(8), Right: unit.Dp(8),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, unit.Sp(11), fmt.Sprintf("%d.", idx+1))
				lbl.Color = theme.ColorTabText
				return lbl.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					name := entry.Exercise
					dur := ""
					if ex, ok := sc.resolvedExercises[entry.Exercise]; ok {
						name = ex.Name
						dur = ex.Duration
					}
					label := name
					if dur != "" {
						label += "  " + dur
					}
					lbl := material.Label(th, unit.Sp(12), label)
					lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
					return lbl.Layout(gtx)
				})
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 0)}
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon.IconBtn(gtx, &sc.removeClicks[idx], icon.Close, color.NRGBA{R: 0xff, G: 0x60, B: 0x60, A: 0xff})
			}),
		)
	})
}

func (sc *SessionComposer) layoutVariant(gtx layout.Context, th *material.Theme, variant *model.ExerciseEntry) layout.Dimensions {
	return layout.Inset{
		Top: unit.Dp(1), Bottom: unit.Dp(1),
		Left: unit.Dp(24), Right: unit.Dp(8),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		name := variant.Exercise
		if ex, ok := sc.resolvedExercises[variant.Exercise]; ok {
			name = ex.Name
		}
		lbl := material.Label(th, unit.Sp(11), "-> "+name)
		lbl.Color = theme.ColorTabText
		return lbl.Layout(gtx)
	})
}

func (sc *SessionComposer) computeTotalDuration() string {
	// Sum up durations from resolved exercises.
	// Durations are strings like "15m", "1h30m" — we'll just concatenate for display.
	total := 0 // in minutes
	for _, entry := range sc.session.Exercises {
		if ex, ok := sc.resolvedExercises[entry.Exercise]; ok {
			total += parseDurationMinutes(ex.Duration)
		}
	}
	if total == 0 {
		return i18n.T("session.total_na")
	}
	if total >= 60 {
		return i18n.Tf("session.duration_hm", total/60, total%60)
	}
	return i18n.Tf("session.duration_m", total)
}

// nextCategoryWithAll cycles through categories including "" (All).
func nextCategoryWithAll(current model.Category) model.Category {
	cats := []model.Category{
		"", // All
		model.CategoryWarmup, model.CategoryOffense, model.CategoryDefense,
		model.CategoryTransition, model.CategoryScrimmage, model.CategoryCooldown,
	}
	for i, c := range cats {
		if c == current {
			return cats[(i+1)%len(cats)]
		}
	}
	return ""
}

func parseDurationMinutes(d string) int {
	// Simple parser for "15m", "1h30m", "1h" etc.
	total := 0
	num := 0
	for _, c := range d {
		if c >= '0' && c <= '9' {
			num = num*10 + int(c-'0')
		} else if c == 'h' {
			total += num * 60
			num = 0
		} else if c == 'm' {
			total += num
			num = 0
		}
	}
	return total
}

// SessionListOverlay is a modal overlay for picking a session to open.
type SessionListOverlay struct {
	Visible    bool
	OnSelect   string
	names      []string
	itemClicks [maxSessionItems]widget.Clickable
	closeClick widget.Clickable
	scrollList widget.List
}

// NewSessionListOverlay creates an initialized overlay.
func NewSessionListOverlay() *SessionListOverlay {
	slo := &SessionListOverlay{}
	slo.scrollList.Axis = layout.Vertical
	return slo
}

// Show makes the overlay visible with the given session names.
func (slo *SessionListOverlay) Show(names []string) {
	slo.names = names
	slo.Visible = true
	slo.OnSelect = ""
}

// Hide closes the overlay.
func (slo *SessionListOverlay) Hide() {
	slo.Visible = false
}

// Layout renders the overlay and returns the selected session name, if any.
func (slo *SessionListOverlay) Layout(gtx layout.Context, th *material.Theme) (layout.Dimensions, string) {
	if !slo.Visible {
		return layout.Dimensions{Size: gtx.Constraints.Max}, ""
	}

	selected := ""
	for i := 0; i < len(slo.names) && i < maxSessionItems; i++ {
		if slo.itemClicks[i].Clicked(gtx) {
			selected = slo.names[i]
			slo.Hide()
		}
	}
	if slo.closeClick.Clicked(gtx) {
		slo.Hide()
	}

	// Dim background.
	dimBg := color.NRGBA{A: 0xaa}
	area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	paint.FillShape(gtx.Ops, dimBg, clip.Rect{Max: gtx.Constraints.Max}.Op())
	area.Pop()

	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		panelW := gtx.Dp(unit.Dp(300))
		panelH := gtx.Dp(unit.Dp(400))
		gtx.Constraints = layout.Exact(image.Pt(panelW, panelH))

		panelBg := color.NRGBA{R: 0x35, G: 0x35, B: 0x35, A: 0xff}
		paint.FillShape(gtx.Ops, panelBg, clip.Rect{Max: image.Pt(panelW, panelH)}.Op())

		dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(8), Bottom: unit.Dp(4)}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								lbl := material.Label(th, unit.Sp(14), i18n.T("overlay.open_session"))
								lbl.Color = theme.ColorTabActive
								return lbl.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return icon.IconBtn(gtx, &slo.closeClick, icon.Close, color.NRGBA{R: 0xff, G: 0x60, B: 0x60, A: 0xff})
							}),
						)
					},
				)
			}),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				count := len(slo.names)
				if count > maxSessionItems {
					count = maxSessionItems
				}
				if count == 0 {
					return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(12), i18n.T("overlay.no_sessions"))
						lbl.Color = theme.ColorTabText
						return lbl.Layout(gtx)
					})
				}
				return material.List(th, &slo.scrollList).Layout(gtx, count, func(gtx layout.Context, idx int) layout.Dimensions {
					return material.Clickable(gtx, &slo.itemClicks[idx], func(gtx layout.Context) layout.Dimensions {
						return layout.Inset{
							Top: unit.Dp(4), Bottom: unit.Dp(4),
							Left: unit.Dp(12), Right: unit.Dp(12),
						}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(th, unit.Sp(13), slo.names[idx])
							lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
							return lbl.Layout(gtx)
						})
					})
				})
			}),
		)
		return dims
	}), selected
}
