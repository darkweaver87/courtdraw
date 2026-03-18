package ui

import (
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/i18n"
)

// showAboutDialog displays a dialog with the application version.
func showAboutDialog(w fyne.Window, version string) {
	versionLabel := widget.NewLabel(i18n.T(i18n.KeyPrefsVersion) + " : " + version)
	versionLabel.Wrapping = fyne.TextWrapOff

	content := container.NewVBox(
		widget.NewLabel("CourtDraw"),
		versionLabel,
	)

	d := dialog.NewCustom(
		i18n.T(i18n.KeyAboutTitle),
		i18n.T(i18n.KeyDialogCancel),
		content,
		w,
	)
	d.Resize(fyne.NewSize(300, 150))
	d.Show()
}

// showUpdateDialog displays a dialog notifying the user of a new version,
// with a hyperlink to the GitHub release page.
func showUpdateDialog(w fyne.Window, tag, releaseURL string) {
	msg := widget.NewLabel(i18n.Tf(i18n.KeyVersionNewAvailable, tag))
	msg.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(msg)

	if u, err := url.Parse(releaseURL); err == nil && releaseURL != "" {
		link := widget.NewHyperlink(i18n.T(i18n.KeyVersionReleaseNotes), u)
		content.Add(link)
	}

	d := dialog.NewCustom(
		i18n.T(i18n.KeyVersionUpdateTitle),
		"OK",
		content,
		w,
	)
	d.Resize(fyne.NewSize(350, 180))
	d.Show()
}
