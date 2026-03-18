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
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
)

// ---------- Countdown Timer ----------

// CountdownTimer is a configurable countdown timer with start/pause/reset.
type CountdownTimer struct {
	duration  time.Duration
	remaining time.Duration
	running   bool
	mu        sync.Mutex
	ticker    *time.Ticker
	done      chan struct{}

	display  *canvas.Text
	startBtn *widget.Button
	resetBtn *widget.Button
	content  *fyne.Container
	controls *fyne.Container

	OnAlert func()
	alerted bool
}

// NewCountdownTimer creates a countdown timer with +/- buttons.
func NewCountdownTimer() *CountdownTimer {
	ct := &CountdownTimer{
		duration:  60 * time.Second,
		remaining: 60 * time.Second,
	}

	ct.display = canvas.NewText("01:00", theme.ColorTimerOK)
	ct.display.TextSize = 32
	ct.display.TextStyle.Bold = true
	ct.display.Alignment = fyne.TextAlignCenter

	ct.startBtn = widget.NewButton(i18n.T("coach.start"), func() {
		ct.mu.Lock()
		running := ct.running
		ct.mu.Unlock()
		if running {
			ct.pause()
		} else {
			ct.start()
		}
	})
	ct.startBtn.Importance = widget.HighImportance

	ct.resetBtn = widget.NewButton(i18n.T("coach.reset"), func() {
		ct.reset()
	})
	ct.resetBtn.Importance = widget.LowImportance

	// +/- buttons to adjust duration.
	var adjustRow fyne.CanvasObject
	if isMobile {
		// Mobile: 2 large icon-only buttons (−/+), step = 1 minute.
		subBtn := widget.NewButtonWithIcon("", fynetheme.ContentRemoveIcon(), func() {
			ct.adjustDuration(-60 * time.Second)
		})
		subBtn.Importance = widget.MediumImportance
		addBtn := widget.NewButtonWithIcon("", fynetheme.ContentAddIcon(), func() {
			ct.adjustDuration(60 * time.Second)
		})
		addBtn.Importance = widget.MediumImportance
		adjustRow = container.NewGridWithColumns(2, subBtn, addBtn)
	} else {
		// Desktop: 4 text buttons with fine control.
		subMin := widget.NewButton("-1m", func() { ct.adjustDuration(-60 * time.Second) })
		subMin.Importance = widget.LowImportance
		sub10 := widget.NewButton("-10s", func() { ct.adjustDuration(-10 * time.Second) })
		sub10.Importance = widget.LowImportance
		add10 := widget.NewButton("+10s", func() { ct.adjustDuration(10 * time.Second) })
		add10.Importance = widget.LowImportance
		addMin := widget.NewButton("+1m", func() { ct.adjustDuration(60 * time.Second) })
		addMin.Importance = widget.LowImportance
		adjustRow = container.NewGridWithColumns(4, subMin, sub10, add10, addMin)
	}

	ct.content = container.NewVBox(
		ct.display,
		adjustRow,
	)
	ct.controls = container.NewGridWithColumns(2, ct.startBtn, ct.resetBtn)

	return ct
}

func (ct *CountdownTimer) adjustDuration(delta time.Duration) {
	ct.mu.Lock()
	running := ct.running
	ct.remaining += delta
	if ct.remaining < 0 {
		ct.remaining = 0
	}
	if !running {
		ct.duration = ct.remaining
	}
	rem := ct.remaining
	ct.mu.Unlock()

	ct.refreshDisplay(rem)
}

