package widget

import (
	"image"
	"image/color"
	"strings"

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

// ManagedExercise is an entry in the exercise manager list.
type ManagedExercise struct {
	Name      string             // kebab-case key
	Status    ExerciseSyncStatus
	LocalEx   *model.Exercise
	RemoteEx  *model.Exercise
	DisplayName string
	Category  string
	Duration  string
}

// MgrAction represents an action the user performed in the manager.
type MgrAction int

const (
	MgrActionNone       MgrAction = iota
	MgrActionOpen
	MgrActionImport
	MgrActionUpdate
	MgrActionContribute
	MgrActionRefresh
)

// MgrEvent is returned when the user performs an action on an exercise.
type MgrEvent struct {
	Action MgrAction
	Name   string
}

const maxMgrItems = 200

// ExerciseManager is the widget for the exercise management tab.
type ExerciseManager struct {
	items    []ManagedExercise

	scrollList   widget.List
	searchEditor widget.Editor

	// Status filter: 0=All, 1=Local, 2=Community, 3=Synced, 4=Modified
	filterIndex  int
	filterClicks [5]widget.Clickable

	refreshClick widget.Clickable

	openClicks       [maxMgrItems]widget.Clickable
	importClicks     [maxMgrItems]widget.Clickable
	updateClicks     [maxMgrItems]widget.Clickable
	contributeClicks [maxMgrItems]widget.Clickable
}

// NewExerciseManager creates an initialized ExerciseManager.
func NewExerciseManager() *ExerciseManager {
	mgr := &ExerciseManager{}
	mgr.scrollList.Axis = layout.Vertical
	mgr.searchEditor.SingleLine = true
	return mgr
}

// SetExercises updates the full exercise list.
func (mgr *ExerciseManager) SetExercises(items []ManagedExercise) {
	mgr.items = items
}

// HandleActions reads clickables and returns the first event, if any.
func (mgr *ExerciseManager) HandleActions(gtx layout.Context) MgrEvent {
	// Refresh button.
	if mgr.refreshClick.Clicked(gtx) {
		return MgrEvent{Action: MgrActionRefresh}
	}

	// Filter buttons.
	for i := range mgr.filterClicks {
		if mgr.filterClicks[i].Clicked(gtx) {
			mgr.filterIndex = i
		}
	}

	// Per-row action buttons.
	filtered := mgr.filteredExercises()
	for i := 0; i < len(filtered) && i < maxMgrItems; i++ {
		if mgr.openClicks[i].Clicked(gtx) {
			return MgrEvent{Action: MgrActionOpen, Name: filtered[i].Name}
		}
		if mgr.importClicks[i].Clicked(gtx) {
			return MgrEvent{Action: MgrActionImport, Name: filtered[i].Name}
		}
		if mgr.updateClicks[i].Clicked(gtx) {
			return MgrEvent{Action: MgrActionUpdate, Name: filtered[i].Name}
		}
		if mgr.contributeClicks[i].Clicked(gtx) {
			return MgrEvent{Action: MgrActionContribute, Name: filtered[i].Name}
		}
	}

	return MgrEvent{Action: MgrActionNone}
}

// Layout renders the exercise manager.
func (mgr *ExerciseManager) Layout(gtx layout.Context, th *material.Theme) layout.Dimensions {
	// Fill the entire area with a dark background.
	paint.FillShape(gtx.Ops, theme.ColorDarkBg, clip.Rect{Max: gtx.Constraints.Max}.Op())

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Toolbar: search + filters + refresh.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return mgr.layoutToolbar(gtx, th)
		}),
		// Exercise list.
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return mgr.layoutList(gtx, th)
		}),
	)
}

