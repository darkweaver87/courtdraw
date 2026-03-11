package fynecourt

import (
	"fmt"
	"image"
	"image/color"
	"math"
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
	pressed       bool
	dragPlayerIdx int
	dragAccIdx    int
	dragActive    bool
	dragRotating  bool

	// Zoom indicator overlay.
	zoomLabel *canvas.Text

	// Animation ticker.
	animStop chan struct{}
	animMu   sync.Mutex

	OnChanged func()
}

// Ensure interface compliance.
var _ fyne.Tappable = (*CourtWidget)(nil)
var _ fyne.Draggable = (*CourtWidget)(nil)
var _ fyne.Scrollable = (*CourtWidget)(nil)
var _ desktop.Mouseable = (*CourtWidget)(nil)
var _ fyne.Focusable = (*CourtWidget)(nil)

// NewCourtWidget creates a new court widget.
func NewCourtWidget() *CourtWidget {
	w := &CourtWidget{
		dragPlayerIdx: -1,
		dragAccIdx:    -1,
		zoomLevel:     1.0,
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
	if sel != nil {
		selElem = sel.SelectedElement
	}

	// Accessories.
	for i := range seq.Accessories {
		selected := selElem != nil && selElem.Kind == editor.SelectAccessory && selElem.Index == i && selElem.SeqIndex == w.seqIndex
		court.DrawAccessory(img, &w.viewport, &seq.Accessories[i], selected)
	}

	// Actions.
	for i := range seq.Actions {
		if selElem != nil && selElem.Kind == editor.SelectAction && selElem.Index == i && selElem.SeqIndex == w.seqIndex {
			court.DrawActionHighlight(img, &w.viewport, &seq.Actions[i], seq.Players)
		}
		court.DrawAction(img, &w.viewport, &seq.Actions[i], seq.Players)
	}

	// Players.
	for i := range seq.Players {
		selected := selElem != nil && selElem.Kind == editor.SelectPlayer && selElem.Index == i && selElem.SeqIndex == w.seqIndex
		hasBall := seq.BallCarrier.HasBall(seq.Players[i].ID)
		label := resolvePlayerLabel(&seq.Players[i])
		court.DrawPlayerWithLabel(img, &w.viewport, &seq.Players[i], label, face, selected, hasBall)
	}

	// Callouts.
	for i := range seq.Players {
		if seq.Players[i].Callout != "" {
			calloutText := i18n.T("callout." + string(seq.Players[i].Callout))
			court.DrawCallout(img, &w.viewport, &seq.Players[i], calloutText, face, 0xff)
		}
	}

	// Action-from indicator.
	if sel != nil && sel.ActionFrom != nil {
		for i := range seq.Players {
			if seq.Players[i].ID == *sel.ActionFrom {
				center := w.viewport.RelToPixel(seq.Players[i].Position)
				court.DrawCircleOutline(img, center, court.PlayerRadius+6, 2, court.ColorPass)
				break
			}
		}
	}

	// Rotation handle for selected element.
	if selElem != nil && selElem.SeqIndex == w.seqIndex {
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
		}
	}
}

