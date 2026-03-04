package editor

import (
	"fmt"
	"time"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// ToolType identifies the active editing tool.
type ToolType int

const (
	ToolNone   ToolType = iota
	ToolSelect          // click to select/move elements
	ToolPlayer          // click to place a player (role in ToolRole)
	ToolAction          // two-click flow to create an action
	ToolAccessory       // click to place an accessory
	ToolDelete          // click to remove an element
)

// SelectionKind identifies what type of element is selected.
type SelectionKind int

const (
	SelectPlayer    SelectionKind = iota
	SelectAction
	SelectAccessory
)

// Selection tracks the currently selected element.
type Selection struct {
	Kind     SelectionKind
	Index    int // index within the sequence slice
	SeqIndex int // which sequence
}

// EditorState holds all transient editing state.
type EditorState struct {
	ActiveTool ToolType

	// ToolRole is the player role when ActiveTool == ToolPlayer.
	ToolRole model.PlayerRole
	// ToolQueue is true when the player tool creates a queue player.
	ToolQueue bool
	// ToolActionType is the action type when ActiveTool == ToolAction.
	ToolActionType model.ActionType
	// ToolAccessoryType is the accessory type when ActiveTool == ToolAccessory.
	ToolAccessoryType model.AccessoryType

	// SelectedElement is the currently selected element (nil if nothing selected).
	SelectedElement *Selection

	// ActionFrom holds the player ID for the first click of action creation.
	ActionFrom *string

	// Drag state.
	IsDragging   bool
	IsRotating   bool
	DragStartPos model.Position

	// Modified tracks unsaved changes.
	Modified bool

	// DeleteRequested is set when the user clicks the Delete tool while
	// an element is selected. The court widget consumes this flag.
	DeleteRequested bool

	// Status bar fields.
	StatusMsg   string
	StatusLevel int // 0=info, 1=error
	StatusAt    time.Time
}

// Select marks an element as selected.
func (s *EditorState) Select(kind SelectionKind, index, seqIndex int) {
	s.SelectedElement = &Selection{
		Kind:     kind,
		Index:    index,
		SeqIndex: seqIndex,
	}
}

// Deselect clears the selection.
func (s *EditorState) Deselect() {
	s.SelectedElement = nil
}

// SetTool changes the active tool and clears partial state.
func (s *EditorState) SetTool(tool ToolType) {
	s.ActiveTool = tool
	s.ActionFrom = nil
}

// SetPlayerTool activates the player placement tool with a specific role.
func (s *EditorState) SetPlayerTool(role model.PlayerRole) {
	s.ActiveTool = ToolPlayer
	s.ToolRole = role
	s.ToolQueue = false
	s.ActionFrom = nil
}

// SetQueueTool activates the player placement tool in queue mode.
func (s *EditorState) SetQueueTool() {
	s.ActiveTool = ToolPlayer
	s.ToolRole = model.RoleAttacker
	s.ToolQueue = true
	s.ActionFrom = nil
}

// SetActionTool activates the action creation tool with a specific type.
func (s *EditorState) SetActionTool(actionType model.ActionType) {
	s.ActiveTool = ToolAction
	s.ToolActionType = actionType
	s.ActionFrom = nil
}

// SetAccessoryTool activates the accessory placement tool with a specific type.
func (s *EditorState) SetAccessoryTool(accessoryType model.AccessoryType) {
	s.ActiveTool = ToolAccessory
	s.ToolAccessoryType = accessoryType
	s.ActionFrom = nil
}

// SetStatus sets the status bar message and level.
func (s *EditorState) SetStatus(msg string, level int) {
	s.StatusMsg = msg
	s.StatusLevel = level
	s.StatusAt = time.Now()
}

// MarkModified flags the exercise as having unsaved changes.
func (s *EditorState) MarkModified() {
	s.Modified = true
}

// ClearModified clears the modified flag (after save).
func (s *EditorState) ClearModified() {
	s.Modified = false
}

// NextPlayerID scans existing players in the sequence and returns the next
// available ID like "p1", "p2", etc.
func NextPlayerID(seq *model.Sequence) string {
	max := 0
	for _, p := range seq.Players {
		var n int
		if _, err := fmt.Sscanf(p.ID, "p%d", &n); err == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("p%d", max+1)
}

// NextAccessoryID scans existing accessories and returns the next available ID.
func NextAccessoryID(seq *model.Sequence) string {
	max := 0
	for _, a := range seq.Accessories {
		var n int
		if _, err := fmt.Sscanf(a.ID, "acc%d", &n); err == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("acc%d", max+1)
}
