package ui

import (
	"context"
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
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/anim"
	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/pdf"
	"github.com/darkweaver87/courtdraw/internal/store"
	"github.com/darkweaver87/courtdraw/internal/ui/editor"
	"github.com/darkweaver87/courtdraw/internal/ui/fynecourt"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
)

// App is the main application state.
type App struct {
	window   fyne.Window
	store    store.Store
	settings *store.Settings
	library  *store.Library
	version  string
	syncing  bool

	exercise    *model.Exercise
	editorState editor.EditorState
	playback    *anim.Playback
	editLang    string

	// Fyne widgets.
	court        *fynecourt.CourtWidget
	fileToolbar  *FileToolbar
	toolPalette  *ToolPalette
	propsPanel   *PropertiesPanel
	seqTimeline  *SeqTimeline
	instrPanel   *InstructionsPanel
	animControls *AnimControls
	statusBar    *StatusBar
	tooltipLayer *TooltipLayer
	sessionTab   *SessionTab

	// Tab management.
	tabs                *container.AppTabs
	sessionNeedsRefresh bool

	// Edit language buttons.
	editLangBtns  [2]*widget.Button
	editLangLabel *canvas.Text

	// Responsive containers (for language rebuild).
	editorResponsive *ResponsiveContainer
}

// NewApp creates a new App instance.
func NewApp(st store.Store, settings *store.Settings, lib *store.Library, w fyne.Window, version string) *App {
	a := &App{
		window:   w,
		store:    st,
		settings: settings,
		library:  lib,
		version:  version,
		editLang: string(i18n.CurrentLang()),
	}
	a.editorState.ActiveTool = editor.ToolSelect
	return a
}

// BuildUI creates the full application UI and returns the root canvas object.
func (a *App) BuildUI() fyne.CanvasObject {
	// Create all panel widgets.
	a.court = fynecourt.NewCourtWidget()
	a.court.SetEditorState(&a.editorState)
	a.court.OnChanged = func() {
		a.refreshEditor()
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
		a.court.Refresh()
	}

	a.propsPanel = NewPropertiesPanel()
	a.propsPanel.OnModified = func() {
		a.court.Refresh()
		a.fileToolbar.SetModified(a.editorState.Modified)
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

	a.instrPanel = NewInstructionsPanel()
	a.instrPanel.OnModified = func() {
		a.fileToolbar.SetModified(a.editorState.Modified)
	}

	a.animControls = NewAnimControls()
	a.animControls.OnStateChanged = func() {
		a.syncAnimState()
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
	}
	a.sessionTab.OnStatus = func(msg string, level int) {
		a.statusBar.SetStatus(msg, level)
	}

	// Build editor tab content.
	editorContent := a.buildEditorTab()

	// Build tabs.
	a.tabs = container.NewAppTabs(
		container.NewTabItem(i18n.T("tab.exercise_editor"), editorContent),
		container.NewTabItem(i18n.T("tab.session"), a.sessionTab.Widget()),
	)
	a.tabs.OnChanged = func(tab *container.TabItem) {
		if a.tabs.SelectedIndex() == 1 {
			a.sessionNeedsRefresh = true
			a.refreshSessionTab()
		}
	}

	return container.NewStack(
		container.NewBorder(nil, a.statusBar.Widget(), nil, nil, a.tabs),
		a.tooltipLayer.Widget(),
	)
}

func (a *App) buildEditorTab() fyne.CanvasObject {
	// Edit language bar.
	a.editLangBtns[0] = widget.NewButton("EN", func() {
		if a.editLang != "en" {
			a.editLang = "en"
			a.switchLang("en")
		}
	})
	a.editLangBtns[1] = widget.NewButton("FR", func() {
		if a.editLang != "fr" {
			a.editLang = "fr"
			a.switchLang("fr")
		}
	})
	a.updateLangBtnStyles()

	a.editLangLabel = canvas.NewText(i18n.T("edit_lang.label")+":", color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff})
	a.editLangLabel.TextSize = 11

	a.editorResponsive = NewResponsiveContainer(a.buildEditorDesktop, a.buildEditorMobile)
	return a.editorResponsive
}

