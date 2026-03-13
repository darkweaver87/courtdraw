package ui

import (
	"fmt"
	"image/color"
	"math"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/anim"
	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/ui/fynecourt"
	"github.com/darkweaver87/courtdraw/internal/ui/icon"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
)

// TrainingMode is the full-screen view for running a session during practice.
type TrainingMode struct {
	window    fyne.Window
	exercises []*model.Exercise // resolved, ordered
	session   *model.Session

	currentIdx int
	seqIdx     int

	// Court widget (read-only).
	court *fynecourt.CourtWidget

	// Animation playback.
	playback *anim.Playback

	// Navigation.
	prevBtn      *widget.Button
	nextBtn      *widget.Button
	progressText *canvas.Text
	nameText     *canvas.Text

	// Progress bar (colored segments).
	progressBar *fyne.Container

	// Sequence navigation + animation controls.
	seqPrevBtn *widget.Button
	seqNextBtn *widget.Button
	seqLabel   *canvas.Text
	playBtn    *widget.Button
	pauseBtn   *widget.Button
	speedBtn   *widget.Button

	// Description.
	descText *widget.Label

	// Instructions.
	instrBox    *fyne.Container
	instrScroll *container.Scroll

	// Metadata.
	categoryText  *canvas.Text
	intensityDots *fyne.Container
	durationText  *canvas.Text

	// Exercise timer (timebox).
	timerDisplay *canvas.Text
	timerBanner  *canvas.Text
	timerMu      sync.Mutex
	timerRunning bool
	timerDone    chan struct{}
	timerTicker  *time.Ticker
	timerRemain  time.Duration

	// Coach tools.
	coachTools *CoachToolsPanel

	// Quit callback.
	quitBtn *widget.Button
	OnExit  func()

	// Responsive.
	responsive *ResponsiveContainer
}

// NewTrainingMode creates the training mode view.
func NewTrainingMode(w fyne.Window, session *model.Session, exercises []*model.Exercise, onExit func()) *TrainingMode {
	tm := &TrainingMode{
		window:    w,
		session:   session,
		exercises: exercises,
		OnExit:    onExit,
	}

	// Court widget — read-only (no editor state).
	tm.court = fynecourt.NewCourtWidget()

	// Navigation buttons.
	tm.prevBtn = widget.NewButtonWithIcon("", icon.Prev(), func() { tm.navigatePrev() })
	tm.prevBtn.Importance = widget.MediumImportance
	tm.nextBtn = widget.NewButtonWithIcon("", icon.Next(), func() { tm.navigateNext() })
	tm.nextBtn.Importance = widget.MediumImportance

	tm.progressText = canvas.NewText("1 / 1", color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff})
	tm.progressText.TextSize = 14
	tm.progressText.Alignment = fyne.TextAlignCenter

	tm.nameText = canvas.NewText("", color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	tm.nameText.TextSize = 16
	tm.nameText.TextStyle.Bold = true
	tm.nameText.Alignment = fyne.TextAlignCenter

	// Progress bar.
	tm.progressBar = container.NewHBox()

	// Animation controls.
	tm.playBtn = widget.NewButtonWithIcon("", icon.Play(), func() { tm.playAnimation() })
	tm.playBtn.Importance = widget.HighImportance
	tm.pauseBtn = widget.NewButtonWithIcon("", icon.Pause(), func() { tm.pauseAnimation() })
	tm.pauseBtn.Importance = widget.HighImportance
	tm.pauseBtn.Hide()
	tm.speedBtn = widget.NewButton("1.0x", func() { tm.cycleSpeed() })
	tm.speedBtn.Importance = widget.LowImportance

	// Sequence navigation.
	tm.seqPrevBtn = widget.NewButtonWithIcon("", icon.Prev(), func() { tm.navigateSeqPrev() })
	tm.seqPrevBtn.Importance = widget.LowImportance
	tm.seqNextBtn = widget.NewButtonWithIcon("", icon.Next(), func() { tm.navigateSeqNext() })
	tm.seqNextBtn.Importance = widget.LowImportance
	tm.seqLabel = canvas.NewText("", color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff})
	tm.seqLabel.TextSize = 12
	tm.seqLabel.Alignment = fyne.TextAlignCenter

	// Description.
	tm.descText = widget.NewLabel("")
	tm.descText.Wrapping = fyne.TextWrapWord

	// Instructions.
	tm.instrBox = container.NewVBox()
	tm.instrScroll = container.NewVScroll(tm.instrBox)
	tm.instrScroll.SetMinSize(fyne.NewSize(0, 100))

	// Metadata.
	tm.categoryText = canvas.NewText("", color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff})
	tm.categoryText.TextSize = 12
	tm.intensityDots = container.NewHBox()
	tm.durationText = canvas.NewText("", color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff})
	tm.durationText.TextSize = 12

	// Exercise timer.
	tm.timerDisplay = canvas.NewText("--:--", theme.ColorTimerOK)
	tm.timerDisplay.TextSize = 28
	tm.timerDisplay.TextStyle.Bold = true
	tm.timerDisplay.Alignment = fyne.TextAlignCenter

	tm.timerBanner = canvas.NewText("", theme.ColorTimerExpired)
	tm.timerBanner.TextSize = 14
	tm.timerBanner.TextStyle.Bold = true
	tm.timerBanner.Alignment = fyne.TextAlignCenter

	// Coach tools.
	tm.coachTools = NewCoachToolsPanel()
	tm.coachTools.SetAlertCallback(systemBeep)

	// Quit button.
	tm.quitBtn = widget.NewButtonWithIcon(i18n.T("training.quit"), icon.Back(), func() {
		tm.Stop()
		if tm.OnExit != nil {
			tm.OnExit()
		}
	})
	tm.quitBtn.Importance = widget.DangerImportance

	// Responsive layout.
	tm.responsive = NewResponsiveContainer(tm.buildDesktopLayout, tm.buildMobileLayout)

	// Load first exercise.
	if len(tm.exercises) > 0 {
		tm.loadExercise(0)
	}

	return tm
}

