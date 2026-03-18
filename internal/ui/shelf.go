package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/ui/editor"
	"github.com/darkweaver87/courtdraw/internal/ui/icon"
)

// ---------------------------------------------------------------------------
// Editor mode
// ---------------------------------------------------------------------------

// EditorMode represents the active mode of the mobile editor.
type EditorMode int

const (
	ModeEdition   EditorMode = iota // Draw / edit elements
	ModeAnimation                   // Playback + action timeline
	ModeNotes                       // Instructions / notes view
	ModeSession                     // Session composer
	ModeMyFiles                     // File manager
	ModeTraining                    // Training mode (run session on the court)
)

// ---------------------------------------------------------------------------
// Shelf categories (Draw mode)
// ---------------------------------------------------------------------------

type shelfCategory int

const (
	shelfTools       shelfCategory = iota // select + delete
	shelfPlayers                          // player roles + queue
	shelfActions                          // action types
	shelfAccessories                      // cones, ladder, chair
)

const numShelfCategories = 4

// Grid cell size for shelf tool buttons.
var shelfCellSize fyne.Size

func init() {
	if isMobile {
		shelfCellSize = fyne.NewSize(64, 64)
	} else {
		shelfCellSize = fyne.NewSize(36, 36)
	}
}

// ---------------------------------------------------------------------------
// EditorShelf — bottom shelf + tab bar for Draw mode
// ---------------------------------------------------------------------------

// EditorShelf provides the bottom tool shelf + category tab bar.
type EditorShelf struct {
	state   *editor.EditorState
	palette *ToolPalette
	active  shelfCategory
	allBtns []*TipButton

	// Shelf content per category.
	toolsContent fyne.CanvasObject
	playerContent fyne.CanvasObject
	actionContent fyne.CanvasObject
	accContent    fyne.CanvasObject

	// Layout elements.
	shelfStack  *fyne.Container // swaps shelf content
	shelfOuter  *fyne.Container // collapsible area (shelf + chevron)
	collapsed   bool
	chevronBtn  *TipButton
	tabLabels   [numShelfCategories]*canvas.Text
	tabIndicators [numShelfCategories]*canvas.Rectangle

	OnToolChanged func()
}

// NewEditorShelf creates the Draw mode shelf.
func NewEditorShelf(state *editor.EditorState, palette *ToolPalette) *EditorShelf {
	ms := &EditorShelf{state: state, palette: palette}
	ms.build()
	return ms
}

func (ms *EditorShelf) build() {
	// --- Tools: select + delete ---
	selectBtn := ms.addBtn(icon.ToolSelect, "tool.select", func() {
		ms.state.SetTool(editor.ToolSelect)
		ms.syncHighlights()
	})
	deleteBtn := ms.addBtn(icon.Delete(), "tool.delete", func() {
		if ms.state.SelectedElement != nil {
			ms.state.DeleteRequested = true
		} else {
			ms.state.SetTool(editor.ToolDelete)
		}
		ms.syncHighlights()
	})
	deleteBtn.SetImportance(widget.DangerImportance)
	ms.toolsContent = container.NewGridWrap(shelfCellSize, selectBtn, deleteBtn)

	// --- Players: 8 roles + queue = 9 buttons, 5 columns ---
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
	playerGrid := container.NewGridWrap(shelfCellSize)
	for i, role := range playerRoles {
		r := role
		btn := ms.addBtn(playerIcons[i], playerKeys[i], func() {
			ms.state.SetPlayerTool(r)
			ms.syncHighlights()
			ms.collapse()
		})
		playerGrid.Add(btn)
	}
	queueBtn := ms.addBtn(icon.PlayerQueue, "tool.player.queue", func() {
		ms.state.SetQueueTool()
		ms.syncHighlights()
		ms.collapse()
	})
	playerGrid.Add(queueBtn)
	ms.playerContent = playerGrid

	// --- Actions: 9 buttons, 3 columns ---
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
	actionGrid := container.NewGridWrap(shelfCellSize)
	for i, at := range actionTypes {
		actionType := at
		btn := ms.addBtn(actionIcons[i], actionKeys[i], func() {
			ms.state.SetActionTool(actionType)
			ms.syncHighlights()
			ms.collapse()
		})
		actionGrid.Add(btn)
	}
	ms.actionContent = actionGrid

	// --- Accessories: 3 buttons, 3 columns ---
	accTypes := []model.AccessoryType{
		model.AccessoryCone, model.AccessoryAgilityLadder, model.AccessoryChair,
	}
	accKeys := []string{"tool.accessory.cone", "tool.accessory.ladder", "tool.accessory.chair"}
	accIcons := []fyne.Resource{icon.AccCone, icon.AccLadder, icon.AccChair}
	accGrid := container.NewGridWrap(shelfCellSize)
	for i, at := range accTypes {
		accType := at
		btn := ms.addBtn(accIcons[i], accKeys[i], func() {
			ms.state.SetAccessoryTool(accType)
			ms.syncHighlights()
			ms.collapse()
		})
		accGrid.Add(btn)
	}
	ms.accContent = accGrid

	ms.shelfStack = container.NewStack(ms.toolsContent)
}

