package ui

import (
	"fmt"
	"image/color"
	"strconv"
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

// TeamTab provides the team roster management UI.
type TeamTab struct {
	box    *fyne.Container
	store  store.Store
	window fyne.Window
	onStatus func(string, int)

	// Team list view (no team loaded).
	teamListBox    *fyne.Container
	teamListScroll *container.Scroll
	teamListView   fyne.CanvasObject

	// Team detail view (team loaded).
	team       *model.Team
	detailView fyne.CanvasObject

	// Detail view widgets.
	nameEntry   *widget.Entry
	clubEntry   *widget.Entry
	seasonEntry *widget.Entry

	staffBox   *fyne.Container
	playerBox  *fyne.Container

	// Content stack switches between list and detail.
	contentStack *fyne.Container
}

// NewTeamTab creates a new team management tab.
func NewTeamTab(s store.Store, w fyne.Window, onStatus func(string, int)) *TeamTab {
	tt := &TeamTab{
		store:    s,
		window:   w,
		onStatus: onStatus,
	}

	// Team list view.
	tt.teamListBox = container.NewVBox()
	tt.teamListScroll = container.NewVScroll(tt.teamListBox)

	newBtn := widget.NewButtonWithIcon(i18n.T(i18n.KeyTeamNew), fynetheme.ContentAddIcon(), func() {
		tt.createNewTeam()
	})
	newBtn.Importance = widget.HighImportance

	tt.teamListView = container.NewBorder(
		nil,
		container.NewPadded(newBtn),
		nil, nil,
		tt.teamListScroll,
	)

	// Detail view (built lazily when a team is loaded).
	tt.nameEntry = widget.NewEntry()
	tt.nameEntry.PlaceHolder = i18n.T(i18n.KeyTeamName)
	// Save name on submit (Enter key) — not on every keystroke to avoid file-per-character.
	tt.nameEntry.OnSubmitted = func(_ string) { tt.saveNameChange(strings.TrimSpace(tt.nameEntry.Text)) }

	tt.clubEntry = widget.NewEntry()
	tt.clubEntry.PlaceHolder = i18n.T(i18n.KeyTeamClub)
	tt.clubEntry.OnChanged = func(_ string) { tt.autoSave() }

	tt.seasonEntry = widget.NewEntry()
	tt.seasonEntry.PlaceHolder = i18n.T(i18n.KeyTeamSeason)
	tt.seasonEntry.OnChanged = func(_ string) { tt.autoSave() }

	tt.staffBox = container.NewVBox()
	tt.playerBox = container.NewVBox()

	addMemberBtn := widget.NewButtonWithIcon(i18n.T(i18n.KeyTeamAddMember), fynetheme.ContentAddIcon(), func() {
		tt.showMemberDialog(nil)
	})
	addMemberBtn.Importance = widget.LowImportance

	backBtn := widget.NewButtonWithIcon("", fynetheme.NavigateBackIcon(), func() {
		tt.autoSave() // save pending name/club/season changes
		tt.showTeamList()
	})
	backBtn.Importance = widget.LowImportance

	headerForm := container.NewVBox(
		tt.nameEntry,
		tt.clubEntry,
		tt.seasonEntry,
	)

	staffHeader := newSectionHeader(i18n.T(i18n.KeyTeamStaff))
	playerHeader := newSectionHeader(i18n.T(i18n.KeyTeamPlayers))

	detailContent := container.NewVBox(
		headerForm,
		widget.NewSeparator(),
		staffHeader,
		tt.staffBox,
		widget.NewSeparator(),
		playerHeader,
		tt.playerBox,
		widget.NewSeparator(),
		container.NewPadded(addMemberBtn),
	)
	detailScroll := container.NewVScroll(detailContent)

	tt.detailView = container.NewBorder(
		container.NewHBox(backBtn),
		nil, nil, nil,
		detailScroll,
	)

	// Content stack starts with team list.
	tt.contentStack = container.NewStack(tt.teamListView)
	bg := canvas.NewRectangle(theme.ColorDarkBg)
	tt.box = container.NewStack(bg, container.NewPadded(tt.contentStack))
	return tt
}

// Widget returns the root canvas object.
func (tt *TeamTab) Widget() fyne.CanvasObject {
	return tt.box
}

// RefreshTeamList reloads teams from store and updates the list view.
func (tt *TeamTab) RefreshTeamList() {
	tt.teamListBox.RemoveAll()

	entries, err := tt.store.ListTeams()
	if err != nil {
		tt.teamListBox.Add(widget.NewLabel(i18n.T(i18n.KeyTeamNoTeams)))
		return
	}

	if len(entries) == 0 {
		tt.teamListBox.Add(widget.NewLabel(i18n.T(i18n.KeyTeamNoTeams)))
		return
	}

	subtleColor := color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff}

	for _, entry := range entries {
		e := entry // capture

		nameLabel := widget.NewLabel(e.Name)
		nameLabel.TextStyle = fyne.TextStyle{Bold: true}
		nameLabel.Wrapping = fyne.TextWrapWord

		meta := ""
		if e.Club != "" {
			meta = e.Club
		}
		if e.Season != "" {
			if meta != "" {
				meta += " - "
			}
			meta += e.Season
		}
		membersStr := fmt.Sprintf("%d %s", e.Members, strings.ToLower(i18n.T(i18n.KeyTeamMembers)))
		if meta != "" {
			meta += " - "
		}
		meta += membersStr

		metaText := canvas.NewText(meta, subtleColor)
		metaText.TextSize = 11

		openBtn := widget.NewButtonWithIcon("", icon.Open(), func() {
			tt.loadTeam(e.File)
		})
		openBtn.Importance = widget.LowImportance

		deleteBtn := widget.NewButtonWithIcon("", icon.Delete(), func() {
			tt.confirmDeleteTeam(e)
		})
		deleteBtn.Importance = widget.DangerImportance

		row := container.NewBorder(
			nil, metaText, nil,
			container.NewHBox(openBtn, deleteBtn),
			nameLabel,
		)
		tt.teamListBox.Add(row)
		tt.teamListBox.Add(widget.NewSeparator())
	}
}

