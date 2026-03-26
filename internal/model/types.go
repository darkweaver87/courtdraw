package model

import (
	"image/color"
	"slices"
)

// Position represents a relative coordinate on the court [0.0–1.0].
// [0,0] = bottom-left, [1,1] = top-right.
type Position [2]float64

func (p Position) X() float64 { return p[0] }
func (p Position) Y() float64 { return p[1] }

// CourtType defines half or full court.
type CourtType string

const (
	HalfCourt CourtType = "half_court"
	FullCourt CourtType = "full_court"
)

// CourtStandard defines the court standard.
type CourtStandard string

const (
	FIBA CourtStandard = "fiba"
	NBA  CourtStandard = "nba"
)

// PlayerRole defines the player's role or position.
type PlayerRole string

const (
	RoleAttacker      PlayerRole = "attacker"
	RoleDefender      PlayerRole = "defender"
	RoleCoach         PlayerRole = "coach"
	RolePointGuard    PlayerRole = "point_guard"
	RoleShootingGuard PlayerRole = "shooting_guard"
	RoleSmallForward  PlayerRole = "small_forward"
	RolePowerForward  PlayerRole = "power_forward"
	RoleCenter        PlayerRole = "center"
)

// ActionType defines the type of action/movement.
type ActionType string

const (
	ActionPass    ActionType = "pass"
	ActionDribble ActionType = "dribble"
	ActionCut     ActionType = "cut"
	ActionScreen  ActionType = "screen"
	ActionShot    ActionType = "shot"
	ActionHandoff ActionType = "handoff"

	// ActionSprint and other legacy types are mapped to canonical types via NormalizeActionType.
	ActionSprint     ActionType = "sprint"      // → Cut
	ActionShotLayup  ActionType = "shot_layup"   // → Shot
	ActionShotPushup ActionType = "shot_pushup"  // → Shot
	ActionShotJump   ActionType = "shot_jumpshot" // → Shot
	ActionCloseOut   ActionType = "close_out"    // → Cut
	ActionContest    ActionType = "contest"      // → Cut
	ActionReverse    ActionType = "reverse"      // → Cut
)

// NormalizeActionType maps legacy action types to their canonical equivalents.
func NormalizeActionType(at ActionType) ActionType {
	switch at {
	case ActionSprint, ActionCloseOut, ActionContest, ActionReverse:
		return ActionCut
	case ActionShotLayup, ActionShotPushup, ActionShotJump:
		return ActionShot
	default:
		return at
	}
}

// IsShot returns true if the action type is a shot.
func IsShot(at ActionType) bool {
	return NormalizeActionType(at) == ActionShot
}

// IsMovementAction returns true if the action type physically moves the source player.
func IsMovementAction(at ActionType) bool {
	n := NormalizeActionType(at)
	switch n {
	case ActionDribble, ActionCut, ActionScreen:
		return true
	default:
		return false
	}
}

// AccessoryType defines the type of court accessory.
type AccessoryType string

const (
	AccessoryCone          AccessoryType = "cone"
	AccessoryAgilityLadder AccessoryType = "agility_ladder"
	AccessoryChair         AccessoryType = "chair"
)

// Category classifies an exercise.
type Category string

const (
	CategoryWarmup     Category = "warmup"
	CategoryOffense    Category = "offense"
	CategoryDefense    Category = "defense"
	CategoryTransition Category = "transition"
	CategoryScrimmage  Category = "scrimmage"
	CategoryCooldown   Category = "cooldown"
)

// AgeGroup classifies the target age group for an exercise.
type AgeGroup string

const (
	AgeGroupU9     AgeGroup = "u9"
	AgeGroupU11    AgeGroup = "u11"
	AgeGroupU13    AgeGroup = "u13"
	AgeGroupU15    AgeGroup = "u15"
	AgeGroupU17    AgeGroup = "u17"
	AgeGroupU19    AgeGroup = "u19"
	AgeGroupSenior AgeGroup = "senior"
)

// Intensity levels (0–3).
type Intensity int

const (
	IntensityRest   Intensity = 0
	IntensityLow    Intensity = 1
	IntensityMedium Intensity = 2
	IntensityMax    Intensity = 3
)

// Orientation defines the court rendering rotation (0°, 90°, 180°, 270° clockwise).
type Orientation string

const (
	OrientationPortrait      Orientation = "portrait"       // 0° — basket at bottom (default)
	OrientationLandscape     Orientation = "landscape"      // 90° CW — basket at left
	OrientationPortraitFlip  Orientation = "portrait_flip"  // 180° — basket at top
	OrientationLandscapeFlip Orientation = "landscape_flip" // 270° CW — basket at right
)

// NextRotationCW returns the next orientation after a 90° clockwise rotation.
func NextRotationCW(o Orientation) Orientation {
	switch o {
	case OrientationLandscape:
		return OrientationPortraitFlip
	case OrientationPortraitFlip:
		return OrientationLandscapeFlip
	case OrientationLandscapeFlip:
		return OrientationPortrait
	default:
		return OrientationLandscape
	}
}

// IsLandscape returns true for 90° and 270° orientations (swapped dimensions).
func (o Orientation) IsLandscape() bool {
	return o == OrientationLandscape || o == OrientationLandscapeFlip
}

// CalloutType defines a predefined shout a player makes during a sequence.
type CalloutType string

