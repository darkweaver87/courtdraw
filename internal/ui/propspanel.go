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
	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/ui/editor"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
)

// PropertiesPanel is the right sidebar showing element properties and exercise metadata.
type PropertiesPanel struct {
	box     *fyne.Container
	content *fyne.Container

	// Metadata editors.
	nameEntry        *widget.Entry
	descriptionEntry *widget.Entry
	durationEntry    *widget.Entry
	tagsEntry        *widget.Entry

	// Dropdown selectors.
	courtStdSelect  *widget.Select
	courtTypeSelect *widget.Select
	categorySelect  *widget.Select
	ageGroupSelect  *widget.Select

	// Intensity.
	intensityBtns [3]*TipButton

	// Player editors.
	playerLabelEntry *widget.Entry
	playerRoleSelect *widget.Select
	ballCheck        *widget.Check
	queueCheck       *widget.Check
	calloutSelect    *widget.Select
	rotationSlider   *widget.Slider
	rotationLabel    *canvas.Text

	// Sync tracking.
	syncedPlayerIdx int
	syncedKind      editor.SelectionKind
	syncedSeqIdx    int
	metaSynced      bool
	syncedEditLang  string

	exercise *model.Exercise
	state    *editor.EditorState
	seqIndex int
	editLang string

	OnModified func()
}

