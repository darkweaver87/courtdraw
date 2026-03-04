package widget

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/darkweaver87/courtdraw/internal/court"
	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
)

const (
	playerRadius       = 14
	playerOutlineWidth = 2
	queueRadius        = 8
	queueSpacing       = 20
)

var highlightColor = color.NRGBA{R: 0xff, G: 0xff, B: 0x00, A: 0xcc} // yellow

// roleLabelI18n returns the short translated label for a player role.
// Uses i18n keys "role.<role>" (e.g. "role.point_guard" → "MJ" in French).
func roleLabelI18n(role model.PlayerRole) string {
	key := "role." + string(role)
	label := i18n.T(key)
	// If key is not found (returned as-is), fall back to model.RoleLabel.
	if label == key {
		return model.RoleLabel(role)
	}
	return label
}

const (
	ballRadius      = 6
	ballOutlineWidth = 1.5
	ballOffsetX     = 8
	ballOffsetY     = 8
)

// drawDirectionArrow draws a small white semi-transparent triangle inside the
// player circle, pointing in the direction of rotation (0° = up / facing basket).
func drawDirectionArrow(ops *op.Ops, center f32.Point, rotation float64, alpha uint8) {
	rad := rotation * math.Pi / 180
	cos := float32(math.Cos(rad))
	sin := float32(math.Sin(rad))

	r := float32(playerRadius)
	// Triangle proportions relative to circle radius.
	tipDist := r * 0.75  // tip distance from center
	halfW := r * 0.3     // half-width of base

	// Rotate a local point by rotation angle.
	rotate := func(lx, ly float32) f32.Point {
		return f32.Point{
			X: center.X + lx*cos - ly*sin,
			Y: center.Y + lx*sin + ly*cos,
		}
	}

	// Triangle: tip at top (along rotation direction), base behind.
	tip := rotate(0, -tipDist)
	left := rotate(-halfW, tipDist*0.3)
	right := rotate(halfW, tipDist*0.3)

	var path clip.Path
	path.Begin(ops)
	path.MoveTo(tip)
	path.LineTo(left)
	path.LineTo(right)
	path.Close()

	outline := clip.Outline{Path: path.End()}.Op()
	a := uint8(float64(alpha) * 0.6)
	paint.FillShape(ops, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: a}, outline)
}

var ballColor = color.NRGBA{R: 0xf4, G: 0xa2, B: 0x61, A: 0xff} // #f4a261 orange

// DrawBall draws a small orange basketball circle at the given pixel position.
func DrawBall(ops *op.Ops, center f32.Point) {
	ballCenter := f32.Point{X: center.X + ballOffsetX, Y: center.Y + ballOffsetY}
	court.DrawCircleFill(ops, ballCenter, ballRadius, ballColor)
	court.DrawCircleOutline(ops, ballCenter, ballRadius, ballOutlineWidth,
		color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff})
}

// DrawBallWithOpacity draws a ball with the given opacity.
func DrawBallWithOpacity(ops *op.Ops, center f32.Point, opacity float64) {
	if opacity <= 0 {
		return
	}
	alpha := uint8(opacity * 255)
	ballCenter := f32.Point{X: center.X + ballOffsetX, Y: center.Y + ballOffsetY}
	col := ballColor
	col.A = alpha
	court.DrawCircleFill(ops, ballCenter, ballRadius, col)
	court.DrawCircleOutline(ops, ballCenter, ballRadius, ballOutlineWidth,
		color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: alpha})
}