func (tt *TeamTab) showTeamList() {
	tt.team = nil
	tt.RefreshTeamList()
	tt.contentStack.Objects = []fyne.CanvasObject{tt.teamListView}
	tt.contentStack.Refresh()
}

func (tt *TeamTab) loadTeam(file string) {
	// Extract name from file: remove .yaml extension.
	name := strings.TrimSuffix(file, ".yaml")
	team, err := tt.store.LoadTeam(name)
	if err != nil {
		tt.emitStatus(fmt.Sprintf("Error: %v", err), 1)
		return
	}
	tt.team = team
	tt.showDetail()
}

func (tt *TeamTab) showDetail() {
	if tt.team == nil {
		return
	}

	// Populate header fields without triggering autoSave.
	tt.nameEntry.OnSubmitted = nil
	tt.clubEntry.OnChanged = nil
	tt.seasonEntry.OnChanged = nil
	tt.nameEntry.SetText(tt.team.Name)
	tt.clubEntry.SetText(tt.team.Club)
	tt.seasonEntry.SetText(tt.team.Season)
	tt.nameEntry.OnSubmitted = func(_ string) { tt.saveNameChange(strings.TrimSpace(tt.nameEntry.Text)) }
	tt.clubEntry.OnChanged = func(_ string) { tt.autoSave() }
	tt.seasonEntry.OnChanged = func(_ string) { tt.autoSave() }

	tt.refreshMembers()
	tt.contentStack.Objects = []fyne.CanvasObject{tt.detailView}
	tt.contentStack.Refresh()
}

