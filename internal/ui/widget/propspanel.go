package widget

import (
	"fmt"
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
	"github.com/darkweaver87/courtdraw/internal/ui/editor"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
)

// PropertiesPanel is the right sidebar showing element properties and exercise metadata.
type PropertiesPanel struct {
	scrollList widget.List

	// Element editors.
	playerLabelEditor  widget.Editor
	playerRoleClick    widget.Clickable
	ballCarrierClick   widget.Clickable

	// Exercise metadata editors.
	nameEditor     widget.Editor
	durationEditor widget.Editor
	tagsEditor     widget.Editor

	courtStdClick  widget.Clickable
	courtTypeClick widget.Clickable
	categoryClick  widget.Clickable

	intensityClicks [4]widget.Clickable

	// Track which element we last synced to, to avoid overwriting user typing.
	syncedPlayerIdx int
	syncedKind      editor.SelectionKind
	syncedSeqIdx    int
	metaSynced      bool
}

// NewPropertiesPanel creates an initialized properties panel.
func NewPropertiesPanel() *PropertiesPanel {
	pp := &PropertiesPanel{
		syncedPlayerIdx: -1,
		syncedSeqIdx:    -1,
	}
	pp.scrollList.Axis = layout.Vertical
	pp.playerLabelEditor.SingleLine = true
	pp.nameEditor.SingleLine = true
	pp.durationEditor.SingleLine = true
	pp.tagsEditor.SingleLine = true
	return pp
}

// Layout renders the properties panel.
func (pp *PropertiesPanel) Layout(gtx layout.Context, th *material.Theme, exercise *model.Exercise, state *editor.EditorState, seqIndex int) layout.Dimensions {
	// Background.
	panelBg := color.NRGBA{R: 0x30, G: 0x30, B: 0x30, A: 0xff}
	size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, panelBg, clip.Rect{Max: size}.Op())

	if exercise == nil {
		return layout.Dimensions{Size: size}
	}

	// Sync metadata editors on first frame.
	if !pp.metaSynced {
		pp.nameEditor.SetText(exercise.Name)
		pp.durationEditor.SetText(exercise.Duration)
		pp.tagsEditor.SetText(strings.Join(exercise.Tags, ", "))
		pp.metaSynced = true
	}

	// Build list of items to render.
	var items []func(gtx layout.Context) layout.Dimensions

	// Element properties section (only shown if selection is in the active sequence).
	sel := state.SelectedElement
	if sel != nil && sel.SeqIndex == seqIndex && seqIndex < len(exercise.Sequences) {
		seq := &exercise.Sequences[seqIndex]

		// Sync editor text when selection changes.
		if sel.Kind != pp.syncedKind || sel.Index != pp.syncedPlayerIdx || sel.SeqIndex != pp.syncedSeqIdx {
			pp.syncedKind = sel.Kind
			pp.syncedPlayerIdx = sel.Index
			pp.syncedSeqIdx = sel.SeqIndex
			if sel.Kind == editor.SelectPlayer && sel.Index < len(seq.Players) {
				pp.playerLabelEditor.SetText(seq.Players[sel.Index].Label)
			}
		}

		items = append(items, func(gtx layout.Context) layout.Dimensions {
			return pp.layoutSectionHeader(gtx, th, i18n.T("props.element"))
		})

		switch sel.Kind {
		case editor.SelectPlayer:
			if sel.Index < len(seq.Players) {
				p := &seq.Players[sel.Index]
				items = append(items, pp.playerPropsItems(gtx, th, p, seq, state)...)
			}
		case editor.SelectAccessory:
			if sel.Index < len(seq.Accessories) {
				a := &seq.Accessories[sel.Index]
				items = append(items, pp.accessoryPropsItems(th, a)...)
			}
		case editor.SelectAction:
			if sel.Index < len(seq.Actions) {
				a := &seq.Actions[sel.Index]
				items = append(items, pp.actionPropsItems(th, a)...)
			}
		}

		items = append(items, func(gtx layout.Context) layout.Dimensions {
			return pp.layoutSeparator(gtx)
		})
	}

	// Exercise metadata section.
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return pp.layoutSectionHeader(gtx, th, i18n.T("props.exercise"))
	})
	items = append(items, pp.metadataItems(gtx, th, exercise, state)...)

	return material.List(th, &pp.scrollList).Layout(gtx, len(items), func(gtx layout.Context, idx int) layout.Dimensions {
		return items[idx](gtx)
	})
}