func (ct *CountdownTimer) start() {
	ct.mu.Lock()
	if ct.running {
		ct.mu.Unlock()
		return
	}
	if ct.remaining <= 0 {
		ct.remaining = ct.duration
		ct.alerted = false
	}
	ct.running = true
	ct.done = make(chan struct{})
	ct.ticker = time.NewTicker(100 * time.Millisecond)
	ct.mu.Unlock()

	ct.startBtn.SetText(i18n.T("coach.pause"))

	go func() {
		last := time.Now()
		for {
			select {
			case <-ct.done:
				return
			case now := <-ct.ticker.C:
				dt := now.Sub(last)
				last = now
				ct.mu.Lock()
				ct.remaining -= dt
				rem := ct.remaining
				alerted := ct.alerted
				ct.mu.Unlock()

				if rem <= 0 && !alerted {
					ct.mu.Lock()
					ct.alerted = true
					ct.mu.Unlock()
					if ct.OnAlert != nil {
						ct.OnAlert()
					}
				}

				fyne.Do(func() {
					ct.mu.Lock()
					still := ct.running
					ct.mu.Unlock()
					if still {
						ct.refreshDisplay(rem)
					}
				})
			}
		}
	}()
}

func (ct *CountdownTimer) pause() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	if !ct.running {
		return
	}
	ct.running = false
	ct.ticker.Stop()
	close(ct.done)
	ct.startBtn.SetText(i18n.T("coach.start"))
}

func (ct *CountdownTimer) reset() {
	ct.pause()
	ct.mu.Lock()
	ct.remaining = ct.duration
	ct.alerted = false
	rem := ct.remaining
	ct.mu.Unlock()
	ct.refreshDisplay(rem)
	ct.startBtn.SetText(i18n.T("coach.start"))
}

func (ct *CountdownTimer) refreshDisplay(rem time.Duration) {
	if rem < 0 {
		ct.display.Color = theme.ColorTimerExpired
		totalSecs := int((-rem).Seconds())
		m := totalSecs / 60
		s := totalSecs % 60
		ct.display.Text = fmt.Sprintf("-%02d:%02d", m, s)
	} else {
		ct.display.Color = theme.ColorTimerOK
		totalSecs := int(math.Ceil(rem.Seconds()))
		m := totalSecs / 60
		s := totalSecs % 60
		ct.display.Text = fmt.Sprintf("%02d:%02d", m, s)
	}
	ct.display.Refresh()
}

// Stop halts the timer.
func (ct *CountdownTimer) Stop() {
	ct.pause()
}

// ContentWidget returns the countdown display area.
func (ct *CountdownTimer) ContentWidget() fyne.CanvasObject {
	return ct.content
}

// ControlsWidget returns the countdown control buttons.
func (ct *CountdownTimer) ControlsWidget() fyne.CanvasObject {
	return ct.controls
}

// ---------- Stopwatch ----------

// Stopwatch counts up from zero.
type Stopwatch struct {
	elapsed time.Duration
	running bool
	mu      sync.Mutex
	ticker  *time.Ticker
	done    chan struct{}

	display  *canvas.Text
	startBtn *widget.Button
	resetBtn *widget.Button
	content  *fyne.Container
	controls *fyne.Container
}

