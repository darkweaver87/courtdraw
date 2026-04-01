package ui

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
	"gopkg.in/yaml.v3"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/anim"
	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/pdf"
	"github.com/darkweaver87/courtdraw/internal/share"
	"github.com/darkweaver87/courtdraw/internal/store"
	"github.com/darkweaver87/courtdraw/internal/ui/editor"
	"github.com/darkweaver87/courtdraw/internal/ui/fynecourt"
	"github.com/darkweaver87/courtdraw/internal/ui/icon"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
)

// App is the main application state.
type App struct {
	window   fyne.Window
	store    store.Store
	settings *store.Settings
	library  *store.Library
	syncing bool

	exercise      *model.Exercise
	exerciseSHA   string // SHA of exercise at load/save time
	editorState   editor.EditorState
	playback      *anim.Playback
	editLang      string

	// Fyne widgets.
	court        *fynecourt.CourtWidget
	fileToolbar  *FileToolbar
	toolPalette  *ToolPalette
	propsPanel   *PropertiesPanel
	seqTimeline  *SeqTimeline
	instrPanel   *InstructionsPanel
	animControls    *AnimControls
	actionTimeline  *ActionTimeline
	viewTools       *ViewTools
	statusBar       *StatusBar
	tooltipLayer *TooltipLayer
	sessionTab   *SessionTab
	myFilesTab   *MyFilesTab
	teamTab      *TeamTab
	matchTab     *MatchTab

	// Navigation state.
	sessionNeedsRefresh bool
	myFilesNeedsRefresh bool

	// Language selector.
	langBtn       *TipButton    // flag icon button, opens dropdown
	editLangLabel *canvas.Text  // kept for desktop compat (unused in unified layout)

	// Unified layout.
	editorShelf    *EditorShelf
	editorMode     EditorMode
	modeSwitchFunc func(EditorMode) // set by buildUnifiedRoot
	modeLabel      *canvas.Text     // current mode name in top bar
	moreBtn        *TipButton       // "more" menu button (tooltip needs lang refresh)

	// Training mode.
	trainingMode  *TrainingMode
	normalContent fyne.CanvasObject

	// Match live mode.
	matchLive *MatchLive

	// Undo/redo history.
	history     *editor.History
	undoRedoing bool // guard to prevent pushing snapshots during undo/redo restore

	// QR scan import.
	scanPending bool

	// Empty state overlay.
	emptyState      *fyne.Container
	emptyTitle      *canvas.Text
	emptySub        *canvas.Text
	emptyNewBtn     *widget.Button
	emptyRecentList *fyne.Container

}

// NewApp creates a new App instance.
func NewApp(st store.Store, settings *store.Settings, lib *store.Library, w fyne.Window) *App {
	a := &App{
		window:   w,
		store:    st,
		settings: settings,
		library:  lib,
		editLang: string(i18n.CurrentLang()),
	}
	a.editorState.ActiveTool = editor.ToolSelect
	a.history = editor.NewHistory()
	return a
}

// BuildUI creates the full application UI and returns the root canvas object.
func (a *App) BuildUI() fyne.CanvasObject {
	// Create all panel widgets.
	a.court = fynecourt.NewCourtWidget()
	a.court.ShowApron = a.settings.ApronVisible()
	a.court.SetEditorState(&a.editorState)
	a.court.OnChanged = func() {
		a.pushSnapshot()
		a.refreshEditor()
	}
	a.court.OnDragEnd = func() {
		a.pushSnapshot() // just push snapshot, no refreshEditor (avoids layout issues)
	}

	a.fileToolbar = NewFileToolbar()
	a.fileToolbar.OnAction = func(action FileAction) {
		a.handleFileAction(action)
	}

	a.toolPalette = NewToolPalette(&a.editorState)
	a.toolPalette.OnToolChanged = func() {
		if a.editorState.DeleteRequested {
			a.editorState.DeleteRequested = false
			a.court.DeleteSelected()
		}
		a.viewTools.SyncToolHighlight()
		a.court.Refresh()
	}

	a.propsPanel = NewPropertiesPanel()
	a.propsPanel.Window = a.window
	a.propsPanel.OnModified = func() {
		a.pushSnapshot()
		a.court.Refresh()
		a.updateWindowTitle()
	}

	a.seqTimeline = NewSeqTimeline()
	a.seqTimeline.OnSeqChanged = func(idx int) {
		a.court.SetSequence(idx)
		a.editorState.Deselect()
		a.editorState.ActionFrom = nil
		a.refreshEditor()
	}
	a.seqTimeline.OnAddSeq = func() {
		a.addSequence()
	}
	a.seqTimeline.OnDeleteSeq = func(idx int) {
		a.deleteSequence(idx)
	}
	a.seqTimeline.SetWindow(a.window)
	a.seqTimeline.OnSeqRenamed = func(idx int, newLabel string) {
		if a.exercise == nil || idx >= len(a.exercise.Sequences) {
			return
		}
		a.pushSnapshot()
		if a.editLang != "" && a.editLang != "en" {
			// Update the translated label.
			tr := a.exercise.EnsureI18n(a.editLang)
			for len(tr.Sequences) <= idx {
				tr.Sequences = append(tr.Sequences, model.SequenceI18n{})
			}
			tr.Sequences[idx].Label = newLabel
			a.exercise.SetI18n(a.editLang, tr)
		} else {
			// Update the primary label.
			a.exercise.Sequences[idx].Label = newLabel
		}
		a.editorState.Modified = true
		a.refreshEditor()
	}
	a.seqTimeline.OnSettings = func() {
		a.showExerciseSettingsDialog()
	}
	a.viewTools = NewViewTools()
	a.viewTools.ActiveTool = &a.editorState.ActiveTool
	a.viewTools.OnSelect = func() {
		a.editorState.SetTool(editor.ToolSelect)
		a.viewTools.SyncToolHighlight()
		a.court.Refresh()
	}
	a.viewTools.OnEraser = func() {
		a.editorState.SetTool(editor.ToolDelete)
		a.viewTools.SyncToolHighlight()
		a.court.Refresh()
	}
	a.viewTools.SyncToolHighlight()
	a.viewTools.OnZoomIn = func() { a.court.ZoomIn() }
	a.viewTools.OnZoomOut = func() { a.court.ZoomOut() }
	a.viewTools.OnZoomReset = func() { a.court.ResetZoom() }
	a.viewTools.OnRotate = func() { a.toggleOrientation() }
	a.viewTools.OnToggleApron = func() { a.toggleApron() }
	a.viewTools.SetApronVisible(a.court.ShowApron)

	a.instrPanel = NewInstructionsPanel()
	a.instrPanel.OnModified = func() {
		a.pushSnapshot()
		a.updateWindowTitle()
	}

	a.animControls = NewAnimControls()
	a.animControls.OnStateChanged = func() {
		a.syncAnimState()
	}

	a.actionTimeline = NewActionTimeline()
	a.actionTimeline.OnModified = func() {
		a.pushSnapshot()
		a.court.Refresh()
		a.refreshEditor()
	}

	a.statusBar = NewStatusBar()
	a.tooltipLayer = NewTooltipLayer()
	SetTipLayer(a.tooltipLayer)

	a.sessionTab = NewSessionTab()
	a.sessionTab.OnAction = func(ev SessionTabEvent) {
		a.handleSessionAction(ev)
	}
	a.sessionTab.OnSessionChanged = func() {
		a.resolveSessionExercises()
		a.updateWindowTitle()
	}
	a.sessionTab.OnStatus = func(msg string, level int) {
		a.statusBar.SetStatus(msg, level)
	}
	a.sessionTab.LoadExercise = func(name string) (*model.Exercise, error) {
		return a.loadExerciseAny(name)
	}

	a.myFilesTab = NewMyFilesTab()
	a.myFilesTab.OnAction = func(ev MyFilesEvent) {
		a.handleMyFilesAction(ev)
	}

	a.teamTab = NewTeamTab(a.store, a.window, func(msg string, level int) {
		a.statusBar.SetStatus(msg, level)
	})

	a.matchTab = NewMatchTab(a.store, a.window, func(msg string, level int) {
		a.statusBar.SetStatus(msg, level)
	})
	a.matchTab.OnStartMatch = func(match *model.Match, team *model.Team) {
		a.enterMatchLive(match, team)
	}
	a.matchTab.OnShowSummary = func(match *model.Match) {
		a.showMatchSummary(match)
	}

	// Init language buttons.
	a.initLangButtons()

	// Undo/redo keyboard shortcuts.
	a.window.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyZ,
		Modifier: fyne.KeyModifierControl,
	}, func(_ fyne.Shortcut) { a.Undo() })
	a.window.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyZ,
		Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift,
	}, func(_ fyne.Shortcut) { a.Redo() })
	a.window.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyY,
		Modifier: fyne.KeyModifierControl,
	}, func(_ fyne.Shortcut) { a.Redo() })
	// Also register built-in undo shortcut (Fyne sends this to focused widgets first).
	a.window.Canvas().AddShortcut(&fyne.ShortcutUndo{}, func(_ fyne.Shortcut) { a.Undo() })

	// Delete key shortcut (was in court's TypedKey, moved here since court is no longer Focusable).
	a.window.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		if e.Name == fyne.KeyDelete || e.Name == fyne.KeyBackspace {
			if a.editorMode == ModeEdition {
				a.court.DeleteSelected()
			}
		}
	})

	// Wire undo/redo callbacks from seq timeline.
	a.seqTimeline.OnUndo = func() { a.Undo() }
	a.seqTimeline.OnRedo = func() { a.Redo() }

	// Unified layout: mode selector replaces tabs on all platforms.
	root := a.buildUnifiedRoot()
	return container.NewStack(
		container.NewBorder(nil, a.statusBar.Widget(), nil, nil, root),
		a.tooltipLayer.Widget(),
	)
}

// initLangButtons creates the language toggle buttons (shared by desktop and mobile).
func (a *App) initLangButtons() {
	if a.langBtn != nil {
		return // already initialized
	}
	// Determine initial flag.
	currentFlag := icon.FlagEN
	if a.editLang == "fr" {
		currentFlag = icon.FlagFR
	}
	a.langBtn = NewTipButton(currentFlag, i18n.T(i18n.KeyLangTooltip), nil)
	a.langBtn.onTapped = func() {
		enItem := fyne.NewMenuItem("🇬🇧 EN", func() {
			if a.editLang != "en" {
				a.editLang = "en"
				a.langBtn.Icon = icon.FlagEN
				a.langBtn.Refresh()
				a.switchLang("en")
			}
		})
		frItem := fyne.NewMenuItem("🇫🇷 FR", func() {
			if a.editLang != "fr" {
				a.editLang = "fr"
				a.langBtn.Icon = icon.FlagFR
				a.langBtn.Refresh()
				a.switchLang("fr")
			}
		})
		menu := widget.NewPopUpMenu(fyne.NewMenu("", enItem, frItem), a.window.Canvas())
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(a.langBtn)
		menu.ShowAtPosition(fyne.NewPos(pos.X, pos.Y+a.langBtn.Size().Height+4))
	}

	a.editLangLabel = canvas.NewText("", color.Transparent)
	a.editLangLabel.TextSize = 1
}


