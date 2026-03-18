package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/ui/icon"
)

// FileAction represents a file operation triggered by the toolbar.
type FileAction int

const (
	FileActionNone FileAction = iota
	FileActionNew
	FileActionOpen
	FileActionSave
	FileActionSaveAs
	FileActionImport
	FileActionRecent
	FileActionPreferences
	FileActionAbout
)

// FileToolbar provides New, Open, Recent, Save, Save As, Import, About, Preferences buttons.
type FileToolbar struct {
	btns    [8]*TipButton // new, open, recent, save, save_as, import, about, preferences
	saveBtn *TipButton
	box     *fyne.Container
	OnAction func(FileAction)
}

// NewFileToolbar creates the file toolbar.
func NewFileToolbar() *FileToolbar {
	ft := &FileToolbar{}

	ft.btns[0] = NewTipButton(icon.New(), i18n.T(i18n.KeyTooltipNew), func() {
		if ft.OnAction != nil {
			ft.OnAction(FileActionNew)
		}
	})
	ft.btns[1] = NewTipButton(icon.Open(), i18n.T(i18n.KeyTooltipOpen), func() {
		if ft.OnAction != nil {
			ft.OnAction(FileActionOpen)
		}
	})
	ft.btns[2] = NewTipButton(icon.Refresh(), i18n.T(i18n.KeyTooltipRecent), func() {
		if ft.OnAction != nil {
			ft.OnAction(FileActionRecent)
		}
	})
	ft.btns[3] = NewTipButton(icon.Save(), i18n.T(i18n.KeyTooltipSave), func() {
		if ft.OnAction != nil {
			ft.OnAction(FileActionSave)
		}
	})
	ft.saveBtn = ft.btns[3]
	ft.btns[4] = NewTipButton(icon.Duplicate(), i18n.T(i18n.KeyTooltipSaveAs), func() {
		if ft.OnAction != nil {
			ft.OnAction(FileActionSaveAs)
		}
	})
	ft.btns[5] = NewTipButton(icon.Import(), i18n.T(i18n.KeyTooltipImport), func() {
		if ft.OnAction != nil {
			ft.OnAction(FileActionImport)
		}
	})
	ft.btns[6] = NewTipButton(icon.Info(), i18n.T(i18n.KeyTooltipAbout), func() {
		if ft.OnAction != nil {
			ft.OnAction(FileActionAbout)
		}
	})
	ft.btns[7] = NewTipButton(icon.Settings(), i18n.T(i18n.KeyTooltipPreferences), func() {
		if ft.OnAction != nil {
			ft.OnAction(FileActionPreferences)
		}
	})

	bg := canvas.NewRectangle(color.NRGBA{R: 0x2e, G: 0x2e, B: 0x2e, A: 0xff})
	buttons := container.NewHBox(ft.btns[0], ft.btns[1], ft.btns[2], ft.btns[3], ft.btns[4], ft.btns[5], layout.NewSpacer(), ft.btns[6], ft.btns[7])
	ft.box = container.NewStack(bg, buttons)
	return ft
}

// RefreshLanguage updates tooltip text for the current language.
func (ft *FileToolbar) RefreshLanguage() {
	keys := [8]string{i18n.KeyTooltipNew, i18n.KeyTooltipOpen, i18n.KeyTooltipRecent, i18n.KeyTooltipSave, i18n.KeyTooltipSaveAs, i18n.KeyTooltipImport, i18n.KeyTooltipAbout, i18n.KeyTooltipPreferences}
	for i, key := range keys {
		ft.btns[i].SetTooltip(i18n.T(key))
	}
}

// Widget returns the toolbar widget.
func (ft *FileToolbar) Widget() fyne.CanvasObject {
	return ft.box
}

// Btn returns an individual toolbar button by action (for use in custom layouts).
func (ft *FileToolbar) Btn(action FileAction) *TipButton {
	switch action {
	case FileActionNew:
		return ft.btns[0]
	case FileActionOpen:
		return ft.btns[1]
	case FileActionRecent:
		return ft.btns[2]
	case FileActionSave:
		return ft.btns[3]
	case FileActionSaveAs:
		return ft.btns[4]
	case FileActionImport:
		return ft.btns[5]
	case FileActionAbout:
		return ft.btns[6]
	case FileActionPreferences:
		return ft.btns[7]
	}
	return nil
}

// SetModified updates save button appearance when exercise has unsaved changes.
func (ft *FileToolbar) SetModified(modified bool) {
	if modified {
		ft.saveBtn.SetText("*")
	} else {
		ft.saveBtn.SetText("")
	}
}
