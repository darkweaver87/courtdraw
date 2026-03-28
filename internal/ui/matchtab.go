package ui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/store"
	"github.com/darkweaver87/courtdraw/internal/ui/icon"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
)

// MatchTab provides the match management UI (list + creation form).
type MatchTab struct {
	box    *fyne.Container
	store  store.Store
	window fyne.Window
	onStatus func(string, int)

	// Callbacks.
	OnStartMatch func(match *model.Match, team *model.Team)
	OnShowSummary func(match *model.Match)

	// Match list view.
	matchListBox    *fyne.Container
	matchListScroll *container.Scroll
	matchListView   fyne.CanvasObject

	// Creation form view.
	formView fyne.CanvasObject

	// Form fields.
	teamSelect      *widget.Select
	opponentEntry   *widget.Entry
	dateEntry       *widget.Entry
	timeEntry       *widget.Entry
	locationEntry   *widget.Entry
	competitionEntry *widget.Entry
	homeAwayRadio   *widget.RadioGroup
	periodSelect    *widget.Select
	playerChecks    []*widget.Check
	startingChecks  []*widget.Check
	playerMembers   []model.Member

	// Resolved teams for the dropdown.
	teamEntries []store.TeamIndexEntry

	// Content stack.
	contentStack *fyne.Container
}

// NewMatchTab creates a new match management tab.
func NewMatchTab(s store.Store, w fyne.Window, onStatus func(string, int)) *MatchTab {
	mt := &MatchTab{
		store:    s,
		window:   w,
		onStatus: onStatus,
	}

	// Match list view.
	mt.matchListBox = container.NewVBox()
	mt.matchListScroll = container.NewVScroll(mt.matchListBox)

	newBtn := widget.NewButtonWithIcon(i18n.T(i18n.KeyMatchNew), fynetheme.ContentAddIcon(), func() {
		mt.showCreationForm()
	})
	newBtn.Importance = widget.HighImportance

	mt.matchListView = container.NewBorder(
		nil,
		container.NewPadded(newBtn),
		nil, nil,
		mt.matchListScroll,
	)

	// Content stack starts with match list.
	mt.contentStack = container.NewStack(mt.matchListView)
	bg := canvas.NewRectangle(theme.ColorDarkBg)
	mt.box = container.NewStack(bg, container.NewPadded(mt.contentStack))
	return mt
}

// Widget returns the root canvas object.
func (mt *MatchTab) Widget() fyne.CanvasObject {
	return mt.box
}

// RefreshMatchList reloads matches from store and updates the list view.
func (mt *MatchTab) RefreshMatchList() {
	mt.matchListBox.RemoveAll()

	entries, err := mt.store.ListMatches()
	if err != nil {
		mt.matchListBox.Add(widget.NewLabel(i18n.T(i18n.KeyMatchNoMatches)))
		return
	}

	if len(entries) == 0 {
		mt.matchListBox.Add(widget.NewLabel(i18n.T(i18n.KeyMatchNoMatches)))
		return
	}

	subtleColor := color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff}

	for _, entry := range entries {
		e := entry // capture

		// Title: team vs opponent.
		title := e.TeamName + " vs " + e.Opponent
		nameLabel := widget.NewLabel(title)
		nameLabel.TextStyle = fyne.TextStyle{Bold: true}
		nameLabel.Wrapping = fyne.TextWrapWord

		// Meta: date, home/away, status.
		meta := e.Date
		if e.HomeAway != "" {
			if meta != "" {
				meta += " - "
			}
			if e.HomeAway == "home" {
				meta += i18n.T(i18n.KeyMatchHome)
			} else {
				meta += i18n.T(i18n.KeyMatchAway)
			}
		}
		if e.Status != "" {
			if meta != "" {
				meta += " - "
			}
			meta += mt.statusDisplayName(e.Status)
		}

		metaText := canvas.NewText(meta, subtleColor)
		metaText.TextSize = 11

		// Status badge color.
		statusColor := mt.statusColor(e.Status)
		badge := canvas.NewRectangle(statusColor)
		badge.SetMinSize(fyne.NewSize(8, 8))
		badge.CornerRadius = 4

		openBtn := widget.NewButtonWithIcon("", icon.Open(), func() {
			mt.openMatch(e)
		})
		openBtn.Importance = widget.LowImportance

		deleteBtn := widget.NewButtonWithIcon("", icon.Delete(), func() {
			mt.confirmDeleteMatch(e)
		})
		deleteBtn.Importance = widget.DangerImportance

		row := container.NewBorder(
			nil, metaText,
			container.NewCenter(badge),
			container.NewHBox(openBtn, deleteBtn),
			nameLabel,
		)
		mt.matchListBox.Add(row)
		mt.matchListBox.Add(widget.NewSeparator())
	}
}