// NewStopwatch creates a new stopwatch.
func NewStopwatch() *Stopwatch {
	sw := &Stopwatch{}

	sw.display = canvas.NewText("00:00.000", color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	sw.display.TextSize = 32
	sw.display.TextStyle.Bold = true
	sw.display.Alignment = fyne.TextAlignCenter

	sw.startBtn = widget.NewButton(i18n.T("coach.start"), func() {
		sw.mu.Lock()
		running := sw.running
		sw.mu.Unlock()
		if running {
			sw.stop()
		} else {
			sw.start()
		}
	})
	sw.startBtn.Importance = widget.HighImportance

	sw.resetBtn = widget.NewButton(i18n.T("coach.reset"), func() {
		sw.resetTimer()
	})
	sw.resetBtn.Importance = widget.LowImportance

	sw.content = container.NewVBox(sw.display)
	sw.controls = container.NewGridWithColumns(2, sw.startBtn, sw.resetBtn)

	return sw
}

func (sw *Stopwatch) start() {
	sw.mu.Lock()
	if sw.running {
		sw.mu.Unlock()
		return
	}
	sw.running = true
	sw.done = make(chan struct{})
	sw.ticker = time.NewTicker(10 * time.Millisecond)
	sw.mu.Unlock()

	sw.startBtn.SetText(i18n.T("coach.pause"))

	go func() {
		last := time.Now()
		for {
			select {
			case <-sw.done:
				return
			case now := <-sw.ticker.C:
				dt := now.Sub(last)
				last = now
				sw.mu.Lock()
				sw.elapsed += dt
				e := sw.elapsed
				sw.mu.Unlock()

				fyne.Do(func() {
					sw.mu.Lock()
					still := sw.running
					sw.mu.Unlock()
					if !still {
						return
					}
					total := int(e.Seconds())
					m := total / 60
					s := total % 60
					ms := e.Milliseconds() % 1000
					sw.display.Text = fmt.Sprintf("%02d:%02d.%03d", m, s, ms)
					sw.display.Refresh()
				})
			}
		}
	}()
}

func (sw *Stopwatch) stop() {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	if !sw.running {
		return
	}
	sw.running = false
	sw.ticker.Stop()
	close(sw.done)
	sw.startBtn.SetText(i18n.T("coach.start"))
}

func (sw *Stopwatch) resetTimer() {
	sw.stop()
	sw.mu.Lock()
	sw.elapsed = 0
	sw.mu.Unlock()
	sw.display.Text = "00:00.000"
	sw.display.Refresh()
	sw.startBtn.SetText(i18n.T("coach.start"))
}

// Stop halts the stopwatch.
func (sw *Stopwatch) Stop() {
	sw.stop()
}

// ContentWidget returns the stopwatch display area.
func (sw *Stopwatch) ContentWidget() fyne.CanvasObject {
	return sw.content
}

// ControlsWidget returns the stopwatch control buttons.
func (sw *Stopwatch) ControlsWidget() fyne.CanvasObject {
	return sw.controls
}

// ---------- Luc Léger ----------

// legerStage defines a stage of the Luc Léger (20m shuttle run) test.
type legerStage struct {
	Speed    float64 // km/h
	Shuttles int     // number of shuttles in this stage
}

// Official Luc Léger 20m shuttle run protocol.
var legerProtocol = []legerStage{
	{8.5, 7},
	{9.0, 8},
	{9.5, 8},
	{10.0, 9},
	{10.5, 9},
	{11.0, 10},
	{11.5, 10},
	{12.0, 11},
	{12.5, 11},
	{13.0, 11},
	{13.5, 12},
	{14.0, 12},
	{14.5, 13},
	{15.0, 13},
	{15.5, 13},
	{16.0, 14},
	{16.5, 14},
	{17.0, 15},
	{17.5, 15},
	{18.0, 16},
	{18.5, 16},
}

// LucLeger implements the Luc Léger beep test.
type LucLeger struct {
	stage   int // 0-based index into legerProtocol
	shuttle int // 0-based within current stage
	running bool
	mu      sync.Mutex
	ticker  *time.Ticker
	done    chan struct{}

	stageLabel   *canvas.Text
	shuttleLabel *canvas.Text
	speedLabel   *canvas.Text
	startBtn     *widget.Button
	resetBtn     *widget.Button
	content      *fyne.Container
	controls     *fyne.Container

	OnBeep func()
}

// NewLucLeger creates a Luc Léger beep test UI.
func NewLucLeger() *LucLeger {
	ll := &LucLeger{}

	ll.stageLabel = canvas.NewText(fmt.Sprintf(i18n.T("coach.stage"), 1), color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	ll.stageLabel.TextSize = 28
	ll.stageLabel.TextStyle.Bold = true
	ll.stageLabel.Alignment = fyne.TextAlignCenter

	ll.shuttleLabel = canvas.NewText(fmt.Sprintf(i18n.T("coach.shuttle"), 0, legerProtocol[0].Shuttles),
		color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff})
	ll.shuttleLabel.TextSize = 18
	ll.shuttleLabel.Alignment = fyne.TextAlignCenter

	ll.speedLabel = canvas.NewText(fmt.Sprintf("%.1f km/h", legerProtocol[0].Speed),
		color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff})
	ll.speedLabel.TextSize = 14
	ll.speedLabel.Alignment = fyne.TextAlignCenter

	ll.startBtn = widget.NewButton(i18n.T("coach.start"), func() {
		ll.mu.Lock()
		running := ll.running
		ll.mu.Unlock()
		if running {
			ll.pause()
		} else {
			ll.start()
		}
	})
	ll.startBtn.Importance = widget.HighImportance

	ll.resetBtn = widget.NewButton(i18n.T("coach.reset"), func() {
		ll.resetTest()
	})
	ll.resetBtn.Importance = widget.LowImportance

	ll.content = container.NewVBox(ll.stageLabel, ll.shuttleLabel, ll.speedLabel)
	ll.controls = container.NewGridWithColumns(2, ll.startBtn, ll.resetBtn)

	return ll
}

