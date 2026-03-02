package ui

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"

	gioapp "gioui.org/app"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"gopkg.in/yaml.v3"

	"github.com/darkweaver87/courtdraw/internal/anim"
	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/pdf"
	"github.com/darkweaver87/courtdraw/internal/store"
	"github.com/darkweaver87/courtdraw/internal/ui/editor"
	"github.com/darkweaver87/courtdraw/internal/ui/icon"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
	uiwidget "github.com/darkweaver87/courtdraw/internal/ui/widget"
)

// App is the main application state.
type App struct {
	theme   *material.Theme
	store   store.Store
	court   uiwidget.CourtWidget
	exercise *model.Exercise

	editorState   editor.EditorState
	toolPalette   *uiwidget.ToolPalette
	propsPanel    *uiwidget.PropertiesPanel
	seqTimeline   uiwidget.SeqTimeline
	instrPanel    *uiwidget.InstructionsPanel
	fileToolbar   uiwidget.FileToolbar
	exerciseList  *uiwidget.ExerciseListOverlay

	playback      *anim.Playback
	animControls  uiwidget.AnimControls

	sessionComposer *uiwidget.SessionComposer
	library         *store.Library
	libraryOverlay  *uiwidget.LibraryOverlay

	exerciseManager *uiwidget.ExerciseManager
	mgrNeedsRefresh bool

	window         *gioapp.Window
	pendingPDFPath chan string

	activeTab          int
	tabClickables      [3]widget.Clickable
	langClick          widget.Clickable
	composerNeedsRefresh bool
}

// NewApp creates a new App instance.
// libraryDir is the path to the community library directory (can be empty).
func NewApp(th *material.Theme, st store.Store, libraryDir string) *App {
	a := &App{
		theme:           th,
		store:           st,
		toolPalette:     uiwidget.NewToolPalette(),
		propsPanel:      uiwidget.NewPropertiesPanel(),
		instrPanel:      uiwidget.NewInstructionsPanel(),
		exerciseList:    uiwidget.NewExerciseListOverlay(),
		sessionComposer: uiwidget.NewSessionComposer(),
		libraryOverlay:  uiwidget.NewLibraryOverlay(),
		exerciseManager: uiwidget.NewExerciseManager(),
		pendingPDFPath: make(chan string, 1),
	}
	if libraryDir != "" {
		a.library = store.NewLibrary(libraryDir)
	}
	a.editorState.ActiveTool = editor.ToolSelect
	return a
}

// SetWindow stores the window reference for async invalidation.
func (a *App) SetWindow(w *gioapp.Window) {
	a.window = w
}

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
}

// LoadFirstExercise attempts to load the first available exercise.
func (a *App) LoadFirstExercise() error {
	names, err := a.store.ListExercises()
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return nil
	}
	ex, err := a.store.LoadExercise(names[0])
	if err != nil {
		return err
	}
	a.SetExercise(ex)
	return nil
}

// Layout renders the full application UI.
func (a *App) Layout(gtx layout.Context) layout.Dimensions {
	// handle tab clicks
	for i := range a.tabClickables {
		if a.tabClickables[i].Clicked(gtx) {
			if a.activeTab != i {
				a.activeTab = i
				if i == 1 {
					a.composerNeedsRefresh = true
				}
				if i == 2 {
					a.mgrNeedsRefresh = true
				}
			}
		}
	}

	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return a.layoutTabBar(gtx)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return a.layoutContent(gtx)
				}),
			)
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			if !a.exerciseList.Visible {
				return layout.Dimensions{}
			}
			dims, selected := a.exerciseList.Layout(gtx, a.theme)
			if selected != "" {
				a.openExercise(selected)
			}
			return dims
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			if a.libraryOverlay == nil || !a.libraryOverlay.Visible {
				return layout.Dimensions{}
			}
			dims, selected := a.libraryOverlay.Layout(gtx, a.theme)
			if selected != "" {
				a.importExercise(selected)
			}
			return dims
		}),
	)
}

