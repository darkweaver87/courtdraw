package court

import (
	"image"
	"image/color"
	"math"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// Accessory visual constants (base sizes at 1x ElementScale).
// Sized so that ElementScaleForCourt maps them to real-world dimensions.
// Players and small accessories use ×2 real size for visibility.
// The agility ladder uses ×0.75 (real: 0.50m × 4.0m).
const (
	AccessoryConeSize     = 21  // ×2 real cone base ~0.20m
	AccessoryLadderWidth  = 20  // ×0.75 real ladder width 0.50m
	AccessoryLadderLength = 160 // ×0.75 real ladder length 4.0m
	AccessoryLadderRungs  = 7
	AccessoryChairSize    = 36  // ×1.5 real chair seat ~0.45m
)

var (
	ColorCone   = color.NRGBA{R: 0xff, G: 0xa5, B: 0x00, A: 0xff}
	ColorLadder = color.NRGBA{R: 0xff, G: 0xd7, B: 0x00, A: 0xff}
	ColorChair  = color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}
)

// DrawAccessory draws a court accessory as a geometric shape.
func DrawAccessory(img *image.RGBA, vp *Viewport, acc *model.Accessory, selected bool) {
	center := vp.RelToPixel(acc.Position)

	if selected {
		DrawCircleOutline(img, center, vp.S(AccessoryConeSize+6), vp.S(2), HighlightColor)
	}

	switch acc.Type {
	case model.AccessoryCone:
		drawCone(img, vp, center, acc.Rotation)
	case model.AccessoryAgilityLadder:
		drawLadder(img, vp, center, acc.Rotation)
	case model.AccessoryChair:
		drawChair(img, vp, center, acc.Rotation)
	}
}

// DrawAccessoryWithOpacity draws an accessory (opacity currently ignored for simplicity).
func DrawAccessoryWithOpacity(img *image.RGBA, vp *Viewport, acc *model.Accessory, opacity float64) {
	if opacity <= 0 {
		return
	}
	DrawAccessory(img, vp, acc, false)
}

func drawCone(img *image.RGBA, vp *Viewport, center Point, rotation float64) {
	s := vp.Sf(AccessoryConeSize)
	rad := rotation * math.Pi / 180
	cos := float32(math.Cos(rad))
	sin := float32(math.Sin(rad))

	rotate := func(lx, ly float32) Point {
		return Point{
			X: center.X + lx*cos - ly*sin,
			Y: center.Y + lx*sin + ly*cos,
		}
	}

	top := rotate(0, -s)
	left := rotate(-s*0.7, s*0.5)
	right := rotate(s*0.7, s*0.5)

	DrawTriangleFill(img, top, left, right, ColorCone)
}

func drawLadder(img *image.RGBA, vp *Viewport, center Point, rotation float64) {
	w := vp.Sf(AccessoryLadderWidth)
	h := vp.Sf(AccessoryLadderLength)
	lw := vp.S(1.5)

	rad := rotation * math.Pi / 180
	cos := float32(math.Cos(rad))
	sin := float32(math.Sin(rad))

	corners := [4]Point{
		{X: -w / 2, Y: -h / 2},
		{X: w / 2, Y: -h / 2},
		{X: w / 2, Y: h / 2},
		{X: -w / 2, Y: h / 2},
	}

	for i, c := range corners {
		corners[i] = Point{
			X: center.X + c.X*cos - c.Y*sin,
			Y: center.Y + c.X*sin + c.Y*cos,
		}
	}

	// Outline.
	DrawLine(img, corners[0], corners[1], lw, ColorLadder)
	DrawLine(img, corners[1], corners[2], lw, ColorLadder)
	DrawLine(img, corners[2], corners[3], lw, ColorLadder)
	DrawLine(img, corners[3], corners[0], lw, ColorLadder)

	// Rungs.
	for i := 1; i < AccessoryLadderRungs; i++ {
		t := float32(i) / float32(AccessoryLadderRungs)
		left := Point{
			X: corners[0].X + (corners[3].X-corners[0].X)*t,
			Y: corners[0].Y + (corners[3].Y-corners[0].Y)*t,
		}
		right := Point{
			X: corners[1].X + (corners[2].X-corners[1].X)*t,
			Y: corners[1].Y + (corners[2].Y-corners[1].Y)*t,
		}
		DrawLine(img, left, right, lw, ColorLadder)
	}
}

func drawChair(img *image.RGBA, vp *Viewport, center Point, rotation float64) {
	s := vp.Sf(AccessoryChairSize)
	lw := vp.S(2)
	rad := rotation * math.Pi / 180
	cos := float32(math.Cos(rad))
	sin := float32(math.Sin(rad))

	rotate := func(x, y float32) Point {
		return Point{
			X: center.X + x*cos - y*sin,
			Y: center.Y + x*sin + y*cos,
		}
	}

	half := s * 0.5

	// Seat: filled rectangle.
	seatColor := color.NRGBA{R: 0x90, G: 0x90, B: 0x90, A: 0xaa}
	tl := rotate(-half, -half)
	tr := rotate(half, -half)
	br := rotate(half, half)
	bl := rotate(-half, half)
	DrawTriangleFill(img, tl, tr, br, seatColor)
	DrawTriangleFill(img, tl, br, bl, seatColor)

	// Seat outline.
	DrawLine(img, tl, tr, lw, ColorChair)
	DrawLine(img, tr, br, lw, ColorChair)
	DrawLine(img, br, bl, lw, ColorChair)
	DrawLine(img, bl, tl, lw, ColorChair)

	// Backrest: thick bar at the top.
	backLw := vp.S(4)
	DrawLine(img, rotate(-half, -half-backLw/2), rotate(half, -half-backLw/2), backLw, ColorChair)

	// Legs: 4 small circles at corners.
	legR := vp.S(2.5)
	legColor := color.NRGBA{R: 0x60, G: 0x60, B: 0x60, A: 0xff}
	DrawCircleFill(img, tl, legR, legColor)
	DrawCircleFill(img, tr, legR, legColor)
	DrawCircleFill(img, br, legR, legColor)
	DrawCircleFill(img, bl, legR, legColor)
}