// Widget returns the training mode root layout.
func (tm *TrainingMode) Widget() fyne.CanvasObject {
	return tm.responsive
}

// Stop halts all timers and cleans up.
func (tm *TrainingMode) Stop() {
	tm.stopAnimation()
	tm.stopExerciseTimer()
	tm.coachTools.Stop()
}

// --- Animation ---

func (tm *TrainingMode) playAnimation() {
	if tm.playback == nil {
		return
	}
	tm.playback.SetLoop(true)
	tm.playback.Play()
	tm.court.SetAnimMode(true)
	tm.playBtn.Hide()
	tm.pauseBtn.Show()
}

func (tm *TrainingMode) pauseAnimation() {
	if tm.playback == nil {
		return
	}
	tm.playback.Pause()
	tm.court.SetAnimMode(false)
	// Sync sequence index from playback.
	tm.seqIdx = tm.playback.SeqIndex()
	tm.court.SetSequence(tm.seqIdx)
	tm.pauseBtn.Hide()
	tm.playBtn.Show()
	tm.refreshSeqUI()
}

func (tm *TrainingMode) stopAnimation() {
	if tm.playback != nil {
		tm.playback.Stop()
		tm.court.SetAnimMode(false)
	}
	tm.pauseBtn.Hide()
	tm.playBtn.Show()
}

func (tm *TrainingMode) cycleSpeed() {
	if tm.playback == nil {
		return
	}
	tm.playback.CycleSpeed()
	tm.speedBtn.SetText(fmt.Sprintf("%.1fx", tm.playback.Speed()))
}

// --- Navigation ---

func (tm *TrainingMode) navigatePrev() {
	if tm.currentIdx > 0 {
		tm.loadExercise(tm.currentIdx - 1)
	}
}

func (tm *TrainingMode) navigateNext() {
	if tm.currentIdx < len(tm.exercises)-1 {
		tm.loadExercise(tm.currentIdx + 1)
	}
}