// DrawPlayerWithOpacity draws a player with a given opacity (0.0–1.0).
func DrawPlayerWithOpacity(gtx layout.Context, th *material.Theme, vp *court.Viewport, player *model.Player, opacity float64, hasBall bool) {
	if opacity <= 0 {
		return
	}
	alpha := uint8(opacity * 255)

	center := vp.RelToPixel(player.Position)
	col := model.RoleColor(player.Role)
	col.A = alpha

	// filled circle
	court.DrawCircleFill(gtx.Ops, center, playerRadius, col)

	// white outline
	outlineCol := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: alpha}
	court.DrawCircleOutline(gtx.Ops, center, playerRadius, playerOutlineWidth, outlineCol)

	// Direction arrow.
	drawDirectionArrow(gtx.Ops, center, player.Rotation, alpha)

	// label text — translate default role labels, keep custom ones
	label := player.Label
	if label == "" || label == model.RoleLabel(player.Role) {
		label = roleLabelI18n(player.Role)
	}

	lbl := material.Label(th, unit.Sp(11), label)
	lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: alpha}
	lbl.Alignment = text.Middle

	macro := op.Record(gtx.Ops)
	labelGtx := gtx
	labelGtx.Constraints = layout.Constraints{
		Max: image.Pt(playerRadius*3, playerRadius*2),
	}
	dims := lbl.Layout(labelGtx)
	call := macro.Stop()

	offsetX := int(center.X) - dims.Size.X/2
	offsetY := int(center.Y) - dims.Size.Y/2

	clipStack := clip.Rect{
		Min: image.Pt(offsetX, offsetY),
		Max: image.Pt(offsetX+dims.Size.X, offsetY+dims.Size.Y),
	}.Push(gtx.Ops)
	transStack := op.Offset(image.Pt(offsetX, offsetY)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	transStack.Pop()
	clipStack.Pop()

	// Ball indicator.
	if hasBall {
		DrawBallWithOpacity(gtx.Ops, center, opacity)
	}
}

// DrawPlayer draws a player circle with role color and label.
func DrawPlayer(ops *op.Ops, vp *court.Viewport, player *model.Player) {
	center := vp.RelToPixel(player.Position)
	col := model.RoleColor(player.Role)

	// filled circle
	court.DrawCircleFill(ops, center, playerRadius, col)

	// white outline
	court.DrawCircleOutline(ops, center, playerRadius, playerOutlineWidth,
		color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})

	// queue: smaller grey circles behind
	if player.Type == "queue" && player.Count > 1 {
		drawQueue(ops, vp, player)
	}
}

// DrawPlayerWithLabel draws a player with text label using Gio layout.
// If selected is true, draws a highlight ring around the player.
// If hasBall is true, draws a ball indicator near the player.
func DrawPlayerWithLabel(gtx layout.Context, th *material.Theme, vp *court.Viewport, player *model.Player, selected bool, hasBall bool) {
	center := vp.RelToPixel(player.Position)
	col := model.RoleColor(player.Role)

	// Selection highlight: larger ring.
	if selected {
		court.DrawCircleOutline(gtx.Ops, center, playerRadius+4, 2.5, highlightColor)
	}

	// filled circle
	court.DrawCircleFill(gtx.Ops, center, playerRadius, col)

	// white outline
	court.DrawCircleOutline(gtx.Ops, center, playerRadius, playerOutlineWidth,
		color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})

	// Direction arrow.
	drawDirectionArrow(gtx.Ops, center, player.Rotation, 0xff)

	// label text — translate default role labels, keep custom ones
	label := player.Label
	if label == "" || label == model.RoleLabel(player.Role) {
		label = roleLabelI18n(player.Role)
	}

	// render label centered on the player circle
	lbl := material.Label(th, unit.Sp(11), label)
	lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	lbl.Alignment = text.Middle

	// measure and position the label
	macro := op.Record(gtx.Ops)
	labelGtx := gtx
	labelGtx.Constraints = layout.Constraints{
		Max: image.Pt(playerRadius*3, playerRadius*2),
	}
	dims := lbl.Layout(labelGtx)
	call := macro.Stop()

	// offset to center the label on the player
	offsetX := int(center.X) - dims.Size.X/2
	offsetY := int(center.Y) - dims.Size.Y/2

	clipStack := clip.Rect{
		Min: image.Pt(offsetX, offsetY),
		Max: image.Pt(offsetX+dims.Size.X, offsetY+dims.Size.Y),
	}.Push(gtx.Ops)
	transStack := op.Offset(image.Pt(offsetX, offsetY)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	transStack.Pop()
	clipStack.Pop()

	// Ball indicator.
	if hasBall {
		DrawBall(gtx.Ops, center)
	}

	// queue: smaller grey circles behind
	if player.Type == "queue" && player.Count > 1 {
		drawQueue(gtx.Ops, vp, player)
	}
}

