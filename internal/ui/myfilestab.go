package ui

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/ui/icon"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
)

// MyFilesAction describes an action triggered from the My Files tab.
type MyFilesAction int

const (
	MyFilesActionOpenSession MyFilesAction = iota
	MyFilesActionDeleteSession
	MyFilesActionOpenExercise
	MyFilesActionDeleteExercise
)

// MyFilesEvent is returned when the user performs an action.
type MyFilesEvent struct {
	Action MyFilesAction
	Name   string
}

// SessionFileItem holds display data for a session in the file manager.
type SessionFileItem struct {
	Name          string
	Title         string
	Date          string
	ExerciseCount int
}

// ExerciseFileItem holds display data for an exercise in the file manager.
type ExerciseFileItem struct {
	Name        string
	DisplayName string
	Category    string
	Duration    string
	IsOrphan    bool
}

// MyFilesTab provides a file management view with two columns: sessions and exercises.
type MyFilesTab struct {
	box *fyne.Container

	// Session column.
	sessionItems       []SessionFileItem
	sessionSearchEntry *widget.Entry
	sessionBox         *fyne.Container
	sessionScroll      *container.Scroll

	// Exercise column.
	exerciseItems       []ExerciseFileItem
	exerciseSearchEntry *widget.Entry
	exerciseBox         *fyne.Container
	exerciseScroll      *container.Scroll
	filterOrphan        bool
	filterAllBtn        *widget.Button
	filterOrphanBtn     *widget.Button

	// Responsive.
	responsive *ResponsiveContainer

	// Callbacks.
	OnAction func(MyFilesEvent)
	OnStatus func(string, int)
}

// NewMyFilesTab creates a new My Files tab.
func NewMyFilesTab() *MyFilesTab {
	mf := &MyFilesTab{}

	// Session column.
	mf.sessionSearchEntry = widget.NewEntry()
	mf.sessionSearchEntry.PlaceHolder = i18n.T("myfiles.search_sessions")
	mf.sessionSearchEntry.OnChanged = func(_ string) { mf.refreshSessionList() }
	mf.sessionBox = container.NewVBox()
	mf.sessionScroll = container.NewVScroll(mf.sessionBox)

	// Exercise column.
	mf.exerciseSearchEntry = widget.NewEntry()
	mf.exerciseSearchEntry.PlaceHolder = i18n.T("myfiles.search_exercises")
	mf.exerciseSearchEntry.OnChanged = func(_ string) { mf.refreshExerciseList() }
	mf.exerciseBox = container.NewVBox()
	mf.exerciseScroll = container.NewVScroll(mf.exerciseBox)

	mf.filterAllBtn = widget.NewButton(i18n.T("myfiles.filter_all"), func() {
		mf.filterOrphan = false
		mf.updateFilterStyles()
		mf.refreshExerciseList()
	})
	mf.filterOrphanBtn = widget.NewButton(i18n.T("myfiles.filter_orphan"), func() {
		mf.filterOrphan = true
		mf.updateFilterStyles()
		mf.refreshExerciseList()
	})
	mf.updateFilterStyles()

	bg := canvas.NewRectangle(theme.ColorDarkBg)
	mf.responsive = NewResponsiveContainer(mf.buildDesktopLayout, mf.buildMobileLayout)
	mf.box = container.NewStack(bg, mf.responsive)
	return mf
}

// Widget returns the root canvas object.
func (mf *MyFilesTab) Widget() fyne.CanvasObject {
	return mf.box
}

// SetSessions updates the session list data.
func (mf *MyFilesTab) SetSessions(items []SessionFileItem) {
	mf.sessionItems = items
	mf.refreshSessionList()
}

// SetExercises updates the exercise list data.
func (mf *MyFilesTab) SetExercises(items []ExerciseFileItem) {
	mf.exerciseItems = items
	mf.refreshExerciseList()
}

func (mf *MyFilesTab) updateFilterStyles() {
	if mf.filterOrphan {
		mf.filterAllBtn.Importance = widget.LowImportance
		mf.filterOrphanBtn.Importance = widget.MediumImportance
	} else {
		mf.filterAllBtn.Importance = widget.MediumImportance
		mf.filterOrphanBtn.Importance = widget.LowImportance
	}
	mf.filterAllBtn.Refresh()
	mf.filterOrphanBtn.Refresh()
}

func (mf *MyFilesTab) buildDesktopLayout() fyne.CanvasObject {
	sessHeader := canvas.NewText(i18n.T("myfiles.sessions"), color.White)
	sessHeader.TextSize = 14
	sessHeader.TextStyle = fyne.TextStyle{Bold: true}

	sessCol := container.NewBorder(
		container.NewVBox(sessHeader, mf.sessionSearchEntry),
		nil, nil, nil,
		mf.sessionScroll,
	)

	exHeader := canvas.NewText(i18n.T("myfiles.exercises"), color.White)
	exHeader.TextSize = 14
	exHeader.TextStyle = fyne.TextStyle{Bold: true}
	filterRow := container.NewGridWithColumns(2, mf.filterAllBtn, mf.filterOrphanBtn)

	exCol := container.NewBorder(
		container.NewVBox(exHeader, mf.exerciseSearchEntry, filterRow),
		nil, nil, nil,
		mf.exerciseScroll,
	)

	split := container.NewHSplit(sessCol, exCol)
	split.SetOffset(0.5)
	return split
}

