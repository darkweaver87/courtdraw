package court

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/font"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// Player visual constants (base sizes at 1x zoom).
const (
	PlayerRadius       = 14
	PlayerOutlineWidth = 2
	QueueRadius        = 8
	QueueSpacing       = 20
	BallRadius         = 6
	BallOutlineWidth   = 1.5
	BallOffsetX        = 8
	BallOffsetY        = 8
)

var (
	HighlightColor = color.NRGBA{R: 0xff, G: 0xff, B: 0x00, A: 0xcc}
	BallColor      = color.NRGBA{R: 0xf4, G: 0xa2, B: 0x61, A: 0xff}
)

// DrawPlayerWithLabel draws a player circle with role color, label, optional selection highlight and ball.
func DrawPlayerWithLabel(img *image.RGBA, vp *Viewport, player *model.Player, label string, face font.Face, selected bool, hasBall bool) {
	center := vp.RelToPixel(player.Position)
	col := model.RoleColor(player.Role)

	pr := vp.S(PlayerRadius)
	pw := vp.S(PlayerOutlineWidth)

	if selected {
		DrawCircleOutline(img, center, pr+vp.S(4), vp.S(2.5), HighlightColor)
	}

	DrawCircleFill(img, center, pr, col)
	DrawCircleOutline(img, center, pr, pw,
		color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})

	drawDirectionArrow(img, vp, center, player.Rotation, 0xff)

	if face != nil && label != "" {
		DrawText(img, label, center, face, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	}

	if hasBall {
		DrawBallScaled(img, vp, center)
	}

	if player.Type == "queue" && player.Count > 1 {
		drawQueue(img, vp, center, player, col)
	}
}

// DrawPlayerWithOpacity draws a player with a given opacity (0.0–1.0).
func DrawPlayerWithOpacity(img *image.RGBA, vp *Viewport, player *model.Player, label string, face font.Face, opacity float64, hasBall bool) {
	if opacity <= 0 {
		return
	}
	alpha := uint8(opacity * 255)

	center := vp.RelToPixel(player.Position)
	col := model.RoleColor(player.Role)
	col.A = alpha

	pr := vp.S(PlayerRadius)
	pw := vp.S(PlayerOutlineWidth)

	DrawCircleFill(img, center, pr, col)
	DrawCircleOutline(img, center, pr, pw,
		color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: alpha})

	drawDirectionArrow(img, vp, center, player.Rotation, alpha)

	if face != nil && label != "" {
		DrawText(img, label, center, face, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: alpha})
	}

	if hasBall {
		DrawBallWithOpacity(img, center, opacity)
	}
}

// DrawPlayerSimple draws a player circle with role color (no text label).
func DrawPlayerSimple(img *image.RGBA, vp *Viewport, player *model.Player) {
	center := vp.RelToPixel(player.Position)
	col := model.RoleColor(player.Role)

	pr := vp.S(PlayerRadius)
	pw := vp.S(PlayerOutlineWidth)

	DrawCircleFill(img, center, pr, col)
	DrawCircleOutline(img, center, pr, pw,
		color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})

	if player.Type == "queue" && player.Count > 1 {
		drawQueue(img, vp, center, player, col)
	}
}

// DrawBall draws a small orange basketball circle at the given pixel position.
func DrawBall(img *image.RGBA, center Point) {
	ballCenter := Pt(center.X+BallOffsetX, center.Y+BallOffsetY)
	DrawCircleFill(img, ballCenter, BallRadius, BallColor)
	DrawCircleOutline(img, ballCenter, BallRadius, BallOutlineWidth,
		color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff})
}

// DrawBallScaled draws a ball scaled by the viewport zoom.
func DrawBallScaled(img *image.RGBA, vp *Viewport, center Point) {
	ballCenter := Pt(center.X+vp.Sf(BallOffsetX), center.Y+vp.Sf(BallOffsetY))
	DrawCircleFill(img, ballCenter, vp.S(BallRadius), BallColor)
	DrawCircleOutline(img, ballCenter, vp.S(BallRadius), vp.S(BallOutlineWidth),
		color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff})
}

