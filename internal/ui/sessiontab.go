package ui

import (
	"fmt"
	"image/color"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/anim"
	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/ui/fynecourt"
	"github.com/darkweaver87/courtdraw/internal/ui/icon"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
)

// ExerciseSyncStatus represents the sync state of a managed exercise.
type ExerciseSyncStatus int

const (
	StatusLocalOnly  ExerciseSyncStatus = iota
	StatusRemoteOnly
	StatusSynced
	StatusModified
)

// ManagedExercise is an entry in the exercise library list.
type ManagedExercise struct {
	Name              string
	Status            ExerciseSyncStatus
	LocalEx           *model.Exercise
	RemoteEx          *model.Exercise
	DisplayName       string
	RemoteDisplayName string
	Category          string
	AgeGroup          string
	CourtType         string
	Duration          string
	Tags              []string
	RemoteTags        []string
}

// EffectiveDisplayName returns the display name based on the filter index.
// filterIndex 1 (local) uses the local display name; otherwise remote takes priority.
func (m ManagedExercise) EffectiveDisplayName(filterIndex int) string {
	if filterIndex != 1 && m.RemoteDisplayName != "" {
		return m.RemoteDisplayName
	}
	return m.DisplayName
}

// EffectiveTags returns the tags based on the filter index.
func (m ManagedExercise) EffectiveTags(filterIndex int) []string {
	if filterIndex != 1 && len(m.RemoteTags) > 0 {
		return m.RemoteTags
	}
	return m.Tags
}

const maxMgrItems = 200
const maxSessionItems = 50

// SessionTabAction represents an action the user performed in the session tab.
type SessionTabAction int

const (
	SessionTabActionNone           SessionTabAction = iota
	SessionTabActionNew
	SessionTabActionOpen
	SessionTabActionSave
	SessionTabActionGenerate
	SessionTabActionRefresh
	SessionTabActionOpenExercise
	SessionTabActionUpdate
	SessionTabActionContribute
	SessionTabActionDeleteExercise
	SessionTabActionDeleteSession
	SessionTabActionRecent
	SessionTabActionTraining
	SessionTabActionShare
	SessionTabActionImportBundle
)

// SessionTabEvent is returned when the user performs an action.
type SessionTabEvent struct {
	Action SessionTabAction
	Name   string
}

// SessionTab merges the exercise library and session composer into a single 3-column widget.
type SessionTab struct {
	box *fyne.Container

	// Library column.
	items           []ManagedExercise
	searchEntry     *widget.Entry
	filterIndex     int
	filterCategory  model.Category
	filterCourtType model.CourtType
	filterTags      []string
	libraryBox      *fyne.Container
	libraryScroll   *container.Scroll
	selectedIdx     int
	categorySelect  *widget.Select
	courtTypeSelect *widget.Select
	tagChecks       *widget.CheckGroup
	tagScroll       *container.Scroll

	// Filter buttons for language refresh.
	filterBtns [3]*widget.Button

	// Toolbar buttons (stored as fields for reuse across layouts).
	newBtn        *TipButton
	openBtn       *TipButton
	recentBtn     *TipButton
	saveBtn       *TipButton
	genBtn        *TipButton
	refreshBtn    *TipButton
	addBtn        *TipButton
	openExBtn     *TipButton
	contributeBtn *TipButton
	trainingBtn   *TipButton
	shareBtn      *TipButton
	importBtn     *TipButton

	// Session column.
	session            *model.Session
	resolvedExercises  map[string]*model.Exercise
	modified           bool
	sessionList        *DragList
	titleEntry     *widget.Entry
	dateEntry      *widget.Entry
	subtitleEntry  *widget.Entry
	ageGroupEntry  *widget.Entry
	philosophyEntry *widget.Entry

	// Total duration.
	totalLabel *canvas.Text

	// Preview column.
	previewCourt  *fynecourt.CourtWidget
	previewPB     *anim.Playback
	previewLabel  *canvas.Text
	previewName   string // name of currently previewed exercise (avoid restart)
	previewDetail *fyne.Container // exercise details (duration, intensity, instructions)

	// Event channel — consumed by App each frame.
	pendingEvent SessionTabEvent

	// Session overlay.
	sessionOverlay *SessionListOverlay
	responsive     *ResponsiveContainer

	OnAction         func(SessionTabEvent)
	OnSessionChanged func() // called when exercises are added/removed/reordered
	OnStatus         func(string, int)
	LoadExercise     func(name string) (*model.Exercise, error) // fallback loader
}

