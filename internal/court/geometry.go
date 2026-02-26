package court

import (
	"image"
	"math"

	"gioui.org/f32"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// CourtGeometry holds the physical dimensions of a basketball court in meters.
type CourtGeometry struct {
	Width                float64 // sideline to sideline
	Length               float64 // baseline to baseline
	BasketOffset         float64 // basket center from endline
	LaneWidth            float64 // paint width
	LaneLength           float64 // paint length (from baseline)
	ThreePointRadius     float64 // 3pt arc from basket center
	ThreePointCornerDist float64 // 3pt corner line distance from sideline
	FreeThrowRadius      float64 // FT semicircle radius
	CenterCircleRadius   float64 // center circle radius
	RestrictedAreaRadius float64 // no-charge semicircle radius
	BackboardWidth       float64 // backboard width
	RimDiameter          float64 // rim diameter
	LineWidth            float64 // line width
}

// Viewport maps relative court coordinates to pixel positions.
type Viewport struct {
	OffsetX float64
	OffsetY float64
	Width   float64
	Height  float64
}

// RelToPixel converts a relative position [0,1] to pixel coordinates.
// Y-flip: model [0,0] = bottom-left, screen [0,0] = top-left.
func (v *Viewport) RelToPixel(pos model.Position) f32.Point {
	return f32.Point{
		X: float32(v.OffsetX + pos.X()*v.Width),
		Y: float32(v.OffsetY + (1.0-pos.Y())*v.Height),
	}
}

// PixelToRel converts pixel coordinates to a relative position [0,1].
func (v *Viewport) PixelToRel(p f32.Point) model.Position {
	return model.Position{
		(float64(p.X) - v.OffsetX) / v.Width,
		1.0 - (float64(p.Y)-v.OffsetY)/v.Height,
	}
}

// MeterToPixel converts a distance in meters to pixels using the viewport and geometry.
func (v *Viewport) MeterToPixel(meters float64, geom *CourtGeometry, courtType model.CourtType) float64 {
	courtW, courtH := courtDimensions(geom, courtType)
	// use the smaller axis scale to stay proportional
	scaleX := v.Width / courtW
	scaleY := v.Height / courtH
	scale := math.Min(scaleX, scaleY)
	return meters * scale
}

// courtDimensions returns the logical width/height of the court based on type.
// Court is drawn vertically: width = court.Width, height depends on half/full.
func courtDimensions(geom *CourtGeometry, courtType model.CourtType) (float64, float64) {
	w := geom.Width
	h := geom.Length
	if courtType == model.HalfCourt {
		h = geom.Length / 2
	}
	return w, h
}

// ComputeViewport computes a Viewport that fits the court into the given widget size
// while maintaining the court's aspect ratio.
func ComputeViewport(courtType model.CourtType, geom *CourtGeometry, widgetSize image.Point, padding int) Viewport {
	courtW, courtH := courtDimensions(geom, courtType)
	aspect := courtW / courtH

	availW := float64(widgetSize.X - 2*padding)
	availH := float64(widgetSize.Y - 2*padding)
	if availW <= 0 || availH <= 0 {
		return Viewport{}
	}

	var vpW, vpH float64
	if availW/availH > aspect {
		// height-constrained
		vpH = availH
		vpW = vpH * aspect
	} else {
		// width-constrained
		vpW = availW
		vpH = vpW / aspect
	}

	return Viewport{
		OffsetX: (float64(widgetSize.X) - vpW) / 2,
		OffsetY: (float64(widgetSize.Y) - vpH) / 2,
		Width:   vpW,
		Height:  vpH,
	}
}
