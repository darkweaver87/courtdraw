package fynecourt

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"slices"
	"sync"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/darkweaver87/courtdraw/internal/anim"
	"github.com/darkweaver87/courtdraw/internal/court"
	"github.com/darkweaver87/courtdraw/internal/i18n"
	"github.com/darkweaver87/courtdraw/internal/model"
	"github.com/darkweaver87/courtdraw/internal/ui/editor"
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

func getScaledFace(zoom float64) font.Face {
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

// CourtWidget renders a basketball court with exercise elements.
type CourtWidget struct {
	widget.BaseWidget

	exercise    *model.Exercise
	seqIndex    int
	editorState *editor.EditorState
	viewport    court.Viewport
	baseVP      court.Viewport // viewport before zoom/pan
	geom        *court.CourtGeometry

	playback *anim.Playback
	animMode bool

	raster *canvas.Raster

	// Pixel dimensions from last draw.
	pixelW, pixelH int

	// Cached court background (lines only, no elements).
	courtBg        *image.RGBA
	courtBgW       int
	courtBgH       int
	courtBgType    model.CourtType
	courtBgStd     model.CourtStandard
	courtBgZoom    float64
	courtBgPanX    float64
	courtBgPanY    float64

	// Zoom and pan state.
	zoomLevel   float64 // 1.0 = normal, max 5.0
	panX, panY  float64 // offset in relative coordinates
	dragPanning bool    // true when panning instead of dragging an element

	// Mobile tap fix: track whether handlePress was called for current gesture.
	pressHandled bool

	// Pointer state.
	pressed          bool
	dragPlayerIdx    int
	dragAccIdx       int
	dragActionEndIdx  int // action whose To endpoint is being dragged (-1 = none)
	dragWaypointIdx   int // waypoint being dragged (-1 = endpoint, -2 = creating new)
	dragPlayerStep    int // step of the ghost being dragged (0 = base position, -1 = final/current)
	selectedStep      int // step of the selected ghost (-1 = final, 0 = base, >0 = intermediate)
	dragActive        bool
	dragRotating     bool

	// Zoom indicator overlay.
	zoomLabel *canvas.Text

	// Animation ticker.
	animStop chan struct{}
	animMu   sync.Mutex
	readOnly bool // true when editing is blocked (animation mode, training mode)

	// Selection pulse ticker — runs at ~15fps when an element is selected.
	pulseStop chan struct{}
	pulseMu   sync.Mutex

	OnChanged func()
}

// Ensure interface compliance.
var _ fyne.Tappable = (*CourtWidget)(nil)
var _ fyne.Draggable = (*CourtWidget)(nil)
var _ fyne.Scrollable = (*CourtWidget)(nil)
var _ desktop.Mouseable = (*CourtWidget)(nil)
var _ desktop.Hoverable = (*CourtWidget)(nil)
var _ fyne.Focusable = (*CourtWidget)(nil)

// NewCourtWidget creates a new court widget.
func NewCourtWidget() *CourtWidget {
	w := &CourtWidget{
		dragPlayerIdx:    -1,
		dragAccIdx:       -1,
		dragActionEndIdx: -1,
		zoomLevel:        1.0,
	}
	w.ExtendBaseWidget(w)
	w.raster = canvas.NewRaster(w.draw)

	w.zoomLabel = canvas.NewText("", color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xcc})
	w.zoomLabel.TextSize = 14
	w.zoomLabel.TextStyle.Bold = true

	return w
}

// SetExercise sets the exercise to display.
func (w *CourtWidget) SetExercise(ex *model.Exercise) {
	w.exercise = ex
	w.seqIndex = 0
	w.courtBg = nil // invalidate court background cache
	w.zoomLevel = 1.0
	w.panX = 0
	w.panY = 0
	w.updateZoomLabel()
	w.updateGeometry()
	w.Refresh()
}

// SetSequence changes which sequence is displayed.
func (w *CourtWidget) SetSequence(index int) {
	if w.exercise != nil && index >= 0 && index < len(w.exercise.Sequences) {
		w.seqIndex = index
		w.Refresh()
	}
}

// SeqIndex returns the current sequence index.
func (w *CourtWidget) SeqIndex() int {
	return w.seqIndex
}

// Viewport returns a pointer to the current viewport.
func (w *CourtWidget) Viewport() *court.Viewport {
	return &w.viewport
}

// SetEditorState sets the editor state for interactive editing.
func (w *CourtWidget) SetEditorState(state *editor.EditorState) {
	w.editorState = state
}

// SetReadOnly blocks or allows editing on the court (used in animation/training modes).
func (w *CourtWidget) SetReadOnly(on bool) {
	w.readOnly = on
}

// SetPlayback sets the playback engine for animation.
func (w *CourtWidget) SetPlayback(pb *anim.Playback) {
	w.playback = pb
}

// SetAnimMode enables or disables animation mode.
func (w *CourtWidget) SetAnimMode(on bool) {
	w.animMu.Lock()
	defer w.animMu.Unlock()

	if on && !w.animMode {
		w.animMode = true
		w.animStop = make(chan struct{})
		go w.animLoop(w.animStop)
	} else if !on && w.animMode {
		w.animMode = false
		close(w.animStop)
	}
}

func (w *CourtWidget) animLoop(stop chan struct{}) {
	ticker := time.NewTicker(time.Second / 60)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			fyne.Do(func() {
				w.Refresh()
			})
		}
	}
}

// startPulseTicker starts a ~15fps refresh ticker for the selection pulse animation.
func (w *CourtWidget) startPulseTicker() {
	w.pulseMu.Lock()
	defer w.pulseMu.Unlock()
	if w.pulseStop != nil {
		return // already running
	}
	w.pulseStop = make(chan struct{})
	go func(stop chan struct{}) {
		ticker := time.NewTicker(time.Second / 15)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				fyne.Do(func() {
					w.Refresh()
				})
			}
		}
	}(w.pulseStop)
}

// stopPulseTicker stops the selection pulse ticker.
func (w *CourtWidget) stopPulseTicker() {
	w.pulseMu.Lock()
	defer w.pulseMu.Unlock()
	if w.pulseStop != nil {
		close(w.pulseStop)
		w.pulseStop = nil
	}
}

func (w *CourtWidget) updateGeometry() {
	if w.exercise == nil {
		return
	}
	switch w.exercise.CourtStandard {
	case model.NBA:
		w.geom = court.NBAGeometry()
	default:
		w.geom = court.FIBAGeometry()
	}
}

func (w *CourtWidget) currentSequence() *model.Sequence {
	if w.exercise == nil || w.seqIndex >= len(w.exercise.Sequences) {
		return nil
	}
	return &w.exercise.Sequences[w.seqIndex]
}

// draw is the raster generator function.
func (w *CourtWidget) draw(pixW, pixH int) image.Image {
	w.pixelW = pixW
	w.pixelH = pixH

	if w.exercise == nil || w.geom == nil {
		return image.NewRGBA(image.Rect(0, 0, pixW, pixH))
	}

	w.updateGeometry()
	w.baseVP = court.ComputeViewport(
		w.exercise.CourtType,
		w.geom,
		image.Pt(pixW, pixH),
		10,
	)
	w.viewport = w.applyZoomPan(w.baseVP)

	// Build or reuse cached court background.
	w.ensureCourtBg(pixW, pixH, &w.viewport)

	// Fresh frame buffer, copy background.
	img := image.NewRGBA(image.Rect(0, 0, pixW, pixH))
	copy(img.Pix, w.courtBg.Pix)

	// Scale font with both element scale and zoom so labels fill the head.
	es := w.viewport.ElementScale
	if es <= 0 {
		es = 1.0
	}
	face := getScaledFace(es * w.zoomLevel)

	// Animation mode: call Update() at render time for smooth interpolation.
	if w.animMode && w.playback != nil && w.playback.State() == anim.StatePlaying {
		frame, _ := w.playback.Update()
		w.seqIndex = w.playback.SeqIndex()
		w.drawAnimatedFrame(img, face, &frame)
		return img
	}

	// Static mode: draw current sequence.
	seq := w.currentSequence()
	if seq != nil {
		w.drawSequence(img, face, seq)
	}

	return img
}