func (a *App) layoutTabBar(gtx layout.Context) layout.Dimensions {
	barHeight := gtx.Dp(unit.Dp(theme.TabBarHeight))

	// background
	rect := clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, barHeight)}.Op()
	paint.FillShape(gtx.Ops, theme.ColorDarkBg, rect)

	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		// app name
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(12), Right: unit.Dp(20)}.Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(a.theme, unit.Sp(16), i18n.T("app.title"))
					lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
					lbl.Font.Weight = font.Bold
					return lbl.Layout(gtx)
				},
			)
		}),
		// exercise editor tab
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.layoutTab(gtx, 0, i18n.T("tab.exercise_editor"))
		}),
		// session composer tab
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.layoutTab(gtx, 1, i18n.T("tab.session_composer"))
		}),
		// exercise manager tab
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.layoutTab(gtx, 2, i18n.T("tab.exercise_manager"))
		}),
		// spacer
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 0)}
		}),
		// language toggle
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if a.langClick.Clicked(gtx) {
				a.cycleLang()
			}
			return layout.Inset{Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return icon.IconTextBtn(gtx, a.theme, &a.langClick, icon.Language, strings.ToUpper(string(i18n.CurrentLang())), theme.ColorTabActive)
			})
		}),
	)
}

func (a *App) layoutTab(gtx layout.Context, index int, title string) layout.Dimensions {
	return material.Clickable(gtx, &a.tabClickables[index], func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{
			Top: unit.Dp(8), Bottom: unit.Dp(8),
			Left: unit.Dp(16), Right: unit.Dp(16),
		}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			col := theme.ColorTabText
			if a.activeTab == index {
				col = theme.ColorTabActive
			}
			lbl := material.Label(a.theme, unit.Sp(14), title)
			lbl.Color = col
			return lbl.Layout(gtx)
		})
	})
}

func (a *App) layoutContent(gtx layout.Context) layout.Dimensions {
	switch a.activeTab {
	case 0:
		return a.layoutExerciseEditor(gtx)
	case 1:
		return a.layoutSessionComposer(gtx)
	case 2:
		return a.layoutExerciseManager(gtx)
	default:
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}
}

func (a *App) layoutExerciseEditor(gtx layout.Context) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// File toolbar.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			dims, action := a.fileToolbar.Layout(gtx, a.theme, a.editorState.Modified)
			a.handleFileAction(action)
			return dims
		}),
		// Rest of the editor (or empty state).
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if a.exercise == nil {
				return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(a.theme, unit.Sp(18), i18n.T("tab.no_exercise"))
					lbl.Color = theme.ColorTabText
					return lbl.Layout(gtx)
				})
			}
			return a.layoutEditorContent(gtx)
		}),
	)
}