func (a *App) buildEditorDesktop() fyne.CanvasObject {
	langBar := container.NewHBox(a.editLangLabel, a.editLangBtns[0], a.editLangBtns[1], layout.NewSpacer())

	// Top section: toolbar + lang bar + timeline.
	topSection := container.NewVBox(
		a.fileToolbar.Widget(),
		langBar,
		a.seqTimeline.Widget(),
	)

	// Bottom bar: anim controls.
	bottomBar := a.animControls.Widget()

	// Middle section: palette | court | properties — resizable splits.
	leftSplit := container.NewHSplit(a.toolPalette.Widget(), a.court)
	leftSplit.SetOffset(0.12) // ~12% for palette

	middle := container.NewHSplit(leftSplit, a.propsPanel.Widget())
	middle.SetOffset(0.78) // ~78% for palette+court, ~22% for properties

	// Vertical split: court area (top) | instructions (bottom) — resizable.
	mainArea := container.NewBorder(topSection, bottomBar, nil, nil, middle)
	vSplit := container.NewVSplit(mainArea, a.instrPanel.Widget())
	vSplit.SetOffset(0.82) // ~82% court, ~18% instructions

	return vSplit
}

func (a *App) buildEditorMobile() fyne.CanvasObject {
	langBar := container.NewHBox(a.editLangLabel, a.editLangBtns[0], a.editLangBtns[1], layout.NewSpacer())

	// Court tab: toolbar + lang + court + zoom slider + timeline + anim controls.
	zoomLabel := canvas.NewText("1.0x", color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff})
	zoomLabel.TextSize = 14
	zoomSlider := widget.NewSlider(1.0, 5.0)
	zoomSlider.Step = 0.1
	zoomSlider.Value = 1.0
	zoomSlider.OnChanged = func(v float64) {
		a.court.SetZoom(v)
		zoomLabel.Text = fmt.Sprintf("%.1fx", v)
		zoomLabel.Refresh()
	}
	zoomReset := widget.NewButton("1:1", func() {
		a.court.ResetZoom()
		zoomSlider.SetValue(1.0)
		zoomLabel.Text = "1.0x"
		zoomLabel.Refresh()
	})
	zoomReset.Importance = widget.LowImportance
	zoomBar := container.NewBorder(nil, nil,
		zoomLabel,
		container.NewGridWrap(fyne.NewSize(48, 36), zoomReset),
		zoomSlider,
	)
	courtTop := container.NewVBox(
		a.fileToolbar.Widget(),
		langBar,
		zoomBar,
	)
	courtBottom := container.NewVBox(
		a.seqTimeline.Widget(),
		a.animControls.Widget(),
	)
	courtTab := container.NewBorder(courtTop, courtBottom, nil, nil, a.court)

	// Tools tab: tool palette (full screen, scrollable).
	toolsTab := container.NewScroll(a.toolPalette.Widget())

	// Props tab: properties + instructions (vertical split 60/40).
	propsTab := container.NewVSplit(a.propsPanel.Widget(), a.instrPanel.Widget())
	propsTab.SetOffset(0.6)

	tabs := container.NewAppTabs(
		container.NewTabItem(i18n.T("mobile.tab.court"), courtTab),
		container.NewTabItem(i18n.T("mobile.tab.tools"), toolsTab),
		container.NewTabItem(i18n.T("mobile.tab.props"), propsTab),
	)
	tabs.SetTabLocation(container.TabLocationBottom)

	return tabs
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
	// Now refresh editor (calls propsPanel.Update which uses new options).
	a.propsPanel.SyncFromExercise()
	a.instrPanel.ForceResync()
	a.refreshEditor()
	a.updateLangBtnStyles()
	// Update tab labels to reflect new UI language.
	if a.tabs != nil && len(a.tabs.Items) >= 2 {
		a.tabs.Items[0].Text = i18n.T("tab.exercise_editor")
		a.tabs.Items[1].Text = i18n.T("tab.session")
		a.tabs.Refresh()
	}
	a.editLangLabel.Text = i18n.T("edit_lang.label") + ":"
	a.editLangLabel.Refresh()
	a.statusBar.SetStatus("", 0)
	if a.editorResponsive != nil {
		a.editorResponsive.ForceRebuild()
	}
	a.sessionNeedsRefresh = true
	// Persist language choice.
	if ys, ok := a.store.(*store.YAMLStore); ok {
		settings, _ := ys.LoadSettings()
		settings.Language = lang
		ys.SaveSettings(settings)
	}
}

