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
	MyFilesActionShareSession
	MyFilesActionImportBundle
	MyFilesActionOpenExercise
	MyFilesActionDeleteExercise
	MyFilesActionContributeExercise
	MyFilesActionShareExercise
	MyFilesActionOpenTeam
	MyFilesActionDeleteTeam
	MyFilesActionShareTeam
	MyFilesActionOpenMatch
	MyFilesActionDeleteMatch
	MyFilesActionShareMatch
	MyFilesActionImportExercise
	MyFilesActionImportTeam
	MyFilesActionImportMatch
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

// TeamFileItem holds display data for a team in the file manager.
type TeamFileItem struct {
	Name        string
	DisplayName string
	Club        string
	Season      string
	MemberCount int
}

// MatchFileItem holds display data for a match in the file manager.
type MatchFileItem struct {
	Name        string
	TeamName    string
	Opponent    string
	Date        string
	HomeScore   int
	AwayScore   int
	Status      string // planned/live/finished
}

// MyFilesTab provides a single-column file management view with toggle buttons.
type MyFilesTab struct {
	box *fyne.Container

	// Session list.
	sessionItems       []SessionFileItem
	sessionSearchEntry *widget.Entry
	sessionBox         *fyne.Container
	sessionScroll      *container.Scroll

	// Exercise list.
	exerciseItems       []ExerciseFileItem
	exerciseSearchEntry *widget.Entry
	exerciseBox         *fyne.Container
	exerciseScroll      *container.Scroll
	filterOrphan        bool
	filterAllBtn        *widget.Button
	filterOrphanBtn     *widget.Button

	// Team list.
	teamItems       []TeamFileItem
	teamSearchEntry *widget.Entry
	teamBox         *fyne.Container
	teamScroll      *container.Scroll

	// Match list.
	matchItems       []MatchFileItem
	matchSearchEntry *widget.Entry
	matchBox         *fyne.Container
	matchScroll      *container.Scroll

	// View switching.
	contentStack *fyne.Container
	sessionView  fyne.CanvasObject
	exerciseView fyne.CanvasObject
	teamView     fyne.CanvasObject
	matchView    fyne.CanvasObject
	sesToggle    *widget.Button
	exToggle     *widget.Button
	teamToggle   *widget.Button
	matchToggle  *widget.Button

	// Callbacks.
	OnAction func(MyFilesEvent)
	OnStatus func(string, int)
}