func (a *App) layoutEditorContent(gtx layout.Context) layout.Dimensions {
	isAnimating := a.playback != nil && a.playback.State() == anim.StatePlaying

	// Track playback seqIndex before anim controls process clicks,
	// so we can detect changes from prev/next buttons.
	prevPBSeq := -1
	if a.playback != nil {
		prevPBSeq = a.playback.SeqIndex()
	}

	seqIdx := a.court.SeqIndex()
	var currentSeq *model.Sequence
	if seqIdx < len(a.exercise.Sequences) {
		currentSeq = &a.exercise.Sequences[seqIdx]
	}

	dims := layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Top: sequence timeline.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.seqTimeline.Layout(gtx, a.theme, a.exercise, &a.court, &a.editorState)
		}),
		// Middle: tool palette | court | properties.
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				// Left: tool palette (120dp).
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(120))
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return a.toolPalette.Layout(gtx, a.theme, &a.editorState)
				}),
				// Center: court canvas (with animation support).
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					if isAnimating && a.playback != nil {
						frame, needRedraw := a.playback.Update()
						if needRedraw {
							gtx.Execute(op.InvalidateCmd{})
						}
						// Sync court seq index during playback for timeline highlight.
						a.court.SetSequence(a.playback.SeqIndex())
						return a.court.LayoutAnimated(gtx, a.theme, &frame)
					}
					return a.court.Layout(gtx, a.theme, &a.editorState)
				}),
				// Right: properties panel (220dp).
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max.X = gtx.Dp(unit.Dp(220))
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
					return a.propsPanel.Layout(gtx, a.theme, a.exercise, &a.editorState, seqIdx)
				}),
			)
		}),
		// Animation controls.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			numSeqs := len(a.exercise.Sequences)
			return a.animControls.Layout(gtx, a.theme, a.playback, numSeqs)
		}),
		// Bottom: instructions panel.
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.instrPanel.Layout(gtx, a.theme, currentSeq, seqIdx, &a.editorState)
		}),
	)

	// Post-layout sync: if anim controls changed the playback seqIndex
	// (via prev/next), sync court to match. Otherwise, sync playback
	// from the court (e.g. user clicked a timeline tab).
	// Re-check state now (Play may have been clicked during this frame).
	if a.playback != nil && a.playback.State() != anim.StatePlaying {
		newPBSeq := a.playback.SeqIndex()
		if newPBSeq != prevPBSeq {
			// Anim controls changed it — sync court from playback.
			a.court.SetSequence(newPBSeq)
		} else {
			// Timeline or other UI changed court — sync playback from court.
			a.playback.SetSeqIndex(a.court.SeqIndex())
		}
	}

	return dims
}

func (a *App) layoutSessionComposer(gtx layout.Context) layout.Dimensions {
	// Only refresh library when flagged dirty (tab switch, save, import).
	if a.composerNeedsRefresh {
		a.refreshLibraryForComposer()
		a.composerNeedsRefresh = false
	}

	// Re-resolve exercises when the session list changes (e.g. after adding/removing).
	if a.sessionComposer.Modified() {
		a.resolveSessionExercises()
	}

	// Check for pending PDF export from native dialog.
	select {
	case path := <-a.pendingPDFPath:
		a.generatePDFTo(path)
	default:
	}

	// Handle session actions.
	action := a.sessionComposer.HandleActions(gtx)
	switch action {
	case uiwidget.SessionActionNew:
		a.NewSession()
	case uiwidget.SessionActionOpen:
		a.showOpenSessionDialog()
	case uiwidget.SessionActionSave:
		a.saveSession()
	case uiwidget.SessionActionGenerate:
		a.showPdfExportDialog()
	}

	// Handle session list overlay selection.
	if a.sessionComposer != nil {
		slo := a.sessionComposer.SessionListOverlay()
		if slo != nil && slo.OnSelect != "" {
			name := slo.OnSelect
			slo.OnSelect = ""
			a.openSession(name)
		}
	}

	return a.sessionComposer.Layout(gtx, a.theme)
}

func (a *App) refreshLibraryForComposer() {
	var allNames []string
	var allExercises []*model.Exercise
	seen := make(map[string]bool)
	lang := string(i18n.CurrentLang())

	// Index community i18n data for merging into local exercises.
	communityI18n := make(map[string]map[string]model.ExerciseI18n)
	if a.library != nil {
		if libNames, err := a.library.ListExercises(); err == nil {
			for _, name := range libNames {
				if libEx, err := a.library.LoadExercise(name); err == nil && libEx.I18n != nil {
					communityI18n[name] = libEx.I18n
				}
			}
		}
	}

	// Local exercises first.
	names, err := a.store.ListExercises()
	if err == nil {
		for _, name := range names {
			ex, err := a.store.LoadExercise(name)
			if err == nil {
				if ex.I18n == nil {
					if ci, ok := communityI18n[name]; ok {
						ex.I18n = ci
					}
				}
				allNames = append(allNames, name)
				allExercises = append(allExercises, ex.Localized(lang))
				seen[name] = true
			}
		}
	}

	// Community library exercises (not already local).
	if a.library != nil {
		libNames, err := a.library.ListExercises()
		if err == nil {
			for _, name := range libNames {
				if seen[name] {
					continue
				}
				ex, err := a.library.LoadExercise(name)
				if err == nil {
					allNames = append(allNames, name)
					allExercises = append(allExercises, ex.Localized(lang))
				}
			}
		}
	}

	a.sessionComposer.SetLibrary(allNames, allExercises)
	a.resolveSessionExercises()
}

