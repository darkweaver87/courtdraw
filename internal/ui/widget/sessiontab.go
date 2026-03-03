package widget

import (
	"fmt"
	"image"
	"image/color"
	"sort"
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

// ExerciseSyncStatus represents the sync state of a managed exercise.
type ExerciseSyncStatus int

const (
	StatusLocalOnly  ExerciseSyncStatus = iota // exists only locally
	StatusRemoteOnly                           // exists only in community library
	StatusSynced                               // local and remote are identical
	StatusModified                             // local differs from remote
)

// ManagedExercise is an entry in the exercise library list.
type ManagedExercise struct {
	Name        string             // kebab-case key
	Status      ExerciseSyncStatus
	LocalEx     *model.Exercise
	RemoteEx    *model.Exercise
	DisplayName string
	Category    string
	AgeGroup    string
	Duration    string
	Tags        []string
}

const maxMgrItems = 200
const maxLibraryItems = 100
const maxSessionItems = 50
const maxCoachNotes = 20
const maxTagFilters = 10

// SessionTabAction represents an action the user performed in the session tab.
type SessionTabAction int

const (
	SessionTabActionNone           SessionTabAction = iota
	SessionTabActionNew                             // new session
	SessionTabActionOpen                            // open session dialog
	SessionTabActionSave                            // save session
	SessionTabActionGenerate                        // generate PDF
	SessionTabActionRefresh                         // refresh library
	SessionTabActionOpenExercise                    // open exercise in editor
	SessionTabActionUpdate                          // update from remote
	SessionTabActionContribute                      // contribute exercise
	SessionTabActionDeleteExercise                  // delete local exercise
)

// SessionTabEvent is returned when the user performs an action.
type SessionTabEvent struct {
	Action SessionTabAction
	Name   string // exercise name (for exercise actions)
}

// SessionTab merges the exercise manager and session composer into a single
// 3-column widget: library | preview | session.
type SessionTab struct {
	// --- Library column (from ExerciseManager) ---
	items        []ManagedExercise
	scrollList   widget.List
	searchEditor widget.Editor
	filterIndex  int
	filterClicks [5]widget.Clickable
	refreshClick widget.Clickable
	rowClicks    [maxMgrItems]widget.Clickable

	// Category filter (from SessionComposer).
	categoryFilter widget.Clickable
	filterCategory model.Category
	catPopup       PopupSelector

	// Age group filter.
	ageGroupFilter widget.Clickable
	filterAgeGroup model.AgeGroup
	agePopup       PopupSelector

	// Tag filter (multi-select, intersection).
	filterTags      []string
	addTagClick     widget.Clickable
	tagRemoveClicks [maxTagFilters]widget.Clickable
	tagPopup        PopupSelector

	// --- Preview column ---
	previewCourt     PreviewCourt
	selectedName     string
	selectedStatus   ExerciseSyncStatus
	addToSessionClick widget.Clickable
	openExerciseClick widget.Clickable
	updateClick       widget.Clickable
	contributeClick   widget.Clickable
	deleteClick       widget.Clickable

	// --- Session column (from SessionComposer) ---
	session           *model.Session
	resolvedExercises map[string]*model.Exercise

	sessionList   widget.List
	removeClicks  [maxSessionItems]widget.Clickable

	titleEditor    widget.Editor
	dateEditor     widget.Editor
	todayClick     widget.Clickable
	calendarClick  widget.Clickable
	datePicker     DatePicker
	subtitleEditor widget.Editor
	ageGroupEditor widget.Editor
	philEditor     widget.Editor

	noteEditors   [maxCoachNotes]widget.Editor
	addNoteClick  widget.Clickable
	delNoteClicks [maxCoachNotes]widget.Clickable

	// File operations.
	newClick      widget.Clickable
	openClick     widget.Clickable
	saveClick     widget.Clickable
	generateClick widget.Clickable

	// Session list overlay (for Open).
	sessionListOverlay *SessionListOverlay

	// Sync flags.
	metaSynced bool
	modified   bool

	// Scroll for session panel.
	sessionScrollList widget.List

	initialized bool
}

// NewSessionTab creates a new SessionTab.
func NewSessionTab() *SessionTab {
	st := &SessionTab{
		resolvedExercises:  make(map[string]*model.Exercise),
		sessionListOverlay: NewSessionListOverlay(),
	}
	st.scrollList.Axis = layout.Vertical
	st.searchEditor.SingleLine = true
	st.titleEditor.SingleLine = true
	st.dateEditor.SingleLine = true
	st.subtitleEditor.SingleLine = true
	st.ageGroupEditor.SingleLine = true
	return st
}

// SetExercises updates the full exercise list (library column).
func (st *SessionTab) SetExercises(items []ManagedExercise) {
	st.items = items
}

// SetSession sets the current session.
func (st *SessionTab) SetSession(s *model.Session) {
	st.session = s
	st.metaSynced = false
	st.modified = false
}

// Session returns the current session.
func (st *SessionTab) Session() *model.Session {
	return st.session
}

// Modified returns whether the session has unsaved changes.
func (st *SessionTab) Modified() bool {
	return st.modified
}

// ClearModified clears the modified flag.
func (st *SessionTab) ClearModified() {
	st.modified = false
}

// SetResolvedExercises sets resolved exercises for display.
func (st *SessionTab) SetResolvedExercises(resolved map[string]*model.Exercise) {
	st.resolvedExercises = resolved
}

// SessionListOverlay returns the session list overlay for external control.
func (st *SessionTab) SessionListOverlay() *SessionListOverlay {
	return st.sessionListOverlay
}

// HandleActions reads clickables and returns the first event, if any.
func (st *SessionTab) HandleActions(gtx layout.Context) SessionTabEvent {
	// Session file buttons.
	if st.newClick.Clicked(gtx) {
		return SessionTabEvent{Action: SessionTabActionNew}
	}
	if st.openClick.Clicked(gtx) {
		return SessionTabEvent{Action: SessionTabActionOpen}
	}
	if st.saveClick.Clicked(gtx) {
		return SessionTabEvent{Action: SessionTabActionSave}
	}
	if st.generateClick.Clicked(gtx) {
		return SessionTabEvent{Action: SessionTabActionGenerate}
	}

	// Refresh button.
	if st.refreshClick.Clicked(gtx) {
		return SessionTabEvent{Action: SessionTabActionRefresh}
	}

	// Filter buttons.
	for i := range st.filterClicks {
		if st.filterClicks[i].Clicked(gtx) {
			st.filterIndex = i
		}
	}

	// Row selection in library.
	filtered := st.filteredExercises()
	for i := 0; i < len(filtered) && i < maxMgrItems; i++ {
		if st.rowClicks[i].Clicked(gtx) {
			name := filtered[i].Name
			if name != st.selectedName {
				st.selectedName = name
				st.selectedStatus = filtered[i].Status
				ex := filtered[i].LocalEx
				if ex == nil {
					ex = filtered[i].RemoteEx
				}
				st.previewCourt.SetExercise(ex)
			}
		}
	}

	// "Add to session" button.
	if st.addToSessionClick.Clicked(gtx) && st.selectedName != "" && st.session != nil {
		st.addExerciseByRef(st.selectedName)
	}

	// Preview column management buttons (singular, for selected exercise).
	if st.openExerciseClick.Clicked(gtx) && st.selectedName != "" {
		return SessionTabEvent{Action: SessionTabActionOpenExercise, Name: st.selectedName}
	}
	if st.updateClick.Clicked(gtx) && st.selectedName != "" {
		return SessionTabEvent{Action: SessionTabActionUpdate, Name: st.selectedName}
	}
	if st.contributeClick.Clicked(gtx) && st.selectedName != "" {
		return SessionTabEvent{Action: SessionTabActionContribute, Name: st.selectedName}
	}
	if st.deleteClick.Clicked(gtx) && st.selectedName != "" {
		return SessionTabEvent{Action: SessionTabActionDeleteExercise, Name: st.selectedName}
	}

	return SessionTabEvent{Action: SessionTabActionNone}
}

// Layout renders the session tab with 3-column layout.
func (st *SessionTab) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	if !st.initialized {
		st.sessionList.Axis = layout.Vertical
		st.sessionScrollList.Axis = layout.Vertical
		st.initialized = true
	}

	// Process popup clicks from previous frame.
	if key, ok := st.catPopup.Update(gtx); ok {
		st.filterCategory = model.Category(key)
	}
	if key, ok := st.agePopup.Update(gtx); ok {
		st.filterAgeGroup = model.AgeGroup(key)
	}
	if key, ok := st.tagPopup.Update(gtx); ok && key != "" {
		st.filterTags = append(st.filterTags, key)
	}

	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			paint.FillShape(gtx.Ops, theme.ColorDarkBg, clip.Rect{Max: gtx.Constraints.Max}.Op())

			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				// Toolbar row.
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return st.layoutToolbar(gtx, th)
				}),
				// 3-column content.
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						// Left: library (~30%).
						layout.Flexed(0.30, func(gtx layout.Context) layout.Dimensions {
							return st.layoutLibrary(gtx, th)
						}),
						// Center: preview (~35%).
						layout.Flexed(0.35, func(gtx layout.Context) layout.Dimensions {
							return st.layoutPreview(gtx, th)
						}),
						// Right: session (~35%).
						layout.Flexed(0.35, func(gtx layout.Context) layout.Dimensions {
							return st.layoutSessionPanel(gtx, th)
						}),
					)
				}),
			)
		}),
		// Session list overlay.
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			if st.sessionListOverlay == nil || !st.sessionListOverlay.Visible {
				return layout.Dimensions{}
			}
			dims, selected := st.sessionListOverlay.Layout(gtx, th)
			if selected != "" {
				st.sessionListOverlay.OnSelect = selected
			}
			return dims
		}),
	)
}