func (pp *PropertiesPanel) playerPropsItems(_ layout.Context, th *material.Theme, p *model.Player, seq *model.Sequence, state *editor.EditorState) []func(layout.Context) layout.Dimensions {
	var items []func(layout.Context) layout.Dimensions

	// Label editor.
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return pp.layoutField(gtx, th, i18n.T("props.label"), &pp.playerLabelEditor, func(text string) {
			p.Label = text
			state.MarkModified()
		})
	})

	// Role (cycle on click).
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		if pp.playerRoleClick.Clicked(gtx) {
			p.Role = nextPlayerRole(p.Role)
			state.MarkModified()
		}
		return pp.layoutClickField(gtx, th, i18n.T("props.role"), string(p.Role), &pp.playerRoleClick)
	})

	// Ball carrier toggle.
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		if pp.ballCarrierClick.Clicked(gtx) {
			if seq.BallCarrier == p.ID {
				seq.BallCarrier = ""
			} else {
				seq.BallCarrier = p.ID
			}
			state.MarkModified()
		}
		label := i18n.T("props.no_ball")
		if seq.BallCarrier == p.ID {
			label = i18n.T("props.has_ball")
		}
		return pp.layoutClickField(gtx, th, i18n.T("props.ball"), label, &pp.ballCarrierClick)
	})

	// Position (read-only).
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		posStr := fmt.Sprintf("(%.2f, %.2f)", p.Position.X(), p.Position.Y())
		return pp.layoutReadonly(gtx, th, i18n.T("props.position"), posStr)
	})

	return items
}

func (pp *PropertiesPanel) accessoryPropsItems(th *material.Theme, a *model.Accessory) []func(layout.Context) layout.Dimensions {
	var items []func(layout.Context) layout.Dimensions

	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return pp.layoutReadonly(gtx, th, i18n.T("props.type"), string(a.Type))
	})
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return pp.layoutReadonly(gtx, th, i18n.T("props.rotation"), fmt.Sprintf("%.0f°", a.Rotation))
	})

	return items
}

func (pp *PropertiesPanel) actionPropsItems(th *material.Theme, a *model.Action) []func(layout.Context) layout.Dimensions {
	var items []func(layout.Context) layout.Dimensions

	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return pp.layoutReadonly(gtx, th, i18n.T("props.type"), string(a.Type))
	})
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		fromStr := refString(a.From)
		return pp.layoutReadonly(gtx, th, i18n.T("props.from"), fromStr)
	})
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		toStr := refString(a.To)
		return pp.layoutReadonly(gtx, th, i18n.T("props.to"), toStr)
	})

	return items
}

