package court

import (
	"image"
	"image/color"
	"image/jpeg"
	"sync"

	textureassets "github.com/darkweaver87/courtdraw/assets/textures"
	"github.com/darkweaver87/courtdraw/internal/model"
)

var (
	woodOnce sync.Once
	woodTile *image.RGBA
)

// WoodFloorTexture returns the cached wood floor tile (256x256), or nil if not available.
func WoodFloorTexture() *image.RGBA {
	woodOnce.Do(func() {
		f, err := textureassets.FS.Open("wood-floor.jpg")
		if err != nil {
			return
		}
		defer f.Close()
		img, err := jpeg.Decode(f)
		if err != nil {
			return
		}
		b := img.Bounds()
		rgba := image.NewRGBA(b)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				rgba.Set(x, y, img.At(x, y))
			}
		}
		woodTile = rgba
	})
	return woodTile
}

// Warm overlay applied on top of texture (flat clipart look).
const overlayR, overlayG, overlayB = 225.0, 200.0, 170.0
const overlayAlpha = 0.50

// PlankWidthMeters is the real-world width of a single floor plank.
const PlankWidthMeters = 0.08

// PlanksPerTile is the number of planks visible in the tile texture.
const PlanksPerTile = 4

// TileRectScaled fills a rectangle by tiling a texture image scaled to real-world plank dimensions.
func TileRectScaled(dst *image.RGBA, topLeft, botRight Point, tile *image.RGBA, vp *Viewport, geom *CourtGeometry, courtType model.CourtType) {
	if vp == nil || geom == nil {
		TileRect(dst, topLeft, botRight, tile)
		return
	}
	// How many pixels = 1 meter on screen.
	pxPerMeter := vp.MeterToPixel(1.0, geom, courtType)
	// Target tile width: PlanksPerTile planks at PlankWidthMeters each.
	targetTileW := pxPerMeter * PlankWidthMeters * PlanksPerTile
	if targetTileW < 8 {
		targetTileW = 8
	}

	tb := tile.Bounds()
	tw := tb.Dx()
	th := tb.Dy()
	if tw == 0 || th == 0 {
		return
	}

	// Scale factor: how much to shrink/stretch the tile.
	scale := float64(tw) / targetTileW

	x0, y0 := int(topLeft.X), int(topLeft.Y)
	x1, y1 := int(botRight.X), int(botRight.Y)
	if x0 > x1 { x0, x1 = x1, x0 }
	if y0 > y1 { y0, y1 = y1, y0 }

	texAlpha := 1.0 - overlayAlpha
	dstBounds := dst.Bounds()
	for y := y0; y < y1; y++ {
		if y < dstBounds.Min.Y || y >= dstBounds.Max.Y {
			continue
		}
		ty := int(float64(y-y0)*scale) % th
		for x := x0; x < x1; x++ {
			if x < dstBounds.Min.X || x >= dstBounds.Max.X {
				continue
			}
			tx := int(float64(x-x0)*scale) % tw
			c := tile.RGBAAt(tb.Min.X+tx, tb.Min.Y+ty)
			r := uint8(float64(c.R)*texAlpha + overlayR*overlayAlpha)
			g := uint8(float64(c.G)*texAlpha + overlayG*overlayAlpha)
			b := uint8(float64(c.B)*texAlpha + overlayB*overlayAlpha)
			dst.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 0xff})
		}
	}
}

// TileRect fills a rectangle by tiling a texture image with a cream wash overlay (unscaled).
func TileRect(dst *image.RGBA, topLeft, botRight Point, tile *image.RGBA) {
	x0 := int(topLeft.X)
	y0 := int(topLeft.Y)
	x1 := int(botRight.X)
	y1 := int(botRight.Y)
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}

	tb := tile.Bounds()
	tw := tb.Dx()
	th := tb.Dy()
	if tw == 0 || th == 0 {
		return
	}

	texAlpha := 1.0 - overlayAlpha

	dstBounds := dst.Bounds()
	for y := y0; y < y1; y++ {
		if y < dstBounds.Min.Y || y >= dstBounds.Max.Y {
			continue
		}
		ty := (y - y0) % th
		for x := x0; x < x1; x++ {
			if x < dstBounds.Min.X || x >= dstBounds.Max.X {
				continue
			}
			tx := (x - x0) % tw
			c := tile.RGBAAt(tb.Min.X+tx, tb.Min.Y+ty)
			r := uint8(float64(c.R)*texAlpha + overlayR*overlayAlpha)
			g := uint8(float64(c.G)*texAlpha + overlayG*overlayAlpha)
			b := uint8(float64(c.B)*texAlpha + overlayB*overlayAlpha)
			dst.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 0xff})
		}
	}
}