func (a *App) updateLangBtnStyles() {
	if a.editLang == "en" {
		a.editLangBtns[0].Importance = widget.HighImportance
		a.editLangBtns[1].Importance = widget.LowImportance
	} else {
		a.editLangBtns[0].Importance = widget.LowImportance
		a.editLangBtns[1].Importance = widget.HighImportance
	}
	a.editLangBtns[0].Refresh()
	a.editLangBtns[1].Refresh()
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
	a.fileToolbar.SetModified(a.editorState.Modified)

	if a.playback != nil {
		a.animControls.SetPlayback(a.playback, len(a.exercise.Sequences))
	}

	// Forward editor state status to the status bar.
	if a.editorState.StatusMsg != "" {
		a.statusBar.SetStatus(a.editorState.StatusMsg, a.editorState.StatusLevel)
		a.editorState.StatusMsg = ""
	}
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
	})
}

func (a *App) showAbout() {
	showAboutDialog(a.window, a.version)
}

// CheckVersionAtStartup checks GitHub for a newer release in the background.
func (a *App) CheckVersionAtStartup() {
	if a.version == "dev" || a.version == "" {
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
		if info.Tag != "" && info.Tag != a.version && info.Tag > a.version {
			fyne.Do(func() {
				showUpdateDialog(a.window, info.Tag, info.URL)
			})
		}
	}()
}

// --- Exercise management ---

// SetExercise sets the current exercise.
func (a *App) SetExercise(ex *model.Exercise) {
	a.exercise = ex
	a.court.SetExercise(ex)
	a.editorState.Deselect()
	a.editorState.ClearModified()
	a.propsPanel.SyncFromExercise()
	if ex != nil {
		a.playback = anim.NewPlayback(ex)
	} else {
		a.playback = nil
	}
	a.court.SetPlayback(a.playback)
	a.refreshEditor()
	a.updateWindowTitle()
}

func (a *App) updateWindowTitle() {
	title := i18n.T("app.title")
	if a.exercise != nil && a.exercise.Name != "" {
		displayName := a.exercise.Localized(a.editLang).Name
		fileName := store.ToKebab(a.exercise.Name) + ".yaml"
		if displayName != a.exercise.Name && displayName != "" {
			title += " — " + displayName + " [" + fileName + "]"
		} else {
			title += " — " + fileName
		}
	}
	a.window.SetTitle(title)
}

// NewExercise creates a blank exercise.
func (a *App) NewExercise() {
	ex := &model.Exercise{
		Name:          i18n.T("default.exercise_name"),
		CourtType:     model.HalfCourt,
		CourtStandard: model.FIBA,
		Sequences: []model.Sequence{
			{Label: i18n.T("default.sequence_label")},
		},
	}
	if a.editLang != "" && a.editLang != "en" {
		tr := ex.EnsureI18n(a.editLang)
		tr.Name = ex.Name
		ex.SetI18n(a.editLang, tr)
	}
	a.SetExercise(ex)
	a.statusBar.SetStatus(i18n.T("status.new_exercise"), 0)
}

