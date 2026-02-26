package court

import (
	"gioui.org/op"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// NBA court dimensions in meters.
func NBAGeometry() *CourtGeometry {
	return &CourtGeometry{
		Width:                15.24,
		Length:               28.65,
		BasketOffset:         1.6002,
		LaneWidth:            4.877,
		LaneLength:           5.791,
		ThreePointRadius:     7.24,
		ThreePointCornerDist: 0.914,
		FreeThrowRadius:      1.829,
		CenterCircleRadius:   1.829,
		RestrictedAreaRadius: 1.219,
		BackboardWidth:       1.829,
		RimDiameter:          0.457,
		LineWidth:            0.0508,
	}
}

// DrawNBACourt draws an NBA basketball court.
func DrawNBACourt(ops *op.Ops, courtType model.CourtType, vp *Viewport, geom *CourtGeometry) {
	drawCourt(ops, courtType, vp, geom)
}