// ensureCourtBg caches the court line drawing. Only redraws when size or court type changes.
func (w *CourtWidget) ensureCourtBg(pixW, pixH int, vp *court.Viewport) {
	if w.courtBg != nil && w.courtBgW == pixW && w.courtBgH == pixH &&
		w.courtBgType == w.exercise.CourtType && w.courtBgStd == w.exercise.CourtStandard &&
		w.courtBgZoom == w.zoomLevel && w.courtBgPanX == w.panX && w.courtBgPanY == w.panY {
		return
	}
	w.courtBg = image.NewRGBA(image.Rect(0, 0, pixW, pixH))
	switch w.exercise.CourtStandard {
	case model.NBA:
		court.DrawNBACourt(w.courtBg, w.exercise.CourtType, vp, w.geom)
	default:
		court.DrawFIBACourt(w.courtBg, w.exercise.CourtType, vp, w.geom)
	}
	w.courtBgW = pixW
	w.courtBgH = pixH
	w.courtBgType = w.exercise.CourtType
	w.courtBgStd = w.exercise.CourtStandard
	w.courtBgZoom = w.zoomLevel
	w.courtBgPanX = w.panX
	w.courtBgPanY = w.panY
}

func (w *CourtWidget) drawSequence(img *image.RGBA, face font.Face, seq *model.Sequence) {
	sel := w.editorState
	var selElem *editor.Selection
	var hovElem *editor.Selection
	if sel != nil {
		selElem = sel.SelectedElement
		hovElem = sel.HoveredElement
	}

	// Manage pulse ticker based on selection state.
	if selElem != nil {
		w.startPulseTicker()
	} else {
		w.stopPulseTicker()
	}

	// Accessories.
	for i := range seq.Accessories {
		selected := selElem != nil && selElem.Kind == editor.SelectAccessory && selElem.Index == i && selElem.SeqIndex == w.seqIndex
		court.DrawAccessory(img, &w.viewport, &seq.Accessories[i], selected)
	}

	// Actions — draw with cumulative positions so chained actions connect properly.
	maxStep := model.MaxStep(seq)
	for i := range seq.Actions {
		step := seq.Actions[i].EffectiveStep()
		playersAtStep := stepPlayers(seq, maxStep, step)
		if selElem != nil && selElem.Kind == editor.SelectAction && selElem.Index == i && selElem.SeqIndex == w.seqIndex {
			court.DrawActionHighlight(img, &w.viewport, &seq.Actions[i], playersAtStep)
		}
		court.DrawAction(img, &w.viewport, &seq.Actions[i], playersAtStep)
	}
	// Step badges (only when multiple steps exist).
	if maxStep > 1 {
		for i := range seq.Actions {
			step := seq.Actions[i].EffectiveStep()
			court.DrawStepBadge(img, &w.viewport, &seq.Actions[i], stepPlayers(seq, maxStep, step), face)
		}
	}

	// Ghost players at initial and intermediate positions.
	finalPlayers := stepPlayers(seq, maxStep, maxStep+1)
	w.drawGhostPlayers(img, seq, maxStep, finalPlayers, selElem)

	// Players and balls at final cumulative positions.
	w.drawPlayersAndBalls(img, face, seq, finalPlayers, selElem)

	// Callouts.
	for i := range finalPlayers {
		if finalPlayers[i].Callout != "" {
			calloutText := i18n.T("callout." + string(finalPlayers[i].Callout))
			court.DrawCallout(img, &w.viewport, &finalPlayers[i], calloutText, face, 0xff)
		}
	}

	// Interactive overlays: hover, ghost arrow, action-from indicator, rotation handle.
	w.drawHoverHighlights(img, seq, sel, selElem, hovElem)
	w.drawActionPreview(img, seq, sel)
	w.drawSelectionOverlays(img, seq, selElem)
}

// drawGhostPlayers draws ghost players at initial and intermediate step positions.
func (w *CourtWidget) drawGhostPlayers(img *image.RGBA, seq *model.Sequence, maxStep int, finalPlayers []model.Player, selElem *editor.Selection) {
	// Initial position ghosts.
	for i := range seq.Players {
		if seq.Players[i].Position != finalPlayers[i].Position {
			isGhostSel := selElem != nil && selElem.Kind == editor.SelectPlayer && selElem.Index == i && selElem.SeqIndex == w.seqIndex && w.selectedStep == 0
			if isGhostSel {
				court.DrawPlayerWithLabel(img, &w.viewport, &seq.Players[i], "", nil, true, false)
			} else {
				court.DrawPlayerWithOpacity(img, &w.viewport, &seq.Players[i], "", nil, 0.2, false)
			}
		}
	}
	// Intermediate step ghosts.
	if maxStep > 1 {
		for step := 2; step <= maxStep; step++ {
			ghostPlayers := stepPlayers(seq, maxStep, step)
			for i := range ghostPlayers {
				if ghostPlayers[i].Position != seq.Players[i].Position &&
					ghostPlayers[i].Position != finalPlayers[i].Position {
					isGhostSel := selElem != nil && selElem.Kind == editor.SelectPlayer && selElem.Index == i && selElem.SeqIndex == w.seqIndex && w.selectedStep == step-1
					if isGhostSel {
						court.DrawPlayerWithLabel(img, &w.viewport, &ghostPlayers[i], "", nil, true, false)
					} else {
						court.DrawPlayerWithOpacity(img, &w.viewport, &ghostPlayers[i], "", nil, 0.2, false)
					}
				}
			}
		}
	}
}

// drawPlayersAndBalls renders players at final positions with balls (accounting for passes and shots).
func (w *CourtWidget) drawPlayersAndBalls(img *image.RGBA, face font.Face, seq *model.Sequence, finalPlayers []model.Player, selElem *editor.Selection) {
	ballStates := anim.ComputeFinalBallState(seq)
	finalCarriers := make([]string, 0)
	for _, bs := range ballStates {
		if !bs.IsShot && bs.CarrierID != "" {
			finalCarriers = append(finalCarriers, bs.CarrierID)
		}
	}
	for i := range finalPlayers {
		isSelected := selElem != nil && selElem.Kind == editor.SelectPlayer && selElem.Index == i && selElem.SeqIndex == w.seqIndex
		// Selection ring on final player only if selectedStep is final (-1).
		selected := isSelected && w.selectedStep == -1
		hasBall := slices.Contains(finalCarriers, finalPlayers[i].ID)
		label := resolvePlayerLabel(&finalPlayers[i])
		court.DrawPlayerWithLabel(img, &w.viewport, &finalPlayers[i], label, face, selected, hasBall)
	}
	// Ball at basket after shot.
	for _, bs := range ballStates {
		if bs.IsShot {
			ballPx := w.viewport.RelToPixel(bs.ShotPos)
			court.DrawBallScaled(img, &w.viewport, ballPx)
		}
	}
}

// drawHoverHighlights renders hover feedback for the element under the cursor.
func (w *CourtWidget) drawHoverHighlights(img *image.RGBA, seq *model.Sequence, sel *editor.EditorState, selElem, hovElem *editor.Selection) {
	if hovElem == nil || hovElem.SeqIndex != w.seqIndex {
		return
	}
	if selElem != nil && *hovElem == *selElem {
		return
	}

	switch hovElem.Kind {
	case editor.SelectAccessory:
		if hovElem.Index < len(seq.Accessories) {
			center := w.viewport.RelToPixel(seq.Accessories[hovElem.Index].Position)
			court.DrawAccessoryHoverHighlight(img, &w.viewport, center)
		}
	case editor.SelectPlayer:
		if hovElem.Index < len(seq.Players) {
			center := w.viewport.RelToPixel(seq.Players[hovElem.Index].Position)
			if sel != nil && sel.ActiveTool == editor.ToolAction && sel.ActionFrom != nil {
				court.DrawActionTargetHighlight(img, &w.viewport, center)
			} else {
				court.DrawHoverHighlight(img, &w.viewport, center)
			}
		}
	}
}

