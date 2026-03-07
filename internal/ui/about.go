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
	versionLabel := widget.NewLabel(i18n.T("prefs.version") + " : " + version)
	versionLabel.Wrapping = fyne.TextWrapOff

	content := container.NewVBox(
		widget.NewLabel("CourtDraw"),
		versionLabel,
	)

	d := dialog.NewCustom(
		i18n.T("about.title"),
		i18n.T("dialog.cancel"),
		content,
		w,
	)
	d.Resize(fyne.NewSize(300, 150))
	d.Show()
}

// showUpdateDialog displays a dialog notifying the user of a new version,
// with a hyperlink to the GitHub release page.
func showUpdateDialog(w fyne.Window, tag, releaseURL string) {
	msg := widget.NewLabel(i18n.Tf("version.new_available", tag))
	msg.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(msg)

	if u, err := url.Parse(releaseURL); err == nil && releaseURL != "" {
		link := widget.NewHyperlink(i18n.T("version.release_notes"), u)
		content.Add(link)
	}

	d := dialog.NewCustom(
		i18n.T("version.update_title"),
		"OK",
		content,
		w,
	)
	d.Resize(fyne.NewSize(350, 180))
	d.Show()
}