// --- Toolbar ---

func (st *SessionTab) layoutToolbar(gtx layout.Context, th *material.Theme) layout.Dimensions {
	barH := gtx.Dp(unit.Dp(28))
	bg := color.NRGBA{R: 0x2e, G: 0x2e, B: 0x2e, A: 0xff}
	paint.FillShape(gtx.Ops, bg, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, barH)}.Op())

	saveColor := theme.ColorTabText
	if st.modified {
		saveColor = theme.ColorCoach
	}

	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return icon.IconBtnTooltip(gtx, th, &st.newClick, icon.New, theme.ColorTabText, i18n.T("tooltip.new"))
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return icon.IconBtnTooltip(gtx, th, &st.openClick, icon.Open, theme.ColorTabText, i18n.T("tooltip.open"))
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return icon.IconBtnTooltip(gtx, th, &st.saveClick, icon.Save, saveColor, i18n.T("tooltip.save"))
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return icon.IconBtnTooltip(gtx, th, &st.refreshClick, icon.Refresh, theme.ColorTabText, i18n.T("mgr.refresh"))
		}),
		// Spacer.
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 0)}
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return icon.IconBtnTooltip(gtx, th, &st.generateClick, icon.PDF, theme.ColorCoach, i18n.T("tooltip.generate_pdf"))
			})
		}),
	)
}