// DrawCallout draws a speech bubble with the player's callout text above the player.
func DrawCallout(gtx layout.Context, th *material.Theme, vp *court.Viewport, player *model.Player) {
	if player.Callout == "" {
		return
	}
	drawCalloutAt(gtx, th, vp, player, 0xff)
}

// DrawCalloutWithOpacity draws a speech bubble with the given opacity.
func DrawCalloutWithOpacity(gtx layout.Context, th *material.Theme, vp *court.Viewport, player *model.Player, opacity float64) {
	if player.Callout == "" || opacity <= 0 {
		return
	}
	drawCalloutAt(gtx, th, vp, player, uint8(opacity*255))
}

func drawCalloutAt(gtx layout.Context, th *material.Theme, vp *court.Viewport, player *model.Player, alpha uint8) {
	label := i18n.T("callout." + string(player.Callout))
	center := vp.RelToPixel(player.Position)

	// Measure text.
	lbl := material.Label(th, unit.Sp(9), label)
	lbl.Color = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: alpha}
	lbl.Alignment = text.Middle

	macro := op.Record(gtx.Ops)
	labelGtx := gtx
	labelGtx.Constraints = layout.Constraints{
		Max: image.Pt(80, 20),
	}
	dims := lbl.Layout(labelGtx)
	call := macro.Stop()

	padX := 4
	padY := 2
	bubbleW := dims.Size.X + padX*2
	bubbleH := dims.Size.Y + padY*2

	// Position: centered above the player circle.
	bubbleX := int(center.X) - bubbleW/2
	bubbleY := int(center.Y) - playerRadius - 6 - bubbleH

	// Draw rounded background.
	bgCol := color.NRGBA{R: 0x20, G: 0x20, B: 0x20, A: uint8(float64(alpha) * 0.85)}
	bgRect := clip.RRect{
		Rect: image.Rect(bubbleX, bubbleY, bubbleX+bubbleW, bubbleY+bubbleH),
		NE:   3, NW: 3, SE: 3, SW: 3,
	}
	bgStack := bgRect.Push(gtx.Ops)
	paint.Fill(gtx.Ops, bgCol)
	bgStack.Pop()

	// Draw text.
	textX := bubbleX + padX
	textY := bubbleY + padY
	clipStack := clip.Rect{
		Min: image.Pt(textX, textY),
		Max: image.Pt(textX+dims.Size.X, textY+dims.Size.Y),
	}.Push(gtx.Ops)
	transStack := op.Offset(image.Pt(textX, textY)).Push(gtx.Ops)
	call.Add(gtx.Ops)
	transStack.Pop()
	clipStack.Pop()
}

func drawQueue(ops *op.Ops, vp *court.Viewport, player *model.Player) {
	center := vp.RelToPixel(player.Position)
	count := player.Count
	if count > 4 {
		count = 4
	}
	col := model.RoleColor(player.Role)
	// Lighten for queue circles (less prominent than the main player).
	col.A = 0xaa
	for i := 1; i < count; i++ {
		offset := float32(i) * queueSpacing
		qCenter := f32.Point{X: center.X, Y: center.Y + offset}
		court.DrawCircleFill(ops, qCenter, queueRadius, col)
		court.DrawCircleOutline(ops, qCenter, queueRadius, 1,
			color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xaa})
	}
}
