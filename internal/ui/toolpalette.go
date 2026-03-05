package ui

import (
	"image/color"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/ui/editor"
	"github.com/darkweaver87/courtdraw/internal/ui/icon"
)

// toolEntry pairs a TipButton with its i18n key for language refresh.
type toolEntry struct {
	btn *TipButton
	key string
}

// headerEntry pairs a header text with its i18n key.
type headerEntry struct {
	text *canvas.Text
	key  string
}

// Active tool highlight color.
var toolActiveColor = color.NRGBA{R: 0x29, G: 0x6d, B: 0xd4, A: 0xcc}

// ToolPalette is the left sidebar with grouped tool buttons.
type ToolPalette struct {
	box     *fyne.Container
	state   *editor.EditorState
	tools   []toolEntry
	headers []headerEntry
	allBtns []*TipButton // all tool buttons for highlight management

	OnToolChanged func()
}

// button grid cell size.
var toolGridCell fyne.Size

func init() {
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		toolGridCell = fyne.NewSize(54, 54)
	} else {
		toolGridCell = fyne.NewSize(40, 40)
	}
}

// NewToolPalette creates and initializes a tool palette.
func NewToolPalette(state *editor.EditorState) *ToolPalette {
	tp := &ToolPalette{
		state: state,
	}

	vbox := container.NewVBox()

	// Select tool.
	selectGrid := container.NewGridWrap(toolGridCell,
		tp.makeTool("tool.select", icon.ToolSelect, func() {
			state.SetTool(editor.ToolSelect)
			tp.updateActive()
		}),
	)
	vbox.Add(selectGrid)

	// --- Players ---
	vbox.Add(tp.makeHeader("tool.header.players"))

	playerRoles := []model.PlayerRole{
		model.RoleAttacker, model.RoleDefender, model.RoleCoach,
		model.RolePointGuard, model.RoleShootingGuard, model.RoleSmallForward,
		model.RolePowerForward, model.RoleCenter,
	}
	playerKeys := []string{
		"tool.player.attacker", "tool.player.defender", "tool.player.coach",
		"tool.player.pg", "tool.player.sg", "tool.player.sf",
		"tool.player.pf", "tool.player.center",
	}
	playerIcons := []fyne.Resource{
		icon.PlayerAttacker, icon.PlayerDefender, icon.PlayerCoach,
		icon.PlayerPG, icon.PlayerSG, icon.PlayerSF,
		icon.PlayerPF, icon.PlayerCenter,
	}
	playerGrid := container.NewGridWrap(toolGridCell)
	for i, role := range playerRoles {
		r := role
		playerGrid.Add(tp.makeTool(playerKeys[i], playerIcons[i], func() {
			state.SetPlayerTool(r)
			tp.updateActive()
		}))
	}
	// Queue tool in the same grid.
	playerGrid.Add(tp.makeTool("tool.player.queue", icon.PlayerQueue, func() {
		state.SetQueueTool()
		tp.updateActive()
	}))
	vbox.Add(playerGrid)

	// --- Actions ---
	vbox.Add(tp.makeHeader("tool.header.actions"))

	actionTypes := []model.ActionType{
		model.ActionPass, model.ActionDribble, model.ActionSprint,
		model.ActionShotLayup, model.ActionScreen, model.ActionCut,
		model.ActionCloseOut, model.ActionContest, model.ActionReverse,
	}
	actionKeys := []string{
		"tool.action.pass", "tool.action.dribble", "tool.action.sprint",
		"tool.action.shot", "tool.action.screen", "tool.action.cut",
		"tool.action.close_out", "tool.action.contest", "tool.action.reverse",
	}
	actionIcons := []fyne.Resource{
		icon.ActionPass, icon.ActionDribble, icon.ActionSprint,
		icon.ActionShot, icon.ActionScreen, icon.ActionCut,
		icon.ActionCloseOut, icon.ActionContest, icon.ActionReverse,
	}
	actionGrid := container.NewGridWrap(toolGridCell)
	for i, at := range actionTypes {
		actionType := at
		actionGrid.Add(tp.makeTool(actionKeys[i], actionIcons[i], func() {
			state.SetActionTool(actionType)
			tp.updateActive()
		}))
	}
	vbox.Add(actionGrid)

	// --- Accessories ---
	vbox.Add(tp.makeHeader("tool.header.accessories"))

	accTypes := []model.AccessoryType{
		model.AccessoryCone, model.AccessoryAgilityLadder, model.AccessoryChair,
	}
	accKeys := []string{"tool.accessory.cone", "tool.accessory.ladder", "tool.accessory.chair"}
	accIcons := []fyne.Resource{icon.AccCone, icon.AccLadder, icon.AccChair}
	accGrid := container.NewGridWrap(toolGridCell)
	for i, at := range accTypes {
		accType := at
		accGrid.Add(tp.makeTool(accKeys[i], accIcons[i], func() {
			state.SetAccessoryTool(accType)
			tp.updateActive()
		}))
	}
	vbox.Add(accGrid)

	// --- Delete ---
	vbox.Add(widget.NewSeparator())
	deleteGrid := container.NewGridWrap(toolGridCell,
		tp.makeTool("tool.delete", icon.Delete(), func() {
			if state.SelectedElement != nil {
				state.DeleteRequested = true
			} else {
				state.SetTool(editor.ToolDelete)
			}
			tp.updateActive()
		}),
	)
	vbox.Add(deleteGrid)

	bg := canvas.NewRectangle(color.NRGBA{R: 0x30, G: 0x30, B: 0x30, A: 0xff})
	scroll := container.NewVScroll(vbox)
	tp.box = container.NewStack(bg, scroll)
	return tp
}