// --- Library column (left, ~30%) ---

func (st *SessionTab) layoutLibrary(gtx layout.Context, th *material.Theme) layout.Dimensions {
	bg := color.NRGBA{R: 0x2c, G: 0x2c, B: 0x2c, A: 0xff}
	paint.FillShape(gtx.Ops, bg, clip.Rect{Max: gtx.Constraints.Max}.Op())

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Header.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layoutSectionTitle(gtx, th, i18n.T("session.exercise_library"))
		}),
		// Search bar.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8), Bottom: unit.Dp(4)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					edBg := color.NRGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xff}
					return layoutEditorWithBg(gtx, th, &st.searchEditor, edBg)
				},
			)
		}),
		// Status filter chips.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			filterLabels := []string{
				i18n.T("mgr.filter_all"),
				i18n.T("mgr.filter_local"),
				i18n.T("mgr.filter_remote"),
				i18n.T("mgr.filter_synced"),
				i18n.T("mgr.filter_modified"),
			}
			return layout.Inset{Left: unit.Dp(8), Bottom: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					st.filterChips(th, filterLabels)...,
				)
			})
		}),
		// Category filter.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if st.categoryFilter.Clicked(gtx) {
				st.catPopup.Show(categoryPopupOptions())
			}
			label := i18n.T("session.category_all")
			if st.filterCategory != "" {
				label = i18n.T("category." + string(st.filterCategory))
			}
			dims := layout.Inset{Left: unit.Dp(8), Bottom: unit.Dp(4)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return material.Clickable(gtx, &st.categoryFilter, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(11), i18n.Tf("session.category_label", label))
						lbl.Color = theme.ColorTabText
						return lbl.Layout(gtx)
					})
				},
			)
			if st.catPopup.Visible {
				st.catPopup.LayoutBelow(gtx, th, dims)
			}
			return dims
		}),
		// Age group filter.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if st.ageGroupFilter.Clicked(gtx) {
				st.agePopup.Show(ageGroupPopupOptions())
			}
			label := i18n.T("session.category_all")
			if st.filterAgeGroup != "" {
				label = i18n.T("age_group." + string(st.filterAgeGroup))
			}
			dims := layout.Inset{Left: unit.Dp(8), Bottom: unit.Dp(4)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return material.Clickable(gtx, &st.ageGroupFilter, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(11), i18n.Tf("session.age_group_label", label))
						lbl.Color = theme.ColorTabText
						return lbl.Layout(gtx)
					})
				},
			)
			if st.agePopup.Visible {
				st.agePopup.LayoutBelow(gtx, th, dims)
			}
			return dims
		}),
		// Tag filter: [+ Tag] button + active tag chips with ×.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			// Process tag chip removals.
			for i := 0; i < len(st.filterTags) && i < maxTagFilters; i++ {
				if st.tagRemoveClicks[i].Clicked(gtx) {
					st.filterTags = append(st.filterTags[:i], st.filterTags[i+1:]...)
					break
				}
			}
			// [+ Tag] button opens popup.
			if st.addTagClick.Clicked(gtx) {
				st.tagPopup.Show(st.tagPopupOptions())
			}

			return layout.Inset{Left: unit.Dp(8), Bottom: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					children := make([]layout.FlexChild, 0, len(st.filterTags)+1)
					// Active tag chips.
					for i := 0; i < len(st.filterTags) && i < maxTagFilters; i++ {
						idx := i
						tag := st.filterTags[i]
						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return st.layoutTagChip(gtx, th, idx, tag)
						}))
					}
					// [+ Tag] button.
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						dims := material.Clickable(gtx, &st.addTagClick, func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(th, unit.Sp(11), i18n.T("session.add_tag"))
							lbl.Color = theme.ColorCoach
							return lbl.Layout(gtx)
						})
						if st.tagPopup.Visible {
							st.tagPopup.LayoutBelow(gtx, th, dims)
						}
						return dims
					}))
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
				},
			)
		}),
		// Exercise list.
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			filtered := st.filteredExercises()
			count := len(filtered)
			if count > maxMgrItems {
				count = maxMgrItems
			}
			if count == 0 {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, unit.Sp(12), i18n.T("mgr.no_exercises"))
					lbl.Color = theme.ColorTabText
					return lbl.Layout(gtx)
				})
			}
			return material.List(th, &st.scrollList).Layout(gtx, count, func(gtx layout.Context, idx int) layout.Dimensions {
				item := filtered[idx]
				return st.layoutLibraryRow(gtx, th, idx, item)
			})
		}),
	)
}