// addBtn creates a TipButton for the shelf and registers it.
func (ms *EditorShelf) addBtn(res fyne.Resource, key string, onTap func()) *TipButton {
	btn := NewTipButton(res, i18n.T(key), onTap)
	if !isMobile {
		// On desktop, cap size to keep shelf compact.
		sz := shelfCellSize
		btn.MaxSize = &sz
	}
	ms.allBtns = append(ms.allBtns, btn)
	return btn
}

func (ms *EditorShelf) syncHighlights() {
	for _, btn := range ms.allBtns {
		btn.OverrideColor = nil
		btn.Refresh()
	}
	// Index map: 0=select, 1=delete, 2..9=players, 10=queue, 11..19=actions, 20..22=accessories
	var idx int = -1
	switch ms.state.ActiveTool {
	case editor.ToolSelect:
		idx = 0
	case editor.ToolDelete:
		idx = 1
	case editor.ToolPlayer:
		roles := []model.PlayerRole{
			model.RoleAttacker, model.RoleDefender, model.RoleCoach,
			model.RolePointGuard, model.RoleShootingGuard, model.RoleSmallForward,
			model.RolePowerForward, model.RoleCenter,
		}
		for i, r := range roles {
			if ms.state.ToolRole == r && !ms.state.ToolQueue {
				idx = 2 + i
				break
			}
		}
		if ms.state.ToolQueue {
			idx = 10
		}
	case editor.ToolAction:
		actions := []model.ActionType{
			model.ActionPass, model.ActionDribble, model.ActionSprint,
			model.ActionShotLayup, model.ActionScreen, model.ActionCut,
			model.ActionCloseOut, model.ActionContest, model.ActionReverse,
		}
		for i, a := range actions {
			if ms.state.ToolActionType == a {
				idx = 11 + i
				break
			}
		}
	case editor.ToolAccessory:
		accTypes := []model.AccessoryType{
			model.AccessoryCone, model.AccessoryAgilityLadder, model.AccessoryChair,
		}
		for i, a := range accTypes {
			if ms.state.ToolAccessoryType == a {
				idx = 20 + i
				break
			}
		}
	}
	if idx >= 0 && idx < len(ms.allBtns) {
		ms.allBtns[idx].OverrideColor = toolActiveColor
		ms.allBtns[idx].Refresh()
	}
	if ms.palette != nil {
		ms.palette.ForceUpdateActive()
	}
	if ms.OnToolChanged != nil {
		ms.OnToolChanged()
	}
}

func (ms *EditorShelf) collapse() {
	if ms.collapsed {
		return
	}
	ms.collapsed = true
	ms.shelfOuter.Hide()
	ms.chevronBtn.Icon = icon.ChevronUp
	ms.chevronBtn.Refresh()
}

func (ms *EditorShelf) expand() {
	ms.collapsed = false
	ms.shelfOuter.Show()
	ms.chevronBtn.Icon = icon.ChevronDown
	ms.chevronBtn.Refresh()
}