// NewMyFilesTab creates a new My Files tab.
func NewMyFilesTab() *MyFilesTab {
	mf := &MyFilesTab{}

	// Session search.
	mf.sessionSearchEntry = widget.NewEntry()
	mf.sessionSearchEntry.PlaceHolder = i18n.T(i18n.KeyMyfilesSearchSessions)
	mf.sessionSearchEntry.OnChanged = func(_ string) { mf.refreshSessionList() }
	mf.sessionBox = container.NewVBox()
	mf.sessionScroll = container.NewVScroll(mf.sessionBox)

	// Exercise search.
	mf.exerciseSearchEntry = widget.NewEntry()
	mf.exerciseSearchEntry.PlaceHolder = i18n.T(i18n.KeyMyfilesSearchExercises)
	mf.exerciseSearchEntry.OnChanged = func(_ string) { mf.refreshExerciseList() }
	mf.exerciseBox = container.NewVBox()
	mf.exerciseScroll = container.NewVScroll(mf.exerciseBox)

	// Exercise filter buttons (same style as session tab).
	mf.filterAllBtn = widget.NewButton(i18n.T(i18n.KeyMyfilesFilterAll), func() {
		mf.filterOrphan = false
		mf.updateFilterStyles()
		mf.refreshExerciseList()
	})
	mf.filterAllBtn.Importance = widget.HighImportance
	mf.filterOrphanBtn = widget.NewButton(i18n.T(i18n.KeyMyfilesFilterOrphan), func() {
		mf.filterOrphan = true
		mf.updateFilterStyles()
		mf.refreshExerciseList()
	})
	mf.filterOrphanBtn.Importance = widget.LowImportance

	// Team search.
	mf.teamSearchEntry = widget.NewEntry()
	mf.teamSearchEntry.PlaceHolder = i18n.T(i18n.KeyMyfilesSearchTeams)
	mf.teamSearchEntry.OnChanged = func(_ string) { mf.refreshTeamList() }
	mf.teamBox = container.NewVBox()
	mf.teamScroll = container.NewVScroll(mf.teamBox)

	// Match search.
	mf.matchSearchEntry = widget.NewEntry()
	mf.matchSearchEntry.PlaceHolder = i18n.T(i18n.KeyMyfilesSearchMatches)
	mf.matchSearchEntry.OnChanged = func(_ string) { mf.refreshMatchList() }
	mf.matchBox = container.NewVBox()
	mf.matchScroll = container.NewVScroll(mf.matchBox)

	// Import buttons per section.
	importSessionBtn := widget.NewButtonWithIcon(i18n.T(i18n.KeyImportBundle), icon.Import(), func() {
		mf.emitAction(MyFilesActionImportBundle, "")
	})
	importSessionBtn.Importance = widget.LowImportance

	importExerciseBtn := widget.NewButtonWithIcon(i18n.T(i18n.KeyImportBundle), icon.Import(), func() {
		mf.emitAction(MyFilesActionImportExercise, "")
	})
	importExerciseBtn.Importance = widget.LowImportance

	importTeamBtn := widget.NewButtonWithIcon(i18n.T(i18n.KeyImportBundle), icon.Import(), func() {
		mf.emitAction(MyFilesActionImportTeam, "")
	})
	importTeamBtn.Importance = widget.LowImportance

	importMatchBtn := widget.NewButtonWithIcon(i18n.T(i18n.KeyImportBundle), icon.Import(), func() {
		mf.emitAction(MyFilesActionImportMatch, "")
	})
	importMatchBtn.Importance = widget.LowImportance

	// --- Session view ---
	mf.sessionView = container.NewBorder(
		mf.sessionSearchEntry,
		container.NewPadded(importSessionBtn),
		nil, nil,
		mf.sessionScroll,
	)

	// --- Exercise view ---
	filterRow := container.NewGridWithColumns(2, mf.filterAllBtn, mf.filterOrphanBtn)
	mf.exerciseView = container.NewBorder(
		container.NewVBox(mf.exerciseSearchEntry, filterRow),
		container.NewPadded(importExerciseBtn),
		nil, nil,
		mf.exerciseScroll,
	)

	// --- Team view ---
	mf.teamView = container.NewBorder(
		mf.teamSearchEntry,
		container.NewPadded(importTeamBtn),
		nil, nil,
		mf.teamScroll,
	)

	// --- Match view ---
	mf.matchView = container.NewBorder(
		mf.matchSearchEntry,
		container.NewPadded(importMatchBtn),
		nil, nil,
		mf.matchScroll,
	)

	// --- Toggle buttons ---
	mf.sesToggle = widget.NewButton(i18n.T(i18n.KeyMyfilesSessions), func() { mf.showSessions() })
	mf.sesToggle.Importance = widget.HighImportance
	mf.exToggle = widget.NewButton(i18n.T(i18n.KeyMyfilesExercises), func() { mf.showExercises() })
	mf.exToggle.Importance = widget.LowImportance
	mf.teamToggle = widget.NewButton(i18n.T(i18n.KeyMyfilesTeams), func() { mf.showTeams() })
	mf.teamToggle.Importance = widget.LowImportance
	mf.matchToggle = widget.NewButton(i18n.T(i18n.KeyMyfilesMatches), func() { mf.showMatches() })
	mf.matchToggle.Importance = widget.LowImportance
	toggleBar := container.NewGridWithColumns(4, mf.sesToggle, mf.exToggle, mf.teamToggle, mf.matchToggle)

	// --- Content stack ---
	mf.contentStack = container.NewStack(mf.sessionView)

	bg := canvas.NewRectangle(theme.ColorDarkBg)
	mf.box = container.NewStack(bg, container.NewBorder(toggleBar, nil, nil, nil, mf.contentStack))
	return mf
}