func (mgr *ExerciseManager) layoutToolbar(gtx layout.Context, th *material.Theme) layout.Dimensions {
	barH := gtx.Dp(unit.Dp(44))
	rect := clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, barH)}.Op()
	paint.FillShape(gtx.Ops, color.NRGBA{R: 0x30, G: 0x30, B: 0x30, A: 0xff}, rect)

	return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(12), Right: unit.Dp(8)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				// Search box with dark background.
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Right: unit.Dp(12)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Max.X = gtx.Dp(unit.Dp(200))
						gtx.Constraints.Min.X = gtx.Constraints.Max.X
						// Draw a rounded dark background for the editor.
						return layout.Background{}.Layout(gtx,
							func(gtx layout.Context) layout.Dimensions {
								r := gtx.Dp(unit.Dp(4))
								bounds := image.Rect(0, 0, gtx.Constraints.Min.X, gtx.Constraints.Min.Y)
								searchBg := color.NRGBA{R: 0x1a, G: 0x1a, B: 0x1a, A: 0xff}
								paint.FillShape(gtx.Ops, searchBg, clip.RRect{Rect: bounds, NE: r, NW: r, SE: r, SW: r}.Op(gtx.Ops))
								return layout.Dimensions{Size: bounds.Max}
							},
							func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									ed := material.Editor(th, &mgr.searchEditor, i18n.T("mgr.search"))
									ed.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
									ed.HintColor = color.NRGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff}
									return ed.Layout(gtx)
								})
							},
						)
					})
				}),
				// Filter buttons.
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					filterLabels := []string{
						i18n.T("mgr.filter_all"),
						i18n.T("mgr.filter_local"),
						i18n.T("mgr.filter_remote"),
						i18n.T("mgr.filter_synced"),
						i18n.T("mgr.filter_modified"),
					}
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						mgr.filterChips(th, filterLabels)...,
					)
				}),
				// Spacer.
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 0)}
				}),
				// Refresh button.
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return icon.IconBtnTooltip(gtx, th, &mgr.refreshClick, icon.Refresh,
						theme.ColorTabActive, i18n.T("mgr.refresh"))
				}),
			)
		},
	)
}

func (mgr *ExerciseManager) filterChips(th *material.Theme, labels []string) []layout.FlexChild {
	children := make([]layout.FlexChild, len(labels))
	for i, label := range labels {
		idx := i
		lbl := label
		children[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Clickable(gtx, &mgr.filterClicks[idx], func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Left: unit.Dp(2), Right: unit.Dp(2), Top: unit.Dp(2), Bottom: unit.Dp(2)}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						col := theme.ColorTabText
						if mgr.filterIndex == idx {
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

func (mgr *ExerciseManager) layoutList(gtx layout.Context, th *material.Theme) layout.Dimensions {
	filtered := mgr.filteredExercises()
	count := len(filtered)
	if count > maxMgrItems {
		count = maxMgrItems
	}

	if count == 0 {
		return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, unit.Sp(14), i18n.T("mgr.no_exercises"))
			lbl.Color = theme.ColorTabText
			return lbl.Layout(gtx)
		})
	}

	return material.List(th, &mgr.scrollList).Layout(gtx, count, func(gtx layout.Context, idx int) layout.Dimensions {
		item := filtered[idx]
		return mgr.layoutRow(gtx, th, idx, item)
	})
}

func (mgr *ExerciseManager) layoutRow(gtx layout.Context, th *material.Theme, idx int, item ManagedExercise) layout.Dimensions {
	// Alternate row background for contrast.
	var rowBg color.NRGBA
	if idx%2 == 0 {
		rowBg = color.NRGBA{R: 0x2e, G: 0x2e, B: 0x2e, A: 0xff}
	} else {
		rowBg = color.NRGBA{R: 0x26, G: 0x26, B: 0x26, A: 0xff}
	}
	paint.FillShape(gtx.Ops, rowBg, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(36)))}.Op())

	return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(12), Right: unit.Dp(8)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				// Name.
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					displayName := item.DisplayName
					if displayName == "" {
						displayName = item.Name
					}
					lbl := material.Label(th, unit.Sp(13), displayName)
					lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
					return lbl.Layout(gtx)
				}),
				// Category.
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if item.Category == "" {
						return layout.Dimensions{}
					}
					return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(11), i18n.T("category."+item.Category))
						lbl.Color = theme.ColorTabText
						return lbl.Layout(gtx)
					})
				}),
				// Duration.
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if item.Duration == "" {
						return layout.Dimensions{}
					}
					return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(th, unit.Sp(11), item.Duration)
						lbl.Color = theme.ColorTabText
						return lbl.Layout(gtx)
					})
				}),
				// Status badge.
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return mgr.layoutBadge(gtx, th, item.Status)
					})
				}),
				// Action buttons.
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return mgr.layoutActions(gtx, th, idx, item.Status)
				}),
			)
		},
	)
}