func (mt *MatchTab) showMatchList() {
	mt.RefreshMatchList()
	mt.contentStack.Objects = []fyne.CanvasObject{mt.matchListView}
	mt.contentStack.Refresh()
}

func (mt *MatchTab) openMatch(entry store.MatchIndexEntry) {
	name := strings.TrimSuffix(entry.File, ".yaml")
	match, err := mt.store.LoadMatch(name)
	if err != nil {
		mt.emitStatus(fmt.Sprintf("Error: %v", err), 1)
		return
	}

	if match.Status == "finished" {
		if mt.OnShowSummary != nil {
			mt.OnShowSummary(match)
		}
		return
	}

	// Load the team for the match.
	team := mt.loadTeamForMatch(match)

	if mt.OnStartMatch != nil {
		mt.OnStartMatch(match, team)
	}
}

func (mt *MatchTab) loadTeamForMatch(match *model.Match) *model.Team {
	if match.TeamFile == "" {
		return nil
	}
	name := strings.TrimSuffix(match.TeamFile, ".yaml")
	team, err := mt.store.LoadTeam(name)
	if err != nil {
		return nil
	}
	return team
}

func (mt *MatchTab) confirmDeleteMatch(entry store.MatchIndexEntry) {
	title := entry.TeamName + " vs " + entry.Opponent
	msg := fmt.Sprintf(i18n.T(i18n.KeyMatchConfirmDelete), title)
	dialog.ShowConfirm(i18n.T(i18n.KeyMatchTitle), msg, func(ok bool) {
		if !ok {
			return
		}
		name := strings.TrimSuffix(entry.File, ".yaml")
		if err := mt.store.DeleteMatch(name); err != nil {
			mt.emitStatus(fmt.Sprintf("Error: %v", err), 1)
			return
		}
		mt.emitStatus(fmt.Sprintf(i18n.T(i18n.KeyStatusMatchDeleted), title), 2)
		mt.RefreshMatchList()
	}, mt.window)
}