// buildUnifiedRoot creates the complete mobile layout with mode selector
// replacing the desktop AppTabs. All modes (Edition, Animation, Notes,
// Session, My Files) are accessible from the dropdown.
func (a *App) buildUnifiedRoot() fyne.CanvasObject {
	a.editorMode = ModeEdition

	// ── Mode label (displayed in top bar) ──
	a.modeLabel = canvas.NewText(i18n.T(i18n.KeyModeEdition), color.White)
	if isMobile {
		a.modeLabel.TextSize = 20
	} else {
		a.modeLabel.TextSize = 12
	}
	a.modeLabel.TextStyle.Bold = true

	// ── Build all mode content panes ──

	// Edition mode: shelf + tab bar (bottom).
	a.editorShelf = NewEditorShelf(&a.editorState, a.toolPalette)
	a.editorShelf.SetZoomController(a.court)
	a.editorShelf.OnToolChanged = func() {
		if a.editorState.DeleteRequested {
			a.editorState.DeleteRequested = false
			a.court.DeleteSelected()
		}
		a.pushSnapshot()
		a.viewTools.SyncToolHighlight()
		a.court.Refresh()
		a.refreshEditor()
	}
	editionBottom := a.editorShelf.Widget()

	// Animation mode: playback controls (timeline is a side/bottom panel).
	animBottom := a.animControls.Widget()

	// Notes mode: full-page scrollable view (built dynamically on mode switch).
	notesContent := container.NewStack()

	// Training mode: full-page session picker (built dynamically on mode switch).
	trainingContent := container.NewStack()

	// Session mode: full session tab content (wrapped in Stack for rebuild).
	sessionContent := container.NewStack(a.sessionTab.Widget())

	// My Files mode: full my files tab content.
	myFilesContent := a.myFilesTab.Widget()

	// Team mode: full team tab content.
	teamContent := a.teamTab.Widget()

	// Match mode: full match tab content.
	matchContent := a.matchTab.Widget()

	// ── Content stack: court area (for editor modes) or full-page content ──
	// Sequence bar (shown only in Edition/Animation modes).
	seqBar := a.seqTimeline.Widget()

	// Empty state overlay (shown when no exercise is loaded).
	a.emptyTitle = canvas.NewText(i18n.T(i18n.KeyEmptyStateTitle), color.NRGBA{R: 0xaa, G: 0xaa, B: 0xaa, A: 0xff})
	a.emptyTitle.TextSize = 18
	a.emptyTitle.Alignment = fyne.TextAlignCenter
	a.emptySub = canvas.NewText(i18n.T(i18n.KeyEmptyStateSubtitle), color.NRGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff})
	a.emptySub.TextSize = 14
	a.emptySub.Alignment = fyne.TextAlignCenter
	a.emptyNewBtn = widget.NewButton(i18n.T(i18n.KeyTooltipNew), func() {
		a.handleFileAction(FileActionNew)
	})
	a.emptyNewBtn.Importance = widget.HighImportance
	a.emptyRecentList = container.NewVBox()
	a.refreshEmptyRecent()
	emptyContent := container.NewVBox(
		a.emptyTitle,
		a.emptySub,
		container.NewCenter(a.emptyNewBtn),
		a.emptyRecentList,
	)
	a.emptyState = container.NewCenter(emptyContent)

	// Hide seq bar initially (no exercise loaded).
	seqBar.Hide()

	// Action timeline panel (left of court).
	timelineWidget := a.actionTimeline.Widget()
	timelineBg := canvas.NewRectangle(color.NRGBA{R: 0x28, G: 0x28, B: 0x28, A: 0xff})
	timelinePanel := container.NewStack(timelineBg, container.NewPadded(timelineWidget))
	timelinePanel.Hide()

	// View tools panel (integrated into seq bar, right side).
	viewToolsPanel := a.viewTools.Widget()
	seqBarWithTools := container.NewBorder(nil, nil, nil, viewToolsPanel, seqBar)
	courtWithSeq := container.NewBorder(seqBarWithTools, nil, nil, timelinePanel, container.NewStack(a.court, a.emptyState))

	// Bottom area: swapped between edition shelf and animation controls.
	bottomStack := container.NewStack(editionBottom)
	courtSection := container.NewBorder(nil, bottomStack, nil, nil, courtWithSeq)

	// Full-page modes (no court).
	notesContent.Hide()
	trainingContent.Hide()
	sessionContent.Hide()
	myFilesContent.Hide()
	teamContent.Hide()
	matchContent.Hide()
	mainStack := container.NewStack(courtSection, notesContent, trainingContent, sessionContent, myFilesContent, teamContent, matchContent)

	// ── Mode switching logic ──
	var switchMode func(EditorMode)
	switchMode = func(mode EditorMode) {
		a.editorMode = mode

		// Hide everything.
		courtSection.Hide()
		notesContent.Hide()
		trainingContent.Hide()
		sessionContent.Hide()
		myFilesContent.Hide()
		teamContent.Hide()
		matchContent.Hide()

		switch mode {
		case ModeEdition:
			bottomStack.Objects = []fyne.CanvasObject{editionBottom}
			bottomStack.Refresh()
			courtSection.Show()
			seqBar.Show()
			timelinePanel.Hide()
			a.court.SetReadOnly(false)
			a.modeLabel.Text = i18n.T(i18n.KeyModeEdition)
		case ModeAnimation:
			bottomStack.Objects = []fyne.CanvasObject{animBottom}
			bottomStack.Refresh()
			courtSection.Show()
			seqBar.Show()
			timelinePanel.Show()
			a.court.SetReadOnly(true)
			a.modeLabel.Text = i18n.T(i18n.KeyModeAnimation)
			// Warn about validation issues.
			a.checkExerciseValidation()
		case ModeNotes:
			notesContent.Objects = []fyne.CanvasObject{a.buildNotesView()}
			notesContent.Show()
			a.modeLabel.Text = i18n.T(i18n.KeyModeNotes)
		case ModeSession:
			// Rebuild the entire session tab widget (same pattern as Notes/Training).
			a.sessionTab.Rebuild()
			a.sessionTab.SetExercises(a.buildManagedExercises())
			a.resolveSessionExercises()
			sessionContent.Objects = []fyne.CanvasObject{a.sessionTab.Widget()}
			sessionContent.Show()
			a.modeLabel.Text = i18n.T(i18n.KeyModeSession)
		case ModeMyFiles:
			myFilesContent.Show()
			a.myFilesNeedsRefresh = true
			a.refreshMyFilesTab()
			a.modeLabel.Text = i18n.T(i18n.KeyModeMyfiles)
		case ModeTraining:
			trainingContent.Objects = []fyne.CanvasObject{a.buildTrainingPicker()}
			trainingContent.Show()
			a.modeLabel.Text = i18n.T(i18n.KeyModeTraining)
		case ModeTeam:
			teamContent.Show()
			a.teamTab.RefreshTeamList()
			a.modeLabel.Text = i18n.T(i18n.KeyModeTeam)
		case ModeMatch:
			matchContent.Show()
			a.matchTab.RefreshMatchList()
			a.modeLabel.Text = i18n.T(i18n.KeyModeMatch)
		}
		a.modeLabel.Refresh()
		mainStack.Refresh()
		bottomStack.Refresh()
		a.updateWindowTitle()
	}

	// ── Mode selector dropdown ──
	// Mode icons mapping.
	modeIcons := map[EditorMode]fyne.Resource{
		ModeEdition:   fynetheme.DocumentCreateIcon(),
		ModeAnimation: fynetheme.MediaPlayIcon(),
		ModeNotes:     fynetheme.DocumentIcon(),
		ModeSession:   fynetheme.FolderIcon(),
		ModeMyFiles:   fynetheme.StorageIcon(),
		ModeTraining:  fynetheme.MediaPlayIcon(),
		ModeTeam:      fynetheme.AccountIcon(),
		ModeMatch:     fynetheme.MediaRecordIcon(),
	}
	modeIcon := canvas.NewImageFromResource(modeIcons[ModeEdition])
	modeIcon.FillMode = canvas.ImageFillContain
	modeIconSz := float32(18)
	if isMobile {
		modeIconSz = 32
	}
	modeIcon.SetMinSize(fyne.NewSize(modeIconSz, modeIconSz))
	modeChevron := canvas.NewImageFromResource(fynetheme.MoveDownIcon())
	modeChevron.FillMode = canvas.ImageFillContain
	modeChevron.SetMinSize(fyne.NewSize(12, 12))

	modeBg := canvas.NewRectangle(color.NRGBA{R: 0x22, G: 0x55, B: 0x44, A: 0xff})
	modeBg.CornerRadius = 6

	// Mode color mapping: edition domain = green, organization domain = blue.
	modeColors := map[EditorMode]color.NRGBA{
		ModeEdition:   {R: 0x22, G: 0x55, B: 0x44, A: 0xff}, // green — creation
		ModeAnimation: {R: 0x22, G: 0x55, B: 0x44, A: 0xff}, // green — creation
		ModeNotes:     {R: 0x22, G: 0x55, B: 0x44, A: 0xff}, // green — creation
		ModeSession:   {R: 0x22, G: 0x44, B: 0x66, A: 0xff}, // blue — organization
		ModeMyFiles:   {R: 0x22, G: 0x44, B: 0x66, A: 0xff}, // blue — organization
		ModeTraining:  {R: 0x22, G: 0x44, B: 0x66, A: 0xff}, // blue — organization
		ModeTeam:      {R: 0x22, G: 0x44, B: 0x66, A: 0xff}, // blue — organization
		ModeMatch:     {R: 0x66, G: 0x22, B: 0x22, A: 0xff}, // red — live match
	}

	// Update mode icon + color when switching.
	origSwitchMode := switchMode
	switchMode = func(mode EditorMode) {
		origSwitchMode(mode)
		if res, ok := modeIcons[mode]; ok {
			modeIcon.Resource = res
			modeIcon.Refresh()
		}
		if c, ok := modeColors[mode]; ok {
			modeBg.FillColor = c
			modeBg.Refresh()
		}
	}
	a.modeSwitchFunc = switchMode

	modeBtn := newTabTappable(
		container.NewHBox(modeIcon, container.NewPadded(a.modeLabel), modeChevron),
		func() {
			editionItem := fyne.NewMenuItem(i18n.T(i18n.KeyModeEdition), func() { switchMode(ModeEdition) })
			editionItem.Icon = fynetheme.DocumentCreateIcon()
			animItem := fyne.NewMenuItem(i18n.T(i18n.KeyModeAnimation), func() { switchMode(ModeAnimation) })
			animItem.Icon = fynetheme.MediaPlayIcon()
			notesItem := fyne.NewMenuItem(i18n.T(i18n.KeyModeNotes), func() { switchMode(ModeNotes) })
			notesItem.Icon = fynetheme.DocumentIcon()
			sessionItem := fyne.NewMenuItem(i18n.T(i18n.KeyModeSession), func() { switchMode(ModeSession) })
			sessionItem.Icon = fynetheme.FolderIcon()
			myFilesItem := fyne.NewMenuItem(i18n.T(i18n.KeyModeMyfiles), func() { switchMode(ModeMyFiles) })
			myFilesItem.Icon = fynetheme.StorageIcon()
			trainingItem := fyne.NewMenuItem(i18n.T(i18n.KeyModeTraining), func() { switchMode(ModeTraining) })
			trainingItem.Icon = fynetheme.MediaPlayIcon()
			teamItem := fyne.NewMenuItem(i18n.T(i18n.KeyModeTeam), func() { switchMode(ModeTeam) })
			teamItem.Icon = fynetheme.AccountIcon()
			matchItem := fyne.NewMenuItem(i18n.T(i18n.KeyModeMatch), func() { switchMode(ModeMatch) })
			matchItem.Icon = fynetheme.MediaRecordIcon()

			menu := widget.NewPopUpMenu(fyne.NewMenu("",
				editionItem, animItem, notesItem,
				fyne.NewMenuItemSeparator(),
				sessionItem, trainingItem, myFilesItem, teamItem, matchItem,
			), a.window.Canvas())
			pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(a.modeLabel)
			menu.ShowAtPosition(fyne.NewPos(pos.X, pos.Y+a.modeLabel.Size().Height+4))
		},
	)
	modeSelector := container.NewStack(modeBg, container.NewPadded(modeBtn))

	// ── Menu "more" (⋯): new, open, recent, import, save as, about ──
	a.moreBtn = NewTipButton(icon.DragHandle(), i18n.T(i18n.KeyTooltipMore), nil)
	a.moreBtn.onTapped = func() {
		var items []*fyne.MenuItem

		aboutItem := fyne.NewMenuItem(i18n.T(i18n.KeyTooltipAbout), func() { a.handleFileAction(FileActionAbout) })
		aboutItem.Icon = fynetheme.InfoIcon()

		switch a.editorMode {
		case ModeEdition, ModeAnimation, ModeNotes:
			// Exercise file operations.
			newItem := fyne.NewMenuItem(i18n.T(i18n.KeyTooltipNew), func() { a.handleFileAction(FileActionNew) })
			newItem.Icon = fynetheme.DocumentCreateIcon()
			openItem := fyne.NewMenuItem(i18n.T(i18n.KeyTooltipOpen), func() { a.handleFileAction(FileActionOpen) })
			openItem.Icon = fynetheme.FolderOpenIcon()
			recentItem := fyne.NewMenuItem(i18n.T(i18n.KeyTooltipRecent), func() { a.handleFileAction(FileActionRecent) })
			recentItem.Icon = fynetheme.HistoryIcon()
			importItem := fyne.NewMenuItem(i18n.T(i18n.KeyTooltipImport), func() { a.handleFileAction(FileActionImport) })
			importItem.Icon = fynetheme.DownloadIcon()
			saveAsItem := fyne.NewMenuItem(i18n.T(i18n.KeyTooltipSaveAs), func() { a.handleFileAction(FileActionSaveAs) })
			saveAsItem.Icon = fynetheme.DocumentSaveIcon()
			items = append(items, newItem, openItem, recentItem, importItem,
				fyne.NewMenuItemSeparator(), saveAsItem)

		case ModeSession:
			// Session file operations (labels are generic — context is given by the mode).
			newItem := fyne.NewMenuItem(i18n.T(i18n.KeyTooltipNew), func() { a.handleSessionAction(SessionTabEvent{Action: SessionTabActionNew}) })
			newItem.Icon = fynetheme.DocumentCreateIcon()
			openItem := fyne.NewMenuItem(i18n.T(i18n.KeyTooltipOpen), func() { a.handleSessionAction(SessionTabEvent{Action: SessionTabActionOpen}) })
			openItem.Icon = fynetheme.FolderOpenIcon()
			recentItem := fyne.NewMenuItem(i18n.T(i18n.KeyTooltipRecent), func() { a.handleSessionAction(SessionTabEvent{Action: SessionTabActionRecent}) })
			recentItem.Icon = fynetheme.HistoryIcon()
			saveItem := fyne.NewMenuItem(i18n.T(i18n.KeyTooltipSave), func() { a.handleSessionAction(SessionTabEvent{Action: SessionTabActionSave}) })
			saveItem.Icon = fynetheme.DocumentSaveIcon()
			pdfItem := fyne.NewMenuItem(i18n.T(i18n.KeyTooltipPdf), func() { a.handleSessionAction(SessionTabEvent{Action: SessionTabActionGenerate}) })
			pdfItem.Icon = fynetheme.DocumentPrintIcon()
			items = append(items, newItem, openItem, recentItem,
				fyne.NewMenuItemSeparator(), saveItem, pdfItem)
		}

		items = append(items, fyne.NewMenuItemSeparator(), aboutItem)

		menu := widget.NewPopUpMenu(fyne.NewMenu("", items...), a.window.Canvas())
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(a.moreBtn)
		menu.ShowAtPosition(fyne.NewPos(pos.X, pos.Y+a.moreBtn.Size().Height+4))
	}

	// ── Top bar ──
	topBarBg := canvas.NewRectangle(color.NRGBA{R: 0x1a, G: 0x1a, B: 0x1a, A: 0xff})
	topBarH := float32(48)
	if isMobile {
		topBarH = 72
	}
	topBarBg.SetMinSize(fyne.NewSize(0, topBarH))
	topBarContent := container.NewHBox(
		modeSelector,
		layout.NewSpacer(),
		a.fileToolbar.Btn(FileActionSave),
		a.moreBtn,
		a.langBtn,
		a.fileToolbar.Btn(FileActionPreferences),
	)
	topBar := container.NewStack(topBarBg, container.NewPadded(topBarContent))

	return container.NewBorder(topBar, nil, nil, nil, mainStack)
}