// NewSessionTab creates a new session tab.
func NewSessionTab() *SessionTab {
	st := &SessionTab{
		selectedIdx: -1,
	}

	// Search.
	st.searchEntry = widget.NewEntry()
	st.searchEntry.SetPlaceHolder(i18n.T("session.search"))
	st.searchEntry.OnChanged = func(string) {
		st.refreshLibraryList()
	}

	// Library list (VBox + scroll for variable row heights).
	st.libraryBox = container.NewVBox()
	st.libraryScroll = container.NewVScroll(st.libraryBox)

	// Session metadata entries.
	st.titleEntry = widget.NewEntry()
	st.titleEntry.SetPlaceHolder(i18n.T("session.title"))
	st.titleEntry.OnChanged = func(s string) {
		if st.session != nil {
			st.session.Title = s
			st.modified = true
		}
	}
	st.dateEntry = widget.NewEntry()
	st.dateEntry.SetPlaceHolder("YYYY-MM-DD")
	st.dateEntry.OnChanged = func(s string) {
		if st.session != nil {
			st.session.Date = s
			st.modified = true
		}
	}
	st.subtitleEntry = widget.NewEntry()
	st.subtitleEntry.SetPlaceHolder(i18n.T("session.subtitle"))
	st.subtitleEntry.OnChanged = func(s string) {
		if st.session != nil {
			st.session.Subtitle = s
			st.modified = true
		}
	}
	st.ageGroupEntry = widget.NewEntry()
	st.ageGroupEntry.SetPlaceHolder(i18n.T("session.age_group"))
	st.ageGroupEntry.OnChanged = func(s string) {
		if st.session != nil {
			st.session.AgeGroup = s
			st.modified = true
		}
	}
	st.philosophyEntry = widget.NewMultiLineEntry()
	st.philosophyEntry.SetPlaceHolder(i18n.T("session.philosophy"))
	st.philosophyEntry.OnChanged = func(s string) {
		if st.session != nil {
			st.session.Philosophy = s
			st.modified = true
		}
	}

	// Preview detail container (exercise metadata + instructions).
	st.previewDetail = container.NewVBox()

	// Session exercise list with drag-and-drop reordering.
	st.sessionList = NewDragList()
	st.sessionList.OnReorder = func(from, to int) {
		st.applyReorder(from, to)
	}
	st.sessionList.OnDelete = func(idx int) {
		st.removeExercise(idx)
	}
	st.sessionList.OnMove = func(idx, dir int) {
		st.moveExercise(idx, dir)
	}
	st.sessionList.OnPreview = func(idx int) {
		st.previewSessionExercise(idx)
	}

	// Category filter.
	st.categorySelect = widget.NewSelect(st.buildCategoryOptions(), func(s string) {
		st.filterCategory = st.categoryKeyFromLabel(s)
		st.refreshLibraryList()
	})
	st.categorySelect.SetSelected(st.buildCategoryOptions()[0])

	// Court type filter.
	st.courtTypeSelect = widget.NewSelect(st.buildCourtTypeOptions(), func(s string) {
		st.filterCourtType = st.courtTypeKeyFromLabel(s)
		st.refreshLibraryList()
	})
	st.courtTypeSelect.SetSelected(st.buildCourtTypeOptions()[0])

	// Tag filter (multi-select with CheckGroup).
	st.tagChecks = widget.NewCheckGroup(nil, func(selected []string) {
		st.filterTags = selected
		st.refreshLibraryList()
	})
	st.tagChecks.Horizontal = true
	st.tagScroll = container.NewHScroll(st.tagChecks)
	st.tagScroll.SetMinSize(fyne.NewSize(0, 36))

	// Total duration label.
	st.totalLabel = canvas.NewText("", color.NRGBA{R: 0xf4, G: 0xa2, B: 0x61, A: 0xff})
	st.totalLabel.TextSize = 12
	st.totalLabel.TextStyle.Bold = true

	// Preview court.
	st.previewCourt = fynecourt.NewCourtWidget()
	st.previewLabel = canvas.NewText(i18n.T("session.select_to_preview"), color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff})
	st.previewLabel.TextSize = 14
	st.previewLabel.Alignment = fyne.TextAlignCenter

	// Session overlay.
	st.sessionOverlay = NewSessionListOverlay()

	st.buildLayout()
	return st
}