// Widget returns the root canvas object.
func (mf *MyFilesTab) Widget() fyne.CanvasObject {
	return mf.box
}

func (mf *MyFilesTab) setActiveToggle(active *widget.Button) {
	for _, btn := range []*widget.Button{mf.sesToggle, mf.exToggle, mf.teamToggle, mf.matchToggle} {
		if btn == active {
			btn.Importance = widget.HighImportance
		} else {
			btn.Importance = widget.LowImportance
		}
		btn.Refresh()
	}
}

func (mf *MyFilesTab) showSessions() {
	mf.contentStack.Objects = []fyne.CanvasObject{mf.sessionView}
	mf.contentStack.Refresh()
	mf.setActiveToggle(mf.sesToggle)
}

func (mf *MyFilesTab) showExercises() {
	mf.contentStack.Objects = []fyne.CanvasObject{mf.exerciseView}
	mf.contentStack.Refresh()
	mf.setActiveToggle(mf.exToggle)
}

func (mf *MyFilesTab) showTeams() {
	mf.contentStack.Objects = []fyne.CanvasObject{mf.teamView}
	mf.contentStack.Refresh()
	mf.setActiveToggle(mf.teamToggle)
}

func (mf *MyFilesTab) showMatches() {
	mf.contentStack.Objects = []fyne.CanvasObject{mf.matchView}
	mf.contentStack.Refresh()
	mf.setActiveToggle(mf.matchToggle)
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

// SetTeams updates the team list data.
func (mf *MyFilesTab) SetTeams(items []TeamFileItem) {
	mf.teamItems = items
	mf.refreshTeamList()
}

// SetMatches updates the match list data.
func (mf *MyFilesTab) SetMatches(items []MatchFileItem) {
	mf.matchItems = items
	mf.refreshMatchList()
}

func (mf *MyFilesTab) updateFilterStyles() {
	if mf.filterOrphan {
		mf.filterAllBtn.Importance = widget.LowImportance
		mf.filterOrphanBtn.Importance = widget.HighImportance
	} else {
		mf.filterAllBtn.Importance = widget.HighImportance
		mf.filterOrphanBtn.Importance = widget.LowImportance
	}
	mf.filterAllBtn.Refresh()
	mf.filterOrphanBtn.Refresh()
}

func (mf *MyFilesTab) refreshSessionList() {
	mf.sessionBox.RemoveAll()

	query := strings.ToLower(mf.sessionSearchEntry.Text)
	count := 0
	for _, item := range mf.sessionItems {
		if query != "" && !strings.Contains(strings.ToLower(item.Title), query) &&
			!strings.Contains(strings.ToLower(item.Name), query) {
			continue
		}
		count++

		titleLabel := widget.NewLabel(item.Title)
		titleLabel.TextStyle = fyne.TextStyle{Bold: true}
		titleLabel.Wrapping = fyne.TextWrapWord

		subtleColor := color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff}
		dateText := canvas.NewText(fmt.Sprintf(i18n.T(i18n.KeyMyfilesSessionDate), item.Date), subtleColor)
		dateText.TextSize = 11
		countText := canvas.NewText(fmt.Sprintf(i18n.T(i18n.KeyMyfilesSessionExercises), item.ExerciseCount), subtleColor)
		countText.TextSize = 11

		openBtn := widget.NewButtonWithIcon("", icon.Open(), func() {
			mf.emitAction(MyFilesActionOpenSession, item.Name)
		})
		openBtn.Importance = widget.LowImportance

		shareBtn := widget.NewButtonWithIcon("", icon.Share(), func() {
			mf.emitAction(MyFilesActionShareSession, item.Name)
		})
		shareBtn.Importance = widget.LowImportance

		deleteBtn := widget.NewButtonWithIcon("", icon.Delete(), func() {
			mf.emitAction(MyFilesActionDeleteSession, item.Name)
		})
		deleteBtn.Importance = widget.DangerImportance

		info := container.NewHBox(dateText, countText)
		row := container.NewBorder(
			nil, info, nil,
			container.NewHBox(openBtn, shareBtn, deleteBtn),
			titleLabel,
		)
		mf.sessionBox.Add(row)
		mf.sessionBox.Add(widget.NewSeparator())
	}

	if count == 0 {
		mf.sessionBox.Add(widget.NewLabel(i18n.T(i18n.KeyMyfilesNoSessions)))
	}
}