// drawActionPreview renders the action-from indicator and ghost arrow with magnetic snap.
func (w *CourtWidget) drawActionPreview(img *image.RGBA, seq *model.Sequence, sel *editor.EditorState) {
	if sel == nil || sel.ActionFrom == nil {
		return
	}

	// Use cumulative positions so the preview starts from the player's current step position.
	maxStep := model.MaxStep(seq)
	positions := anim.ComputeStepPositions(seq, maxStep, 1.0)

	// Action-from indicator ring at cumulative position.
	var fromPos court.Point
	if pos, ok := positions[*sel.ActionFrom]; ok {
		fromPos = w.viewport.RelToPixel(pos)
	}
	court.DrawCircleOutline(img, fromPos, court.PlayerRadius+6, 2, court.ActionLineColor)

	// For passes, show subtle highlight on all valid target players.
	if sel.ToolActionType == model.ActionPass {
		for i := range seq.Players {
			if seq.Players[i].ID == *sel.ActionFrom {
				continue
			}
			targetPos := positions[seq.Players[i].ID]
			center := w.viewport.RelToPixel(targetPos)
			court.DrawHoverHighlight(img, &w.viewport, center)
		}
	}

	// For shots, always show ghost arrow to basket with target circle.
	if model.IsShot(sel.ToolActionType) && w.geom != nil {
		basketPos := court.BasketRelativePosition(w.geom, w.exercise.CourtType)
		basketPx := w.viewport.RelToPixel(basketPos)
		court.DrawActionTargetHighlight(img, &w.viewport, basketPx)
		court.DrawActionPreview(img, &w.viewport, fromPos, basketPx, sel.ToolActionType, 0.5)
		return
	}

	if sel.PreviewMousePos == nil {
		return
	}
	toPos := *sel.PreviewMousePos

	// Magnetic snap: find nearest player within 30dp.
	toPos = w.magneticSnap(img, seq, sel, toPos)

	court.DrawActionPreview(img, &w.viewport, fromPos, toPos, sel.ToolActionType, 0.5)
}

// magneticSnap finds the nearest player within 30dp and snaps the arrow endpoint to it.
func (w *CourtWidget) magneticSnap(img *image.RGBA, seq *model.Sequence, sel *editor.EditorState, pos court.Point) court.Point {
	snapDist := w.viewport.Sd(court.MagneticSnapDist)
	snapIdx := -1
	bestDist := snapDist
	for i := range seq.Players {
		if seq.Players[i].ID == *sel.ActionFrom {
			continue
		}
		center := w.viewport.RelToPixel(seq.Players[i].Position)
		dx := float64(pos.X - center.X)
		dy := float64(pos.Y - center.Y)
		d := math.Sqrt(dx*dx + dy*dy)
		if d < bestDist {
			bestDist = d
			snapIdx = i
		}
	}
	if snapIdx >= 0 {
		snapped := w.viewport.RelToPixel(seq.Players[snapIdx].Position)
		court.DrawActionTargetHighlight(img, &w.viewport, snapped)
		return snapped
	}
	return pos
}

// drawSelectionOverlays renders the rotation handle for the selected element.
func (w *CourtWidget) drawSelectionOverlays(img *image.RGBA, seq *model.Sequence, selElem *editor.Selection) {
	if selElem == nil || selElem.SeqIndex != w.seqIndex {
		return
	}
	switch selElem.Kind {
	case editor.SelectPlayer:
		if selElem.Index < len(seq.Players) {
			p := &seq.Players[selElem.Index]
			center := w.viewport.RelToPixel(p.Position)
			court.DrawRotationHandle(img, &w.viewport, center, p.Rotation)
		}
	case editor.SelectAccessory:
		if selElem.Index < len(seq.Accessories) {
			a := &seq.Accessories[selElem.Index]
			center := w.viewport.RelToPixel(a.Position)
			court.DrawRotationHandle(img, &w.viewport, center, a.Rotation)
		}
	case editor.SelectAction:
		// Show basket target highlight when dragging a shot near the basket.
		if selElem.Index < len(seq.Actions) && w.geom != nil {
			act := &seq.Actions[selElem.Index]
			if model.IsShot(act.Type) {
				basketPos := court.BasketRelativePosition(w.geom, w.exercise.CourtType)
				basketPx := w.viewport.RelToPixel(basketPos)
				toPx := w.viewport.RelToPixel(act.To.Position)
				dx := float64(toPx.X - basketPx.X)
				dy := float64(toPx.Y - basketPx.Y)
				if math.Sqrt(dx*dx+dy*dy) < 2 {
					court.DrawActionTargetHighlight(img, &w.viewport, basketPx)
				}
			}
			// Draw waypoint handles (blue circles) on selected action (edit mode only).
			if !w.readOnly {
				wpRadius := w.viewport.S(8)
				for _, wp := range act.Waypoints {
					center := w.viewport.RelToPixel(wp)
					court.DrawCircleFill(img, center, wpRadius, color.NRGBA{R: 0x29, G: 0x6d, B: 0xd4, A: 0xff})
					court.DrawCircleOutline(img, center, wpRadius, w.viewport.S(2), color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
				}
			}
			// If no waypoints, show a draggable midpoint handle (edit mode only).
			if !w.readOnly && len(act.Waypoints) == 0 {
				maxStep := model.MaxStep(seq)
				playersAtStep := stepPlayers(seq, maxStep, act.EffectiveStep())
				from := court.ResolveRef(&w.viewport, act.From, playersAtStep)
				to := court.ResolveRef(&w.viewport, act.To, playersAtStep)
				mid := court.Pt((from.X+to.X)/2, (from.Y+to.Y)/2)
				court.DrawCircleFill(img, mid, w.viewport.S(6), color.NRGBA{R: 0x29, G: 0x6d, B: 0xd4, A: 0xaa})
				court.DrawCircleOutline(img, mid, w.viewport.S(6), w.viewport.S(1.5), color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xaa})
			}
		}
	}
}

func (w *CourtWidget) drawAnimatedFrame(img *image.RGBA, face font.Face, frame *anim.AnimatedFrame) {
	vp := &w.viewport
	// Accessories.
	for i := range frame.Accessories {
		court.DrawAccessoryWithOpacity(img, vp, &frame.Accessories[i].Accessory, frame.Accessories[i].Opacity)
	}

	// Actions with progress — use step-aware positions so arrows stay anchored.
	seq := w.currentSequence()
	maxStep := 0
	if seq != nil {
		maxStep = model.MaxStep(seq)
	}
	for i := range frame.Actions {
		act := &frame.Actions[i]
		var actionPlayers []model.Player
		if seq != nil && maxStep > 0 {
			actionPlayers = stepPlayers(seq, maxStep, act.EffectiveStep())
		} else {
			actionPlayers = make([]model.Player, len(frame.Players))
			for j := range frame.Players {
				actionPlayers[j] = frame.Players[j].Player
			}
		}
		court.DrawActionWithProgress(img, vp, &act.Action, actionPlayers, act.Progress)
	}

	// Players with opacity.
	for i := range frame.Players {
		label := resolvePlayerLabel(&frame.Players[i].Player)
		court.DrawPlayerWithOpacity(img, vp, &frame.Players[i].Player, label, face, frame.Players[i].Opacity, false)
	}

	// Callouts.
	for i := range frame.Players {
		if frame.Players[i].Callout != "" {
			calloutText := i18n.T("callout." + string(frame.Players[i].Callout))
			alpha := uint8(frame.Players[i].Opacity * 255)
			court.DrawCallout(img, vp, &frame.Players[i].Player, calloutText, face, alpha)
		}
	}

	// Balls — position at carrier's hand (or in-flight position).
	playerByID := make(map[string]*model.Player, len(frame.Players))
	for i := range frame.Players {
		playerByID[frame.Players[i].ID] = &frame.Players[i].Player
	}
	for _, b := range frame.Balls {
		if b.Opacity > 0 {
			ballPixel := vp.RelToPixel(b.Pos)
			// Apply hand offset only when ball is in a player's hands (not in flight).
			if !b.InFlight {
				if p, ok := playerByID[b.CarrierID]; ok {
					tmp := *p
					tmp.Position = b.Pos
					ballPixel = court.BallPosForPlayer(vp, ballPixel, &tmp)
				}
			}
			court.DrawBallWithOpacity(img, vp, ballPixel, b.Opacity)
		}
	}
}

