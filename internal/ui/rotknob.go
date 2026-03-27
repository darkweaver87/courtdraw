package ui

import (
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// RotKnob is a circular rotation knob widget.
// Drag around the center to set the angle (0–360°).
type RotKnob struct {
	widget.BaseWidget

	Value     float64 // current angle in degrees (0–360)
	OnChanged func(float64)

	dragging bool
}

var _ fyne.Draggable = (*RotKnob)(nil)
var _ fyne.Tappable = (*RotKnob)(nil)
var _ desktop.Hoverable = (*RotKnob)(nil)

// NewRotKnob creates a new rotation knob.
func NewRotKnob(onChange func(float64)) *RotKnob {
	rk := &RotKnob{OnChanged: onChange}
	rk.ExtendBaseWidget(rk)
	return rk
}

func (rk *RotKnob) MinSize() fyne.Size {
	return fyne.NewSize(28, 28)
}

func (rk *RotKnob) CreateRenderer() fyne.WidgetRenderer {
	return newRotKnobRenderer(rk)
}

func (rk *RotKnob) angleFromPos(pos fyne.Position) float64 {
	sz := rk.Size()
	cx, cy := sz.Width/2, sz.Height/2
	dx := float64(pos.X - cx)
	dy := float64(pos.Y - cy)
	// atan2 gives angle from +X axis, CCW. We want 0°=up, CW.
	angle := math.Atan2(dx, -dy) * 180 / math.Pi
	if angle < 0 {
		angle += 360
	}
	// Snap to 5° increments.
	angle = math.Round(angle/5) * 5
	if angle >= 360 {
		angle = 0
	}
	return angle
}

func (rk *RotKnob) Tapped(e *fyne.PointEvent) {
	rk.Value = rk.angleFromPos(e.Position)
	if rk.OnChanged != nil {
		rk.OnChanged(rk.Value)
	}
	rk.Refresh()
}

func (rk *RotKnob) Dragged(e *fyne.DragEvent) {
	rk.dragging = true
	rk.Value = rk.angleFromPos(e.Position)
	if rk.OnChanged != nil {
		rk.OnChanged(rk.Value)
	}
	rk.Refresh()
}

func (rk *RotKnob) DragEnd() {
	rk.dragging = false
}

func (rk *RotKnob) MouseIn(_ *desktop.MouseEvent)    {}
func (rk *RotKnob) MouseMoved(_ *desktop.MouseEvent)  {}
func (rk *RotKnob) MouseOut()                         {}

// --- Renderer ---

type rotKnobRenderer struct {
	knob    *RotKnob
	bg      *canvas.Circle
	ring    *canvas.Circle
	needle  *canvas.Line
}

func (r *rotKnobRenderer) Layout(sz fyne.Size) {
	r.bg.Resize(sz)
	r.ring.Resize(sz)
	r.bg.Move(fyne.NewPos(0, 0))
	r.ring.Move(fyne.NewPos(0, 0))
	r.updateNeedle(sz)
}

func (r *rotKnobRenderer) MinSize() fyne.Size {
	return fyne.NewSize(28, 28)
}

func (r *rotKnobRenderer) Refresh() {
	r.updateNeedle(r.knob.Size())
	r.needle.Refresh()
}

func (r *rotKnobRenderer) updateNeedle(sz fyne.Size) {
	cx := sz.Width / 2
	cy := sz.Height / 2
	radius := float64(cx) * 0.8
	angle := r.knob.Value * math.Pi / 180
	// 0° = up, clockwise.
	nx := cx + float32(radius*math.Sin(angle))
	ny := cy - float32(radius*math.Cos(angle))
	r.needle.Position1 = fyne.NewPos(cx, cy)
	r.needle.Position2 = fyne.NewPos(nx, ny)
}

func (r *rotKnobRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bg, r.ring, r.needle}
}

func (r *rotKnobRenderer) Destroy() {}

func newRotKnobRenderer(knob *RotKnob) *rotKnobRenderer {
	bg := canvas.NewCircle(color.NRGBA{R: 0x3a, G: 0x3a, B: 0x3a, A: 0xff})
	ring := canvas.NewCircle(color.Transparent)
	ring.StrokeColor = color.NRGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff}
	ring.StrokeWidth = 1.5
	needle := canvas.NewLine(color.NRGBA{R: 0x29, G: 0x6d, B: 0xd4, A: 0xff})
	needle.StrokeWidth = 2
	return &rotKnobRenderer{knob: knob, bg: bg, ring: ring, needle: needle}
}