func (pp *PropertiesPanel) metadataItems(_ layout.Context, th *material.Theme, ex *model.Exercise, state *editor.EditorState) []func(layout.Context) layout.Dimensions {
	var items []func(layout.Context) layout.Dimensions

	// Name.
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return pp.layoutField(gtx, th, i18n.T("props.name"), &pp.nameEditor, func(text string) {
			ex.Name = text
			state.MarkModified()
		})
	})

	// Court standard (toggle).
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		if pp.courtStdClick.Clicked(gtx) {
			if ex.CourtStandard == model.FIBA {
				ex.CourtStandard = model.NBA
			} else {
				ex.CourtStandard = model.FIBA
			}
			state.MarkModified()
		}
		return pp.layoutClickField(gtx, th, i18n.T("props.standard"), strings.ToUpper(string(ex.CourtStandard)), &pp.courtStdClick)
	})

	// Court type (toggle).
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		if pp.courtTypeClick.Clicked(gtx) {
			if ex.CourtType == model.HalfCourt {
				ex.CourtType = model.FullCourt
			} else {
				ex.CourtType = model.HalfCourt
			}
			state.MarkModified()
		}
		label := i18n.T("props.court_half")
		if ex.CourtType == model.FullCourt {
			label = i18n.T("props.court_full")
		}
		return pp.layoutClickField(gtx, th, i18n.T("props.court"), label, &pp.courtTypeClick)
	})

	// Duration.
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return pp.layoutField(gtx, th, i18n.T("props.duration"), &pp.durationEditor, func(text string) {
			ex.Duration = text
			state.MarkModified()
		})
	})

	// Intensity (0-3 dots).
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		for i := range pp.intensityClicks {
			if pp.intensityClicks[i].Clicked(gtx) {
				newLevel := model.Intensity(i + 1)
				if ex.Intensity == newLevel {
					ex.Intensity = 0 // toggle off if already at this level
				} else {
					ex.Intensity = newLevel
				}
				state.MarkModified()
			}
		}
		return pp.layoutIntensity(gtx, th, ex.Intensity)
	})

	// Category (cycle).
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		if pp.categoryClick.Clicked(gtx) {
			ex.Category = nextCategory(ex.Category)
			state.MarkModified()
		}
		label := i18n.T("category." + string(ex.Category))
		if ex.Category == "" {
			label = i18n.T("props.category_none")
		}
		return pp.layoutClickField(gtx, th, i18n.T("props.category"), label, &pp.categoryClick)
	})

	// Tags.
	items = append(items, func(gtx layout.Context) layout.Dimensions {
		return pp.layoutField(gtx, th, i18n.T("props.tags"), &pp.tagsEditor, func(text string) {
			parts := strings.Split(text, ",")
			tags := make([]string, 0, len(parts))
			for _, t := range parts {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
			ex.Tags = tags
			state.MarkModified()
		})
	})

	return items
}

// Layout helpers.

func (pp *PropertiesPanel) layoutSectionHeader(gtx layout.Context, th *material.Theme, title string) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(8), Left: unit.Dp(8), Bottom: unit.Dp(4)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, unit.Sp(12), title)
			lbl.Color = theme.ColorTabActive
			return lbl.Layout(gtx)
		},
	)
}

func (pp *PropertiesPanel) layoutSeparator(gtx layout.Context) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			h := gtx.Dp(unit.Dp(1))
			w := gtx.Constraints.Max.X
			paint.FillShape(gtx.Ops, theme.ColorTabText,
				clip.Rect{Max: image.Pt(w, h)}.Op())
			return layout.Dimensions{Size: image.Pt(w, h)}
		},
	)
}

func (pp *PropertiesPanel) layoutField(gtx layout.Context, th *material.Theme, label string, ed *widget.Editor, onChange func(string)) layout.Dimensions {
	// Apply changes if editor content changed.
	for {
		evt, ok := ed.Update(gtx)
		if !ok {
			break
		}
		if _, isChange := evt.(widget.ChangeEvent); isChange {
			onChange(ed.Text())
		}
	}

	return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, unit.Sp(10), label)
					lbl.Color = theme.ColorTabText
					return lbl.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					// Editor with background.
					return layout.Inset{Top: unit.Dp(1)}.Layout(gtx,
						func(gtx layout.Context) layout.Dimensions {
							edBg := color.NRGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xff}
							return layoutEditorWithBg(gtx, th, ed, edBg)
						},
					)
				}),
			)
		},
	)
}

func (pp *PropertiesPanel) layoutReadonly(gtx layout.Context, th *material.Theme, label, value string) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, unit.Sp(10), label)
					lbl.Color = theme.ColorTabText
					return lbl.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, unit.Sp(12), value)
					lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
					return lbl.Layout(gtx)
				}),
			)
		},
	)
}

