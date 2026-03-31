package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// PeriodFormat defines the match period structure.
type PeriodFormat string

const (
	PeriodFormat4x8  PeriodFormat = "4x8"
	PeriodFormat4x10 PeriodFormat = "4x10"
	PeriodFormat2x20 PeriodFormat = "2x20"
)

// EventType defines the type of match event.
type EventType string

const (
	EventSubIn       EventType = "sub_in"
	EventSubOut      EventType = "sub_out"
	EventFoul        EventType = "foul"
	EventScore       EventType = "score"
	EventPeriodStart EventType = "period_start"
	EventPeriodEnd   EventType = "period_end"
	EventTimeout     EventType = "timeout"
)

// MatchEvent represents a timestamped event during a match.
type MatchEvent struct {
	Type      EventType `yaml:"type"`
	Timestamp int       `yaml:"timestamp"`           // seconds from match start
	Period    int       `yaml:"period"`
	PlayerID  string    `yaml:"player_id,omitempty"`
	Points    int       `yaml:"points,omitempty"`     // 1, 2, or 3 for score events
	IsHome    bool      `yaml:"is_home,omitempty"`    // which team scored
}

// RosterEntry represents a player selected for a match.
type RosterEntry struct {
	MemberID  string `yaml:"member_id"`
	Number    int    `yaml:"number"`
	FirstName string `yaml:"first_name"`
	LastName  string `yaml:"last_name"`
	Starting  bool   `yaml:"starting,omitempty"`
}

// Match represents a basketball match with roster and live events.
type Match struct {
	ID           string       `yaml:"id,omitempty"`
	TeamName     string       `yaml:"team_name"`
	TeamFile     string       `yaml:"team_file,omitempty"`
	Opponent     string       `yaml:"opponent"`
	Date         string       `yaml:"date"`
	Time         string       `yaml:"time,omitempty"`
	Location     string       `yaml:"location,omitempty"`
	Competition  string       `yaml:"competition,omitempty"`
	HomeAway     string       `yaml:"home_away"`
	PeriodFormat PeriodFormat `yaml:"period_format"`
	Roster       []RosterEntry `yaml:"roster"`
	Events       []MatchEvent `yaml:"events,omitempty"`
	HomeScore    int          `yaml:"home_score"`
	AwayScore    int          `yaml:"away_score"`
	Status       string       `yaml:"status"` // "planned", "live", "finished"
}

// Validation errors for Match.
var (
	ErrMatchNoTeam          = errors.New("team name is required")
	ErrMatchNoOpponent      = errors.New("opponent is required")
	ErrMatchNoDate          = errors.New("date is required")
	ErrMatchNoFormat        = errors.New("period format is required")
	ErrMatchNoHomeAway      = errors.New("home/away is required")
	ErrMatchDuplicateNumber = errors.New("duplicate jersey number in roster")
)

// Validate checks match data integrity.
func (m *Match) Validate() error {
	if strings.TrimSpace(m.TeamName) == "" {
		return ErrMatchNoTeam
	}
	if strings.TrimSpace(m.Opponent) == "" {
		return ErrMatchNoOpponent
	}
	if strings.TrimSpace(m.Date) == "" {
		return ErrMatchNoDate
	}
	if m.PeriodFormat == "" {
		return ErrMatchNoFormat
	}
	if m.HomeAway != "home" && m.HomeAway != "away" {
		return ErrMatchNoHomeAway
	}
	return nil
}

// StartingFive returns roster entries marked as starting.
func (m *Match) StartingFive() []RosterEntry {
	var starters []RosterEntry
	for _, r := range m.Roster {
		if r.Starting {
			starters = append(starters, r)
		}
	}
	return starters
}

// PlayerFouls counts foul events for a given member ID.
func (m *Match) PlayerFouls(memberID string) int {
	count := 0
	for _, e := range m.Events {
		if e.Type == EventFoul && e.PlayerID == memberID {
			count++
		}
	}
	return count
}

// PlayerPlayingSeconds computes total on-court seconds for a player up to currentTimestamp.
// It reconstructs intervals from sub_in/sub_out events, treating starting players as on-court from timestamp 0.
func (m *Match) PlayerPlayingSeconds(memberID string, currentTimestamp int) int {
	// Determine if player is in starting five.
	onCourt := false
	for _, r := range m.Roster {
		if r.MemberID == memberID && r.Starting {
			onCourt = true
			break
		}
	}

	total := 0
	lastIn := 0
	if !onCourt {
		lastIn = -1 // not on court
	}

	for _, e := range m.Events {
		if e.Timestamp > currentTimestamp {
			break
		}
		if e.PlayerID != memberID {
			continue
		}
		switch e.Type {
		case EventSubIn:
			if !onCourt {
				onCourt = true
				lastIn = e.Timestamp
			}
		case EventSubOut:
			if onCourt {
				total += e.Timestamp - lastIn
				onCourt = false
				lastIn = -1
			}
		}
	}

	// If still on court, count up to currentTimestamp.
	if onCourt && lastIn >= 0 {
		total += currentTimestamp - lastIn
	}
	return total
}

