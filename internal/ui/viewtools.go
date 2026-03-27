package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/ui/editor"
	"github.com/darkweaver87/courtdraw/internal/ui/icon"
)

// ViewTools is a collapsible vertical panel with court view controls
// (apron toggle, rotate, zoom). Placed on the right side of the court.
type ViewTools struct {
	box       *fyne.Container
	btnsBox   *fyne.Container
	chevron   *TipButton
	selectBtn *TipButton
	eraserBtn *TipButton
	apronBtn  *TipButton
	collapsed bool

	ActiveTool    *editor.ToolType // pointer to EditorState.ActiveTool for highlight sync
	OnSelect      func()
	OnEraser      func()
	OnToggleApron func()
	OnRotate      func()
	OnZoomIn      func()
	OnZoomOut     func()
	OnZoomReset   func()
}

// NewViewTools creates a new view tools panel.
func NewViewTools() *ViewTools {
	vt := &ViewTools{}

	vt.selectBtn = NewTipButton(icon.ToolSelect, i18n.T(i18n.KeyToolSelect), func() {
		if vt.OnSelect != nil {
			vt.OnSelect()
		}
		vt.SyncToolHighlight()
	})
	vt.eraserBtn = NewTipButton(icon.Delete(), i18n.T(i18n.KeyToolEraser), func() {
		if vt.OnEraser != nil {
			vt.OnEraser()
		}
		vt.SyncToolHighlight()
	})
	vt.apronBtn = NewTipButton(theme.VisibilityIcon(), i18n.T(i18n.KeyTooltipApron), func() {
		if vt.OnToggleApron != nil {
			vt.OnToggleApron()
		}
	})
	rotateBtn := NewTipButton(theme.ViewRefreshIcon(), i18n.T(i18n.KeyTooltipRotate), func() {
		if vt.OnRotate != nil {
			vt.OnRotate()
		}
	})
	zoomInBtn := NewTipButton(theme.ZoomInIcon(), i18n.T(i18n.KeyTooltipZoomIn), func() {
		if vt.OnZoomIn != nil {
			vt.OnZoomIn()
		}
	})
	zoomResetBtn := NewTipButton(theme.ZoomFitIcon(), i18n.T(i18n.KeyTooltipZoomReset), func() {
		if vt.OnZoomReset != nil {
			vt.OnZoomReset()
		}
	})
	zoomOutBtn := NewTipButton(theme.ZoomOutIcon(), i18n.T(i18n.KeyTooltipZoomOut), func() {
		if vt.OnZoomOut != nil {
			vt.OnZoomOut()
		}
	})

	vt.btnsBox = container.NewVBox(vt.selectBtn, vt.eraserBtn, widget.NewSeparator(), vt.apronBtn, rotateBtn, widget.NewSeparator(), zoomInBtn, zoomResetBtn, zoomOutBtn)

	vt.chevron = NewTipButton(icon.ChevronRight, "", func() {
		if vt.collapsed {
			vt.collapsed = false
			vt.btnsBox.Show()
			vt.chevron.Icon = icon.ChevronRight
		} else {
			vt.collapsed = true
			vt.btnsBox.Hide()
			vt.chevron.Icon = icon.ChevronLeft
		}
		vt.chevron.Refresh()
		vt.box.Refresh()
	})

	// Collapsed by default on mobile.
	if isMobile {
		vt.collapsed = true
		vt.btnsBox.Hide()
		vt.chevron.Icon = icon.ChevronLeft
	}

	chevronSize := fyne.NewSize(24, 24)
	if isMobile {
		chevronSize = fyne.NewSize(40, 40)
	}

	bg := canvas.NewRectangle(color.NRGBA{R: 0x28, G: 0x28, B: 0x28, A: 0xff})
	inner := container.NewBorder(container.NewGridWrap(chevronSize, vt.chevron), nil, nil, nil, vt.btnsBox)
	vt.box = container.NewStack(bg, inner)
	return vt
}

// Widget returns the panel widget.
func (vt *ViewTools) Widget() fyne.CanvasObject {
	return vt.box
}

// SyncToolHighlight updates the select/eraser button highlights based on the active tool.
func (vt *ViewTools) SyncToolHighlight() {
	vt.selectBtn.OverrideColor = nil
	vt.eraserBtn.OverrideColor = nil
	if vt.ActiveTool != nil {
		switch *vt.ActiveTool {
		case editor.ToolSelect, editor.ToolNone:
			vt.selectBtn.OverrideColor = toolActiveColor
		case editor.ToolDelete:
			vt.eraserBtn.OverrideColor = toolActiveColor
		}
	}
	vt.selectBtn.Refresh()
	vt.eraserBtn.Refresh()
}

// SetApronVisible updates the apron button icon.
func (vt *ViewTools) SetApronVisible(visible bool) {
	if visible {
		vt.apronBtn.Icon = theme.VisibilityIcon()
	} else {
		vt.apronBtn.Icon = theme.VisibilityOffIcon()
	}
	vt.apronBtn.Refresh()
}