// NewSession creates a blank session.
func (a *App) NewSession() {
	s := &model.Session{
		Title: i18n.T("default.session_name"),
		Date:  time.Now().Format("2006-01-02"),
	}
	a.sessionTab.SetSession(s)
	a.statusBar.SetStatus(i18n.T("status.new_session"), 0)
}

func (a *App) addSequence() {
	if a.exercise == nil {
		return
	}
	var newSeq model.Sequence
	currentIdx := a.court.SeqIndex()
	if currentIdx < len(a.exercise.Sequences) {
		current := &a.exercise.Sequences[currentIdx]
		newSeq.Players = make([]model.Player, len(current.Players))
		copy(newSeq.Players, current.Players)
		newSeq.Accessories = make([]model.Accessory, len(current.Accessories))
		copy(newSeq.Accessories, current.Accessories)
		newSeq.BallCarrier = current.BallCarrier
		for _, act := range current.Actions {
			if act.Type == model.ActionPass && act.To.IsPlayer {
				newSeq.BallCarrier = act.To.PlayerID
			}
		}
	}
	a.exercise.Sequences = append(a.exercise.Sequences, newSeq)
	newIdx := len(a.exercise.Sequences) - 1
	a.court.SetSequence(newIdx)
	a.editorState.Deselect()
	a.editorState.MarkModified()
	a.statusBar.SetStatus(i18n.Tf("status.seq_added", newIdx+1), 0)
	a.refreshEditor()
}

func (a *App) deleteSequence(idx int) {
	if a.exercise == nil || len(a.exercise.Sequences) <= 1 {
		return
	}
	if idx < 0 || idx >= len(a.exercise.Sequences) {
		return
	}
	a.exercise.Sequences = append(a.exercise.Sequences[:idx], a.exercise.Sequences[idx+1:]...)
	// Adjust current index.
	newIdx := idx
	if newIdx >= len(a.exercise.Sequences) {
		newIdx = len(a.exercise.Sequences) - 1
	}
	a.court.SetSequence(newIdx)
	a.editorState.Deselect()
	a.editorState.MarkModified()
	a.statusBar.SetStatus(i18n.Tf("status.seq_deleted", idx+1), 0)
	a.refreshEditor()
}

// --- File operations ---

func (a *App) handleFileAction(action FileAction) {
	switch action {
	case FileActionNew:
		a.NewExercise()
		a.editorState.MarkModified()
	case FileActionOpen:
		a.showOpenDialog()
	case FileActionSave:
		a.saveExercise()
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
	a.statusBar.SetStatus(i18n.Tf("status.opened", name), 0)
}

func (a *App) saveExercise() {
	if a.exercise == nil {
		return
	}
	if err := a.store.SaveExercise(a.exercise); err != nil {
		log.Printf("save exercise: %v", err)
		a.statusBar.SetStatus(i18n.T("status.save_error"), 1)
		return
	}
	a.editorState.ClearModified()
	a.recordRecentFile(store.ToKebab(a.exercise.Name))
	a.fileToolbar.SetModified(false)
	a.sessionNeedsRefresh = true
	a.refreshSessionTab()
	fileName := store.ToKebab(a.exercise.Name) + ".yaml"
	if ys, ok := a.store.(*store.YAMLStore); ok {
		fileName = filepath.Join(ys.ExercisesDir(), fileName)
	}
	a.statusBar.SetStatus(i18n.Tf("status.saved", fileName), 0)
	a.updateWindowTitle()
}

func (a *App) saveAsExercise() {
	if a.exercise == nil {
		return
	}

	entry := widget.NewEntry()
	entry.SetPlaceHolder(i18n.T("save_as.placeholder"))
	entry.SetText(a.exercise.Localized(a.editLang).Name)

	d := dialog.NewForm(i18n.T("save_as.title"), i18n.T("prefs.save"), i18n.T("dialog.cancel"),
		[]*widget.FormItem{widget.NewFormItem(i18n.T("save_as.name_label"), entry)},
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
					dialog.ShowConfirm(i18n.T("save_as.overwrite_title"),
						i18n.Tf("save_as.overwrite_msg", kebab+".yaml"),
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
		a.statusBar.SetStatus(i18n.T("status.save_error"), 0)
		return
	}
	a.editorState.ClearModified()
	fileName := store.ToKebab(newName) + ".yaml"
	if ys, ok := a.store.(*store.YAMLStore); ok {
		fileName = filepath.Join(ys.ExercisesDir(), fileName)
	}
	a.statusBar.SetStatus(i18n.Tf("status.saved", fileName), 0)
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
			obj.(*widget.Label).SetText(items[id])
		},
	)
	var d dialog.Dialog
	list.OnSelected = func(id widget.ListItemID) {
		if id < len(items) {
			a.openExercise(items[id])
			d.Hide()
		}
	}
	d = dialog.NewCustom(i18n.T("tooltip.open"), i18n.T("dialog.cancel"), list, a.window)
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
			obj.(*widget.Label).SetText(items[id])
		},
	)
	var d dialog.Dialog
	list.OnSelected = func(id widget.ListItemID) {
		if id < len(items) {
			a.importExercise(items[id])
			d.Hide()
		}
	}
	d = dialog.NewCustom(i18n.T("tooltip.import"), i18n.T("dialog.cancel"), list, a.window)
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
	a.recordRecentFile(store.ToKebab(ex.Name))
	a.statusBar.SetStatus(i18n.Tf("status.imported", ex.Name), 0)
}