func (st *SessionTab) buildLayout() {
	// Toolbar buttons — icon + tooltip.
	st.newBtn = NewTipButton(icon.New(), i18n.T("session.new"), func() { st.emitAction(SessionTabActionNew, "") })
	st.openBtn = NewTipButton(icon.Open(), i18n.T("session.open"), func() { st.emitAction(SessionTabActionOpen, "") })
	st.recentBtn = NewTipButton(icon.Refresh(), i18n.T("session.recent"), func() { st.emitAction(SessionTabActionRecent, "") })
	st.saveBtn = NewTipButton(icon.Save(), i18n.T("session.save"), func() { st.emitAction(SessionTabActionSave, "") })
	st.genBtn = NewTipButton(fynetheme.DocumentPrintIcon(), i18n.T("session.generate_pdf"), func() { st.emitAction(SessionTabActionGenerate, "") })
	st.refreshBtn = NewTipButton(icon.Refresh(), i18n.T("mgr.refresh"), func() { st.emitAction(SessionTabActionRefresh, "") })
	st.trainingBtn = NewTipButton(icon.Training(), i18n.T("tooltip.training"), func() { st.emitAction(SessionTabActionTraining, "") })
	st.trainingBtn.SetImportance(widget.HighImportance)
	st.shareBtn = NewTipButton(icon.Share(), i18n.T("share.button"), func() { st.emitAction(SessionTabActionShare, "") })
	st.importBtn = NewTipButton(icon.Import(), i18n.T("import.bundle"), func() { st.emitAction(SessionTabActionImportBundle, "") })

	// Add to session button.
	st.addBtn = NewTipButton(fynetheme.NavigateNextIcon(), i18n.T("session.add_to_session"), func() {
		filtered := st.filteredExercises()
		if st.selectedIdx >= 0 && st.selectedIdx < len(filtered) {
			st.addExerciseByRef(filtered[st.selectedIdx].Name)
		}
	})
	st.addBtn.SetImportance(widget.MediumImportance)

	// Action buttons for selected exercise.
	st.openExBtn = NewTipButton(icon.Open(), i18n.T("session.open_exercise"), func() {
		filtered := st.filteredExercises()
		if st.selectedIdx >= 0 && st.selectedIdx < len(filtered) {
			st.emitAction(SessionTabActionOpenExercise, filtered[st.selectedIdx].Name)
		}
	})
	st.contributeBtn = NewTipButton(icon.Upload(), i18n.T("mgr.contribute"), func() {
		filtered := st.filteredExercises()
		if st.selectedIdx >= 0 && st.selectedIdx < len(filtered) {
			st.emitAction(SessionTabActionContribute, filtered[st.selectedIdx].Name)
		}
	})

	// Tooltips above for buttons at bottom of preview column.
	st.addBtn.TooltipAbove = true
	st.openExBtn.TooltipAbove = true
	st.contributeBtn.TooltipAbove = true

	// Library column.
	st.filterBtns[0] = widget.NewButton(i18n.T("filter.all"), func() { st.filterIndex = 0; st.refreshLibraryList() })
	st.filterBtns[1] = widget.NewButton(i18n.T("filter.local"), func() { st.filterIndex = 1; st.refreshLibraryList() })
	st.filterBtns[2] = widget.NewButton(i18n.T("filter.community"), func() { st.filterIndex = 2; st.refreshLibraryList() })
	for _, btn := range st.filterBtns {
		btn.Importance = widget.LowImportance
	}

	bg := canvas.NewRectangle(theme.ColorDarkBg)
	st.responsive = NewResponsiveContainer(st.buildDesktopLayout, st.buildMobileLayout)
	st.box = container.NewStack(bg, st.responsive)
}

func (st *SessionTab) buildDesktopLayout() fyne.CanvasObject {
	statusRow := container.NewGridWithColumns(3, st.filterBtns[0], st.filterBtns[1], st.filterBtns[2])
	filterGrid := container.NewGridWithColumns(2, st.categorySelect, st.courtTypeSelect)
	searchRow := container.NewBorder(nil, nil, nil, st.refreshBtn, st.searchEntry)

	libraryCol := container.NewBorder(
		container.NewVBox(searchRow, statusRow, filterGrid, st.tagScroll),
		nil, nil, nil,
		st.libraryScroll,
	)

	detailScroll := container.NewVScroll(st.previewDetail)
	detailScroll.SetMinSize(fyne.NewSize(0, 80))
	courtAndDetail := container.NewVSplit(st.previewCourt, detailScroll)
	courtAndDetail.SetOffset(0.6)
	previewCol := container.NewBorder(
		st.previewLabel,
		container.NewVBox(st.addBtn, container.NewGridWithColumns(2, st.openExBtn, st.contributeBtn)),
		nil, nil,
		courtAndDetail,
	)

	metadataForm := container.NewVBox(
		st.titleEntry, st.dateEntry, st.subtitleEntry, st.ageGroupEntry,
	)
	sessionToolbar := container.NewHBox(st.newBtn, st.openBtn, st.recentBtn, st.saveBtn, st.genBtn, st.shareBtn, st.importBtn, st.trainingBtn, layout.NewSpacer())
	sessionCol := container.NewBorder(
		container.NewVBox(sessionToolbar, metadataForm),
		container.NewVBox(container.NewPadded(st.totalLabel), st.philosophyEntry),
		nil, nil,
		st.sessionList,
	)

	rightSplit := container.NewHSplit(previewCol, sessionCol)
	rightSplit.SetOffset(0.5)
	mainSplit := container.NewHSplit(libraryCol, rightSplit)
	mainSplit.SetOffset(0.3)

	return mainSplit
}