// shuttleInterval returns the time between beeps for a given stage.
func shuttleInterval(stage int) time.Duration {
	if stage < 0 || stage >= len(legerProtocol) {
		stage = len(legerProtocol) - 1
	}
	speedMS := legerProtocol[stage].Speed * 1000.0 / 3600.0 // m/s
	return time.Duration(20.0/speedMS*1000) * time.Millisecond
}

func (ll *LucLeger) start() {
	ll.mu.Lock()
	if ll.running {
		ll.mu.Unlock()
		return
	}
	ll.running = true
	ll.done = make(chan struct{})
	interval := shuttleInterval(ll.stage)
	ll.ticker = time.NewTicker(interval)
	ll.mu.Unlock()

	ll.startBtn.SetText(i18n.T("coach.pause"))

	go func() {
		for {
			select {
			case <-ll.done:
				return
			case <-ll.ticker.C:
				ll.mu.Lock()
				ll.shuttle++
				stg := legerProtocol[ll.stage]
				if ll.shuttle >= stg.Shuttles {
					ll.shuttle = 0
					ll.stage++
					if ll.stage >= len(legerProtocol) {
						ll.stage = len(legerProtocol) - 1
						ll.running = false
						ll.ticker.Stop()
						ll.mu.Unlock()
						fyne.Do(func() {
							ll.refreshLabels()
							ll.startBtn.SetText(i18n.T("coach.start"))
						})
						return
					}
					// Update ticker interval for new stage.
					ll.ticker.Stop()
					ll.ticker = time.NewTicker(shuttleInterval(ll.stage))
				}
				ll.mu.Unlock()

				if ll.OnBeep != nil {
					ll.OnBeep()
				}

				fyne.Do(func() {
					ll.mu.Lock()
					still := ll.running
					ll.mu.Unlock()
					if still {
						ll.refreshLabels()
					}
				})
			}
		}
	}()
}

func (ll *LucLeger) pause() {
	ll.mu.Lock()
	defer ll.mu.Unlock()
	if !ll.running {
		return
	}
	ll.running = false
	ll.ticker.Stop()
	close(ll.done)
	ll.startBtn.SetText(i18n.T("coach.start"))
}

func (ll *LucLeger) resetTest() {
	ll.pause()
	ll.mu.Lock()
	ll.stage = 0
	ll.shuttle = 0
	ll.mu.Unlock()
	ll.refreshLabels()
	ll.startBtn.SetText(i18n.T("coach.start"))
}