// CurrentLineup returns the member IDs currently on court by applying all sub events to the starting five.
func (m *Match) CurrentLineup() []string {
	onCourt := make(map[string]bool)
	for _, r := range m.Roster {
		if r.Starting {
			onCourt[r.MemberID] = true
		}
	}
	for _, e := range m.Events {
		switch e.Type {
		case EventSubIn:
			onCourt[e.PlayerID] = true
		case EventSubOut:
			delete(onCourt, e.PlayerID)
		}
	}
	lineup := make([]string, 0, len(onCourt))
	for id := range onCourt {
		lineup = append(lineup, id)
	}
	return lineup
}

// PeriodDurationMinutes returns the duration of each period in minutes.
func (m *Match) PeriodDurationMinutes() int {
	switch m.PeriodFormat {
	case PeriodFormat4x8:
		return 8
	case PeriodFormat4x10:
		return 10
	case PeriodFormat2x20:
		return 20
	default:
		return 10
	}
}

// TotalPeriods returns the number of periods in the match.
func (m *Match) TotalPeriods() int {
	switch m.PeriodFormat {
	case PeriodFormat4x8:
		return 4
	case PeriodFormat4x10:
		return 4
	case PeriodFormat2x20:
		return 2
	default:
		return 4
	}
}

// AddEvent appends a match event.
func (m *Match) AddEvent(evt MatchEvent) {
	m.Events = append(m.Events, evt)
}

// Substitute adds a sub_in and sub_out event pair.
func (m *Match) Substitute(playerInID, playerOutID string, timestamp, period int) {
	m.Events = append(m.Events,
		MatchEvent{Type: EventSubOut, Timestamp: timestamp, Period: period, PlayerID: playerOutID},
		MatchEvent{Type: EventSubIn, Timestamp: timestamp, Period: period, PlayerID: playerInID},
	)
}

// PlayerScorePoints returns the total points scored by a player across all events.
func (m *Match) PlayerScorePoints(memberID string) int {
	total := 0
	for _, e := range m.Events {
		if e.Type == EventScore && e.PlayerID == memberID {
			total += e.Points
		}
	}
	return total
}

// PeriodScores returns the score breakdown per period as a map: period → [home, away].
func (m *Match) PeriodScores() map[int][2]int {
	scores := make(map[int][2]int)
	for _, e := range m.Events {
		if e.Type != EventScore {
			continue
		}
		s := scores[e.Period]
		if e.IsHome {
			s[0] += e.Points
		} else {
			s[1] += e.Points
		}
		scores[e.Period] = s
	}
	return scores
}

// PeriodScoresText returns a formatted string of scores per period: "P1 12-45 | P2 8-42".
func (m *Match) PeriodScoresText() string {
	ps := m.PeriodScores()
	if len(ps) == 0 {
		return ""
	}
	// Find max period.
	maxP := 0
	for p := range ps {
		if p > maxP {
			maxP = p
		}
	}
	var parts []string
	for p := 1; p <= maxP; p++ {
		s := ps[p]
		parts = append(parts, fmt.Sprintf("P%d %d-%d", p, s[0], s[1]))
	}
	return strings.Join(parts, " | ")
}

// ValidateRosterNumbers checks for duplicate jersey numbers in the roster.
// Returns an error with the duplicated number if found.
func (m *Match) ValidateRosterNumbers() error {
	seen := make(map[int]string, len(m.Roster))
	for _, r := range m.Roster {
		if r.Number == 0 {
			continue // unnumbered players are allowed
		}
		if prev, ok := seen[r.Number]; ok {
			return fmt.Errorf("%w: #%d (%s and %s)", ErrMatchDuplicateNumber, r.Number, prev, r.FirstName+" "+r.LastName)
		}
		seen[r.Number] = r.FirstName + " " + r.LastName
	}
	return nil
}

// SortRoster sorts roster entries by number first, then alphabetically by last name.
// Players with number 0 are sorted after numbered players.
func (m *Match) SortRoster() {
	sort.Slice(m.Roster, func(i, j int) bool {
		ri, rj := m.Roster[i], m.Roster[j]
		// Both have numbers — sort by number.
		if ri.Number != 0 && rj.Number != 0 {
			if ri.Number != rj.Number {
				return ri.Number < rj.Number
			}
			return ri.LastName < rj.LastName
		}
		// One has number, the other doesn't — numbered first.
		if ri.Number != rj.Number {
			return ri.Number != 0
		}
		// Neither has number — sort by last name.
		return ri.LastName < rj.LastName
	})
}