func (mgr *ExerciseManager) layoutBadge(gtx layout.Context, th *material.Theme, status ExerciseSyncStatus) layout.Dimensions {
	label, col := statusLabelColor(status)

	padH := gtx.Dp(unit.Dp(6))
	padV := gtx.Dp(unit.Dp(2))

	// Measure text.
	lbl := material.Label(th, unit.Sp(10), label)
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
			return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx,
				lbl.Layout,
			)
		}),
	)
}

func statusLabelColor(status ExerciseSyncStatus) (string, color.NRGBA) {
	switch status {
	case StatusLocalOnly:
		return i18n.T("mgr.status_local"), color.NRGBA{R: 0xe0, G: 0x8a, B: 0x20, A: 0xff} // orange
	case StatusRemoteOnly:
		return i18n.T("mgr.status_remote"), color.NRGBA{R: 0x2a, G: 0x6f, B: 0xdb, A: 0xff} // blue
	case StatusSynced:
		return i18n.T("mgr.status_synced"), color.NRGBA{R: 0x2d, G: 0x8a, B: 0x4e, A: 0xff} // green
	case StatusModified:
		return i18n.T("mgr.status_modified"), color.NRGBA{R: 0xc1, G: 0x12, B: 0x1f, A: 0xff} // red
	default:
		return "?", theme.ColorTabText
	}
}

func (mgr *ExerciseManager) layoutActions(gtx layout.Context, th *material.Theme, idx int, status ExerciseSyncStatus) layout.Dimensions {
	var children []layout.FlexChild

	white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}

	switch status {
	case StatusLocalOnly:
		children = append(children,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon.IconTextBtn(gtx, th, &mgr.openClicks[idx], icon.Open, i18n.T("mgr.open"), white)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon.IconTextBtn(gtx, th, &mgr.contributeClicks[idx], icon.Upload, i18n.T("mgr.contribute"), white)
			}),
		)
	case StatusRemoteOnly:
		children = append(children,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon.IconTextBtn(gtx, th, &mgr.importClicks[idx], icon.Import, i18n.T("mgr.import"), white)
			}),
		)
	case StatusSynced:
		children = append(children,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon.IconTextBtn(gtx, th, &mgr.openClicks[idx], icon.Open, i18n.T("mgr.open"), white)
			}),
		)
	case StatusModified:
		children = append(children,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon.IconTextBtn(gtx, th, &mgr.openClicks[idx], icon.Open, i18n.T("mgr.open"), white)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon.IconTextBtn(gtx, th, &mgr.updateClicks[idx], icon.Sync, i18n.T("mgr.update"), white)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return icon.IconTextBtn(gtx, th, &mgr.contributeClicks[idx], icon.Upload, i18n.T("mgr.contribute"), white)
			}),
		)
	}

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
}

func (mgr *ExerciseManager) filteredExercises() []ManagedExercise {
	search := strings.ToLower(strings.TrimSpace(mgr.searchEditor.Text()))

	var result []ManagedExercise
	for _, item := range mgr.items {
		// Status filter.
		switch mgr.filterIndex {
		case 1: // Local
			if item.Status != StatusLocalOnly {
				continue
			}
		case 2: // Community
			if item.Status != StatusRemoteOnly {
				continue
			}
		case 3: // Synced
			if item.Status != StatusSynced {
				continue
			}
		case 4: // Modified
			if item.Status != StatusModified {
				continue
			}
		}

		// Search filter.
		if search != "" {
			name := strings.ToLower(item.Name)
			display := strings.ToLower(item.DisplayName)
			cat := strings.ToLower(item.Category)
			if !strings.Contains(name, search) && !strings.Contains(display, search) && !strings.Contains(cat, search) {
				continue
			}
		}

		result = append(result, item)
	}
	return result
}