// showCreationForm displays the match creation form.
func (mt *MatchTab) showCreationForm() {
	// Load teams for selector.
	mt.teamEntries = nil
	if entries, err := mt.store.ListTeams(); err == nil {
		mt.teamEntries = entries
	}

	teamOptions := make([]string, len(mt.teamEntries))
	for i, e := range mt.teamEntries {
		teamOptions[i] = e.Name
	}

	mt.teamSelect = widget.NewSelect(teamOptions, func(selected string) {
		mt.onTeamSelected(selected)
	})
	mt.teamSelect.PlaceHolder = i18n.T(i18n.KeyMatchSelectTeam)

	mt.opponentEntry = widget.NewEntry()
	mt.opponentEntry.PlaceHolder = i18n.T(i18n.KeyMatchOpponent)

	mt.dateEntry = widget.NewEntry()
	mt.dateEntry.SetText(time.Now().Format("2006-01-02"))

	mt.timeEntry = widget.NewEntry()
	mt.timeEntry.PlaceHolder = i18n.T(i18n.KeyMatchTime)

	mt.locationEntry = widget.NewEntry()
	mt.locationEntry.PlaceHolder = i18n.T(i18n.KeyMatchLocation)

	mt.competitionEntry = widget.NewEntry()
	mt.competitionEntry.PlaceHolder = i18n.T(i18n.KeyMatchCompetition)

	mt.homeAwayRadio = widget.NewRadioGroup(
		[]string{i18n.T(i18n.KeyMatchHome), i18n.T(i18n.KeyMatchAway)},
		nil,
	)
	mt.homeAwayRadio.Horizontal = true
	mt.homeAwayRadio.SetSelected(i18n.T(i18n.KeyMatchHome))

	periodOptions := []string{
		i18n.T(i18n.KeyPeriod4x8),
		i18n.T(i18n.KeyPeriod4x10),
		i18n.T(i18n.KeyPeriod2x20),
	}
	mt.periodSelect = widget.NewSelect(periodOptions, nil)
	mt.periodSelect.SetSelected(i18n.T(i18n.KeyPeriod4x10))

	// Player selection area (populated when team is selected).
	playerBox := container.NewVBox()
	startingHeader := newSectionHeader(i18n.T(i18n.KeyMatchSelectPlayers))
	playerSection := container.NewVBox(startingHeader, playerBox)
	playerSection.Hide()

	mt.teamSelect.OnChanged = func(selected string) {
		mt.onTeamSelected(selected)
		playerBox.RemoveAll()
		mt.playerChecks = nil
		mt.startingChecks = nil
		mt.playerMembers = nil

		if selected == "" {
			playerSection.Hide()
			return
		}

		// Find team entry and load full team.
		for _, e := range mt.teamEntries {
			if e.Name == selected {
				name := strings.TrimSuffix(e.File, ".yaml")
				team, err := mt.store.LoadTeam(name)
				if err != nil {
					break
				}
				players := team.Players()
				mt.playerMembers = players

				for _, p := range players {
					player := p // capture
					rosterCheck := widget.NewCheck(player.DisplayLabel(), nil)
					rosterCheck.SetChecked(true)
					startCheck := widget.NewCheck(i18n.T(i18n.KeyMatchStartingFive), nil)

					row := container.NewBorder(nil, nil, rosterCheck, startCheck, nil)
					playerBox.Add(row)
					mt.playerChecks = append(mt.playerChecks, rosterCheck)
					mt.startingChecks = append(mt.startingChecks, startCheck)
				}
				// Auto-select first 5 as starters.
				for i := range mt.startingChecks {
					if i < 5 {
						mt.startingChecks[i].SetChecked(true)
					}
				}
				playerSection.Show()
				break
			}
		}
	}

	// Create button.
	createBtn := widget.NewButtonWithIcon(i18n.T(i18n.KeyMatchCreate), fynetheme.ContentAddIcon(), func() {
		mt.createMatch()
	})
	createBtn.Importance = widget.HighImportance

	// Back button.
	backBtn := widget.NewButtonWithIcon("", fynetheme.NavigateBackIcon(), func() {
		mt.showMatchList()
	})
	backBtn.Importance = widget.LowImportance

	formItems := container.NewVBox(
		widget.NewFormItem(i18n.T(i18n.KeyMatchSelectTeam), mt.teamSelect).Widget,
		widget.NewSeparator(),
		mt.opponentEntry,
		mt.dateEntry,
		mt.timeEntry,
		mt.locationEntry,
		mt.competitionEntry,
		widget.NewSeparator(),
		widget.NewLabel(i18n.T(i18n.KeyMatchHomeAway)),
		mt.homeAwayRadio,
		widget.NewSeparator(),
		widget.NewLabel(i18n.T(i18n.KeyMatchPeriodFormat)),
		mt.periodSelect,
		widget.NewSeparator(),
		playerSection,
		widget.NewSeparator(),
		container.NewPadded(createBtn),
	)

	formScroll := container.NewVScroll(formItems)
	mt.formView = container.NewBorder(
		container.NewHBox(backBtn, widget.NewLabel(i18n.T(i18n.KeyMatchNew))),
		nil, nil, nil,
		formScroll,
	)

	mt.contentStack.Objects = []fyne.CanvasObject{mt.formView}
	mt.contentStack.Refresh()
}

func (mt *MatchTab) onTeamSelected(_ string) {
	// placeholder for future logic
}