func (a *App) switchLang(lang string) {
	i18n.SetLang(i18n.Lang(lang))
	// Refresh translatable text in all panels BEFORE refreshEditor,
	// so Select options are updated before Update() tries to SetSelected.
	a.fileToolbar.RefreshLanguage()
	a.toolPalette.RefreshLanguage()
	a.propsPanel.RefreshLanguage()
	a.animControls.RefreshLanguage()
	a.instrPanel.RefreshLanguage()
	a.sessionTab.RefreshLanguage()
	a.myFilesTab.RefreshLanguage()
	a.teamTab.RefreshLanguage()
	a.matchTab.RefreshLanguage()
	// Update shelf labels before refreshEditor so props rebuild picks up new language.
	if a.editorShelf != nil {
		a.editorShelf.RefreshLanguage()
	}
	// Now refresh editor (calls propsPanel.Update which uses new options).
	a.propsPanel.SyncFromExercise()
	a.instrPanel.ForceResync()
	a.refreshEditor()
	a.updateLangBtnStyles()
	if a.moreBtn != nil {
		a.moreBtn.SetTooltip(i18n.T(i18n.KeyTooltipMore))
	}
	if a.langBtn != nil {
		a.langBtn.SetTooltip(i18n.T(i18n.KeyLangTooltip))
	}
	// Refresh empty state texts.
	if a.emptyTitle != nil {
		a.emptyTitle.Text = i18n.T(i18n.KeyEmptyStateTitle)
		a.emptyTitle.Refresh()
		a.emptySub.Text = i18n.T(i18n.KeyEmptyStateSubtitle)
		a.emptySub.Refresh()
		a.emptyNewBtn.SetText(i18n.T(i18n.KeyTooltipNew))
		a.refreshEmptyRecent()
	}
	// Refresh mode label text for current mode.
	if a.modeLabel != nil {
		modeKeys := map[EditorMode]string{
			ModeEdition:   "mode.edition",
			ModeAnimation: "mode.animation",
			ModeNotes:     "mode.notes",
			ModeSession:   "mode.session",
			ModeMyFiles:   "mode.myfiles",
			ModeTraining:  "mode.training",
			ModeTeam:      "mode.team",
			ModeMatch:     "mode.match",
		}
		if key, ok := modeKeys[a.editorMode]; ok {
			a.modeLabel.Text = i18n.T(key)
			a.modeLabel.Refresh()
		}
	}
	// Rebuild notes view if currently in Notes mode (instructions are language-dependent).
	// Rebuild dynamic mode views if currently active.
	if a.modeSwitchFunc != nil {
		switch a.editorMode {
		case ModeNotes:
			a.modeSwitchFunc(ModeNotes)
		case ModeTraining:
			a.modeSwitchFunc(ModeTraining)
		case ModeSession:
			a.modeSwitchFunc(ModeSession)
		case ModeMyFiles:
			a.modeSwitchFunc(ModeMyFiles)
		case ModeTeam:
			a.modeSwitchFunc(ModeTeam)
		case ModeMatch:
			a.modeSwitchFunc(ModeMatch)
		}
	}
	a.statusBar.SetStatus("", 0)
	a.sessionNeedsRefresh = true
	a.refreshSessionTab()
	// Persist language choice.
	if ys, ok := a.store.(*store.YAMLStore); ok {
		settings, _ := ys.LoadSettings()
		settings.Language = lang
		_ = ys.SaveSettings(settings)
	}
}

func (a *App) updateLangBtnStyles() {
	if a.langBtn == nil {
		return
	}
	if a.editLang == "fr" {
		a.langBtn.Icon = icon.FlagFR
	} else {
		a.langBtn.Icon = icon.FlagEN
	}
	a.langBtn.Refresh()
}

// checkExerciseValidation shows a status bar warning if the exercise has validation issues.
func (a *App) checkExerciseValidation() {
	if a.exercise == nil {
		return
	}
	errors, warnings := 0, 0
	for _, seq := range a.exercise.Sequences {
		issues := model.ValidateActions(&seq)
		for _, issueList := range issues {
			for _, issue := range issueList {
				if issue.IsError {
					errors++
				} else {
					warnings++
				}
			}
		}
	}
	if errors > 0 {
		a.statusBar.SetStatus(i18n.Tf(i18n.KeyValidationErrors, errors), StatusError)
	} else if warnings > 0 {
		a.statusBar.SetStatus(i18n.Tf(i18n.KeyValidationWarnings, warnings), StatusWarning)
	}
}

// refreshEditor updates all editor panels to reflect current state.
func (a *App) refreshEditor() {
	if a.exercise == nil {
		return
	}
	seqIdx := a.court.SeqIndex()
	a.seqTimeline.Update(a.exercise, seqIdx, a.editLang)
	a.propsPanel.Update(a.exercise, &a.editorState, seqIdx, a.editLang)
	a.instrPanel.Update(a.exercise, &a.editorState, seqIdx, a.editLang)
	a.updateWindowTitle()

	if a.playback != nil {
		a.animControls.SetPlayback(a.playback, len(a.exercise.Sequences))
	}

	// Update shelf element props and action timeline.
	a.editorShelf.UpdateElementProps(a.exercise, &a.editorState, seqIdx)
	a.actionTimeline.Update(a.exercise, &a.editorState, seqIdx)

	// Forward editor state status to the status bar.
	if a.editorState.StatusMsg != "" {
		a.statusBar.SetStatus(a.editorState.StatusMsg, a.editorState.StatusLevel)
		a.editorState.StatusMsg = ""
	}

	// Update undo/redo button state.
	a.seqTimeline.UpdateUndoRedo(a.history.CanUndo(), a.history.CanRedo())
}

// pushSnapshot records the current exercise state in the undo history.
// Guarded by undoRedoing to prevent recursion during Undo/Redo.
func (a *App) pushSnapshot() {
	if a.undoRedoing || a.exercise == nil {
		return
	}
	if !a.editorState.JustMutated {
		return
	}
	a.editorState.JustMutated = false
	a.history.SaveState(a.exercise)
	a.seqTimeline.UpdateUndoRedo(a.history.CanUndo(), a.history.CanRedo())
}

// Undo restores the exercise to the previous state in the history.
func (a *App) Undo() {
	if a.exercise == nil {
		return
	}
	ex := a.history.Undo(a.exercise)
	if ex == nil {
		return
	}
	a.restoreFromHistory(ex)
}

// Redo restores the state that was undone.
func (a *App) Redo() {
	if a.exercise == nil {
		return
	}
	ex := a.history.Redo(a.exercise)
	if ex == nil {
		return
	}
	a.restoreFromHistory(ex)
}

// restoreFromHistory replaces the current exercise with a history snapshot.
func (a *App) restoreFromHistory(ex *model.Exercise) {
	a.undoRedoing = true
	defer func() {
		a.undoRedoing = false
		a.editorState.JustMutated = false // prevent phantom snapshot after undo
	}()

	a.exercise = ex
	seqIdx := a.court.SeqIndex()
	if seqIdx >= len(ex.Sequences) {
		seqIdx = len(ex.Sequences) - 1
	}
	if seqIdx < 0 {
		seqIdx = 0
	}
	a.court.SetExercise(ex)
	a.court.SetSequence(seqIdx)
	a.editorState.Deselect()
	a.editorState.Modified = true
	a.refreshEditor()
}

func (a *App) syncAnimState() {
	if a.playback == nil {
		return
	}
	state := a.playback.State()

	// Toggle animation mode on court widget.
	a.court.SetAnimMode(state == anim.StatePlaying)

	if state != anim.StatePlaying {
		// Sync court seq from playback.
		a.court.SetSequence(a.playback.SeqIndex())
		a.refreshEditor()
	}

	a.animControls.Refresh()
}

func (a *App) showPreferences() {
	ys, _ := a.store.(*store.YAMLStore)
	showPrefsDialog(a.window, a.settings, ys, func(langChanged bool) {
		if langChanged {
			a.switchLang(a.settings.Language)
		}
		// Apply apron visibility change.
		a.court.ShowApron = a.settings.ApronVisible()
		a.court.InvalidateBackground()
		a.court.Refresh()
		a.viewTools.SetApronVisible(a.court.ShowApron)
	})
}

