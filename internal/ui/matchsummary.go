package ui

import (
	"fmt"
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

	content := container.NewVBox(
		titleText,
		scoreHeader,
		dateText,
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

// Widget returns the summary root layout.
func (ms *MatchSummary) Widget() fyne.CanvasObject {
	return ms.box
}