func (tm *TrainingMode) navigateSeqPrev() {
	if tm.seqIdx > 0 {
		tm.stopAnimation()
		tm.seqIdx--
		tm.court.SetSequence(tm.seqIdx)
		if tm.playback != nil {
			tm.playback.SetSeqIndex(tm.seqIdx)
		}
		tm.refreshSeqUI()
	}
}

func (tm *TrainingMode) navigateSeqNext() {
	ex := tm.exercises[tm.currentIdx]
	if tm.seqIdx < len(ex.Sequences)-1 {
		tm.stopAnimation()
		tm.seqIdx++
		tm.court.SetSequence(tm.seqIdx)
		if tm.playback != nil {
			tm.playback.SetSeqIndex(tm.seqIdx)
		}
		tm.refreshSeqUI()
	}
}

// --- Exercise loading ---

func (tm *TrainingMode) loadExercise(idx int) {
	if idx < 0 || idx >= len(tm.exercises) {
		return
	}

	// Stop previous animation.
	tm.stopAnimation()

	tm.currentIdx = idx
	tm.seqIdx = 0
	ex := tm.exercises[idx]

	// Court.
	tm.court.SetExercise(ex)
	tm.court.SetSequence(0)

	// Playback — create and auto-play if multiple sequences.
	tm.playback = anim.NewPlayback(ex)
	tm.court.SetPlayback(tm.playback)
	if len(ex.Sequences) > 1 {
		tm.playback.SetLoop(true)
		tm.playback.Play()
		tm.court.SetAnimMode(true)
		tm.playBtn.Hide()
		tm.pauseBtn.Show()
	} else {
		tm.court.SetAnimMode(false)
		tm.playBtn.Show()
		tm.pauseBtn.Hide()
	}

	// Navigation.
	tm.progressText.Text = fmt.Sprintf(i18n.T("training.progress"), idx+1, len(tm.exercises))
	tm.progressText.Refresh()

	tm.nameText.Text = ex.Name
	tm.nameText.Refresh()

	tm.prevBtn.Enable()
	tm.nextBtn.Enable()
	if idx == 0 {
		tm.prevBtn.Disable()
	}
	if idx == len(tm.exercises)-1 {
		tm.nextBtn.Disable()
	}

	// Metadata.
	if ex.Category != "" {
		tm.categoryText.Text = i18n.T("category." + string(ex.Category))
		tm.categoryText.Color = theme.CategoryColor(ex.Category)
	} else {
		tm.categoryText.Text = ""
	}
	tm.categoryText.Refresh()

	tm.durationText.Text = ex.Duration
	if tm.durationText.Text == "" {
		tm.durationText.Text = i18n.T("training.no_duration")
	}
	tm.durationText.Refresh()

	// Intensity dots.
	tm.intensityDots.Objects = intensityColorDots(int(ex.Intensity))
	tm.intensityDots.Refresh()

	// Description.
	if ex.Description != "" {
		tm.descText.SetText(ex.Description)
		tm.descText.Show()
	} else {
		tm.descText.SetText("")
		tm.descText.Hide()
	}

	// Progress bar.
	tm.buildProgressBar()

	// Sequence / instructions.
	tm.refreshSeqUI()

	// Timer.
	tm.startExerciseTimer(ex)
}