func (st *SessionTab) buildMobileLayout() fyne.CanvasObject {
	// Library tab.
	statusRow := container.NewGridWithColumns(3, st.filterBtns[0], st.filterBtns[1], st.filterBtns[2])
	filterGrid := container.NewGridWithColumns(2, st.categorySelect, st.courtTypeSelect)
	searchRow := container.NewBorder(nil, nil, nil, st.refreshBtn, st.searchEntry)
	libraryTab := container.NewBorder(
		container.NewVBox(searchRow, statusRow, filterGrid, st.tagScroll),
		nil, nil, nil,
		st.libraryScroll,
	)

	// Preview tab.
	mobileDetailScroll := container.NewVScroll(st.previewDetail)
	mobileDetailScroll.SetMinSize(fyne.NewSize(0, 80))
	mobileCourtAndDetail := container.NewVSplit(st.previewCourt, mobileDetailScroll)
	mobileCourtAndDetail.SetOffset(0.6)
	previewTab := container.NewBorder(
		st.previewLabel,
		container.NewVBox(st.addBtn, container.NewGridWithColumns(2, st.openExBtn, st.contributeBtn)),
		nil, nil,
		mobileCourtAndDetail,
	)

	// Session tab.
	metadataForm := container.NewVBox(
		st.titleEntry, st.dateEntry, st.subtitleEntry, st.ageGroupEntry,
	)
	sessionToolbar := container.NewHBox(st.newBtn, st.openBtn, st.recentBtn, st.saveBtn, st.genBtn, st.shareBtn, st.importBtn, st.trainingBtn, layout.NewSpacer())
	sessionTab := container.NewBorder(
		container.NewVBox(sessionToolbar, metadataForm),
		container.NewVBox(container.NewPadded(st.totalLabel), st.philosophyEntry),
		nil, nil,
		st.sessionList,
	)

	tabs := container.NewAppTabs(
		container.NewTabItem(i18n.T("mobile.session.library"), libraryTab),
		container.NewTabItem(i18n.T("mobile.session.preview"), previewTab),
		container.NewTabItem(i18n.T("mobile.session.session"), sessionTab),
	)
	tabs.SetTabLocation(container.TabLocationBottom)
	return tabs
}

// Widget returns the session tab widget.
func (st *SessionTab) Widget() fyne.CanvasObject {
	return st.box
}

// SetExercises sets the managed exercises list.
func (st *SessionTab) SetExercises(items []ManagedExercise) {
	st.items = items
	st.rebuildTagOptions()
	st.refreshLibraryList()
}

func (st *SessionTab) rebuildTagOptions() {
	tagSet := make(map[string]bool)
	for _, item := range st.items {
		for _, t := range item.Tags {
			tagSet[t] = true
		}
	}
	sorted := make([]string, 0, len(tagSet))
	for t := range tagSet {
		sorted = append(sorted, t)
	}
	sort.Strings(sorted)
	st.tagChecks.Options = sorted
	// Preserve selected tags that still exist.
	var valid []string
	for _, ft := range st.filterTags {
		if tagSet[ft] {
			valid = append(valid, ft)
		}
	}
	st.tagChecks.Selected = valid
	st.filterTags = valid
	st.tagChecks.Refresh()
}

// SetSession sets the current session.
func (st *SessionTab) SetSession(s *model.Session) {
	st.session = s
	st.modified = false
	if s != nil {
		st.titleEntry.SetText(s.Title)
		st.dateEntry.SetText(s.Date)
		st.subtitleEntry.SetText(s.Subtitle)
		st.ageGroupEntry.SetText(s.AgeGroup)
		st.philosophyEntry.SetText(s.Philosophy)
	}
	st.refreshSessionList()
	st.refreshTotal()
}

// Session returns the current session.
func (st *SessionTab) Session() *model.Session {
	return st.session
}

// Modified returns whether the session has unsaved changes.
func (st *SessionTab) Modified() bool {
	return st.modified
}

// ClearModified clears the modified flag.
func (st *SessionTab) ClearModified() {
	st.modified = false
}

// SetResolvedExercises sets the map of exercise name → exercise for display.
func (st *SessionTab) SetResolvedExercises(m map[string]*model.Exercise) {
	st.resolvedExercises = m
	st.refreshSessionList()
	st.refreshTotal()
}

// SessionListOverlay returns the session list overlay for open/recent dialogs.
func (st *SessionTab) SessionListOverlay() *SessionListOverlay {
	return st.sessionOverlay
}