// Widget returns the tool palette widget.
func (tp *ToolPalette) Widget() fyne.CanvasObject {
	return tp.box
}

// RefreshLanguage updates all tooltips and headers for the current language.
func (tp *ToolPalette) RefreshLanguage() {
	for _, t := range tp.tools {
		t.btn.SetTooltip(i18n.T(t.key))
	}
	for _, h := range tp.headers {
		h.text.Text = i18n.T(h.key)
		h.text.Refresh()
	}
}

func (tp *ToolPalette) makeTool(key string, res fyne.Resource, onTap func()) *TipButton {
	btn := NewTipButton(res, i18n.T(key), onTap)
	tp.tools = append(tp.tools, toolEntry{btn: btn, key: key})
	tp.allBtns = append(tp.allBtns, btn)
	return btn
}

func (tp *ToolPalette) makeHeader(key string) fyne.CanvasObject {
	lbl := canvas.NewText(i18n.T(key), color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff})
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		lbl.TextSize = 16
	} else {
		lbl.TextSize = 11
	}
	tp.headers = append(tp.headers, headerEntry{text: lbl, key: key})
	return container.NewPadded(lbl)
}

func (tp *ToolPalette) updateActive() {
	// Determine which button index is active based on editor state.
	activeIdx := -1
	state := tp.state
	switch state.ActiveTool {
	case editor.ToolSelect:
		activeIdx = 0 // select button is first
	case editor.ToolPlayer:
		// Player buttons start at index 1.
		roles := []model.PlayerRole{
			model.RoleAttacker, model.RoleDefender, model.RoleCoach,
			model.RolePointGuard, model.RoleShootingGuard, model.RoleSmallForward,
			model.RolePowerForward, model.RoleCenter,
		}
		for i, r := range roles {
			if state.ToolRole == r && !state.ToolQueue {
				activeIdx = 1 + i
				break
			}
		}
		if state.ToolQueue {
			activeIdx = 9 // queue button
		}
	case editor.ToolAction:
		actions := []model.ActionType{
			model.ActionPass, model.ActionDribble, model.ActionSprint,
			model.ActionShotLayup, model.ActionScreen, model.ActionCut,
			model.ActionCloseOut, model.ActionContest, model.ActionReverse,
		}
		for i, a := range actions {
			if state.ToolActionType == a {
				activeIdx = 10 + i
				break
			}
		}
	case editor.ToolAccessory:
		accTypes := []model.AccessoryType{
			model.AccessoryCone, model.AccessoryAgilityLadder, model.AccessoryChair,
		}
		for i, a := range accTypes {
			if state.ToolAccessoryType == a {
				activeIdx = 19 + i
				break
			}
		}
	case editor.ToolDelete:
		activeIdx = 22
	}

	// Update all button highlights.
	for i, btn := range tp.allBtns {
		if i == activeIdx {
			btn.OverrideColor = toolActiveColor
		} else {
			btn.OverrideColor = nil
		}
		btn.Refresh()
	}

	if tp.OnToolChanged != nil {
		tp.OnToolChanged()
	}
}