func (mf *MyFilesTab) refreshExerciseList() {
	mf.exerciseBox.RemoveAll()

	query := strings.ToLower(mf.exerciseSearchEntry.Text)
	count := 0
	for _, item := range mf.exerciseItems {
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
			meta += i18n.T(i18n.KeyMyfilesFilterOrphan)
		}
		metaText := canvas.NewText(meta, subtleColor)
		metaText.TextSize = 11

		openBtn := widget.NewButtonWithIcon("", icon.Open(), func() {
			mf.emitAction(MyFilesActionOpenExercise, item.Name)
		})
		openBtn.Importance = widget.LowImportance

		shareBtn := widget.NewButtonWithIcon("", icon.Share(), func() {
			mf.emitAction(MyFilesActionShareExercise, item.Name)
		})
		shareBtn.Importance = widget.LowImportance

		contributeBtn := widget.NewButtonWithIcon("", icon.Upload(), func() {
			mf.emitAction(MyFilesActionContributeExercise, item.Name)
		})
		contributeBtn.Importance = widget.LowImportance

		deleteBtn := widget.NewButtonWithIcon("", icon.Delete(), func() {
			mf.emitAction(MyFilesActionDeleteExercise, item.Name)
		})
		deleteBtn.Importance = widget.DangerImportance

		row := container.NewBorder(
			nil, metaText, nil,
			container.NewHBox(openBtn, shareBtn, contributeBtn, deleteBtn),
			nameLabel,
		)
		mf.exerciseBox.Add(row)
		mf.exerciseBox.Add(widget.NewSeparator())
	}

	if count == 0 {
		mf.exerciseBox.Add(widget.NewLabel(i18n.T(i18n.KeyMyfilesNoExercises)))
	}
}

func (mf *MyFilesTab) refreshTeamList() {
	mf.teamBox.RemoveAll()

	query := strings.ToLower(mf.teamSearchEntry.Text)
	count := 0
	for _, item := range mf.teamItems {
		if query != "" && !strings.Contains(strings.ToLower(item.DisplayName), query) &&
			!strings.Contains(strings.ToLower(item.Club), query) {
			continue
		}
		count++

		nameLabel := widget.NewLabel(item.DisplayName)
		nameLabel.TextStyle = fyne.TextStyle{Bold: true}
		nameLabel.Wrapping = fyne.TextWrapWord

		subtleColor := color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff}
		meta := item.Club
		if item.Season != "" {
			if meta != "" {
				meta += " · "
			}
			meta += item.Season
		}
		meta += fmt.Sprintf(" · %d members", item.MemberCount)
		metaText := canvas.NewText(meta, subtleColor)
		metaText.TextSize = 11

		openBtn := widget.NewButtonWithIcon("", icon.Open(), func() {
			mf.emitAction(MyFilesActionOpenTeam, item.Name)
		})
		openBtn.Importance = widget.LowImportance

		shareBtn := widget.NewButtonWithIcon("", icon.Share(), func() {
			mf.emitAction(MyFilesActionShareTeam, item.Name)
		})
		shareBtn.Importance = widget.LowImportance

		deleteBtn := widget.NewButtonWithIcon("", icon.Delete(), func() {
			mf.emitAction(MyFilesActionDeleteTeam, item.Name)
		})
		deleteBtn.Importance = widget.DangerImportance

		row := container.NewBorder(
			nil, metaText, nil,
			container.NewHBox(openBtn, shareBtn, deleteBtn),
			nameLabel,
		)
		mf.teamBox.Add(row)
		mf.teamBox.Add(widget.NewSeparator())
	}

	if count == 0 {
		mf.teamBox.Add(widget.NewLabel(i18n.T(i18n.KeyMyfilesNoTeams)))
	}
}