func (tm *TrainingMode) refreshSeqUI() {
	ex := tm.exercises[tm.currentIdx]
	numSeqs := len(ex.Sequences)

	// Sequence controls visibility.
	if numSeqs <= 1 {
		tm.seqLabel.Text = ""
		tm.seqPrevBtn.Hide()
		tm.seqNextBtn.Hide()
		tm.playBtn.Hide()
		tm.pauseBtn.Hide()
		tm.speedBtn.Hide()
	} else {
		tm.seqLabel.Text = fmt.Sprintf(i18n.T("training.seq"), tm.seqIdx+1, numSeqs)
		tm.seqPrevBtn.Show()
		tm.seqNextBtn.Show()
		tm.speedBtn.Show()
		if tm.seqIdx == 0 {
			tm.seqPrevBtn.Disable()
		} else {
			tm.seqPrevBtn.Enable()
		}
		if tm.seqIdx >= numSeqs-1 {
			tm.seqNextBtn.Disable()
		} else {
			tm.seqNextBtn.Enable()
		}
	}
	tm.seqLabel.Refresh()

	// Instructions — show ALL sequences' instructions, current highlighted.
	tm.instrBox.Objects = nil

	for si, seq := range ex.Sequences {
		if seq.Label == "" && len(seq.Instructions) == 0 {
			continue
		}

		// Sequence header.
		headerColor := color.NRGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff}
		if si == tm.seqIdx {
			headerColor = color.NRGBA{R: 0xff, G: 0xb7, B: 0x03, A: 0xff} // gold for current
		}

		label := seq.Label
		if label == "" {
			label = fmt.Sprintf(i18n.T("training.seq"), si+1, numSeqs)
		}
		lbl := canvas.NewText(label, headerColor)
		lbl.TextSize = 14
		lbl.TextStyle.Bold = true
		tm.instrBox.Add(lbl)

		for _, instr := range seq.Instructions {
			txt := widget.NewLabel("  • " + instr)
			txt.Wrapping = fyne.TextWrapWord
			tm.instrBox.Add(txt)
		}
	}

	tm.instrBox.Refresh()
}

// --- Progress bar ---

func (tm *TrainingMode) buildProgressBar() {
	tm.progressBar.Objects = nil
	done := color.NRGBA{R: 0x4c, G: 0xaf, B: 0x50, A: 0xff}   // green for completed
	todo := color.NRGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xff}   // dark grey for upcoming
	current := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff} // white for current
	for i := range tm.exercises {
		var c color.NRGBA
		switch {
		case i < tm.currentIdx:
			c = done
		case i == tm.currentIdx:
			c = current
		default:
			c = todo
		}
		rect := canvas.NewRectangle(c)
		rect.SetMinSize(fyne.NewSize(0, 8))
		tm.progressBar.Add(rect)
	}
	// Use grid layout so segments share width equally.
	tm.progressBar.Layout = layout.NewGridWrapLayout(fyne.NewSize(
		float32(int(600)/max(len(tm.exercises), 1)), 8))
	tm.progressBar.Refresh()
}

// --- Exercise timer ---

func (tm *TrainingMode) startExerciseTimer(ex *model.Exercise) {
	tm.stopExerciseTimer()

	minutes := parseDurationMinutes(ex.Duration)
	if minutes <= 0 {
		tm.timerDisplay.Text = "--:--"
		tm.timerDisplay.Color = theme.ColorTimerOK
		tm.timerDisplay.Refresh()
		tm.timerBanner.Text = ""
		tm.timerBanner.Refresh()
		return
	}

	tm.timerMu.Lock()
	tm.timerRemain = time.Duration(minutes) * time.Minute
	tm.timerRunning = true
	tm.timerDone = make(chan struct{})
	tm.timerTicker = time.NewTicker(100 * time.Millisecond)
	tm.timerMu.Unlock()

	// Show initial value.
	tm.refreshTimerDisplay(time.Duration(minutes) * time.Minute)
	tm.timerBanner.Text = ""
	tm.timerBanner.Refresh()

	go func() {
		last := time.Now()
		alerted := false
		for {
			select {
			case <-tm.timerDone:
				return
			case now := <-tm.timerTicker.C:
				dt := now.Sub(last)
				last = now
				tm.timerMu.Lock()
				tm.timerRemain -= dt
				rem := tm.timerRemain
				tm.timerMu.Unlock()

				expired := rem <= 0 && !alerted
				if expired {
					alerted = true
				}

				fyne.Do(func() {
					tm.refreshTimerDisplay(rem)
					if expired {
						tm.timerBanner.Text = i18n.T("training.time_expired")
						tm.timerBanner.Refresh()
					}
				})
			}
		}
	}()
}

