package ui

import (
	"fmt"
	"image"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/ui/icon"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
)

// MatchSummary displays the post-match summary.
type MatchSummary struct {
	match *model.Match
	box   *fyne.Container
}

// NewMatchSummary creates the match summary view.
func NewMatchSummary(match *model.Match, onClose func()) *MatchSummary {
	ms := &MatchSummary{match: match}

	// Final score header.
	homeName := match.TeamName
	awayName := match.Opponent
	if match.HomeAway == "away" {
		homeName = match.Opponent
		awayName = match.TeamName
	}

	scoreHeader := canvas.NewText(
		fmt.Sprintf("%s  %d - %d  %s", homeName, match.HomeScore, match.AwayScore, awayName),
		color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
	)
	scoreHeader.TextSize = 24
	scoreHeader.TextStyle.Bold = true
	scoreHeader.Alignment = fyne.TextAlignCenter

	titleText := canvas.NewText(i18n.T(i18n.KeyMatchSummaryTitle), color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff})
	titleText.TextSize = 16
	titleText.TextStyle.Bold = true
	titleText.Alignment = fyne.TextAlignCenter

	dateText := canvas.NewText(match.Date, color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff})
	dateText.TextSize = 12
	dateText.Alignment = fyne.TextAlignCenter

	// Compute total match seconds for playing time calculations.
	totalMatchSecs := 0
	for _, e := range match.Events {
		if e.Timestamp > totalMatchSecs {
			totalMatchSecs = e.Timestamp
		}
	}

	// Playing time section.
	playingTimeHeader := newSectionHeader(i18n.T(i18n.KeyMatchSummaryPlayingTime))
	playingTimeBox := container.NewVBox()

	// Find max playing time for proportional bars.
	maxPlayTime := 1 // avoid div by zero
	for _, r := range match.Roster {
		pt := match.PlayerPlayingSeconds(r.MemberID, totalMatchSecs)
		if pt > maxPlayTime {
			maxPlayTime = pt
		}
	}

	for _, r := range match.Roster {
		pt := match.PlayerPlayingSeconds(r.MemberID, totalMatchSecs)
		ptStr := fmt.Sprintf("%d:%02d", pt/60, pt%60)

		label := canvas.NewText(
			fmt.Sprintf("#%d %s", r.Number, r.FirstName),
			color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff},
		)
		label.TextSize = 13

		timeLabel := canvas.NewText(ptStr, color.NRGBA{R: 0x99, G: 0x99, B: 0x99, A: 0xff})
		timeLabel.TextSize = 12

		// Proportional bar.
		barWidth := float32(200) * float32(pt) / float32(maxPlayTime)
		if barWidth < 2 {
			barWidth = 2
		}
		bar := canvas.NewRectangle(color.NRGBA{R: 0x44, G: 0x88, B: 0xcc, A: 0xff})
		bar.SetMinSize(fyne.NewSize(barWidth, 12))
		bar.CornerRadius = 3

		row := container.NewBorder(
			nil, nil,
			label, timeLabel,
			container.NewHBox(bar),
		)
		playingTimeBox.Add(row)
	}

	// Scoring section — per-player point totals.
	scoringHeader := newSectionHeader(i18n.T(i18n.KeyMatchSummaryScoring))
	scoringBox := container.NewVBox()

	for _, r := range match.Roster {
		pts := match.PlayerScorePoints(r.MemberID)
		if pts == 0 {
			continue
		}

		label := canvas.NewText(
			fmt.Sprintf("#%d %s %s", r.Number, r.FirstName, r.LastName),
			color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff},
		)
		label.TextSize = 13

		ptsText := canvas.NewText(
			fmt.Sprintf("%d pts", pts),
			color.NRGBA{R: 0x4c, G: 0xaf, B: 0x50, A: 0xff},
		)
		ptsText.TextSize = 13
		ptsText.TextStyle.Bold = true

		row := container.NewBorder(nil, nil, label, ptsText, nil)
		scoringBox.Add(row)
	}

	if len(scoringBox.Objects) == 0 {
		scoringBox.Add(widget.NewLabel(i18n.T(i18n.KeyMatchSummaryNoData)))
	}

	// Fouls section.
	foulsHeader := newSectionHeader(i18n.T(i18n.KeyMatchSummaryFouls))
	foulsBox := container.NewVBox()

	for _, r := range match.Roster {
		fouls := match.PlayerFouls(r.MemberID)
		if fouls == 0 {
			continue
		}

		label := canvas.NewText(
			fmt.Sprintf("#%d %s", r.Number, r.FirstName),
			color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff},
		)
		label.TextSize = 13

		foulText := canvas.NewText(
			fmt.Sprintf("%d", fouls),
			color.NRGBA{R: 0xff, G: 0x44, B: 0x44, A: 0xff},
		)
		foulText.TextSize = 13
		foulText.TextStyle.Bold = true

		row := container.NewBorder(nil, nil, label, foulText, nil)
		foulsBox.Add(row)
	}

	if len(foulsBox.Objects) == 0 {
		foulsBox.Add(widget.NewLabel(i18n.T(i18n.KeyMatchSummaryNoData)))
	}

	// Close button.
	closeBtn := widget.NewButtonWithIcon(i18n.T(i18n.KeyDialogCancel), icon.Back(), func() {
		if onClose != nil {
			onClose()
		}
	})
	closeBtn.Importance = widget.MediumImportance

	// Score evolution graph + period breakdown.
	periodScoresHeader := newSectionHeader(i18n.T(i18n.KeyMatchSummaryPeriodScores))
	periodScoresBox := container.NewVBox()

	// Period breakdown text.
	ps := match.PeriodScores()
	maxP := 0
	for p := range ps {
		if p > maxP {
			maxP = p
		}
	}
	for p := 1; p <= maxP; p++ {
		s := ps[p]
		row := canvas.NewText(
			fmt.Sprintf("P%d :  %d - %d", p, s[0], s[1]),
			color.NRGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff},
		)
		row.TextSize = 12
		periodScoresBox.Add(row)
	}

	// Score evolution graph.
	scoreGraph := ms.buildScoreGraph(match)
	if scoreGraph != nil {
		periodScoresBox.Add(scoreGraph)
	}
	if maxP == 0 && scoreGraph == nil {
		periodScoresBox.Add(widget.NewLabel(i18n.T(i18n.KeyMatchSummaryNoData)))
	}

	content := container.NewVBox(
		titleText,
		scoreHeader,
		dateText,
		widget.NewSeparator(),
		periodScoresHeader,
		periodScoresBox,
		widget.NewSeparator(),
		scoringHeader,
		scoringBox,
		widget.NewSeparator(),
		playingTimeHeader,
		playingTimeBox,
		widget.NewSeparator(),
		foulsHeader,
		foulsBox,
	)

	scroll := container.NewVScroll(content)

	bg := canvas.NewRectangle(theme.ColorDarkBg)
	bottom := container.NewHBox(layout.NewSpacer(), closeBtn, layout.NewSpacer())

	ms.box = container.NewStack(bg, container.NewBorder(nil, container.NewPadded(bottom), nil, nil, container.NewPadded(scroll)))
	return ms
}

