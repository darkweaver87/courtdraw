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
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/store"
	"github.com/darkweaver87/courtdraw/internal/ui/icon"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
)

// MatchLive is the full-screen live match scoring view.
// Layout: top header bar + two side-by-side columns (on-court | bench).
// Each player row: [#] [Name] [foul dots] [time] [F+] [+1] [+2] [+3].
// Substitution via tap-tap: tap bench player (highlights) then tap on-court player (swap).
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

	// Substitution state (tap-tap, bidirectional).
	selectedSubID    string // member ID selected for sub (can be on-court or bench)
	selectedSubOnCourt bool  // true if selected player is on court

	// UI header elements.
	homeNameText     *canvas.Text
	awayNameText     *canvas.Text
	homeScoreText    *canvas.Text
	awayScoreText    *canvas.Text
	periodScoresText *canvas.Text // "P1 12-45 | P2 8-42"
	periodText       *canvas.Text
	clockText        *canvas.Text
	playPauseBtn     *widget.Button

	// On-court and bench columns.
	oncourtBox    *fyne.Container
	benchBox      *fyne.Container

	// Cached time labels for lightweight tick updates (no rebuild).
	playerTimeTexts map[string]*canvas.Text // memberID → time label

	// Root widget.
	root fyne.CanvasObject
}

// NewMatchLive creates the live match view.
func NewMatchLive(w fyne.Window, match *model.Match, team *model.Team, s store.Store, onExit func()) *MatchLive {
	ml := &MatchLive{
		window:          w,
		match:           match,
		team:            team,
		store:           s,
		OnExit:          onExit,
		currentPeriod:   1,
		playerTimeTexts: make(map[string]*canvas.Text),
	}

	// Sort roster once for consistent ordering everywhere.
	match.SortRoster()

	// Restore state from existing events (resuming a saved match).
	for _, e := range match.Events {
		if e.Type == model.EventPeriodStart && e.Period > ml.currentPeriod {
			ml.currentPeriod = e.Period
		}
		if e.Timestamp > ml.matchSeconds {
			ml.matchSeconds = e.Timestamp
		}
	}

	// Scoreboard names.
	homeName := match.TeamName
	awayName := match.Opponent
	if match.HomeAway == "away" {
		homeName = match.Opponent
		awayName = match.TeamName
	}

	white := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}

	ml.homeNameText = canvas.NewText(homeName, white)
	ml.homeNameText.TextSize = 16
	ml.homeNameText.TextStyle.Bold = true
	ml.homeNameText.Alignment = fyne.TextAlignTrailing

	ml.awayNameText = canvas.NewText(awayName, white)
	ml.awayNameText.TextSize = 16
	ml.awayNameText.TextStyle.Bold = true
	ml.awayNameText.Alignment = fyne.TextAlignLeading

	ml.homeScoreText = canvas.NewText(fmt.Sprintf("%d", match.HomeScore), white)
	ml.homeScoreText.TextSize = 36
	ml.homeScoreText.TextStyle.Bold = true
	ml.homeScoreText.Alignment = fyne.TextAlignCenter

	ml.awayScoreText = canvas.NewText(fmt.Sprintf("%d", match.AwayScore), white)
	ml.awayScoreText.TextSize = 36
	ml.awayScoreText.TextStyle.Bold = true
	ml.awayScoreText.Alignment = fyne.TextAlignCenter

	ml.periodScoresText = canvas.NewText(match.PeriodScoresText(), color.NRGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff})
	ml.periodScoresText.TextSize = 10
	ml.periodScoresText.Alignment = fyne.TextAlignCenter

	ml.periodText = canvas.NewText(
		fmt.Sprintf(i18n.T(i18n.KeyMatchLivePeriod), ml.currentPeriod),
		color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff},
	)
	ml.periodText.TextSize = 13
	ml.periodText.Alignment = fyne.TextAlignCenter

	periodDuration := time.Duration(match.PeriodDurationMinutes()) * time.Minute
	// Compute elapsed time in current period to restore clock on resume.
	periodStartSecs := 0
	for _, e := range match.Events {
		if e.Type == model.EventPeriodStart && e.Period == ml.currentPeriod {
			periodStartSecs = e.Timestamp
		}
	}
	elapsedInPeriod := time.Duration(ml.matchSeconds-periodStartSecs) * time.Second
	ml.clockRemain = periodDuration - elapsedInPeriod
	if ml.clockRemain < 0 {
		ml.clockRemain = 0
	}

	ml.clockText = canvas.NewText(ml.formatDuration(ml.clockRemain), theme.ColorTimerOK)
	ml.clockText.TextSize = 28
	ml.clockText.TextStyle.Bold = true
	ml.clockText.Alignment = fyne.TextAlignCenter

	// Play/Pause toggle button.
	ml.playPauseBtn = widget.NewButtonWithIcon("", icon.Play(), func() {
		ml.toggleClock()
	})
	ml.playPauseBtn.Importance = widget.HighImportance

	// Next period button.
	nextPeriodBtn := widget.NewButtonWithIcon("", icon.Next(), func() {
		ml.confirmNextPeriod()
	})
	nextPeriodBtn.Importance = widget.MediumImportance

	// Quit button.
	quitBtn := widget.NewButtonWithIcon(i18n.T(i18n.KeyMatchLiveQuit), icon.Back(), func() {
		ml.confirmQuit()
	})
	quitBtn.Importance = widget.DangerImportance

	// On-court and bench containers.
	ml.oncourtBox = container.NewVBox()
	ml.benchBox = container.NewVBox()

	// Set match status to live.
	if match.Status == "planned" {
		match.Status = "live"
		ml.autoSave()
	}

	// Build the layout.
	ml.root = ml.buildLayout(quitBtn, nextPeriodBtn)

	// Initial UI refresh.
	ml.refreshLineup()

	return ml
}

