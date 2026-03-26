package court

import (
	"image"
	"math"
	"testing"

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

	courtW, courtH := courtDimensions(geom, model.HalfCourt)
	expectedAspect := courtW / courtH
	gotAspect := vp.Width / vp.Height
	if !approxEqual(gotAspect, expectedAspect, 0.01) {
		t.Fatalf("aspect ratio: got %.3f, want %.3f", gotAspect, expectedAspect)
	}

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

	bl := vp.RelToPixel(model.Position{0, 0})
	if bl.Y != 200 {
		t.Fatalf("[0,0] Y: got %.1f, want 200", bl.Y)
	}

	tl := vp.RelToPixel(model.Position{0, 1})
	if tl.Y != 0 {
		t.Fatalf("[0,1] Y: got %.1f, want 0", tl.Y)
	}
}

func TestMeterToPixel(t *testing.T) {
	geom := FIBAGeometry()
	vp := ComputeViewport(model.HalfCourt, geom, image.Pt(800, 600), 0)

	px := vp.MeterToPixel(1.0, geom, model.HalfCourt)
	if px <= 0 {
		t.Fatalf("1 meter = %.2f pixels, expected positive", px)
	}

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

func TestComputeViewportOriented_Landscape(t *testing.T) {
	geom := FIBAGeometry()
	vp := ComputeViewportOriented(model.HalfCourt, geom, image.Pt(800, 600), 10, model.OrientationLandscape, false)

	if !vp.Landscape {
		t.Fatal("expected landscape=true")
	}
	if vp.Width <= 0 || vp.Height <= 0 {
		t.Fatal("viewport has zero dimensions")
	}

	courtW, courtH := courtDimensions(geom, model.HalfCourt)
	expectedAspect := courtW / courtH
	gotAspect := vp.Width / vp.Height
	if !approxEqual(gotAspect, expectedAspect, 0.01) {
		t.Fatalf("landscape viewport aspect: got %.3f, want %.3f", gotAspect, expectedAspect)
	}
}

func TestRotateImage_AllOrientations(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 6))
	src.Set(0, 0, image.White)

	r90 := RotateImage(src, model.OrientationLandscape)
	if r90.Bounds().Dx() != 6 || r90.Bounds().Dy() != 4 {
		t.Fatalf("90° size: %dx%d, want 6x4", r90.Bounds().Dx(), r90.Bounds().Dy())
	}

	r180 := RotateImage(src, model.OrientationPortraitFlip)
	if r180.Bounds().Dx() != 4 || r180.Bounds().Dy() != 6 {
		t.Fatalf("180° size: %dx%d, want 4x6", r180.Bounds().Dx(), r180.Bounds().Dy())
	}

	r270 := RotateImage(src, model.OrientationLandscapeFlip)
	if r270.Bounds().Dx() != 6 || r270.Bounds().Dy() != 4 {
		t.Fatalf("270° size: %dx%d, want 6x4", r270.Bounds().Dx(), r270.Bounds().Dy())
	}

	r0 := RotateImage(src, model.OrientationPortrait)
	if r0 != src {
		t.Fatal("0° should return same pointer")
	}
}

func TestScreenToPortrait_AllOrientations(t *testing.T) {
	// 90° CW: screen(0,0) on 800×600 → portrait(0, 799)
	p := ScreenToPortrait(Pt(0, 0), model.OrientationLandscape, 800, 600)
	if p.X != 0 || p.Y != 799 {
		t.Fatalf("90° screen(0,0) got (%.0f,%.0f), want (0,799)", p.X, p.Y)
	}

	// 180°: screen(0,0) on 800×600 → portrait(799, 599)
	p = ScreenToPortrait(Pt(0, 0), model.OrientationPortraitFlip, 800, 600)
	if p.X != 799 || p.Y != 599 {
		t.Fatalf("180° screen(0,0) got (%.0f,%.0f), want (799,599)", p.X, p.Y)
	}

	// 270° CW: screen(0,0) on 800×600 → portrait(599, 0)
	p = ScreenToPortrait(Pt(0, 0), model.OrientationLandscapeFlip, 800, 600)
	if p.X != 599 || p.Y != 0 {
		t.Fatalf("270° screen(0,0) got (%.0f,%.0f), want (599,0)", p.X, p.Y)
	}

	// 0°: passthrough
	p = ScreenToPortrait(Pt(42, 77), model.OrientationPortrait, 800, 600)
	if p.X != 42 || p.Y != 77 {
		t.Fatalf("0° screen(42,77) got (%.0f,%.0f), want (42,77)", p.X, p.Y)
	}
}

func TestNextRotationCW(t *testing.T) {
	o := model.OrientationPortrait
	o = model.NextRotationCW(o)
	if o != model.OrientationLandscape {
		t.Fatalf("expected landscape, got %s", o)
	}
	o = model.NextRotationCW(o)
	if o != model.OrientationPortraitFlip {
		t.Fatalf("expected portrait_flip, got %s", o)
	}
	o = model.NextRotationCW(o)
	if o != model.OrientationLandscapeFlip {
		t.Fatalf("expected landscape_flip, got %s", o)
	}
	o = model.NextRotationCW(o)
	if o != model.OrientationPortrait {
		t.Fatalf("expected portrait, got %s", o)
	}
}

func TestPixelToRel_CornerMapping(t *testing.T) {
	vp := Viewport{OffsetX: 50, OffsetY: 50, Width: 200, Height: 400}

	pos := vp.PixelToRel(Pt(50, 50))
	if !approxEqual(pos.X(), 0, 0.001) || !approxEqual(pos.Y(), 1, 0.001) {
		t.Fatalf("top-left pixel: got %v, want [0,1]", pos)
	}
}