func (st *SessionTab) filterChips(th *material.Theme, labels []string) []layout.FlexChild {
	children := make([]layout.FlexChild, len(labels))
	for i, label := range labels {
		idx := i
		lbl := label
		children[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Clickable(gtx, &st.filterClicks[idx], func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(2), Right: unit.Dp(2), Top: unit.Dp(2), Bottom: unit.Dp(2)}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						col := theme.ColorTabText
						if st.filterIndex == idx {
							col = theme.ColorTabActive
						}
						l := material.Label(th, unit.Sp(12), lbl)
						l.Color = col
						return l.Layout(gtx)
					},
				)
			})
		})
	}
	return children
}

// layoutTagChip renders a single active tag chip with a × remove button.
func (st *SessionTab) layoutTagChip(gtx layout.Context, th *material.Theme, idx int, tag string) layout.Dimensions {
	chipBg := color.NRGBA{R: 0x3a, G: 0x5a, B: 0x8a, A: 0xff}
	return layout.Inset{Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return material.Clickable(gtx, &st.tagRemoveClicks[idx], func(gtx layout.Context) layout.Dimensions {
			return layout.Background{}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					rr := gtx.Dp(unit.Dp(8))
					paint.FillShape(gtx.Ops, chipBg,
						clip.RRect{
							Rect: image.Rectangle{Max: gtx.Constraints.Min},
							NE:   rr, NW: rr, SE: rr, SW: rr,
						}.Op(gtx.Ops))
					return layout.Dimensions{Size: gtx.Constraints.Min}
				},
				func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx,
						func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(th, unit.Sp(10), tag+" ×")
							lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
							return lbl.Layout(gtx)
						},
					)
				},
			)
		})
	})
}

func (st *SessionTab) layoutLibraryRow(gtx layout.Context, th *material.Theme, idx int, item ManagedExercise) layout.Dimensions {
	var rowBg color.NRGBA
	if item.Name == st.selectedName {
		rowBg = color.NRGBA{R: 0x3a, G: 0x3a, B: 0x50, A: 0xff}
	} else if idx%2 == 0 {
		rowBg = color.NRGBA{R: 0x2e, G: 0x2e, B: 0x2e, A: 0xff}
	} else {
		rowBg = color.NRGBA{R: 0x26, G: 0x26, B: 0x26, A: 0xff}
	}
	paint.FillShape(gtx.Ops, rowBg, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(36)))}.Op())

	return material.Clickable(gtx, &st.rowClicks[idx], func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(8), Right: unit.Dp(4)}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					// Name.
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						displayName := item.DisplayName
						if displayName == "" {
							displayName = item.Name
						}
						lbl := material.Label(th, unit.Sp(12), displayName)
						lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
						return lbl.Layout(gtx)
					}),
					// Status badge.
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return st.layoutBadge(gtx, th, item.Status)
					}),
				)
			},
		)
	})
}