func (a *App) recordRecentFile(name string) {
	if ys, ok := a.store.(*store.YAMLStore); ok {
		ys.RecordRecentFile(name)
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
			obj.(*widget.Label).SetText(items[id])
		},
	)
	var d dialog.Dialog
	list.OnSelected = func(id widget.ListItemID) {
		if id < len(items) {
			a.openExercise(items[id])
			d.Hide()
		}
	}
	d = dialog.NewCustom(i18n.T("tooltip.recent"), i18n.T("dialog.cancel"), list, a.window)
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
	a.statusBar.SetStatus(i18n.Tf("status.deleted", name), 0)
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
			ys.RebuildExerciseIndex()
		}
		a.sessionTab.SetExercises(a.buildManagedExercises())
		a.syncLibrary()
	case SessionTabActionOpenExercise:
		a.openExercise(ev.Name)
		a.tabs.SelectIndex(0)
	case SessionTabActionUpdate:
		a.updateExerciseFromRemote(ev.Name)
	case SessionTabActionContribute:
		a.contributeExercise(ev.Name)
	case SessionTabActionDeleteExercise:
		a.deleteExercise(ev.Name)
	case SessionTabActionRecent:
		a.showRecentSessions()
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
			obj.(*widget.Label).SetText(items[id])
		},
	)
	var d dialog.Dialog
	list.OnSelected = func(id widget.ListItemID) {
		if id < len(items) {
			a.openSession(items[id])
			d.Hide()
		}
	}
	d = dialog.NewCustom(i18n.T("session.open"), i18n.T("dialog.cancel"), list, a.window)
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
			obj.(*widget.Label).SetText(items[id])
		},
	)
	var d dialog.Dialog
	list.OnSelected = func(id widget.ListItemID) {
		if id < len(items) {
			a.openSession(items[id])
			d.Hide()
		}
	}
	d = dialog.NewCustom(i18n.T("session.recent"), i18n.T("dialog.cancel"), list, a.window)
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
	if ys, ok := a.store.(*store.YAMLStore); ok {
		ys.RecordRecentSession(name)
	}
	a.statusBar.SetStatus(i18n.Tf("status.opened", name), 0)
}

