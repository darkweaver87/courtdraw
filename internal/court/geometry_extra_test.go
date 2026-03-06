package court

import (
	"image"
	"math"
	"testing"

	"github.com/darkweaver87/courtdraw/internal/model"
)

func TestComputeViewport_WidthConstrained(t *testing.T) {
	geom := FIBAGeometry()
	vp := ComputeViewport(model.HalfCourt, geom, image.Pt(200, 800), 0)

	if vp.Width <= 0 || vp.Height <= 0 {
		t.Fatal("viewport has zero dimensions")
	}
	// Court viewport is smaller than widget because apron takes space.
	courtW, _ := courtDimensions(geom, model.HalfCourt)
	totalW := courtW + 2*ApronMeters
	expectedW := 200 * courtW / totalW
	if !approxEqual(vp.Width, expectedW, 1) {
		t.Errorf("expected width ~%.0f, got %.1f", expectedW, vp.Width)
	}
}

func TestComputeViewport_HeightConstrained(t *testing.T) {
	geom := FIBAGeometry()
	vp := ComputeViewport(model.HalfCourt, geom, image.Pt(2000, 100), 0)

	if vp.Width <= 0 || vp.Height <= 0 {
		t.Fatal("viewport has zero dimensions")
	}
	// Court viewport is smaller than widget because apron takes space.
	_, courtH := courtDimensions(geom, model.HalfCourt)
	totalH := courtH + 2*ApronMeters
	expectedH := 100 * courtH / totalH
	if !approxEqual(vp.Height, expectedH, 1) {
		t.Errorf("expected height ~%.0f, got %.1f", expectedH, vp.Height)
	}
}

func TestComputeViewport_WithPadding(t *testing.T) {
	geom := FIBAGeometry()
	vp := ComputeViewport(model.HalfCourt, geom, image.Pt(800, 600), 50)

	if vp.Width > 800-2*50 {
		t.Errorf("viewport width %.1f exceeds padded area %d", vp.Width, 800-2*50)
	}
	if vp.Height > 600-2*50 {
		t.Errorf("viewport height %.1f exceeds padded area %d", vp.Height, 600-2*50)
	}
}

func TestComputeViewport_NegativePadding(t *testing.T) {
	geom := FIBAGeometry()
	vp := ComputeViewport(model.HalfCourt, geom, image.Pt(100, 100), 60)

	if vp.Width != 0 || vp.Height != 0 {
		t.Errorf("expected zero viewport when padding exceeds widget, got %.1fx%.1f", vp.Width, vp.Height)
	}
}

func TestComputeViewport_SquareWidget(t *testing.T) {
	geom := FIBAGeometry()
	vp := ComputeViewport(model.HalfCourt, geom, image.Pt(500, 500), 0)

	courtW, courtH := courtDimensions(geom, model.HalfCourt)
	expectedAspect := courtW / courtH
	gotAspect := vp.Width / vp.Height
	if !approxEqual(gotAspect, expectedAspect, 0.01) {
		t.Errorf("aspect ratio: got %.3f, want %.3f", gotAspect, expectedAspect)
	}
}

func TestRelToPixel_Center(t *testing.T) {
	vp := Viewport{OffsetX: 100, OffsetY: 50, Width: 400, Height: 300}

	center := vp.RelToPixel(model.Position{0.5, 0.5})
	if !approxEqual(float64(center.X), 300, 0.5) {
		t.Errorf("center X: got %.1f, want 300", center.X)
	}
	if !approxEqual(float64(center.Y), 200, 0.5) {
		t.Errorf("center Y: got %.1f, want 200", center.Y)
	}
}

func TestRelToPixel_WithOffset(t *testing.T) {
	vp := Viewport{OffsetX: 100, OffsetY: 200, Width: 400, Height: 300}

	bl := vp.RelToPixel(model.Position{0, 0})
	if !approxEqual(float64(bl.X), 100, 0.5) || !approxEqual(float64(bl.Y), 500, 0.5) {
		t.Errorf("[0,0] with offset: got (%.1f, %.1f), want (100, 500)", bl.X, bl.Y)
	}

	tr := vp.RelToPixel(model.Position{1, 1})
	if !approxEqual(float64(tr.X), 500, 0.5) || !approxEqual(float64(tr.Y), 200, 0.5) {
		t.Errorf("[1,1] with offset: got (%.1f, %.1f), want (500, 200)", tr.X, tr.Y)
	}
}

func TestPixelToRel_OutOfBounds(t *testing.T) {
	vp := Viewport{OffsetX: 100, OffsetY: 100, Width: 200, Height: 200}

	pos := vp.PixelToRel(Pt(50, 50))
	if pos.X() >= 0 || pos.Y() <= 1 {
		t.Errorf("expected negative X and Y>1 for out-of-bounds pixel, got (%.2f, %.2f)", pos.X(), pos.Y())
	}
}