func (tt *TeamTab) refreshMembers() {
	if tt.team == nil {
		return
	}

	// Staff section.
	tt.staffBox.RemoveAll()
	staff := tt.team.Staff()
	if len(staff) == 0 {
		tt.staffBox.Add(widget.NewLabel(i18n.T(i18n.KeyTeamNoMembers)))
	} else {
		for _, m := range staff {
			member := m // capture
			parts := []string{strings.ToUpper(member.LastName) + " " + member.FirstName, tt.roleDisplayName(member.Role)}
			if member.LicenseNumber != "" {
				parts = append(parts, member.LicenseNumber)
			}
			label := canvas.NewText(strings.Join(parts, "  ·  "), color.NRGBA{R: 0xdd, G: 0xdd, B: 0xdd, A: 0xff})
			label.TextSize = 12

			editBtn := widget.NewButtonWithIcon("", fynetheme.SettingsIcon(), func() {
				tt.showMemberDialog(&member)
			})
			editBtn.Importance = widget.LowImportance

			row := container.NewBorder(nil, nil, nil, editBtn, label)
			tt.staffBox.Add(row)
		}
	}

	// Players section.
	tt.playerBox.RemoveAll()
	players := tt.team.Players()
	if len(players) == 0 {
		tt.playerBox.Add(widget.NewLabel(i18n.T(i18n.KeyTeamNoMembers)))
	} else {
		for _, m := range players {
			member := m // capture
			// Single line: #N NOM Prénom · DD/MM/YYYY · licence
			parts := []string{}
			if member.Number > 0 {
				parts = append(parts, fmt.Sprintf("#%d", member.Number))
			}
			parts = append(parts, strings.ToUpper(member.LastName)+" "+member.FirstName)
			if member.BirthDate != "" {
				parts = append(parts, member.BirthDate)
			}
			if member.LicenseNumber != "" {
				parts = append(parts, member.LicenseNumber)
			}
			label := canvas.NewText(strings.Join(parts, "  ·  "), color.NRGBA{R: 0xdd, G: 0xdd, B: 0xdd, A: 0xff})
			label.TextSize = 12

			editBtn := widget.NewButtonWithIcon("", fynetheme.SettingsIcon(), func() {
				tt.showMemberDialog(&member)
			})
			editBtn.Importance = widget.LowImportance

			row := container.NewBorder(nil, nil, nil, editBtn, label)
			tt.playerBox.Add(row)
		}
	}
}

func (tt *TeamTab) createNewTeam() {
	team := &model.Team{
		Name:   i18n.T(i18n.KeyTeamNew),
		Season: currentSeason(),
	}
	if err := tt.store.SaveTeam(team); err != nil {
		tt.emitStatus(fmt.Sprintf("Error: %v", err), 1)
		return
	}
	tt.team = team
	tt.emitStatus(fmt.Sprintf(i18n.T(i18n.KeyStatusTeamSaved), team.Name), 2)
	tt.showDetail()
}

func (tt *TeamTab) confirmDeleteTeam(entry store.TeamIndexEntry) {
	msg := fmt.Sprintf(i18n.T(i18n.KeyTeamConfirmDelete), entry.Name)
	dialog.ShowConfirm(i18n.T(i18n.KeyTeamTitle), msg, func(ok bool) {
		if !ok {
			return
		}
		name := strings.TrimSuffix(entry.File, ".yaml")
		if err := tt.store.DeleteTeam(name); err != nil {
			tt.emitStatus(fmt.Sprintf("Error: %v", err), 1)
			return
		}
		tt.emitStatus(fmt.Sprintf(i18n.T(i18n.KeyStatusTeamDeleted), entry.Name), 2)
		tt.RefreshTeamList()
	}, tt.window)
}

func (tt *TeamTab) showRenameDialog() {
	if tt.team == nil {
		return
	}
	entry := widget.NewEntry()
	entry.SetText(tt.team.Name)
	dlg := dialog.NewForm(i18n.T(i18n.KeyTeamName), "OK", i18n.T(i18n.KeyDialogCancel),
		[]*widget.FormItem{widget.NewFormItem(i18n.T(i18n.KeyTeamName), entry)},
		func(ok bool) {
			if !ok {
				return
			}
			tt.saveNameChange(strings.TrimSpace(entry.Text))
		},
		tt.window,
	)
	dlg.Show()
}

// Save explicitly saves the current team (triggered by the save button).
func (tt *TeamTab) Save() {
	tt.autoSave()
	if tt.team != nil {
		tt.emitStatus(fmt.Sprintf(i18n.T(i18n.KeyStatusTeamSaved), tt.team.Name), 2)
	}
}

func (tt *TeamTab) autoSave() {
	if tt.team == nil {
		return
	}
	tt.team.Club = tt.clubEntry.Text
	tt.team.Season = tt.seasonEntry.Text
	// Save name change if it differs (handles focus-lost / tab switch).
	newName := strings.TrimSpace(tt.nameEntry.Text)
	if newName != "" && newName != tt.team.Name {
		oldName := store.TeamFileName(tt.team)
		tt.team.Name = newName
		if err := tt.store.SaveTeam(tt.team); err != nil {
			tt.emitStatus(fmt.Sprintf("Error: %v", err), 1)
			return
		}
		if oldName != store.TeamFileName(tt.team) {
			_ = tt.store.DeleteTeam(oldName)
		}
		return
	}
	if err := tt.store.SaveTeam(tt.team); err != nil {
		tt.emitStatus(fmt.Sprintf("Error: %v", err), 1)
	}
}