func (st *SessionTab) layoutBadge(gtx layout.Context, th *material.Theme, status ExerciseSyncStatus) layout.Dimensions {
	label, col := statusLabelColor(status)

	padH := gtx.Dp(unit.Dp(4))
	padV := gtx.Dp(unit.Dp(1))

	lbl := material.Label(th, unit.Sp(9), label)
	lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}

	return layout.Stack{Alignment: layout.Center}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			sz := gtx.Constraints.Min
			r := gtx.Dp(unit.Dp(3))
			bounds := image.Rect(0, 0, sz.X+2*padH, sz.Y+2*padV)
			paint.FillShape(gtx.Ops, col, clip.RRect{Rect: bounds, NE: r, NW: r, SE: r, SW: r}.Op(gtx.Ops))
			return layout.Dimensions{Size: bounds.Max}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(1), Bottom: unit.Dp(1), Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx,
				lbl.Layout,
			)
		}),
	)
}

// --- Preview column (center, ~35%) ---

func (st *SessionTab) layoutPreview(gtx layout.Context, th *material.Theme) layout.Dimensions {
	bg := color.NRGBA{R: 0x2a, G: 0x2a, B: 0x2a, A: 0xff}
	paint.FillShape(gtx.Ops, bg, clip.Rect{Max: gtx.Constraints.Max}.Op())

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Top action buttons: Add to session + Open in editor.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if st.selectedName == "" {
				return layout.Dimensions{}
			}
			return layout.Inset{Top: unit.Dp(4), Left: unit.Dp(8), Right: unit.Dp(8), Bottom: unit.Dp(4)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return icon.IconTextBtn(gtx, th, &st.addToSessionClick, icon.Add, i18n.T("session.add_to_session"), theme.ColorCoach)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return icon.IconTextBtn(gtx, th, &st.openExerciseClick, icon.Open, i18n.T("mgr.open"), white)
							})
						}),
					)
				},
			)
		}),
		// Court preview.
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return st.previewCourt.Layout(gtx, th)
		}),
		// Contextual management buttons (bottom).
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if st.selectedName == "" {
				return layout.Dimensions{}
			}
			return st.layoutMgmtButtons(gtx, th)
		}),
	)
}

func (st *SessionTab) layoutMgmtButtons(gtx layout.Context, th *material.Theme) layout.Dimensions {
	white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	deleteCol := color.NRGBA{R: 0xff, G: 0x60, B: 0x60, A: 0xff}

	var children []layout.FlexChild

	switch st.selectedStatus {
	case StatusLocalOnly:
		children = append(children,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon.IconTextBtn(gtx, th, &st.contributeClick, icon.Upload, i18n.T("mgr.contribute"), white)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon.IconTextBtn(gtx, th, &st.deleteClick, icon.Delete, i18n.T("mgr.delete"), deleteCol)
			}),
		)
	case StatusSynced:
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return icon.IconTextBtn(gtx, th, &st.deleteClick, icon.Delete, i18n.T("mgr.delete"), deleteCol)
		}))
	case StatusModified:
		children = append(children,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon.IconTextBtn(gtx, th, &st.updateClick, icon.Sync, i18n.T("mgr.update"), white)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon.IconTextBtn(gtx, th, &st.contributeClick, icon.Upload, i18n.T("mgr.contribute"), white)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon.IconTextBtn(gtx, th, &st.deleteClick, icon.Delete, i18n.T("mgr.delete"), deleteCol)
			}),
		)
	}

	if len(children) == 0 {
		return layout.Dimensions{}
	}

	return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8), Top: unit.Dp(4), Bottom: unit.Dp(4)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx, children...)
		},
	)
}

// --- Session column (right, ~35%) ---

