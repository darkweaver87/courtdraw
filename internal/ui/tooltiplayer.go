package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// TooltipLayer is a transparent layer placed on top of the UI.
// It renders tooltip text near the hovered button without intercepting events
// (canvas primitives don't implement Tappable/Draggable).
type TooltipLayer struct {
	box   *fyne.Container // container.NewWithoutLayout
	bg    *canvas.Rectangle
	label *canvas.Text
}

// NewTooltipLayer creates the tooltip layer.
func NewTooltipLayer() *TooltipLayer {
	tl := &TooltipLayer{}
	tl.bg = canvas.NewRectangle(color.NRGBA{R: 0x22, G: 0x22, B: 0x22, A: 0xee})
	tl.bg.CornerRadius = 3
	tl.label = canvas.NewText("", color.NRGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff})
	tl.label.TextSize = 11
	tl.bg.Hide()
	tl.label.Hide()
	tl.box = container.NewWithoutLayout(tl.bg, tl.label)
	return tl
}

// Widget returns the layer container to include in the UI tree.
func (tl *TooltipLayer) Widget() fyne.CanvasObject {
	return tl.box
}

// Show displays the tooltip at the given absolute position.
// The X coordinate is clamped so the tooltip stays within the window.
func (tl *TooltipLayer) Show(text string, pos fyne.Position) {
	tl.label.Text = text
	tl.label.Refresh()
	ts := tl.label.MinSize()
	pad := float32(4)
	bgW := ts.Width + pad*2

	// Clamp X so tooltip doesn't overflow the right edge.
	x := pos.X
	if canvas := fyne.CurrentApp().Driver().CanvasForObject(tl.box); canvas != nil {
		winW := canvas.Size().Width
		if x+bgW > winW {
			x = winW - bgW
		}
	}
	if x < 0 {
		x = 0
	}

	tl.bg.Resize(fyne.NewSize(bgW, ts.Height+pad))
	tl.bg.Move(fyne.NewPos(x, pos.Y))
	tl.label.Move(fyne.NewPos(x+pad, pos.Y+pad/2))
	tl.bg.Show()
	tl.label.Show()
}

// Hide hides the tooltip.
func (tl *TooltipLayer) Hide() {
	tl.bg.Hide()
	tl.label.Hide()
}
