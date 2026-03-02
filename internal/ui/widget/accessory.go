package widget

import (
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/darkweaver87/courtdraw/internal/court"
	"github.com/darkweaver87/courtdraw/internal/model"
)

// Accessory visual constants.
const (
	coneSize     = 10
	ladderWidth  = 12
	ladderLength = 40
	ladderRungs  = 5
	chairSize    = 12
)

var (
	colorCone   = color.NRGBA{R: 0xff, G: 0xa5, B: 0x00, A: 0xff} // orange
	colorLadder = color.NRGBA{R: 0xff, G: 0xd7, B: 0x00, A: 0xff} // gold
	colorChair  = color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff} // grey
)

// DrawAccessoryWithOpacity draws a court accessory with the given opacity.
func DrawAccessoryWithOpacity(ops *op.Ops, vp *court.Viewport, acc *model.Accessory, opacity float64) {
	if opacity <= 0 {
		return
	}
	// For simplicity, accessories are drawn fully at any opacity > 0.
	// True alpha blending would require changes to all draw* helpers.
	// We use the same drawing for now; fade is primarily visual on players.
	DrawAccessory(ops, vp, acc, false)
}

// DrawAccessory draws a court accessory as a simple geometric shape.
// If selected is true, draws a highlight outline around it.
func DrawAccessory(ops *op.Ops, vp *court.Viewport, acc *model.Accessory, selected bool) {
	center := vp.RelToPixel(acc.Position)

	if selected {
		court.DrawCircleOutline(ops, center, coneSize+6, 2, highlightColor)
	}

	switch acc.Type {
	case model.AccessoryCone:
		drawCone(ops, center)
	case model.AccessoryAgilityLadder:
		drawLadder(ops, center, acc.Rotation)
	case model.AccessoryChair:
		drawChair(ops, center, acc.Rotation)
	}
}

// drawCone draws a triangle (cone marker).
func drawCone(ops *op.Ops, center f32.Point) {
	s := float32(coneSize)
	top := f32.Point{X: center.X, Y: center.Y - s}
	left := f32.Point{X: center.X - s*0.7, Y: center.Y + s*0.5}
	right := f32.Point{X: center.X + s*0.7, Y: center.Y + s*0.5}

	var path clip.Path
	path.Begin(ops)
	path.MoveTo(top)
	path.LineTo(left)
	path.LineTo(right)
	path.Close()

	outline := clip.Outline{Path: path.End()}.Op()
	paint.FillShape(ops, colorCone, outline)
}

// drawLadder draws a rectangle with rungs (agility ladder).
func drawLadder(ops *op.Ops, center f32.Point, rotation float64) {
	w := float32(ladderWidth)
	h := float32(ladderLength)

	// rotation in radians
	rad := rotation * math.Pi / 180

	// rotated rectangle
	corners := [4]f32.Point{
		{X: -w / 2, Y: -h / 2},
		{X: w / 2, Y: -h / 2},
		{X: w / 2, Y: h / 2},
		{X: -w / 2, Y: h / 2},
	}

	cos := float32(math.Cos(rad))
	sin := float32(math.Sin(rad))

	for i, c := range corners {
		corners[i] = f32.Point{
			X: center.X + c.X*cos - c.Y*sin,
			Y: center.Y + c.X*sin + c.Y*cos,
		}
	}

	// outline
	court.DrawLine(ops, corners[0], corners[1], 1.5, colorLadder)
	court.DrawLine(ops, corners[1], corners[2], 1.5, colorLadder)
	court.DrawLine(ops, corners[2], corners[3], 1.5, colorLadder)
	court.DrawLine(ops, corners[3], corners[0], 1.5, colorLadder)

	// rungs
	for i := 1; i < ladderRungs; i++ {
		t := float32(i) / float32(ladderRungs)
		left := f32.Point{
			X: corners[0].X + (corners[3].X-corners[0].X)*t,
			Y: corners[0].Y + (corners[3].Y-corners[0].Y)*t,
		}
		right := f32.Point{
			X: corners[1].X + (corners[2].X-corners[1].X)*t,
			Y: corners[1].Y + (corners[2].Y-corners[1].Y)*t,
		}
		court.DrawLine(ops, left, right, 1.5, colorLadder)
	}
}

// drawChair draws an L-shape (folding chair used as screen).
func drawChair(ops *op.Ops, center f32.Point, rotation float64) {
	s := float32(chairSize)
	rad := rotation * math.Pi / 180
	cos := float32(math.Cos(rad))
	sin := float32(math.Sin(rad))

	rotate := func(x, y float32) f32.Point {
		return f32.Point{
			X: center.X + x*cos - y*sin,
			Y: center.Y + x*sin + y*cos,
		}
	}

	// L-shape: vertical bar + horizontal seat
	// vertical (back of chair)
	court.DrawLine(ops, rotate(0, -s), rotate(0, s*0.3), 2.5, colorChair)
	// horizontal (seat)
	court.DrawLine(ops, rotate(0, s*0.3), rotate(s*0.7, s*0.3), 2.5, colorChair)
}