func (a *App) showAbout() {
	showAboutDialog(a.window, a.appVersion())
}

// Cleanup stops all background goroutines (timers, animations) before app exit.
func (a *App) Cleanup() {
	if a.playback != nil {
		a.playback.Stop()
	}
	if a.trainingMode != nil {
		a.trainingMode.Stop()
	}
}

func (a *App) CheckVersionAtStartup() {
	version := a.appVersion()
	if version == "dev" || version == "" {
		return
	}
	token := a.settings.GithubToken
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	go func() {
		info, err := store.CheckLatestVersion(context.Background(), token)
		if err != nil || info == nil {
			return
		}
		if info.Tag == "" || info.Tag == "v"+version || info.Tag <= "v"+version {
			return
		}
		fyne.Do(func() {
			tag := info.Tag
			url := info.URL

			// Always show update button in status bar.
			label := fmt.Sprintf("%s %s", i18n.T(i18n.KeyUpdateAvailable), tag)
			a.statusBar.ShowUpdateAvailable(label, func() {
				showUpdateDialog(a.window, tag, url)
			})

			// Only show the popup dialog if not already dismissed for this version.
			if a.settings.DismissedVersion != tag {
				showUpdateDialog(a.window, tag, url)
				a.settings.DismissedVersion = tag
				if ys, ok := a.store.(*store.YAMLStore); ok {
					_ = ys.SaveSettings(a.settings)
				}
			}
		})
	}()
}

// appVersion returns the application version from Fyne metadata.
func (a *App) appVersion() string {
	return fyne.CurrentApp().Metadata().Version
}

// --- Exercise management ---

// SetExercise sets the current exercise.
func (a *App) SetExercise(ex *model.Exercise) {
	a.exercise = ex
	a.court.SetExercise(ex)
	a.editorState.Deselect()
	a.propsPanel.SyncFromExercise()
	if a.emptyState != nil {
		if ex != nil {
			a.emptyState.Hide()
			a.seqTimeline.Widget().Show()
		} else {
			a.refreshEmptyRecent()
			a.emptyState.Show()
			a.seqTimeline.Widget().Hide()
		}
	}
	if ex != nil {
		a.playback = anim.NewPlayback(ex)
	} else {
		a.playback = nil
	}
	a.court.SetPlayback(a.playback)
	// Reset undo/redo history and save initial state.
	a.history.Clear()
	if ex != nil {
		a.history.SaveState(ex)
	}
	// Snapshot SHA before refreshEditor to avoid false mismatch during Update callbacks.
	a.snapshotExerciseSHA()
	a.refreshEditor()
	// Re-snapshot after refreshEditor — Update() callbacks may have normalized fields.
	a.snapshotExerciseSHA()
	// Rebuild notes view if currently visible.
	if a.editorMode == ModeNotes && a.modeSwitchFunc != nil {
		a.modeSwitchFunc(ModeNotes)
	}
}

func (a *App) updateWindowTitle() {
	title := i18n.T(i18n.KeyAppTitle)
	switch a.editorMode {
	case ModeEdition, ModeAnimation, ModeNotes:
		if a.exercise != nil && a.exercise.Name != "" {
			displayName := a.exercise.Localized(a.editLang).Name
			fileName := store.ToKebab(a.exercise.Name) + ".yaml"
			if displayName != a.exercise.Name && displayName != "" {
				title += " — " + displayName + " [" + fileName + "]"
			} else {
				title += " — " + fileName
			}
			if a.editorState.Modified {
				title += " *"
			}
		}
	case ModeSession:
		if s := a.sessionTab.Session(); s != nil && s.Title != "" {
			title += " — " + s.Title
			if a.sessionTab.IsModified() {
				title += " *"
			}
		}
	}
	a.window.SetTitle(title)
	// Update save button indicator.
	a.fileToolbar.SetModified(a.isCurrentModeModified())
}

// isCurrentModeModified returns whether the current mode has unsaved changes.
func (a *App) isCurrentModeModified() bool {
	switch a.editorMode {
	case ModeEdition, ModeAnimation, ModeNotes:
		return a.isExerciseModified()
	case ModeSession:
		return a.sessionTab.IsModified()
	}
	return false
}

// exerciseChecksum computes a short hash of the exercise serialized as YAML.
func exerciseChecksum(ex *model.Exercise) string {
	if ex == nil {
		return ""
	}
	data, err := yaml.Marshal(ex)
	if err != nil {
		return ""
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:8])
}

// snapshotExerciseSHA stores the current exercise checksum (call after load/save/new).
func (a *App) snapshotExerciseSHA() {
	a.exerciseSHA = exerciseChecksum(a.exercise)
	if a.exercise != nil {
		data, _ := yaml.Marshal(a.exercise)
		_ = os.WriteFile("/tmp/courtdraw_snapshot.yaml", data, 0644) //nolint:gosec // debug snapshot
	}
}

// isExerciseModified returns true if the exercise has changed since last snapshot.
// TODO: SHA-based detection disabled — needs fix for false positives from programmatic SetText.
func (a *App) isExerciseModified() bool {
	return false
}

// NewExercise creates a blank exercise.
func (a *App) NewExercise() {
	// Always store English as the primary language, with translations in i18n.
	enName := i18n.TLang("en", "default.exercise_name")
	enSeqLabel := i18n.TLang("en", "default.sequence_label")
	courtStandard := model.FIBA
	if a.settings != nil && a.settings.DefaultCourtStandard == "nba" {
		courtStandard = model.NBA
	}
	courtType := model.HalfCourt
	if a.settings != nil && a.settings.DefaultCourtType == "full_court" {
		courtType = model.FullCourt
	}
	orientation := model.OrientationLandscape
	if isMobile {
		orientation = model.OrientationPortrait
	}
	if a.settings != nil && a.settings.DefaultOrientation != "" {
		orientation = model.Orientation(a.settings.DefaultOrientation)
	}
	ex := &model.Exercise{
		Name:          enName,
		CourtType:     courtType,
		CourtStandard: courtStandard,
		Orientation:   orientation,
		Sequences: []model.Sequence{
			{Label: enSeqLabel},
		},
	}
	// Add translation for non-English languages.
	if a.editLang != "" && a.editLang != "en" {
		tr := ex.EnsureI18n(a.editLang)
		tr.Name = i18n.TLang(a.editLang, "default.exercise_name")
		tr.Sequences = []model.SequenceI18n{
			{Label: i18n.TLang(a.editLang, "default.sequence_label")},
		}
		ex.SetI18n(a.editLang, tr)
	}
	a.SetExercise(ex)
	a.statusBar.SetStatus(i18n.T(i18n.KeyStatusNewExercise), 0)
}

// NewSession creates a blank session.
func (a *App) NewSession() {
	s := &model.Session{
		Title: i18n.T(i18n.KeyDefaultSessionName),
		Date:  time.Now().Format("2006-01-02"),
	}
	a.sessionTab.SetSession(s)
	a.statusBar.SetStatus(i18n.T(i18n.KeyStatusNewSession), 0)
}

func (a *App) addSequence() {
	if a.exercise == nil {
		return
	}
	a.pushSnapshot()
	var newSeq model.Sequence
	currentIdx := a.court.SeqIndex()
	if currentIdx < len(a.exercise.Sequences) {
		current := &a.exercise.Sequences[currentIdx]
		newSeq.Players = make([]model.Player, len(current.Players))
		copy(newSeq.Players, current.Players)
		newSeq.Accessories = make([]model.Accessory, len(current.Accessories))
		copy(newSeq.Accessories, current.Accessories)
		newSeq.BallCarrier = append(model.BallCarriers{}, current.BallCarrier...)
		for _, act := range current.Actions {
			if act.Type == model.ActionPass && act.To.IsPlayer {
				newSeq.BallCarrier.RemoveBall(act.From.PlayerID)
				newSeq.BallCarrier.AddBall(act.To.PlayerID)
			}
		}
	}
	a.exercise.Sequences = append(a.exercise.Sequences, newSeq)
	newIdx := len(a.exercise.Sequences) - 1
	a.court.SetSequence(newIdx)
	a.editorState.Deselect()
	a.editorState.MarkModified()
	a.statusBar.SetStatus(i18n.Tf(i18n.KeyStatusSeqAdded, newIdx+1), 0)
	a.refreshEditor()
}

func (a *App) deleteSequence(idx int) {
	if a.exercise == nil || len(a.exercise.Sequences) <= 1 {
		return
	}
	if idx < 0 || idx >= len(a.exercise.Sequences) {
		return
	}
	a.pushSnapshot()
	a.exercise.Sequences = append(a.exercise.Sequences[:idx], a.exercise.Sequences[idx+1:]...)
	// Adjust current index.
	newIdx := idx
	if newIdx >= len(a.exercise.Sequences) {
		newIdx = len(a.exercise.Sequences) - 1
	}
	a.court.SetSequence(newIdx)
	a.editorState.Deselect()
	a.editorState.MarkModified()
	a.statusBar.SetStatus(i18n.Tf(i18n.KeyStatusSeqDeleted, idx+1), 0)
	a.refreshEditor()
}

// --- File operations ---

func (a *App) handleFileAction(action FileAction) {
	switch action {
	case FileActionNew:
		a.NewExercise()
	case FileActionOpen:
		a.showOpenDialog()
	case FileActionSave:
		switch a.editorMode {
		case ModeTeam:
			if a.teamTab != nil {
				a.teamTab.Save()
			}
		default:
			a.saveExercise()
		}
	case FileActionSaveAs:
		a.saveAsExercise()
	case FileActionImport:
		a.showImportDialog()
	case FileActionRecent:
		a.showRecentFiles()
	case FileActionPreferences:
		a.showPreferences()
	case FileActionAbout:
		a.showAbout()
	}
}

func (a *App) loadExerciseAny(name string) (*model.Exercise, error) {
	ex, err := a.store.LoadExercise(name)
	if err == nil {
		return ex, nil
	}
	if a.library != nil {
		return a.library.LoadExercise(name)
	}
	return nil, err
}

func (a *App) openExercise(name string) {
	ex, err := a.loadExerciseAny(name)
	if err != nil {
		log.Printf("load exercise %s: %v", name, err)
		return
	}
	a.SetExercise(ex)
	a.recordRecentFile(name)
	a.statusBar.SetStatus(i18n.Tf(i18n.KeyStatusOpened, name), 0)
}

func (a *App) saveExercise() {
	if a.exercise == nil {
		return
	}
	if err := a.store.SaveExercise(a.exercise); err != nil {
		log.Printf("save exercise: %v", err)
		a.statusBar.SetStatus(i18n.T(i18n.KeyStatusSaveError), 1)
		return
	}
	a.snapshotExerciseSHA()
	a.recordRecentFile(store.ToKebab(a.exercise.Name))
	a.sessionNeedsRefresh = true
	a.refreshSessionTab()
	fileName := store.ToKebab(a.exercise.Name) + ".yaml"
	if ys, ok := a.store.(*store.YAMLStore); ok {
		fileName = filepath.Join(ys.ExercisesDir(), fileName)
	}
	a.statusBar.SetStatus(i18n.Tf(i18n.KeyStatusSaved, fileName), StatusSuccess)
	a.updateWindowTitle()
}

