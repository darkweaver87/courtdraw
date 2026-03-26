package icon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"

	"fyne.io/fyne/v2"
)

const iconSize = 32

// GenerateActionIcon creates a runtime icon showing the action's line style.
func GenerateActionIcon(style string) fyne.Resource {
	img := image.NewRGBA(image.Rect(0, 0, iconSize, iconSize))
	col := color.NRGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff}
	lw := 2

	switch style {
	case "dribble":
		drawZigzagIcon(img, col, lw)
	case "pass":
		drawDashedIcon(img, col, lw)
	case "cut":
		drawSolidArrowIcon(img, col, lw)
	case "screen":
		drawScreenIcon(img, col, lw)
	case "shot":
		drawShotIcon(img, col, lw)
	case "handoff":
		drawHandoffIcon(img, col, lw)
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return fyne.NewStaticResource(style+"-icon.png", buf.Bytes())
}

func iconLine(img *image.RGBA, x1, y1, x2, y2 int, col color.NRGBA, width int) {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	dist := math.Sqrt(dx*dx + dy*dy)
	if dist < 1 {
		return
	}
	for s := 0; s <= int(dist); s++ {
		t := float64(s) / dist
		cx := float64(x1) + dx*t
		cy := float64(y1) + dy*t
		for w := -width / 2; w <= width/2; w++ {
			px := int(cx) + int(float64(w)*(-dy/dist))
			py := int(cy) + int(float64(w)*(dx/dist))
			if px >= 0 && px < iconSize && py >= 0 && py < iconSize {
				img.Set(px, py, col)
			}
		}
	}
}

func iconArrow(img *image.RGBA, tipX, tipY int, col color.NRGBA, lw int) {
	// V-shaped arrowhead pointing right.
	iconLine(img, tipX-6, tipY-5, tipX, tipY, col, lw)
	iconLine(img, tipX-6, tipY+5, tipX, tipY, col, lw)
}

func drawZigzagIcon(img *image.RGBA, col color.NRGBA, lw int) {
	pts := [][2]int{{3, 20}, {9, 10}, {15, 22}, {21, 12}, {27, 16}}
	for i := 1; i < len(pts); i++ {
		iconLine(img, pts[i-1][0], pts[i-1][1], pts[i][0], pts[i][1], col, lw)
	}
	iconArrow(img, 27, 16, col, lw)
}

func drawDashedIcon(img *image.RGBA, col color.NRGBA, lw int) {
	y := 16
	for x := 3; x < 22; x += 6 {
		end := min(x+4, 22)
		iconLine(img, x, y, end, y, col, lw)
	}
	iconLine(img, 22, y, 28, y, col, lw)
	iconArrow(img, 28, y, col, lw)
}

func drawSolidArrowIcon(img *image.RGBA, col color.NRGBA, lw int) {
	iconLine(img, 3, 16, 28, 16, col, lw)
	iconArrow(img, 28, 16, col, lw)
}

func drawScreenIcon(img *image.RGBA, col color.NRGBA, lw int) {
	iconLine(img, 5, 16, 18, 16, col, lw)
	// T-bar.
	iconLine(img, 18, 7, 18, 25, col, lw+1)
}

func drawShotIcon(img *image.RGBA, col color.NRGBA, lw int) {
	y := 16
	for x := 3; x < 18; x += 6 {
		end := min(x+4, 18)
		iconLine(img, x, y, end, y, col, lw)
	}
	// Target circle at end.
	cx, cy, r := 24, 16, 5
	for a := 0; a < 360; a += 3 {
		rad := float64(a) * math.Pi / 180
		px := cx + int(float64(r)*math.Cos(rad))
		py := cy + int(float64(r)*math.Sin(rad))
		if px >= 0 && px < iconSize && py >= 0 && py < iconSize {
			img.Set(px, py, col)
		}
	}
	iconLine(img, cx-3, cy, cx+3, cy, col, 1)
	iconLine(img, cx, cy-3, cx, cy+3, col, 1)
}

// GenerateBallIcon creates a basketball icon (orange circle with seam lines).
func GenerateBallIcon() fyne.Resource {
	size := 48 // higher res for smoothness
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	ball := color.NRGBA{R: 0xf4, G: 0xa2, B: 0x61, A: 0xff}
	seam := color.NRGBA{R: 0x6b, G: 0x3a, B: 0x1a, A: 0xff}
	outline := color.NRGBA{R: 0x4a, G: 0x2a, B: 0x10, A: 0xff}

	cx, cy := float64(size)/2, float64(size)/2
	r := float64(size)/2 - 3

	// Filled circle with anti-aliased edge.
	for y := range size {
		for x := range size {
			dx := float64(x) - cx + 0.5
			dy := float64(y) - cy + 0.5
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= r-1 {
				img.Set(x, y, ball)
			} else if dist <= r {
				// Anti-alias edge.
				alpha := uint8((r - dist) * 255)
				img.Set(x, y, color.NRGBA{R: ball.R, G: ball.G, B: ball.B, A: alpha})
			}
		}
	}

	// Outline circle.
	for a := range 720 {
		rad := float64(a) * math.Pi / 360
		for _, off := range []float64{0, 0.5, 1.0} {
			px := int(cx + (r-off)*math.Cos(rad))
			py := int(cy + (r-off)*math.Sin(rad))
			if px >= 0 && px < size && py >= 0 && py < size {
				img.Set(px, py, outline)
			}
		}
	}

	// Horizontal seam.
	for x := int(cx - r + 3); x <= int(cx+r-3); x++ {
		img.Set(x, int(cy), seam)
		img.Set(x, int(cy)-1, seam)
	}
	// Vertical seam.
	for y := int(cy - r + 3); y <= int(cy+r-3); y++ {
		img.Set(int(cx), y, seam)
		img.Set(int(cx)-1, y, seam)
	}

	// Curved seams (the two arcs that make a basketball).
	for a := -80; a <= 80; a++ {
		rad := float64(a) * math.Pi / 180
		// Left arc.
		arcR := r * 0.75
		lx := int(cx - r*0.35 + arcR*math.Cos(rad))
		ly := int(cy + arcR*math.Sin(rad))
		if lx >= 0 && lx < size && ly >= 0 && ly < size {
			img.Set(lx, ly, seam)
			if lx+1 < size {
				img.Set(lx+1, ly, seam)
			}
		}
		// Right arc.
		rx := int(cx + r*0.35 - arcR*math.Cos(rad))
		ry := int(cy + arcR*math.Sin(rad))
		if rx >= 0 && rx < size && ry >= 0 && ry < size {
			img.Set(rx, ry, seam)
			if rx-1 >= 0 {
				img.Set(rx-1, ry, seam)
			}
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return fyne.NewStaticResource("ball-icon.png", buf.Bytes())
}

func drawHandoffIcon(img *image.RGBA, col color.NRGBA, lw int) {
	iconLine(img, 3, 16, 28, 16, col, lw)
	// Two perpendicular bars.
	iconLine(img, 13, 9, 13, 23, col, lw)
	iconLine(img, 18, 9, 18, 23, col, lw)
}
