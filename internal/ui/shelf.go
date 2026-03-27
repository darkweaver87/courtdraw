package ui

import (
	"fmt"
	"image/color"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/anim"
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
	shelfPlayers     shelfCategory = iota // player roles + queue
	shelfActions                          // action types
	shelfAccessories                      // cones, ladder, chair
)

const numShelfCategories = 3

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

// ZoomController is implemented by the court widget for zoom operations.
type ZoomController interface {
	ZoomIn()
	ZoomOut()
	ResetZoom()
}

// EditorShelf provides the bottom tool shelf + category tab bar.
type EditorShelf struct {
	state    *editor.EditorState
	palette  *ToolPalette
	exercise *model.Exercise
	seqIdx   int
	zoomer   ZoomController
	active   shelfCategory
	allBtns  []*TipButton

	// Shelf content per category.
	playerContent fyne.CanvasObject
	actionContent fyne.CanvasObject
	accContent    fyne.CanvasObject

	// Compact element properties (shown in tools shelf when element selected).
	propsContent    *fyne.Container
	propsTitle      *canvas.Text
	propsLabelE     *widget.Entry
	propsRoleSel    *widget.Select
	propsBallChk    *widget.Check
	propsBallBtn    *TipButton
	propsCalloutSel *widget.Select
	propsPosXEntry  *widget.Entry
	propsPosYEntry  *widget.Entry
	propsDpad       *fyne.Container
	propsRotKnob    *RotKnob
	propsDeleteBtn  *TipButton
	propsUpdating   bool             // prevents recursive OnChanged → refreshEditor loop
	propsSyncedSel  *editor.Selection // tracks which element is currently displayed

	// Button-to-i18n key mapping for tooltip refresh.
	btnKeys       []string
	ballToggleIdx int // index of ball toggle in allBtns

	// Layout elements.
	shelfStack    *fyne.Container // swaps shelf content
	shelfOuter    *fyne.Container // collapsible area (shelf + chevron)
	collapsed     bool
	chevronBtn    *TipButton
	tabLabels     [numShelfCategories]*canvas.Text
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
	// --- Players: 8 roles + queue = 9 buttons, 5 columns ---
	playerRoles := []model.PlayerRole{
		model.RoleAttacker, model.RoleDefender, model.RoleCoach,
		model.RolePointGuard, model.RoleShootingGuard, model.RoleSmallForward,
		model.RolePowerForward, model.RoleCenter,
	}
	playerKeys := []string{
		i18n.KeyToolPlayerAttacker, i18n.KeyToolPlayerDefender, i18n.KeyToolPlayerCoach,
		i18n.KeyToolPlayerPg, i18n.KeyToolPlayerSg, i18n.KeyToolPlayerSf,
		i18n.KeyToolPlayerPf, i18n.KeyToolPlayerCenter,
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
			if ms.state.ActiveTool == editor.ToolPlayer && ms.state.ToolRole == r && !ms.state.ToolQueue {
				ms.state.SetTool(editor.ToolSelect)
			} else {
				ms.state.SetPlayerTool(r)
				if r == model.RoleDefender {
					ms.state.ToolWithBall = false
				}
			}
			ms.syncHighlights()
		})
		playerGrid.Add(btn)
	}
	queueBtn := ms.addBtn(icon.PlayerQueue, i18n.KeyToolPlayerQueue, func() {
		ms.state.SetQueueTool()
		ms.syncHighlights()
	})
	playerGrid.Add(queueBtn)
	ballToggle := ms.addBtn(icon.BallIcon, i18n.KeyPropsBall, func() {
		ms.state.ToolWithBall = !ms.state.ToolWithBall
		ms.syncHighlights()
	})
	ms.ballToggleIdx = len(ms.allBtns) - 1
	playerGrid.Add(ballToggle)
	ms.ballToggleIdx = len(ms.allBtns) - 1
	ms.playerContent = playerGrid

	// --- Actions: 6 buttons (standard basketball conventions) ---
	actionTypes := []model.ActionType{
		model.ActionDribble, model.ActionPass, model.ActionCut,
		model.ActionScreen, model.ActionShot, model.ActionHandoff,
	}
	actionKeys := []string{
		i18n.KeyToolActionDribble, i18n.KeyToolActionPass, i18n.KeyToolActionCut,
		i18n.KeyToolActionScreen, i18n.KeyToolActionShot, i18n.KeyToolActionHandoff,
	}
	actionIcons := []fyne.Resource{
		icon.ActionDribble, icon.ActionPass, icon.ActionCut,
		icon.ActionScreen, icon.ActionShot, icon.ActionHandoffRes,
	}
	actionGrid := container.NewGridWrap(shelfCellSize)
	for i, at := range actionTypes {
		actionType := at
		btn := ms.addBtn(actionIcons[i], actionKeys[i], func() {
			// Toggle off if same tool re-clicked.
			if ms.state.ActiveTool == editor.ToolAction && ms.state.ToolActionType == actionType {
				ms.state.SetTool(editor.ToolSelect)
				ms.syncHighlights()
				return
			}
			ms.state.SetActionTool(actionType)
			// If a player is already selected, use it as action source immediately.
			if sel := ms.state.SelectedElement; sel != nil && sel.Kind == editor.SelectPlayer {
				if ms.exercise != nil && sel.SeqIndex < len(ms.exercise.Sequences) {
					seq := &ms.exercise.Sequences[sel.SeqIndex]
					if sel.Index < len(seq.Players) {
						id := seq.Players[sel.Index].ID
						finalCarriers := anim.ComputeFinalBallCarriers(seq)
						hasBall := slices.Contains(finalCarriers, id)
						if model.RequiresBall(actionType) && !hasBall {
							ms.state.SetStatus(i18n.T(i18n.KeyStatusRequiresBall), 1)
						} else {
							ms.state.ActionFrom = &id
						}
					}
				}
			}
			ms.syncHighlights()
		})
		actionGrid.Add(btn)
	}
	ms.actionContent = actionGrid

	// --- Accessories: 3 buttons, 3 columns ---
	accTypes := []model.AccessoryType{
		model.AccessoryCone, model.AccessoryAgilityLadder, model.AccessoryChair,
	}
	accKeys := []string{i18n.KeyToolAccessoryCone, i18n.KeyToolAccessoryLadder, i18n.KeyToolAccessoryChair}
	accIcons := []fyne.Resource{icon.AccCone, icon.AccLadder, icon.AccChair}
	accGrid := container.NewGridWrap(shelfCellSize)
	for i, at := range accTypes {
		accType := at
		btn := ms.addBtn(accIcons[i], accKeys[i], func() {
			ms.state.SetAccessoryTool(accType)
			ms.syncHighlights()
		})
		accGrid.Add(btn)
	}
	ms.accContent = accGrid

	ms.shelfStack = container.NewStack(ms.playerContent)

	// --- Compact element properties (shown when element selected in tools tab) ---
	ms.propsTitle = canvas.NewText("", color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff})
	ms.propsTitle.TextStyle.Bold = true
	if isMobile {
		ms.propsTitle.TextSize = 14
	} else {
		ms.propsTitle.TextSize = 12
	}
	ms.propsLabelE = widget.NewEntry()
	ms.propsLabelE.PlaceHolder = i18n.T(i18n.KeyPropsName)
	ms.propsLabelE.OnChanged = func(_ string) {} // wired in UpdateElementProps
	ms.propsRoleSel = widget.NewSelect(nil, func(_ string) {})
	ms.propsBallChk = widget.NewCheck(i18n.T(i18n.KeyPropsBall), func(_ bool) {})
	ms.propsBallBtn = NewTipButton(icon.BallIcon, i18n.T(i18n.KeyPropsBall), func() {})
	ms.propsCalloutSel = widget.NewSelect(nil, func(_ string) {})
	ms.propsCalloutSel.PlaceHolder = i18n.T(i18n.KeyPropsCallout)
	ms.propsPosXEntry = widget.NewEntry()
	ms.propsPosXEntry.PlaceHolder = "X"
	ms.propsPosYEntry = widget.NewEntry()
	ms.propsPosYEntry.PlaceHolder = "Y"
	// D-pad for position nudging.
	dpadStep := 0.01 // 1% of court per press
	dpadBtnSize := fyne.NewSize(28, 28)
	if isMobile {
		dpadBtnSize = fyne.NewSize(40, 40)
	}
	ms.propsRotKnob = NewRotKnob(func(_ float64) {}) // wired in buildPropsLayout

	dpadUp := NewTipButton(fynetheme.MoveUpIcon(), "", func() { ms.nudgeSelection(0, dpadStep) })
	dpadDown := NewTipButton(fynetheme.MoveDownIcon(), "", func() { ms.nudgeSelection(0, -dpadStep) })
	dpadLeft := NewTipButton(fynetheme.NavigateBackIcon(), "", func() { ms.nudgeSelection(-dpadStep, 0) })
	dpadRight := NewTipButton(fynetheme.NavigateNextIcon(), "", func() { ms.nudgeSelection(dpadStep, 0) })
	empty := canvas.NewRectangle(color.Transparent)
	empty.SetMinSize(dpadBtnSize)
	dpadGrid := container.NewGridWithColumns(3,
		empty, container.NewGridWrap(dpadBtnSize, dpadUp), empty,
		container.NewGridWrap(dpadBtnSize, dpadLeft), container.NewGridWrap(dpadBtnSize, ms.propsRotKnob), container.NewGridWrap(dpadBtnSize, dpadRight),
		empty, container.NewGridWrap(dpadBtnSize, dpadDown), empty,
	)
	ms.propsDpad = container.NewCenter(dpadGrid)
	ms.propsDeleteBtn = NewTipButton(icon.Delete(), i18n.T(i18n.KeyToolDelete), func() {
		ms.state.DeleteRequested = true
		ms.propsSyncedSel = nil
		ms.propsContent.RemoveAll()
		ms.refreshShelfContent()
		if ms.OnToolChanged != nil {
			ms.OnToolChanged()
		}
	})
	ms.propsDeleteBtn.SetImportance(widget.DangerImportance)
	ms.propsContent = container.NewVBox()
}