func (ml *MatchLive) buildLayout(quitBtn, nextPeriodBtn *widget.Button) fyne.CanvasObject {
	// --- Header bar ---
	// [Quit] [HomeName  SCORE : SCORE  AwayName] [Play/Pause] [NextPeriod] [Period X  MM:SS]
	scoreBg := canvas.NewRectangle(color.NRGBA{R: 0x1a, G: 0x1a, B: 0x2e, A: 0xff})
	scoreBg.CornerRadius = 8

	colonText := canvas.NewText(":", color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	colonText.TextSize = 36
	colonText.TextStyle.Bold = true
	colonText.Alignment = fyne.TextAlignCenter

	// Team-level score buttons.
	homeP1 := widget.NewButton("+1", func() { ml.addTeamScore(true, 1) })
	homeP2 := widget.NewButton("+2", func() { ml.addTeamScore(true, 2) })
	homeP3 := widget.NewButton("+3", func() { ml.addTeamScore(true, 3) })
	homeP1.Importance = widget.LowImportance
	homeP2.Importance = widget.LowImportance
	homeP3.Importance = widget.LowImportance
	homeScoreBtns := container.NewHBox(homeP1, homeP2, homeP3)

	awayP1 := widget.NewButton("+1", func() { ml.addTeamScore(false, 1) })
	awayP2 := widget.NewButton("+2", func() { ml.addTeamScore(false, 2) })
	awayP3 := widget.NewButton("+3", func() { ml.addTeamScore(false, 3) })
	awayP1.Importance = widget.LowImportance
	awayP2.Importance = widget.LowImportance
	awayP3.Importance = widget.LowImportance
	awayScoreBtns := container.NewHBox(awayP1, awayP2, awayP3)

	scoreCenter := container.NewHBox(
		ml.homeScoreText,
		colonText,
		ml.awayScoreText,
	)

	// Header layout — stacked for mobile, horizontal for desktop.
	var headerContent *fyne.Container
	if isMobile {
		// Line 1: HomeTeam  SCORE : SCORE  AwayTeam
		scoreLine := container.NewHBox(
			ml.homeNameText, layout.NewSpacer(),
			scoreCenter,
			layout.NewSpacer(), ml.awayNameText,
		)
		// Line 2: [Quit] [▶] [⏭] [Période X] [MM:SS]
		controlsLine := container.NewHBox(
			quitBtn, ml.playPauseBtn, nextPeriodBtn,
			layout.NewSpacer(),
			ml.periodText, ml.clockText,
		)
		// Line 3: [home +1+2+3]  [away +1+2+3]
		scoreBtnsLine := container.NewHBox(
			homeScoreBtns, layout.NewSpacer(), awayScoreBtns,
		)
		headerContent = container.NewVBox(scoreLine, ml.periodScoresText, controlsLine, scoreBtnsLine)
	} else {
		scoreLine := container.NewHBox(
			ml.homeNameText, layout.NewSpacer(),
			scoreCenter,
			layout.NewSpacer(), ml.awayNameText,
		)
		clockInfo := container.NewHBox(ml.periodText, ml.clockText)
		controlsLine := container.NewHBox(
			quitBtn, homeScoreBtns,
			layout.NewSpacer(),
			ml.playPauseBtn, nextPeriodBtn, clockInfo,
			layout.NewSpacer(),
			awayScoreBtns,
		)
		headerContent = container.NewVBox(scoreLine, ml.periodScoresText, controlsLine)
	}
	header := container.NewStack(scoreBg, container.NewPadded(headerContent))

	// --- Two-column body ---
	oncourtHeader := newSectionHeader(i18n.T(i18n.KeyMatchLiveOncourt))
	benchHeader := newSectionHeader(i18n.T(i18n.KeyMatchLiveBench))

	oncourtCol := container.NewVBox(oncourtHeader, ml.oncourtBox)
	benchCol := container.NewVBox(benchHeader, ml.benchBox)

	oncourtScroll := container.NewVScroll(oncourtCol)
	benchScroll := container.NewVScroll(benchCol)

	columns := container.NewGridWithColumns(2, oncourtScroll, benchScroll)

	return container.NewBorder(header, nil, nil, nil, columns)
}

// Widget returns the match live root layout.
func (ml *MatchLive) Widget() fyne.CanvasObject {
	return ml.root
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
		ml.playPauseBtn.SetIcon(icon.Play())
	} else {
		ml.startClock()
		ml.playPauseBtn.SetIcon(icon.Pause())
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
	ml.clockTicker = time.NewTicker(1 * time.Second)
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
						ml.playPauseBtn.SetIcon(icon.Play())
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
		ml.showPeriodSummary(true)
		return
	}

	ml.showPeriodSummary(false)
}