func (a *App) saveSession() {
	s := a.sessionTab.Session()
	if s == nil {
		return
	}
	if err := a.store.SaveSession(s); err != nil {
		log.Printf("save session: %v", err)
		a.statusBar.SetStatus(i18n.T("status.save_error"), 1)
		return
	}
	a.sessionTab.ClearModified()
	name := store.SessionFileName(s)
	if name != "" {
		if ys, ok := a.store.(*store.YAMLStore); ok {
			ys.RecordRecentSession(name)
		}
	}
	a.statusBar.SetStatus(i18n.Tf("status.saved", s.Title), 0)
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

func (a *App) showPdfExportDialog() {
	s := a.sessionTab.Session()
	if s == nil {
		log.Printf("no session to generate PDF for")
		return
	}

	d := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			return
		}
		path := writer.URI().Path()
		writer.Close()
		if !strings.HasSuffix(strings.ToLower(path), ".pdf") {
			path += ".pdf"
		}
		a.generatePDFTo(path)
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
		title = stripDiacritics(i18n.T("pdf.filename_prefix"))
	} else {
		title = stripDiacritics(title)
	}
	filename := title
	if s.Date != "" {
		filename += " - " + s.Date
	}
	return filename
}

func (a *App) generatePDFTo(path string) {
	s := a.sessionTab.Session()
	if s == nil {
		log.Printf("no session to generate PDF for")
		return
	}
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)
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
	if err := pdf.Generate(s, loader, path); err != nil {
		log.Printf("generate PDF: %v", err)
		return
	}
	log.Printf("PDF generated: %s", path)
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
		a.statusBar.SetStatus(i18n.T("contribute.no_token"), 1)
		return
	}

	a.statusBar.SetStatus(i18n.T("contribute.creating_pr"), 0)

	// Run in background to avoid blocking UI.
	go func() {
		prURL, err := createContributionPR(token, name, data)
		fyne.Do(func() {
			if err != nil {
				log.Printf("contribute PR failed: %v", err)
				a.statusBar.SetStatus(i18n.T("contribute.error")+": "+err.Error(), 1)
				return
			}
			a.statusBar.SetStatus(i18n.T("contribute.pr_created"), 0)
			if prURL != "" {
				openBrowser(prURL)
			}
		})
	}()
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
				if ex, err := a.library.LoadExercise(name); err == nil {
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
		category := ""
		ageGroup := ""
		courtType := ""
		duration := ""
		var tags []string
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

		items = append(items, ManagedExercise{
			Name:        name,
			Status:      status,
			LocalEx:     local,
			RemoteEx:    remote,
			DisplayName: displayName,
			Category:    category,
			AgeGroup:    ageGroup,
			CourtType:   courtType,
			Duration:    duration,
			Tags:        tags,
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
	a.statusBar.SetStatus(i18n.T("sync.in_progress"), 0)

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
				a.statusBar.SetStatus(i18n.T("sync.failed"), 1)
			} else {
				total := len(result.Added) + len(result.Updated) + len(result.Removed)
				if total == 0 {
					a.statusBar.SetStatus(i18n.T("sync.up_to_date"), 0)
				} else {
					a.statusBar.SetStatus(
						i18n.Tf("sync.completed", len(result.Added), len(result.Updated), len(result.Removed)), 0)
				}
			}
			a.sessionNeedsRefresh = true
			a.refreshSessionTab()
		})
	}()
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

// cycleLang cycles through supported languages and persists the choice.
func (a *App) cycleLang() {
	langs := i18n.SupportedLangs()
	cur := i18n.CurrentLang()
	for idx, l := range langs {
		if l == cur {
			next := langs[(idx+1)%len(langs)]
			i18n.SetLang(next)
			if ys, ok := a.store.(*store.YAMLStore); ok {
				settings, _ := ys.LoadSettings()
				settings.Language = string(next)
				ys.SaveSettings(settings)
			}
			return
		}
	}
}

// Suppress unused imports.
var _ = theme.ColorDarkBg
var _ = canvas.NewRectangle
