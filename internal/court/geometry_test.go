package court

import (
	"image"
	"math"
	"testing"

	"gioui.org/f32"

	"github.com/darkweaver87/courtdraw/internal/model"
)

func approxEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestComputeViewport_HalfCourt(t *testing.T) {
	geom := FIBAGeometry()
	vp := ComputeViewport(model.HalfCourt, geom, image.Pt(800, 600), 10)

	if vp.Width <= 0 || vp.Height <= 0 {
		t.Fatal("viewport has zero dimensions")
	}

	// aspect ratio should match court
	courtW, courtH := courtDimensions(geom, model.HalfCourt)
	expectedAspect := courtW / courtH
	gotAspect := vp.Width / vp.Height
	if !approxEqual(gotAspect, expectedAspect, 0.01) {
		t.Fatalf("aspect ratio: got %.3f, want %.3f", gotAspect, expectedAspect)
	}

	// should fit in widget
	if vp.OffsetX+vp.Width > 800 || vp.OffsetY+vp.Height > 600 {
		t.Fatal("viewport exceeds widget bounds")
	}
}

func TestComputeViewport_FullCourt(t *testing.T) {
	geom := NBAGeometry()
	vp := ComputeViewport(model.FullCourt, geom, image.Pt(600, 800), 20)

	if vp.Width <= 0 || vp.Height <= 0 {
		t.Fatal("viewport has zero dimensions")
	}

	courtW, courtH := courtDimensions(geom, model.FullCourt)
	expectedAspect := courtW / courtH
	gotAspect := vp.Width / vp.Height
	if !approxEqual(gotAspect, expectedAspect, 0.01) {
		t.Fatalf("aspect ratio: got %.3f, want %.3f", gotAspect, expectedAspect)
	}
}

func TestRelToPixel_RoundTrip(t *testing.T) {
	geom := FIBAGeometry()
	vp := ComputeViewport(model.HalfCourt, geom, image.Pt(1000, 800), 10)

	tests := []model.Position{
		{0, 0},
		{1, 1},
		{0.5, 0.5},
		{0.25, 0.75},
	}

	for _, pos := range tests {
		pixel := vp.RelToPixel(pos)
		back := vp.PixelToRel(pixel)
		if !approxEqual(back.X(), pos.X(), 0.001) || !approxEqual(back.Y(), pos.Y(), 0.001) {
			t.Errorf("round-trip %v → pixel %v → %v", pos, pixel, back)
		}
	}
}

func TestRelToPixel_YFlip(t *testing.T) {
	vp := Viewport{OffsetX: 0, OffsetY: 0, Width: 100, Height: 200}

	// bottom-left [0,0] → screen top-bottom: should be at Y=200 (bottom of screen)
	bl := vp.RelToPixel(model.Position{0, 0})
	if bl.Y != 200 {
		t.Fatalf("[0,0] Y: got %.1f, want 200", bl.Y)
	}

	// top-left [0,1] → screen top: should be at Y=0
	tl := vp.RelToPixel(model.Position{0, 1})
	if tl.Y != 0 {
		t.Fatalf("[0,1] Y: got %.1f, want 0", tl.Y)
	}
}

func TestMeterToPixel(t *testing.T) {
	geom := FIBAGeometry()
	vp := ComputeViewport(model.HalfCourt, geom, image.Pt(800, 600), 0)

	// 1 meter should be some positive pixel distance
	px := vp.MeterToPixel(1.0, geom, model.HalfCourt)
	if px <= 0 {
		t.Fatalf("1 meter = %.2f pixels, expected positive", px)
	}

	// 2 meters should be roughly double
	px2 := vp.MeterToPixel(2.0, geom, model.HalfCourt)
	if !approxEqual(px2, px*2, 0.01) {
		t.Fatalf("2 meters = %.2f, expected %.2f", px2, px*2)
	}
}

func TestComputeViewport_ZeroSize(t *testing.T) {
	geom := FIBAGeometry()
	vp := ComputeViewport(model.HalfCourt, geom, image.Pt(0, 0), 0)
	if vp.Width != 0 || vp.Height != 0 {
		t.Fatal("expected zero viewport for zero widget")
	}
}

func TestPixelToRel_CornerMapping(t *testing.T) {
	vp := Viewport{OffsetX: 50, OffsetY: 50, Width: 200, Height: 400}

	// pixel at offset corner = rel [0,1] (top-left of screen = top-left of court)
	pos := vp.PixelToRel(f32.Point{X: 50, Y: 50})
	if !approxEqual(pos.X(), 0, 0.001) || !approxEqual(pos.Y(), 1, 0.001) {
		t.Fatalf("top-left pixel: got %v, want [0,1]", pos)
	}
}
