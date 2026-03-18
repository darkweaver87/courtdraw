package court

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

// shapeBounds computes a tight bounding box for a set of points with padding,
// clamped to the image bounds.
func shapeBounds(img *image.RGBA, pad float32, pts ...Point) image.Rectangle {
	minX := pts[0].X
	minY := pts[0].Y
	maxX := pts[0].X
	maxY := pts[0].Y
	for _, p := range pts[1:] {
		if p.X < minX {
			minX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	b := image.Rect(
		int(minX-pad), int(minY-pad),
		int(maxX+pad)+1, int(maxY+pad)+1,
	)
	return b.Intersect(img.Bounds())
}

// DrawLine draws a straight line from p1 to p2 with the given width.
func DrawLine(img *image.RGBA, p1, p2 Point, width float32, col color.NRGBA) {
	dx := p2.X - p1.X
	dy := p2.Y - p1.Y
	dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if dist < 0.1 {
		return
	}

	px := -dy / dist * width / 2
	py := dx / dist * width / 2

	corners := []Point{
		{p1.X + px, p1.Y + py},
		{p2.X + px, p2.Y + py},
		{p2.X - px, p2.Y - py},
		{p1.X - px, p1.Y - py},
	}

	bounds := shapeBounds(img, 1, corners...)
	if bounds.Empty() {
		return
	}
	ox := float32(bounds.Min.X)
	oy := float32(bounds.Min.Y)

	var r vector.Rasterizer
	r.Reset(bounds.Dx(), bounds.Dy())
	r.MoveTo(corners[0].X-ox, corners[0].Y-oy)
	r.LineTo(corners[1].X-ox, corners[1].Y-oy)
	r.LineTo(corners[2].X-ox, corners[2].Y-oy)
	r.LineTo(corners[3].X-ox, corners[3].Y-oy)
	r.ClosePath()
	r.Draw(img, bounds, image.NewUniform(col), image.Point{})
}

// DrawDashedLine draws a dashed line from p1 to p2.
func DrawDashedLine(img *image.RGBA, p1, p2 Point, width, dashLen, gapLen float32, col color.NRGBA) {
	dx := p2.X - p1.X
	dy := p2.Y - p1.Y
	totalLen := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if totalLen < 1 {
		return
	}

	ux := dx / totalLen
	uy := dy / totalLen

	var offset float32
	for offset < totalLen {
		segEnd := offset + dashLen
		if segEnd > totalLen {
			segEnd = totalLen
		}
		start := Pt(p1.X+ux*offset, p1.Y+uy*offset)
		end := Pt(p1.X+ux*segEnd, p1.Y+uy*segEnd)
		DrawLine(img, start, end, width, col)
		offset = segEnd + gapLen
	}
}

// circleSegments returns the number of segments for a circle of the given radius.
func circleSegments(radius float32) int {
	if radius < 10 {
		return 16
	}
	if radius < 30 {
		return 24
	}
	return 32
}

// DrawCircleOutline draws a circle outline at center with radius.
func DrawCircleOutline(img *image.RGBA, center Point, radius, width float32, col color.NRGBA) {
	segments := circleSegments(radius)
	outerR := radius + width/2
	innerR := radius - width/2
	if innerR < 0 {
		innerR = 0
	}

	bounds := shapeBounds(img, 1, Pt(center.X-outerR, center.Y-outerR), Pt(center.X+outerR, center.Y+outerR))
	if bounds.Empty() {
		return
	}
	ox := float32(bounds.Min.X)
	oy := float32(bounds.Min.Y)

	var r vector.Rasterizer
	r.Reset(bounds.Dx(), bounds.Dy())

	// Outer circle (clockwise).
	for i := 0; i <= segments; i++ {
		angle := float64(i) * 2 * math.Pi / float64(segments)
		x := center.X + outerR*float32(math.Cos(angle)) - ox
		y := center.Y + outerR*float32(math.Sin(angle)) - oy
		if i == 0 {
			r.MoveTo(x, y)
		} else {
			r.LineTo(x, y)
		}
	}
	r.ClosePath()

	// Inner circle (counter-clockwise for non-zero winding subtraction).
	for i := segments; i >= 0; i-- {
		angle := float64(i) * 2 * math.Pi / float64(segments)
		x := center.X + innerR*float32(math.Cos(angle)) - ox
		y := center.Y + innerR*float32(math.Sin(angle)) - oy
		if i == segments {
			r.MoveTo(x, y)
		} else {
			r.LineTo(x, y)
		}
	}
	r.ClosePath()

	r.Draw(img, bounds, image.NewUniform(col), image.Point{})
}

// DrawEllipseFill draws a filled ellipse at center with rx (horizontal) and ry (vertical) radii.
func DrawEllipseFill(img *image.RGBA, center Point, rx, ry float32, col color.NRGBA) {
	segments := circleSegments(max(rx, ry))

	bounds := shapeBounds(img, 1, Pt(center.X-rx, center.Y-ry), Pt(center.X+rx, center.Y+ry))
	if bounds.Empty() {
		return
	}
	ox := float32(bounds.Min.X)
	oy := float32(bounds.Min.Y)

	var r vector.Rasterizer
	r.Reset(bounds.Dx(), bounds.Dy())

	for i := 0; i <= segments; i++ {
		angle := float64(i) * 2 * math.Pi / float64(segments)
		x := center.X + rx*float32(math.Cos(angle)) - ox
		y := center.Y + ry*float32(math.Sin(angle)) - oy
		if i == 0 {
			r.MoveTo(x, y)
		} else {
			r.LineTo(x, y)
		}
	}
	r.ClosePath()

	r.Draw(img, bounds, image.NewUniform(col), image.Point{})
}

// DrawRotatedEllipseFill draws a filled ellipse rotated by angle (degrees, 0=up).
func DrawRotatedEllipseFill(img *image.RGBA, center Point, rx, ry float32, angleDeg float64, col color.NRGBA) {
	segments := circleSegments(max(rx, ry))
	rad := angleDeg * math.Pi / 180
	cosA := float32(math.Cos(rad))
	sinA := float32(math.Sin(rad))

	// Compute bounding box from rotated extents.
	maxR := max(rx, ry)
	bounds := shapeBounds(img, 1, Pt(center.X-maxR, center.Y-maxR), Pt(center.X+maxR, center.Y+maxR))
	if bounds.Empty() {
		return
	}
	ox := float32(bounds.Min.X)
	oy := float32(bounds.Min.Y)

	var r vector.Rasterizer
	r.Reset(bounds.Dx(), bounds.Dy())

	for i := 0; i <= segments; i++ {
		t := float64(i) * 2 * math.Pi / float64(segments)
		// Ellipse point before rotation.
		lx := rx * float32(math.Cos(t))
		ly := ry * float32(math.Sin(t))
		// Apply rotation.
		x := center.X + lx*cosA - ly*sinA - ox
		y := center.Y + lx*sinA + ly*cosA - oy
		if i == 0 {
			r.MoveTo(x, y)
		} else {
			r.LineTo(x, y)
		}
	}
	r.ClosePath()

	r.Draw(img, bounds, image.NewUniform(col), image.Point{})
}

// DrawCircleFill draws a filled circle at center with radius.
func DrawCircleFill(img *image.RGBA, center Point, radius float32, col color.NRGBA) {
	segments := circleSegments(radius)

	bounds := shapeBounds(img, 1, Pt(center.X-radius, center.Y-radius), Pt(center.X+radius, center.Y+radius))
	if bounds.Empty() {
		return
	}
	ox := float32(bounds.Min.X)
	oy := float32(bounds.Min.Y)

	var r vector.Rasterizer
	r.Reset(bounds.Dx(), bounds.Dy())

	for i := 0; i <= segments; i++ {
		angle := float64(i) * 2 * math.Pi / float64(segments)
		x := center.X + radius*float32(math.Cos(angle)) - ox
		y := center.Y + radius*float32(math.Sin(angle)) - oy
		if i == 0 {
			r.MoveTo(x, y)
		} else {
			r.LineTo(x, y)
		}
	}
	r.ClosePath()

	r.Draw(img, bounds, image.NewUniform(col), image.Point{})
}

// DrawArc draws an arc centered at center from startAngle to endAngle (radians, CCW).
func DrawArc(img *image.RGBA, center Point, radius float32, startAngle, endAngle float64, width float32, col color.NRGBA) {
	arcLen := endAngle - startAngle
	segments := int(math.Ceil(math.Abs(arcLen) / (math.Pi / 16)))
	segments = max(segments, 4)

	outerR := radius + width/2
	innerR := radius - width/2
	if innerR < 0 {
		innerR = 0
	}

	bounds := shapeBounds(img, 1, Pt(center.X-outerR, center.Y-outerR), Pt(center.X+outerR, center.Y+outerR))
	if bounds.Empty() {
		return
	}
	ox := float32(bounds.Min.X)
	oy := float32(bounds.Min.Y)

	step := arcLen / float64(segments)

	var r vector.Rasterizer
	r.Reset(bounds.Dx(), bounds.Dy())

	// Outer arc forward.
	for i := 0; i <= segments; i++ {
		angle := startAngle + float64(i)*step
		x := center.X + outerR*float32(math.Cos(angle)) - ox
		y := center.Y - outerR*float32(math.Sin(angle)) - oy
		if i == 0 {
			r.MoveTo(x, y)
		} else {
			r.LineTo(x, y)
		}
	}

	// Inner arc backward.
	for i := segments; i >= 0; i-- {
		angle := startAngle + float64(i)*step
		x := center.X + innerR*float32(math.Cos(angle)) - ox
		y := center.Y - innerR*float32(math.Sin(angle)) - oy
		r.LineTo(x, y)
	}
	r.ClosePath()

	r.Draw(img, bounds, image.NewUniform(col), image.Point{})
}

// DrawRect draws a rectangle outline.
func DrawRect(img *image.RGBA, min, max Point, width float32, col color.NRGBA) {
	DrawLine(img, min, Pt(max.X, min.Y), width, col)
	DrawLine(img, Pt(max.X, min.Y), max, width, col)
	DrawLine(img, max, Pt(min.X, max.Y), width, col)
	DrawLine(img, Pt(min.X, max.Y), min, width, col)
}

// DrawRectFill draws a filled rectangle.
func DrawRectFill(img *image.RGBA, min, max Point, col color.NRGBA) {
	bounds := shapeBounds(img, 0, min, max)
	if bounds.Empty() {
		return
	}
	ox := float32(bounds.Min.X)
	oy := float32(bounds.Min.Y)

	var r vector.Rasterizer
	r.Reset(bounds.Dx(), bounds.Dy())
	r.MoveTo(min.X-ox, min.Y-oy)
	r.LineTo(max.X-ox, min.Y-oy)
	r.LineTo(max.X-ox, max.Y-oy)
	r.LineTo(min.X-ox, max.Y-oy)
	r.ClosePath()
	r.Draw(img, bounds, image.NewUniform(col), image.Point{})
}

// DrawZigzag draws a zigzag line from p1 to p2.
func DrawZigzag(img *image.RGBA, p1, p2 Point, width, amplitude float32, segments int, col color.NRGBA) {
	if segments < 2 {
		segments = 6
	}
	dx := p2.X - p1.X
	dy := p2.Y - p1.Y
	totalLen := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if totalLen < 1 {
		return
	}

	// Perpendicular direction.
	px := -dy / totalLen
	py := dx / totalLen

	prev := p1
	for i := 1; i <= segments; i++ {
		t := float32(i) / float32(segments)
		mid := Pt(p1.X+dx*t, p1.Y+dy*t)
		sign := float32(1)
		if i%2 == 0 {
			sign = -1
		}
		pt := Pt(mid.X+px*amplitude*sign, mid.Y+py*amplitude*sign)
		DrawLine(img, prev, pt, width, col)
		prev = pt
	}
	DrawLine(img, prev, p2, width, col)
}

// DrawArrowhead draws a triangular arrowhead at tip pointing in the direction from->tip.
func DrawArrowhead(img *image.RGBA, from, tip Point, size float32, col color.NRGBA) {
	dx := tip.X - from.X
	dy := tip.Y - from.Y
	dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if dist < 0.1 {
		return
	}

	ux := dx / dist
	uy := dy / dist
	px := -uy
	py := ux

	base := Pt(tip.X-ux*size, tip.Y-uy*size)
	left := Pt(base.X+px*size*0.5, base.Y+py*size*0.5)
	right := Pt(base.X-px*size*0.5, base.Y-py*size*0.5)

	DrawTriangleFill(img, tip, left, right, col)
}

// DrawTriangleFill draws a filled triangle.
func DrawTriangleFill(img *image.RGBA, p1, p2, p3 Point, col color.NRGBA) {
	bounds := shapeBounds(img, 1, p1, p2, p3)
	if bounds.Empty() {
		return
	}
	ox := float32(bounds.Min.X)
	oy := float32(bounds.Min.Y)

	var r vector.Rasterizer
	r.Reset(bounds.Dx(), bounds.Dy())
	r.MoveTo(p1.X-ox, p1.Y-oy)
	r.LineTo(p2.X-ox, p2.Y-oy)
	r.LineTo(p3.X-ox, p3.Y-oy)
	r.ClosePath()
	r.Draw(img, bounds, image.NewUniform(col), image.Point{})
}

// DrawText draws centered text at the given position.
func DrawText(img *image.RGBA, text string, center Point, face font.Face, col color.NRGBA) {
	if face == nil || text == "" {
		return
	}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: face,
	}
	b, _ := d.BoundString(text)
	// Center the text bounding box on center.
	d.Dot = fixed.Point26_6{
		X: fixed.I(int(center.X)) - (b.Min.X+b.Max.X)/2,
		Y: fixed.I(int(center.Y)) - (b.Min.Y+b.Max.Y)/2,
	}
	d.DrawString(text)
}

// DrawRoundedRectFill draws a filled rounded rectangle.
func DrawRoundedRectFill(img *image.RGBA, min, max Point, radius float32, col color.NRGBA) {
	if radius <= 0 {
		DrawRectFill(img, min, max, col)
		return
	}

	bounds := shapeBounds(img, 1, min, max)
	if bounds.Empty() {
		return
	}
	ox := float32(bounds.Min.X)
	oy := float32(bounds.Min.Y)

	var r vector.Rasterizer
	r.Reset(bounds.Dx(), bounds.Dy())

	// Build rounded rect path: start at top-left + radius.
	r.MoveTo(min.X+radius-ox, min.Y-oy)
	r.LineTo(max.X-radius-ox, min.Y-oy)
	addCornerArc(&r, max.X-radius-ox, min.Y+radius-oy, radius, -math.Pi/2, 0)
	r.LineTo(max.X-ox, max.Y-radius-oy)
	addCornerArc(&r, max.X-radius-ox, max.Y-radius-oy, radius, 0, math.Pi/2)
	r.LineTo(min.X+radius-ox, max.Y-oy)
	addCornerArc(&r, min.X+radius-ox, max.Y-radius-oy, radius, math.Pi/2, math.Pi)
	r.LineTo(min.X-ox, min.Y+radius-oy)
	addCornerArc(&r, min.X+radius-ox, min.Y+radius-oy, radius, math.Pi, 3*math.Pi/2)
	r.ClosePath()

	r.Draw(img, bounds, image.NewUniform(col), image.Point{})
}

// addCornerArc adds a quarter-circle arc to the rasterizer.
func addCornerArc(r *vector.Rasterizer, cx, cy, radius float32, startAngle, endAngle float64) {
	segments := 8
	step := (endAngle - startAngle) / float64(segments)
	for i := 1; i <= segments; i++ {
		angle := startAngle + float64(i)*step
		x := cx + radius*float32(math.Cos(angle))
		y := cy + radius*float32(math.Sin(angle))
		r.LineTo(x, y)
	}
}
