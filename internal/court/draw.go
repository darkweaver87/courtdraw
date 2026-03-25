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
// segmentLen is the target length of each zigzag segment (for consistent appearance).
func DrawZigzag(img *image.RGBA, p1, p2 Point, width, amplitude float32, segmentLen float32, col color.NRGBA) {
	dx := p2.X - p1.X
	dy := p2.Y - p1.Y
	totalLen := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if totalLen < 1 {
		return
	}
	segments := max(2, int(totalLen/segmentLen))

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

// DrawScreenBar draws a T-shaped perpendicular bar at the screen position.
// tip is the screen endpoint, from is the direction the screen line comes from.
func DrawScreenBar(img *image.RGBA, tip, from Point, barLen, barWidth float32, col color.NRGBA) {
	dx := tip.X - from.X
	dy := tip.Y - from.Y
	dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if dist < 0.1 {
		return
	}
	// Perpendicular direction.
	px := -dy / dist
	py := dx / dist
	p1 := Pt(tip.X+px*barLen/2, tip.Y+py*barLen/2)
	p2 := Pt(tip.X-px*barLen/2, tip.Y-py*barLen/2)
	DrawLine(img, p1, p2, barWidth, col)
}

// DrawHandoffBars draws two short perpendicular bars at the midpoint of a handoff action.
func DrawHandoffBars(img *image.RGBA, from, to Point, lineWidth, barLen float32, col color.NRGBA) {
	dx := to.X - from.X
	dy := to.Y - from.Y
	dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if dist < 1 {
		return
	}
	px := -dy / dist
	py := dx / dist
	// Two bars at 40% and 60% along the line.
	for _, t := range []float32{0.4, 0.6} {
		mid := Pt(from.X+dx*t, from.Y+dy*t)
		p1 := Pt(mid.X+px*barLen, mid.Y+py*barLen)
		p2 := Pt(mid.X-px*barLen, mid.Y-py*barLen)
		DrawLine(img, p1, p2, lineWidth, col)
	}
}

// DistToPolyline returns the shortest distance from point p to any segment of the polyline.
func DistToPolyline(p Point, pts []Point) float64 {
	if len(pts) < 2 {
		return math.MaxFloat64
	}
	best := math.MaxFloat64
	for i := 1; i < len(pts); i++ {
		d := DistToSegment(p, pts[i-1], pts[i])
		if d < best {
			best = d
		}
	}
	return best
}

// --- Curved path utilities ---

// BezierPath generates a polyline from→waypoints→to using quadratic Bézier curves.
// Waypoints are points ON the curve (pass-through), not control points.
// The Bézier control points are computed so the curve passes through each waypoint.
// numSegments is the number of line segments per curve section.
func BezierPath(from, to Point, waypoints []Point, numSegments int) []Point {
	if len(waypoints) == 0 {
		return []Point{from, to}
	}
	if numSegments < 4 {
		numSegments = 16
	}

	if len(waypoints) == 1 {
		// Single waypoint: compute control point so curve passes through waypoint at t=0.5.
		// C = 2*M - 0.5*(P0 + P2)
		m := waypoints[0]
		ctrl := Pt(2*m.X-0.5*(from.X+to.X), 2*m.Y-0.5*(from.Y+to.Y))
		return quadBezierPoints(from, ctrl, to, numSegments)
	}

	// Multiple waypoints: chain quadratic Bézier segments.
	// Convert pass-through points to control points.
	points := []Point{from}
	for i, wp := range waypoints {
		var p0, p2 Point
		if i == 0 {
			p0 = from
		} else {
			// Midpoint between previous and current waypoint.
			prev := waypoints[i-1]
			p0 = Pt((prev.X+wp.X)/2, (prev.Y+wp.Y)/2)
		}
		if i == len(waypoints)-1 {
			p2 = to
		} else {
			next := waypoints[i+1]
			p2 = Pt((wp.X+next.X)/2, (wp.Y+next.Y)/2)
		}
		// Control point so curve passes through wp at t=0.5.
		ctrl := Pt(2*wp.X-0.5*(p0.X+p2.X), 2*wp.Y-0.5*(p0.Y+p2.Y))
		segs := max(4, numSegments/len(waypoints))
		bezPts := quadBezierPoints(p0, ctrl, p2, segs)
		// Skip first point (already in previous segment).
		points = append(points, bezPts[1:]...)
	}
	return points
}

func quadBezierPoints(p0, ctrl, p2 Point, numSegments int) []Point {
	points := make([]Point, numSegments+1)
	points[0] = p0
	for j := 1; j <= numSegments; j++ {
		t := float32(j) / float32(numSegments)
		u := 1 - t
		x := u*u*p0.X + 2*u*t*ctrl.X + t*t*p2.X
		y := u*u*p0.Y + 2*u*t*ctrl.Y + t*t*p2.Y
		points[j] = Pt(x, y)
	}
	return points
}

// PolylineLength computes the total length of a polyline.
func PolylineLength(pts []Point) float64 {
	total := 0.0
	for i := 1; i < len(pts); i++ {
		dx := float64(pts[i].X - pts[i-1].X)
		dy := float64(pts[i].Y - pts[i-1].Y)
		total += math.Sqrt(dx*dx + dy*dy)
	}
	return total
}

// PolylinePointAt returns the point at a given fraction (0–1) along the polyline.
func PolylinePointAt(pts []Point, frac float64) Point {
	if len(pts) < 2 || frac <= 0 {
		return pts[0]
	}
	if frac >= 1 {
		return pts[len(pts)-1]
	}
	total := PolylineLength(pts)
	target := total * frac
	walked := 0.0
	for i := 1; i < len(pts); i++ {
		dx := float64(pts[i].X - pts[i-1].X)
		dy := float64(pts[i].Y - pts[i-1].Y)
		segLen := math.Sqrt(dx*dx + dy*dy)
		if walked+segLen >= target {
			t := (target - walked) / segLen
			return Pt(
				pts[i-1].X+float32(t)*float32(dx),
				pts[i-1].Y+float32(t)*float32(dy),
			)
		}
		walked += segLen
	}
	return pts[len(pts)-1]
}

// DrawPolyline draws a series of connected line segments.
func DrawPolyline(img *image.RGBA, pts []Point, width float32, col color.NRGBA) {
	for i := 1; i < len(pts); i++ {
		DrawLine(img, pts[i-1], pts[i], width, col)
	}
}

// DrawDashedPolyline draws a dashed line following a polyline path.
func DrawDashedPolyline(img *image.RGBA, pts []Point, width, dashLen, gapLen float32, col color.NRGBA) {
	if len(pts) < 2 {
		return
	}
	drawing := true
	remaining := dashLen
	for i := 1; i < len(pts); i++ {
		from := pts[i-1]
		to := pts[i]
		dx := to.X - from.X
		dy := to.Y - from.Y
		segLen := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		if segLen < 0.1 {
			continue
		}
		ux := dx / segLen
		uy := dy / segLen
		pos := float32(0)
		for pos < segLen {
			chunk := remaining
			if pos+chunk > segLen {
				chunk = segLen - pos
			}
			if drawing {
				p1 := Pt(from.X+ux*pos, from.Y+uy*pos)
				p2 := Pt(from.X+ux*(pos+chunk), from.Y+uy*(pos+chunk))
				DrawLine(img, p1, p2, width, col)
			}
			pos += chunk
			remaining -= chunk
			if remaining <= 0 {
				drawing = !drawing
				if drawing {
					remaining = dashLen
				} else {
					remaining = gapLen
				}
			}
		}
	}
}

// DrawZigzagPolyline draws a zigzag line following a polyline path.
func DrawZigzagPolyline(img *image.RGBA, pts []Point, width, amplitude float32, segmentLen float32, col color.NRGBA) {
	if len(pts) < 2 {
		return
	}
	total := float32(PolylineLength(pts))
	if total < 1 {
		return
	}
	segments := max(2, int(total/segmentLen))
	prev := pts[0]
	for i := 1; i <= segments; i++ {
		t := float64(i) / float64(segments)
		mid := PolylinePointAt(pts, t)
		// Compute perpendicular from tangent at this point.
		near := PolylinePointAt(pts, max(0, t-0.01))
		dx := mid.X - near.X
		dy := mid.Y - near.Y
		d := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		var nx, ny float32
		if d > 0.01 {
			nx = -dy / d
			ny = dx / d
		}
		sign := float32(1)
		if i%2 == 0 {
			sign = -1
		}
		pt := Pt(mid.X+nx*amplitude*sign, mid.Y+ny*amplitude*sign)
		DrawLine(img, prev, pt, width, col)
		prev = pt
	}
	DrawLine(img, prev, pts[len(pts)-1], width, col)
}

// DrawArrowheadAtEnd draws an arrowhead at the end of a polyline, tangent-aligned.
func DrawArrowheadAtEnd(img *image.RGBA, pts []Point, size float32, col color.NRGBA) {
	if len(pts) < 2 {
		return
	}
	tip := pts[len(pts)-1]
	// Use the last segment for direction.
	from := pts[len(pts)-2]
	DrawArrowhead(img, from, tip, size, col)
}
