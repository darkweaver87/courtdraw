package model

import "image/color"

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
	ActionPass       ActionType = "pass"
	ActionDribble    ActionType = "dribble"
	ActionSprint     ActionType = "sprint"
	ActionShotLayup  ActionType = "shot_layup"
	ActionShotPushup ActionType = "shot_pushup"
	ActionShotJump   ActionType = "shot_jumpshot"
	ActionScreen     ActionType = "screen"
	ActionCut        ActionType = "cut"
	ActionCloseOut   ActionType = "close_out"
	ActionContest    ActionType = "contest"
	ActionReverse    ActionType = "reverse"
)

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

// Intensity levels (0–3).
type Intensity int

const (
	IntensityRest   Intensity = 0
	IntensityLow    Intensity = 1
	IntensityMedium Intensity = 2
	IntensityMax    Intensity = 3
)

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