// NewPropertiesPanel creates a new properties panel.
func NewPropertiesPanel() *PropertiesPanel {
	pp := &PropertiesPanel{
		syncedPlayerIdx: -1,
		syncedSeqIdx:    -1,
	}

	pp.nameEntry = widget.NewEntry()
	pp.nameEntry.OnChanged = func(s string) { pp.onNameChanged(s) }

	pp.descriptionEntry = widget.NewEntry()
	pp.descriptionEntry.OnChanged = func(s string) { pp.onDescriptionChanged(s) }

	pp.durationEntry = widget.NewEntry()
	pp.durationEntry.OnChanged = func(s string) { pp.onDurationChanged(s) }

	pp.tagsEntry = widget.NewEntry()
	pp.tagsEntry.OnChanged = func(s string) { pp.onTagsChanged(s) }

	pp.courtStdSelect = widget.NewSelect([]string{"FIBA", "NBA"}, func(s string) {
		if pp.exercise != nil {
			pp.exercise.CourtStandard = model.CourtStandard(strings.ToLower(s))
			pp.markModified()
		}
	})

	pp.courtTypeSelect = widget.NewSelect(
		[]string{i18n.T("props.court_half"), i18n.T("props.court_full")},
		func(s string) {
			if pp.exercise == nil {
				return
			}
			if s == i18n.T("props.court_full") {
				pp.exercise.CourtType = model.FullCourt
			} else {
				pp.exercise.CourtType = model.HalfCourt
			}
			pp.markModified()
		},
	)

	categories := []string{
		i18n.T("props.category_none"),
		i18n.T("category." + string(model.CategoryWarmup)),
		i18n.T("category." + string(model.CategoryOffense)),
		i18n.T("category." + string(model.CategoryDefense)),
		i18n.T("category." + string(model.CategoryTransition)),
		i18n.T("category." + string(model.CategoryScrimmage)),
		i18n.T("category." + string(model.CategoryCooldown)),
	}
	categoryKeys := []model.Category{
		"", model.CategoryWarmup, model.CategoryOffense, model.CategoryDefense,
		model.CategoryTransition, model.CategoryScrimmage, model.CategoryCooldown,
	}
	pp.categorySelect = widget.NewSelect(categories, func(s string) {
		if pp.exercise == nil {
			return
		}
		for i, label := range pp.categorySelect.Options {
			if label == s && i < len(categoryKeys) {
				pp.exercise.Category = categoryKeys[i]
				pp.markModified()
				return
			}
		}
	})

	ageGroups := []string{
		i18n.T("props.category_none"),
		"U9", "U11", "U13", "U15", "U17", "U19",
		i18n.T("age_group." + string(model.AgeGroupSenior)),
	}
	ageGroupKeys := []model.AgeGroup{
		"", model.AgeGroupU9, model.AgeGroupU11, model.AgeGroupU13,
		model.AgeGroupU15, model.AgeGroupU17, model.AgeGroupU19, model.AgeGroupSenior,
	}
	pp.ageGroupSelect = widget.NewSelect(ageGroups, func(s string) {
		if pp.exercise == nil {
			return
		}
		for i, label := range pp.ageGroupSelect.Options {
			if label == s && i < len(ageGroupKeys) {
				pp.exercise.AgeGroup = ageGroupKeys[i]
				pp.markModified()
				return
			}
		}
	})

	// Intensity buttons — green / yellow / red.
	intensityLabels := [3]string{"●", "●●", "●●●"}
	for i := 0; i < 3; i++ {
		level := i + 1
		pp.intensityBtns[i] = NewTipButton(nil, "", func() {
			if pp.exercise == nil {
				return
			}
			newLevel := model.Intensity(level)
			if pp.exercise.Intensity == newLevel {
				pp.exercise.Intensity = 0
			} else {
				pp.exercise.Intensity = newLevel
			}
			pp.markModified()
			pp.refreshIntensity()
		})
		pp.intensityBtns[i].SetText(intensityLabels[i])
	}

	// Player editors.
	pp.playerLabelEntry = widget.NewEntry()
	pp.playerLabelEntry.OnChanged = func(s string) { pp.onPlayerLabelChanged(s) }

	roles := []string{
		i18n.T("tool.player.attacker"), i18n.T("tool.player.defender"), i18n.T("tool.player.coach"),
		i18n.T("tool.player.pg"), i18n.T("tool.player.sg"), i18n.T("tool.player.sf"),
		i18n.T("tool.player.pf"), i18n.T("tool.player.center"),
	}
	roleKeys := []model.PlayerRole{
		model.RoleAttacker, model.RoleDefender, model.RoleCoach,
		model.RolePointGuard, model.RoleShootingGuard, model.RoleSmallForward,
		model.RolePowerForward, model.RoleCenter,
	}
	pp.playerRoleSelect = widget.NewSelect(roles, func(s string) {
		pp.onPlayerRoleChanged(s, pp.playerRoleSelect.Options, roleKeys)
	})

	pp.ballCheck = widget.NewCheck(i18n.T("props.ball"), func(checked bool) {
		pp.toggleBallCarrier(checked)
	})
	pp.queueCheck = widget.NewCheck(i18n.T("tool.player.queue"), func(checked bool) {
		pp.toggleQueue(checked)
	})

	pp.rotationSlider = widget.NewSlider(0, 360)
	pp.rotationSlider.Step = 5
	pp.rotationSlider.OnChanged = func(v float64) { pp.onRotationChanged(v) }
	pp.rotationLabel = canvas.NewText("0°", color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	pp.rotationLabel.TextSize = 12

	calloutLabels := []string{i18n.T("callout.none")}
	for _, c := range model.AllCallouts() {
		calloutLabels = append(calloutLabels, i18n.T("callout."+string(c)))
	}
	pp.calloutSelect = widget.NewSelect(calloutLabels, func(s string) {
		pp.onCalloutChanged(s, pp.calloutSelect.Options)
	})

	pp.content = container.NewVBox()
	bg := canvas.NewRectangle(color.NRGBA{R: 0x30, G: 0x30, B: 0x30, A: 0xff})
	scroll := container.NewVScroll(pp.content)
	pp.box = container.NewStack(bg, scroll)
	return pp
}

// Widget returns the properties panel widget.
func (pp *PropertiesPanel) Widget() fyne.CanvasObject {
	return pp.box
}

// Update syncs the panel with the current exercise/selection state.
func (pp *PropertiesPanel) Update(exercise *model.Exercise, state *editor.EditorState, seqIndex int, editLang string) {
	pp.exercise = exercise
	pp.state = state
	pp.seqIndex = seqIndex
	pp.editLang = editLang

	if exercise == nil {
		pp.content.RemoveAll()
		return
	}

	// Detect sync needs.
	if pp.syncedEditLang != editLang {
		pp.metaSynced = false
		pp.syncedEditLang = editLang
	}

	pp.content.RemoveAll()

	// Element properties.
	sel := state.SelectedElement
	if sel != nil && sel.SeqIndex == seqIndex && seqIndex < len(exercise.Sequences) {
		seq := &exercise.Sequences[seqIndex]

		// Sync player editor.
		if sel.Kind != pp.syncedKind || sel.Index != pp.syncedPlayerIdx || sel.SeqIndex != pp.syncedSeqIdx {
			pp.syncedKind = sel.Kind
			pp.syncedPlayerIdx = sel.Index
			pp.syncedSeqIdx = sel.SeqIndex
			if sel.Kind == editor.SelectPlayer && sel.Index < len(seq.Players) {
				pp.playerLabelEntry.SetText(seq.Players[sel.Index].Label)
			}
		}

		pp.content.Add(pp.makeSection(i18n.T("props.element")))

		switch sel.Kind {
		case editor.SelectPlayer:
			if sel.Index < len(seq.Players) {
				pp.addPlayerProps(seq, sel.Index)
			}
		case editor.SelectAccessory:
			if sel.Index < len(seq.Accessories) {
				pp.addAccessoryProps(&seq.Accessories[sel.Index])
			}
		case editor.SelectAction:
			if sel.Index < len(seq.Actions) {
				pp.addActionProps(&seq.Actions[sel.Index])
			}
		}

		pp.content.Add(widget.NewSeparator())
	}

	// Exercise metadata.
	pp.content.Add(pp.makeSection(i18n.T("props.exercise")))
	pp.addMetadataFields(exercise, editLang)
}

// RefreshLanguage rebuilds all translatable Select options and labels.
func (pp *PropertiesPanel) RefreshLanguage() {
	// Court type.
	pp.courtTypeSelect.Options = []string{i18n.T("props.court_half"), i18n.T("props.court_full")}
	pp.courtTypeSelect.Refresh()

	// Category.
	pp.categorySelect.Options = []string{
		i18n.T("props.category_none"),
		i18n.T("category." + string(model.CategoryWarmup)),
		i18n.T("category." + string(model.CategoryOffense)),
		i18n.T("category." + string(model.CategoryDefense)),
		i18n.T("category." + string(model.CategoryTransition)),
		i18n.T("category." + string(model.CategoryScrimmage)),
		i18n.T("category." + string(model.CategoryCooldown)),
	}
	pp.categorySelect.Refresh()

	// Age group.
	pp.ageGroupSelect.Options = []string{
		i18n.T("props.category_none"),
		"U9", "U11", "U13", "U15", "U17", "U19",
		i18n.T("age_group." + string(model.AgeGroupSenior)),
	}
	pp.ageGroupSelect.Refresh()

	// Player role.
	pp.playerRoleSelect.Options = []string{
		i18n.T("tool.player.attacker"), i18n.T("tool.player.defender"), i18n.T("tool.player.coach"),
		i18n.T("tool.player.pg"), i18n.T("tool.player.sg"), i18n.T("tool.player.sf"),
		i18n.T("tool.player.pf"), i18n.T("tool.player.center"),
	}
	pp.playerRoleSelect.Refresh()

	// Callout.
	calloutLabels := []string{i18n.T("callout.none")}
	for _, c := range model.AllCallouts() {
		calloutLabels = append(calloutLabels, i18n.T("callout."+string(c)))
	}
	pp.calloutSelect.Options = calloutLabels
	pp.calloutSelect.Refresh()

	// Checkboxes.
	pp.ballCheck.Text = i18n.T("props.ball")
	pp.ballCheck.Refresh()
	pp.queueCheck.Text = i18n.T("tool.player.queue")
	pp.queueCheck.Refresh()

	// Force re-sync of metadata so Select values match new options.
	pp.metaSynced = false
}

// SyncFromExercise resets sync flags so editors are refreshed.
func (pp *PropertiesPanel) SyncFromExercise() {
	pp.metaSynced = false
	pp.syncedPlayerIdx = -1
	pp.syncedSeqIdx = -1
}

func (pp *PropertiesPanel) addPlayerProps(seq *model.Sequence, idx int) {
	p := &seq.Players[idx]

	pp.content.Add(pp.makeField(i18n.T("props.label"), pp.playerLabelEntry))
	pp.content.Add(pp.makeField(i18n.T("props.role"), pp.playerRoleSelect))
	pp.playerRoleSelect.SetSelected(roleDisplayLabel(p.Role))

	// Ball carrier checkbox.
	pp.ballCheck.SetChecked(seq.BallCarrier == p.ID)
	pp.content.Add(container.NewPadded(pp.ballCheck))

	// Queue checkbox.
	pp.queueCheck.SetChecked(p.Type == "queue")
	pp.content.Add(container.NewPadded(pp.queueCheck))

	// Callout.
	if p.Callout != "" {
		pp.calloutSelect.SetSelected(i18n.T("callout." + string(p.Callout)))
	} else {
		pp.calloutSelect.SetSelected(i18n.T("callout.none"))
	}
	pp.content.Add(pp.makeField(i18n.T("props.callout"), pp.calloutSelect))

	// Position with ± buttons.
	pp.content.Add(pp.makePositionEditor(p))

	// Rotation slider.
	pp.rotationSlider.Value = p.Rotation
	pp.rotationLabel.Text = fmt.Sprintf("%.0f°", p.Rotation)
	pp.content.Add(pp.makeField(i18n.T("props.rotation"),
		container.NewBorder(nil, nil, nil, pp.rotationLabel, pp.rotationSlider)))
}

func (pp *PropertiesPanel) addAccessoryProps(a *model.Accessory) {
	pp.content.Add(pp.makeReadonly(i18n.T("props.type"), string(a.Type)))

	// Position with ± buttons.
	pp.content.Add(pp.makeAccessoryPositionEditor(a))

	// Rotation slider.
	pp.rotationSlider.Value = a.Rotation
	pp.rotationLabel.Text = fmt.Sprintf("%.0f°", a.Rotation)
	pp.content.Add(pp.makeField(i18n.T("props.rotation"),
		container.NewBorder(nil, nil, nil, pp.rotationLabel, pp.rotationSlider)))
}

func (pp *PropertiesPanel) addActionProps(a *model.Action) {
	pp.content.Add(pp.makeReadonly(i18n.T("props.type"), string(a.Type)))
	pp.content.Add(pp.makeReadonly(i18n.T("props.from"), refString(a.From)))
	pp.content.Add(pp.makeReadonly(i18n.T("props.to"), refString(a.To)))
}

func (pp *PropertiesPanel) addMetadataFields(ex *model.Exercise, editLang string) {
	// Add widgets to the container first, then sync values.
	// This ensures SetText/SetSelected happen while widgets are attached,
	// avoiding Fyne visual stale-state issues.
	pp.content.Add(pp.makeField(i18n.T("props.name"), pp.nameEntry))
	pp.content.Add(pp.makeField(i18n.T("props.description"), pp.descriptionEntry))

	// Non-translatable fields — always visible regardless of editLang.
	pp.content.Add(pp.makeField(i18n.T("props.standard"), pp.courtStdSelect))
	pp.content.Add(pp.makeField(i18n.T("props.court"), pp.courtTypeSelect))
	pp.content.Add(pp.makeField(i18n.T("props.duration"), pp.durationEntry))

	pp.refreshIntensity()
	intensityRow := container.NewHBox(pp.intensityBtns[0], pp.intensityBtns[1], pp.intensityBtns[2])
	pp.content.Add(pp.makeField(i18n.T("props.intensity"), intensityRow))

	pp.content.Add(pp.makeField(i18n.T("props.category"), pp.categorySelect))
	pp.content.Add(pp.makeField(i18n.T("props.age_group"), pp.ageGroupSelect))
	pp.content.Add(pp.makeField(i18n.T("props.tags"), pp.tagsEntry))

	// Sync text values after widgets are attached.
	if !pp.metaSynced {
		if editLang == "en" {
			pp.nameEntry.SetText(ex.Name)
			pp.descriptionEntry.SetText(ex.Description)
			pp.tagsEntry.SetText(strings.Join(ex.Tags, ", "))
		} else {
			tr := ex.EnsureI18n(editLang)
			pp.nameEntry.SetText(tr.Name)
			pp.descriptionEntry.SetText(tr.Description)
			pp.tagsEntry.SetText(strings.Join(tr.Tags, ", "))
		}
		// Duration is not translatable — always sync from exercise.
		pp.durationEntry.SetText(ex.Duration)
		pp.metaSynced = true
	}

	// Always sync Select widgets and intensity.
	pp.courtStdSelect.SetSelected(strings.ToUpper(string(ex.CourtStandard)))
	if ex.CourtType == model.FullCourt {
		pp.courtTypeSelect.SetSelected(i18n.T("props.court_full"))
	} else {
		pp.courtTypeSelect.SetSelected(i18n.T("props.court_half"))
	}
	pp.syncCategorySelect(ex)
	pp.syncAgeGroupSelect(ex)
}

var intensityColors = [3]color.Color{
	color.NRGBA{R: 0x4c, G: 0xaf, B: 0x50, A: 0xff}, // green
	color.NRGBA{R: 0xff, G: 0xc1, B: 0x07, A: 0xff}, // yellow
	color.NRGBA{R: 0xf4, G: 0x43, B: 0x36, A: 0xff}, // red
}

func (pp *PropertiesPanel) refreshIntensity() {
	if pp.exercise == nil {
		return
	}
	for i := 0; i < 3; i++ {
		level := i + 1
		if int(pp.exercise.Intensity) >= level {
			pp.intensityBtns[i].OverrideColor = intensityColors[i]
		} else {
			pp.intensityBtns[i].OverrideColor = nil
		}
		pp.intensityBtns[i].Refresh()
	}
}

// --- Event handlers ---

func (pp *PropertiesPanel) onNameChanged(s string) {
	if pp.exercise == nil {
		return
	}
	if pp.editLang == "en" {
		pp.exercise.Name = s
	} else {
		tr := pp.exercise.EnsureI18n(pp.editLang)
		tr.Name = s
		pp.exercise.SetI18n(pp.editLang, tr)
	}
	pp.markModified()
}

func (pp *PropertiesPanel) onDescriptionChanged(s string) {
	if pp.exercise == nil {
		return
	}
	if pp.editLang == "en" {
		pp.exercise.Description = s
	} else {
		tr := pp.exercise.EnsureI18n(pp.editLang)
		tr.Description = s
		pp.exercise.SetI18n(pp.editLang, tr)
	}
	pp.markModified()
}

func (pp *PropertiesPanel) onDurationChanged(s string) {
	if pp.exercise == nil {
		return
	}
	pp.exercise.Duration = s
	pp.markModified()
}

func (pp *PropertiesPanel) onTagsChanged(s string) {
	if pp.exercise == nil {
		return
	}
	parts := strings.Split(s, ",")
	tags := make([]string, 0, len(parts))
	for _, t := range parts {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}
	if pp.editLang == "en" {
		pp.exercise.Tags = tags
	} else {
		tr := pp.exercise.EnsureI18n(pp.editLang)
		tr.Tags = tags
		pp.exercise.SetI18n(pp.editLang, tr)
	}
	pp.markModified()
}

func (pp *PropertiesPanel) onPlayerLabelChanged(s string) {
	sel := pp.state.SelectedElement
	if sel == nil || sel.Kind != editor.SelectPlayer {
		return
	}
	if pp.exercise == nil || pp.seqIndex >= len(pp.exercise.Sequences) {
		return
	}
	seq := &pp.exercise.Sequences[pp.seqIndex]
	if sel.Index < len(seq.Players) {
		seq.Players[sel.Index].Label = s
		pp.markModified()
	}
}

func (pp *PropertiesPanel) onPlayerRoleChanged(s string, labels []string, keys []model.PlayerRole) {
	sel := pp.state.SelectedElement
	if sel == nil || sel.Kind != editor.SelectPlayer {
		return
	}
	if pp.exercise == nil || pp.seqIndex >= len(pp.exercise.Sequences) {
		return
	}
	seq := &pp.exercise.Sequences[pp.seqIndex]
	if sel.Index >= len(seq.Players) {
		return
	}
	for i, label := range labels {
		if label == s {
			p := &seq.Players[sel.Index]
			oldDefault := model.RoleLabel(p.Role)
			p.Role = keys[i]
			if p.Label == "" || p.Label == oldDefault {
				p.Label = model.RoleLabel(p.Role)
				pp.playerLabelEntry.SetText(p.Label)
			}
			pp.markModified()
			return
		}
	}
}

func (pp *PropertiesPanel) toggleBallCarrier(checked bool) {
	sel := pp.state.SelectedElement
	if sel == nil || sel.Kind != editor.SelectPlayer {
		return
	}
	if pp.exercise == nil || pp.seqIndex >= len(pp.exercise.Sequences) {
		return
	}
	seq := &pp.exercise.Sequences[pp.seqIndex]
	if sel.Index >= len(seq.Players) {
		return
	}
	p := &seq.Players[sel.Index]
	if checked {
		seq.BallCarrier = p.ID
	} else {
		seq.BallCarrier = ""
	}
	pp.markModified()
}

func (pp *PropertiesPanel) toggleQueue(checked bool) {
	sel := pp.state.SelectedElement
	if sel == nil || sel.Kind != editor.SelectPlayer {
		return
	}
	if pp.exercise == nil || pp.seqIndex >= len(pp.exercise.Sequences) {
		return
	}
	seq := &pp.exercise.Sequences[pp.seqIndex]
	if sel.Index >= len(seq.Players) {
		return
	}
	p := &seq.Players[sel.Index]
	if checked {
		p.Type = "queue"
		if p.Count < 2 {
			p.Count = 3
		}
	} else {
		p.Type = ""
		p.Count = 0
	}
	pp.markModified()
}

func (pp *PropertiesPanel) onRotationChanged(v float64) {
	sel := pp.state.SelectedElement
	if sel == nil || pp.exercise == nil || pp.seqIndex >= len(pp.exercise.Sequences) {
		return
	}
	seq := &pp.exercise.Sequences[pp.seqIndex]
	switch sel.Kind {
	case editor.SelectPlayer:
		if sel.Index < len(seq.Players) {
			seq.Players[sel.Index].Rotation = v
		}
	case editor.SelectAccessory:
		if sel.Index < len(seq.Accessories) {
			seq.Accessories[sel.Index].Rotation = v
		}
	default:
		return
	}
	pp.rotationLabel.Text = fmt.Sprintf("%.0f°", v)
	pp.rotationLabel.Refresh()
	pp.markModified()
}

func (pp *PropertiesPanel) adjustPlayerPos(axis int, delta float64) {
	sel := pp.state.SelectedElement
	if sel == nil || sel.Kind != editor.SelectPlayer {
		return
	}
	if pp.exercise == nil || pp.seqIndex >= len(pp.exercise.Sequences) {
		return
	}
	seq := &pp.exercise.Sequences[pp.seqIndex]
	if sel.Index >= len(seq.Players) {
		return
	}
	p := &seq.Players[sel.Index]
	p.Position[axis] += delta
	if p.Position[axis] < 0 {
		p.Position[axis] = 0
	}
	if p.Position[axis] > 1 {
		p.Position[axis] = 1
	}
	pp.markModified()
}

func (pp *PropertiesPanel) adjustAccessoryPos(axis int, delta float64) {
	sel := pp.state.SelectedElement
	if sel == nil || sel.Kind != editor.SelectAccessory {
		return
	}
	if pp.exercise == nil || pp.seqIndex >= len(pp.exercise.Sequences) {
		return
	}
	seq := &pp.exercise.Sequences[pp.seqIndex]
	if sel.Index >= len(seq.Accessories) {
		return
	}
	a := &seq.Accessories[sel.Index]
	a.Position[axis] += delta
	if a.Position[axis] < 0 {
		a.Position[axis] = 0
	}
	if a.Position[axis] > 1 {
		a.Position[axis] = 1
	}
	pp.markModified()
}

func (pp *PropertiesPanel) onCalloutChanged(s string, labels []string) {
	sel := pp.state.SelectedElement
	if sel == nil || sel.Kind != editor.SelectPlayer {
		return
	}
	if pp.exercise == nil || pp.seqIndex >= len(pp.exercise.Sequences) {
		return
	}
	seq := &pp.exercise.Sequences[pp.seqIndex]
	if sel.Index >= len(seq.Players) {
		return
	}
	if s == labels[0] { // "None"
		seq.Players[sel.Index].Callout = ""
	} else {
		allCallouts := model.AllCallouts()
		for i, label := range labels[1:] {
			if label == s && i < len(allCallouts) {
				seq.Players[sel.Index].Callout = allCallouts[i]
				break
			}
		}
	}
	pp.markModified()
}

func (pp *PropertiesPanel) syncCategorySelect(ex *model.Exercise) {
	if ex.Category == "" {
		pp.categorySelect.SetSelected(i18n.T("props.category_none"))
	} else {
		pp.categorySelect.SetSelected(i18n.T("category." + string(ex.Category)))
	}
}

func (pp *PropertiesPanel) syncAgeGroupSelect(ex *model.Exercise) {
	if ex.AgeGroup == "" {
		pp.ageGroupSelect.SetSelected(i18n.T("props.category_none"))
	} else if ex.AgeGroup == model.AgeGroupSenior {
		pp.ageGroupSelect.SetSelected(i18n.T("age_group." + string(ex.AgeGroup)))
	} else {
		pp.ageGroupSelect.SetSelected(string(ex.AgeGroup))
	}
}

func (pp *PropertiesPanel) markModified() {
	if pp.state != nil {
		pp.state.MarkModified()
	}
	if pp.OnModified != nil {
		pp.OnModified()
	}
}

// --- Position editor helpers ---

const posStep = 0.01

func (pp *PropertiesPanel) makePositionEditor(p *model.Player) fyne.CanvasObject {
	xLabel := canvas.NewText(fmt.Sprintf("%.2f", p.Position.X()), color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	xLabel.TextSize = 12
	xLabel.Alignment = fyne.TextAlignCenter
	yLabel := canvas.NewText(fmt.Sprintf("%.2f", p.Position.Y()), color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	yLabel.TextSize = 12
	yLabel.Alignment = fyne.TextAlignCenter

	xMinus := widget.NewButton("-", func() {
		pp.adjustPlayerPos(0, -posStep)
		xLabel.Text = fmt.Sprintf("%.2f", pp.exercise.Sequences[pp.seqIndex].Players[pp.state.SelectedElement.Index].Position.X())
		xLabel.Refresh()
	})
	xMinus.Importance = widget.LowImportance
	xPlus := widget.NewButton("+", func() {
		pp.adjustPlayerPos(0, posStep)
		xLabel.Text = fmt.Sprintf("%.2f", pp.exercise.Sequences[pp.seqIndex].Players[pp.state.SelectedElement.Index].Position.X())
		xLabel.Refresh()
	})
	xPlus.Importance = widget.LowImportance

	yMinus := widget.NewButton("-", func() {
		pp.adjustPlayerPos(1, -posStep)
		yLabel.Text = fmt.Sprintf("%.2f", pp.exercise.Sequences[pp.seqIndex].Players[pp.state.SelectedElement.Index].Position.Y())
		yLabel.Refresh()
	})
	yMinus.Importance = widget.LowImportance
	yPlus := widget.NewButton("+", func() {
		pp.adjustPlayerPos(1, posStep)
		yLabel.Text = fmt.Sprintf("%.2f", pp.exercise.Sequences[pp.seqIndex].Players[pp.state.SelectedElement.Index].Position.Y())
		yLabel.Refresh()
	})
	yPlus.Importance = widget.LowImportance

	xRow := container.NewHBox(xMinus, xLabel, xPlus)
	yRow := container.NewHBox(yMinus, yLabel, yPlus)
	posRow := container.NewGridWithColumns(2, xRow, yRow)
	return pp.makeField(i18n.T("props.position"), posRow)
}

func (pp *PropertiesPanel) makeAccessoryPositionEditor(a *model.Accessory) fyne.CanvasObject {
	xLabel := canvas.NewText(fmt.Sprintf("%.2f", a.Position.X()), color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	xLabel.TextSize = 12
	xLabel.Alignment = fyne.TextAlignCenter
	yLabel := canvas.NewText(fmt.Sprintf("%.2f", a.Position.Y()), color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	yLabel.TextSize = 12
	yLabel.Alignment = fyne.TextAlignCenter

	xMinus := widget.NewButton("-", func() {
		pp.adjustAccessoryPos(0, -posStep)
		xLabel.Text = fmt.Sprintf("%.2f", pp.exercise.Sequences[pp.seqIndex].Accessories[pp.state.SelectedElement.Index].Position.X())
		xLabel.Refresh()
	})
	xMinus.Importance = widget.LowImportance
	xPlus := widget.NewButton("+", func() {
		pp.adjustAccessoryPos(0, posStep)
		xLabel.Text = fmt.Sprintf("%.2f", pp.exercise.Sequences[pp.seqIndex].Accessories[pp.state.SelectedElement.Index].Position.X())
		xLabel.Refresh()
	})
	xPlus.Importance = widget.LowImportance

	yMinus := widget.NewButton("-", func() {
		pp.adjustAccessoryPos(1, -posStep)
		yLabel.Text = fmt.Sprintf("%.2f", pp.exercise.Sequences[pp.seqIndex].Accessories[pp.state.SelectedElement.Index].Position.Y())
		yLabel.Refresh()
	})
	yMinus.Importance = widget.LowImportance
	yPlus := widget.NewButton("+", func() {
		pp.adjustAccessoryPos(1, posStep)
		yLabel.Text = fmt.Sprintf("%.2f", pp.exercise.Sequences[pp.seqIndex].Accessories[pp.state.SelectedElement.Index].Position.Y())
		yLabel.Refresh()
	})
	yPlus.Importance = widget.LowImportance

	xRow := container.NewHBox(xMinus, xLabel, xPlus)
	yRow := container.NewHBox(yMinus, yLabel, yPlus)
	posRow := container.NewGridWithColumns(2, xRow, yRow)
	return pp.makeField(i18n.T("props.position"), posRow)
}

// --- Layout helpers ---

func (pp *PropertiesPanel) makeSection(title string) fyne.CanvasObject {
	lbl := canvas.NewText(strings.ToUpper(title), theme.ColorCoach)
	lbl.TextSize = 11
	lbl.TextStyle.Bold = true
	return container.NewVBox(widget.NewSeparator(), container.NewPadded(lbl))
}

func (pp *PropertiesPanel) makeField(label string, w fyne.CanvasObject) fyne.CanvasObject {
	lbl := canvas.NewText(label, color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff})
	lbl.TextSize = 10
	return container.NewVBox(container.NewPadded(lbl), container.NewPadded(w))
}

func (pp *PropertiesPanel) makeReadonly(label, value string) fyne.CanvasObject {
	lbl := canvas.NewText(label, color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff})
	lbl.TextSize = 10
	val := canvas.NewText(value, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	val.TextSize = 12
	return container.NewVBox(container.NewPadded(lbl), container.NewPadded(val))
}

// --- Helpers ---

func roleDisplayLabel(role model.PlayerRole) string {
	switch role {
	case model.RoleAttacker:
		return i18n.T("tool.player.attacker")
	case model.RoleDefender:
		return i18n.T("tool.player.defender")
	case model.RoleCoach:
		return i18n.T("tool.player.coach")
	case model.RolePointGuard:
		return i18n.T("tool.player.pg")
	case model.RoleShootingGuard:
		return i18n.T("tool.player.sg")
	case model.RoleSmallForward:
		return i18n.T("tool.player.sf")
	case model.RolePowerForward:
		return i18n.T("tool.player.pf")
	case model.RoleCenter:
		return i18n.T("tool.player.center")
	default:
		return string(role)
	}
}

func refString(ref model.ActionRef) string {
	if ref.IsPlayer {
		return ref.PlayerID
	}
	return fmt.Sprintf("(%.2f, %.2f)", ref.Position.X(), ref.Position.Y())
}
