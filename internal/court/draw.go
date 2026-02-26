package court

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// DrawLine draws a straight line from p1 to p2.
func DrawLine(ops *op.Ops, p1, p2 f32.Point, width float32, col color.NRGBA) {
	var path clip.Path
	path.Begin(ops)
	path.MoveTo(p1)
	path.LineTo(p2)
	stroke := clip.Stroke{
		Path:  path.End(),
		Width: width,
	}.Op()
	paint.FillShape(ops, col, stroke)
}

// DrawDashedLine draws a dashed line from p1 to p2.
func DrawDashedLine(ops *op.Ops, p1, p2 f32.Point, width, dashLen, gapLen float32, col color.NRGBA) {
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
		start := f32.Point{X: p1.X + ux*offset, Y: p1.Y + uy*offset}
		end := f32.Point{X: p1.X + ux*segEnd, Y: p1.Y + uy*segEnd}
		DrawLine(ops, start, end, width, col)
		offset = segEnd + gapLen
	}
}

// DrawCircleOutline draws a circle outline at center with radius.
func DrawCircleOutline(ops *op.Ops, center f32.Point, radius, width float32, col color.NRGBA) {
	var path clip.Path
	path.Begin(ops)

	// approximate circle with arcs
	drawCirclePath(&path, center, radius)

	stroke := clip.Stroke{
		Path:  path.End(),
		Width: width,
	}.Op()
	paint.FillShape(ops, col, stroke)
}

// DrawCircleFill draws a filled circle at center with radius.
func DrawCircleFill(ops *op.Ops, center f32.Point, radius float32, col color.NRGBA) {
	r := int(math.Ceil(float64(radius)))
	cx := int(center.X)
	cy := int(center.Y)
	ellipse := clip.Ellipse{
		Min: image.Pt(cx-r, cy-r),
		Max: image.Pt(cx+r, cy+r),
	}.Op(ops)
	paint.FillShape(ops, col, ellipse)
}

// DrawArc draws an arc centered at center from startAngle to endAngle (radians, CCW).
func DrawArc(ops *op.Ops, center f32.Point, radius float32, startAngle, endAngle float64, width float32, col color.NRGBA) {
	var path clip.Path
	path.Begin(ops)

	// number of segments for smooth arc
	arcLen := endAngle - startAngle
	segments := int(math.Ceil(math.Abs(arcLen) / (math.Pi / 32)))
	if segments < 4 {
		segments = 4
	}

	step := arcLen / float64(segments)

	startPt := f32.Point{
		X: center.X + radius*float32(math.Cos(startAngle)),
		Y: center.Y - radius*float32(math.Sin(startAngle)),
	}
	path.MoveTo(startPt)

	for i := 1; i <= segments; i++ {
		angle := startAngle + float64(i)*step
		pt := f32.Point{
			X: center.X + radius*float32(math.Cos(angle)),
			Y: center.Y - radius*float32(math.Sin(angle)),
		}
		path.LineTo(pt)
	}

	stroke := clip.Stroke{
		Path:  path.End(),
		Width: width,
	}.Op()
	paint.FillShape(ops, col, stroke)
}

// DrawRect draws a rectangle outline.
func DrawRect(ops *op.Ops, min, max f32.Point, width float32, col color.NRGBA) {
	DrawLine(ops, min, f32.Pt(max.X, min.Y), width, col)
	DrawLine(ops, f32.Pt(max.X, min.Y), max, width, col)
	DrawLine(ops, max, f32.Pt(min.X, max.Y), width, col)
	DrawLine(ops, f32.Pt(min.X, max.Y), min, width, col)
}

// DrawRectFill draws a filled rectangle.
func DrawRectFill(ops *op.Ops, min, max f32.Point, col color.NRGBA) {
	r := clip.Rect{
		Min: image.Pt(int(min.X), int(min.Y)),
		Max: image.Pt(int(max.X), int(max.Y)),
	}.Op()
	paint.FillShape(ops, col, r)
}

// DrawZigzag draws a zigzag line from p1 to p2.
func DrawZigzag(ops *op.Ops, p1, p2 f32.Point, width, amplitude float32, segments int, col color.NRGBA) {
	if segments < 2 {
		segments = 6
	}
	dx := p2.X - p1.X
	dy := p2.Y - p1.Y
	totalLen := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if totalLen < 1 {
		return
	}

	// unit vectors along and perpendicular
	ux := dx / totalLen
	uy := dy / totalLen
	px := -uy
	py := ux

	var path clip.Path
	path.Begin(ops)
	path.MoveTo(p1)

	for i := 1; i <= segments; i++ {
		t := float32(i) / float32(segments)
		mid := f32.Point{X: p1.X + dx*t, Y: p1.Y + dy*t}
		// alternate left/right
		sign := float32(1)
		if i%2 == 0 {
			sign = -1
		}
		pt := f32.Point{
			X: mid.X + px*amplitude*sign,
			Y: mid.Y + py*amplitude*sign,
		}
		path.LineTo(pt)
	}
	path.LineTo(p2)

	stroke := clip.Stroke{
		Path:  path.End(),
		Width: width,
	}.Op()
	paint.FillShape(ops, col, stroke)
}

// DrawArrowhead draws a triangular arrowhead at tip pointing in the direction from→tip.
func DrawArrowhead(ops *op.Ops, from, tip f32.Point, size float32, col color.NRGBA) {
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

	base := f32.Point{X: tip.X - ux*size, Y: tip.Y - uy*size}
	left := f32.Point{X: base.X + px*size*0.5, Y: base.Y + py*size*0.5}
	right := f32.Point{X: base.X - px*size*0.5, Y: base.Y - py*size*0.5}

	var path clip.Path
	path.Begin(ops)
	path.MoveTo(tip)
	path.LineTo(left)
	path.LineTo(right)
	path.Close()

	outline := clip.Outline{Path: path.End()}.Op()
	paint.FillShape(ops, col, outline)
}

// drawCirclePath adds a circle to a clip.Path as line segments.
func drawCirclePath(path *clip.Path, center f32.Point, radius float32) {
	segments := 64
	step := 2 * math.Pi / float64(segments)

	start := f32.Point{
		X: center.X + radius,
		Y: center.Y,
	}
	path.MoveTo(start)

	for i := 1; i <= segments; i++ {
		angle := float64(i) * step
		pt := f32.Point{
			X: center.X + radius*float32(math.Cos(angle)),
			Y: center.Y - radius*float32(math.Sin(angle)),
		}
		path.LineTo(pt)
	}
}