func (st *SessionTab) layoutSessionPanel(gtx layout.Context, th *material.Theme) layout.Dimensions {
	bg := color.NRGBA{R: 0x30, G: 0x30, B: 0x30, A: 0xff}
	paint.FillShape(gtx.Ops, bg, clip.Rect{Max: gtx.Constraints.Max}.Op())

	if st.session == nil {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, unit.Sp(14), i18n.T("tab.no_session"))
			lbl.Color = theme.ColorTabText
			return lbl.Layout(gtx)
		})
	}

	// Sync editors.
	if !st.metaSynced {
		st.titleEditor.SetText(st.session.Title)
		st.dateEditor.SetText(st.session.Date)
		st.subtitleEditor.SetText(st.session.Subtitle)
		st.ageGroupEditor.SetText(st.session.AgeGroup)
		st.philEditor.SetText(st.session.Philosophy)
		for i := 0; i < len(st.session.CoachNotes) && i < maxCoachNotes; i++ {
			st.noteEditors[i].SetText(st.session.CoachNotes[i])
		}
		st.metaSynced = true
	}

	// Process text changes.
	st.processEditorChanges(gtx)

	// Build items list.
	var items []func(gtx layout.Context) layout.Dimensions

	// Header.
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return layoutSectionTitle(gtx, th, i18n.T("session.header"))
	})

	// Metadata fields.
	items = append(items, st.metaField(th, i18n.T("session.title"), &st.titleEditor)...)
	items = append(items, st.dateField(gtx, th)...)
	items = append(items, st.metaField(th, i18n.T("session.subtitle"), &st.subtitleEditor)...)
	items = append(items, st.metaField(th, i18n.T("session.age_group"), &st.ageGroupEditor)...)

	// Exercise list header.
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return layoutSectionTitle(gtx, th, i18n.T("session.exercises"))
	})

	// Exercise entries.
	exCount := len(st.session.Exercises)
	if exCount > maxSessionItems {
		exCount = maxSessionItems
	}
	for i := 0; i < exCount; i++ {
		idx := i
		entry := &st.session.Exercises[idx]
		if st.removeClicks[idx].Clicked(gtx) {
			st.session.Exercises = append(st.session.Exercises[:idx], st.session.Exercises[idx+1:]...)
			st.modified = true
			break
		}
		items = append(items, func(gtx layout.Context) layout.Dimensions {
			return st.layoutSessionExercise(gtx, th, idx, entry)
		})
		for vi := range entry.Variants {
			vIdx := vi
			variant := &entry.Variants[vIdx]
			items = append(items, func(gtx layout.Context) layout.Dimensions {
				return st.layoutVariant(gtx, th, variant)
			})
		}
	}

	// Total duration.
	totalDuration := st.computeTotalDuration()
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(4), Left: unit.Dp(8)}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, unit.Sp(11), i18n.Tf("session.total_format", totalDuration))
				lbl.Color = theme.ColorCoach
				return lbl.Layout(gtx)
			},
		)
	})

	// Coach notes header.
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return layoutSectionTitle(gtx, th, i18n.T("session.coach_notes"))
	})

	// Coach notes.
	for i := 0; i < len(st.session.CoachNotes) && i < maxCoachNotes; i++ {
		idx := i
		items = append(items, func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8), Top: unit.Dp(1)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							edBg := color.NRGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xff}
							return layoutEditorWithBg(gtx, th, &st.noteEditors[idx], edBg)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if st.delNoteClicks[idx].Clicked(gtx) {
								st.session.CoachNotes = append(st.session.CoachNotes[:idx], st.session.CoachNotes[idx+1:]...)
								st.modified = true
								st.metaSynced = false
							}
							return icon.IconBtn(gtx, &st.delNoteClicks[idx], icon.Close, color.NRGBA{R: 0xff, G: 0x60, B: 0x60, A: 0xff})
						}),
					)
				},
			)
		})
	}

	// Add note button.
	if st.addNoteClick.Clicked(gtx) && st.session != nil && len(st.session.CoachNotes) < maxCoachNotes {
		st.session.CoachNotes = append(st.session.CoachNotes, "")
		st.metaSynced = false
		st.modified = true
	}
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: unit.Dp(4), Top: unit.Dp(2)}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				return icon.IconTextBtn(gtx, th, &st.addNoteClick, icon.Add, i18n.T("session.add_note"), theme.ColorCoach)
			},
		)
	})

	// Philosophy.
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return layoutSectionTitle(gtx, th, i18n.T("session.philosophy"))
	})
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				edBg := color.NRGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xff}
				return layoutEditorWithBg(gtx, th, &st.philEditor, edBg)
			},
		)
	})

	return material.List(th, &st.sessionScrollList).Layout(gtx, len(items), func(gtx layout.Context, idx int) layout.Dimensions {
		return items[idx](gtx)
	})
}