func (tt *TeamTab) saveNameChange(newName string) {
	if tt.team == nil {
		return
	}
	if newName == "" || newName == tt.team.Name {
		return
	}
	// Delete old file, save with new name.
	oldName := store.TeamFileName(tt.team)
	tt.team.Name = newName
	if err := tt.store.SaveTeam(tt.team); err != nil {
		tt.emitStatus(fmt.Sprintf("Error: %v", err), 1)
		return
	}
	// Remove old file if name changed.
	if oldName != store.TeamFileName(tt.team) {
		_ = tt.store.DeleteTeam(oldName)
	}
	tt.nameEntry.SetText(tt.team.Name)
	tt.emitStatus(fmt.Sprintf(i18n.T(i18n.KeyStatusTeamSaved), tt.team.Name), 2)
}

func (tt *TeamTab) showMemberDialog(existing *model.Member) {
	if tt.team == nil {
		return
	}

	isNew := existing == nil
	var m model.Member
	if !isNew {
		m = *existing
	} else {
		m.Role = model.MemberRolePlayer
		m.ID = tt.team.NextMemberID()
	}

	firstNameEntry := widget.NewEntry()
	firstNameEntry.SetText(m.FirstName)

	lastNameEntry := widget.NewEntry()
	lastNameEntry.SetText(m.LastName)

	roleOptions := []string{
		i18n.T(i18n.KeyMemberRolePlayer),
		i18n.T(i18n.KeyMemberRoleCoach),
		i18n.T(i18n.KeyMemberRoleAssistant),
	}
	roleSelect := widget.NewSelect(roleOptions, nil)
	roleSelect.SetSelected(tt.roleDisplayName(m.Role))

	numberEntry := widget.NewEntry()
	if m.Number > 0 {
		numberEntry.SetText(strconv.Itoa(m.Number))
	}

	licenseEntry := widget.NewEntry()
	licenseEntry.SetText(m.LicenseNumber)

	licenseTypeOptions := []string{
		i18n.T(i18n.KeyMemberLicenseCompetition),
		i18n.T(i18n.KeyMemberLicenseLoisir),
		i18n.T(i18n.KeyMemberLicenseMiniBask),
	}
	licenseTypeSelect := widget.NewSelect(licenseTypeOptions, nil)
	switch m.LicenseType {
	case model.LicenseLoisir:
		licenseTypeSelect.SetSelected(licenseTypeOptions[1])
	case model.LicenseMiniBask:
		licenseTypeSelect.SetSelected(licenseTypeOptions[2])
	case model.LicenseCompetition:
		licenseTypeSelect.SetSelected(licenseTypeOptions[0])
	}

	birthDateEntry := widget.NewEntry()
	birthDateEntry.PlaceHolder = "JJ/MM/AAAA"
	birthDateEntry.SetText(m.BirthDate)

	ageCatLabel := widget.NewLabel(m.AgeCategory(time.Now().Year()))
	birthDateEntry.OnChanged = func(s string) {
		// Try to extract year from various formats.
		var y int
		if len(s) == 10 { // DD/MM/YYYY or YYYY-MM-DD
			if _, err := fmt.Sscanf(s[6:], "%d", &y); err != nil || y < 1900 {
				fmt.Sscanf(s[:4], "%d", &y)
			}
		} else if len(s) == 4 {
			fmt.Sscanf(s, "%d", &y)
		}
		if y > 1900 {
			tmp := model.Member{BirthYear: y}
			ageCatLabel.SetText(tmp.AgeCategory(time.Now().Year()))
		} else {
			ageCatLabel.SetText("")
		}
	}

	posOptions := []string{"PG", "SG", "SF", "PF", "C"}
	posSelect := widget.NewSelect(posOptions, nil)
	if m.Position != "" {
		posSelect.SetSelected(strings.ToUpper(m.Position))
	}

	emailEntry := widget.NewEntry()
	emailEntry.SetText(m.Email)

	phoneEntry := widget.NewEntry()
	phoneEntry.SetText(m.Phone)

	// Build form items. Position is only shown for players.
	items := []*widget.FormItem{
		widget.NewFormItem(i18n.T(i18n.KeyMemberFirstName), firstNameEntry),
		widget.NewFormItem(i18n.T(i18n.KeyMemberLastName), lastNameEntry),
		widget.NewFormItem(i18n.T(i18n.KeyMemberRole), roleSelect),
		widget.NewFormItem(i18n.T(i18n.KeyMemberNumber), numberEntry),
		widget.NewFormItem(i18n.T(i18n.KeyMemberLicense), licenseEntry),
		widget.NewFormItem(i18n.T(i18n.KeyMemberLicenseType), licenseTypeSelect),
		widget.NewFormItem(i18n.T(i18n.KeyMemberBirthDate), birthDateEntry),
		widget.NewFormItem(i18n.T(i18n.KeyMemberAgeCategory), ageCatLabel),
		widget.NewFormItem(i18n.T(i18n.KeyMemberPosition), posSelect),
		widget.NewFormItem(i18n.T(i18n.KeyMemberEmail), emailEntry),
		widget.NewFormItem(i18n.T(i18n.KeyMemberPhone), phoneEntry),
	}

	title := i18n.T(i18n.KeyTeamAddMember)
	if !isNew {
		title = m.FullName()
	}

	dlg := dialog.NewForm(title, i18n.T(i18n.KeyPrefsSave), i18n.T(i18n.KeyDialogCancel), items, func(ok bool) {
		if !ok {
			return
		}
		m.FirstName = firstNameEntry.Text
		m.LastName = lastNameEntry.Text
		m.Role = tt.roleFromDisplay(roleSelect.Selected)
		if n, err := strconv.Atoi(numberEntry.Text); err == nil {
			m.Number = n
		} else {
			m.Number = 0
		}
		m.LicenseNumber = licenseEntry.Text
		switch licenseTypeSelect.Selected {
		case licenseTypeOptions[1]:
			m.LicenseType = model.LicenseLoisir
		case licenseTypeOptions[2]:
			m.LicenseType = model.LicenseMiniBask
		default:
			m.LicenseType = model.LicenseCompetition
		}
		m.BirthDate = birthDateEntry.Text
		// Extract year for backward compat.
		m.BirthYear = m.BirthYearEffective()
		if m.Role == model.MemberRolePlayer {
			m.Position = strings.ToLower(posSelect.Selected)
		} else {
			m.Position = ""
		}
		m.Email = emailEntry.Text
		m.Phone = phoneEntry.Text

		if isNew {
			tt.team.Members = append(tt.team.Members, m)
			tt.emitStatus(fmt.Sprintf(i18n.T(i18n.KeyStatusMemberAdded), m.FullName()), 2)
		} else {
			for idx := range tt.team.Members {
				if tt.team.Members[idx].ID == m.ID {
					tt.team.Members[idx] = m
					break
				}
			}
		}

		if err := tt.store.SaveTeam(tt.team); err != nil {
			tt.emitStatus(fmt.Sprintf("Error: %v", err), 1)
		} else {
			tt.emitStatus(fmt.Sprintf(i18n.T(i18n.KeyStatusTeamSaved), tt.team.Name), 2)
		}
		tt.refreshMembers()
	}, tt.window)

	// Add delete button for existing members.
	if !isNew {
		dlg.SetOnClosed(func() {})
		deleteBtn := widget.NewButtonWithIcon(i18n.T(i18n.KeyTeamRemoveMember), icon.Delete(), func() {
			dlg.Hide()
			tt.confirmRemoveMember(m)
		})
		deleteBtn.Importance = widget.DangerImportance
		dlg.SetDismissText(i18n.T(i18n.KeyDialogCancel))

		// We can't add extra buttons to dialog.NewForm directly, but we
		// place the delete button as a custom form item at the end.
		items = append(items, widget.NewFormItem("", deleteBtn))
		dlg = dialog.NewForm(title, i18n.T(i18n.KeyPrefsSave), i18n.T(i18n.KeyDialogCancel), items, func(ok bool) {
			if !ok {
				return
			}
			m.FirstName = firstNameEntry.Text
			m.LastName = lastNameEntry.Text
			m.Role = tt.roleFromDisplay(roleSelect.Selected)
			if n, err := strconv.Atoi(numberEntry.Text); err == nil {
				m.Number = n
			} else {
				m.Number = 0
			}
			m.LicenseNumber = licenseEntry.Text
			switch licenseTypeSelect.Selected {
			case licenseTypeOptions[1]:
				m.LicenseType = model.LicenseLoisir
			case licenseTypeOptions[2]:
				m.LicenseType = model.LicenseMiniBask
			default:
				m.LicenseType = model.LicenseCompetition
			}
			m.BirthDate = birthDateEntry.Text
			m.BirthYear = m.BirthYearEffective()
			if m.Role == model.MemberRolePlayer {
				m.Position = strings.ToLower(posSelect.Selected)
			} else {
				m.Position = ""
			}
			m.Email = emailEntry.Text
			m.Phone = phoneEntry.Text

			for idx := range tt.team.Members {
				if tt.team.Members[idx].ID == m.ID {
					tt.team.Members[idx] = m
					break
				}
			}
			if err := tt.store.SaveTeam(tt.team); err != nil {
				tt.emitStatus(fmt.Sprintf("Error: %v", err), 1)
			} else {
				tt.emitStatus(fmt.Sprintf(i18n.T(i18n.KeyStatusTeamSaved), tt.team.Name), 2)
			}
			tt.refreshMembers()
		}, tt.window)
	}

	dlg.Resize(fyne.NewSize(400, 500))
	dlg.Show()
}

