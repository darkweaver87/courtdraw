package ui

import (
	"image/color"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// TipButton is a custom icon button that handles hover events directly
// (no inner widget.Button that would steal events).
// Tooltip text is shown in the shared status bar on hover (no overlay).
type TipButton struct {
	widget.BaseWidget
	Icon          fyne.Resource
	tooltip       string
	onTapped      func()
	importance    widget.ButtonImportance
	text          string
	hovered       bool
	OverrideColor color.Color  // if non-nil, used instead of importance-based color
	TooltipAbove  bool         // show tooltip above the button instead of below
	MaxSize       *fyne.Size   // if set, caps MinSize to this value (for compact layouts)
}

var (
	tipIconSize float32 = 20
	tipPadding  float32 = 4
)

func init() {
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		tipIconSize = 36
		tipPadding = 8
	}
}

var (
	tipHoverBg   = color.NRGBA{R: 0x55, G: 0x55, B: 0x55, A: 0x80}
	tipPrimaryBg = color.NRGBA{R: 0x29, G: 0x6d, B: 0xd4, A: 0xff}
	tipDangerBg  = color.NRGBA{R: 0xd4, G: 0x29, B: 0x29, A: 0xff}
	tipMediumBg  = color.NRGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xff}
)

// tipLayer is the shared tooltip layer for showing tooltip text.
// Set by App at init time.
var tipLayer *TooltipLayer

// SetTipLayer sets the shared tooltip layer used for tooltip display.
func SetTipLayer(tl *TooltipLayer) {
	tipLayer = tl
}

var _ fyne.Tappable = (*TipButton)(nil)
var _ desktop.Hoverable = (*TipButton)(nil)

// NewTipButton creates an icon button with a hover tooltip.
func NewTipButton(res fyne.Resource, tooltip string, onTap func()) *TipButton {
	tb := &TipButton{
		Icon:       res,
		tooltip:    tooltip,
		onTapped:   onTap,
		importance: widget.LowImportance,
	}
	tb.ExtendBaseWidget(tb)
	return tb
}

// SetImportance sets the button visual style.
func (tb *TipButton) SetImportance(imp widget.ButtonImportance) {
	tb.importance = imp
	tb.Refresh()
}

// SetTooltip updates the tooltip text.
func (tb *TipButton) SetTooltip(s string) {
	tb.tooltip = s
}

// SetText sets optional text displayed next to the icon.
func (tb *TipButton) SetText(s string) {
	tb.text = s
	tb.Refresh()
}

// InnerButton returns self for API compatibility.
func (tb *TipButton) InnerButton() *TipButton {
	return tb
}

// Tapped handles tap/click.
func (tb *TipButton) Tapped(*fyne.PointEvent) {
	if tb.onTapped != nil {
		tb.onTapped()
	}
}

// MouseIn highlights the button and shows tooltip below it.
func (tb *TipButton) MouseIn(*desktop.MouseEvent) {
	tb.hovered = true
	tb.Refresh()
	if tb.tooltip != "" && tipLayer != nil {
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(tb)
		if tb.TooltipAbove {
			tipLayer.Show(tb.tooltip, fyne.NewPos(pos.X, pos.Y-24))
		} else {
			tipLayer.Show(tb.tooltip, fyne.NewPos(pos.X, pos.Y+tb.Size().Height+2))
		}
	}
}

// MouseMoved is required by desktop.Hoverable.
func (tb *TipButton) MouseMoved(*desktop.MouseEvent) {}

// MouseOut removes the highlight and hides the tooltip.
func (tb *TipButton) MouseOut() {
	tb.hovered = false
	tb.Refresh()
	if tipLayer != nil {
		tipLayer.Hide()
	}
}

// CreateRenderer returns the widget renderer.
func (tb *TipButton) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.Transparent)
	bg.CornerRadius = 4
	ico := canvas.NewImageFromResource(tb.Icon)
	ico.FillMode = canvas.ImageFillContain
	txt := canvas.NewText("", color.White)
	txt.TextSize = 12
	return &tipBtnRenderer{tb: tb, bg: bg, ico: ico, txt: txt}
}

type tipBtnRenderer struct {
	tb  *TipButton
	bg  *canvas.Rectangle
	ico *canvas.Image
	txt *canvas.Text
}

func (r *tipBtnRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.bg.Move(fyne.NewPos(0, 0))
	p := tipPadding
	if r.tb.text == "" {
		// Icon fills the cell minus padding.
		is := min(size.Width, size.Height) - p*2
		if is < 1 {
			is = 1
		}
		r.ico.Resize(fyne.NewSize(is, is))
		r.ico.Move(fyne.NewPos((size.Width-is)/2, (size.Height-is)/2))
		r.txt.Move(fyne.NewPos(-100, -100)) // offscreen
	} else {
		is := tipIconSize
		r.ico.Resize(fyne.NewSize(is, is))
		r.ico.Move(fyne.NewPos(p, (size.Height-is)/2))
		r.txt.Move(fyne.NewPos(p+is+p/2, (size.Height-r.txt.MinSize().Height)/2))
	}
}

func (r *tipBtnRenderer) MinSize() fyne.Size {
	is := tipIconSize
	p := tipPadding
	w := is + p*2
	h := is + p*2
	if r.tb.text != "" {
		w += r.txt.MinSize().Width + p/2
	}
	if r.tb.MaxSize != nil {
		if w > r.tb.MaxSize.Width {
			w = r.tb.MaxSize.Width
		}
		if h > r.tb.MaxSize.Height {
			h = r.tb.MaxSize.Height
		}
	}
	return fyne.NewSize(w, h)
}

func (r *tipBtnRenderer) Refresh() {
	r.txt.Text = r.tb.text
	r.txt.Refresh()
	if r.tb.Icon != nil {
		r.ico.Resource = r.tb.Icon
		r.ico.Refresh()
	}
	if r.tb.OverrideColor != nil {
		r.bg.FillColor = r.tb.OverrideColor
		r.bg.Refresh()
		return
	}
	switch r.tb.importance {
	case widget.HighImportance:
		r.bg.FillColor = tipPrimaryBg
	case widget.DangerImportance:
		r.bg.FillColor = tipDangerBg
	case widget.MediumImportance:
		if r.tb.hovered {
			r.bg.FillColor = color.NRGBA{R: 0x55, G: 0x55, B: 0x55, A: 0xff}
		} else {
			r.bg.FillColor = tipMediumBg
		}
	default: // LowImportance
		if r.tb.hovered {
			r.bg.FillColor = tipHoverBg
		} else {
			r.bg.FillColor = color.Transparent
		}
	}
	r.bg.Refresh()
}

func (r *tipBtnRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bg, r.ico, r.txt}
}

func (r *tipBtnRenderer) Destroy() {}
