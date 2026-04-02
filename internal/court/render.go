package court

import (
	"image"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"

	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
)

var (
	parsedFont     *opentype.Font
	parsedFontOnce sync.Once
	faceCacheMu    sync.Mutex
	faceCache      = map[float64]font.Face{}
)

// baseFontSize is chosen so that a 1–2 character label fills the player
// head circle (HeadRadius = 14px → 28px diameter).
const baseFontSize = 18

// ScaledFace returns a font.Face at the given zoom level, caching results.
// The font is Go Regular (goregular.TTF).
func ScaledFace(zoom float64) font.Face {
	parsedFontOnce.Do(func() {
		parsedFont, _ = opentype.Parse(goregular.TTF)
	})
	if parsedFont == nil {
		return nil
	}
	size := baseFontSize * zoom
	faceCacheMu.Lock()
	defer faceCacheMu.Unlock()
	if f, ok := faceCache[size]; ok {
		return f
	}
	f, _ := opentype.NewFace(parsedFont, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	faceCache[size] = f
	return f
}

// BallState represents a ball's carrier and position after all steps.
// This mirrors anim.BallState to avoid an import cycle between court and anim.
type BallState struct {
	CarrierID string
	ShotPos   model.Position
	IsShot    bool
}

// StepPlayersFunc computes player positions at a given step.
// Signature: (seq, maxStep, step) → []model.Player with cumulative positions.
type StepPlayersFunc func(seq *model.Sequence, maxStep, step int) []model.Player

// FinalBallStateFunc computes ball states after all steps.
type FinalBallStateFunc func(seq *model.Sequence) []BallState

// RenderSequence renders an exercise sequence to an image at the given size.
// This is the single source of truth for court rendering — used by the screen
// widget, PDF export, and future GIF/MP4 export.
//
// The image includes: court lines, wood texture, apron bands, players (with
// labels, body, head), actions (with correct line styles), accessories, ball
// indicators, step badges, ghost players at initial/intermediate positions,
// and callouts.
//
// It does NOT include: selection highlights, hover effects, pulse animation,
// ghost arrows, waypoint handles, or any interactive overlays.
//
// stepPlayersFn and finalBallStateFn are injected to avoid an import cycle
// between court and anim. Pass anim.ComputeStepPositions-based and
// anim.ComputeFinalBallState-based adapters from the call site.
func RenderSequence(ex *model.Exercise, seqIndex int, width, height int, stepPlayersFn StepPlayersFunc, finalBallStateFn FinalBallStateFunc) *image.RGBA {
	if width <= 0 || height <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	if ex == nil {
		return image.NewRGBA(image.Rect(0, 0, width, height))
	}

	// Select geometry.
	var geom *CourtGeometry
	switch ex.CourtStandard {
	case model.NBA:
		geom = NBAGeometry()
	default:
		geom = FIBAGeometry()
	}

	// Compute viewport — always portrait, no apron for export.
	vp := computeViewportPortrait(ex.CourtType, geom, image.Pt(width, height), 10, true)
	vp.HideApron = true

	// Draw court background.
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	switch ex.CourtStandard {
	case model.NBA:
		DrawNBACourt(img, ex.CourtType, &vp, geom)
	default:
		DrawFIBACourt(img, ex.CourtType, &vp, geom)
	}

	// Bail out if no sequences or out of range.
	if seqIndex < 0 || seqIndex >= len(ex.Sequences) {
		return img
	}

	seq := &ex.Sequences[seqIndex]

	// Font face scaled by element scale.
	es := vp.ElementScale
	if es <= 0 {
		es = 1.0
	}
	face := ScaledFace(es)

	// Draw static sequence elements.
	renderSequenceElements(img, &vp, seq, face, stepPlayersFn, finalBallStateFn)

	return img
}

// renderSequenceElements draws all static elements of a sequence onto img.
func renderSequenceElements(img *image.RGBA, vp *Viewport, seq *model.Sequence, face font.Face, stepPlayersFn StepPlayersFunc, finalBallStateFn FinalBallStateFunc) {
	// Accessories.
	for i := range seq.Accessories {
		DrawAccessory(img, vp, &seq.Accessories[i], false)
	}

	// Actions — draw with cumulative positions so chained actions connect properly.
	maxStep := model.MaxStep(seq)
	for i := range seq.Actions {
		step := seq.Actions[i].EffectiveStep()
		playersAtStep := stepPlayersFn(seq, maxStep, step)
		DrawAction(img, vp, &seq.Actions[i], playersAtStep)
	}

	// Step badges (only when multiple steps exist).
	if maxStep > 1 {
		for i := range seq.Actions {
			step := seq.Actions[i].EffectiveStep()
			DrawStepBadge(img, vp, &seq.Actions[i], stepPlayersFn(seq, maxStep, step), face)
		}
	}

	// Ghost players at initial and intermediate positions.
	finalPlayers := stepPlayersFn(seq, maxStep, maxStep+1)
	renderGhostPlayers(img, vp, seq, maxStep, finalPlayers, stepPlayersFn)

	// Players and balls at final cumulative positions.
	renderPlayersAndBalls(img, vp, face, seq, finalPlayers, finalBallStateFn)

	// Callouts.
	for i := range finalPlayers {
		if finalPlayers[i].Callout != "" {
			calloutText := i18n.T("callout." + string(finalPlayers[i].Callout))
			DrawCallout(img, vp, &finalPlayers[i], calloutText, face, 0xff)
		}
	}
}

// renderGhostPlayers draws ghost players at initial and intermediate step positions.
func renderGhostPlayers(img *image.RGBA, vp *Viewport, seq *model.Sequence, maxStep int, finalPlayers []model.Player, stepPlayersFn StepPlayersFunc) {
	// Initial position ghosts.
	for i := range seq.Players {
		if seq.Players[i].Position != finalPlayers[i].Position {
			DrawPlayerWithOpacity(img, vp, &seq.Players[i], "", nil, 0.2, false)
		}
	}
	// Intermediate step ghosts.
	if maxStep > 1 {
		for step := 2; step <= maxStep; step++ {
			ghostPlayers := stepPlayersFn(seq, maxStep, step)
			for i := range ghostPlayers {
				if ghostPlayers[i].Position != seq.Players[i].Position &&
					ghostPlayers[i].Position != finalPlayers[i].Position {
					DrawPlayerWithOpacity(img, vp, &ghostPlayers[i], "", nil, 0.2, false)
				}
			}
		}
	}
}

// renderPlayersAndBalls renders players at final positions with balls.
func renderPlayersAndBalls(img *image.RGBA, vp *Viewport, face font.Face, seq *model.Sequence, finalPlayers []model.Player, finalBallStateFn FinalBallStateFunc) {
	ballStates := finalBallStateFn(seq)
	finalCarriers := make([]string, 0)
	for _, bs := range ballStates {
		if !bs.IsShot && bs.CarrierID != "" {
			finalCarriers = append(finalCarriers, bs.CarrierID)
		}
	}
	for i := range finalPlayers {
		hasBall := containsStr(finalCarriers, finalPlayers[i].ID)
		label := resolvePlayerLabel(&finalPlayers[i])
		DrawPlayerWithLabel(img, vp, &finalPlayers[i], label, face, false, hasBall)
	}
	// Ball at basket after shot.
	for _, bs := range ballStates {
		if bs.IsShot {
			ballPx := vp.RelToPixel(bs.ShotPos)
			DrawBallScaled(img, vp, ballPx)
		}
	}
}

// resolvePlayerLabel returns the display label for a player.
func resolvePlayerLabel(p *model.Player) string {
	label := p.Label
	if label == "" || label == model.RoleLabel(p.Role) {
		key := "role." + string(p.Role)
		tr := i18n.T(key)
		if tr != key {
			return tr
		}
		return model.RoleLabel(p.Role)
	}
	return label
}

// containsStr checks if a string slice contains a given string.
func containsStr(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