// addBtn creates a TipButton for the shelf and registers it.
func (ms *EditorShelf) addBtn(res fyne.Resource, key string, onTap func()) *TipButton {
	btn := NewTipButton(res, i18n.T(key), onTap)
	ms.btnKeys = append(ms.btnKeys, key)
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
	// Index map: 0..7=players, 8=queue, 9=ball, 10..15=actions, 16..18=accessories
	idx := -1
	switch ms.state.ActiveTool {
	case editor.ToolPlayer:
		roles := []model.PlayerRole{
			model.RoleAttacker, model.RoleDefender, model.RoleCoach,
			model.RolePointGuard, model.RoleShootingGuard, model.RoleSmallForward,
			model.RolePowerForward, model.RoleCenter,
		}
		for i, r := range roles {
			if ms.state.ToolRole == r && !ms.state.ToolQueue {
				idx = i
				break
			}
		}
		if ms.state.ToolQueue {
			idx = 8
		}
	case editor.ToolAction:
		actions := []model.ActionType{
			model.ActionDribble, model.ActionPass, model.ActionCut,
			model.ActionScreen, model.ActionShot, model.ActionHandoff,
		}
		for i, a := range actions {
			if ms.state.ToolActionType == a {
				idx = 10 + i
				break
			}
		}
	case editor.ToolAccessory:
		accTypes := []model.AccessoryType{
			model.AccessoryCone, model.AccessoryAgilityLadder, model.AccessoryChair,
		}
		for i, a := range accTypes {
			if ms.state.ToolAccessoryType == a {
				idx = 16 + i
				break
			}
		}
	}
	if idx >= 0 && idx < len(ms.allBtns) {
		ms.allBtns[idx].OverrideColor = toolActiveColor
		ms.allBtns[idx].Refresh()
	}
	// Ball toggle highlight (independent of tool selection).
	if ms.ballToggleIdx >= 0 && ms.ballToggleIdx < len(ms.allBtns) && ms.allBtns[ms.ballToggleIdx] != nil {
		if ms.state.ToolWithBall {
			ms.allBtns[ms.ballToggleIdx].OverrideColor = &color.NRGBA{R: 0xf4, G: 0xa2, B: 0x61, A: 0xff}
		}
		ms.allBtns[ms.ballToggleIdx].Refresh()
	}
	if ms.palette != nil {
		ms.palette.ForceUpdateActive()
	}
	if ms.OnToolChanged != nil {
		ms.OnToolChanged()
	}
}