func (tt *TeamTab) confirmRemoveMember(m model.Member) {
	msg := fmt.Sprintf(i18n.T(i18n.KeyTeamConfirmRemoveMember), m.FullName())
	dialog.ShowConfirm(i18n.T(i18n.KeyTeamTitle), msg, func(ok bool) {
		if !ok {
			return
		}
		for idx := range tt.team.Members {
			if tt.team.Members[idx].ID == m.ID {
				tt.team.Members = append(tt.team.Members[:idx], tt.team.Members[idx+1:]...)
				break
			}
		}
		if err := tt.store.SaveTeam(tt.team); err != nil {
			tt.emitStatus(fmt.Sprintf("Error: %v", err), 1)
		}
		tt.emitStatus(fmt.Sprintf(i18n.T(i18n.KeyStatusMemberRemoved), m.FullName()), 2)
		tt.refreshMembers()
	}, tt.window)
}

func (tt *TeamTab) roleDisplayName(role model.MemberRole) string {
	switch role {
	case model.MemberRoleCoach:
		return i18n.T(i18n.KeyMemberRoleCoach)
	case model.MemberRoleAssistant:
		return i18n.T(i18n.KeyMemberRoleAssistant)
	default:
		return i18n.T(i18n.KeyMemberRolePlayer)
	}
}