// loadExerciseAny tries loading from local store first, then community library.
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

func (a *App) resolveSessionExercises() {
	if a.sessionComposer.Session() == nil {
		return
	}
	resolved := make(map[string]*model.Exercise)
	for _, entry := range a.sessionComposer.Session().Exercises {
		if _, ok := resolved[entry.Exercise]; !ok {
			ex, err := a.loadExerciseAny(entry.Exercise)
			if err == nil {
				resolved[entry.Exercise] = ex
			}
		}
	}
	a.sessionComposer.SetResolvedExercises(resolved)
}

// NewSession creates a blank session.
func (a *App) NewSession() {
	s := &model.Session{
		Title: i18n.T("default.session_name"),
		Date:  time.Now().Format("2006-01-02"),
	}
	a.sessionComposer.SetSession(s)
}

func (a *App) showOpenSessionDialog() {
	names, err := a.store.ListSessions()
	if err != nil {
		log.Printf("list sessions: %v", err)
		return
	}
	slo := a.sessionComposer.SessionListOverlay()
	if slo != nil {
		slo.Show(names)
	}
}

func (a *App) openSession(name string) {
	s, err := a.store.LoadSession(name)
	if err != nil {
		log.Printf("load session %s: %v", name, err)
		return
	}
	a.sessionComposer.SetSession(s)
}

func (a *App) saveSession() {
	s := a.sessionComposer.Session()
	if s == nil {
		return
	}
	if err := a.store.SaveSession(s); err != nil {
		log.Printf("save session: %v", err)
		return
	}
	a.sessionComposer.ClearModified()
}