func (a *App) saveAsExercise() {
	if a.exercise == nil {
		return
	}

	entry := widget.NewEntry()
	entry.SetPlaceHolder(i18n.T(i18n.KeySaveAsPlaceholder))
	entry.SetText(a.exercise.Localized(a.editLang).Name)

	d := dialog.NewForm(i18n.T(i18n.KeySaveAsTitle), i18n.T(i18n.KeyPrefsSave), i18n.T(i18n.KeyDialogCancel),
		[]*widget.FormItem{widget.NewFormItem(i18n.T(i18n.KeySaveAsNameLabel), entry)},
		func(ok bool) {
			if !ok {
				return
			}
			newName := strings.TrimSpace(entry.Text)
			if newName == "" {
				return
			}

			// Check if file already exists.
			kebab := store.ToKebab(newName)
			if ys, ok := a.store.(*store.YAMLStore); ok {
				path := filepath.Join(ys.ExercisesDir(), kebab+".yaml")
				if _, err := os.Stat(path); err == nil {
					dialog.ShowConfirm(i18n.T(i18n.KeySaveAsOverwriteTitle),
						i18n.Tf(i18n.KeySaveAsOverwriteMsg, kebab+".yaml"),
						func(overwrite bool) {
							if overwrite {
								a.doSaveAs(newName)
							}
						}, a.window)
					return
				}
			}
			a.doSaveAs(newName)
		}, a.window)
	d.Resize(fyne.NewSize(400, 150))
	d.Show()
}

func (a *App) doSaveAs(newName string) {
	a.exercise.Name = newName
	if err := a.store.SaveExercise(a.exercise); err != nil {
		log.Printf("save as: %v", err)
		a.statusBar.SetStatus(i18n.T(i18n.KeyStatusSaveError), 0)
		return
	}
	a.editorState.ClearModified()
	fileName := store.ToKebab(newName) + ".yaml"
	if ys, ok := a.store.(*store.YAMLStore); ok {
		fileName = filepath.Join(ys.ExercisesDir(), fileName)
	}
	a.statusBar.SetStatus(i18n.Tf(i18n.KeyStatusSaved, fileName), StatusSuccess)
	a.updateWindowTitle()
	a.refreshEditor()
}

func (a *App) showOpenDialog() {
	names, err := a.store.ListExercises()
	if err != nil {
		log.Printf("list exercises: %v", err)
		return
	}
	items := make([]string, len(names))
	copy(items, names)
	list := widget.NewList(
		func() int { return len(items) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if lbl, ok := obj.(*widget.Label); ok {
				lbl.SetText(items[id])
			}
		},
	)
	var d dialog.Dialog
	list.OnSelected = func(id widget.ListItemID) {
		if id < len(items) {
			a.openExercise(items[id])
			d.Hide()
		}
	}
	d = dialog.NewCustom(i18n.T(i18n.KeyTooltipOpen), i18n.T(i18n.KeyDialogCancel), list, a.window)
	d.Resize(fyne.NewSize(400, 500))
	d.Show()
}

func (a *App) showImportDialog() {
	if a.library == nil {
		log.Printf("no library directory configured")
		return
	}
	names, err := a.library.ListExercises()
	if err != nil {
		log.Printf("list library exercises: %v", err)
		return
	}
	items := make([]string, len(names))
	copy(items, names)
	list := widget.NewList(
		func() int { return len(items) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if lbl, ok := obj.(*widget.Label); ok {
				lbl.SetText(items[id])
			}
		},
	)
	var d dialog.Dialog
	list.OnSelected = func(id widget.ListItemID) {
		if id < len(items) {
			a.importExercise(items[id])
			d.Hide()
		}
	}
	d = dialog.NewCustom(i18n.T(i18n.KeyTooltipImport), i18n.T(i18n.KeyDialogCancel), list, a.window)
	d.Resize(fyne.NewSize(400, 500))
	d.Show()
}

func (a *App) importExercise(name string) {
	if a.library == nil {
		return
	}
	ex, err := a.library.LoadExercise(name)
	if err != nil {
		log.Printf("load library exercise %s: %v", name, err)
		return
	}
	if err := a.store.SaveExercise(ex); err != nil {
		log.Printf("save imported exercise: %v", err)
		return
	}
	a.SetExercise(ex)
	a.sessionNeedsRefresh = true
	a.refreshSessionTab()
	a.recordRecentFile(store.ToKebab(ex.Name))
	a.statusBar.SetStatus(i18n.Tf(i18n.KeyStatusImported, ex.Name), StatusSuccess)
}

func (a *App) recordRecentFile(name string) {
	if ys, ok := a.store.(*store.YAMLStore); ok {
		ys.RecordRecentFile(name)
	}
}