func (mt *MatchTab) createMatch() {
	if mt.teamSelect.Selected == "" {
		mt.emitStatus(i18n.T(i18n.KeyMatchSelectTeam), 3)
		return
	}
	if strings.TrimSpace(mt.opponentEntry.Text) == "" {
		mt.emitStatus(i18n.T(i18n.KeyMatchOpponent), 3)
		return
	}

	// Determine home/away.
	homeAway := "home"
	if mt.homeAwayRadio.Selected == i18n.T(i18n.KeyMatchAway) {
		homeAway = "away"
	}

	// Determine period format.
	periodFormat := model.PeriodFormat4x10
	switch mt.periodSelect.Selected {
	case i18n.T(i18n.KeyPeriod4x8):
		periodFormat = model.PeriodFormat4x8
	case i18n.T(i18n.KeyPeriod2x20):
		periodFormat = model.PeriodFormat2x20
	}

	// Build roster.
	var roster []model.RosterEntry
	for i, m := range mt.playerMembers {
		if i < len(mt.playerChecks) && mt.playerChecks[i].Checked {
			entry := model.RosterEntry{
				MemberID:  m.ID,
				Number:    m.Number,
				FirstName: m.FirstName,
				LastName:  m.LastName,
			}
			if i < len(mt.startingChecks) && mt.startingChecks[i].Checked {
				entry.Starting = true
			}
			roster = append(roster, entry)
		}
	}

	// Find team file.
	teamFile := ""
	for _, e := range mt.teamEntries {
		if e.Name == mt.teamSelect.Selected {
			teamFile = e.File
			break
		}
	}

	match := &model.Match{
		TeamName:     mt.teamSelect.Selected,
		TeamFile:     teamFile,
		Opponent:     strings.TrimSpace(mt.opponentEntry.Text),
		Date:         strings.TrimSpace(mt.dateEntry.Text),
		Time:         strings.TrimSpace(mt.timeEntry.Text),
		Location:     strings.TrimSpace(mt.locationEntry.Text),
		Competition:  strings.TrimSpace(mt.competitionEntry.Text),
		HomeAway:     homeAway,
		PeriodFormat: periodFormat,
		Roster:       roster,
		Status:       "planned",
	}

	if err := match.Validate(); err != nil {
		mt.emitStatus(fmt.Sprintf("Error: %v", err), 1)
		return
	}
	starters := match.StartingFive()
	if len(starters) < 5 {
		mt.emitStatus("Minimum 5 titulaires requis", 1)
		return
	}

	if err := mt.store.SaveMatch(match); err != nil {
		mt.emitStatus(fmt.Sprintf("Error: %v", err), 1)
		return
	}

	mt.emitStatus(fmt.Sprintf(i18n.T(i18n.KeyStatusMatchSaved), match.TeamName+" vs "+match.Opponent), 2)

	// Load team and start live match.
	team := mt.loadTeamForMatch(match)
	if mt.OnStartMatch != nil {
		mt.OnStartMatch(match, team)
	}
}

func (mt *MatchTab) statusDisplayName(status string) string {
	switch status {
	case "planned":
		return i18n.T(i18n.KeyMatchStatusPlanned)
	case "live":
		return i18n.T(i18n.KeyMatchStatusLive)
	case "finished":
		return i18n.T(i18n.KeyMatchStatusFinished)
	default:
		return status
	}
}

func (mt *MatchTab) statusColor(status string) color.NRGBA {
	switch status {
	case "planned":
		return color.NRGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff} // gray
	case "live":
		return color.NRGBA{R: 0x4c, G: 0xaf, B: 0x50, A: 0xff} // green
	case "finished":
		return color.NRGBA{R: 0x22, G: 0x88, B: 0xdd, A: 0xff} // blue
	default:
		return color.NRGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff}
	}
}

func (mt *MatchTab) emitStatus(msg string, level int) {
	if mt.onStatus != nil {
		mt.onStatus(msg, level)
	}
}

// RefreshLanguage updates all translatable text.
func (mt *MatchTab) RefreshLanguage() {
	mt.RefreshMatchList()
}