func (mf *MyFilesTab) refreshMatchList() {
	mf.matchBox.RemoveAll()

	query := strings.ToLower(mf.matchSearchEntry.Text)
	count := 0
	for _, item := range mf.matchItems {
		if query != "" && !strings.Contains(strings.ToLower(item.TeamName), query) &&
			!strings.Contains(strings.ToLower(item.Opponent), query) {
			continue
		}
		count++

		title := fmt.Sprintf("%s vs %s", item.TeamName, item.Opponent)
		titleLabel := widget.NewLabel(title)
		titleLabel.TextStyle = fyne.TextStyle{Bold: true}
		titleLabel.Wrapping = fyne.TextWrapWord

		subtleColor := color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff}
		meta := item.Date
		if item.Status == "finished" {
			meta += fmt.Sprintf(" · %d-%d", item.HomeScore, item.AwayScore)
		}
		meta += fmt.Sprintf(" (%s)", item.Status)
		metaText := canvas.NewText(meta, subtleColor)
		metaText.TextSize = 11

		openBtn := widget.NewButtonWithIcon("", icon.Open(), func() {
			mf.emitAction(MyFilesActionOpenMatch, item.Name)
		})
		openBtn.Importance = widget.LowImportance

		shareBtn := widget.NewButtonWithIcon("", icon.Share(), func() {
			mf.emitAction(MyFilesActionShareMatch, item.Name)
		})
		shareBtn.Importance = widget.LowImportance

		deleteBtn := widget.NewButtonWithIcon("", icon.Delete(), func() {
			mf.emitAction(MyFilesActionDeleteMatch, item.Name)
		})
		deleteBtn.Importance = widget.DangerImportance

		row := container.NewBorder(
			nil, metaText, nil,
			container.NewHBox(openBtn, shareBtn, deleteBtn),
			titleLabel,
		)
		mf.matchBox.Add(row)
		mf.matchBox.Add(widget.NewSeparator())
	}

	if count == 0 {
		mf.matchBox.Add(widget.NewLabel(i18n.T(i18n.KeyMyfilesNoMatches)))
	}
}

func (mf *MyFilesTab) emitAction(action MyFilesAction, name string) {
	if mf.OnAction != nil {
		mf.OnAction(MyFilesEvent{Action: action, Name: name})
	}
}

// RefreshLanguage updates all translatable text.
func (mf *MyFilesTab) RefreshLanguage() {
	mf.sessionSearchEntry.PlaceHolder = i18n.T(i18n.KeyMyfilesSearchSessions)
	mf.sessionSearchEntry.Refresh()
	mf.exerciseSearchEntry.PlaceHolder = i18n.T(i18n.KeyMyfilesSearchExercises)
	mf.exerciseSearchEntry.Refresh()
	mf.teamSearchEntry.PlaceHolder = i18n.T(i18n.KeyMyfilesSearchTeams)
	mf.teamSearchEntry.Refresh()
	mf.matchSearchEntry.PlaceHolder = i18n.T(i18n.KeyMyfilesSearchMatches)
	mf.matchSearchEntry.Refresh()
	mf.filterAllBtn.SetText(i18n.T(i18n.KeyMyfilesFilterAll))
	mf.filterOrphanBtn.SetText(i18n.T(i18n.KeyMyfilesFilterOrphan))
	if mf.sesToggle != nil {
		mf.sesToggle.SetText(i18n.T(i18n.KeyMyfilesSessions))
	}
	if mf.exToggle != nil {
		mf.exToggle.SetText(i18n.T(i18n.KeyMyfilesExercises))
	}
	if mf.teamToggle != nil {
		mf.teamToggle.SetText(i18n.T(i18n.KeyMyfilesTeams))
	}
	if mf.matchToggle != nil {
		mf.matchToggle.SetText(i18n.T(i18n.KeyMyfilesMatches))
	}
	mf.refreshSessionList()
	mf.refreshExerciseList()
	mf.refreshTeamList()
	mf.refreshMatchList()
}