func (ms *EditorShelf) selectCategory(cat shelfCategory) {
	if cat == ms.active && !ms.collapsed {
		ms.collapse()
		return
	}
	ms.active = cat
	ms.expand()
	if cat == shelfTools {
		ms.state.SetTool(editor.ToolSelect)
		ms.syncHighlights()
	}
	var content fyne.CanvasObject
	switch cat {
	case shelfTools:
		content = ms.toolsContent
	case shelfPlayers:
		content = ms.playerContent
	case shelfActions:
		content = ms.actionContent
	case shelfAccessories:
		content = ms.accContent
	}
	ms.shelfStack.Objects = []fyne.CanvasObject{content}
	ms.shelfStack.Refresh()

	for i := range numShelfCategories {
		if shelfCategory(i) == cat {
			ms.tabIndicators[i].FillColor = tabActiveColor
			ms.tabLabels[i].Color = tabLabelActiveColor
		} else {
			ms.tabIndicators[i].FillColor = color.Transparent
			ms.tabLabels[i].Color = tabInactiveColor
		}
		ms.tabIndicators[i].Refresh()
		ms.tabLabels[i].Refresh()
	}
}

// Widget returns the complete shelf + tab bar layout.
func (ms *EditorShelf) Widget() fyne.CanvasObject {
	// Chevron (top-right of shelf content).
	ms.chevronBtn = NewTipButton(icon.ChevronDown, "", func() {
		if ms.collapsed {
			ms.expand()
		} else {
			ms.collapse()
		}
	})

	// Shelf content area with chevron top-right.
	shelfBg := canvas.NewRectangle(color.NRGBA{R: 0x28, G: 0x28, B: 0x28, A: 0xff})
	chevronSz := fyne.NewSize(24, 24)
	if isMobile {
		chevronSz = fyne.NewSize(40, 40)
	}
	chevronWrap := container.NewGridWrap(chevronSz, ms.chevronBtn)
	chevronCol := container.NewVBox(chevronWrap)
	shelfInner := container.NewBorder(nil, nil, nil, chevronCol, ms.shelfStack)
	ms.shelfOuter = container.NewStack(shelfBg, container.NewPadded(shelfInner))

	// Tab bar.
	tabs := []struct {
		ico   fyne.Resource
		label string
		cat   shelfCategory
	}{
		{icon.ToolSelect, i18n.T("mobile.shelf.tools"), shelfTools},
		{icon.PlayerAttacker, i18n.T("mobile.shelf.players"), shelfPlayers},
		{icon.ActionPass, i18n.T("mobile.shelf.actions"), shelfActions},
		{icon.AccCone, i18n.T("mobile.shelf.accessories"), shelfAccessories},
	}
	tabItems := make([]fyne.CanvasObject, len(tabs))
	for i, t := range tabs {
		idx := i
		cat := t.cat
		indicator := canvas.NewRectangle(color.Transparent)
		indicator.SetMinSize(fyne.NewSize(0, 3))
		if idx == 0 {
			indicator.FillColor = tabActiveColor
		}
		ms.tabIndicators[idx] = indicator

		ico := canvas.NewImageFromResource(t.ico)
		ico.FillMode = canvas.ImageFillContain
		ico.SetMinSize(fyne.NewSize(tabIconSize, tabIconSize))

		lbl := canvas.NewText(t.label, tabInactiveColor)
		lbl.TextSize = tabFontSize
		lbl.Alignment = fyne.TextAlignCenter
		if idx == 0 {
			lbl.Color = tabLabelActiveColor
		}
		ms.tabLabels[idx] = lbl

		col := container.NewVBox(indicator, container.NewCenter(ico), lbl)
		tabItems[idx] = newTabTappable(col, func() { ms.selectCategory(cat) })
	}
	tabBg := canvas.NewRectangle(tabBgColor)
	tabBg.SetMinSize(fyne.NewSize(0, tabBarHeight))
	tabGrid := container.NewGridWithColumns(len(tabItems), tabItems...)
	tabBar := container.NewStack(tabBg, container.NewPadded(tabGrid))

	return container.NewVBox(ms.shelfOuter, tabBar)
}

// RefreshLanguage updates tab labels.
func (ms *EditorShelf) RefreshLanguage() {
	keys := []string{
		"mobile.shelf.tools", "mobile.shelf.players",
		"mobile.shelf.actions", "mobile.shelf.accessories",
	}
	for i, k := range keys {
		if ms.tabLabels[i] != nil {
			ms.tabLabels[i].Text = i18n.T(k)
			ms.tabLabels[i].Refresh()
		}
	}
}