// DrawBallWithOpacity draws a ball with the given opacity.
func DrawBallWithOpacity(img *image.RGBA, center Point, opacity float64) {
	if opacity <= 0 {
		return
	}
	alpha := uint8(opacity * 255)
	ballCenter := Pt(center.X+BallOffsetX, center.Y+BallOffsetY)
	col := BallColor
	col.A = alpha
	DrawCircleFill(img, ballCenter, BallRadius, col)
	DrawCircleOutline(img, ballCenter, BallRadius, BallOutlineWidth,
		color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: alpha})
}

// DrawCallout draws a speech bubble with text above the player.
func DrawCallout(img *image.RGBA, vp *Viewport, player *model.Player, calloutText string, face font.Face, alpha uint8) {
	if calloutText == "" || face == nil {
		return
	}

	center := vp.RelToPixel(player.Position)
	pr := vp.S(PlayerRadius)

	// Measure text bounds.
	d := &font.Drawer{Face: face}
	b, _ := d.BoundString(calloutText)
	textW := (b.Max.X - b.Min.X).Ceil()
	textH := (b.Max.Y - b.Min.Y).Ceil()

	padX := 4
	padY := 2
	bubbleW := textW + padX*2
	bubbleH := textH + padY*2

	bubbleX := int(center.X) - bubbleW/2
	bubbleY := int(center.Y) - int(pr+0.5) - 6 - bubbleH

	// Draw background.
	bgAlpha := uint8(float64(alpha) * 0.85)
	bgCol := color.NRGBA{R: 0x20, G: 0x20, B: 0x20, A: bgAlpha}
	DrawRoundedRectFill(img,
		Pt(float32(bubbleX), float32(bubbleY)),
		Pt(float32(bubbleX+bubbleW), float32(bubbleY+bubbleH)),
		3, bgCol)

	// Draw text.
	textCenter := Pt(float32(bubbleX+bubbleW/2), float32(bubbleY+bubbleH/2))
	DrawText(img, calloutText, textCenter, face, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: alpha})
}

// DrawRotationHandle draws the rotation handle (line + yellow circle) for the selected element.
func DrawRotationHandle(img *image.RGBA, vp *Viewport, center Point, rotation float64) {
	handleCenter := RotationHandlePosScaled(vp, center, rotation)
	hr := vp.S(RotationHandleRadius)
	DrawLine(img, center, handleCenter, vp.S(1.0),
		color.NRGBA{R: 0xff, G: 0xff, B: 0x00, A: 0x99})
	DrawCircleFill(img, handleCenter, hr,
		color.NRGBA{R: 0xff, G: 0xff, B: 0x00, A: 0xcc})
	DrawCircleOutline(img, handleCenter, hr, vp.S(1.0),
		color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xcc})
}

// Rotation handle constants (base sizes at 1x zoom).
const (
	RotationHandleDist   = 24
	RotationHandleRadius = 5
)

// RotationHandlePos computes the pixel position of the rotation handle (unscaled, for legacy use).
func RotationHandlePos(center Point, rotation float64) Point {
	rad := rotation * math.Pi / 180
	return Point{
		X: center.X + float32(math.Sin(rad))*RotationHandleDist,
		Y: center.Y - float32(math.Cos(rad))*RotationHandleDist,
	}
}

// RotationHandlePosScaled computes the pixel position of the rotation handle, scaled by zoom.
func RotationHandlePosScaled(vp *Viewport, center Point, rotation float64) Point {
	dist := vp.S(RotationHandleDist)
	rad := rotation * math.Pi / 180
	return Point{
		X: center.X + float32(math.Sin(rad))*dist,
		Y: center.Y - float32(math.Cos(rad))*dist,
	}
}

// HitTestRotationHandleScaled checks if pos hits the rotation handle, with zoom scaling.
func HitTestRotationHandleScaled(vp *Viewport, center Point, rotation float64, pos Point) bool {
	handleCenter := RotationHandlePosScaled(vp, center, rotation)
	dx := pos.X - handleCenter.X
	dy := pos.Y - handleCenter.Y
	dist := math.Sqrt(float64(dx*dx + dy*dy))
	return dist <= vp.Sd(RotationHandleRadius+4)
}

