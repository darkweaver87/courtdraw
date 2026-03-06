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
	FileActionDuplicate
	FileActionImport
	FileActionRecent
	FileActionPreferences
)

// FileToolbar provides New, Open, Recent, Save, Duplicate, Import, Preferences buttons.
type FileToolbar struct {
	btns    [7]*TipButton // new, open, recent, save, duplicate, import, preferences
	saveBtn *TipButton
	box     *fyne.Container
	OnAction func(FileAction)
}

// NewFileToolbar creates the file toolbar.
func NewFileToolbar() *FileToolbar {
	ft := &FileToolbar{}

	ft.btns[0] = NewTipButton(icon.New(), i18n.T("tooltip.new"), func() {
		if ft.OnAction != nil {
			ft.OnAction(FileActionNew)
		}
	})
	ft.btns[1] = NewTipButton(icon.Open(), i18n.T("tooltip.open"), func() {
		if ft.OnAction != nil {
			ft.OnAction(FileActionOpen)
		}
	})
	ft.btns[2] = NewTipButton(icon.Refresh(), i18n.T("tooltip.recent"), func() {
		if ft.OnAction != nil {
			ft.OnAction(FileActionRecent)
		}
	})
	ft.btns[3] = NewTipButton(icon.Save(), i18n.T("tooltip.save"), func() {
		if ft.OnAction != nil {
			ft.OnAction(FileActionSave)
		}
	})
	ft.saveBtn = ft.btns[3]
	ft.btns[4] = NewTipButton(icon.Duplicate(), i18n.T("tooltip.duplicate"), func() {
		if ft.OnAction != nil {
			ft.OnAction(FileActionDuplicate)
		}
	})
	ft.btns[5] = NewTipButton(icon.Import(), i18n.T("tooltip.import"), func() {
		if ft.OnAction != nil {
			ft.OnAction(FileActionImport)
		}
	})
	ft.btns[6] = NewTipButton(icon.Settings(), i18n.T("tooltip.preferences"), func() {
		if ft.OnAction != nil {
			ft.OnAction(FileActionPreferences)
		}
	})

	bg := canvas.NewRectangle(color.NRGBA{R: 0x2e, G: 0x2e, B: 0x2e, A: 0xff})
	buttons := container.NewHBox(ft.btns[0], ft.btns[1], ft.btns[2], ft.btns[3], ft.btns[4], ft.btns[5], layout.NewSpacer(), ft.btns[6])
	ft.box = container.NewStack(bg, buttons)
	return ft
}

// RefreshLanguage updates tooltip text for the current language.
func (ft *FileToolbar) RefreshLanguage() {
	keys := [7]string{"tooltip.new", "tooltip.open", "tooltip.recent", "tooltip.save", "tooltip.duplicate", "tooltip.import", "tooltip.preferences"}
	for i, key := range keys {
		ft.btns[i].SetTooltip(i18n.T(key))
	}
}

// Widget returns the toolbar widget.
func (ft *FileToolbar) Widget() fyne.CanvasObject {
	return ft.box
}

// SetModified updates save button appearance when exercise has unsaved changes.
func (ft *FileToolbar) SetModified(modified bool) {
	if modified {
		ft.saveBtn.SetText("*")
	} else {
		ft.saveBtn.SetText("")
	}
}