func (ml *MatchLive) showPeriodSummary(isFinal bool) {
	if isFinal {
		title := fmt.Sprintf(i18n.T(i18n.KeyMatchHalftimeTitle), ml.currentPeriod)
		msg := fmt.Sprintf(i18n.T(i18n.KeyMatchHalftimeSummary),
			ml.match.HomeScore, ml.match.AwayScore)
		dialog.ShowConfirm(title, msg+"\n\n"+i18n.T(i18n.KeyMatchLiveEndMatch)+"?", func(end bool) {
			if end {
				ml.finishMatch()
			} else {
				ml.advancePeriod()
			}
		}, ml.window)
	} else {
		ml.advancePeriod()
	}
}

func (ml *MatchLive) confirmNextPeriod() {
	ml.stopClock()
	ml.playPauseBtn.SetIcon(icon.Play())

	// End current period if clock was running.
	ml.match.AddEvent(model.MatchEvent{
		Type:      model.EventPeriodEnd,
		Timestamp: ml.matchSeconds,
		Period:    ml.currentPeriod,
	})
	ml.autoSave()

	totalPeriods := ml.match.TotalPeriods()
	if ml.currentPeriod >= totalPeriods {
		ml.showPeriodSummary(true)
		return
	}

	ml.showPeriodSummary(false)
}

func (ml *MatchLive) advancePeriod() {
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
}

// --- Scoring ---

func (ml *MatchLive) addPlayerScore(memberID string, points int) {
	// Determine if the player's team is home.
	isHome := ml.match.HomeAway == "home"

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
		PlayerID:  memberID,
	})
	ml.refreshPeriodScores()
	ml.autoSave()
	ml.refreshLineup()
}

func (ml *MatchLive) addTeamScore(isHome bool, points int) {
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
	ml.refreshPeriodScores()
	ml.autoSave()
}