// drawDirectionArrow draws a small white semi-transparent triangle inside the
// player circle, pointing in the direction of rotation.
func drawDirectionArrow(img *image.RGBA, vp *Viewport, center Point, rotation float64, alpha uint8) {
	rad := rotation * math.Pi / 180
	cos := float32(math.Cos(rad))
	sin := float32(math.Sin(rad))

	r := vp.Sf(PlayerRadius)
	tipDist := r * 0.75
	halfW := r * 0.3

	rotate := func(lx, ly float32) Point {
		return Point{
			X: center.X + lx*cos - ly*sin,
			Y: center.Y + lx*sin + ly*cos,
		}
	}

	tip := rotate(0, -tipDist)
	left := rotate(-halfW, tipDist*0.3)
	right := rotate(halfW, tipDist*0.3)

	a := uint8(float64(alpha) * 0.6)
	DrawTriangleFill(img, tip, left, right, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: a})
}

func drawQueue(img *image.RGBA, vp *Viewport, center Point, player *model.Player, col color.NRGBA) {
	count := player.Count
	if count > 4 {
		count = 4
	}
	qCol := col
	qCol.A = 0xaa
	qr := vp.S(QueueRadius)
	qs := vp.Sf(QueueSpacing)
	for i := 1; i < count; i++ {
		offset := float32(i) * qs
		qCenter := Pt(center.X, center.Y+offset)
		DrawCircleFill(img, qCenter, qr, qCol)
		DrawCircleOutline(img, qCenter, qr, 1,
			color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xaa})
	}
}

// HitTestPlayer returns the index of the player under pos, or -1.
func HitTestPlayer(vp *Viewport, seq *model.Sequence, pos Point) int {
	hitR := vp.Sd(PlayerRadius + 4)
	for i := len(seq.Players) - 1; i >= 0; i-- {
		center := vp.RelToPixel(seq.Players[i].Position)
		dx := pos.X - center.X
		dy := pos.Y - center.Y
		dist := math.Sqrt(float64(dx*dx + dy*dy))
		if dist <= hitR {
			return i
		}
	}
	return -1
}

// HitTestRotationHandle checks if pos hits the rotation handle of an element at center with given rotation.
func HitTestRotationHandle(center Point, rotation float64, pos Point) bool {
	handleCenter := RotationHandlePos(center, rotation)
	dx := pos.X - handleCenter.X
	dy := pos.Y - handleCenter.Y
	dist := math.Sqrt(float64(dx*dx + dy*dy))
	return dist <= RotationHandleRadius+4
}

// ClampPosition clamps a relative position to [0,1].
func ClampPosition(p model.Position) model.Position {
	if p[0] < 0 {
		p[0] = 0
	}
	if p[0] > 1 {
		p[0] = 1
	}
	if p[1] < 0 {
		p[1] = 0
	}
	if p[1] > 1 {
		p[1] = 1
	}
	return p
}

// HitTestAccessory returns the index of the accessory under pos, or -1.
func HitTestAccessory(vp *Viewport, seq *model.Sequence, pos Point) int {
	hitRadius := vp.Sd(AccessoryConeSize + 6)
	for i := len(seq.Accessories) - 1; i >= 0; i-- {
		center := vp.RelToPixel(seq.Accessories[i].Position)
		dx := pos.X - center.X
		dy := pos.Y - center.Y
		dist := math.Sqrt(float64(dx*dx + dy*dy))
		if dist <= hitRadius {
			return i
		}
	}
	return -1
}

// HitTestAction checks if pos is near the midpoint of an action line.
func HitTestAction(vp *Viewport, seq *model.Sequence, pos Point) int {
	hitThreshold := vp.Sd(12)
	for i := len(seq.Actions) - 1; i >= 0; i-- {
		from := ResolveRef(vp, seq.Actions[i].From, seq.Players)
		to := ResolveRef(vp, seq.Actions[i].To, seq.Players)
		mid := Pt((from.X+to.X)/2, (from.Y+to.Y)/2)
		dx := pos.X - mid.X
		dy := pos.Y - mid.Y
		dist := math.Sqrt(float64(dx*dx + dy*dy))
		if dist <= hitThreshold {
			return i
		}
	}
	return -1
}
