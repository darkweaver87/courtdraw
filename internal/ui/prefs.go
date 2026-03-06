package ui

import (
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/store"
)

// showPrefsDialog displays the preferences dialog.
func showPrefsDialog(w fyne.Window, settings *store.Settings, ys *store.YAMLStore, onSaved func(langChanged bool)) {
	// GitHub Token.
	tokenEntry := widget.NewPasswordEntry()
	tokenEntry.SetPlaceHolder(i18n.T("prefs.token_placeholder"))
	token := settings.GithubToken
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	tokenEntry.SetText(token)

	// Language.
	langSelect := widget.NewSelect([]string{"en", "fr"}, nil)
	langSelect.SetSelected(settings.Language)
	if langSelect.Selected == "" {
		langSelect.SetSelected(string(i18n.CurrentLang()))
	}

	// Exercise directories.
	dirsEntry := widget.NewMultiLineEntry()
	dirsEntry.SetPlaceHolder(i18n.T("prefs.dirs_placeholder"))
	dirsEntry.SetText(strings.Join(settings.ExerciseDirs, "\n"))
	dirsEntry.SetMinRowsVisible(3)

	form := container.NewVBox(
		widget.NewLabel(i18n.T("prefs.github_token")),
		tokenEntry,
		widget.NewLabel(i18n.T("prefs.language")),
		langSelect,
		widget.NewLabel(i18n.T("prefs.exercise_dirs")),
		dirsEntry,
	)

	d := dialog.NewCustomConfirm(
		i18n.T("prefs.title"),
		i18n.T("prefs.save"),
		i18n.T("dialog.cancel"),
		form,
		func(ok bool) {
			if !ok {
				return
			}
			oldLang := settings.Language
			settings.GithubToken = strings.TrimSpace(tokenEntry.Text)
			settings.Language = langSelect.Selected

			// Parse directories.
			var dirs []string
			for _, line := range strings.Split(dirsEntry.Text, "\n") {
				d := strings.TrimSpace(line)
				if d != "" {
					dirs = append(dirs, d)
				}
			}
			settings.ExerciseDirs = dirs

			if ys != nil {
				ys.SaveSettings(settings)
			}

			langChanged := oldLang != settings.Language
			if onSaved != nil {
				onSaved(langChanged)
			}
		},
		w,
	)
	d.Resize(fyne.NewSize(450, 380))
	d.Show()
}
