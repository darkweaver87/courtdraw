package ui

import (
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/store"
)

// showPrefsDialog displays the preferences dialog.
func showPrefsDialog(w fyne.Window, settings *store.Settings, ys *store.YAMLStore, onSaved func(langChanged bool)) {
	// GitHub Token.
	tokenEntry := widget.NewPasswordEntry()
	tokenEntry.SetPlaceHolder(i18n.T(i18n.KeyPrefsTokenPlaceholder))
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

	// Exercise directory — default to store's exercises dir.
	defaultDir := ""
	if ys != nil {
		defaultDir = ys.ExercisesDir()
	}
	dirValue := settings.ExerciseDir
	if dirValue == "" {
		dirValue = defaultDir
	}

	dirEntry := widget.NewEntry()
	dirEntry.SetText(dirValue)
	dirEntry.SetPlaceHolder(defaultDir)

	browseBtn := widget.NewButton(i18n.T(i18n.KeyPrefsBrowse), func() {
		fd := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			dirEntry.SetText(uri.Path())
		}, w)
		// Try to start from the current value.
		if current := strings.TrimSpace(dirEntry.Text); current != "" {
			if listable, err := storage.ListerForURI(storage.NewFileURI(current)); err == nil {
				fd.SetLocation(listable)
			}
		}
		fd.Show()
	})
	browseBtn.Importance = widget.LowImportance

	dirRow := container.NewBorder(nil, nil, nil, browseBtn, dirEntry)

	// PDF export directory — default to home.
	pdfDirValue := settings.PdfExportDir
	if pdfDirValue == "" {
		pdfDirValue, _ = os.UserHomeDir()
	}

	pdfDirEntry := widget.NewEntry()
	pdfDirEntry.SetText(pdfDirValue)

	pdfBrowseBtn := widget.NewButton(i18n.T(i18n.KeyPrefsBrowse), func() {
		fd := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			pdfDirEntry.SetText(uri.Path())
		}, w)
		if current := strings.TrimSpace(pdfDirEntry.Text); current != "" {
			if listable, err := storage.ListerForURI(storage.NewFileURI(current)); err == nil {
				fd.SetLocation(listable)
			}
		}
		fd.Show()
	})
	pdfBrowseBtn.Importance = widget.LowImportance

	pdfDirRow := container.NewBorder(nil, nil, nil, pdfBrowseBtn, pdfDirEntry)

	form := container.NewVBox(
		widget.NewLabel(i18n.T(i18n.KeyPrefsGithubToken)),
		tokenEntry,
		widget.NewLabel(i18n.T(i18n.KeyPrefsLanguage)),
		langSelect,
		widget.NewLabel(i18n.T(i18n.KeyPrefsExerciseDir)),
		dirRow,
		widget.NewLabel(i18n.T(i18n.KeyPrefsPdfExportDir)),
		pdfDirRow,
		layout.NewSpacer(),
	)

	d := dialog.NewCustomConfirm(
		i18n.T(i18n.KeyPrefsTitle),
		i18n.T(i18n.KeyPrefsSave),
		i18n.T(i18n.KeyDialogCancel),
		form,
		func(ok bool) {
			if !ok {
				return
			}
			oldLang := settings.Language
			settings.GithubToken = strings.TrimSpace(tokenEntry.Text)
			settings.Language = langSelect.Selected
			settings.ExerciseDir = strings.TrimSpace(dirEntry.Text)
			settings.PdfExportDir = strings.TrimSpace(pdfDirEntry.Text)

			if ys != nil {
				_ = ys.SaveSettings(settings)
			}

			langChanged := oldLang != settings.Language
			if onSaved != nil {
				onSaved(langChanged)
			}
		},
		w,
	)
	d.Resize(fyne.NewSize(450, 420))
	d.Show()
}
