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
func DrawPlayerWithLabel(gtx layout.Context, th *material.Theme, vp *court.Viewport, player *model.Player) {
	center := vp.RelToPixel(player.Position)
	col := model.RoleColor(player.Role)

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

	stack := clip.Rect{
		Min: image.Pt(offsetX, offsetY),
		Max: image.Pt(offsetX+dims.Size.X, offsetY+dims.Size.Y),
	}.Push(gtx.Ops)
	op.Offset(image.Pt(offsetX, offsetY)).Add(gtx.Ops)
	call.Add(gtx.Ops)
	stack.Pop()

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
