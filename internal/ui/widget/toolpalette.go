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

// toolEntry represents a single tool button in the palette.
type toolEntry struct {
	clickable widget.Clickable
	label     string
	color     color.NRGBA      // label/icon tint color
	icon      *widget.Icon     // optional Material Design icon (nil = use pngIcon or colored dot)
	pngIcon   *icon.PngIcon    // optional PNG icon for basketball tools
}

// ToolPalette is the left sidebar with grouped tool buttons.
type ToolPalette struct {
	selectTool toolEntry

	// Player tools.
	playerTools [9]toolEntry
	playerRoles [9]model.PlayerRole

	// Action tools.
	actionTools [9]toolEntry
	actionTypes [9]model.ActionType

	// Accessory tools.
	accessoryTools [3]toolEntry
	accessoryTypes [3]model.AccessoryType

	// Delete tool.
	deleteTool toolEntry

	scrollList widget.List
}

// NewToolPalette creates and initializes a tool palette.
func NewToolPalette() *ToolPalette {
	tp := &ToolPalette{}

	tp.selectTool = toolEntry{label: "tool.select", color: color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, icon: icon.Select}

	// Players.
	tp.playerRoles = [9]model.PlayerRole{
		model.RoleAttacker, model.RoleDefender, model.RoleCoach,
		model.RolePointGuard, model.RoleShootingGuard, model.RoleSmallForward,
		model.RolePowerForward, model.RoleCenter,
	}
	playerLabels := [9]string{"tool.player.attacker", "tool.player.defender", "tool.player.coach", "tool.player.pg", "tool.player.sg", "tool.player.sf", "tool.player.pf", "tool.player.center", "tool.player.queue"}
	playerColors := [9]color.NRGBA{
		model.ColorAttack, model.ColorDefense, model.ColorCoach,
		model.ColorAttack, model.ColorAttack, model.ColorAttack,
		model.ColorAttack, model.ColorAttack, model.ColorNeutral,
	}
	playerPngs := [9]*icon.PngIcon{
		icon.PlayerAttacker, icon.PlayerDefender, icon.PlayerCoach,
		icon.PlayerPG, icon.PlayerSG, icon.PlayerSF,
		icon.PlayerPF, icon.PlayerCenter, icon.PlayerQueue,
	}
	for i := range tp.playerTools {
		tp.playerTools[i] = toolEntry{label: playerLabels[i], color: playerColors[i], pngIcon: playerPngs[i]}
	}

	// Actions.
	tp.actionTypes = [9]model.ActionType{
		model.ActionPass, model.ActionDribble, model.ActionSprint,
		model.ActionShotLayup, model.ActionScreen, model.ActionCut,
		model.ActionCloseOut, model.ActionContest, model.ActionReverse,
	}
	actionLabels := [9]string{"tool.action.pass", "tool.action.dribble", "tool.action.sprint", "tool.action.shot", "tool.action.screen", "tool.action.cut", "tool.action.close_out", "tool.action.contest", "tool.action.reverse"}
	actionColors := [9]color.NRGBA{
		colorPass, colorDribble, colorSprint,
		colorSprint, colorScreen, colorCut,
		colorCloseOut, colorCloseOut, colorSprint,
	}
	actionPngs := [9]*icon.PngIcon{
		icon.ActionPass, icon.ActionDribble, icon.ActionSprint,
		icon.ActionShot, icon.ActionScreen, icon.ActionCut,
		icon.ActionCloseOut, icon.ActionContest, icon.ActionReverse,
	}
	for i := range tp.actionTools {
		tp.actionTools[i] = toolEntry{label: actionLabels[i], color: actionColors[i], pngIcon: actionPngs[i]}
	}

	// Accessories.
	tp.accessoryTypes = [3]model.AccessoryType{
		model.AccessoryCone, model.AccessoryAgilityLadder, model.AccessoryChair,
	}
	accLabels := [3]string{"tool.accessory.cone", "tool.accessory.ladder", "tool.accessory.chair"}
	accColors := [3]color.NRGBA{colorCone, colorLadder, colorChair}
	accPngs := [3]*icon.PngIcon{icon.AccCone, icon.AccLadder, icon.AccChair}
	for i := range tp.accessoryTools {
		tp.accessoryTools[i] = toolEntry{label: accLabels[i], color: accColors[i], pngIcon: accPngs[i]}
	}

	tp.deleteTool = toolEntry{label: "tool.delete", color: color.NRGBA{R: 0xff, G: 0x40, B: 0x40, A: 0xff}, icon: icon.Delete}

	tp.scrollList.Axis = layout.Vertical

	return tp
}