// buildScoreGraph creates a score evolution graph using canvas.Raster.
func (ms *MatchSummary) buildScoreGraph(match *model.Match) fyne.CanvasObject {
	// Build score progression: list of (timestamp, homeScore, awayScore).
	type scorePoint struct {
		t    int
		home int
		away int
	}
	var points []scorePoint
	homeRunning, awayRunning := 0, 0
	points = append(points, scorePoint{0, 0, 0})
	for _, e := range match.Events {
		if e.Type != model.EventScore {
			continue
		}
		if e.IsHome {
			homeRunning += e.Points
		} else {
			awayRunning += e.Points
		}
		points = append(points, scorePoint{e.Timestamp, homeRunning, awayRunning})
	}
	if len(points) < 2 {
		return nil
	}

	// Add final point at match end.
	lastT := points[len(points)-1].t
	if lastT == 0 {
		lastT = 1
	}

	maxScore := 1
	for _, p := range points {
		if p.home > maxScore {
			maxScore = p.home
		}
		if p.away > maxScore {
			maxScore = p.away
		}
	}

	homeColor := color.NRGBA{R: 0x44, G: 0x88, B: 0xff, A: 0xff}
	awayColor := color.NRGBA{R: 0xff, G: 0x66, B: 0x44, A: 0xff}
	gridColor := color.NRGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xff}

	raster := canvas.NewRasterWithPixels(func(x, y, w, h int) color.Color {
		if w <= 0 || h <= 0 {
			return color.Transparent
		}
		// Grid lines every 10 points.
		scoreAtY := maxScore - (y * maxScore / h)
		if scoreAtY > 0 && scoreAtY%10 == 0 && y > 0 {
			expectedY := h - (scoreAtY * h / maxScore)
			if y == expectedY {
				return gridColor
			}
		}
		return color.Transparent
	})
	raster.SetMinSize(fyne.NewSize(400, 150))

	// Draw lines on top using canvas.Line objects.
	graphContainer := container.NewStack(raster)

	// We can't easily draw polylines with canvas objects in a Stack.
	// Instead, use a custom raster that draws the full graph.
	graphRaster := canvas.NewRaster(func(w, h int) image.Image {
		if w <= 0 || h <= 0 {
			return image.NewRGBA(image.Rect(0, 0, 1, 1))
		}
		img := image.NewRGBA(image.Rect(0, 0, w, h))

		// Draw grid.
		for score := 10; score <= maxScore; score += 10 {
			gy := h - (score * h / maxScore)
			if gy >= 0 && gy < h {
				for gx := 0; gx < w; gx += 4 {
					if gx < w {
						img.Set(gx, gy, gridColor)
					}
				}
			}
		}

		// Draw lines.
		drawLine := func(x0, y0, x1, y1 int, c color.NRGBA) {
			dx := x1 - x0
			dy := y1 - y0
			steps := dx
			if dy < 0 {
				dy = -dy
			}
			if dy > steps {
				steps = dy
			}
			if steps == 0 {
				return
			}
			for s := 0; s <= steps; s++ {
				px := x0 + s*(x1-x0)/steps
				py := y0 + s*(y1-y0)/steps
				if px >= 0 && px < w && py >= 0 && py < h {
					img.Set(px, py, c)
					// Thicker line.
					if py+1 < h {
						img.Set(px, py+1, c)
					}
				}
			}
		}

		for i := 1; i < len(points); i++ {
			p0 := points[i-1]
			p1 := points[i]
			x0 := p0.t * w / lastT
			x1 := p1.t * w / lastT
			yH0 := h - (p0.home * h / maxScore)
			yH1 := h - (p1.home * h / maxScore)
			yA0 := h - (p0.away * h / maxScore)
			yA1 := h - (p1.away * h / maxScore)
			drawLine(x0, yH0, x1, yH1, homeColor)
			drawLine(x0, yA0, x1, yA1, awayColor)
		}

		return img
	})
	graphRaster.SetMinSize(fyne.NewSize(400, 150))

	// Legend.
	homeLegend := canvas.NewText(match.TeamName, homeColor)
	homeLegend.TextSize = 11
	awayLegend := canvas.NewText(match.Opponent, awayColor)
	awayLegend.TextSize = 11
	legend := container.NewHBox(homeLegend, layout.NewSpacer(), awayLegend)

	_ = graphContainer // unused, replaced by graphRaster
	return container.NewVBox(graphRaster, legend)
}

// Widget returns the summary root layout.
func (ms *MatchSummary) Widget() fyne.CanvasObject {
	return ms.box
}