func (ll *LucLeger) refreshLabels() {
	ll.mu.Lock()
	stage := ll.stage
	shuttle := ll.shuttle
	ll.mu.Unlock()

	stg := legerProtocol[stage]
	ll.stageLabel.Text = fmt.Sprintf(i18n.T("coach.stage"), stage+1)
	ll.stageLabel.Refresh()
	ll.shuttleLabel.Text = fmt.Sprintf(i18n.T("coach.shuttle"), shuttle, stg.Shuttles)
	ll.shuttleLabel.Refresh()
	ll.speedLabel.Text = fmt.Sprintf("%.1f km/h", stg.Speed)
	ll.speedLabel.Refresh()
}

// Stop halts the Luc Léger test.
func (ll *LucLeger) Stop() {
	ll.pause()
}

// ContentWidget returns the Luc Léger display area.
func (ll *LucLeger) ContentWidget() fyne.CanvasObject {
	return ll.content
}

// ControlsWidget returns the Luc Léger control buttons.
func (ll *LucLeger) ControlsWidget() fyne.CanvasObject {
	return ll.controls
}

// ---------- CoachToolsPanel ----------

// CoachToolsPanel groups all coach tools with manual tab buttons.
type CoachToolsPanel struct {
	countdown *CountdownTimer
	stopwatch *Stopwatch
	leger     *LucLeger
}

// NewCoachToolsPanel creates the coach tools panel.
func NewCoachToolsPanel() *CoachToolsPanel {
	return &CoachToolsPanel{
		countdown: NewCountdownTimer(),
		stopwatch: NewStopwatch(),
		leger:     NewLucLeger(),
	}
}

// BuildWidget creates the coach tools UI. Must be called at layout time.
func (p *CoachToolsPanel) BuildWidget() fyne.CanvasObject {
	type toolPanel struct {
		content  fyne.CanvasObject
		controls fyne.CanvasObject
	}
	panels := []toolPanel{
		{p.countdown.ContentWidget(), p.countdown.ControlsWidget()},
		{p.stopwatch.ContentWidget(), p.stopwatch.ControlsWidget()},
		{p.leger.ContentWidget(), p.leger.ControlsWidget()},
	}

	labels := []string{
		i18n.T("coach.countdown"),
		i18n.T("coach.stopwatch"),
		i18n.T("coach.leger"),
	}

	// Fixed-height content area so layout doesn't shift between tools.
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(0, 100))
	contentArea := container.NewStack(spacer, panels[0].content)
	controlsArea := container.NewStack(panels[0].controls)

	tabBtns := make([]*widget.Button, 0, len(labels))
	for i, label := range labels {
		idx := i
		btn := widget.NewButton(label, nil)
		btn.OnTapped = func() {
			contentArea.Objects = []fyne.CanvasObject{spacer, panels[idx].content}
			contentArea.Refresh()
			controlsArea.Objects = []fyne.CanvasObject{panels[idx].controls}
			controlsArea.Refresh()
			for j, b := range tabBtns {
				if j == idx {
					b.Importance = widget.HighImportance
				} else {
					b.Importance = widget.MediumImportance
				}
				b.Refresh()
			}
		}
		tabBtns = append(tabBtns, btn)
	}

	// Select first tab.
	tabBtns[0].Importance = widget.HighImportance

	tabBar := container.NewHBox()
	for _, btn := range tabBtns {
		tabBar.Add(btn)
	}

	return container.NewBorder(tabBar, controlsArea, nil, nil, contentArea)
}

// Stop halts all running tools.
func (p *CoachToolsPanel) Stop() {
	p.countdown.Stop()
	p.stopwatch.Stop()
	p.leger.Stop()
}

// Widget returns the coach tools UI (lazy build).
func (p *CoachToolsPanel) Widget() fyne.CanvasObject {
	return p.BuildWidget()
}

// SetAlertCallback sets the alert callback for countdown and Luc Léger beep.
func (p *CoachToolsPanel) SetAlertCallback(fn func()) {
	p.countdown.OnAlert = fn
	p.leger.OnBeep = fn
}