const (
	CalloutBlock  CalloutType = "block"
	CalloutShoot  CalloutType = "shoot"
	CalloutHere   CalloutType = "here"
	CalloutScreen CalloutType = "screen"
	CalloutSwitch CalloutType = "switch"
	CalloutHelp   CalloutType = "help"
	CalloutBall   CalloutType = "ball"
	CalloutGo     CalloutType = "go"
)

// AllCallouts returns all callout types in cycle order.
func AllCallouts() []CalloutType {
	return []CalloutType{
		CalloutBlock, CalloutShoot, CalloutHere, CalloutScreen,
		CalloutSwitch, CalloutHelp, CalloutBall, CalloutGo,
	}
}

// Color palette from spec.
var (
	ColorAttack  = color.NRGBA{R: 0xe6, G: 0x39, B: 0x46, A: 0xff} // #e63946
	ColorDefense = color.NRGBA{R: 0x1d, G: 0x35, B: 0x57, A: 0xff} // #1d3557
	ColorCoach   = color.NRGBA{R: 0xf4, G: 0xa2, B: 0x61, A: 0xff} // #f4a261
	ColorNeutral = color.NRGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff} // #888888
)

// RoleColor returns the display color for a player role.
func RoleColor(role PlayerRole) color.NRGBA {
	switch role {
	case RoleDefender:
		return ColorDefense
	case RoleCoach:
		return ColorCoach
	case RoleAttacker, RolePointGuard, RoleShootingGuard,
		RoleSmallForward, RolePowerForward, RoleCenter:
		return ColorAttack
	default:
		return ColorNeutral
	}
}

// RequiresBall returns true if the action type requires the player to have the ball.
func RequiresBall(at ActionType) bool {
	n := NormalizeActionType(at)
	switch n {
	case ActionPass, ActionDribble, ActionShot, ActionHandoff:
		return true
	}
	return false
}

// ActionIssue represents a validation issue on an action.
type ActionIssue struct {
	IsError bool   // true = strong error (red), false = warning (orange)
	Message string // i18n key for the message
}

// ValidateActions checks action consistency across steps.
// Returns a map of action index → list of issues.
func ValidateActions(seq *Sequence) map[int][]ActionIssue {
	issues := make(map[int][]ActionIssue)

	// Track ball carriers progressively by step.
	carriers := make([]string, len(seq.BallCarrier))
	copy(carriers, seq.BallCarrier)

	// Build player ID set for orphan detection.
	playerIDs := make(map[string]bool, len(seq.Players))
	for i := range seq.Players {
		playerIDs[seq.Players[i].ID] = true
	}

	maxStep := MaxStep(seq)
	for step := 1; step <= maxStep; step++ {
		for i := range seq.Actions {
			if seq.Actions[i].EffectiveStep() != step {
				continue
			}
			act := &seq.Actions[i]
			nat := NormalizeActionType(act.Type)

			// --- Strong errors (red) ---

			// Ball required but player doesn't have it.
			if RequiresBall(nat) && act.From.IsPlayer {
				if !slices.Contains(carriers, act.From.PlayerID) {
					issues[i] = append(issues[i], ActionIssue{IsError: true, Message: "status.requires_ball"})
				}
			}

			// Pass/Handoff target must be a player.
			if (nat == ActionPass || nat == ActionHandoff) && !act.To.IsPlayer {
				issues[i] = append(issues[i], ActionIssue{IsError: true, Message: "status.pass_requires_player"})
			}

			// --- Warnings (orange) ---

			// Cut with ball → should probably be a Dribble.
			if nat == ActionCut && act.From.IsPlayer && slices.Contains(carriers, act.From.PlayerID) {
				issues[i] = append(issues[i], ActionIssue{IsError: false, Message: "validation.cut_with_ball"})
			}

			// Screen with ball → unusual.
			if nat == ActionScreen && act.From.IsPlayer && slices.Contains(carriers, act.From.PlayerID) {
				issues[i] = append(issues[i], ActionIssue{IsError: false, Message: "validation.screen_with_ball"})
			}

			// Orphan action: references a player that doesn't exist.
			if act.From.IsPlayer && !playerIDs[act.From.PlayerID] {
				issues[i] = append(issues[i], ActionIssue{IsError: false, Message: "validation.player_not_found"})
			}
			if act.To.IsPlayer && !playerIDs[act.To.PlayerID] {
				issues[i] = append(issues[i], ActionIssue{IsError: false, Message: "validation.player_not_found"})
			}
		}

		// Apply passes/handoffs at this step to update carriers for next step.
		for i := range seq.Actions {
			if seq.Actions[i].EffectiveStep() != step {
				continue
			}
			act := &seq.Actions[i]
			nat := NormalizeActionType(act.Type)
			if (nat == ActionPass || nat == ActionHandoff) && act.From.IsPlayer && act.To.IsPlayer {
				for j, c := range carriers {
					if c == act.From.PlayerID {
						carriers[j] = act.To.PlayerID
						break
					}
				}
			}
		}
	}
	return issues
}

// RoleLabel returns the short display label for a player role.
func RoleLabel(role PlayerRole) string {
	switch role {
	case RoleAttacker:
		return "A"
	case RoleDefender:
		return "D"
	case RoleCoach:
		return "C"
	case RolePointGuard:
		return "PG"
	case RoleShootingGuard:
		return "SG"
	case RoleSmallForward:
		return "SF"
	case RolePowerForward:
		return "PF"
	case RoleCenter:
		return "C"
	default:
		return "?"
	}
}