func (st *SessionTab) processEditorChanges(gtx layout.Context) {
	if st.session == nil {
		return
	}
	for {
		evt, ok := st.titleEditor.Update(gtx)
		if !ok {
			break
		}
		if _, isChange := evt.(widget.ChangeEvent); isChange {
			st.session.Title = st.titleEditor.Text()
			st.modified = true
		}
	}
	for {
		evt, ok := st.dateEditor.Update(gtx)
		if !ok {
			break
		}
		if _, isChange := evt.(widget.ChangeEvent); isChange {
			st.session.Date = st.dateEditor.Text()
			st.modified = true
		}
	}
	for {
		evt, ok := st.subtitleEditor.Update(gtx)
		if !ok {
			break
		}
		if _, isChange := evt.(widget.ChangeEvent); isChange {
			st.session.Subtitle = st.subtitleEditor.Text()
			st.modified = true
		}
	}
	for {
		evt, ok := st.ageGroupEditor.Update(gtx)
		if !ok {
			break
		}
		if _, isChange := evt.(widget.ChangeEvent); isChange {
			st.session.AgeGroup = st.ageGroupEditor.Text()
			st.modified = true
		}
	}
	for {
		evt, ok := st.philEditor.Update(gtx)
		if !ok {
			break
		}
		if _, isChange := evt.(widget.ChangeEvent); isChange {
			st.session.Philosophy = st.philEditor.Text()
			st.modified = true
		}
	}
	for i := 0; i < len(st.session.CoachNotes) && i < maxCoachNotes; i++ {
		for {
			evt, ok := st.noteEditors[i].Update(gtx)
			if !ok {
				break
			}
			if _, isChange := evt.(widget.ChangeEvent); isChange {
				st.session.CoachNotes[i] = st.noteEditors[i].Text()
				st.modified = true
			}
		}
	}
}

func (st *SessionTab) metaField(th *material.Theme, label string, ed *widget.Editor) []func(layout.Context) layout.Dimensions {
	return []func(layout.Context) layout.Dimensions{
		func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(15), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(th, unit.Sp(10), label)
							lbl.Color = theme.ColorTabText
							return lbl.Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Top: unit.Dp(2)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								edBg := color.NRGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xff}
								return layoutEditorWithBg(gtx, th, ed, edBg)
							})
						}),
					)
				},
			)
		},
	}
}

func (st *SessionTab) dateField(gtx layout.Context, th *material.Theme) []func(layout.Context) layout.Dimensions {
	if st.todayClick.Clicked(gtx) && st.session != nil {
		today := time.Now().Format("2006-01-02")
		st.dateEditor.SetText(today)
		st.session.Date = today
		st.modified = true
	}

	if st.calendarClick.Clicked(gtx) && st.session != nil {
		t, err := time.Parse("2006-01-02", st.dateEditor.Text())
		if err != nil {
			t = time.Now()
		}
		st.datePicker.Show(t)
	}

	if st.datePicker.Result != "" && st.session != nil {
		st.dateEditor.SetText(st.datePicker.Result)
		st.session.Date = st.datePicker.Result
		st.modified = true
		st.datePicker.Result = ""
	}

	return []func(layout.Context) layout.Dimensions{
		func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Top: unit.Dp(15), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx,
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
									txt := st.dateEditor.Text()
									if txt != "" && !isValidDate(txt) {
										edBg = color.NRGBA{R: 0x60, G: 0x30, B: 0x30, A: 0xff}
									}
									return layoutEditorWithBg(gtx, th, &st.dateEditor, edBg)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return icon.IconBtnTooltip(gtx, th, &st.calendarClick, icon.Calendar, theme.ColorCoach, i18n.T("session.calendar"))
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return icon.IconBtnTooltip(gtx, th, &st.todayClick, icon.Today, theme.ColorCoach, i18n.T("session.today"))
								}),
							)
						}),
					)

					if st.datePicker.Visible {
						st.datePicker.LayoutDropdown(gtx, th, dims)
					}

					return dims
				},
			)
		},
	}
}

func (st *SessionTab) layoutSessionExercise(gtx layout.Context, th *material.Theme, idx int, entry *model.ExerciseEntry) layout.Dimensions {
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
					if ex, ok := st.resolvedExercises[entry.Exercise]; ok {
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
				return icon.IconBtn(gtx, &st.removeClicks[idx], icon.Close, color.NRGBA{R: 0xff, G: 0x60, B: 0x60, A: 0xff})
			}),
		)
	})
}

func (st *SessionTab) layoutVariant(gtx layout.Context, th *material.Theme, variant *model.ExerciseEntry) layout.Dimensions {
	return layout.Inset{
		Top: unit.Dp(1), Bottom: unit.Dp(1),
		Left: unit.Dp(24), Right: unit.Dp(8),
	}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		name := variant.Exercise
		if ex, ok := st.resolvedExercises[variant.Exercise]; ok {
			name = ex.Name
		}
		lbl := material.Label(th, unit.Sp(11), "-> "+name)
		lbl.Color = theme.ColorTabText
		return lbl.Layout(gtx)
	})
}

