package court

import (
	"image"
	"math"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// ApronMeters is the width of the run-off area around the court (FIBA standard: 2m).
const ApronMeters = 2.0

// Point is a 2D coordinate in pixel space.
type Point struct {
	X, Y float32
}

// Pt creates a Point from x, y coordinates.
func Pt(x, y float32) Point {
	return Point{X: x, Y: y}
}

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
	OffsetX      float64
	OffsetY      float64
	Width        float64
	Height       float64
	Scale        float64 // zoom scale factor (1.0 = normal)
	ElementScale float64 // court-type scale for players/accessories (1.0 = half court reference)
}

// S scales a pixel value by element scale and zoom, returns float32.
func (v *Viewport) S(px float64) float32 {
	es := v.ElementScale
	if es <= 0 {
		es = 1.0
	}
	s := px * es
	if v.Scale > 1.0 {
		s *= v.Scale
	}
	return float32(s)
}

// Sf is an alias for S.
func (v *Viewport) Sf(px float64) float32 {
	return v.S(px)
}

// Sd scales a pixel value by element scale and zoom, returns float64.
func (v *Viewport) Sd(px float64) float64 {
	es := v.ElementScale
	if es <= 0 {
		es = 1.0
	}
	s := px * es
	if v.Scale > 1.0 {
		s *= v.Scale
	}
	return s
}

// RelToPixel converts a relative position [0,1] to pixel coordinates.
// Y-flip: model [0,0] = bottom-left, screen [0,0] = top-left.
func (v *Viewport) RelToPixel(pos model.Position) Point {
	return Point{
		X: float32(v.OffsetX + pos.X()*v.Width),
		Y: float32(v.OffsetY + (1.0-pos.Y())*v.Height),
	}
}

// PixelToRel converts pixel coordinates to a relative position [0,1].
func (v *Viewport) PixelToRel(p Point) model.Position {
	return model.Position{
		(float64(p.X) - v.OffsetX) / v.Width,
		1.0 - (float64(p.Y)-v.OffsetY)/v.Height,
	}
}

// MeterToPixel converts a distance in meters to pixels using the viewport and geometry.
func (v *Viewport) MeterToPixel(meters float64, geom *CourtGeometry, courtType model.CourtType) float64 {
	courtW, courtH := courtDimensions(geom, courtType)
	scaleX := v.Width / courtW
	scaleY := v.Height / courtH
	scale := math.Min(scaleX, scaleY)
	return meters * scale
}

// courtDimensions returns the logical width/height of the court based on type.
func courtDimensions(geom *CourtGeometry, courtType model.CourtType) (float64, float64) {
	w := geom.Width
	h := geom.Length
	if courtType == model.HalfCourt {
		h = geom.Length / 2
	}
	return w, h
}

// ComputeViewport computes a Viewport that fits the court (plus 2m apron)
// into the given widget size while maintaining the court's aspect ratio.
func ComputeViewport(courtType model.CourtType, geom *CourtGeometry, widgetSize image.Point, padding int) Viewport {
	courtW, courtH := courtDimensions(geom, courtType)

	// Total area includes 2m apron on each side.
	totalW := courtW + 2*ApronMeters
	totalH := courtH + 2*ApronMeters
	totalAspect := totalW / totalH

	availW := float64(widgetSize.X - 2*padding)
	availH := float64(widgetSize.Y - 2*padding)
	if availW <= 0 || availH <= 0 {
		return Viewport{}
	}

	// Fit total area (court + apron) into available space.
	var fitW, fitH float64
	if availW/availH > totalAspect {
		fitH = availH
		fitW = fitH * totalAspect
	} else {
		fitW = availW
		fitH = fitW / totalAspect
	}

	pxPerMeter := fitW / totalW
	vpW := courtW * pxPerMeter
	vpH := courtH * pxPerMeter
	apronPx := ApronMeters * pxPerMeter

	totalOriginX := (float64(widgetSize.X) - fitW) / 2
	totalOriginY := (float64(widgetSize.Y) - fitH) / 2

	// Element scale: proportional to pixel density, normalized so half court = 1.0.
	halfW, halfH := courtDimensions(geom, model.HalfCourt)
	halfTotalW := halfW + 2*ApronMeters
	halfTotalH := halfH + 2*ApronMeters
	halfAspect := halfTotalW / halfTotalH
	var halfFitW float64
	if availW/availH > halfAspect {
		halfFitW = availH * halfAspect
	} else {
		halfFitW = availW
	}
	halfPxPerMeter := halfFitW / halfTotalW

	elementScale := 1.0
	if halfPxPerMeter > 0 {
		elementScale = pxPerMeter / halfPxPerMeter
	}
	// Cap so elements don't appear oversized relative to the court.
	// The apron shrinks the court viewport; apply an extra 0.85 factor
	// so players look well-proportioned on half court.
	maxScale := geom.Width / (geom.Width + 2*ApronMeters) * 0.85
	if elementScale > maxScale {
		elementScale = maxScale
	}

	return Viewport{
		OffsetX:      totalOriginX + apronPx,
		OffsetY:      totalOriginY + apronPx,
		Width:        vpW,
		Height:       vpH,
		ElementScale: elementScale,
	}
}