// buildNotesView creates the full-page notes/instructions view.
// Each sequence is shown with a small court diagram and a text entry for instructions.
func (a *App) buildNotesView() fyne.CanvasObject {
	if a.exercise == nil {
		placeholder := canvas.NewText(i18n.T(i18n.KeyModeNotesPlaceholder), color.NRGBA{R: 0xaa, G: 0xaa, B: 0xaa, A: 0xff})
		placeholder.Alignment = fyne.TextAlignCenter
		return container.NewCenter(placeholder)
	}

	vbox := container.NewVBox()

	for i, seq := range a.exercise.Sequences {
		seqIdx := i

		// Sequence label.
		label := seq.Label
		if a.editLang != "" && a.editLang != "en" && a.exercise.I18n != nil {
			if tr, ok := a.exercise.I18n[a.editLang]; ok && i < len(tr.Sequences) && tr.Sequences[i].Label != "" {
				label = tr.Sequences[i].Label
			}
		}
		if label == "" {
			label = i18n.Tf(i18n.KeySeqFormat, i+1)
		}
		seqLabel := canvas.NewText(label, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
		seqLabel.TextSize = 16
		seqLabel.TextStyle.Bold = true

		// Small court diagram (static render of this sequence).
		courtPreview := fynecourt.NewCourtWidget()
		courtPreview.SetExercise(a.exercise)
		courtPreview.SetSequence(seqIdx)

		// Instructions text entry — resolve current instructions for the edit language.
		instrText := strings.Join(seq.Instructions, "\n")
		if a.editLang != "" && a.editLang != "en" && a.exercise.I18n != nil {
			if tr, ok := a.exercise.I18n[a.editLang]; ok && i < len(tr.Sequences) && len(tr.Sequences[i].Instructions) > 0 {
				instrText = strings.Join(tr.Sequences[i].Instructions, "\n")
			}
		}

		instrEntry := widget.NewMultiLineEntry()
		instrEntry.Wrapping = fyne.TextWrapWord
		instrEntry.SetPlaceHolder(i18n.T(i18n.KeyInstrPlaceholder))
		instrEntry.SetText(instrText)
		instrEntry.SetMinRowsVisible(3)
		instrEntry.OnChanged = func(text string) {
			lines := splitInstructions(text)
			if a.editLang != "" && a.editLang != "en" {
				tr := a.exercise.EnsureI18n(a.editLang)
				for len(tr.Sequences) <= seqIdx {
					tr.Sequences = append(tr.Sequences, model.SequenceI18n{})
				}
				tr.Sequences[seqIdx].Instructions = lines
				a.exercise.SetI18n(a.editLang, tr)
			} else {
				a.exercise.Sequences[seqIdx].Instructions = lines
			}
			a.editorState.Modified = true
		}

		// Layout: label on top, then court + instructions side by side.
		seqRow := container.NewBorder(nil, nil, courtPreview, nil, instrEntry)
		sep := widget.NewSeparator()
		vbox.Add(container.NewVBox(container.NewPadded(seqLabel), container.NewPadded(seqRow), sep))
	}

	return container.NewVScroll(vbox)
}

func splitInstructions(text string) []string {
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	var result []string
	for _, l := range lines {
		if l != "" {
			result = append(result, l)
		}
	}
	return result
}

func (a *App) showExerciseSettingsDialog() {
	if a.exercise == nil {
		return
	}
	ex := a.exercise

	// Resolve translated name/description if editing in non-English.
	name := ex.Name
	desc := ex.Description
	if a.editLang != "" && a.editLang != "en" && ex.I18n != nil {
		if tr, ok := ex.I18n[a.editLang]; ok {
			if tr.Name != "" {
				name = tr.Name
			}
			if tr.Description != "" {
				desc = tr.Description
			}
		}
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetText(name)

	descEntry := widget.NewMultiLineEntry()
	descEntry.SetText(desc)
	descEntry.SetMinRowsVisible(2)

	courtStdOptions := []string{"FIBA", "NBA"}
	courtStdSelect := widget.NewSelect(courtStdOptions, nil)
	if ex.CourtStandard == model.NBA {
		courtStdSelect.SetSelected("NBA")
	} else {
		courtStdSelect.SetSelected("FIBA")
	}

	courtTypeOptions := []string{i18n.T(i18n.KeyCourtHalf), i18n.T(i18n.KeyCourtFull)}
	courtTypeSelect := widget.NewSelect(courtTypeOptions, nil)
	if ex.CourtType == model.FullCourt {
		courtTypeSelect.SetSelected(courtTypeOptions[1])
	} else {
		courtTypeSelect.SetSelected(courtTypeOptions[0])
	}

	orientLabels := []string{
		i18n.T(i18n.KeyPropsOrientPortrait),
		i18n.T(i18n.KeyPropsOrientLandscape),
		i18n.T(i18n.KeyPropsOrientPortraitFlip),
		i18n.T(i18n.KeyPropsOrientLandscapeFlip),
	}
	orientKeys := []model.Orientation{
		model.OrientationPortrait, model.OrientationLandscape,
		model.OrientationPortraitFlip, model.OrientationLandscapeFlip,
	}
	orientSelect := widget.NewSelect(orientLabels, nil)
	for idx, k := range orientKeys {
		if k == ex.Orientation {
			orientSelect.SetSelected(orientLabels[idx])
			break
		}
	}
	if orientSelect.Selected == "" {
		orientSelect.SetSelected(orientLabels[0])
	}

	durationEntry := widget.NewEntry()
	durationEntry.SetText(ex.Duration)

	tagsEntry := widget.NewEntry()
	tagsEntry.SetPlaceHolder("tag1, tag2, ...")
	if len(ex.Tags) > 0 {
		tagsEntry.SetText(joinTags(ex.Tags))
	}

	items := []*widget.FormItem{
		widget.NewFormItem(i18n.T(i18n.KeyPropsName), nameEntry),
		widget.NewFormItem(i18n.T(i18n.KeyPropsDescription), descEntry),
		widget.NewFormItem(i18n.T(i18n.KeyPropsCourtStandard), courtStdSelect),
		widget.NewFormItem(i18n.T(i18n.KeyPropsCourtType), courtTypeSelect),
		widget.NewFormItem(i18n.T(i18n.KeyPropsOrientation), orientSelect),
		widget.NewFormItem(i18n.T(i18n.KeyPropsDuration), durationEntry),
		widget.NewFormItem(i18n.T(i18n.KeyPropsTags), tagsEntry),
	}

	dlg := dialog.NewForm(i18n.T(i18n.KeySettingsExerciseTitle), i18n.T(i18n.KeySeqRenameOk), i18n.T(i18n.KeySeqRenameCancel),
		items,
		func(ok bool) {
			if !ok {
				return
			}
			a.pushSnapshot()
			if a.editLang != "" && a.editLang != "en" {
				tr := ex.EnsureI18n(a.editLang)
				tr.Name = nameEntry.Text
				tr.Description = descEntry.Text
				ex.SetI18n(a.editLang, tr)
			} else {
				ex.Name = nameEntry.Text
				ex.Description = descEntry.Text
			}
			if courtStdSelect.Selected == "NBA" {
				ex.CourtStandard = model.NBA
			} else {
				ex.CourtStandard = model.FIBA
			}
			a.applyCourtTypeSwitch(courtTypeSelect.Selected == courtTypeOptions[1])
			for idx, label := range orientLabels {
				if label == orientSelect.Selected {
					ex.Orientation = orientKeys[idx]
					break
				}
			}
			ex.Duration = strings.TrimSpace(durationEntry.Text)
			ex.Tags = splitTags(tagsEntry.Text)
			a.editorState.Modified = true
			a.refreshEditor()
			a.court.Refresh()
		},
		a.window,
	)
	dlg.Resize(fyne.NewSize(400, 400))
	dlg.Show()
}

// toggleApron toggles apron band visibility and persists the setting.
func (a *App) toggleApron() {
	a.court.ShowApron = !a.court.ShowApron
	a.court.InvalidateBackground()
	a.court.Refresh()
	a.viewTools.SetApronVisible(a.court.ShowApron)
	// Persist to settings.
	v := a.court.ShowApron
	a.settings.ShowApron = &v
	if ys, ok := a.store.(*store.YAMLStore); ok {
		_ = ys.SaveSettings(a.settings)
	}
}

// toggleOrientation rotates the court 90° clockwise.
func (a *App) toggleOrientation() {
	if a.exercise == nil {
		return
	}
	a.pushSnapshot()
	a.exercise.Orientation = model.NextRotationCW(a.exercise.Orientation)
	a.editorState.Modified = true
	a.court.Refresh()
	a.refreshEditor()
	a.updateWindowTitle()
}

// applyCourtTypeSwitch handles smart court type switching with position remapping.
// wantFull indicates whether the desired court type is full court.
func (a *App) applyCourtTypeSwitch(wantFull bool) {
	ex := a.exercise
	if ex == nil {
		return
	}
	if wantFull && ex.CourtType == model.FullCourt {
		return
	}
	if !wantFull && ex.CourtType == model.HalfCourt {
		return
	}

	if ex.CourtType == model.HalfCourt && wantFull {
		ex.RemapPositionsHalfToFull()
		ex.CourtType = model.FullCourt
		return
	}

	// Full → Half: detect which half has elements.
	half := ex.FullCourtPlayerHalf()
	if half == "mixed" {
		dialog.ShowInformation(
			i18n.T(i18n.KeyCourtSwitchBlockedTitle),
			i18n.T(i18n.KeyCourtSwitchBlockedMsg),
			a.window,
		)
		return
	}
	ex.RemapPositionsFullToHalf(half == "bottom")
	ex.CourtType = model.HalfCourt
}

func joinTags(tags []string) string {
	return strings.Join(tags, ", ")
}

func splitTags(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var tags []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// refreshEmptyRecent populates the recent exercises list on the empty state screen.
func (a *App) refreshEmptyRecent() {
	if a.emptyRecentList == nil {
		return
	}
	a.emptyRecentList.RemoveAll()
	ys, ok := a.store.(*store.YAMLStore)
	if !ok {
		return
	}
	recent := ys.RecentFiles(5)
	if len(recent) == 0 {
		return
	}
	header := canvas.NewText(i18n.T(i18n.KeyTooltipRecent), color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff})
	header.TextSize = 13
	header.Alignment = fyne.TextAlignCenter
	a.emptyRecentList.Add(header)
	for _, name := range recent {
		n := name
		btn := widget.NewButton(n, func() {
			a.openExercise(n)
		})
		a.emptyRecentList.Add(container.NewCenter(btn))
	}
}

func (a *App) showRecentFiles() {
	ys, ok := a.store.(*store.YAMLStore)
	if !ok {
		return
	}
	recent := ys.RecentFiles(10)
	if len(recent) == 0 {
		return
	}
	items := make([]string, len(recent))
	copy(items, recent)
	list := widget.NewList(
		func() int { return len(items) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if lbl, ok := obj.(*widget.Label); ok {
				lbl.SetText(items[id])
			}
		},
	)
	var d dialog.Dialog
	list.OnSelected = func(id widget.ListItemID) {
		if id < len(items) {
			a.openExercise(items[id])
			d.Hide()
		}
	}
	d = dialog.NewCustom(i18n.T(i18n.KeyTooltipRecent), i18n.T(i18n.KeyDialogCancel), list, a.window)
	d.Resize(fyne.NewSize(400, 400))
	d.Show()
}

func (a *App) deleteExercise(name string) {
	if err := a.store.DeleteExercise(name); err != nil {
		log.Printf("delete exercise %s: %v", name, err)
		return
	}
	if a.exercise != nil && store.ToKebab(a.exercise.Name) == name {
		a.SetExercise(nil)
	}
	a.sessionNeedsRefresh = true
	a.refreshSessionTab()
	a.statusBar.SetStatus(i18n.Tf(i18n.KeyStatusDeleted, name), 0)
}

// --- Session operations ---

func (a *App) handleSessionAction(ev SessionTabEvent) {
	switch ev.Action {
	case SessionTabActionNew:
		a.NewSession()
	case SessionTabActionOpen:
		a.showOpenSessionDialog()
	case SessionTabActionSave:
		a.saveSession()
	case SessionTabActionGenerate:
		a.showPdfExportDialog()
	case SessionTabActionRefresh:
		if ys, ok := a.store.(*store.YAMLStore); ok {
			if errs := ys.RebuildExerciseIndex(); len(errs) > 0 {
				a.statusBar.SetStatus(i18n.Tf(i18n.KeyStatusIndexParseErrors, len(errs), strings.Join(errs, "; ")), 1)
			}
		}
		a.sessionTab.SetExercises(a.buildManagedExercises())
		a.syncLibrary()
	case SessionTabActionOpenExercise:
		a.openExercise(ev.Name)
		if a.modeSwitchFunc != nil {
			a.modeSwitchFunc(ModeEdition)
		}
	case SessionTabActionUpdate:
		a.updateExerciseFromRemote(ev.Name)
	case SessionTabActionContribute:
		a.contributeExercise(ev.Name)
	case SessionTabActionDeleteExercise:
		a.deleteExercise(ev.Name)
	case SessionTabActionRecent:
		a.showRecentSessions()
	case SessionTabActionTraining:
		a.enterTrainingMode()
	case SessionTabActionShare:
		a.shareSession()
	case SessionTabActionImportBundle:
		a.showImportBundleDialog()
	}
}

func (a *App) refreshSessionTab() {
	if a.sessionNeedsRefresh {
		a.sessionTab.SetExercises(a.buildManagedExercises())
		a.sessionNeedsRefresh = false
	}
	// Always resolve exercises so total duration is computed.
	a.resolveSessionExercises()
}

func (a *App) resolveSessionExercises() {
	if a.sessionTab.Session() == nil {
		return
	}
	lang := string(i18n.CurrentLang())
	resolved := make(map[string]*model.Exercise)
	for _, entry := range a.sessionTab.Session().Exercises {
		if _, ok := resolved[entry.Exercise]; !ok {
			ex, err := a.loadExerciseAny(entry.Exercise)
			if err == nil {
				resolved[entry.Exercise] = ex.Localized(lang)
			}
		}
	}
	a.sessionTab.SetResolvedExercises(resolved)
}

func (a *App) showOpenSessionDialog() {
	names, err := a.store.ListSessions()
	if err != nil {
		log.Printf("list sessions: %v", err)
		return
	}
	items := make([]string, len(names))
	copy(items, names)
	list := widget.NewList(
		func() int { return len(items) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if lbl, ok := obj.(*widget.Label); ok {
				lbl.SetText(items[id])
			}
		},
	)
	var d dialog.Dialog
	list.OnSelected = func(id widget.ListItemID) {
		if id < len(items) {
			a.openSession(items[id])
			a.sessionTab.ShowSession()
			d.Hide()
		}
	}
	d = dialog.NewCustom(i18n.T(i18n.KeySessionOpen), i18n.T(i18n.KeyDialogCancel), list, a.window)
	d.Resize(fyne.NewSize(400, 500))
	d.Show()
}

func (a *App) showRecentSessions() {
	ys, ok := a.store.(*store.YAMLStore)
	if !ok {
		return
	}
	recent := ys.RecentSessions(10)
	if len(recent) == 0 {
		return
	}
	items := make([]string, len(recent))
	copy(items, recent)
	list := widget.NewList(
		func() int { return len(items) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if lbl, ok := obj.(*widget.Label); ok {
				lbl.SetText(items[id])
			}
		},
	)
	var d dialog.Dialog
	list.OnSelected = func(id widget.ListItemID) {
		if id < len(items) {
			a.openSession(items[id])
			a.sessionTab.ShowSession()
			d.Hide()
		}
	}
	d = dialog.NewCustom(i18n.T(i18n.KeySessionRecent), i18n.T(i18n.KeyDialogCancel), list, a.window)
	d.Resize(fyne.NewSize(400, 400))
	d.Show()
}

func (a *App) openSession(name string) {
	s, err := a.store.LoadSession(name)
	if err != nil {
		log.Printf("load session %s: %v", name, err)
		return
	}
	a.sessionTab.SetSession(s)
	a.resolveSessionExercises()
	if ys, ok := a.store.(*store.YAMLStore); ok {
		ys.RecordRecentSession(name)
	}
	a.statusBar.SetStatus(i18n.Tf(i18n.KeyStatusOpened, name), 0)
}

func (a *App) saveSession() {
	s := a.sessionTab.Session()
	if s == nil {
		return
	}
	if err := a.store.SaveSession(s); err != nil {
		log.Printf("save session: %v", err)
		a.statusBar.SetStatus(i18n.T(i18n.KeyStatusSaveError), 1)
		return
	}
	a.sessionTab.ClearModified()
	name := store.SessionFileName(s)
	if name != "" {
		if ys, ok := a.store.(*store.YAMLStore); ok {
			ys.RecordRecentSession(name)
		}
	}
	a.statusBar.SetStatus(i18n.Tf(i18n.KeyStatusSaved, s.Title), StatusSuccess)
}

func (a *App) deleteSession(name string) {
	if err := a.store.DeleteSession(name); err != nil {
		log.Printf("delete session %s: %v", name, err)
		return
	}
	s := a.sessionTab.Session()
	if s != nil && store.SessionFileName(s) == name {
		a.NewSession()
	}
	log.Printf("deleted session: %s", name)
}

// --- My Files tab ---

func (a *App) refreshMyFilesTab() {
	if !a.myFilesNeedsRefresh {
		return
	}
	sessions, exercises := a.buildMyFilesData()
	a.myFilesTab.SetSessions(sessions)
	a.myFilesTab.SetExercises(exercises)
	a.myFilesTab.SetTeams(a.buildMyFilesTeams())
	a.myFilesTab.SetMatches(a.buildMyFilesMatches())
	a.myFilesNeedsRefresh = false
}

func (a *App) buildMyFilesData() ([]SessionFileItem, []ExerciseFileItem) {
	// Collect all exercises referenced by any session.
	referenced := make(map[string]bool)
	sessionNames, _ := a.store.ListSessions()
	sessionItems := make([]SessionFileItem, 0, len(sessionNames))
	for _, name := range sessionNames {
		s, err := a.store.LoadSession(name)
		if err != nil {
			continue
		}
		exNames := share.CollectExerciseNames(s)
		for _, en := range exNames {
			referenced[en] = true
		}
		sessionItems = append(sessionItems, SessionFileItem{
			Name:          name,
			Title:         s.Title,
			Date:          s.Date,
			ExerciseCount: len(s.Exercises),
		})
	}

	// Build exercise list (local only) with orphan flag.
	lang := string(i18n.CurrentLang())
	localNames, _ := a.store.ListExercises()
	exerciseItems := make([]ExerciseFileItem, 0, len(localNames))
	for _, name := range localNames {
		ex, err := a.store.LoadExercise(name)
		if err != nil {
			continue
		}
		loc := ex.Localized(lang)
		cat := string(ex.Category)
		if cat != "" {
			cat = i18n.T("category." + cat)
		}
		exerciseItems = append(exerciseItems, ExerciseFileItem{
			Name:        name,
			DisplayName: loc.Name,
			Category:    cat,
			Duration:    ex.Duration,
			IsOrphan:    !referenced[name],
		})
	}

	return sessionItems, exerciseItems
}

func (a *App) buildMyFilesTeams() []TeamFileItem {
	entries, err := a.store.ListTeams()
	if err != nil {
		return nil
	}
	items := make([]TeamFileItem, 0, len(entries))
	for _, e := range entries {
		name := strings.TrimSuffix(e.File, ".yaml")
		items = append(items, TeamFileItem{
			Name:        name,
			DisplayName: e.Name,
			Club:        e.Club,
			Season:      e.Season,
			MemberCount: e.Members,
		})
	}
	return items
}

func (a *App) buildMyFilesMatches() []MatchFileItem {
	entries, err := a.store.ListMatches()
	if err != nil {
		return nil
	}
	items := make([]MatchFileItem, 0, len(entries))
	for _, e := range entries {
		name := strings.TrimSuffix(e.File, ".yaml")
		item := MatchFileItem{
			Name:     name,
			TeamName: e.TeamName,
			Opponent: e.Opponent,
			Date:     e.Date,
			Status:   e.Status,
		}
		// Load full match to get scores if finished.
		if e.Status == "finished" {
			if m, loadErr := a.store.LoadMatch(name); loadErr == nil {
				item.HomeScore = m.HomeScore
				item.AwayScore = m.AwayScore
			}
		}
		items = append(items, item)
	}
	return items
}

func (a *App) handleMyFilesAction(ev MyFilesEvent) {
	switch ev.Action {
	case MyFilesActionOpenSession:
		a.openSession(ev.Name)
		if a.modeSwitchFunc != nil {
			a.modeSwitchFunc(ModeSession)
		}
		a.sessionTab.ShowSession()
	case MyFilesActionDeleteSession:
		a.deleteSessionWithOrphanCleanup(ev.Name)
	case MyFilesActionShareSession:
		a.openSession(ev.Name)
		a.shareSession()
	case MyFilesActionImportBundle:
		a.showImportBundleDialog()
	case MyFilesActionOpenExercise:
		a.openExercise(ev.Name)
		if a.modeSwitchFunc != nil {
			a.modeSwitchFunc(ModeEdition)
		}
	case MyFilesActionContributeExercise:
		a.contributeExercise(ev.Name)
	case MyFilesActionDeleteExercise:
		dialog.ShowConfirm(
			i18n.T(i18n.KeyMyfilesDeleteExercise),
			fmt.Sprintf(i18n.T(i18n.KeyMyfilesConfirmDeleteExercise), ev.Name),
			func(ok bool) {
				if ok {
					a.deleteExercise(ev.Name)
					a.myFilesNeedsRefresh = true
					a.sessionNeedsRefresh = true
					a.refreshMyFilesTab()
				}
			}, a.window)
	case MyFilesActionShareExercise:
		ex, err := a.store.LoadExercise(ev.Name)
		if err != nil {
			a.statusBar.SetStatus(fmt.Sprintf(i18n.T(i18n.KeyShareError), err), 1)
			return
		}
		data, _ := yaml.Marshal(ex)
		a.shareGenericData(data, ev.Name)
	case MyFilesActionOpenTeam:
		if a.modeSwitchFunc != nil {
			a.modeSwitchFunc(ModeTeam)
		}
		a.teamTab.loadTeam(ev.Name + ".yaml")
	case MyFilesActionDeleteTeam:
		dialog.ShowConfirm(
			i18n.T(i18n.KeyTeamConfirmDelete),
			fmt.Sprintf(i18n.T(i18n.KeyMyfilesConfirmDeleteTeam), ev.Name),
			func(ok bool) {
				if ok {
					if err := a.store.DeleteTeam(ev.Name); err != nil {
						a.statusBar.SetStatus(fmt.Sprintf("Error: %v", err), 1)
						return
					}
					a.statusBar.SetStatus(fmt.Sprintf(i18n.T(i18n.KeyStatusTeamDeleted), ev.Name), 2)
					a.myFilesNeedsRefresh = true
					a.refreshMyFilesTab()
				}
			}, a.window)
	case MyFilesActionShareTeam:
		team, err := a.store.LoadTeam(ev.Name)
		if err != nil {
			a.statusBar.SetStatus(fmt.Sprintf(i18n.T(i18n.KeyShareError), err), 1)
			return
		}
		data, _ := yaml.Marshal(team)
		a.shareGenericData(data, ev.Name)
	case MyFilesActionOpenMatch:
		if a.modeSwitchFunc != nil {
			a.modeSwitchFunc(ModeMatch)
		}
		entries, _ := a.store.ListMatches()
		for _, e := range entries {
			name := strings.TrimSuffix(e.File, ".yaml")
			if name == ev.Name {
				a.matchTab.openMatch(e)
				break
			}
		}
	case MyFilesActionDeleteMatch:
		dialog.ShowConfirm(
			i18n.T(i18n.KeyMatchConfirmDelete),
			fmt.Sprintf(i18n.T(i18n.KeyMyfilesConfirmDeleteMatch), ev.Name),
			func(ok bool) {
				if ok {
					if err := a.store.DeleteMatch(ev.Name); err != nil {
						a.statusBar.SetStatus(fmt.Sprintf("Error: %v", err), 1)
						return
					}
					a.statusBar.SetStatus(fmt.Sprintf(i18n.T(i18n.KeyStatusMatchDeleted), ev.Name), 2)
					a.myFilesNeedsRefresh = true
					a.refreshMyFilesTab()
				}
			}, a.window)
	case MyFilesActionShareMatch:
		a.shareMatchWithTeamQR(ev.Name)
	case MyFilesActionImportExercise:
		a.showGenericImportDialog(func(data []byte) error {
			var ex model.Exercise
			if err := yaml.Unmarshal(data, &ex); err != nil {
				return err
			}
			return a.store.SaveExercise(&ex)
		})
	case MyFilesActionImportTeam:
		a.showGenericImportDialog(func(data []byte) error {
			var team model.Team
			if err := yaml.Unmarshal(data, &team); err != nil {
				return err
			}
			return a.store.SaveTeam(&team)
		})
	case MyFilesActionImportMatch:
		a.showGenericImportDialog(func(data []byte) error {
			docs := splitYAMLDocs(data)
			if len(docs) == 0 {
				return fmt.Errorf("empty file")
			}
			var match model.Match
			if err := yaml.Unmarshal(docs[0], &match); err != nil {
				return err
			}
			if len(docs) > 1 {
				var team model.Team
				if err := yaml.Unmarshal(docs[1], &team); err == nil && team.Name != "" {
					_ = a.store.SaveTeam(&team)
				}
			}
			return a.store.SaveMatch(&match)
		})
	}
}


func (a *App) shareMatchWithTeamQR(name string) {
	match, err := a.store.LoadMatch(name)
	if err != nil {
		a.statusBar.SetStatus(fmt.Sprintf(i18n.T(i18n.KeyShareError), err), 1)
		return
	}
	teamFile := strings.TrimSuffix(match.TeamFile, ".yaml")
	if teamFile == "" {
		teamFile = store.TeamFileName(&model.Team{Name: match.TeamName})
	}
	team, _ := a.store.LoadTeam(teamFile)

	matchData, _ := yaml.Marshal(match)
	var buf []byte
	buf = append(buf, matchData...)
	if team != nil {
		buf = append(buf, []byte("\n---\n")...)
		teamData, _ := yaml.Marshal(team)
		buf = append(buf, teamData...)
	}
	a.shareGenericData(buf, name)
}

func splitYAMLDocs(data []byte) [][]byte {
	parts := strings.Split(string(data), "\n---\n")
	var docs [][]byte
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			docs = append(docs, []byte(trimmed))
		}
	}
	return docs
}