func actionLabel(at model.ActionType) string {
	switch model.NormalizeActionType(at) {
	case model.ActionPass:
		return i18n.T(i18n.KeyToolActionPass)
	case model.ActionDribble:
		return i18n.T(i18n.KeyToolActionDribble)
	case model.ActionCut:
		return i18n.T(i18n.KeyToolActionCut)
	case model.ActionScreen:
		return i18n.T(i18n.KeyToolActionScreen)
	case model.ActionShot:
		return i18n.T(i18n.KeyToolActionShot)
	case model.ActionHandoff:
		return i18n.T(i18n.KeyToolActionHandoff)
	default:
		return string(at)
	}
}

func accessoryLabel(at model.AccessoryType) string {
	switch at {
	case model.AccessoryCone:
		return i18n.T(i18n.KeyToolAccessoryCone)
	case model.AccessoryAgilityLadder:
		return i18n.T(i18n.KeyToolAccessoryLadder)
	case model.AccessoryChair:
		return i18n.T(i18n.KeyToolAccessoryChair)
	default:
		return string(at)
	}
}

// stepPlayers returns a copy of players with cumulative positions at the START of the given step.
// This ensures that actions at step N draw from positions after steps 1..N-1 completed.
func stepPlayers(seq *model.Sequence, maxStep, step int) []model.Player {
	if maxStep <= 0 {
		return seq.Players
	}
	// Compute positions with all steps before this one fully completed (progress=1)
	// and this step at progress=0.
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

// dpToPixel converts a Fyne dp position to pixel coordinates.
func (w *CourtWidget) dpToPixel(dp fyne.Position) court.Point {
	size := w.Size()
	if size.Width <= 0 || size.Height <= 0 || w.pixelW <= 0 {
		return court.Point{}
	}
	scaleX := float32(w.pixelW) / size.Width
	scaleY := float32(w.pixelH) / size.Height
	return court.Pt(dp.X*scaleX, dp.Y*scaleY)
}

// --- Fyne interfaces ---

func (w *CourtWidget) CreateRenderer() fyne.WidgetRenderer {
	return &courtRenderer{widget: w}
}

func (w *CourtWidget) MinSize() fyne.Size {
	return fyne.NewSize(200, 200)
}

// Tapped handles mobile taps and simple clicks.
func (w *CourtWidget) Tapped(e *fyne.PointEvent) {
	if w.pressHandled {
		w.pressHandled = false // consumed, reset for next gesture
		return                 // already handled by MouseDown
	}
	pos := w.dpToPixel(e.Position)
	w.handlePress(pos)
}

// Dragged handles drag gestures for element movement and panning.
func (w *CourtWidget) Dragged(e *fyne.DragEvent) {
	pos := w.dpToPixel(e.Position)
	// On mobile, Tapped may not fire if the finger moves slightly.
	// Treat the first Dragged call as a press to ensure element creation/selection works.
	if !w.pressHandled {
		w.pressHandled = true
		w.handlePress(pos)
	}
	if w.dragPanning && w.zoomLevel > 1.01 {
		w.handlePanDrag(e.Dragged)
	} else {
		w.handleDrag(pos)
	}
}

func (w *CourtWidget) DragEnd() {
	w.pressHandled = false
	w.handleRelease()
}

// MouseDown handles desktop press events (more precise than Tapped).
func (w *CourtWidget) MouseDown(e *desktop.MouseEvent) {
	if e.Button == desktop.MouseButtonSecondary {
		// Right-click: cancel action chain (keep current tool).
		if w.editorState != nil && w.editorState.ActionFrom != nil {
			w.editorState.ActionFrom = nil
			w.Refresh()
			w.notifyChanged()
		}
		return
	}
	if e.Button != desktop.MouseButtonPrimary {
		return
	}
	pos := w.dpToPixel(e.Position)
	w.pressed = true
	w.pressHandled = true // prevent Tapped from firing after MouseUp
	w.handlePress(pos)
}

func (w *CourtWidget) MouseUp(e *desktop.MouseEvent) {
	if w.pressed {
		w.pressed = false
		// Do NOT reset pressHandled here — Tapped fires AFTER MouseUp on desktop.
		// pressHandled will be reset by DragEnd or next MouseDown.
		w.handleRelease()
	}
}

// FocusGained implements fyne.Focusable for keyboard event support.
func (w *CourtWidget) FocusGained()   {}
func (w *CourtWidget) FocusLost()     {}
func (w *CourtWidget) TypedRune(r rune) {}
func (w *CourtWidget) TypedKey(e *fyne.KeyEvent) {
	if e.Name == fyne.KeyDelete || e.Name == fyne.KeyBackspace {
		w.DeleteSelected()
	}
}

// --- Hover (desktop only) ---

// MouseIn implements desktop.Hoverable.
func (w *CourtWidget) MouseIn(_ *desktop.MouseEvent) {}

// MouseMoved implements desktop.Hoverable.
func (w *CourtWidget) MouseMoved(e *desktop.MouseEvent) {
	if w.readOnly || w.animMode {
		return
	}
	state := w.editorState
	if state == nil {
		return
	}
	seq := w.currentSequence()
	if seq == nil {
		return
	}

	pos := w.dpToPixel(e.Position)
	mousePoint := court.Pt(pos.X, pos.Y)

	// Update hover state.
	var hovered *editor.Selection
	if pi := court.HitTestPlayer(&w.viewport, seq, pos); pi >= 0 {
		hovered = &editor.Selection{Kind: editor.SelectPlayer, Index: pi, SeqIndex: w.seqIndex}
	} else if ai := court.HitTestAccessory(&w.viewport, seq, pos); ai >= 0 {
		hovered = &editor.Selection{Kind: editor.SelectAccessory, Index: ai, SeqIndex: w.seqIndex}
	}

	changed := false
	if (hovered == nil) != (state.HoveredElement == nil) {
		changed = true
	} else if hovered != nil && state.HoveredElement != nil && *hovered != *state.HoveredElement {
		changed = true
	}
	state.HoveredElement = hovered

	// Track mouse position for ghost arrow preview.
	if state.ActiveTool == editor.ToolAction && state.ActionFrom != nil {
		state.PreviewMousePos = &mousePoint
		changed = true
	} else if state.PreviewMousePos != nil {
		state.PreviewMousePos = nil
		changed = true
	}

	if changed {
		w.Refresh()
	}
}

// MouseOut implements desktop.Hoverable.
func (w *CourtWidget) MouseOut() {
	if w.editorState != nil {
		w.editorState.HoveredElement = nil
		w.editorState.PreviewMousePos = nil
		w.Refresh()
	}
}

// --- Zoom and pan ---

const (
	zoomMin  = 1.0
	zoomMax  = 5.0
	zoomStep = 0.15
)

// applyZoomPan transforms a base viewport by applying zoom level and pan offset.
func (w *CourtWidget) applyZoomPan(base court.Viewport) court.Viewport {
	if w.zoomLevel <= 1.0 {
		base.Scale = 1.0
		return base
	}
	z := w.zoomLevel
	newW := base.Width * z
	newH := base.Height * z
	// Center of base viewport.
	cx := base.OffsetX + base.Width/2
	cy := base.OffsetY + base.Height/2
	// Apply pan offset (in relative court coordinates → pixel offset).
	return court.Viewport{
		OffsetX:      cx - newW/2 - w.panX*newW,
		OffsetY:      cy - newH/2 - w.panY*newH,
		Width:        newW,
		Height:       newH,
		Scale:        z,
		ElementScale: base.ElementScale,
	}
}

// clampPan constrains pan offsets so the court stays within view.
func (w *CourtWidget) clampPan() {
	if w.zoomLevel <= 1.0 {
		w.panX = 0
		w.panY = 0
		return
	}
	maxPan := (w.zoomLevel - 1) / (2 * w.zoomLevel)
	if w.panX < -maxPan {
		w.panX = -maxPan
	}
	if w.panX > maxPan {
		w.panX = maxPan
	}
	if w.panY < -maxPan {
		w.panY = -maxPan
	}
	if w.panY > maxPan {
		w.panY = maxPan
	}
}

// ResetZoom resets zoom to 1.0 and pan to origin.
func (w *CourtWidget) ResetZoom() {
	w.zoomLevel = 1.0
	w.panX = 0
	w.panY = 0
	w.courtBg = nil
	w.updateZoomLabel()
	w.Refresh()
}

// ZoomLevel returns the current zoom level.
func (w *CourtWidget) ZoomLevel() float64 {
	return w.zoomLevel
}

// ZoomIn increases zoom by one step.
func (w *CourtWidget) ZoomIn() {
	w.zoomLevel = math.Min(w.zoomLevel+zoomStep, zoomMax)
	w.clampPan()
	w.courtBg = nil
	w.updateZoomLabel()
	w.Refresh()
}

// ZoomOut decreases zoom by one step.
func (w *CourtWidget) ZoomOut() {
	w.zoomLevel = math.Max(w.zoomLevel-zoomStep, zoomMin)
	w.clampPan()
	w.courtBg = nil
	w.updateZoomLabel()
	w.Refresh()
}

// SetZoom sets the zoom level directly (clamped to [1.0, 5.0]).
func (w *CourtWidget) SetZoom(level float64) {
	w.zoomLevel = math.Max(zoomMin, math.Min(zoomMax, level))
	w.clampPan()
	w.courtBg = nil
	w.updateZoomLabel()
	w.Refresh()
}

func (w *CourtWidget) updateZoomLabel() {
	if w.zoomLevel > 1.01 {
		w.zoomLabel.Text = fmt.Sprintf("%.1fx", w.zoomLevel)
	} else {
		w.zoomLabel.Text = ""
	}
	w.zoomLabel.Refresh()
}

// Scrolled implements fyne.Scrollable — handles mouse wheel zoom and pinch-to-zoom on mobile.
func (w *CourtWidget) Scrolled(e *fyne.ScrollEvent) {
	if e.Scrolled.DY > 0 {
		w.zoomLevel = math.Min(w.zoomLevel+zoomStep, zoomMax)
	} else if e.Scrolled.DY < 0 {
		w.zoomLevel = math.Max(w.zoomLevel-zoomStep, zoomMin)
	}
	w.clampPan()
	w.courtBg = nil
	w.updateZoomLabel()
	w.Refresh()
}

// --- Interaction handlers ---

func (w *CourtWidget) handlePress(pos court.Point) { //nolint:gocyclo
	// Block edits in read-only mode (animation, training).
	if w.readOnly || w.animMode {
		return
	}
	state := w.editorState
	if state == nil {
		return
	}
	seq := w.currentSequence()
	if seq == nil {
		return
	}

	w.dragActive = false
	w.dragPlayerIdx = -1
	w.dragAccIdx = -1
	w.dragActionEndIdx = -1
	w.dragRotating = false
	w.dragPanning = false

	// Universal hit test: select and drag elements regardless of active tool.
	if state.ActiveTool == editor.ToolAction && state.ActionFrom != nil {
		// In action targeting mode: clicking the source player cancels and selects it.
		finalSeq := w.seqWithFinalPositions(seq)
		if pi := court.HitTestPlayer(&w.viewport, finalSeq, pos); pi >= 0 && seq.Players[pi].ID == *state.ActionFrom {
			state.SetTool(editor.ToolSelect)
			state.Select(editor.SelectPlayer, pi, w.seqIndex)
			w.dragPlayerIdx = pi
			w.dragActive = true
			state.IsDragging = true
			w.Refresh()
			w.notifyChanged()
			return
		}
		// Other elements: fall through to ToolAction handler below.
	} else {
		if w.trySelectElement(seq, state, pos) {
			return
		}
	}

	// No element hit — handle tool-specific creation or panning.
	switch state.ActiveTool {
	case editor.ToolNone, editor.ToolSelect:
		// If zoomed in, start panning.
		if w.zoomLevel > 1.01 {
			w.dragPanning = true
			w.dragActive = true
			return
		}
		state.Deselect()
		w.Refresh()
		w.notifyChanged()

	case editor.ToolPlayer:
		relPos := w.viewport.PixelToRel(pos)
		relPos = court.ClampPosition(&w.viewport,relPos)
		p := model.Player{
			ID:       editor.NextPlayerID(seq),
			Label:    model.RoleLabel(state.ToolRole),
			Role:     state.ToolRole,
			Position: relPos,
		}
		// Half court: basket is at the bottom (south).
		// Attackers face south (180°), defenders face north with back to basket (0°).
		switch state.ToolRole {
		case model.RoleDefender:
			p.Rotation = 0 // facing north (back to basket)
		default:
			p.Rotation = 180 // facing south (toward basket)
		}
		if state.ToolQueue {
			p.Type = "queue"
			p.Count = 3
		}
		seq.Players = append(seq.Players, p)
		idx := len(seq.Players) - 1
		state.Select(editor.SelectPlayer, idx, w.seqIndex)
		state.MarkModified()
		state.SetStatus(i18n.Tf(i18n.KeyStatusPlayerAdded, p.Label), 0)
		w.Refresh()
		w.notifyChanged()

	case editor.ToolAction:
		// Use cumulative positions for all hit tests in action mode.
		finalSeq := w.seqWithFinalPositions(seq)
		finalCarriers := anim.ComputeFinalBallCarriers(seq)

		if state.ActionFrom == nil {
			sel := state.SelectedElement
			if sel != nil && sel.Kind == editor.SelectPlayer &&
				sel.SeqIndex == w.seqIndex && sel.Index < len(seq.Players) {
				id := seq.Players[sel.Index].ID
				if model.RequiresBall(state.ToolActionType) && !slices.Contains(finalCarriers, id) {
					state.SetStatus(i18n.T(i18n.KeyStatusRequiresBall), 1)
					w.notifyChanged()
					return
				}
				state.ActionFrom = &id
				return
			}
		}

		if state.ActionFrom == nil { //nolint:nestif
			if pi := court.HitTestPlayer(&w.viewport, finalSeq, pos); pi >= 0 {
				id := seq.Players[pi].ID
				if model.RequiresBall(state.ToolActionType) && !slices.Contains(finalCarriers, id) {
					state.SetStatus(i18n.T(i18n.KeyStatusRequiresBall), 1)
					w.notifyChanged()
					return
				}
				state.ActionFrom = &id
				w.Refresh()
				w.notifyChanged()
			} else {
				// Click on empty space — cancel action tool.
				state.SetTool(editor.ToolSelect)
				w.notifyChanged()
			}
		} else {
			toRef := model.ActionRef{}
			switch {
			case model.IsShot(state.ToolActionType) && w.geom != nil:
				// Shots snap to the basket.
				toRef.Position = court.BasketRelativePosition(w.geom, w.exercise.CourtType)
			case state.ToolActionType == model.ActionPass:
				if pi := court.HitTestPlayer(&w.viewport, finalSeq, pos); pi >= 0 {
					toRef.IsPlayer = true
					toRef.PlayerID = seq.Players[pi].ID
				} else {
					// Click on empty space — cancel pass.
					state.ActionFrom = nil
					state.SetTool(editor.ToolSelect)
					w.notifyChanged()
					return
				}
			default:
				if pi := court.HitTestPlayer(&w.viewport, finalSeq, pos); pi >= 0 {
					toRef.IsPlayer = true
					toRef.PlayerID = seq.Players[pi].ID
				} else {
					toRef.Position = court.ClampPosition(&w.viewport, w.viewport.PixelToRel(pos))
				}
			}
			action := model.Action{
				Type: state.ToolActionType,
				From: model.ActionRef{IsPlayer: true, PlayerID: *state.ActionFrom},
				To:   toRef,
				Step: model.MaxStep(seq) + 1,
			}
			seq.Actions = append(seq.Actions, action)
			if state.ToolActionType == model.ActionPass && toRef.IsPlayer {
				nextIdx := w.seqIndex + 1
				if nextIdx < len(w.exercise.Sequences) {
					w.exercise.Sequences[nextIdx].BallCarrier.AddBall(toRef.PlayerID)
				}
			}
			// Chain actions from the same player. Pass → receiver becomes source.
			// Shots end the chain (can't chain after a shot).
			switch {
			case model.IsShot(state.ToolActionType):
				state.ActionFrom = nil
			case state.ToolActionType == model.ActionPass && toRef.IsPlayer:
				state.ActionFrom = &toRef.PlayerID
			}
			state.MarkModified()
			state.SetStatus(i18n.Tf(i18n.KeyStatusActionAdded, actionLabel(action.Type)), 0)
			w.Refresh()
			w.notifyChanged()
		}

	case editor.ToolAccessory:
		relPos := w.viewport.PixelToRel(pos)
		relPos = court.ClampPosition(&w.viewport,relPos)
		acc := model.Accessory{
			Type:     state.ToolAccessoryType,
			ID:       editor.NextAccessoryID(seq),
			Position: relPos,
		}
		seq.Accessories = append(seq.Accessories, acc)
		idx := len(seq.Accessories) - 1
		state.Select(editor.SelectAccessory, idx, w.seqIndex)
		state.MarkModified()
		state.SetStatus(i18n.Tf(i18n.KeyStatusAccessoryAdded, accessoryLabel(acc.Type)), 0)
		w.Refresh()
		w.notifyChanged()

	case editor.ToolDelete:
		if pi := court.HitTestPlayer(&w.viewport, seq, pos); pi >= 0 {
			playerID := seq.Players[pi].ID
			seq.Players = append(seq.Players[:pi], seq.Players[pi+1:]...)
			removeActionsForPlayer(seq, playerID)
			seq.BallCarrier.RemoveBall(playerID)
			state.Deselect()
			state.MarkModified()
			w.Refresh()
			w.notifyChanged()
			return
		}
		if ai := court.HitTestAccessory(&w.viewport, seq, pos); ai >= 0 {
			seq.Accessories = append(seq.Accessories[:ai], seq.Accessories[ai+1:]...)
			state.Deselect()
			state.MarkModified()
			w.Refresh()
			w.notifyChanged()
			return
		}
		if actIdx := court.HitTestAction(&w.viewport, seq, pos); actIdx >= 0 {
			seq.Actions = append(seq.Actions[:actIdx], seq.Actions[actIdx+1:]...)
			model.ReorderSteps(seq)
			state.Deselect()
			state.MarkModified()
			w.Refresh()
			w.notifyChanged()
			return
		}
		w.DeleteSelected()
	}
}

// handlePanDrag handles panning with drag delta (in dp).
func (w *CourtWidget) handlePanDrag(delta fyne.Delta) {
	if w.baseVP.Width <= 0 || w.baseVP.Height <= 0 {
		return
	}
	size := w.Size()
	if size.Width <= 0 || size.Height <= 0 {
		return
	}
	// Convert dp delta to relative pan offset.
	// baseVP dimensions are in pixels, so we need to account for the zoom.
	w.panX += float64(delta.DX) / float64(size.Width)
	w.panY += float64(delta.DY) / float64(size.Height)
	w.clampPan()
	w.courtBg = nil
	w.Refresh()
}

func (w *CourtWidget) handleDrag(pos court.Point) {
	state := w.editorState
	if state == nil || !w.dragActive {
		return
	}

	seq := w.currentSequence()
	if seq == nil {
		return
	}

	if w.dragRotating {
		sel := state.SelectedElement
		if sel == nil {
			return
		}
		finalSeq := w.seqWithFinalPositions(seq)
		var center court.Point
		switch sel.Kind {
		case editor.SelectPlayer:
			if sel.Index < len(finalSeq.Players) {
				center = w.viewport.RelToPixel(finalSeq.Players[sel.Index].Position)
			}
		case editor.SelectAccessory:
			if sel.Index < len(seq.Accessories) {
				center = w.viewport.RelToPixel(seq.Accessories[sel.Index].Position)
			}
		default:
			return
		}
		dx := float64(pos.X - center.X)
		dy := float64(pos.Y - center.Y)
		angle := math.Atan2(dx, -dy) * 180 / math.Pi
		if angle < 0 {
			angle += 360
		}
		angle = math.Round(angle/15) * 15
		if angle >= 360 {
			angle -= 360
		}
		switch sel.Kind {
		case editor.SelectPlayer:
			seq.Players[sel.Index].Rotation = angle
		case editor.SelectAccessory:
			seq.Accessories[sel.Index].Rotation = angle
		}
		state.MarkModified()
		w.Refresh()
		w.notifyChanged()
		return
	}

	relPos := court.ClampPosition(&w.viewport,w.viewport.PixelToRel(pos))

	if w.dragPlayerIdx >= 0 && w.dragPlayerIdx < len(seq.Players) {
		w.dragPlayer(seq, relPos, state)
	}
	if w.dragAccIdx >= 0 && w.dragAccIdx < len(seq.Accessories) {
		seq.Accessories[w.dragAccIdx].Position = relPos
		state.MarkModified()
		w.Refresh()
	}
	if w.dragActionEndIdx >= 0 && w.dragActionEndIdx < len(seq.Actions) {
		if w.dragWaypointIdx >= 0 {
			// Dragging an existing waypoint.
			seq.Actions[w.dragActionEndIdx].Waypoints[w.dragWaypointIdx] = relPos
			state.MarkModified()
			w.Refresh()
		} else {
			w.dragActionEndpoint(seq, pos, relPos, state)
		}
	}
}

// hitTestActionStepAware tests if pos hits any action using step-specific player positions.
func (w *CourtWidget) hitTestActionStepAware(seq *model.Sequence, pos court.Point) int {
	maxStep := model.MaxStep(seq)
	hitThreshold := w.viewport.Sd(court.ActionHitThreshold)
	for i := len(seq.Actions) - 1; i >= 0; i-- {
		act := &seq.Actions[i]
		playersAtStep := stepPlayers(seq, maxStep, act.EffectiveStep())
		from := court.ResolveRef(&w.viewport, act.From, playersAtStep)
		to := court.ResolveRef(&w.viewport, act.To, playersAtStep)
		if len(act.Waypoints) > 0 {
			wps := court.ResolveWaypoints(&w.viewport, act.Waypoints)
			pts := court.BezierPath(from, to, wps, 16)
			if court.DistToPolyline(pos, pts) <= hitThreshold {
				return i
			}
		} else {
			if court.DistToSegment(pos, from, to) <= hitThreshold {
				return i
			}
		}
	}
	return -1
}

// hitTestWaypoint checks if pos hits a waypoint handle or the midpoint of the action.
// Returns waypoint index (>=0), -1 for midpoint (create new), or -2 for no hit.
func (w *CourtWidget) hitTestWaypoint(act *model.Action, seq *model.Sequence, pos court.Point) int {
	hitR := w.viewport.Sd(8)
	// Check existing waypoints.
	for i, wp := range act.Waypoints {
		center := w.viewport.RelToPixel(wp)
		dx := float64(pos.X - center.X)
		dy := float64(pos.Y - center.Y)
		if math.Sqrt(dx*dx+dy*dy) <= hitR {
			return i
		}
	}
	// Check midpoint handle (only if no waypoints).
	if len(act.Waypoints) == 0 {
		maxStep := model.MaxStep(seq)
		playersAtStep := stepPlayers(seq, maxStep, act.EffectiveStep())
		from := court.ResolveRef(&w.viewport, act.From, playersAtStep)
		to := court.ResolveRef(&w.viewport, act.To, playersAtStep)
		mid := court.Pt((from.X+to.X)/2, (from.Y+to.Y)/2)
		dx := float64(pos.X - mid.X)
		dy := float64(pos.Y - mid.Y)
		if math.Sqrt(dx*dx+dy*dy) <= hitR {
			return -1
		}
	}
	return -2
}

// hitTestGhostPlayer checks if pos hits a ghost player (initial or intermediate step position).
// Returns the player index and step number (0 = initial position).
func (w *CourtWidget) hitTestGhostPlayer(seq *model.Sequence, pos court.Point) (int, int) {
	maxStep := model.MaxStep(seq)
	if maxStep <= 0 {
		return -1, 0
	}
	finalPlayers := stepPlayers(seq, maxStep, maxStep+1)

	// Check initial position ghosts.
	for i := range seq.Players {
		if seq.Players[i].Position == finalPlayers[i].Position {
			continue // no ghost at this position
		}
		center := w.viewport.RelToPixel(seq.Players[i].Position)
		dx := pos.X - center.X
		dy := pos.Y - center.Y
		if math.Sqrt(float64(dx*dx+dy*dy)) <= w.viewport.Sd(court.PlayerRadius+4) {
			return i, 0
		}
	}

	// Check intermediate step ghosts.
	for step := 2; step <= maxStep; step++ {
		ghostPlayers := stepPlayers(seq, maxStep, step)
		for i := range ghostPlayers {
			if ghostPlayers[i].Position == seq.Players[i].Position ||
				ghostPlayers[i].Position == finalPlayers[i].Position {
				continue
			}
			center := w.viewport.RelToPixel(ghostPlayers[i].Position)
			dx := pos.X - center.X
			dy := pos.Y - center.Y
			if math.Sqrt(float64(dx*dx+dy*dy)) <= w.viewport.Sd(court.PlayerRadius+4) {
				return i, step - 1 // step-1 = the movement action step that brought player here
			}
		}
	}
	return -1, 0
}

// seqWithFinalPositions returns a shallow copy of the sequence with players at their
// cumulative step positions (for hit testing against visually displayed positions).
func (w *CourtWidget) seqWithFinalPositions(seq *model.Sequence) *model.Sequence {
	maxStep := model.MaxStep(seq)
	if maxStep <= 0 {
		return seq
	}
	final := *seq
	final.Players = make([]model.Player, len(seq.Players))
	copy(final.Players, seq.Players)
	positions := anim.ComputeStepPositions(seq, maxStep, 1.0)
	for i := range final.Players {
		if pos, ok := positions[final.Players[i].ID]; ok {
			final.Players[i].Position = pos
		}
	}
	return &final
}

// trySelectElement checks if the press hits a player, accessory, or action and starts drag.
// Switches to ToolSelect so the shelf opens with props.
func (w *CourtWidget) trySelectElement(seq *model.Sequence, state *editor.EditorState, pos court.Point) bool {
	// Use cumulative positions for hit testing (players are drawn at final positions).
	finalSeq := w.seqWithFinalPositions(seq)
	if w.hitTestRotationHandle(finalSeq, state, pos) {
		w.dragRotating = true
		w.dragActive = true
		state.IsRotating = true
		w.Refresh()
		return true
	}
	// Hit test final player position first.
	if pi := court.HitTestPlayer(&w.viewport, finalSeq, pos); pi >= 0 {
		state.SetTool(editor.ToolSelect)
		state.Select(editor.SelectPlayer, pi, w.seqIndex)
		w.dragPlayerIdx = pi
		w.dragPlayerStep = -1
		w.selectedStep = -1
		w.dragActive = true
		state.IsDragging = true
		w.Refresh()
		w.notifyChanged()
		return true
	}
	// Hit test ghost positions (initial + intermediate steps).
	if pi, step := w.hitTestGhostPlayer(seq, pos); pi >= 0 {
		state.SetTool(editor.ToolSelect)
		state.Select(editor.SelectPlayer, pi, w.seqIndex)
		w.dragPlayerIdx = pi
		w.dragPlayerStep = step
		w.selectedStep = step
		w.dragActive = true
		state.IsDragging = true
		w.Refresh()
		w.notifyChanged()
		return true
	}
	if ai := court.HitTestAccessory(&w.viewport, finalSeq, pos); ai >= 0 {
		state.SetTool(editor.ToolSelect)
		state.Select(editor.SelectAccessory, ai, w.seqIndex)
		w.dragAccIdx = ai
		w.dragActive = true
		state.IsDragging = true
		w.Refresh()
		w.notifyChanged()
		return true
	}
	// Check waypoint handles on currently selected action first.
	if selElem := state.SelectedElement; selElem != nil && selElem.Kind == editor.SelectAction && selElem.Index < len(seq.Actions) {
		act := &seq.Actions[selElem.Index]
		wpIdx := w.hitTestWaypoint(act, seq, pos)
		if wpIdx >= -1 { // -1 = midpoint (create new), >= 0 = existing waypoint
			w.dragActionEndIdx = selElem.Index
			w.dragWaypointIdx = wpIdx
			w.dragActive = true
			state.IsDragging = true
			if wpIdx == -1 {
				// Create waypoint at midpoint.
				maxStep := model.MaxStep(seq)
				playersAtStep := stepPlayers(seq, maxStep, act.EffectiveStep())
				from := court.ResolveRef(&w.viewport, act.From, playersAtStep)
				to := court.ResolveRef(&w.viewport, act.To, playersAtStep)
				mid := w.viewport.PixelToRel(court.Pt((from.X+to.X)/2, (from.Y+to.Y)/2))
				act.Waypoints = append(act.Waypoints, mid)
				w.dragWaypointIdx = len(act.Waypoints) - 1
				state.MarkModified()
			}
			w.Refresh()
			return true
		}
	}
	if actIdx := w.hitTestActionStepAware(seq, pos); actIdx >= 0 {
		state.SetTool(editor.ToolSelect)
		state.Select(editor.SelectAction, actIdx, w.seqIndex)
		w.dragActionEndIdx = actIdx
		w.dragWaypointIdx = -2 // endpoint drag
		w.dragActive = true
		state.IsDragging = true
		w.Refresh()
		w.notifyChanged()
		return true
	}
	return false
}

func (w *CourtWidget) dragPlayer(seq *model.Sequence, relPos model.Position, state *editor.EditorState) {
	pid := seq.Players[w.dragPlayerIdx].ID
	relPos = court.AvoidOverlap(relPos, pid, seq.Players)

	switch {
	case w.dragPlayerStep == 0:
		w.dragPlayerBase(seq, pid, relPos)
	case w.dragPlayerStep > 0:
		w.dragPlayerAtStep(seq, pid, relPos, w.dragPlayerStep)
	default:
		w.dragPlayerFinal(seq, pid, relPos)
	}
	state.MarkModified()
	w.Refresh()
}

func (w *CourtWidget) dragPlayerBase(seq *model.Sequence, pid string, relPos model.Position) {
	oldPos := seq.Players[w.dragPlayerIdx].Position
	dx := relPos[0] - oldPos[0]
	dy := relPos[1] - oldPos[1]
	seq.Players[w.dragPlayerIdx].Position = relPos
	for i := range seq.Actions {
		a := &seq.Actions[i]
		if a.From.IsPlayer && a.From.PlayerID == pid && !a.To.IsPlayer &&
			!model.IsShot(a.Type) && !model.IsMovementAction(a.Type) {
			a.To.Position[0] += dx
			a.To.Position[1] += dy
		}
	}
}

func (w *CourtWidget) dragPlayerAtStep(seq *model.Sequence, pid string, relPos model.Position, step int) {
	for i := range seq.Actions {
		a := &seq.Actions[i]
		if a.From.IsPlayer && a.From.PlayerID == pid && model.IsMovementAction(a.Type) &&
			!a.To.IsPlayer && a.EffectiveStep() == step {
			a.To.Position = relPos
			break
		}
	}
}

func (w *CourtWidget) dragPlayerFinal(seq *model.Sequence, pid string, relPos model.Position) {
	lastMoveIdx := -1
	for i := range seq.Actions {
		a := &seq.Actions[i]
		if a.From.IsPlayer && a.From.PlayerID == pid && model.IsMovementAction(a.Type) && !a.To.IsPlayer {
			lastMoveIdx = i
		}
	}
	if lastMoveIdx >= 0 {
		seq.Actions[lastMoveIdx].To.Position = relPos
	} else {
		w.dragPlayerBase(seq, pid, relPos)
	}
}

func (w *CourtWidget) dragActionEndpoint(seq *model.Sequence, pos court.Point, relPos model.Position, state *editor.EditorState) {
	act := &seq.Actions[w.dragActionEndIdx]
	// Shots snap to the basket when within 30dp.
	if model.IsShot(act.Type) && w.geom != nil {
		basketPos := court.BasketRelativePosition(w.geom, w.exercise.CourtType)
		basketPx := w.viewport.RelToPixel(basketPos)
		dx := float64(pos.X - basketPx.X)
		dy := float64(pos.Y - basketPx.Y)
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist <= w.viewport.Sd(court.MagneticSnapDist) {
			act.To.IsPlayer = false
			act.To.PlayerID = ""
			act.To.Position = basketPos
		} else {
			act.To.IsPlayer = false
			act.To.PlayerID = ""
			act.To.Position = relPos
		}
		state.MarkModified()
		w.Refresh()
		w.notifyChanged()
		return
	}
	pi := court.HitTestPlayer(&w.viewport, seq, pos)
	if pi >= 0 {
		act.To.IsPlayer = true
		act.To.PlayerID = seq.Players[pi].ID
		act.To.Position = model.Position{}
	} else {
		act.To.IsPlayer = false
		act.To.PlayerID = ""
		act.To.Position = relPos
	}
	state.MarkModified()
	w.Refresh()
	w.notifyChanged()
}

func (w *CourtWidget) handleRelease() {
	if w.editorState != nil {
		w.editorState.IsDragging = false
		w.editorState.IsRotating = false
	}
	w.dragActive = false
	w.dragPlayerIdx = -1
	w.dragAccIdx = -1
	w.dragActionEndIdx = -1
	w.dragRotating = false
	w.dragPanning = false
	w.Refresh()
}

// DeleteSelected removes the currently selected element from the sequence.
func (w *CourtWidget) DeleteSelected() {
	state := w.editorState
	if state == nil {
		return
	}
	sel := state.SelectedElement
	if sel == nil || sel.SeqIndex != w.seqIndex {
		return
	}
	seq := w.currentSequence()
	if seq == nil {
		return
	}
	var statusMsg string
	switch sel.Kind {
	case editor.SelectPlayer:
		if sel.Index < len(seq.Players) {
			label := seq.Players[sel.Index].Label
			playerID := seq.Players[sel.Index].ID
			seq.Players = append(seq.Players[:sel.Index], seq.Players[sel.Index+1:]...)
			removeActionsForPlayer(seq, playerID)
			seq.BallCarrier.RemoveBall(playerID)
			statusMsg = i18n.Tf(i18n.KeyStatusPlayerDeleted, label)
		}
	case editor.SelectAccessory:
		if sel.Index < len(seq.Accessories) {
			label := accessoryLabel(seq.Accessories[sel.Index].Type)
			seq.Accessories = append(seq.Accessories[:sel.Index], seq.Accessories[sel.Index+1:]...)
			statusMsg = i18n.Tf(i18n.KeyStatusAccessoryDeleted, label)
		}
	case editor.SelectAction:
		if sel.Index < len(seq.Actions) {
			label := actionLabel(seq.Actions[sel.Index].Type)
			seq.Actions = append(seq.Actions[:sel.Index], seq.Actions[sel.Index+1:]...)
			model.ReorderSteps(seq)
			statusMsg = i18n.Tf(i18n.KeyStatusActionDeleted, label)
		}
	default:
		return
	}
	state.Deselect()
	state.MarkModified()
	state.SetStatus(statusMsg, 0)
	w.Refresh()
	w.notifyChanged()
}

func (w *CourtWidget) hitTestRotationHandle(seq *model.Sequence, state *editor.EditorState, pos court.Point) bool {
	sel := state.SelectedElement
	if sel == nil || sel.SeqIndex != w.seqIndex {
		return false
	}
	switch sel.Kind {
	case editor.SelectPlayer:
		if sel.Index >= len(seq.Players) {
			return false
		}
		center := w.viewport.RelToPixel(seq.Players[sel.Index].Position)
		return court.HitTestRotationHandleScaled(&w.viewport, center, seq.Players[sel.Index].Rotation, pos)
	case editor.SelectAccessory:
		if sel.Index >= len(seq.Accessories) {
			return false
		}
		center := w.viewport.RelToPixel(seq.Accessories[sel.Index].Position)
		return court.HitTestRotationHandleScaled(&w.viewport, center, seq.Accessories[sel.Index].Rotation, pos)
	}
	return false
}

func (w *CourtWidget) notifyChanged() {
	if w.OnChanged != nil {
		w.OnChanged()
	}
}

func removeActionsForPlayer(seq *model.Sequence, playerID string) {
	filtered := seq.Actions[:0]
	for _, a := range seq.Actions {
		if (a.From.IsPlayer && a.From.PlayerID == playerID) ||
			(a.To.IsPlayer && a.To.PlayerID == playerID) {
			continue
		}
		filtered = append(filtered, a)
	}
	seq.Actions = filtered
	model.ReorderSteps(seq)
}

// --- Renderer ---

type courtRenderer struct {
	widget *CourtWidget
}

func (r *courtRenderer) Layout(size fyne.Size) {
	r.widget.raster.Resize(size)
	r.widget.raster.Move(fyne.NewPos(0, 0))
	// Position zoom label at top-right corner.
	r.widget.zoomLabel.Move(fyne.NewPos(size.Width-60, 4))
}

func (r *courtRenderer) MinSize() fyne.Size {
	return fyne.NewSize(200, 200)
}

func (r *courtRenderer) Refresh() {
	r.widget.raster.Refresh()
	r.widget.zoomLabel.Refresh()
}

func (r *courtRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.widget.raster, r.widget.zoomLabel}
}

func (r *courtRenderer) Destroy() {
	w := r.widget
	w.animMu.Lock()
	if w.animMode {
		w.animMode = false
		close(w.animStop)
	}
	w.animMu.Unlock()
	w.stopPulseTicker()
}