func (ml *MatchLive) refreshPeriodScores() {
	ml.periodScoresText.Text = ml.match.PeriodScoresText()
	ml.periodScoresText.Refresh()
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

// --- Substitution (tap-tap, bidirectional) ---

func (ml *MatchLive) toggleSubSelection(memberID string, isOnCourt bool) {
	// Tap same player → deselect.
	if ml.selectedSubID == memberID {
		ml.selectedSubID = ""
		ml.refreshLineup()
		return
	}

	// If a player from the OTHER column is already selected → perform swap.
	if ml.selectedSubID != "" && ml.selectedSubOnCourt != isOnCourt {
		var benchID, oncourtID string
		if isOnCourt {
			oncourtID = memberID
			benchID = ml.selectedSubID
		} else {
			benchID = memberID
			oncourtID = ml.selectedSubID
		}
		ml.match.Substitute(benchID, oncourtID, ml.matchSeconds, ml.currentPeriod)
		ml.selectedSubID = ""
		ml.autoSave()
		ml.refreshLineup()
		return
	}

	// Select this player.
	ml.selectedSubID = memberID
	ml.selectedSubOnCourt = isOnCourt
	ml.refreshLineup()
}

// --- Lineup display ---

func (ml *MatchLive) refreshLineup() {
	ml.oncourtBox.RemoveAll()
	ml.benchBox.RemoveAll()
	ml.playerTimeTexts = make(map[string]*canvas.Text)

	lineupIDs := ml.match.CurrentLineup()
	onCourtSet := make(map[string]bool, len(lineupIDs))
	for _, id := range lineupIDs {
		onCourtSet[id] = true
	}

	// Compute average playing time for fairness coloring.
	totalTime := 0
	playerCount := 0
	for _, r := range ml.match.Roster {
		pt := ml.match.PlayerPlayingSeconds(r.MemberID, ml.matchSeconds)
		totalTime += pt
		playerCount++
	}
	avgTime := 0
	if playerCount > 0 {
		avgTime = totalTime / playerCount
	}

	// Roster is pre-sorted by SortRoster() — iterate in order.
	for _, r := range ml.match.Roster {
		if onCourtSet[r.MemberID] {
			card := ml.buildPlayerRow(r, true, avgTime)
			ml.oncourtBox.Add(card)
		}
	}
	for _, r := range ml.match.Roster {
		if !onCourtSet[r.MemberID] {
			card := ml.buildPlayerRow(r, false, avgTime)
			ml.benchBox.Add(card)
		}
	}

	ml.oncourtBox.Refresh()
	ml.benchBox.Refresh()
}

func (ml *MatchLive) buildPlayerRow(r model.RosterEntry, onCourt bool, avgTime int) fyne.CanvasObject {
	fouls := ml.match.PlayerFouls(r.MemberID)
	playingSecs := ml.match.PlayerPlayingSeconds(r.MemberID, ml.matchSeconds)
	playerPts := ml.match.PlayerScorePoints(r.MemberID)
	playingTime := fmt.Sprintf("%d:%02d", playingSecs/60, playingSecs%60)

	// Jersey number — color reflects foul state.
	numColor := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	if fouls >= 5 {
		numColor = color.NRGBA{R: 0xff, G: 0x33, B: 0x33, A: 0xff} // red
	} else if fouls == 4 {
		numColor = color.NRGBA{R: 0xff, G: 0xaa, B: 0x33, A: 0xff} // orange
	}
	numText := canvas.NewText(fmt.Sprintf("#%d", r.Number), numColor)
	numText.TextSize = 18
	numText.TextStyle.Bold = true

	// Name.
	nameColor := color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff}
	if r.MemberID == ml.selectedSubID {
		nameColor = color.NRGBA{R: 0x44, G: 0x88, B: 0xff, A: 0xff} // blue — selected for sub
	} else if !onCourt && playingSecs == 0 && !r.Starting {
		nameColor = color.NRGBA{R: 0xff, G: 0xd7, B: 0x00, A: 0xff} // yellow — hasn't played
	}
	nameText := canvas.NewText(r.FirstName+" "+r.LastName, nameColor)
	nameText.TextSize = 13

	// Foul dots.
	foulDots := ml.buildFoulDots(fouls)

	// Playing time — colored by fairness relative to average.
	timeColor := color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff} // default gray
	if avgTime > 0 && ml.matchSeconds > 30 {
		ratio := float64(playingSecs) / float64(avgTime)
		if ratio < 0.5 {
			timeColor = color.NRGBA{R: 0xff, G: 0x44, B: 0x44, A: 0xff} // red — way below average
		} else if ratio < 0.75 {
			timeColor = color.NRGBA{R: 0xff, G: 0xaa, B: 0x33, A: 0xff} // orange — below average
		} else if ratio > 1.25 {
			timeColor = color.NRGBA{R: 0x44, G: 0xbb, B: 0x44, A: 0xff} // green — above average
		}
	}
	timeText := canvas.NewText(playingTime, timeColor)
	timeText.TextSize = 11
	timeText.TextStyle.Bold = true
	ml.playerTimeTexts[r.MemberID] = timeText

	// Points label (shown only if > 0).
	var ptsObj fyne.CanvasObject
	if playerPts > 0 {
		ptsLabel := canvas.NewText(fmt.Sprintf("%dpts", playerPts), color.NRGBA{R: 0x4c, G: 0xaf, B: 0x50, A: 0xff})
		ptsLabel.TextSize = 12
		ptsLabel.TextStyle.Bold = true
		ptsObj = ptsLabel
	} else {
		ptsObj = layout.NewSpacer()
	}

	// Info column: number + name on one line, foul dots + time + points on next.
	infoTop := container.NewHBox(numText, nameText)
	infoBottom := container.NewHBox(foulDots, layout.NewSpacer(), ptsObj, timeText)
	info := container.NewVBox(infoTop, infoBottom)

	// Action buttons — compact icons.
	btnSize := fyne.NewSize(36, 36)

	foulBtn := NewTipButton(icon.FoulIcon, i18n.T(i18n.KeyMatchFoulAdd), func() {
		ml.addFoul(r.MemberID)
	})

	plus1 := NewTipButton(icon.ScorePlus1, "+1", func() { ml.addPlayerScore(r.MemberID, 1) })
	plus2 := NewTipButton(icon.ScorePlus2, "+2", func() { ml.addPlayerScore(r.MemberID, 2) })
	plus3 := NewTipButton(icon.ScorePlus3, "+3", func() { ml.addPlayerScore(r.MemberID, 3) })

	// Sub button — swapin for bench (enter court), swapout for on-court (leave court).
	var subRes fyne.Resource
	if onCourt {
		subRes = icon.SwapOut
	} else {
		subRes = icon.SwapIn
	}
	if r.MemberID == ml.selectedSubID {
		subRes = fynetheme.CancelIcon()
	}
	subBtn := NewTipButton(subRes, i18n.T(i18n.KeyMatchLiveSub), func() {
		ml.toggleSubSelection(r.MemberID, onCourt)
	})
	if r.MemberID == ml.selectedSubID {
		subBtn.OverrideColor = toolActiveColor
	} else if ml.selectedSubID != "" && ml.selectedSubOnCourt != onCourt {
		// Opposite column — highlight as valid swap target.
		subBtn.OverrideColor = &color.NRGBA{R: 0x4c, G: 0xaf, B: 0x50, A: 0xff} // green
	}

	var buttons *fyne.Container
	if isMobile {
		buttons = container.NewHBox(
			container.NewGridWrap(btnSize, foulBtn),
			container.NewGridWrap(btnSize, plus1),
			container.NewGridWrap(btnSize, plus2),
			container.NewGridWrap(btnSize, plus3),
			container.NewGridWrap(btnSize, subBtn),
		)
		// Mobile: stack info + buttons vertically to avoid overflow.
		return container.NewVBox(info, buttons, widget.NewSeparator())
	}
	buttons = container.NewHBox(
		container.NewGridWrap(btnSize, foulBtn),
		container.NewGridWrap(btnSize, plus1),
		container.NewGridWrap(btnSize, plus2),
		container.NewGridWrap(btnSize, plus3),
		container.NewGridWrap(btnSize, subBtn),
	)
	content := container.NewBorder(nil, nil, nil, buttons, info)
	return container.NewVBox(content, widget.NewSeparator())
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
	// Lightweight update: only change time text values, don't rebuild buttons.
	for _, r := range ml.match.Roster {
		if txt, ok := ml.playerTimeTexts[r.MemberID]; ok {
			secs := ml.match.PlayerPlayingSeconds(r.MemberID, ml.matchSeconds)
			newTime := fmt.Sprintf("%d:%02d", secs/60, secs%60)
			if txt.Text != newTime {
				txt.Text = newTime
				txt.Refresh()
			}
		}
	}
}

// --- End match ---

func (ml *MatchLive) confirmQuit() {
	dialog.ShowConfirm(
		i18n.T(i18n.KeyMatchLiveQuit),
		i18n.T(i18n.KeyMatchLiveQuitConfirm),
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