func (st *SessionTab) filteredExercises() []ManagedExercise {
	var result []ManagedExercise
	search := strings.ToLower(st.searchEntry.Text)
	for _, item := range st.items {
		// Status filter.
		switch st.filterIndex {
		case 1: // Local only
			if item.Status == StatusRemoteOnly {
				continue
			}
		case 2: // Community only
			if item.Status == StatusLocalOnly {
				continue
			}
		}
		// Category filter.
		if st.filterCategory != "" && model.Category(item.Category) != st.filterCategory {
			continue
		}
		// Court type filter.
		if st.filterCourtType != "" && model.CourtType(item.CourtType) != st.filterCourtType {
			continue
		}
		// Tag filter (intersection: all selected tags must be present).
		if len(st.filterTags) > 0 {
			match := true
			for _, ft := range st.filterTags {
				found := false
				for _, t := range item.EffectiveTags(st.filterIndex) {
					if t == ft {
						found = true
						break
					}
				}
				if !found {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}
		// Search filter.
		if search != "" && !strings.Contains(strings.ToLower(item.EffectiveDisplayName(st.filterIndex)), search) {
			continue
		}
		result = append(result, item)
	}
	return result
}

func (st *SessionTab) refreshLibraryList() {
	st.libraryBox.RemoveAll()
	filtered := st.filteredExercises()
	for i, ex := range filtered {
		idx := i
		mgd := ex

		displayName := mgd.EffectiveDisplayName(st.filterIndex)
		if displayName == "" {
			displayName = mgd.Name
		}
		nameLabel := widget.NewLabel(displayName)
		nameLabel.Wrapping = fyne.TextWrapWord

		catText := canvas.NewText(mgd.Category, color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff})
		catText.TextSize = 11

		eyeBtn := widget.NewButtonWithIcon("", icon.Preview(), func() {
			st.selectedIdx = idx
			st.updatePreview()
		})
		eyeBtn.Importance = widget.LowImportance

		bg := canvas.NewRectangle(color.Transparent)
		tap := newTappableArea(func() {
			st.selectedIdx = idx
			st.updatePreview()
		})

		row := container.NewBorder(nil, nil, nil, container.NewHBox(catText, eyeBtn), nameLabel)
		st.libraryBox.Add(container.NewStack(bg, tap, row))
	}
	if len(filtered) == 0 {
		st.libraryBox.Add(widget.NewLabel(i18n.T("session.no_exercises")))
	}
	st.libraryBox.Refresh()
}

func (st *SessionTab) updatePreview() {
	filtered := st.filteredExercises()
	if st.selectedIdx < 0 || st.selectedIdx >= len(filtered) {
		st.previewLabel.Text = i18n.T("session.select_to_preview")
		st.previewLabel.Refresh()
		st.previewCourt.SetAnimMode(false)
		st.previewCourt.SetExercise(nil)
		st.previewPB = nil
		st.previewName = ""
		st.previewDetail.RemoveAll()
		st.previewDetail.Refresh()
		return
	}
	mgd := filtered[st.selectedIdx]

	// Skip restart if already previewing the same exercise.
	if mgd.Name == st.previewName {
		return
	}

	st.previewLabel.Text = mgd.EffectiveDisplayName(st.filterIndex)
	st.previewLabel.Refresh()
	st.previewName = mgd.Name

	// Restore library action buttons.
	st.addBtn.Show()
	st.openExBtn.Show()
	st.contributeBtn.Show()

	// Load the exercise for the preview court.
	// When filtering community or all, prefer remote exercise for accurate i18n.
	var rawEx *model.Exercise
	if st.filterIndex != 1 && mgd.RemoteEx != nil {
		rawEx = mgd.RemoteEx
	} else if mgd.LocalEx != nil {
		rawEx = mgd.LocalEx
	} else if mgd.RemoteEx != nil {
		rawEx = mgd.RemoteEx
	}
	if rawEx == nil {
		st.previewCourt.SetAnimMode(false)
		st.previewCourt.SetExercise(nil)
		st.previewPB = nil
		st.previewDetail.RemoveAll()
		st.previewDetail.Refresh()
		return
	}

	// Localize exercise for display.
	ex := rawEx.Localized(string(i18n.CurrentLang()))

	st.previewCourt.SetExercise(ex)
	if len(ex.Sequences) > 1 {
		st.previewPB = anim.NewPlayback(ex)
		st.previewCourt.SetPlayback(st.previewPB)
		st.previewPB.Play()
		st.previewCourt.SetAnimMode(true)
	} else {
		st.previewCourt.SetAnimMode(false)
		st.previewPB = nil
	}

	st.populatePreviewDetail(ex)
}

func (st *SessionTab) previewSessionExercise(idx int) {
	if st.session == nil || idx < 0 || idx >= len(st.session.Exercises) {
		return
	}
	name := st.session.Exercises[idx].Exercise
	lang := string(i18n.CurrentLang())

	// Try resolved exercises first, then fallback to direct load.
	ex, ok := st.resolvedExercises[name]
	if !ok || ex == nil {
		if st.LoadExercise != nil {
			raw, err := st.LoadExercise(name)
			if err != nil {
				return
			}
			ex = raw.Localized(lang)
		} else {
			return
		}
	}

	// Deselect library list to avoid confusion.
	// No library selection to clear (VBox-based list).
	st.selectedIdx = -1
	st.previewName = "session:" + name

	st.previewLabel.Text = ex.Name
	st.previewLabel.Refresh()

	// Hide action buttons that don't apply to session exercises.
	st.contributeBtn.Hide()
	st.addBtn.Hide()
	st.openExBtn.Show()

	// Load court preview.
	st.previewCourt.SetExercise(ex)
	if len(ex.Sequences) > 1 {
		st.previewPB = anim.NewPlayback(ex)
		st.previewCourt.SetPlayback(st.previewPB)
		st.previewPB.Play()
		st.previewCourt.SetAnimMode(true)
	} else {
		st.previewCourt.SetAnimMode(false)
		st.previewPB = nil
	}

	st.populatePreviewDetail(ex)
}

func (st *SessionTab) populatePreviewDetail(ex *model.Exercise) {
	st.previewDetail.RemoveAll()

	// Metadata row with colored intensity dots.
	var metaParts []fyne.CanvasObject
	if ex.Duration != "" {
		lbl := canvas.NewText(i18n.T("props.duration")+": "+ex.Duration, color.White)
		lbl.TextSize = 11
		lbl.TextStyle.Bold = true
		metaParts = append(metaParts, lbl)
	}
	if ex.Intensity > 0 {
		intLabel := canvas.NewText(i18n.T("props.intensity")+": ", color.White)
		intLabel.TextSize = 11
		intLabel.TextStyle.Bold = true
		dots := intensityColorDots(int(ex.Intensity))
		metaParts = append(metaParts, container.NewHBox(append([]fyne.CanvasObject{intLabel}, dots...)...))
	}
	if ex.Category != "" {
		lbl := canvas.NewText(i18n.T("props.category")+": "+i18n.T("category."+string(ex.Category)), color.White)
		lbl.TextSize = 11
		lbl.TextStyle.Bold = true
		metaParts = append(metaParts, lbl)
	}
	if len(metaParts) > 0 {
		st.previewDetail.Add(container.NewHBox(metaParts...))
	}

	// Description.
	if ex.Description != "" {
		descLabel := widget.NewLabel(ex.Description)
		descLabel.Wrapping = fyne.TextWrapWord
		st.previewDetail.Add(descLabel)
	}

	// Instructions per sequence.
	for si, seq := range ex.Sequences {
		if len(seq.Instructions) == 0 {
			continue
		}
		label := seq.Label
		if label == "" {
			label = fmt.Sprintf(i18n.T("seq.format"), si+1)
		}
		seqTitle := widget.NewRichTextFromMarkdown("**" + label + "**")
		st.previewDetail.Add(seqTitle)

		for _, instr := range seq.Instructions {
			instrLabel := widget.NewLabel("  - " + instr)
			instrLabel.Wrapping = fyne.TextWrapWord
			st.previewDetail.Add(instrLabel)
		}
	}

	st.previewDetail.Refresh()
}

func (st *SessionTab) addExerciseByRef(name string) {
	if st.session == nil {
		return
	}
	if len(st.session.Exercises) >= maxSessionItems {
		return
	}
	st.session.Exercises = append(st.session.Exercises, model.ExerciseEntry{Exercise: name})
	st.modified = true
	st.refreshSessionList()
	st.notifySessionChanged()
	if st.OnStatus != nil {
		st.OnStatus(fmt.Sprintf(i18n.T("status.exercise_added"), name), 0)
	}
}

func (st *SessionTab) refreshSessionList() {
	if st.session == nil {
		st.sessionList.SetItems(nil)
		return
	}
	lang := string(i18n.CurrentLang())
	items := make([]DragListItem, len(st.session.Exercises))
	for i, entry := range st.session.Exercises {
		displayName := entry.Exercise
		if ex, ok := st.resolvedExercises[entry.Exercise]; ok {
			displayName = ex.Name
		} else if st.LoadExercise != nil {
			if raw, err := st.LoadExercise(entry.Exercise); err == nil {
				loc := raw.Localized(lang)
				displayName = loc.Name
			}
		}
		items[i] = DragListItem{Text: fmt.Sprintf("%d. %s", i+1, displayName)}
	}
	st.sessionList.SetItems(items)
}

func (st *SessionTab) applyReorder(from, to int) {
	if st.session == nil {
		return
	}
	exs := st.session.Exercises
	if from < 0 || from >= len(exs) || to < 0 || to >= len(exs) {
		return
	}
	item := exs[from]
	// Remove from old position.
	exs = append(exs[:from], exs[from+1:]...)
	// Insert at new position.
	exs = append(exs[:to], append([]model.ExerciseEntry{item}, exs[to:]...)...)
	st.session.Exercises = exs
	st.modified = true
	st.refreshSessionList()
	st.notifySessionChanged()
}

func (st *SessionTab) moveExercise(idx, dir int) {
	if st.session == nil {
		return
	}
	newIdx := idx + dir
	if newIdx < 0 || newIdx >= len(st.session.Exercises) {
		return
	}
	st.session.Exercises[idx], st.session.Exercises[newIdx] = st.session.Exercises[newIdx], st.session.Exercises[idx]
	st.modified = true
	st.refreshSessionList()
}

func (st *SessionTab) removeExercise(idx int) {
	if st.session == nil || idx >= len(st.session.Exercises) {
		return
	}
	name := st.session.Exercises[idx].Exercise
	st.session.Exercises = append(st.session.Exercises[:idx], st.session.Exercises[idx+1:]...)
	st.modified = true
	st.refreshSessionList()
	st.notifySessionChanged()
	if st.OnStatus != nil {
		st.OnStatus(fmt.Sprintf(i18n.T("status.exercise_removed"), name), 0)
	}
}

func (st *SessionTab) notifySessionChanged() {
	if st.OnSessionChanged != nil {
		st.OnSessionChanged()
	}
	st.refreshTotal()
}

func (st *SessionTab) refreshTotal() {
	st.totalLabel.Text = fmt.Sprintf(i18n.T("session.total_format"), st.computeTotalDuration())
	st.totalLabel.Refresh()

	// Show/hide training and share buttons based on session exercises.
	hasExercises := st.session != nil && len(st.session.Exercises) > 0
	if st.trainingBtn != nil {
		if hasExercises {
			st.trainingBtn.Show()
		} else {
			st.trainingBtn.Hide()
		}
	}
	if st.shareBtn != nil {
		if hasExercises {
			st.shareBtn.Show()
		} else {
			st.shareBtn.Hide()
		}
	}
}

func (st *SessionTab) emitAction(action SessionTabAction, name string) {
	if st.OnAction != nil {
		st.OnAction(SessionTabEvent{Action: action, Name: name})
	}
}

func (st *SessionTab) buildCategoryOptions() []string {
	return []string{
		i18n.T("session.category_all"),
		i18n.T("category." + string(model.CategoryWarmup)),
		i18n.T("category." + string(model.CategoryOffense)),
		i18n.T("category." + string(model.CategoryDefense)),
		i18n.T("category." + string(model.CategoryTransition)),
		i18n.T("category." + string(model.CategoryScrimmage)),
		i18n.T("category." + string(model.CategoryCooldown)),
	}
}

var categoryKeys = []model.Category{
	"", model.CategoryWarmup, model.CategoryOffense, model.CategoryDefense,
	model.CategoryTransition, model.CategoryScrimmage, model.CategoryCooldown,
}

func (st *SessionTab) categoryKeyFromLabel(s string) model.Category {
	opts := st.buildCategoryOptions()
	for i, label := range opts {
		if label == s && i < len(categoryKeys) {
			return categoryKeys[i]
		}
	}
	return ""
}

func (st *SessionTab) buildCourtTypeOptions() []string {
	return []string{
		i18n.T("session.category_all"),
		i18n.T("court_type." + string(model.HalfCourt)),
		i18n.T("court_type." + string(model.FullCourt)),
	}
}

var courtTypeKeys = []model.CourtType{"", model.HalfCourt, model.FullCourt}

func (st *SessionTab) courtTypeKeyFromLabel(s string) model.CourtType {
	opts := st.buildCourtTypeOptions()
	for i, label := range opts {
		if label == s && i < len(courtTypeKeys) {
			return courtTypeKeys[i]
		}
	}
	return ""
}

// RefreshLanguage updates all translatable text in the session tab.
func (st *SessionTab) RefreshLanguage() {
	st.searchEntry.SetPlaceHolder(i18n.T("session.search"))
	st.titleEntry.SetPlaceHolder(i18n.T("session.title"))
	st.subtitleEntry.SetPlaceHolder(i18n.T("session.subtitle"))
	st.ageGroupEntry.SetPlaceHolder(i18n.T("session.age_group"))
	st.philosophyEntry.SetPlaceHolder(i18n.T("session.philosophy"))

	// Filter buttons.
	filterKeys := [3]string{"filter.all", "filter.local", "filter.community"}
	for i, key := range filterKeys {
		st.filterBtns[i].SetText(i18n.T(key))
	}

	// Category select.
	st.categorySelect.Options = st.buildCategoryOptions()
	// Re-select to update displayed label.
	for i, key := range categoryKeys {
		if key == st.filterCategory {
			st.categorySelect.SetSelected(st.buildCategoryOptions()[i])
			break
		}
	}

	// Court type select.
	st.courtTypeSelect.Options = st.buildCourtTypeOptions()
	for i, key := range courtTypeKeys {
		if key == st.filterCourtType {
			st.courtTypeSelect.SetSelected(st.buildCourtTypeOptions()[i])
			break
		}
	}

	// Preview placeholder.
	if st.selectedIdx < 0 {
		st.previewLabel.Text = i18n.T("session.select_to_preview")
		st.previewLabel.Refresh()
	}

	// Toolbar button tooltips.
	st.newBtn.SetTooltip(i18n.T("session.new"))
	st.openBtn.SetTooltip(i18n.T("session.open"))
	st.recentBtn.SetTooltip(i18n.T("session.recent"))
	st.saveBtn.SetTooltip(i18n.T("session.save"))
	st.genBtn.SetTooltip(i18n.T("session.generate_pdf"))
	st.refreshBtn.SetTooltip(i18n.T("mgr.refresh"))
	st.addBtn.SetTooltip(i18n.T("session.add_to_session"))
	st.openExBtn.SetTooltip(i18n.T("session.open_exercise"))
	st.contributeBtn.SetTooltip(i18n.T("mgr.contribute"))
	st.trainingBtn.SetTooltip(i18n.T("tooltip.training"))
	st.shareBtn.SetTooltip(i18n.T("share.button"))
	st.importBtn.SetTooltip(i18n.T("import.bundle"))

	// Rebuild responsive layout (mobile tab labels).
	if st.responsive != nil {
		st.responsive.ForceRebuild()
	}

	st.refreshTotal()
}

// computeTotalDuration calculates the sum of all exercise durations.
func (st *SessionTab) computeTotalDuration() string {
	if st.session == nil || len(st.session.Exercises) == 0 {
		return "N/A"
	}
	total := 0
	for _, entry := range st.session.Exercises {
		if ex, ok := st.resolvedExercises[entry.Exercise]; ok {
			total += parseDurationMinutes(ex.Duration)
		}
	}
	if total == 0 {
		return "N/A"
	}
	h := total / 60
	m := total % 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh%dm", h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}

// parseDurationMinutes parses a duration string like "1h30m" into minutes.
func parseDurationMinutes(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	total := 0
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else if c == 'h' || c == 'H' {
			total += n * 60
			n = 0
		} else if c == 'm' || c == 'M' {
			total += n
			n = 0
		}
	}
	return total
}

// intensityColorDots returns colored dot characters as canvas.Text objects.
func intensityColorDots(level int) []fyne.CanvasObject {
	colors := [3]color.NRGBA{
		{R: 76, G: 175, B: 80, A: 255},  // green
		{R: 255, G: 193, B: 7, A: 255},  // yellow
		{R: 244, G: 67, B: 54, A: 255},  // red
	}
	off := color.NRGBA{R: 120, G: 120, B: 120, A: 255}
	dots := make([]fyne.CanvasObject, 3)
	for i := 0; i < 3; i++ {
		c := off
		if i < level {
			c = colors[i]
		}
		dot := canvas.NewText("●", c)
		dot.TextSize = 14
		dots[i] = dot
	}
	return dots
}

// --- SessionListOverlay ---

// SessionListOverlay shows a list of sessions for open/recent.
type SessionListOverlay struct {
	Visible    bool
	names      []string
	recentMode bool

	OnSelect string
	OnDelete string
	OnRemove string

	dialog fyne.CanvasObject
}

// NewSessionListOverlay creates a new session list overlay.
func NewSessionListOverlay() *SessionListOverlay {
	return &SessionListOverlay{}
}

// Show shows the overlay with the given names.
func (slo *SessionListOverlay) Show(names []string) {
	slo.names = names
	slo.Visible = true
	slo.recentMode = false
}

// ShowRecent shows the overlay in recent mode.
func (slo *SessionListOverlay) ShowRecent(names []string) {
	slo.names = names
	slo.Visible = true
	slo.recentMode = true
}

// Hide hides the overlay.
func (slo *SessionListOverlay) Hide() {
	slo.Visible = false
}

// --- Helpers for building managed exercises (moved from app.go) ---