func (mf *MyFilesTab) buildMobileLayout() fyne.CanvasObject {
	sessTab := container.NewBorder(
		mf.sessionSearchEntry,
		nil, nil, nil,
		mf.sessionScroll,
	)

	filterRow := container.NewGridWithColumns(2, mf.filterAllBtn, mf.filterOrphanBtn)
	exTab := container.NewBorder(
		container.NewVBox(mf.exerciseSearchEntry, filterRow),
		nil, nil, nil,
		mf.exerciseScroll,
	)

	tabs := container.NewAppTabs(
		container.NewTabItem(i18n.T("mobile.myfiles.sessions"), sessTab),
		container.NewTabItem(i18n.T("mobile.myfiles.exercises"), exTab),
	)
	tabs.SetTabLocation(container.TabLocationBottom)
	return tabs
}

func (mf *MyFilesTab) refreshSessionList() {
	mf.sessionBox.RemoveAll()

	query := strings.ToLower(mf.sessionSearchEntry.Text)
	count := 0
	for _, item := range mf.sessionItems {
		item := item
		if query != "" && !strings.Contains(strings.ToLower(item.Title), query) &&
			!strings.Contains(strings.ToLower(item.Name), query) {
			continue
		}
		count++

		titleLabel := widget.NewLabel(item.Title)
		titleLabel.TextStyle = fyne.TextStyle{Bold: true}
		titleLabel.Wrapping = fyne.TextWrapWord

		subtleColor := color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff}
		dateText := canvas.NewText(fmt.Sprintf(i18n.T("myfiles.session_date"), item.Date), subtleColor)
		dateText.TextSize = 11
		countText := canvas.NewText(fmt.Sprintf(i18n.T("myfiles.session_exercises"), item.ExerciseCount), subtleColor)
		countText.TextSize = 11

		openBtn := widget.NewButtonWithIcon("", icon.Open(), func() {
			mf.emitAction(MyFilesActionOpenSession, item.Name)
		})
		openBtn.Importance = widget.LowImportance

		deleteBtn := widget.NewButtonWithIcon("", icon.Delete(), func() {
			mf.emitAction(MyFilesActionDeleteSession, item.Name)
		})
		deleteBtn.Importance = widget.DangerImportance

		info := container.NewHBox(dateText, countText)
		row := container.NewBorder(
			nil, info, nil,
			container.NewHBox(openBtn, deleteBtn),
			titleLabel,
		)
		mf.sessionBox.Add(row)
		mf.sessionBox.Add(widget.NewSeparator())
	}

	if count == 0 {
		mf.sessionBox.Add(widget.NewLabel(i18n.T("myfiles.no_sessions")))
	}
}

func (mf *MyFilesTab) refreshExerciseList() {
	mf.exerciseBox.RemoveAll()

	query := strings.ToLower(mf.exerciseSearchEntry.Text)
	count := 0
	for _, item := range mf.exerciseItems {
		item := item
		if mf.filterOrphan && !item.IsOrphan {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(item.DisplayName), query) &&
			!strings.Contains(strings.ToLower(item.Name), query) {
			continue
		}
		count++

		nameLabel := widget.NewLabel(item.DisplayName)
		nameLabel.Wrapping = fyne.TextWrapWord

		subtleColor := color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff}
		meta := item.Category
		if item.Duration != "" {
			if meta != "" {
				meta += " · "
			}
			meta += item.Duration
		}
		if item.IsOrphan {
			if meta != "" {
				meta += " · "
			}
			meta += i18n.T("myfiles.filter_orphan")
		}
		metaText := canvas.NewText(meta, subtleColor)
		metaText.TextSize = 11

		openBtn := widget.NewButtonWithIcon("", icon.Open(), func() {
			mf.emitAction(MyFilesActionOpenExercise, item.Name)
		})
		openBtn.Importance = widget.LowImportance

		deleteBtn := widget.NewButtonWithIcon("", icon.Delete(), func() {
			mf.emitAction(MyFilesActionDeleteExercise, item.Name)
		})
		deleteBtn.Importance = widget.DangerImportance

		row := container.NewBorder(
			nil, metaText, nil,
			container.NewHBox(openBtn, deleteBtn),
			nameLabel,
		)
		mf.exerciseBox.Add(row)
		mf.exerciseBox.Add(widget.NewSeparator())
	}

	if count == 0 {
		mf.exerciseBox.Add(widget.NewLabel(i18n.T("myfiles.no_exercises")))
	}
}

func (mf *MyFilesTab) emitAction(action MyFilesAction, name string) {
	if mf.OnAction != nil {
		mf.OnAction(MyFilesEvent{Action: action, Name: name})
	}
}

// RefreshLanguage updates all translatable text.
func (mf *MyFilesTab) RefreshLanguage() {
	mf.sessionSearchEntry.PlaceHolder = i18n.T("myfiles.search_sessions")
	mf.sessionSearchEntry.Refresh()
	mf.exerciseSearchEntry.PlaceHolder = i18n.T("myfiles.search_exercises")
	mf.exerciseSearchEntry.Refresh()
	mf.filterAllBtn.SetText(i18n.T("myfiles.filter_all"))
	mf.filterOrphanBtn.SetText(i18n.T("myfiles.filter_orphan"))
	mf.responsive.ForceRebuild()
	mf.refreshSessionList()
	mf.refreshExerciseList()
}
