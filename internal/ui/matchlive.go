package ui

import (
	"fmt"
	"image/color"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/store"
	"github.com/darkweaver87/courtdraw/internal/ui/icon"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
)

// MatchLive is the full-screen live match scoring view.
type MatchLive struct {
	window fyne.Window
	match  *model.Match
	team   *model.Team
	store  store.Store
	OnExit func()

	// Clock state.
	clockMu      sync.Mutex
	clockRunning bool
	clockRemain  time.Duration
	clockTicker  *time.Ticker
	clockDone    chan struct{}
	currentPeriod int
	matchSeconds  int           // cumulative match seconds for event timestamps
	matchSubSecs  time.Duration // sub-second accumulator for matchSeconds

	// Substitution state.
	selectedBenchID string // member ID of bench player selected for sub

	// UI elements.
	homeNameText  *canvas.Text
	awayNameText  *canvas.Text
	homeScoreText *canvas.Text
	awayScoreText *canvas.Text
	periodText    *canvas.Text
	clockText     *canvas.Text

	// On-court player cards.
	oncourtBox  *fyne.Container
	benchBox    *fyne.Container
	benchScroll *container.Scroll

	// Controls.
	startPeriodBtn *widget.Button
	endMatchBtn    *widget.Button

	// Responsive layout.
	responsive *ResponsiveContainer
}

