package icon

import (
	"bytes"
	"image"
	_ "image/png"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	pngassets "github.com/darkweaver87/courtdraw/assets/icons"
)

// PngIcon holds a decoded PNG image ready for Gio rendering.
type PngIcon struct {
	op   paint.ImageOp
	size image.Point
	ok   bool
}

// LoadPng loads a PNG icon by name from assets/icons/.
func LoadPng(name string) *PngIcon {
	data, err := pngassets.FS.ReadFile(name + ".png")
	if err != nil {
		return &PngIcon{}
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return &PngIcon{}
	}
	return &PngIcon{
		op:   paint.NewImageOp(img),
		size: img.Bounds().Size(),
		ok:   true,
	}
}

// Valid returns whether the icon was loaded successfully.
func (p *PngIcon) Valid() bool {
	return p.ok
}

// Layout renders the PNG icon scaled to the given dp size.
func (p *PngIcon) Layout(gtx layout.Context, sz unit.Dp) layout.Dimensions {
	if !p.ok {
		return layout.Dimensions{}
	}
	pxSize := gtx.Dp(sz)

	// Scale from source size to target size.
	scaleX := float32(pxSize) / float32(p.size.X)
	scaleY := float32(pxSize) / float32(p.size.Y)

	defer clip.Rect{Max: image.Pt(pxSize, pxSize)}.Push(gtx.Ops).Pop()
	aff := f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(scaleX, scaleY))
	defer op.Affine(aff).Push(gtx.Ops).Pop()

	p.op.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)

	return layout.Dimensions{Size: image.Pt(pxSize, pxSize)}
}

// Basketball tool palette icons.
var (
	// Players
	PlayerAttacker = LoadPng("attacker")
	PlayerDefender = LoadPng("defender")
	PlayerCoach    = LoadPng("coach")
	PlayerPG       = LoadPng("pg")
	PlayerSG       = LoadPng("sg")
	PlayerSF       = LoadPng("sf")
	PlayerPF       = LoadPng("pf")
	PlayerCenter   = LoadPng("center")
	PlayerQueue    = LoadPng("queue")

	// Actions
	ActionPass     = LoadPng("pass")
	ActionDribble  = LoadPng("dribble")
	ActionSprint   = LoadPng("sprint")
	ActionShot     = LoadPng("shot")
	ActionScreen   = LoadPng("screen")
	ActionCut      = LoadPng("cut")
	ActionCloseOut = LoadPng("close-out")
	ActionContest  = LoadPng("contest")
	ActionReverse  = LoadPng("reverse")

	// Accessories
	AccCone   = LoadPng("cone")
	AccLadder = LoadPng("ladder")
	AccChair  = LoadPng("chair")
)