func (tm *TrainingMode) stopExerciseTimer() {
	tm.timerMu.Lock()
	defer tm.timerMu.Unlock()
	if !tm.timerRunning {
		return
	}
	tm.timerRunning = false
	tm.timerTicker.Stop()
	close(tm.timerDone)
}

func (tm *TrainingMode) refreshTimerDisplay(rem time.Duration) {
	if rem < 0 {
		tm.timerDisplay.Color = theme.ColorTimerExpired
		totalSecs := int((-rem).Seconds())
		m := totalSecs / 60
		s := totalSecs % 60
		tm.timerDisplay.Text = fmt.Sprintf("-%02d:%02d", m, s)
	} else {
		tm.timerDisplay.Color = theme.ColorTimerOK
		totalSecs := int(math.Ceil(rem.Seconds()))
		m := totalSecs / 60
		s := totalSecs % 60
		tm.timerDisplay.Text = fmt.Sprintf("%02d:%02d", m, s)
	}
	tm.timerDisplay.Refresh()
}

// --- Layouts ---

func (tm *TrainingMode) buildDesktopLayout() fyne.CanvasObject {
	// Top bar: quit | prev | progress + name | next | timer
	topBar := container.NewHBox(
		tm.quitBtn,
		tm.prevBtn,
		layout.NewSpacer(),
		container.NewVBox(tm.progressText, tm.nameText),
		layout.NewSpacer(),
		tm.nextBtn,
		tm.timerDisplay,
	)

	// Sequence + animation controls.
	seqBar := container.NewHBox(
		tm.playBtn, tm.pauseBtn,
		tm.seqPrevBtn,
		tm.seqLabel,
		tm.seqNextBtn,
		tm.speedBtn,
	)

	// Metadata bar.
	metaBar := container.NewHBox(
		tm.categoryText,
		tm.intensityDots,
		tm.durationText,
	)

	// Right panel: metadata + seq + description + instructions (center), coach tools (bottom).
	rightTop := container.NewVBox(metaBar, seqBar, tm.timerBanner, tm.descText)
	rightPanel := container.NewBorder(rightTop, tm.coachTools.Widget(), nil, nil, tm.instrScroll)

	// Main: court left, right panel right.
	mainSplit := container.NewHSplit(tm.court, rightPanel)
	mainSplit.SetOffset(0.6)

	return container.NewBorder(
		container.NewVBox(topBar, tm.progressBar),
		nil, nil, nil,
		mainSplit,
	)
}

func (tm *TrainingMode) buildMobileLayout() fyne.CanvasObject {
	// Top bar: quit | progress | timer | prev | next
	topBar := container.NewHBox(
		tm.quitBtn,
		tm.progressText,
		layout.NewSpacer(),
		tm.timerDisplay,
		tm.prevBtn,
		tm.nextBtn,
	)

	// Court tab.
	seqBar := container.NewHBox(tm.playBtn, tm.pauseBtn, tm.seqPrevBtn, tm.seqLabel, tm.seqNextBtn, tm.speedBtn)
	metaBar := container.NewHBox(tm.categoryText, tm.intensityDots, tm.durationText)
	courtTab := container.NewBorder(
		container.NewVBox(topBar, tm.progressBar, tm.nameText, tm.timerBanner),
		container.NewVBox(metaBar, seqBar),
		nil, nil,
		tm.court,
	)

	// Instructions tab — description at top, scrollable instructions fill remaining space.
	instrTab := container.NewBorder(tm.descText, nil, nil, nil, tm.instrScroll)

	// Tools tab.
	toolsTab := tm.coachTools.Widget()

	tabs := container.NewAppTabs(
		container.NewTabItem(i18n.T("mobile.training.court"), courtTab),
		container.NewTabItem(i18n.T("mobile.training.instructions"), instrTab),
		container.NewTabItem(i18n.T("mobile.training.tools"), toolsTab),
	)
	tabs.SetTabLocation(container.TabLocationBottom)

	return tabs
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