func (tt *TeamTab) roleFromDisplay(display string) model.MemberRole {
	switch display {
	case i18n.T(i18n.KeyMemberRoleCoach):
		return model.MemberRoleCoach
	case i18n.T(i18n.KeyMemberRoleAssistant):
		return model.MemberRoleAssistant
	default:
		return model.MemberRolePlayer
	}
}

func (tt *TeamTab) emitStatus(msg string, level int) {
	if tt.onStatus != nil {
		tt.onStatus(msg, level)
	}
}

// RefreshLanguage updates all translatable text.
func (tt *TeamTab) RefreshLanguage() {
	tt.nameEntry.PlaceHolder = i18n.T(i18n.KeyTeamName)
	tt.nameEntry.Refresh()
	tt.clubEntry.PlaceHolder = i18n.T(i18n.KeyTeamClub)
	tt.clubEntry.Refresh()
	tt.seasonEntry.PlaceHolder = i18n.T(i18n.KeyTeamSeason)
	tt.seasonEntry.Refresh()
	if tt.team != nil {
		tt.refreshMembers()
	} else {
		tt.RefreshTeamList()
	}
}

// newSectionHeader creates a styled section header label.
func newSectionHeader(text string) *canvas.Text {
	t := canvas.NewText(text, color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff})
	t.TextSize = 14
	t.TextStyle = fyne.TextStyle{Bold: true}
	return t
}

// currentSeason returns a season string like "2025-2026" based on current date.
func currentSeason() string {
	now := time.Now()
	year := now.Year()
	// Basketball season typically starts in September.
	if now.Month() >= time.September {
		return fmt.Sprintf("%d-%d", year, year+1)
	}
	return fmt.Sprintf("%d-%d", year-1, year)
}