func (pp *PropertiesPanel) layoutClickField(gtx layout.Context, th *material.Theme, label, value string, click *widget.Clickable) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, unit.Sp(10), label)
					lbl.Color = theme.ColorTabText
					return lbl.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Clickable(gtx, click, func(gtx layout.Context) layout.Dimensions {
						bg := color.NRGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xff}
						return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx,
							func(gtx layout.Context) layout.Dimensions {
								paint.FillShape(gtx.Ops, bg,
									clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(20)))}.Op())
								lbl := material.Label(th, unit.Sp(12), value)
								lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
								return lbl.Layout(gtx)
							},
						)
					})
				}),
			)
		},
	)
}

func (pp *PropertiesPanel) layoutIntensity(gtx layout.Context, th *material.Theme, intensity model.Intensity) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(th, unit.Sp(10), i18n.T("props.intensity"))
					lbl.Color = theme.ColorTabText
					return lbl.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(pp.intensityDot(th, 1, intensity)),
						layout.Rigid(pp.intensityDot(th, 2, intensity)),
						layout.Rigid(pp.intensityDot(th, 3, intensity)),
					)
				}),
			)
		},
	)
}

func (pp *PropertiesPanel) intensityDot(th *material.Theme, level int, current model.Intensity) func(layout.Context) layout.Dimensions {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Right: unit.Dp(4)}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				return material.Clickable(gtx, &pp.intensityClicks[level-1], func(gtx layout.Context) layout.Dimensions {
					sz := gtx.Dp(unit.Dp(14))
					col := color.NRGBA{R: 0x50, G: 0x50, B: 0x50, A: 0xff}
					if int(current) >= level {
						switch level {
						case 1:
							col = theme.ColorCoach
						case 2:
							col = theme.ColorAttack
						case 3:
							col = theme.ColorMaxInt
						}
					}
					_ = th // referenced for consistency
					paint.FillShape(gtx.Ops, col,
						clip.Ellipse{Max: image.Pt(sz, sz)}.Op(gtx.Ops))
					return layout.Dimensions{Size: image.Pt(sz, sz)}
				})
			},
		)
	}
}

// layoutEditorWithBg renders a text editor with a background color.
func layoutEditorWithBg(gtx layout.Context, th *material.Theme, ed *widget.Editor, bg color.NRGBA) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			paint.FillShape(gtx.Ops, bg,
				clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(20)))}.Op())
			e := material.Editor(th, ed, "")
			e.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
			e.TextSize = unit.Sp(12)
			return e.Layout(gtx)
		},
	)
}

// SyncFromExercise resets the metadata sync flag so editors are refreshed.
func (pp *PropertiesPanel) SyncFromExercise() {
	pp.metaSynced = false
	pp.syncedPlayerIdx = -1
	pp.syncedSeqIdx = -1
}

// Helper functions.

func nextPlayerRole(current model.PlayerRole) model.PlayerRole {
	roles := []model.PlayerRole{
		model.RoleAttacker, model.RoleDefender, model.RoleCoach,
		model.RolePointGuard, model.RoleShootingGuard, model.RoleSmallForward,
		model.RolePowerForward, model.RoleCenter,
	}
	for i, r := range roles {
		if r == current {
			return roles[(i+1)%len(roles)]
		}
	}
	return model.RoleAttacker
}

func nextCategory(current model.Category) model.Category {
	cats := []model.Category{
		model.CategoryWarmup, model.CategoryOffense, model.CategoryDefense,
		model.CategoryTransition, model.CategoryScrimmage, model.CategoryCooldown,
	}
	for i, c := range cats {
		if c == current {
			return cats[(i+1)%len(cats)]
		}
	}
	return model.CategoryWarmup
}

func refString(ref model.ActionRef) string {
	if ref.IsPlayer {
		return ref.PlayerID
	}
	return fmt.Sprintf("(%.2f, %.2f)", ref.Position.X(), ref.Position.Y())
}