// NewMatchLive creates the live match view.
func NewMatchLive(w fyne.Window, match *model.Match, team *model.Team, s store.Store, onExit func()) *MatchLive {
	ml := &MatchLive{
		window:        w,
		match:         match,
		team:          team,
		store:         s,
		OnExit:        onExit,
		currentPeriod: 1,
	}

	// Determine current period from events.
	for _, e := range match.Events {
		if e.Type == model.EventPeriodStart && e.Period > ml.currentPeriod {
			ml.currentPeriod = e.Period
		}
	}

	// Scoreboard header.
	homeName := match.TeamName
	awayName := match.Opponent
	if match.HomeAway == "away" {
		homeName = match.Opponent
		awayName = match.TeamName
	}

	ml.homeNameText = canvas.NewText(homeName, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	ml.homeNameText.TextSize = 18
	ml.homeNameText.TextStyle.Bold = true
	ml.homeNameText.Alignment = fyne.TextAlignCenter

	ml.awayNameText = canvas.NewText(awayName, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	ml.awayNameText.TextSize = 18
	ml.awayNameText.TextStyle.Bold = true
	ml.awayNameText.Alignment = fyne.TextAlignCenter

	ml.homeScoreText = canvas.NewText(fmt.Sprintf("%d", match.HomeScore), color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	ml.homeScoreText.TextSize = 48
	ml.homeScoreText.TextStyle.Bold = true
	ml.homeScoreText.Alignment = fyne.TextAlignCenter

	ml.awayScoreText = canvas.NewText(fmt.Sprintf("%d", match.AwayScore), color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	ml.awayScoreText.TextSize = 48
	ml.awayScoreText.TextStyle.Bold = true
	ml.awayScoreText.Alignment = fyne.TextAlignCenter

	ml.periodText = canvas.NewText(
		fmt.Sprintf(i18n.T(i18n.KeyMatchLivePeriod), ml.currentPeriod),
		color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff},
	)
	ml.periodText.TextSize = 14
	ml.periodText.Alignment = fyne.TextAlignCenter

	periodDuration := time.Duration(match.PeriodDurationMinutes()) * time.Minute
	ml.clockRemain = periodDuration

	ml.clockText = canvas.NewText(ml.formatDuration(periodDuration), theme.ColorTimerOK)
	ml.clockText.TextSize = 32
	ml.clockText.TextStyle.Bold = true
	ml.clockText.Alignment = fyne.TextAlignCenter

	// On-court and bench containers.
	ml.oncourtBox = container.NewVBox()
	ml.benchBox = container.NewVBox()
	ml.benchScroll = container.NewVScroll(ml.benchBox)
	ml.benchScroll.SetMinSize(fyne.NewSize(0, 80))

	// Controls.
	ml.startPeriodBtn = widget.NewButton(i18n.T(i18n.KeyMatchLiveStartPeriod), func() {
		ml.toggleClock()
	})
	ml.startPeriodBtn.Importance = widget.HighImportance

	ml.endMatchBtn = widget.NewButton(i18n.T(i18n.KeyMatchLiveEndMatch), func() {
		ml.confirmEndMatch()
	})
	ml.endMatchBtn.Importance = widget.DangerImportance

	// Score buttons.
	// Home score.
	homePlus1 := widget.NewButton(i18n.T(i18n.KeyMatchScorePlus1), func() { ml.addScore(true, 1) })
	homePlus1.Importance = widget.HighImportance
	homePlus2 := widget.NewButton(i18n.T(i18n.KeyMatchScorePlus2), func() { ml.addScore(true, 2) })
	homePlus2.Importance = widget.HighImportance
	homePlus3 := widget.NewButton(i18n.T(i18n.KeyMatchScorePlus3), func() { ml.addScore(true, 3) })
	homePlus3.Importance = widget.HighImportance

	// Away score.
	awayPlus1 := widget.NewButton(i18n.T(i18n.KeyMatchScorePlus1), func() { ml.addScore(false, 1) })
	awayPlus1.Importance = widget.MediumImportance
	awayPlus2 := widget.NewButton(i18n.T(i18n.KeyMatchScorePlus2), func() { ml.addScore(false, 2) })
	awayPlus2.Importance = widget.MediumImportance
	awayPlus3 := widget.NewButton(i18n.T(i18n.KeyMatchScorePlus3), func() { ml.addScore(false, 3) })
	awayPlus3.Importance = widget.MediumImportance

	homeScoreBtns := container.NewGridWithColumns(3, homePlus1, homePlus2, homePlus3)
	awayScoreBtns := container.NewGridWithColumns(3, awayPlus1, awayPlus2, awayPlus3)

	// Timeout button.
	timeoutBtn := widget.NewButton(i18n.T(i18n.KeyMatchLiveTimeout), func() {
		ml.addTimeout()
	})
	timeoutBtn.Importance = widget.LowImportance

	// Quit button.
	quitBtn := widget.NewButtonWithIcon(i18n.T(i18n.KeyMatchLiveEndMatch), icon.Back(), func() {
		ml.confirmQuit()
	})
	quitBtn.Importance = widget.DangerImportance

	// Set match status to live.
	if match.Status == "planned" {
		match.Status = "live"
		ml.autoSave()
	}

	// Build responsive layout.
	ml.responsive = NewResponsiveContainer(
		func() fyne.CanvasObject {
			return ml.buildLayout(homeScoreBtns, awayScoreBtns, timeoutBtn, quitBtn)
		},
		func() fyne.CanvasObject {
			return ml.buildLayout(homeScoreBtns, awayScoreBtns, timeoutBtn, quitBtn)
		},
	)

	// Initial UI refresh.
	ml.refreshLineup()

	return ml
}

func (ml *MatchLive) buildLayout(homeScoreBtns, awayScoreBtns fyne.CanvasObject, timeoutBtn, quitBtn *widget.Button) fyne.CanvasObject {
	// Scoreboard.
	scoreBg := canvas.NewRectangle(color.NRGBA{R: 0x1a, G: 0x1a, B: 0x2e, A: 0xff})
	scoreBg.CornerRadius = 8

	homeCol := container.NewVBox(ml.homeNameText, ml.homeScoreText)
	awayCol := container.NewVBox(ml.awayNameText, ml.awayScoreText)
	centerCol := container.NewVBox(ml.periodText, ml.clockText)

	scoreBoard := container.NewStack(scoreBg, container.NewPadded(
		container.NewHBox(
			layout.NewSpacer(),
			homeCol,
			layout.NewSpacer(),
			centerCol,
			layout.NewSpacer(),
			awayCol,
			layout.NewSpacer(),
		),
	))

	// Score buttons row.
	scoreRow := container.NewGridWithColumns(2, homeScoreBtns, awayScoreBtns)

	// On-court header.
	oncourtHeader := newSectionHeader(i18n.T(i18n.KeyMatchLiveOncourt))
	benchHeader := newSectionHeader(i18n.T(i18n.KeyMatchLiveBench))

	// Clock + period controls.
	clockControls := container.NewHBox(
		ml.startPeriodBtn,
		timeoutBtn,
	)

	// Main content.
	oncourtSection := container.NewVBox(oncourtHeader, ml.oncourtBox)
	benchSection := container.NewVBox(benchHeader, ml.benchScroll)

	rightPanel := container.NewVBox(
		scoreRow,
		widget.NewSeparator(),
		clockControls,
		widget.NewSeparator(),
		oncourtSection,
		widget.NewSeparator(),
		benchSection,
	)
	rightScroll := container.NewVScroll(rightPanel)

	bottom := container.NewHBox(layout.NewSpacer(), quitBtn, ml.endMatchBtn)

	return container.NewBorder(
		scoreBoard,
		container.NewPadded(bottom),
		nil, nil,
		rightScroll,
	)
}

// Widget returns the match live root layout.
func (ml *MatchLive) Widget() fyne.CanvasObject {
	return ml.responsive
}

// Stop halts the clock and cleans up.
func (ml *MatchLive) Stop() {
	ml.stopClock()
}

// --- Clock ---

func (ml *MatchLive) toggleClock() {
	ml.clockMu.Lock()
	running := ml.clockRunning
	ml.clockMu.Unlock()

	if running {
		ml.stopClock()
		ml.startPeriodBtn.SetText(i18n.T(i18n.KeyMatchLiveStartPeriod))
	} else {
		ml.startClock()
		ml.startPeriodBtn.SetText(i18n.T(i18n.KeyMatchLiveEndPeriod))
	}
}

func (ml *MatchLive) startClock() {
	ml.clockMu.Lock()
	if ml.clockRunning {
		ml.clockMu.Unlock()
		return
	}
	ml.clockRunning = true
	ml.clockDone = make(chan struct{})
	ml.clockTicker = time.NewTicker(100 * time.Millisecond)
	ml.clockMu.Unlock()

	// Add period start event if this is a new period.
	hasPeriodStart := false
	for _, e := range ml.match.Events {
		if e.Type == model.EventPeriodStart && e.Period == ml.currentPeriod {
			hasPeriodStart = true
			break
		}
	}
	if !hasPeriodStart {
		ml.match.AddEvent(model.MatchEvent{
			Type:      model.EventPeriodStart,
			Timestamp: ml.matchSeconds,
			Period:    ml.currentPeriod,
		})
		ml.autoSave()
	}

	go func() {
		last := time.Now()
		for {
			select {
			case <-ml.clockDone:
				return
			case now := <-ml.clockTicker.C:
				dt := now.Sub(last)
				last = now
				ml.clockMu.Lock()
				ml.clockRemain -= dt
				ml.matchSubSecs += dt
				if ml.matchSubSecs >= time.Second {
					secs := int(ml.matchSubSecs.Seconds())
					ml.matchSeconds += secs
					ml.matchSubSecs -= time.Duration(secs) * time.Second
				}
				rem := ml.clockRemain
				ml.clockMu.Unlock()

				periodEnded := rem <= 0

				fyne.Do(func() {
					if rem < 0 {
						ml.clockText.Text = "00:00"
						ml.clockText.Color = theme.ColorTimerExpired
					} else {
						ml.clockText.Text = ml.formatDuration(rem)
						ml.clockText.Color = theme.ColorTimerOK
					}
					ml.clockText.Refresh()
					ml.refreshPlayerTimes()
				})

				if periodEnded {
					ml.stopClock()
					fyne.Do(func() {
						ml.onPeriodEnd()
					})
					return
				}
			}
		}
	}()
}

func (ml *MatchLive) stopClock() {
	ml.clockMu.Lock()
	defer ml.clockMu.Unlock()
	if !ml.clockRunning {
		return
	}
	ml.clockRunning = false
	ml.clockTicker.Stop()
	close(ml.clockDone)
}

func (ml *MatchLive) onPeriodEnd() {
	ml.match.AddEvent(model.MatchEvent{
		Type:      model.EventPeriodEnd,
		Timestamp: ml.matchSeconds,
		Period:    ml.currentPeriod,
	})
	ml.autoSave()

	totalPeriods := ml.match.TotalPeriods()
	if ml.currentPeriod >= totalPeriods {
		// Match can end or go to overtime.
		ml.startPeriodBtn.SetText(i18n.T(i18n.KeyMatchLiveStartPeriod))
		ml.showPeriodSummary(true)
		return
	}

	ml.showPeriodSummary(false)
}

func (ml *MatchLive) showPeriodSummary(isFinal bool) {
	title := fmt.Sprintf(i18n.T(i18n.KeyMatchHalftimeTitle), ml.currentPeriod)
	msg := fmt.Sprintf(i18n.T(i18n.KeyMatchHalftimeSummary),
		ml.match.HomeScore, ml.match.AwayScore)

	if isFinal {
		dialog.ShowConfirm(title, msg+"\n\n"+i18n.T(i18n.KeyMatchLiveEndMatch)+"?", func(end bool) {
			if end {
				ml.finishMatch()
			} else {
				// Overtime.
				ml.nextPeriod()
			}
		}, ml.window)
	} else {
		dialog.ShowInformation(title, msg, ml.window)
		ml.nextPeriod()
	}
}

func (ml *MatchLive) nextPeriod() {
	ml.currentPeriod++
	periodDuration := time.Duration(ml.match.PeriodDurationMinutes()) * time.Minute
	ml.clockMu.Lock()
	ml.clockRemain = periodDuration
	ml.clockMu.Unlock()

	ml.periodText.Text = fmt.Sprintf(i18n.T(i18n.KeyMatchLivePeriod), ml.currentPeriod)
	ml.periodText.Refresh()
	ml.clockText.Text = ml.formatDuration(periodDuration)
	ml.clockText.Color = theme.ColorTimerOK
	ml.clockText.Refresh()
	ml.startPeriodBtn.SetText(i18n.T(i18n.KeyMatchLiveStartPeriod))
}

// --- Scoring ---

func (ml *MatchLive) addScore(isHome bool, points int) {
	if isHome {
		ml.match.HomeScore += points
		ml.homeScoreText.Text = fmt.Sprintf("%d", ml.match.HomeScore)
		ml.homeScoreText.Refresh()
	} else {
		ml.match.AwayScore += points
		ml.awayScoreText.Text = fmt.Sprintf("%d", ml.match.AwayScore)
		ml.awayScoreText.Refresh()
	}

	ml.match.AddEvent(model.MatchEvent{
		Type:      model.EventScore,
		Timestamp: ml.matchSeconds,
		Period:    ml.currentPeriod,
		Points:    points,
		IsHome:    isHome,
	})
	ml.autoSave()
}

// --- Fouls ---

func (ml *MatchLive) addFoul(memberID string) {
	ml.match.AddEvent(model.MatchEvent{
		Type:      model.EventFoul,
		Timestamp: ml.matchSeconds,
		Period:    ml.currentPeriod,
		PlayerID:  memberID,
	})
	ml.autoSave()
	ml.refreshLineup()

	fouls := ml.match.PlayerFouls(memberID)
	name := ml.playerName(memberID)
	if fouls >= 5 {
		dialog.ShowInformation(
			i18n.T(i18n.KeyMatchFoulTitle),
			fmt.Sprintf(i18n.T(i18n.KeyMatchFoulFouledOut), name),
			ml.window,
		)
	} else if fouls == 4 {
		dialog.ShowInformation(
			i18n.T(i18n.KeyMatchFoulTitle),
			fmt.Sprintf(i18n.T(i18n.KeyMatchFoulWarning), name, fouls),
			ml.window,
		)
	}
}

// --- Timeout ---

func (ml *MatchLive) addTimeout() {
	ml.stopClock()
	ml.match.AddEvent(model.MatchEvent{
		Type:      model.EventTimeout,
		Timestamp: ml.matchSeconds,
		Period:    ml.currentPeriod,
	})
	ml.autoSave()
	ml.startPeriodBtn.SetText(i18n.T(i18n.KeyMatchLiveStartPeriod))
}

// --- Substitution ---

func (ml *MatchLive) selectBenchPlayer(memberID string) {
	if ml.selectedBenchID == memberID {
		// Deselect.
		ml.selectedBenchID = ""
		ml.refreshLineup()
		return
	}
	ml.selectedBenchID = memberID
	ml.refreshLineup()
}

func (ml *MatchLive) substitutePlayer(oncourtID string) {
	if ml.selectedBenchID == "" {
		return
	}
	ml.match.Substitute(ml.selectedBenchID, oncourtID, ml.matchSeconds, ml.currentPeriod)
	ml.selectedBenchID = ""
	ml.autoSave()
	ml.refreshLineup()
}

// --- Lineup display ---

func (ml *MatchLive) refreshLineup() {
	ml.oncourtBox.RemoveAll()
	ml.benchBox.RemoveAll()

	lineupIDs := ml.match.CurrentLineup()
	onCourtSet := make(map[string]bool, len(lineupIDs))
	for _, id := range lineupIDs {
		onCourtSet[id] = true
	}

	// On-court players.
	for _, r := range ml.match.Roster {
		if !onCourtSet[r.MemberID] {
			continue
		}
		card := ml.buildPlayerCard(r, true)
		ml.oncourtBox.Add(card)
	}

	// Bench players.
	for _, r := range ml.match.Roster {
		if onCourtSet[r.MemberID] {
			continue
		}
		card := ml.buildPlayerCard(r, false)
		ml.benchBox.Add(card)
	}

	ml.oncourtBox.Refresh()
	ml.benchBox.Refresh()
}

func (ml *MatchLive) buildPlayerCard(r model.RosterEntry, onCourt bool) fyne.CanvasObject {
	fouls := ml.match.PlayerFouls(r.MemberID)
	playingSecs := ml.match.PlayerPlayingSeconds(r.MemberID, ml.matchSeconds)
	playingTime := fmt.Sprintf("%d:%02d", playingSecs/60, playingSecs%60)

	// Jersey number (large).
	numText := canvas.NewText(fmt.Sprintf("#%d", r.Number), color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	numText.TextSize = 20
	numText.TextStyle.Bold = true

	// Name.
	nameText := canvas.NewText(r.FirstName, color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff})
	nameText.TextSize = 14

	// Foul dots.
	foulDots := ml.buildFoulDots(fouls)

	// Playing time.
	timeText := canvas.NewText(playingTime, color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff})
	timeText.TextSize = 12

	// Card background color based on fouls.
	bgColor := color.NRGBA{R: 0x2a, G: 0x2a, B: 0x2a, A: 0xff}
	if fouls >= 5 {
		bgColor = color.NRGBA{R: 0x88, G: 0x22, B: 0x22, A: 0xff} // red
	} else if fouls == 4 {
		bgColor = color.NRGBA{R: 0x88, G: 0x66, B: 0x22, A: 0xff} // orange
	}

	// Selected bench player highlight.
	if !onCourt && r.MemberID == ml.selectedBenchID {
		bgColor = color.NRGBA{R: 0x22, G: 0x44, B: 0x88, A: 0xff} // blue
	}

	// Hasn't played indicator for bench.
	if !onCourt && playingSecs == 0 && !r.Starting {
		nameText.Color = color.NRGBA{R: 0xff, G: 0xd7, B: 0x00, A: 0xff} // yellow
	}

	bg := canvas.NewRectangle(bgColor)
	bg.CornerRadius = 6

	info := container.NewVBox(
		container.NewHBox(numText, nameText),
		container.NewHBox(foulDots, layout.NewSpacer(), timeText),
	)

	var buttons fyne.CanvasObject
	if onCourt {
		foulBtn := widget.NewButton(i18n.T(i18n.KeyMatchFoulAdd), func() {
			ml.addFoul(r.MemberID)
		})
		foulBtn.Importance = widget.LowImportance

		subBtn := widget.NewButton(i18n.T(i18n.KeyMatchLiveSub), func() {
			ml.substitutePlayer(r.MemberID)
		})
		subBtn.Importance = widget.LowImportance

		buttons = container.NewHBox(foulBtn, subBtn)
	} else {
		tapBtn := widget.NewButton(i18n.T(i18n.KeyMatchLiveSelectPlayerIn), func() {
			ml.selectBenchPlayer(r.MemberID)
		})
		tapBtn.Importance = widget.LowImportance
		buttons = tapBtn
	}

	content := container.NewBorder(nil, nil, nil, buttons, info)
	return container.NewStack(bg, container.NewPadded(content))
}

func (ml *MatchLive) buildFoulDots(fouls int) fyne.CanvasObject {
	maxFouls := 5
	dots := container.NewHBox()
	for i := 0; i < maxFouls; i++ {
		var c color.NRGBA
		if i < fouls {
			c = color.NRGBA{R: 0xff, G: 0x44, B: 0x44, A: 0xff} // filled red
		} else {
			c = color.NRGBA{R: 0x55, G: 0x55, B: 0x55, A: 0xff} // empty gray
		}
		dot := canvas.NewRectangle(c)
		dot.SetMinSize(fyne.NewSize(10, 10))
		dot.CornerRadius = 5
		dots.Add(dot)
	}
	return dots
}

func (ml *MatchLive) refreshPlayerTimes() {
	// Refresh lineup to update playing times.
	ml.refreshLineup()
}

// --- End match ---

func (ml *MatchLive) confirmEndMatch() {
	dialog.ShowConfirm(
		i18n.T(i18n.KeyMatchLiveEndMatch),
		fmt.Sprintf("%d - %d", ml.match.HomeScore, ml.match.AwayScore),
		func(ok bool) {
			if !ok {
				return
			}
			ml.finishMatch()
		},
		ml.window,
	)
}

func (ml *MatchLive) confirmQuit() {
	dialog.ShowConfirm(
		i18n.T(i18n.KeyMatchLiveEndMatch),
		i18n.T(i18n.KeyMatchLiveEndMatch),
		func(ok bool) {
			if !ok {
				return
			}
			ml.Stop()
			if ml.OnExit != nil {
				ml.OnExit()
			}
		},
		ml.window,
	)
}

func (ml *MatchLive) finishMatch() {
	ml.Stop()
	ml.match.Status = "finished"
	ml.autoSave()
	if ml.OnExit != nil {
		ml.OnExit()
	}
}

// --- Helpers ---

func (ml *MatchLive) autoSave() {
	_ = ml.store.SaveMatch(ml.match)
}

func (ml *MatchLive) playerName(memberID string) string {
	for _, r := range ml.match.Roster {
		if r.MemberID == memberID {
			return r.FirstName + " " + r.LastName
		}
	}
	return memberID
}

func (ml *MatchLive) formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalSecs := int(d.Seconds())
	m := totalSecs / 60
	s := totalSecs % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}