// selectionTab returns the shelf tab that should show properties for the given selection kind.
func selectionTab(kind editor.SelectionKind) shelfCategory {
	switch kind {
	case editor.SelectPlayer:
		return shelfPlayers
	case editor.SelectAccessory:
		return shelfAccessories
	default:
		return shelfActions
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
	// Cancel active action creation (clear ActionFrom) when switching tabs.
	ms.state.ActionFrom = nil
	ms.refreshShelfContent()
	ms.updateTabIndicators()
}

func (ms *EditorShelf) updateTabIndicators() {
	for i := range numShelfCategories {
		if shelfCategory(i) == ms.active {
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

	// Start collapsed — maximize court space on launch.
	ms.collapsed = true
	ms.shelfOuter.Hide()
	ms.chevronBtn.Icon = icon.ChevronUp

	// Tab bar.
	tabs := []struct {
		ico   fyne.Resource
		label string
		cat   shelfCategory
	}{
		{icon.PlayerAttacker, i18n.T(i18n.KeyMobileShelfPlayers), shelfPlayers},
		{icon.ActionPass, i18n.T(i18n.KeyMobileShelfActions), shelfActions},
		{icon.AccCone, i18n.T(i18n.KeyMobileShelfAccessories), shelfAccessories},
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

// SetZoomController sets the zoom controller (court widget).
func (ms *EditorShelf) SetZoomController(z ZoomController) {
	ms.zoomer = z
}

// RefreshLanguage updates tab labels.
func (ms *EditorShelf) RefreshLanguage() {
	keys := []string{
		i18n.KeyMobileShelfPlayers,
		i18n.KeyMobileShelfActions, i18n.KeyMobileShelfAccessories,
	}
	for i, k := range keys {
		if ms.tabLabels[i] != nil {
			ms.tabLabels[i].Text = i18n.T(k)
			ms.tabLabels[i].Refresh()
		}
	}
	// Refresh button tooltips.
	for i, btn := range ms.allBtns {
		if i < len(ms.btnKeys) {
			btn.SetTooltip(i18n.T(ms.btnKeys[i]))
		}
	}
	ms.propsBallChk.Text = i18n.T(i18n.KeyPropsBall)
	ms.propsBallChk.Refresh()
	ms.propsLabelE.PlaceHolder = i18n.T(i18n.KeyPropsName)
	ms.propsLabelE.Refresh()
	ms.propsCalloutSel.PlaceHolder = i18n.T(i18n.KeyPropsCallout)
	ms.propsCalloutSel.Options = allCalloutLabels()
	ms.propsCalloutSel.Refresh()
	ms.propsRoleSel.Options = allRoleLabels()
	ms.propsRoleSel.Refresh()
	// Force props rebuild on next update (labels changed).
	ms.propsSyncedSel = nil
}

// refreshShelfContent updates the shelf content based on active category and selection.
func (ms *EditorShelf) refreshShelfContent() {
	var content fyne.CanvasObject
	switch ms.active {
	case shelfPlayers:
		if ms.propsContent != nil && len(ms.propsContent.Objects) > 0 &&
			ms.propsSyncedSel != nil && ms.propsSyncedSel.Kind == editor.SelectPlayer {
			content = container.NewVBox(ms.playerContent, widget.NewSeparator(), ms.propsContent)
		} else {
			content = ms.playerContent
		}
	case shelfActions:
		if ms.propsSyncedSel != nil && ms.propsSyncedSel.Kind == editor.SelectAction &&
			ms.propsContent != nil && len(ms.propsContent.Objects) > 0 {
			// Action selected → show only props (type buttons replace the creation grid).
			content = ms.propsContent
		} else if ms.propsSyncedSel != nil && ms.propsSyncedSel.Kind == editor.SelectPlayer &&
			ms.exercise != nil && ms.propsSyncedSel.SeqIndex < len(ms.exercise.Sequences) {
			// Player selected → show player's action list.
			actionsList := ms.buildPlayerActionsList()
			if actionsList != nil {
				content = container.NewVBox(ms.actionContent, widget.NewSeparator(), actionsList)
			} else {
				content = ms.actionContent
			}
		} else {
			content = ms.actionContent
		}
	case shelfAccessories:
		if ms.propsContent != nil && len(ms.propsContent.Objects) > 0 &&
			ms.propsSyncedSel != nil && ms.propsSyncedSel.Kind == editor.SelectAccessory {
			content = container.NewVBox(ms.accContent, widget.NewSeparator(), ms.propsContent)
		} else {
			content = ms.accContent
		}
	}
	ms.shelfStack.Objects = []fyne.CanvasObject{content}
	ms.shelfStack.Refresh()
}

// buildPlayerActionsList builds a list of actions involving the selected player, with delete buttons.
func (ms *EditorShelf) buildPlayerActionsList() fyne.CanvasObject {
	sel := ms.propsSyncedSel
	if sel == nil || sel.Kind != editor.SelectPlayer {
		return nil
	}
	if ms.exercise == nil || sel.SeqIndex >= len(ms.exercise.Sequences) {
		return nil
	}
	seq := &ms.exercise.Sequences[sel.SeqIndex]
	if sel.Index >= len(seq.Players) {
		return nil
	}
	playerID := seq.Players[sel.Index].ID

	// Find all actions involving this player.
	var rows []fyne.CanvasObject
	headerTxt := canvas.NewText(
		fmt.Sprintf("%s — %s", i18n.T(i18n.KeyToolHeaderActions), playerID),
		color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff},
	)
	headerTxt.TextStyle.Bold = true
	if isMobile {
		headerTxt.TextSize = 13
	} else {
		headerTxt.TextSize = 11
	}
	rows = append(rows, headerTxt)

	found := false
	for ai, act := range seq.Actions {
		involves := (act.From.IsPlayer && act.From.PlayerID == playerID) ||
			(act.To.IsPlayer && act.To.PlayerID == playerID)
		if !involves {
			continue
		}
		found = true
		actionIdx := ai

		// Build label: "Pass → D1" or "Cut → (0.5, 0.3)"
		label := actionDisplayLabel(act.Type)
		if act.To.IsPlayer {
			label += " → " + act.To.PlayerID
		} else {
			label += fmt.Sprintf(" → (%.2f, %.2f)", act.To.Position.X(), act.To.Position.Y())
		}
		if act.From.IsPlayer && act.From.PlayerID != playerID {
			label = act.From.PlayerID + " → " + label
		}

		lblText := canvas.NewText(label, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
		if isMobile {
			lblText.TextSize = 13
		} else {
			lblText.TextSize = 11
		}

		delBtn := NewTipButton(icon.Delete(), i18n.T(i18n.KeyToolDelete), func() {
			if actionIdx < len(seq.Actions) {
				seq.Actions = append(seq.Actions[:actionIdx], seq.Actions[actionIdx+1:]...)
				model.ReorderSteps(seq)
				ms.state.MarkModified()
				ms.propsSyncedSel = nil // force rebuild
				ms.refreshShelfContent()
				if ms.OnToolChanged != nil {
					ms.OnToolChanged()
				}
			}
		})
		delBtn.SetImportance(widget.DangerImportance)

		btnSize := fyne.NewSize(28, 28)
		if isMobile {
			btnSize = fyne.NewSize(40, 40)
		}
		row := container.NewBorder(nil, nil, nil, container.NewGridWrap(btnSize, delBtn), lblText)
		rows = append(rows, row)
	}

	if !found {
		return nil
	}
	return container.NewVBox(rows...)
}

// UpdateElementProps updates the compact element properties view in the tools shelf.
func (ms *EditorShelf) UpdateElementProps(exercise *model.Exercise, state *editor.EditorState, seqIdx int) {
	if ms.propsUpdating {
		return
	}
	ms.propsUpdating = true
	defer func() { ms.propsUpdating = false }()

	ms.exercise = exercise
	ms.seqIdx = seqIdx
	sel := state.SelectedElement

	// No selection → clear props and refresh current tab.
	if sel == nil || exercise == nil || seqIdx >= len(exercise.Sequences) {
		ms.propsSyncedSel = nil
		ms.propsContent.RemoveAll()
		ms.refreshShelfContent()
		return
	}

	// Auto-switch to the relevant tab when selecting an element.
	// Exception: stay in Actions tab when selecting a player (shows player's action list).
	isSelecting := ms.state.ActiveTool == editor.ToolSelect || ms.state.ActiveTool == editor.ToolNone
	if isSelecting {
		targetTab := selectionTab(sel.Kind)
		stayInActions := ms.active == shelfActions && sel.Kind == editor.SelectPlayer
		if !stayInActions && ms.active != targetTab {
			ms.active = targetTab
			ms.updateTabIndicators()
		}
		ms.expand()
	}

	seq := &exercise.Sequences[seqIdx]

	// Check if the selection changed — if not, just sync values and ensure shelf shows props.
	sameSelection := ms.propsSyncedSel != nil && *ms.propsSyncedSel == *sel
	if sameSelection {
		ms.syncPropsValues(seq, sel)
		ms.refreshShelfContent()
		return
	}

	// Selection changed → full rebuild.
	ms.propsSyncedSel = &editor.Selection{Kind: sel.Kind, Index: sel.Index, SeqIndex: sel.SeqIndex}
	ms.propsContent.RemoveAll()
	ms.buildPropsLayout(exercise, state, seq, sel, seqIdx)
	ms.refreshShelfContent()
}

// buildPropsLayout builds the compact element properties layout for the given selection.
func (ms *EditorShelf) buildPropsLayout(_ *model.Exercise, state *editor.EditorState, seq *model.Sequence, sel *editor.Selection, _ int) {
	switch sel.Kind {
	case editor.SelectPlayer:
		if sel.Index >= len(seq.Players) {
			return
		}
		p := &seq.Players[sel.Index]

		// Wire callbacks.
		ms.propsLabelE.OnChanged = func(s string) {
			if ms.propsUpdating || sel.Index >= len(seq.Players) {
				return
			}
			seq.Players[sel.Index].Label = s
			ms.propsTitle.Text = roleDisplayLabel(seq.Players[sel.Index].Role) + " " + s
			ms.propsTitle.Refresh()
			state.MarkModified()
		}
		ms.propsRoleSel.Options = allRoleLabels()
		ms.propsRoleSel.OnChanged = func(s string) {
			if ms.propsUpdating || sel.Index >= len(seq.Players) {
				return
			}
			if r, ok := roleLabelToRole(s); ok {
				seq.Players[sel.Index].Role = r
				state.MarkModified()
				if ms.OnToolChanged != nil {
					ms.OnToolChanged()
				}
			}
		}
		ms.propsBallBtn.SetOnTapped(func() {
			if ms.propsUpdating || sel.Index >= len(seq.Players) {
				return
			}
			pid := seq.Players[sel.Index].ID
			if seq.BallCarrier.HasBall(pid) {
				seq.BallCarrier.RemoveBall(pid)
			} else {
				seq.BallCarrier.AddBall(pid)
			}
			state.MarkModified()
			ms.propsSyncedSel = nil // force rebuild to update ball highlight
			if ms.OnToolChanged != nil {
				ms.OnToolChanged()
			}
		})
		ms.propsCalloutSel.Options = allCalloutLabels()
		ms.propsCalloutSel.OnChanged = func(s string) {
			if ms.propsUpdating || sel.Index >= len(seq.Players) {
				return
			}
			seq.Players[sel.Index].Callout = calloutLabelToValue(s)
			state.MarkModified()
			if ms.OnToolChanged != nil {
				ms.OnToolChanged()
			}
		}
		ms.wirePosEntries(seq, sel, state)
		ms.propsRotKnob.Value = p.Rotation
		ms.propsRotKnob.Refresh()
		ms.propsRotKnob.OnChanged = func(v float64) {
			if ms.propsUpdating || sel.Index >= len(seq.Players) {
				return
			}
			seq.Players[sel.Index].Rotation = v
			state.MarkModified()
			if ms.OnToolChanged != nil {
				ms.OnToolChanged()
			}
		}

		// Set initial values (including label entry text for first build).
		ms.propsLabelE.SetText(p.Label)
		ms.propsPosXEntry.SetText(fmt.Sprintf("%.0f", p.Position.X()*100))
		ms.propsPosYEntry.SetText(fmt.Sprintf("%.0f", p.Position.Y()*100))
		ms.syncPlayerValues(seq, p)

		// Layout: fields wrap to next line when shelf is too narrow.
		ms.propsLabelE.SetMinRowsVisible(1)
		labelMinW := fyne.NewSize(120, ms.propsLabelE.MinSize().Height)
		if isMobile {
			labelMinW = fyne.NewSize(160, ms.propsLabelE.MinSize().Height)
		}
		calloutMinW := fyne.NewSize(120, ms.propsCalloutSel.MinSize().Height)
		posMinW := fyne.NewSize(55, ms.propsPosXEntry.MinSize().Height)
		xLabel := canvas.NewText("X", color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff})
		xLabel.TextSize = 11
		yLabel := canvas.NewText("Y", color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff})
		yLabel.TextSize = 11
		fields := container.New(newFlowLayout(4, 4),
			container.NewGridWrap(labelMinW, ms.propsLabelE),
			ms.propsRoleSel,
			container.NewGridWrap(shelfCellSize, ms.propsBallBtn),
			container.NewGridWrap(calloutMinW, ms.propsCalloutSel),
			container.NewHBox(xLabel, container.NewGridWrap(posMinW, ms.propsPosXEntry)),
			container.NewHBox(yLabel, container.NewGridWrap(posMinW, ms.propsPosYEntry)),
		)
		delWrap := container.NewGridWrap(shelfCellSize, ms.propsDeleteBtn)
		leftCol := container.NewVBox(ms.propsTitle, fields, container.NewHBox(delWrap))
		ms.propsContent.Add(container.NewBorder(nil, nil, nil, ms.propsDpad, leftCol))

	case editor.SelectAccessory:
		if sel.Index >= len(seq.Accessories) {
			return
		}
		a := &seq.Accessories[sel.Index]
		ms.propsTitle.Text = i18n.T("tool.accessory." + accessoryI18nSuffix(a.Type))
		ms.wirePosEntries(seq, sel, state)
		ms.propsPosXEntry.SetText(fmt.Sprintf("%.0f", a.Position.X()*100))
		ms.propsPosYEntry.SetText(fmt.Sprintf("%.0f", a.Position.Y()*100))
		ms.propsRotKnob.Value = a.Rotation
		ms.propsRotKnob.Refresh()
		ms.propsRotKnob.OnChanged = func(v float64) {
			if ms.propsUpdating || sel.Index >= len(seq.Accessories) {
				return
			}
			seq.Accessories[sel.Index].Rotation = v
			state.MarkModified()
			if ms.OnToolChanged != nil {
				ms.OnToolChanged()
			}
		}
		posW := fyne.NewSize(60, ms.propsPosXEntry.MinSize().Height)
		posXWrap := container.NewGridWrap(posW, ms.propsPosXEntry)
		posYWrap := container.NewGridWrap(posW, ms.propsPosYEntry)
		row1 := container.NewHBox(ms.propsTitle, posXWrap, posYWrap)
		delWrap := container.NewGridWrap(shelfCellSize, ms.propsDeleteBtn)
		row2 := container.NewHBox(delWrap)
		leftCol := container.NewVBox(row1, row2)
		ms.propsContent.Add(container.NewBorder(nil, nil, nil, ms.propsDpad, leftCol))

	case editor.SelectAction:
		if sel.Index >= len(seq.Actions) {
			return
		}
		act := &seq.Actions[sel.Index]
		step := act.EffectiveStep()
		ms.propsTitle.Text = fmt.Sprintf("%s — %s %d", actionDisplayLabel(act.Type), i18n.T(i18n.KeyPropsStep), step)

		// Action type buttons — tap to change type.
		actionTypes := []model.ActionType{
			model.ActionDribble, model.ActionPass, model.ActionCut,
			model.ActionScreen, model.ActionShot, model.ActionHandoff,
		}
		actionIcons := []fyne.Resource{
			icon.ActionDribble, icon.ActionPass, icon.ActionCut,
			icon.ActionScreen, icon.ActionShot, icon.ActionHandoffRes,
		}
		typeGrid := container.NewGridWrap(shelfCellSize)
		for i, at := range actionTypes {
			actionType := at
			btn := NewTipButton(actionIcons[i], actionDisplayLabel(at), func() {
				if ms.propsUpdating || sel.Index >= len(seq.Actions) {
					return
				}
				seq.Actions[sel.Index].Type = actionType
				ms.propsTitle.Text = fmt.Sprintf("%s — %s %d", actionDisplayLabel(actionType), i18n.T(i18n.KeyPropsStep), seq.Actions[sel.Index].EffectiveStep())
				ms.propsTitle.Refresh()
				state.MarkModified()
				ms.propsSyncedSel = nil // force rebuild to update highlights
				if ms.OnToolChanged != nil {
					ms.OnToolChanged()
				}
			})
			if actionType == act.Type {
				btn.OverrideColor = toolActiveColor
			}
			typeGrid.Add(btn)
		}

		stepLabel := canvas.NewText(fmt.Sprintf("%s %d", i18n.T(i18n.KeyPropsStep), step), color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff})
		stepLabel.TextSize = 12
		swapStep := func(delta int) {
			if sel.Index >= len(seq.Actions) {
				return
			}
			cur := seq.Actions[sel.Index].EffectiveStep()
			target := cur + delta
			if target < 1 || target > model.MaxStep(seq) {
				return
			}
			// Swap: all actions at current step go to target, all at target go to current.
			for i := range seq.Actions {
				s := seq.Actions[i].EffectiveStep()
				if s == cur {
					seq.Actions[i].Step = target
				} else if s == target {
					seq.Actions[i].Step = cur
				}
			}
			model.ReorderSteps(seq)
			state.MarkModified()
			ms.propsSyncedSel = nil
			if ms.OnToolChanged != nil {
				ms.OnToolChanged()
			}
		}
		minusBtn := NewTipButton(fynetheme.ContentRemoveIcon(), "", func() { swapStep(-1) })
		plusBtn := NewTipButton(fynetheme.ContentAddIcon(), "", func() { swapStep(1) })
		stepRow := container.NewHBox(stepLabel, minusBtn, plusBtn)
		delWrap := container.NewGridWrap(shelfCellSize, ms.propsDeleteBtn)

		// Validation messages.
		propsCol := container.NewVBox(ms.propsTitle, typeGrid, stepRow, container.NewHBox(delWrap))
		actionIssues := model.ValidateActions(seq)
		if issues, ok := actionIssues[sel.Index]; ok {
			for _, issue := range issues {
				prefix := "⚠️ "
				col := color.NRGBA{R: 0xcc, G: 0x88, B: 0x00, A: 0xff}
				if issue.IsError {
					prefix = "⛔ "
					col = color.NRGBA{R: 0xcc, G: 0x22, B: 0x22, A: 0xff}
				}
				msg := canvas.NewText(prefix+i18n.T(issue.Message), col)
				msg.TextSize = 11
				propsCol.Add(msg)
			}
		}
		ms.propsContent.Add(propsCol)
	}
}

// syncPropsValues updates widget values without rebuilding the layout (avoids entry focus loss).
func (ms *EditorShelf) syncPropsValues(seq *model.Sequence, sel *editor.Selection) {
	switch sel.Kind {
	case editor.SelectPlayer:
		if sel.Index < len(seq.Players) {
			p := &seq.Players[sel.Index]
			ms.syncPlayerValues(seq, p)
			ms.propsPosXEntry.SetText(fmt.Sprintf("%.0f", p.Position.X()*100))
			ms.propsPosYEntry.SetText(fmt.Sprintf("%.0f", p.Position.Y()*100))
		}
	case editor.SelectAccessory:
		if sel.Index < len(seq.Accessories) {
			a := &seq.Accessories[sel.Index]
			ms.propsPosXEntry.SetText(fmt.Sprintf("%.0f", a.Position.X()*100))
			ms.propsPosYEntry.SetText(fmt.Sprintf("%.0f", a.Position.Y()*100))
		}
	}
}

// syncPlayerValues sets widget values for a player without triggering callbacks.
func (ms *EditorShelf) syncPlayerValues(seq *model.Sequence, p *model.Player) {
	ms.propsTitle.Text = roleDisplayLabel(p.Role) + " " + p.Label
	ms.propsTitle.Refresh()
	// Don't SetText on label/pos entries — it would steal focus during typing.
	ms.propsRoleSel.SetSelected(roleDisplayLabel(p.Role))
	if seq.BallCarrier.HasBall(p.ID) {
		ms.propsBallBtn.OverrideColor = &color.NRGBA{R: 0xf4, G: 0xa2, B: 0x61, A: 0xff}
	} else {
		ms.propsBallBtn.OverrideColor = nil
	}
	ms.propsBallBtn.Refresh()
	if p.Callout != "" {
		ms.propsCalloutSel.SetSelected(i18n.T("callout." + string(p.Callout)))
	} else {
		ms.propsCalloutSel.SetSelected(i18n.T(i18n.KeyCalloutNone))
	}
}


// wirePosEntries sets up OnChanged callbacks for X/Y position entries.
func (ms *EditorShelf) wirePosEntries(seq *model.Sequence, sel *editor.Selection, state *editor.EditorState) {
	ms.propsPosXEntry.OnChanged = func(s string) {
		if ms.propsUpdating {
			return
		}
		var v float64
		if _, err := fmt.Sscanf(s, "%f", &v); err != nil {
			return
		}
		pos := ms.getSelPos(seq, sel)
		if pos == nil {
			return
		}
		pos[0] = v / 100
		state.MarkModified()
		if ms.OnToolChanged != nil {
			ms.OnToolChanged()
		}
	}
	ms.propsPosYEntry.OnChanged = func(s string) {
		if ms.propsUpdating {
			return
		}
		var v float64
		if _, err := fmt.Sscanf(s, "%f", &v); err != nil {
			return
		}
		pos := ms.getSelPos(seq, sel)
		if pos == nil {
			return
		}
		pos[1] = v / 100
		state.MarkModified()
		if ms.OnToolChanged != nil {
			ms.OnToolChanged()
		}
	}
}

func (ms *EditorShelf) getSelPos(seq *model.Sequence, sel *editor.Selection) *model.Position {
	switch sel.Kind {
	case editor.SelectPlayer:
		if sel.Index < len(seq.Players) {
			return &seq.Players[sel.Index].Position
		}
	case editor.SelectAccessory:
		if sel.Index < len(seq.Accessories) {
			return &seq.Accessories[sel.Index].Position
		}
	}
	return nil
}

func allRoleLabels() []string {
	return []string{
		i18n.T(i18n.KeyToolPlayerAttacker),
		i18n.T(i18n.KeyToolPlayerDefender),
		i18n.T(i18n.KeyToolPlayerCoach),
		i18n.T(i18n.KeyToolPlayerPg),
		i18n.T(i18n.KeyToolPlayerSg),
		i18n.T(i18n.KeyToolPlayerSf),
		i18n.T(i18n.KeyToolPlayerPf),
		i18n.T(i18n.KeyToolPlayerCenter),
	}
}

// nudgeSelection moves the selected element by dx, dy in relative coordinates.
func (ms *EditorShelf) nudgeSelection(dx, dy float64) {
	sel := ms.state.SelectedElement
	if sel == nil || ms.exercise == nil || ms.seqIdx >= len(ms.exercise.Sequences) {
		return
	}
	seq := &ms.exercise.Sequences[ms.seqIdx]
	switch sel.Kind {
	case editor.SelectPlayer:
		if sel.Index < len(seq.Players) {
			seq.Players[sel.Index].Position[0] += dx
			seq.Players[sel.Index].Position[1] += dy
		}
	case editor.SelectAccessory:
		if sel.Index < len(seq.Accessories) {
			seq.Accessories[sel.Index].Position[0] += dx
			seq.Accessories[sel.Index].Position[1] += dy
		}
	default:
		return
	}
	ms.state.MarkModified()
	if ms.OnToolChanged != nil {
		ms.OnToolChanged()
	}
}

func allCalloutLabels() []string {
	all := model.AllCallouts()
	labels := make([]string, 0, 1+len(all))
	labels = append(labels, i18n.T(i18n.KeyCalloutNone))
	for _, c := range all {
		labels = append(labels, i18n.T("callout."+string(c)))
	}
	return labels
}

func calloutLabelToValue(label string) model.CalloutType {
	if label == i18n.T(i18n.KeyCalloutNone) {
		return ""
	}
	for _, c := range model.AllCallouts() {
		if i18n.T("callout."+string(c)) == label {
			return c
		}
	}
	return ""
}

func accessoryI18nSuffix(t model.AccessoryType) string {
	switch t {
	case model.AccessoryCone:
		return "cone"
	case model.AccessoryAgilityLadder:
		return "ladder"
	case model.AccessoryChair:
		return "chair"
	default:
		return string(t)
	}
}

func actionDisplayLabel(at model.ActionType) string {
	switch model.NormalizeActionType(at) {
	case model.ActionPass:
		return i18n.T(i18n.KeyToolActionPass)
	case model.ActionDribble:
		return i18n.T(i18n.KeyToolActionDribble)
	case model.ActionCut:
		return i18n.T(i18n.KeyToolActionCut)
	case model.ActionScreen:
		return i18n.T(i18n.KeyToolActionScreen)
	case model.ActionShot:
		return i18n.T(i18n.KeyToolActionShot)
	case model.ActionHandoff:
		return i18n.T(i18n.KeyToolActionHandoff)
	default:
		return string(at)
	}
}

func roleLabelToRole(label string) (model.PlayerRole, bool) {
	roles := []model.PlayerRole{
		model.RoleAttacker, model.RoleDefender, model.RoleCoach,
		model.RolePointGuard, model.RoleShootingGuard, model.RoleSmallForward,
		model.RolePowerForward, model.RoleCenter,
	}
	for _, r := range roles {
		if roleDisplayLabel(r) == label {
			return r, true
		}
	}
	return "", false
}
