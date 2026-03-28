package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// MemberRole defines the role of a team member.
type MemberRole string

const (
	MemberRolePlayer    MemberRole = "player"
	MemberRoleCoach     MemberRole = "coach"
	MemberRoleAssistant MemberRole = "assistant"
)

// LicenseType defines the type of federation license.
type LicenseType string

const (
	LicenseCompetition LicenseType = "competition"
	LicenseLoisir      LicenseType = "loisir"
	LicenseMiniBask    LicenseType = "mini_bask"
)

// Member represents a team member (player or staff).
type Member struct {
	ID            string      `yaml:"id"`
	FirstName     string      `yaml:"first_name"`
	LastName      string      `yaml:"last_name"`
	Number        int         `yaml:"number,omitempty"`
	LicenseNumber string      `yaml:"license_number,omitempty"`
	LicenseType   LicenseType `yaml:"license_type,omitempty"`
	BirthDate     string      `yaml:"birth_date,omitempty"` // "YYYY-MM-DD"
	BirthYear     int         `yaml:"birth_year,omitempty"` // kept for backward compat
	Role          MemberRole  `yaml:"role"`
	Position      string      `yaml:"position,omitempty"` // pg, sg, sf, pf, c
	Email         string      `yaml:"email,omitempty"`
	Phone         string      `yaml:"phone,omitempty"`
}

// Team represents a basketball team roster.
type Team struct {
	Name    string   `yaml:"name"`
	Club    string   `yaml:"club,omitempty"`
	Season  string   `yaml:"season,omitempty"` // e.g. "2025-2026"
	Members []Member `yaml:"members"`
}

// Validation errors for Team.
var (
	ErrTeamNoName         = errors.New("team name is required")
	ErrTeamDuplicateMember = errors.New("duplicate member ID")
)

// Players returns members with role "player", sorted by number.
func (t *Team) Players() []Member {
	var players []Member
	for _, m := range t.Members {
		if m.Role == MemberRolePlayer {
			players = append(players, m)
		}
	}
	sort.Slice(players, func(i, j int) bool {
		return players[i].Number < players[j].Number
	})
	return players
}

// Staff returns members with role "coach" or "assistant".
func (t *Team) Staff() []Member {
	var staff []Member
	for _, m := range t.Members {
		if m.Role == MemberRoleCoach || m.Role == MemberRoleAssistant {
			staff = append(staff, m)
		}
	}
	return staff
}

// FullName returns "FirstName LastName".
func (m *Member) FullName() string {
	return strings.TrimSpace(m.FirstName + " " + m.LastName)
}

// DisplayLabel returns "#7 Jean" (number + first name).
func (m *Member) DisplayLabel() string {
	if m.Number > 0 {
		return fmt.Sprintf("#%d %s", m.Number, m.FirstName)
	}
	return m.FirstName
}

// BirthYearEffective returns the birth year from BirthDate or BirthYear.
func (m *Member) BirthYearEffective() int {
	if m.BirthDate != "" && len(m.BirthDate) >= 4 {
		var y int
		if _, err := fmt.Sscanf(m.BirthDate[:4], "%d", &y); err == nil && y > 0 {
			return y
		}
	}
	return m.BirthYear
}

// AgeCategory computes the age category from the member's birth year.
// Returns "U9", "U11", "U13", "U15", "U17", "U19", or "Senior".
func (m *Member) AgeCategory(currentYear int) string {
	by := m.BirthYearEffective()
	if by <= 0 {
		return ""
	}
	age := currentYear - by
	switch {
	case age < 9:
		return "U9"
	case age < 11:
		return "U11"
	case age < 13:
		return "U13"
	case age < 15:
		return "U15"
	case age < 17:
		return "U17"
	case age < 19:
		return "U19"
	default:
		return "Senior"
	}
}

// Validate checks team data integrity: name required, unique member IDs.
func (t *Team) Validate() error {
	if strings.TrimSpace(t.Name) == "" {
		return ErrTeamNoName
	}
	seen := make(map[string]bool, len(t.Members))
	for _, m := range t.Members {
		if m.ID == "" {
			continue
		}
		if seen[m.ID] {
			return fmt.Errorf("%w: %s", ErrTeamDuplicateMember, m.ID)
		}
		seen[m.ID] = true
	}
	return nil
}

// NextMemberID generates the next available member ID ("m1", "m2", etc.).
func (t *Team) NextMemberID() string {
	maxN := 0
	for _, m := range t.Members {
		if strings.HasPrefix(m.ID, "m") {
			var n int
			if _, err := fmt.Sscanf(m.ID, "m%d", &n); err == nil && n > maxN {
				maxN = n
			}
		}
	}
	return fmt.Sprintf("m%d", maxN+1)
}