func (a *App) deleteSessionWithOrphanCleanup(name string) {
	s, err := a.store.LoadSession(name)
	if err != nil {
		a.deleteSession(name)
		a.myFilesNeedsRefresh = true
		a.refreshMyFilesTab()
		return
	}

	dialog.ShowConfirm(
		i18n.T(i18n.KeyMyfilesDeleteSession),
		fmt.Sprintf(i18n.T(i18n.KeyMyfilesConfirmDeleteSession), s.Title),
		func(ok bool) {
			if !ok {
				return
			}
			a.deleteSession(name)
			a.statusBar.SetStatus(fmt.Sprintf(i18n.T(i18n.KeyStatusSessionDeleted), s.Title), 0)
			a.myFilesNeedsRefresh = true
			a.sessionNeedsRefresh = true
			a.refreshMyFilesTab()
		}, a.window)
}

func (a *App) showPdfExportDialog() {
	s := a.sessionTab.Session()
	if s == nil {
		log.Printf("no session to generate PDF for")
		return
	}

	pageLayout := pdf.LayoutPortrait
	opts := []string{
		i18n.T(i18n.KeyPdfLayoutPortrait),
		i18n.T(i18n.KeyPdfLayoutLandscape2Up),
	}
	radio := widget.NewRadioGroup(opts, func(selected string) {
		if selected == opts[1] {
			pageLayout = pdf.LayoutLandscape2Up
		} else {
			pageLayout = pdf.LayoutPortrait
		}
	})
	radio.SetSelected(opts[0])

	content := container.NewVBox(
		widget.NewLabel(i18n.T(i18n.KeyPdfLayoutLabel)),
		radio,
	)

	dialog.ShowCustomConfirm(
		i18n.T(i18n.KeyPdfExportTitle),
		i18n.T(i18n.KeyPdfExportConfirm),
		i18n.T(i18n.KeyPdfExportCancel),
		content,
		func(ok bool) {
			if !ok {
				return
			}
			a.showFileSaveDialog(pageLayout)
		},
		a.window,
	)
}

func (a *App) showFileSaveDialog(pageLayout pdf.PageLayout) {
	d := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			return
		}
		path := writer.URI().Path()
		writer.Close()
		if !strings.HasSuffix(strings.ToLower(path), ".pdf") {
			path += ".pdf"
		}
		a.generatePDFTo(path, pageLayout)
	}, a.window)
	d.SetFileName(a.pdfDefaultFilename() + ".pdf")

	// Set initial location from settings or home directory.
	dir := a.settings.PdfExportDir
	if dir == "" {
		dir, _ = os.UserHomeDir()
	}
	if dir != "" {
		if listable, err := storage.ListerForURI(storage.NewFileURI(dir)); err == nil {
			d.SetLocation(listable)
		}
	}
	d.Show()
}

func (a *App) pdfDefaultFilename() string {
	s := a.sessionTab.Session()
	if s == nil {
		return "session"
	}
	title := strings.TrimSpace(s.Title)
	if title == "" {
		title = stripDiacritics(i18n.T(i18n.KeyPdfFilenamePrefix))
	} else {
		title = stripDiacritics(title)
	}
	filename := title
	if s.Date != "" {
		filename += " - " + s.Date
	}
	return filename
}

func (a *App) generatePDFTo(path string, pageLayout pdf.PageLayout) {
	s := a.sessionTab.Session()
	if s == nil {
		log.Printf("no session to generate PDF for")
		return
	}
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0755)
	lang := string(i18n.CurrentLang())
	loader := func(name string) (*model.Exercise, error) {
		ex, err := a.loadExerciseAny(name)
		if err != nil {
			return nil, err
		}
		if ex.I18n == nil && a.library != nil {
			if libEx, err := a.library.LoadExercise(name); err == nil && libEx.I18n != nil {
				ex.I18n = libEx.I18n
			}
		}
		return ex.Localized(lang), nil
	}
	if err := pdf.Generate(s, loader, path, pageLayout); err != nil {
		log.Printf("generate PDF: %v", err)
		a.statusBar.SetStatus(i18n.T(i18n.KeyStatusPdfError), 1)
		return
	}
	log.Printf("PDF generated: %s", path)
	a.statusBar.SetStatus(i18n.Tf(i18n.KeyStatusPdfGenerated, filepath.Base(path)), StatusSuccess)
}

func (a *App) updateExerciseFromRemote(name string) {
	if a.library == nil {
		return
	}
	ex, err := a.library.LoadExercise(name)
	if err != nil {
		log.Printf("load library exercise %s: %v", name, err)
		return
	}
	if err := a.store.SaveExercise(ex); err != nil {
		log.Printf("save updated exercise: %v", err)
		return
	}
	if a.exercise != nil && store.ToKebab(a.exercise.Name) == name {
		a.SetExercise(ex)
	}
	a.sessionNeedsRefresh = true
	a.refreshSessionTab()
	log.Printf("updated exercise from community: %s", name)
}