func (w *CourtWidget) drawAnimatedFrame(img *image.RGBA, face font.Face, frame *anim.AnimatedFrame) {
	vp := &w.viewport
	// Accessories.
	for i := range frame.Accessories {
		court.DrawAccessoryWithOpacity(img, vp, &frame.Accessories[i].Accessory, frame.Accessories[i].Opacity)
	}

	// Actions with progress.
	players := make([]model.Player, len(frame.Players))
	for i := range frame.Players {
		players[i] = frame.Players[i].Player
	}
	for i := range frame.Actions {
		court.DrawActionWithProgress(img, vp, &frame.Actions[i].Action, players, frame.Actions[i].Progress)
	}

	// Players with opacity.
	for i := range frame.Players {
		label := resolvePlayerLabel(&frame.Players[i].Player)
		court.DrawPlayerWithOpacity(img, vp, &frame.Players[i].Player, label, face, frame.Players[i].Opacity, false)
	}

	// Callouts.
	for i := range frame.Players {
		if frame.Players[i].Player.Callout != "" {
			calloutText := i18n.T("callout." + string(frame.Players[i].Player.Callout))
			alpha := uint8(frame.Players[i].Opacity * 255)
			court.DrawCallout(img, vp, &frame.Players[i].Player, calloutText, face, alpha)
		}
	}

	// Balls.
	for _, b := range frame.Balls {
		if b.Opacity > 0 {
			ballPixel := vp.RelToPixel(b.Pos)
			court.DrawBallWithOpacity(img, vp, ballPixel, b.Opacity)
		}
	}
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

// Tappable — used for mobile and simple clicks.
func (w *CourtWidget) Tapped(e *fyne.PointEvent) {
	if w.pressHandled {
		w.pressHandled = false // consumed, reset for next gesture
		return                 // already handled by MouseDown
	}
	pos := w.dpToPixel(e.Position)
	w.handlePress(pos)
}

// Draggable.
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

// desktop.Mouseable — for desktop press/release (more precise than Tapped).
func (w *CourtWidget) MouseDown(e *desktop.MouseEvent) {
	if e.Button != desktop.MouseButtonPrimary {
		return
	}
	pos := w.dpToPixel(e.PointEvent.Position)
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

// Focusable — for keyboard events.
func (w *CourtWidget) FocusGained()   {}
func (w *CourtWidget) FocusLost()     {}
func (w *CourtWidget) TypedRune(r rune) {}
func (w *CourtWidget) TypedKey(e *fyne.KeyEvent) {
	if e.Name == fyne.KeyDelete || e.Name == fyne.KeyBackspace {
		w.DeleteSelected()
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

func (w *CourtWidget) handlePress(pos court.Point) {
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
	w.dragRotating = false
	w.dragPanning = false

	switch state.ActiveTool {
	case editor.ToolNone, editor.ToolSelect:
		// Check rotation handle first.
		if w.hitTestRotationHandle(seq, state, pos) {
			w.dragRotating = true
			w.dragActive = true
			state.IsRotating = true
			w.Refresh()
			return
		}

		if pi := court.HitTestPlayer(&w.viewport, seq, pos); pi >= 0 {
			state.Select(editor.SelectPlayer, pi, w.seqIndex)
			w.dragPlayerIdx = pi
			w.dragActive = true
			state.IsDragging = true
			w.Refresh()
			w.notifyChanged()
			return
		}
		if ai := court.HitTestAccessory(&w.viewport, seq, pos); ai >= 0 {
			state.Select(editor.SelectAccessory, ai, w.seqIndex)
			w.dragAccIdx = ai
			w.dragActive = true
			state.IsDragging = true
			w.Refresh()
			w.notifyChanged()
			return
		}
		if actIdx := court.HitTestAction(&w.viewport, seq, pos); actIdx >= 0 {
			state.Select(editor.SelectAction, actIdx, w.seqIndex)
			w.Refresh()
			w.notifyChanged()
			return
		}

		// If zoomed in and no element hit, start panning.
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
		state.SetStatus(i18n.Tf("status.player_added", p.Label), 0)
		w.Refresh()
		w.notifyChanged()

	case editor.ToolAction:
		if state.ActionFrom == nil {
			sel := state.SelectedElement
			if sel != nil && sel.Kind == editor.SelectPlayer &&
				sel.SeqIndex == w.seqIndex && sel.Index < len(seq.Players) {
				id := seq.Players[sel.Index].ID
				if model.RequiresBall(state.ToolActionType) && !seq.BallCarrier.HasBall(id) {
					state.SetStatus(i18n.T("status.requires_ball"), 1)
					w.notifyChanged()
					return
				}
				state.ActionFrom = &id
			}
		}

		if state.ActionFrom == nil {
			if pi := court.HitTestPlayer(&w.viewport, seq, pos); pi >= 0 {
				id := seq.Players[pi].ID
				if model.RequiresBall(state.ToolActionType) && !seq.BallCarrier.HasBall(id) {
					state.SetStatus(i18n.T("status.requires_ball"), 1)
					w.notifyChanged()
					return
				}
				state.ActionFrom = &id
			}
		} else {
			toRef := model.ActionRef{}
			if pi := court.HitTestPlayer(&w.viewport, seq, pos); pi >= 0 {
				toRef.IsPlayer = true
				toRef.PlayerID = seq.Players[pi].ID
			} else if state.ToolActionType == model.ActionPass {
				state.SetStatus(i18n.T("status.pass_requires_player"), 1)
				w.notifyChanged()
				return
			} else {
				toRef.Position = court.ClampPosition(&w.viewport,w.viewport.PixelToRel(pos))
			}
			action := model.Action{
				Type: state.ToolActionType,
				From: model.ActionRef{IsPlayer: true, PlayerID: *state.ActionFrom},
				To:   toRef,
			}
			seq.Actions = append(seq.Actions, action)
			if state.ToolActionType == model.ActionPass && toRef.IsPlayer {
				nextIdx := w.seqIndex + 1
				if nextIdx < len(w.exercise.Sequences) {
					w.exercise.Sequences[nextIdx].BallCarrier.AddBall(toRef.PlayerID)
				}
			}
			state.ActionFrom = nil
			state.MarkModified()
			state.SetStatus(i18n.Tf("status.action_added", string(action.Type)), 0)
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
		state.SetStatus(i18n.Tf("status.accessory_added", string(acc.Type)), 0)
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
		var center court.Point
		switch sel.Kind {
		case editor.SelectPlayer:
			if sel.Index < len(seq.Players) {
				center = w.viewport.RelToPixel(seq.Players[sel.Index].Position)
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
		seq.Players[w.dragPlayerIdx].Position = relPos
		state.MarkModified()
		w.Refresh()
	}
	if w.dragAccIdx >= 0 && w.dragAccIdx < len(seq.Accessories) {
		seq.Accessories[w.dragAccIdx].Position = relPos
		state.MarkModified()
		w.Refresh()
	}
}

func (w *CourtWidget) handleRelease() {
	if w.editorState != nil {
		w.editorState.IsDragging = false
		w.editorState.IsRotating = false
	}
	w.dragActive = false
	w.dragPlayerIdx = -1
	w.dragAccIdx = -1
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
	switch sel.Kind {
	case editor.SelectPlayer:
		if sel.Index < len(seq.Players) {
			playerID := seq.Players[sel.Index].ID
			seq.Players = append(seq.Players[:sel.Index], seq.Players[sel.Index+1:]...)
			removeActionsForPlayer(seq, playerID)
			seq.BallCarrier.RemoveBall(playerID)
		}
	case editor.SelectAccessory:
		if sel.Index < len(seq.Accessories) {
			seq.Accessories = append(seq.Accessories[:sel.Index], seq.Accessories[sel.Index+1:]...)
		}
	case editor.SelectAction:
		if sel.Index < len(seq.Actions) {
			seq.Actions = append(seq.Actions[:sel.Index], seq.Actions[sel.Index+1:]...)
		}
	default:
		return
	}
	state.Deselect()
	state.MarkModified()
	state.SetStatus(i18n.T("status.element_deleted"), 0)
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
	defer w.animMu.Unlock()
	if w.animMode {
		w.animMode = false
		close(w.animStop)
	}
}