// Layout renders the tool palette and updates editor state on tool clicks.
func (tp *ToolPalette) Layout(gtx layout.Context, th *material.Theme, state *editor.EditorState) layout.Dimensions {
	// Handle clicks.
	tp.handleClicks(gtx, state)

	// Background.
	panelBg := color.NRGBA{R: 0x30, G: 0x30, B: 0x30, A: 0xff}
	size := image.Pt(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)
	paint.FillShape(gtx.Ops, panelBg, clip.Rect{Max: size}.Op())

	// Total number of items: 1 (select) + 1 (header) + 9 (players) + 1 (header) + 9 (actions) + 1 (header) + 3 (accessories) + 1 (header) + 1 (delete)
	// = 27 items
	type listItem struct {
		kind  string // "header", "select", "player", "action", "accessory", "delete"
		index int
	}

	items := make([]listItem, 0, 30)
	items = append(items, listItem{kind: "select"})
	items = append(items, listItem{kind: "header", index: 0}) // "Players"
	for i := 0; i < 9; i++ {
		items = append(items, listItem{kind: "player", index: i})
	}
	items = append(items, listItem{kind: "header", index: 1}) // "Actions"
	for i := 0; i < 9; i++ {
		items = append(items, listItem{kind: "action", index: i})
	}
	items = append(items, listItem{kind: "header", index: 2}) // "Accessories"
	for i := 0; i < 3; i++ {
		items = append(items, listItem{kind: "accessory", index: i})
	}
	items = append(items, listItem{kind: "header", index: 3}) // "Delete"
	items = append(items, listItem{kind: "delete"})

	headers := [4]string{"tool.header.players", "tool.header.actions", "tool.header.accessories", ""}

	return material.List(th, &tp.scrollList).Layout(gtx, len(items), func(gtx layout.Context, idx int) layout.Dimensions {
		item := items[idx]
		switch item.kind {
		case "header":
			if headers[item.index] == "" {
				// Separator line before delete section.
				return tp.layoutSeparator(gtx)
			}
			return tp.layoutHeader(gtx, th, i18n.T(headers[item.index]))
		case "select":
			active := state.ActiveTool == editor.ToolSelect
			return tp.layoutToolButton(gtx, th, &tp.selectTool, active)
		case "player":
			var active bool
			if item.index == 8 {
				// Queue tool: active when in queue mode.
				active = state.ActiveTool == editor.ToolPlayer && state.ToolQueue
			} else {
				active = state.ActiveTool == editor.ToolPlayer && !state.ToolQueue && state.ToolRole == tp.playerRoles[item.index]
			}
			return tp.layoutToolButton(gtx, th, &tp.playerTools[item.index], active)
		case "action":
			active := state.ActiveTool == editor.ToolAction && state.ToolActionType == tp.actionTypes[item.index]
			return tp.layoutToolButton(gtx, th, &tp.actionTools[item.index], active)
		case "accessory":
			active := state.ActiveTool == editor.ToolAccessory && state.ToolAccessoryType == tp.accessoryTypes[item.index]
			return tp.layoutToolButton(gtx, th, &tp.accessoryTools[item.index], active)
		case "delete":
			active := state.ActiveTool == editor.ToolDelete
			return tp.layoutToolButton(gtx, th, &tp.deleteTool, active)
		}
		return layout.Dimensions{}
	})
}

func (tp *ToolPalette) handleClicks(gtx layout.Context, state *editor.EditorState) {
	if tp.selectTool.clickable.Clicked(gtx) {
		state.SetTool(editor.ToolSelect)
	}
	for i := range tp.playerTools {
		if tp.playerTools[i].clickable.Clicked(gtx) {
			if i == 8 {
				state.SetQueueTool()
			} else {
				state.SetPlayerTool(tp.playerRoles[i])
			}
		}
	}
	for i := range tp.actionTools {
		if tp.actionTools[i].clickable.Clicked(gtx) {
			state.SetActionTool(tp.actionTypes[i])
		}
	}
	for i := range tp.accessoryTools {
		if tp.accessoryTools[i].clickable.Clicked(gtx) {
			state.SetAccessoryTool(tp.accessoryTypes[i])
		}
	}
	if tp.deleteTool.clickable.Clicked(gtx) {
		if state.SelectedElement != nil {
			state.DeleteRequested = true
		} else {
			state.SetTool(editor.ToolDelete)
		}
	}
}

func (tp *ToolPalette) layoutHeader(gtx layout.Context, th *material.Theme, title string) layout.Dimensions {
	return layout.Inset{Top: unit.Dp(8), Left: unit.Dp(8), Bottom: unit.Dp(2)}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(th, unit.Sp(11), title)
			lbl.Color = theme.ColorTabText
			return lbl.Layout(gtx)
		},
	)
}

func (tp *ToolPalette) layoutSeparator(gtx layout.Context) layout.Dimensions {
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

func (tp *ToolPalette) layoutToolButton(gtx layout.Context, th *material.Theme, entry *toolEntry, active bool) layout.Dimensions {
	return material.Clickable(gtx, &entry.clickable, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(4), Right: unit.Dp(4)}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				// Background for active tool.
				if active {
					size := image.Pt(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(24)))
					bg := color.NRGBA{R: 0x50, G: 0x50, B: 0x80, A: 0xff}
					paint.FillShape(gtx.Ops, bg, clip.Rect{Max: size}.Op())
				}

				return layout.Inset{Top: unit.Dp(3), Bottom: unit.Dp(3), Left: unit.Dp(6)}.Layout(gtx,
					func(gtx layout.Context) layout.Dimensions {
						// Icon or colored dot + label.
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if entry.icon != nil {
									return icon.LayoutIcon(gtx, entry.icon, entry.color)
								}
								if entry.pngIcon != nil && entry.pngIcon.Valid() {
									return entry.pngIcon.Layout(gtx, unit.Dp(16))
								}
								sz := gtx.Dp(unit.Dp(8))
								paint.FillShape(gtx.Ops, entry.color,
									clip.Ellipse{Max: image.Pt(sz, sz)}.Op(gtx.Ops))
								return layout.Dimensions{Size: image.Pt(sz, sz)}
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Inset{Left: unit.Dp(6)}.Layout(gtx,
									func(gtx layout.Context) layout.Dimensions {
										lbl := material.Label(th, unit.Sp(12), i18n.T(entry.label))
										lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
										return lbl.Layout(gtx)
									},
								)
							}),
						)
					},
				)
			},
		)
	})
}
