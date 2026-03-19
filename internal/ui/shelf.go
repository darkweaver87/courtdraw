package ui

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	fynetheme "fyne.io/fyne/v2/theme"
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
	state    *editor.EditorState
	palette  *ToolPalette
	exercise *model.Exercise
	seqIdx   int
	active   shelfCategory
	allBtns  []*TipButton

	// Shelf content per category.
	toolsContent  fyne.CanvasObject
	playerContent fyne.CanvasObject
	actionContent fyne.CanvasObject
	accContent    fyne.CanvasObject

	// Compact element properties (shown in tools shelf when element selected).
	propsContent    *fyne.Container
	propsTitle      *canvas.Text
	propsLabelE     *widget.Entry
	propsRoleSel    *widget.Select
	propsBallChk    *widget.Check
	propsCalloutSel *widget.Select
	propsPosXEntry  *widget.Entry
	propsPosYEntry  *widget.Entry
	propsDpad       *fyne.Container
	propsDeleteBtn  *TipButton
	propsUpdating   bool             // prevents recursive OnChanged → refreshEditor loop
	propsSyncedSel  *editor.Selection // tracks which element is currently displayed

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
	// --- Tools: select + delete ---
	selectBtn := ms.addBtn(icon.ToolSelect, i18n.KeyToolSelect, func() {
		ms.state.SetTool(editor.ToolSelect)
		ms.syncHighlights()
	})
	deleteBtn := ms.addBtn(icon.Delete(), i18n.KeyToolDelete, func() {
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
			ms.state.SetPlayerTool(r)
			ms.syncHighlights()
			ms.collapse()
		})
		playerGrid.Add(btn)
	}
	queueBtn := ms.addBtn(icon.PlayerQueue, i18n.KeyToolPlayerQueue, func() {
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
		i18n.KeyToolActionPass, i18n.KeyToolActionDribble, i18n.KeyToolActionSprint,
		i18n.KeyToolActionShot, i18n.KeyToolActionScreen, i18n.KeyToolActionCut,
		i18n.KeyToolActionCloseOut, i18n.KeyToolActionContest, i18n.KeyToolActionReverse,
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
			// If a player is already selected, use it as action source immediately.
			if sel := ms.state.SelectedElement; sel != nil && sel.Kind == editor.SelectPlayer {
				if ms.exercise != nil && sel.SeqIndex < len(ms.exercise.Sequences) {
					seq := &ms.exercise.Sequences[sel.SeqIndex]
					if sel.Index < len(seq.Players) {
						id := seq.Players[sel.Index].ID
						if model.RequiresBall(actionType) && !seq.BallCarrier.HasBall(id) {
							ms.state.SetStatus(i18n.T(i18n.KeyStatusRequiresBall), 1)
						} else {
							ms.state.ActionFrom = &id
						}
					}
				}
			}
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
	accKeys := []string{i18n.KeyToolAccessoryCone, i18n.KeyToolAccessoryLadder, i18n.KeyToolAccessoryChair}
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
	dpadUp := NewTipButton(fynetheme.MoveUpIcon(), "", func() { ms.nudgeSelection(0, dpadStep) })
	dpadDown := NewTipButton(fynetheme.MoveDownIcon(), "", func() { ms.nudgeSelection(0, -dpadStep) })
	dpadLeft := NewTipButton(fynetheme.NavigateBackIcon(), "", func() { ms.nudgeSelection(-dpadStep, 0) })
	dpadRight := NewTipButton(fynetheme.NavigateNextIcon(), "", func() { ms.nudgeSelection(dpadStep, 0) })
	empty := canvas.NewRectangle(color.Transparent)
	empty.SetMinSize(dpadBtnSize)
	dpadGrid := container.NewGridWithColumns(3,
		empty, container.NewGridWrap(dpadBtnSize, dpadUp), empty,
		container.NewGridWrap(dpadBtnSize, dpadLeft), empty, container.NewGridWrap(dpadBtnSize, dpadRight),
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
	idx := -1
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
	ms.refreshShelfContent()

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
		{icon.ToolSelect, i18n.T(i18n.KeyMobileShelfTools), shelfTools},
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

// RefreshLanguage updates tab labels.
func (ms *EditorShelf) RefreshLanguage() {
	keys := []string{
		i18n.KeyMobileShelfTools, i18n.KeyMobileShelfPlayers,
		i18n.KeyMobileShelfActions, i18n.KeyMobileShelfAccessories,
	}
	for i, k := range keys {
		if ms.tabLabels[i] != nil {
			ms.tabLabels[i].Text = i18n.T(k)
			ms.tabLabels[i].Refresh()
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
	case shelfTools:
		if ms.propsContent != nil && len(ms.propsContent.Objects) > 0 {
			content = ms.propsContent
		} else {
			content = ms.toolsContent
		}
	case shelfPlayers:
		content = ms.playerContent
	case shelfActions:
		content = ms.actionContent
	case shelfAccessories:
		content = ms.accContent
	}
	ms.shelfStack.Objects = []fyne.CanvasObject{content}
	ms.shelfStack.Refresh()
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

	// No selection → clear props and show tools.
	if sel == nil || exercise == nil || seqIdx >= len(exercise.Sequences) {
		ms.propsSyncedSel = nil
		ms.propsContent.RemoveAll()
		if ms.active == shelfTools {
			ms.refreshShelfContent()
		}
		return
	}

	seq := &exercise.Sequences[seqIdx]

	// Check if the selection changed — if not, just sync values in place (no layout rebuild).
	sameSelection := ms.propsSyncedSel != nil && *ms.propsSyncedSel == *sel
	if sameSelection {
		ms.syncPropsValues(seq, sel)
		return
	}

	// Selection changed → full rebuild.
	ms.propsSyncedSel = &editor.Selection{Kind: sel.Kind, Index: sel.Index, SeqIndex: sel.SeqIndex}
	ms.propsContent.RemoveAll()
	ms.buildPropsLayout(exercise, state, seq, sel, seqIdx)

	if ms.active == shelfTools {
		ms.refreshShelfContent()
	}
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
		ms.propsBallChk.OnChanged = func(checked bool) {
			if ms.propsUpdating || sel.Index >= len(seq.Players) {
				return
			}
			pid := seq.Players[sel.Index].ID
			if checked {
				seq.BallCarrier.AddBall(pid)
			} else {
				seq.BallCarrier.RemoveBall(pid)
			}
			state.MarkModified()
			if ms.OnToolChanged != nil {
				ms.OnToolChanged()
			}
		}
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
		fields := container.New(newFlowLayout(4, 4),
			container.NewGridWrap(labelMinW, ms.propsLabelE),
			ms.propsRoleSel,
			ms.propsBallChk,
			container.NewGridWrap(calloutMinW, ms.propsCalloutSel),
			container.NewGridWrap(posMinW, ms.propsPosXEntry),
			container.NewGridWrap(posMinW, ms.propsPosYEntry),
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
		ms.propsTitle.Text = actionDisplayLabel(act.Type)
		delWrap := container.NewGridWrap(shelfCellSize, ms.propsDeleteBtn)
		ms.propsContent.Add(container.NewVBox(ms.propsTitle, container.NewHBox(delWrap)))
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
	ms.propsBallChk.SetChecked(seq.BallCarrier.HasBall(p.ID))
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
	switch at {
	case model.ActionPass:
		return i18n.T(i18n.KeyToolActionPass)
	case model.ActionDribble:
		return i18n.T(i18n.KeyToolActionDribble)
	case model.ActionSprint:
		return i18n.T(i18n.KeyToolActionSprint)
	case model.ActionShotLayup, model.ActionShotPushup, model.ActionShotJump:
		return i18n.T(i18n.KeyToolActionShot)
	case model.ActionScreen:
		return i18n.T(i18n.KeyToolActionScreen)
	case model.ActionCut:
		return i18n.T(i18n.KeyToolActionCut)
	case model.ActionCloseOut:
		return i18n.T(i18n.KeyToolActionCloseOut)
	case model.ActionContest:
		return i18n.T(i18n.KeyToolActionContest)
	case model.ActionReverse:
		return i18n.T(i18n.KeyToolActionReverse)
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
