package widget

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/ui/icon"
	"github.com/darkweaver87/courtdraw/internal/ui/theme"
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
)

// FileToolbar provides New, Open, Save, Duplicate, Import buttons.
type FileToolbar struct {
	newClick       widget.Clickable
	openClick      widget.Clickable
	saveClick      widget.Clickable
	duplicateClick widget.Clickable
	importClick    widget.Clickable
}

// Layout renders the file toolbar and returns the action triggered, if any.
func (ft *FileToolbar) Layout(gtx layout.Context, th *material.Theme, modified bool) (layout.Dimensions, FileAction) {
	action := FileActionNone

	if ft.newClick.Clicked(gtx) {
		action = FileActionNew
	}
	if ft.openClick.Clicked(gtx) {
		action = FileActionOpen
	}
	if ft.saveClick.Clicked(gtx) {
		action = FileActionSave
	}
	if ft.duplicateClick.Clicked(gtx) {
		action = FileActionDuplicate
	}
	if ft.importClick.Clicked(gtx) {
		action = FileActionImport
	}

	barHeight := gtx.Dp(unit.Dp(28))

	// Background.
	bg := color.NRGBA{R: 0x2e, G: 0x2e, B: 0x2e, A: 0xff}
	paint.FillShape(gtx.Ops, bg, clip.Rect{Max: image.Pt(gtx.Constraints.Max.X, barHeight)}.Op())

	saveColor := theme.ColorTabText
	if modified {
		saveColor = theme.ColorCoach
	}

	dims := layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return icon.IconBtnTooltip(gtx, th, &ft.newClick, icon.New, theme.ColorTabText, i18n.T("tooltip.new"))
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return icon.IconBtnTooltip(gtx, th, &ft.openClick, icon.Open, theme.ColorTabText, i18n.T("tooltip.open"))
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return icon.IconBtnTooltip(gtx, th, &ft.saveClick, icon.Save, saveColor, i18n.T("tooltip.save"))
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return icon.IconBtnTooltip(gtx, th, &ft.duplicateClick, icon.Duplicate, theme.ColorTabText, i18n.T("tooltip.duplicate"))
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return icon.IconBtnTooltip(gtx, th, &ft.importClick, icon.Import, theme.ColorTabText, i18n.T("tooltip.import"))
		}),
	)

	return dims, action
}