func TestPixelToRel_RoundTrip_AllCorners(t *testing.T) {
	geom := NBAGeometry()
	vp := ComputeViewport(model.FullCourt, geom, image.Pt(1200, 900), 20)

	corners := []model.Position{
		{0, 0},
		{1, 0},
		{0, 1},
		{1, 1},
		{0.5, 0.5},
		{0.1, 0.9},
		{0.9, 0.1},
	}

	for _, pos := range corners {
		pixel := vp.RelToPixel(pos)
		back := vp.PixelToRel(pixel)
		if !approxEqual(back.X(), pos.X(), 0.002) || !approxEqual(back.Y(), pos.Y(), 0.002) {
			t.Errorf("round-trip %v → pixel %v → %v", pos, pixel, back)
		}
	}
}

func TestMeterToPixel_Proportionality(t *testing.T) {
	geom := FIBAGeometry()
	vp := ComputeViewport(model.FullCourt, geom, image.Pt(1000, 600), 10)

	m1 := vp.MeterToPixel(1.0, geom, model.FullCourt)
	m5 := vp.MeterToPixel(5.0, geom, model.FullCourt)
	m10 := vp.MeterToPixel(10.0, geom, model.FullCourt)

	if !approxEqual(m5, m1*5, 0.01) {
		t.Errorf("5m=%.2f, expected %.2f", m5, m1*5)
	}
	if !approxEqual(m10, m1*10, 0.01) {
		t.Errorf("10m=%.2f, expected %.2f", m10, m1*10)
	}
}

func TestMeterToPixel_HalfVsFull(t *testing.T) {
	geom := FIBAGeometry()
	size := image.Pt(800, 600)

	vpHalf := ComputeViewport(model.HalfCourt, geom, size, 0)
	vpFull := ComputeViewport(model.FullCourt, geom, size, 0)

	halfMeter := vpHalf.MeterToPixel(1.0, geom, model.HalfCourt)
	fullMeter := vpFull.MeterToPixel(1.0, geom, model.FullCourt)

	if halfMeter <= fullMeter {
		t.Errorf("expected half-court meter (%.2f) > full-court meter (%.2f)", halfMeter, fullMeter)
	}
}

func TestCourtDimensions_HalfCourt(t *testing.T) {
	geom := FIBAGeometry()
	w, h := courtDimensions(geom, model.HalfCourt)
	if w != 15.0 {
		t.Errorf("expected width 15.0, got %.1f", w)
	}
	if h != 14.0 {
		t.Errorf("expected half-court height 14.0, got %.1f", h)
	}
}

func TestCourtDimensions_FullCourt(t *testing.T) {
	geom := FIBAGeometry()
	w, h := courtDimensions(geom, model.FullCourt)
	if w != 15.0 {
		t.Errorf("expected width 15.0, got %.1f", w)
	}
	if h != 28.0 {
		t.Errorf("expected full-court height 28.0, got %.1f", h)
	}
}

func TestFIBAGeometry_Values(t *testing.T) {
	g := FIBAGeometry()
	if g.Width != 15.0 {
		t.Errorf("FIBA width: got %.1f, want 15.0", g.Width)
	}
	if g.Length != 28.0 {
		t.Errorf("FIBA length: got %.1f, want 28.0", g.Length)
	}
	if g.ThreePointRadius != 6.75 {
		t.Errorf("FIBA 3pt radius: got %.2f, want 6.75", g.ThreePointRadius)
	}
	if g.CenterCircleRadius != 1.80 {
		t.Errorf("FIBA center circle: got %.2f, want 1.80", g.CenterCircleRadius)
	}
}

func TestNBAGeometry_Values(t *testing.T) {
	g := NBAGeometry()
	if g.Width != 15.24 {
		t.Errorf("NBA width: got %.2f, want 15.24", g.Width)
	}
	if g.Length != 28.65 {
		t.Errorf("NBA length: got %.2f, want 28.65", g.Length)
	}
	if g.ThreePointRadius != 7.24 {
		t.Errorf("NBA 3pt radius: got %.2f, want 7.24", g.ThreePointRadius)
	}
}

func TestNBAGeometry_DiffersFromFIBA(t *testing.T) {
	fiba := FIBAGeometry()
	nba := NBAGeometry()

	if fiba.ThreePointRadius == nba.ThreePointRadius {
		t.Error("FIBA and NBA 3pt radius should differ")
	}
	if fiba.Width == nba.Width {
		t.Error("FIBA and NBA court width should differ")
	}
}

func TestComputeViewport_AspectPreserved_NBA(t *testing.T) {
	geom := NBAGeometry()

	sizes := []image.Point{
		image.Pt(400, 400),
		image.Pt(1920, 1080),
		image.Pt(300, 900),
	}

	for _, size := range sizes {
		vp := ComputeViewport(model.HalfCourt, geom, size, 10)
		if vp.Width == 0 {
			continue
		}
		courtW, courtH := courtDimensions(geom, model.HalfCourt)
		expectedAspect := courtW / courtH
		gotAspect := vp.Width / vp.Height
		if !approxEqual(gotAspect, expectedAspect, 0.01) {
			t.Errorf("size %v: aspect %.3f, want %.3f", size, gotAspect, expectedAspect)
		}
	}
}

func TestMeterToPixel_ZeroViewport(t *testing.T) {
	vp := Viewport{}
	geom := FIBAGeometry()

	px := vp.MeterToPixel(1.0, geom, model.HalfCourt)
	if math.IsNaN(px) || math.IsInf(px, 0) {
		t.Errorf("expected finite result for zero viewport, got %f", px)
	}
}