func (st *SessionTab) computeTotalDuration() string {
	total := 0
	for _, entry := range st.session.Exercises {
		if ex, ok := st.resolvedExercises[entry.Exercise]; ok {
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

// addExerciseByRef adds an exercise entry by its store reference (kebab name).
func (st *SessionTab) addExerciseByRef(ref string) {
	if st.session == nil || len(st.session.Exercises) >= maxSessionItems {
		return
	}
	for _, entry := range st.session.Exercises {
		if entry.Exercise == ref {
			return
		}
	}
	st.session.Exercises = append(st.session.Exercises, model.ExerciseEntry{Exercise: ref})
	st.modified = true
}

// --- Shared helpers (migrated from exercisemgr.go and sessioncomposer.go) ---

func (st *SessionTab) filteredExercises() []ManagedExercise {
	search := strings.ToLower(strings.TrimSpace(st.searchEditor.Text()))

	var result []ManagedExercise
	for _, item := range st.items {
		// Status filter.
		switch st.filterIndex {
		case 1:
			if item.Status != StatusLocalOnly {
				continue
			}
		case 2:
			if item.Status != StatusRemoteOnly {
				continue
			}
		case 3:
			if item.Status != StatusSynced {
				continue
			}
		case 4:
			if item.Status != StatusModified {
				continue
			}
		}

		// Category filter.
		if st.filterCategory != "" && model.Category(item.Category) != st.filterCategory {
			continue
		}

		// Age group filter.
		if st.filterAgeGroup != "" && model.AgeGroup(item.AgeGroup) != st.filterAgeGroup {
			continue
		}

		// Tag filter (intersection: exercise must have ALL selected tags).
		if len(st.filterTags) > 0 {
			tagSet := make(map[string]bool, len(item.Tags))
			for _, t := range item.Tags {
				tagSet[t] = true
			}
			allMatch := true
			for _, ft := range st.filterTags {
				if !tagSet[ft] {
					allMatch = false
					break
				}
			}
			if !allMatch {
				continue
			}
		}

		// Search filter (matches name, display name, category, tags).
		if search != "" {
			name := strings.ToLower(item.Name)
			display := strings.ToLower(item.DisplayName)
			cat := strings.ToLower(item.Category)
			match := strings.Contains(name, search) || strings.Contains(display, search) || strings.Contains(cat, search)
			if !match {
				for _, tag := range item.Tags {
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

		result = append(result, item)
	}
	return result
}

// tagPopupOptions builds a sorted list of unique tags from all exercises,
// excluding tags already selected as filters.
func (st *SessionTab) tagPopupOptions() []PopupOption {
	// Build set of already-selected tags.
	selected := make(map[string]bool, len(st.filterTags))
	for _, t := range st.filterTags {
		selected[t] = true
	}
	seen := make(map[string]bool)
	for _, item := range st.items {
		for _, tag := range item.Tags {
			if !selected[tag] {
				seen[tag] = true
			}
		}
	}
	tags := make([]string, 0, len(seen))
	for tag := range seen {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	opts := make([]PopupOption, 0, len(tags))
	for _, tag := range tags {
		opts = append(opts, PopupOption{Key: tag, Label: tag})
	}
	return opts
}

// --- Shared types and helpers ---

func statusLabelColor(status ExerciseSyncStatus) (string, color.NRGBA) {
	switch status {
	case StatusLocalOnly:
		return i18n.T("mgr.status_local"), color.NRGBA{R: 0xe0, G: 0x8a, B: 0x20, A: 0xff}
	case StatusRemoteOnly:
		return i18n.T("mgr.status_remote"), color.NRGBA{R: 0x2a, G: 0x6f, B: 0xdb, A: 0xff}
	case StatusSynced:
		return i18n.T("mgr.status_synced"), color.NRGBA{R: 0x2d, G: 0x8a, B: 0x4e, A: 0xff}
	case StatusModified:
		return i18n.T("mgr.status_modified"), color.NRGBA{R: 0xc1, G: 0x12, B: 0x1f, A: 0xff}
	default:
		return "?", theme.ColorTabText
	}
}

func intensityDots(n int) string {
	dots := ""
	for i := 0; i < 3; i++ {
		if i < n {
			dots += "\u25cf"
		} else {
			dots += "\u25cb"
		}
	}
	return dots
}

func parseDurationMinutes(d string) int {
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

func isValidDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

func nextCategoryWithAll(current model.Category) model.Category {
	cats := []model.Category{
		"",
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
