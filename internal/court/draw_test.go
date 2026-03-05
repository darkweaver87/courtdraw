package court

import (
	"image"
	"image/color"
	"testing"

	"github.com/darkweaver87/courtdraw/internal/model"
)

func TestDrawLine_NosPanic(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	DrawLine(img, Pt(10, 10), Pt(190, 190), 2, color.NRGBA{R: 255, A: 255})
	// Verify some pixels along the line are non-zero.
	if img.RGBAAt(100, 100).A == 0 {
		t.Error("expected non-transparent pixel near line center")
	}
}

func TestDrawLine_TooShort(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	// Should not panic for zero-length line.
	DrawLine(img, Pt(50, 50), Pt(50, 50), 2, color.NRGBA{R: 255, A: 255})
}

func TestDrawCircleFill(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	DrawCircleFill(img, Pt(100, 100), 30, color.NRGBA{R: 255, A: 255})
	// Center should be filled.
	if img.RGBAAt(100, 100).A == 0 {
		t.Error("expected filled center pixel")
	}
	// Far corner should be empty.
	if img.RGBAAt(0, 0).A != 0 {
		t.Error("expected empty corner pixel")
	}
}

func TestDrawCircleOutline(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	DrawCircleOutline(img, Pt(100, 100), 40, 3, color.NRGBA{G: 255, A: 255})
	// Center of circle should be empty (outline only).
	if img.RGBAAt(100, 100).A != 0 {
		t.Error("expected empty center for circle outline")
	}
	// Point on the ring should be filled.
	if img.RGBAAt(140, 100).A == 0 {
		t.Error("expected filled pixel on circle edge")
	}
}

func TestDrawRectFill(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	DrawRectFill(img, Pt(20, 20), Pt(80, 80), color.NRGBA{B: 255, A: 255})
	if img.RGBAAt(50, 50).A == 0 {
		t.Error("expected filled center of rect")
	}
	if img.RGBAAt(0, 0).A != 0 {
		t.Error("expected empty outside rect")
	}
}

func TestDrawArrowhead(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	DrawArrowhead(img, Pt(50, 100), Pt(150, 100), 20, color.NRGBA{R: 255, A: 255})
	// Tip area should have pixels.
	if img.RGBAAt(145, 100).A == 0 {
		t.Error("expected filled pixel near arrowhead tip")
	}
}

func TestDrawDashedLine(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	DrawDashedLine(img, Pt(10, 100), Pt(190, 100), 2, 10, 5, color.NRGBA{R: 255, A: 255})
	// Should have drawn something.
	hasPixel := false
	for x := 10; x < 190; x++ {
		if img.RGBAAt(x, 100).A > 0 {
			hasPixel = true
			break
		}
	}
	if !hasPixel {
		t.Error("expected some filled pixels along dashed line")
	}
}

func TestDrawZigzag(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	DrawZigzag(img, Pt(10, 100), Pt(190, 100), 2, 10, 8, color.NRGBA{R: 255, A: 255})
	// Should have drawn something.
	hasPixel := false
	for x := 0; x < 200; x++ {
		for y := 0; y < 200; y++ {
			if img.RGBAAt(x, y).A > 0 {
				hasPixel = true
				break
			}
		}
		if hasPixel {
			break
		}
	}
	if !hasPixel {
		t.Error("expected some filled pixels in zigzag")
	}
}

func TestDrawArc(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	DrawArc(img, Pt(100, 100), 50, 0, 3.14159, 2, color.NRGBA{G: 255, A: 255})
	// Should have pixels on the arc. The stroke sits at y≈98-99 near angle 0.
	if img.RGBAAt(150, 99).A == 0 {
		t.Error("expected filled pixel on arc near 0 radians")
	}
}

func TestDrawFIBACourt_NoPanic(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 800, 600))
	geom := FIBAGeometry()
	vp := ComputeViewport(model.HalfCourt, geom, image.Pt(800, 600), 10)
	DrawFIBACourt(img, model.HalfCourt, &vp, geom)
	// Verify court was drawn (non-transparent pixels exist).
	hasPixel := false
	for y := 0; y < 600; y++ {
		for x := 0; x < 800; x++ {
			if img.RGBAAt(x, y).A > 0 {
				hasPixel = true
				break
			}
		}
		if hasPixel {
			break
		}
	}
	if !hasPixel {
		t.Error("expected some drawn pixels on the court")
	}
}

func TestDrawNBACourt_FullCourt_NoPanic(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 600, 800))
	geom := NBAGeometry()
	vp := ComputeViewport(model.FullCourt, geom, image.Pt(600, 800), 10)
	DrawNBACourt(img, model.FullCourt, &vp, geom)
}

func TestDrawTriangleFill(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	DrawTriangleFill(img, Pt(100, 20), Pt(20, 180), Pt(180, 180), color.NRGBA{R: 255, A: 255})
	// Center of triangle should be filled.
	if img.RGBAAt(100, 120).A == 0 {
		t.Error("expected filled center of triangle")
	}
}

func TestDrawRoundedRectFill(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	DrawRoundedRectFill(img, Pt(20, 20), Pt(180, 180), 10, color.NRGBA{B: 255, A: 255})
	// Center should be filled.
	if img.RGBAAt(100, 100).A == 0 {
		t.Error("expected filled center of rounded rect")
	}
}