func (a *App) contributeExercise(name string) {
	ex, err := a.store.LoadExercise(name)
	if err != nil {
		log.Printf("load exercise %s for contribution: %v", name, err)
		return
	}

	data, err := yaml.Marshal(ex)
	if err != nil {
		log.Printf("marshal exercise for contribution: %v", err)
		return
	}

	token := a.settings.GithubToken
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		a.statusBar.SetStatus(i18n.T(i18n.KeyContributeNoToken), 1)
		return
	}

	a.statusBar.SetStatus(i18n.T(i18n.KeyContributeCreatingPr), 0)

	// Run in background to avoid blocking UI.
	go func() {
		prURL, err := createContributionPR(token, name, data)
		fyne.Do(func() {
			if err != nil {
				log.Printf("contribute PR failed: %v", err)
				a.statusBar.SetStatus(i18n.T(i18n.KeyContributeError)+": "+err.Error(), 1)
				return
			}
			a.statusBar.SetStatus(i18n.T(i18n.KeyContributePrCreated), 0)
			if prURL != "" {
				_ = openBrowser(prURL)
			}
		})
	}()
}

// --- Training mode ---

// buildTrainingPicker creates a full-page session list for picking a session to train.
func (a *App) buildTrainingPicker() fyne.CanvasObject {
	ys, ok := a.store.(*store.YAMLStore)
	if ok {
		// Ensure the session index is fresh (picks up newly saved sessions).
		ys.RebuildSessionIndex()
	}
	if !ok {
		placeholder := canvas.NewText(i18n.T(i18n.KeyTrainingNoSessions), color.NRGBA{R: 0xaa, G: 0xaa, B: 0xaa, A: 0xff})
		placeholder.Alignment = fyne.TextAlignCenter
		return container.NewCenter(placeholder)
	}
	sessions, _ := ys.ListSessions()
	if len(sessions) == 0 {
		placeholder := canvas.NewText(i18n.T(i18n.KeyTrainingNoSessions), color.NRGBA{R: 0xaa, G: 0xaa, B: 0xaa, A: 0xff})
		placeholder.Alignment = fyne.TextAlignCenter
		return container.NewCenter(placeholder)
	}

	type sessionInfo struct {
		file  string
		title string
		date  string
	}
	infos := make([]sessionInfo, 0, len(sessions))
	for _, name := range sessions {
		s, err := ys.LoadSession(name)
		title := name
		date := ""
		if err == nil {
			if s.Title != "" {
				title = s.Title
			}
			date = s.Date
		}
		infos = append(infos, sessionInfo{file: name, title: title, date: date})
	}

	list := widget.NewList(
		func() int { return len(infos) },
		func() fyne.CanvasObject {
			title := widget.NewLabel("")
			title.TextStyle.Bold = true
			date := widget.NewLabel("")
			date.Importance = widget.LowImportance
			return container.NewHBox(title, layout.NewSpacer(), date)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			row, ok := obj.(*fyne.Container)
			if !ok {
				return
			}
			if lbl, ok := row.Objects[0].(*widget.Label); ok {
				lbl.SetText(infos[id].title)
			}
			if lbl, ok := row.Objects[2].(*widget.Label); ok {
				lbl.SetText(infos[id].date)
			}
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		s, err := ys.LoadSession(infos[id].file)
		if err != nil {
			a.statusBar.SetStatus(fmt.Sprintf("Error: %v", err), 1)
			return
		}
		a.enterTrainingModeWithSession(s)
	}

	header := canvas.NewText(i18n.T(i18n.KeyTrainingPickSession), color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	header.TextSize = 18
	header.TextStyle.Bold = true

	return container.NewBorder(container.NewPadded(header), nil, nil, nil, list)
}

// enterTrainingModeWithSession starts training with the given session.
func (a *App) enterTrainingModeWithSession(s *model.Session) {
	if s == nil || len(s.Exercises) == 0 {
		return
	}
	lang := string(i18n.CurrentLang())
	exercises := make([]*model.Exercise, 0, len(s.Exercises))
	for _, entry := range s.Exercises {
		ex, err := a.loadExerciseAny(entry.Exercise)
		if err != nil {
			continue
		}
		exercises = append(exercises, ex.Localized(lang))
	}
	if len(exercises) == 0 {
		return
	}
	a.normalContent = a.window.Content()
	a.trainingMode = NewTrainingMode(a.window, s, exercises, func() {
		a.exitTrainingMode()
	})
	a.window.SetContent(a.trainingMode.Widget())
}

func (a *App) enterTrainingMode() {
	s := a.sessionTab.Session()
	if s == nil || len(s.Exercises) == 0 {
		return
	}

	// Resolve all exercises in order.
	lang := string(i18n.CurrentLang())
	exercises := make([]*model.Exercise, 0, len(s.Exercises))
	for _, entry := range s.Exercises {
		ex, err := a.loadExerciseAny(entry.Exercise)
		if err != nil {
			continue
		}
		exercises = append(exercises, ex.Localized(lang))
	}
	if len(exercises) == 0 {
		return
	}

	// Store normal content for restoration.
	a.normalContent = a.window.Content()

	// Create training mode.
	a.trainingMode = NewTrainingMode(a.window, s, exercises, func() {
		a.exitTrainingMode()
	})

	a.window.SetContent(a.trainingMode.Widget())
}

func (a *App) exitTrainingMode() {
	if a.trainingMode != nil {
		a.trainingMode.Stop()
		a.trainingMode = nil
	}
	if a.normalContent != nil {
		a.window.SetContent(a.normalContent)
		a.normalContent = nil
	}
	// Return to training picker (not the previous mode).
	if a.modeSwitchFunc != nil {
		a.modeSwitchFunc(ModeTraining)
	}
}

// --- Match live mode ---

func (a *App) enterMatchLive(match *model.Match, team *model.Team) {
	if match == nil {
		return
	}

	// Store normal content for restoration.
	a.normalContent = a.window.Content()

	a.matchLive = NewMatchLive(a.window, match, team, a.store, func() {
		a.exitMatchLive()
	})

	a.window.SetContent(a.matchLive.Widget())
}

func (a *App) exitMatchLive() {
	if a.matchLive != nil {
		a.matchLive.Stop()
		a.matchLive = nil
	}
	if a.normalContent != nil {
		a.window.SetContent(a.normalContent)
		a.normalContent = nil
	}
	// Return to match list.
	if a.modeSwitchFunc != nil {
		a.modeSwitchFunc(ModeMatch)
	}
}

func (a *App) showMatchSummary(match *model.Match) {
	if match == nil {
		return
	}

	// Store normal content for restoration.
	a.normalContent = a.window.Content()

	summary := NewMatchSummary(match, func() {
		if a.normalContent != nil {
			a.window.SetContent(a.normalContent)
			a.normalContent = nil
		}
		if a.modeSwitchFunc != nil {
			a.modeSwitchFunc(ModeMatch)
		}
	})

	a.window.SetContent(summary.Widget())
}

// --- Build managed exercises list ---

func (a *App) buildManagedExercises() []ManagedExercise {
	localMap := make(map[string]*model.Exercise)
	remoteMap := make(map[string]*model.Exercise)
	allNames := make(map[string]bool)

	if names, err := a.store.ListExercises(); err == nil {
		for _, name := range names {
			if ex, err := a.store.LoadExercise(name); err == nil {
				localMap[name] = ex
				allNames[name] = true
			}
		}
	}

	if a.library != nil {
		if names, err := a.library.ListExercises(); err == nil {
			for _, name := range names {
				if ex, err := a.library.LoadExercise(name); err == nil && ex.Name != "" {
					remoteMap[name] = ex
					allNames[name] = true
				}
			}
		}
	}

	sortedNames := make([]string, 0, len(allNames))
	for name := range allNames {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	// Build modtime map from index entries.
	modTimes := make(map[string]time.Time)
	if ys, ok := a.store.(*store.YAMLStore); ok {
		for _, entry := range ys.ExerciseIndexEntries() {
			modTimes[entry.File] = entry.Modified
		}
	}

	lang := string(i18n.CurrentLang())
	items := make([]ManagedExercise, 0, len(sortedNames))
	for _, name := range sortedNames {
		local := localMap[name]
		remote := remoteMap[name]

		var status ExerciseSyncStatus
		switch {
		case local != nil && remote == nil:
			status = StatusLocalOnly
		case local == nil && remote != nil:
			status = StatusRemoteOnly
		case local != nil && remote != nil:
			if exercisesEqual(local, remote) {
				status = StatusSynced
			} else {
				status = StatusModified
			}
		}

		displayName := name
		remoteDisplayName := ""
		category := ""
		ageGroup := ""
		courtType := ""
		duration := ""
		var tags []string
		var remoteTags []string
		// Pick the primary exercise for metadata.
		primary := local
		if primary == nil {
			primary = remote
		}
		if primary != nil {
			if local != nil && local.I18n == nil && remote != nil && remote.I18n != nil {
				local.I18n = remote.I18n
			}
			loc := primary.Localized(lang)
			displayName = loc.Name
			category = string(primary.Category)
			ageGroup = string(primary.AgeGroup)
			courtType = string(primary.CourtType)
			duration = primary.Duration
			tags = loc.Tags
		}
		if remote != nil {
			rloc := remote.Localized(lang)
			remoteDisplayName = rloc.Name
			remoteTags = rloc.Tags
		}

		items = append(items, ManagedExercise{
			Name:              name,
			Status:            status,
			LocalEx:           local,
			RemoteEx:          remote,
			DisplayName:       displayName,
			RemoteDisplayName: remoteDisplayName,
			Category:          category,
			AgeGroup:          ageGroup,
			CourtType:         courtType,
			Duration:          duration,
			Tags:              tags,
			RemoteTags:        remoteTags,
			ModTime:           modTimes[name],
		})
	}
	return items
}

func exercisesEqual(a, b *model.Exercise) bool {
	dataA, errA := yaml.Marshal(a)
	dataB, errB := yaml.Marshal(b)
	if errA != nil || errB != nil {
		return false
	}
	return string(dataA) == string(dataB)
}

// --- Library sync ---

// syncLibrary fetches community exercises from GitHub in the background.
func (a *App) syncLibrary() {
	if a.syncing || a.library == nil {
		return
	}
	a.syncing = true
	a.statusBar.SetStatus(i18n.T(i18n.KeySyncInProgress), 0)

	token := a.settings.GithubToken
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	cacheDir := a.library.Dir()

	go func() {
		result, err := store.SyncLibrary(context.Background(), cacheDir, token)
		fyne.Do(func() {
			a.syncing = false
			if err != nil {
				log.Printf("sync library: %v", err)
				a.statusBar.SetStatus(i18n.T(i18n.KeySyncFailed), 1)
			} else {
				total := len(result.Added) + len(result.Updated) + len(result.Removed)
				if total == 0 {
					a.statusBar.SetStatus(i18n.T(i18n.KeySyncUpToDate), 0)
				} else {
					a.statusBar.SetStatus(
						i18n.Tf(i18n.KeySyncCompleted, len(result.Added), len(result.Updated), len(result.Removed)), 0)
				}
			}
			a.sessionNeedsRefresh = true
			a.refreshSessionTab()
		})
	}()
}

// RebuildIndexAtStartup rebuilds the exercise index and reports parse errors.
func (a *App) RebuildIndexAtStartup() {
	ys, ok := a.store.(*store.YAMLStore)
	if !ok {
		return
	}
	if errs := ys.RebuildExerciseIndex(); len(errs) > 0 {
		a.statusBar.SetStatus(i18n.Tf(i18n.KeyStatusIndexParseErrors, len(errs), strings.Join(errs, "; ")), StatusWarning)
	}
}

// SyncLibraryIfEmpty triggers a sync if the cache has no exercises.
func (a *App) SyncLibraryIfEmpty() {
	if a.library == nil {
		return
	}
	if store.IsCacheEmpty(a.library.Dir()) {
		a.syncLibrary()
	}
}

// --- Utilities ---

func stripDiacritics(s string) string {
	t := norm.NFD.String(s)
	var b strings.Builder
	b.Grow(len(t))
	for _, r := range t {
		if !unicode.Is(unicode.Mn, r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Suppress unused imports.
var _ = theme.ColorDarkBg
var _ = canvas.NewRectangle
