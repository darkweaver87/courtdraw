package widget

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/darkweaver87/courtdraw/internal/court"
	"github.com/darkweaver87/courtdraw/internal/model"
)

const (
	playerRadius       = 14
	playerOutlineWidth = 2
	queueRadius        = 8
	queueSpacing       = 20
)

var highlightColor = color.NRGBA{R: 0xff, G: 0xff, B: 0x00, A: 0xcc} // yellow

const (
	ballRadius      = 6
	ballOutlineWidth = 1.5
	ballOffsetX     = 8
	ballOffsetY     = 8
)

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

	// label text
	label := player.Label
	if label == "" {
		label = model.RoleLabel(player.Role)
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

	// label text
	label := player.Label
	if label == "" {
		label = model.RoleLabel(player.Role)
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

func drawQueue(ops *op.Ops, vp *court.Viewport, player *model.Player) {
	center := vp.RelToPixel(player.Position)
	count := player.Count
	if count > 4 {
		count = 4
	}
	for i := 1; i < count; i++ {
		offset := float32(i) * queueSpacing
		qCenter := f32.Point{X: center.X, Y: center.Y + offset}
		court.DrawCircleFill(ops, qCenter, queueRadius, model.ColorNeutral)
		court.DrawCircleOutline(ops, qCenter, queueRadius, 1,
			color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xaa})
	}
}