func (a *App) showPdfExportDialog() {
	s := a.sessionComposer.Session()
	if s == nil {
		log.Printf("no session to generate PDF for")
		return
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("home dir: %v", err)
		return
	}
	outputDir := homeDir
	if ys, ok := a.store.(*store.YAMLStore); ok {
		if settings, err := ys.LoadSettings(); err == nil && settings.PdfExportDir != "" {
			outputDir = settings.PdfExportDir
		}
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
	defaultPath := filepath.Join(outputDir, filename+".pdf")

	go func() {
		cmd := exec.Command("zenity", "--file-selection", "--save", "--confirm-overwrite",
			"--filename="+defaultPath,
			"--file-filter=PDF | *.pdf")
		out, err := cmd.Output()
		if err != nil {
			return
		}
		path := strings.TrimSpace(string(out))
		if path == "" {
			return
		}
		if !strings.HasSuffix(strings.ToLower(path), ".pdf") {
			path += ".pdf"
		}
		select {
		case a.pendingPDFPath <- path:
		default:
		}
		if a.window != nil {
			a.window.Invalidate()
		}
	}()
}

func (a *App) generatePDFTo(path string) {
	s := a.sessionComposer.Session()
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
		// If local copy lacks i18n, try community library for translations.
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

// handleFileAction processes file toolbar button clicks.
func (a *App) handleFileAction(action uiwidget.FileAction) {
	switch action {
	case uiwidget.FileActionNew:
		a.NewExercise()
		a.editorState.MarkModified()
	case uiwidget.FileActionOpen:
		a.showOpenDialog()
	case uiwidget.FileActionSave:
		a.saveExercise()
	case uiwidget.FileActionDuplicate:
		a.duplicateExercise()
	case uiwidget.FileActionImport:
		a.showImportDialog()
	}
}

// NewExercise creates a blank exercise and sets it as current.
func (a *App) NewExercise() {
	ex := &model.Exercise{
		Name:          i18n.T("default.exercise_name"),
		CourtType:     model.HalfCourt,
		CourtStandard: model.FIBA,
		Sequences: []model.Sequence{
			{Label: i18n.T("default.sequence_label")},
		},
	}
	a.SetExercise(ex)
}

func (a *App) showOpenDialog() {
	names, err := a.store.ListExercises()
	if err != nil {
		log.Printf("list exercises: %v", err)
		return
	}
	a.exerciseList.Show(names)
}

func (a *App) openExercise(name string) {
	ex, err := a.store.LoadExercise(name)
	if err != nil {
		log.Printf("load exercise %s: %v", name, err)
		return
	}
	a.SetExercise(ex)
}

func (a *App) saveExercise() {
	if a.exercise == nil {
		return
	}
	if err := a.store.SaveExercise(a.exercise); err != nil {
		log.Printf("save exercise: %v", err)
		return
	}
	a.editorState.ClearModified()
	a.composerNeedsRefresh = true
	a.mgrNeedsRefresh = true
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
	if a.libraryOverlay != nil {
		a.libraryOverlay.Show(names)
	}
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
	// Save to user's store.
	if err := a.store.SaveExercise(ex); err != nil {
		log.Printf("save imported exercise: %v", err)
		return
	}
	// Open the imported exercise.
	a.SetExercise(ex)
	a.composerNeedsRefresh = true
	a.mgrNeedsRefresh = true
	log.Printf("imported exercise: %s", ex.Name)
}

func (a *App) duplicateExercise() {
	if a.exercise == nil {
		return
	}
	// Deep copy the exercise so the original is not mutated.
	dup := &model.Exercise{
		Name:          a.exercise.Name + i18n.T("file.copy_suffix"),
		Description:   a.exercise.Description,
		CourtType:     a.exercise.CourtType,
		CourtStandard: a.exercise.CourtStandard,
		Duration:      a.exercise.Duration,
		Intensity:     a.exercise.Intensity,
		Category:      a.exercise.Category,
	}
	dup.Tags = make([]string, len(a.exercise.Tags))
	copy(dup.Tags, a.exercise.Tags)
	dup.Sequences = make([]model.Sequence, len(a.exercise.Sequences))
	for i, seq := range a.exercise.Sequences {
		ns := model.Sequence{Label: seq.Label}
		ns.Instructions = make([]string, len(seq.Instructions))
		copy(ns.Instructions, seq.Instructions)
		ns.Players = make([]model.Player, len(seq.Players))
		copy(ns.Players, seq.Players)
		ns.Accessories = make([]model.Accessory, len(seq.Accessories))
		copy(ns.Accessories, seq.Accessories)
		ns.Actions = make([]model.Action, len(seq.Actions))
		copy(ns.Actions, seq.Actions)
		dup.Sequences[i] = ns
	}
	a.SetExercise(dup)
	a.editorState.MarkModified()
}

// --- Exercise Manager ---

func (a *App) layoutExerciseManager(gtx layout.Context) layout.Dimensions {
	if a.mgrNeedsRefresh {
		a.exerciseManager.SetExercises(a.buildManagedExercises())
		a.mgrNeedsRefresh = false
	}

	ev := a.exerciseManager.HandleActions(gtx)
	switch ev.Action {
	case uiwidget.MgrActionRefresh:
		a.exerciseManager.SetExercises(a.buildManagedExercises())
	case uiwidget.MgrActionOpen:
		a.openExercise(ev.Name)
		a.activeTab = 0
	case uiwidget.MgrActionImport:
		a.importExerciseByName(ev.Name)
	case uiwidget.MgrActionUpdate:
		a.updateExerciseFromRemote(ev.Name)
	case uiwidget.MgrActionContribute:
		a.contributeExercise(ev.Name)
	}

	return a.exerciseManager.Layout(gtx, a.theme)
}

func (a *App) buildManagedExercises() []uiwidget.ManagedExercise {
	localMap := make(map[string]*model.Exercise)
	remoteMap := make(map[string]*model.Exercise)
	allNames := make(map[string]bool)

	// Load local exercises.
	if names, err := a.store.ListExercises(); err == nil {
		for _, name := range names {
			if ex, err := a.store.LoadExercise(name); err == nil {
				localMap[name] = ex
				allNames[name] = true
			}
		}
	}

	// Load community exercises.
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

	// Build sorted list.
	sortedNames := make([]string, 0, len(allNames))
	for name := range allNames {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	lang := string(i18n.CurrentLang())
	items := make([]uiwidget.ManagedExercise, 0, len(sortedNames))
	for _, name := range sortedNames {
		local := localMap[name]
		remote := remoteMap[name]

		var status uiwidget.ExerciseSyncStatus
		switch {
		case local != nil && remote == nil:
			status = uiwidget.StatusLocalOnly
		case local == nil && remote != nil:
			status = uiwidget.StatusRemoteOnly
		case local != nil && remote != nil:
			if exercisesEqual(local, remote) {
				status = uiwidget.StatusSynced
			} else {
				status = uiwidget.StatusModified
			}
		}

		displayName := name
		category := ""
		duration := ""
		if local != nil {
			// Merge community i18n into local if missing.
			if local.I18n == nil && remote != nil && remote.I18n != nil {
				local.I18n = remote.I18n
			}
			loc := local.Localized(lang)
			displayName = loc.Name
			category = string(local.Category)
			duration = local.Duration
		} else if remote != nil {
			loc := remote.Localized(lang)
			displayName = loc.Name
			category = string(remote.Category)
			duration = remote.Duration
		}

		items = append(items, uiwidget.ManagedExercise{
			Name:        name,
			Status:      status,
			LocalEx:     local,
			RemoteEx:    remote,
			DisplayName: displayName,
			Category:    category,
			Duration:    duration,
		})
	}
	return items
}

// exercisesEqual compares two exercises by marshalling to YAML.
func exercisesEqual(a, b *model.Exercise) bool {
	dataA, errA := yaml.Marshal(a)
	dataB, errB := yaml.Marshal(b)
	if errA != nil || errB != nil {
		return false
	}
	return string(dataA) == string(dataB)
}

// importExerciseByName imports a community exercise by kebab-case name.
func (a *App) importExerciseByName(name string) {
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
	a.mgrNeedsRefresh = true
	a.composerNeedsRefresh = true
	log.Printf("imported exercise: %s", ex.Name)
}

// updateExerciseFromRemote overwrites local with community version.
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
	// If the editor is showing this exercise, reload it.
	if a.exercise != nil && store.ToKebab(a.exercise.Name) == name {
		a.SetExercise(ex)
	}
	a.mgrNeedsRefresh = true
	a.composerNeedsRefresh = true
	log.Printf("updated exercise from community: %s", name)
}

// contributeExercise saves YAML to contribute dir and opens GitHub.
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

	// Save to ~/.courtdraw/contribute/.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("home dir: %v", err)
		return
	}
	contributeDir := filepath.Join(homeDir, ".courtdraw", "contribute")
	os.MkdirAll(contributeDir, 0755)
	outPath := filepath.Join(contributeDir, name+".yaml")
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		log.Printf("write contribute file: %v", err)
		return
	}
	log.Printf("saved contribution: %s", outPath)

	// Build GitHub new-file URL.
	filename := name + ".yaml"
	encoded := url.QueryEscape(string(data))
	ghURL := fmt.Sprintf("https://github.com/darkweaver87/courtdraw/new/main/library?filename=%s&value=%s", filename, encoded)

	// GitHub URLs have a practical limit; if too long, just open the base.
	if len(ghURL) > 8000 {
		ghURL = "https://github.com/darkweaver87/courtdraw/new/main/library?filename=" + filename
	}

	if err := openBrowser(ghURL); err != nil {
		log.Printf("open browser: %v", err)
	}
}

// stripDiacritics removes combining marks (accents) from a string.
// e.g. "séance" → "seance", "éàü" → "eau".
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
			// Persist asynchronously — best effort.
			if ys, ok := a.store.(*store.YAMLStore); ok {
				settings, _ := ys.LoadSettings()
				settings.Language = string(next)
				ys.SaveSettings(settings)
			}
			return
		}
	}
}
