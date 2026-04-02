package pdf

import (
	"bytes"
	"fmt"
	"image/png"

	"github.com/go-pdf/fpdf"

	"github.com/darkweaver87/courtdraw/internal/anim"
	"github.com/darkweaver87/courtdraw/internal/court"
	"github.com/darkweaver87/courtdraw/internal/model"
)

// courtDiagramDPI controls the rendering resolution for court diagrams.
// Higher values produce sharper images at the cost of larger PDF files.
const courtDiagramDPI = 300

// drawCourtDiagram renders a court diagram using the shared image renderer
// and inserts it into the PDF at the given position.
func drawCourtDiagram(pdf *fpdf.Fpdf, x, y, w, h float64, ex *model.Exercise, seqIdx int) {
	// Convert mm dimensions to pixels at the target DPI.
	pixW := int(w * courtDiagramDPI / 25.4)
	pixH := int(h * courtDiagramDPI / 25.4)
	if pixW < 1 {
		pixW = 1
	}
	if pixH < 1 {
		pixH = 1
	}

	img := court.RenderSequence(ex, seqIdx, pixW, pixH, adaptStepPlayers, adaptFinalBallState)

	// Encode to PNG in memory.
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return
	}

	// Register and insert the image.
	name := fmt.Sprintf("court_%p_%d", ex, seqIdx)
	reader := bytes.NewReader(buf.Bytes())
	opts := fpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}
	info := pdf.RegisterImageOptionsReader(name, opts, reader)
	if info == nil {
		return
	}
	pdf.ImageOptions(name, x, y, w, h, false, opts, 0, "")
}

// adaptStepPlayers adapts anim.ComputeStepPositions to the court.StepPlayersFunc signature.
func adaptStepPlayers(seq *model.Sequence, maxStep, step int) []model.Player {
	if maxStep <= 0 {
		return seq.Players
	}
	t := float64(step-1) / float64(maxStep)
	positions := anim.ComputeStepPositions(seq, maxStep, t)
	players := make([]model.Player, len(seq.Players))
	for i := range seq.Players {
		players[i] = seq.Players[i]
		if pos, ok := positions[players[i].ID]; ok {
			players[i].Position = pos
		}
	}
	return players
}

// adaptFinalBallState adapts anim.ComputeFinalBallState to the court.FinalBallStateFunc signature.
func adaptFinalBallState(seq *model.Sequence) []court.BallState {
	animStates := anim.ComputeFinalBallState(seq)
	states := make([]court.BallState, len(animStates))
	for i, s := range animStates {
		states[i] = court.BallState{
			CarrierID: s.CarrierID,
			ShotPos:   s.ShotPos,
			IsShot:    s.IsShot,
		}
	}
	return states
}
