package model

import "testing"

func testMatch() *Match {
	return &Match{
		TeamName:     "U15 Boys",
		Opponent:     "Villeurbanne",
		Date:         "2026-03-27",
		HomeAway:     "home",
		PeriodFormat: PeriodFormat4x8,
		Status:       "planned",
		Roster: []RosterEntry{
			{MemberID: "m1", Number: 4, FirstName: "Jean", LastName: "Dupont", Starting: true},
			{MemberID: "m2", Number: 7, FirstName: "Pierre", LastName: "Martin", Starting: true},
			{MemberID: "m3", Number: 10, FirstName: "Luc", LastName: "Bernard", Starting: true},
			{MemberID: "m4", Number: 11, FirstName: "Marc", LastName: "Petit", Starting: true},
			{MemberID: "m5", Number: 15, FirstName: "Paul", LastName: "Durand", Starting: true},
			{MemberID: "m6", Number: 23, FirstName: "Hugo", LastName: "Leroy", Starting: false},
			{MemberID: "m7", Number: 33, FirstName: "Tom", LastName: "Moreau", Starting: false},
		},
	}
}

func TestMatch_Validate_Valid(t *testing.T) {
	m := testMatch()
	if err := m.Validate(); err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}
}

func TestMatch_Validate_NoTeam(t *testing.T) {
	m := testMatch()
	m.TeamName = ""
	if err := m.Validate(); err != ErrMatchNoTeam {
		t.Fatalf("expected ErrMatchNoTeam, got: %v", err)
	}
}

func TestMatch_Validate_NoOpponent(t *testing.T) {
	m := testMatch()
	m.Opponent = ""
	if err := m.Validate(); err != ErrMatchNoOpponent {
		t.Fatalf("expected ErrMatchNoOpponent, got: %v", err)
	}
}

func TestMatch_Validate_NoDate(t *testing.T) {
	m := testMatch()
	m.Date = ""
	if err := m.Validate(); err != ErrMatchNoDate {
		t.Fatalf("expected ErrMatchNoDate, got: %v", err)
	}
}

func TestMatch_Validate_NoFormat(t *testing.T) {
	m := testMatch()
	m.PeriodFormat = ""
	if err := m.Validate(); err != ErrMatchNoFormat {
		t.Fatalf("expected ErrMatchNoFormat, got: %v", err)
	}
}

func TestMatch_Validate_NoHomeAway(t *testing.T) {
	m := testMatch()
	m.HomeAway = "invalid"
	if err := m.Validate(); err != ErrMatchNoHomeAway {
		t.Fatalf("expected ErrMatchNoHomeAway, got: %v", err)
	}
}

func TestMatch_StartingFive(t *testing.T) {
	m := testMatch()
	starters := m.StartingFive()
	if len(starters) != 5 {
		t.Fatalf("expected 5 starters, got %d", len(starters))
	}
}

func TestMatch_PlayerFouls(t *testing.T) {
	m := testMatch()
	m.Events = []MatchEvent{
		{Type: EventFoul, PlayerID: "m1", Timestamp: 60, Period: 1},
		{Type: EventFoul, PlayerID: "m1", Timestamp: 120, Period: 1},
		{Type: EventFoul, PlayerID: "m2", Timestamp: 90, Period: 1},
	}
	if got := m.PlayerFouls("m1"); got != 2 {
		t.Fatalf("expected 2 fouls for m1, got %d", got)
	}
	if got := m.PlayerFouls("m2"); got != 1 {
		t.Fatalf("expected 1 foul for m2, got %d", got)
	}
	if got := m.PlayerFouls("m3"); got != 0 {
		t.Fatalf("expected 0 fouls for m3, got %d", got)
	}
}

func TestMatch_PlayerPlayingSeconds_Starter(t *testing.T) {
	m := testMatch()
	// m1 is a starter, no sub events — should be on court the whole time.
	got := m.PlayerPlayingSeconds("m1", 480)
	if got != 480 {
		t.Fatalf("expected 480s for starter m1, got %d", got)
	}
}

func TestMatch_PlayerPlayingSeconds_SubbedOut(t *testing.T) {
	m := testMatch()
	m.Events = []MatchEvent{
		{Type: EventSubOut, PlayerID: "m1", Timestamp: 200, Period: 1},
		{Type: EventSubIn, PlayerID: "m6", Timestamp: 200, Period: 1},
	}
	got := m.PlayerPlayingSeconds("m1", 480)
	if got != 200 {
		t.Fatalf("expected 200s for m1 (subbed out at 200), got %d", got)
	}
}

func TestMatch_PlayerPlayingSeconds_SubbedInAndOut(t *testing.T) {
	m := testMatch()
	// m6 is not a starter, enters at 100, leaves at 300
	m.Events = []MatchEvent{
		{Type: EventSubIn, PlayerID: "m6", Timestamp: 100, Period: 1},
		{Type: EventSubOut, PlayerID: "m6", Timestamp: 300, Period: 1},
	}
	got := m.PlayerPlayingSeconds("m6", 480)
	if got != 200 {
		t.Fatalf("expected 200s for m6, got %d", got)
	}
}

func TestMatch_PlayerPlayingSeconds_StillOnCourt(t *testing.T) {
	m := testMatch()
	// m6 enters at 100, still on court at timestamp 480
	m.Events = []MatchEvent{
		{Type: EventSubIn, PlayerID: "m6", Timestamp: 100, Period: 1},
	}
	got := m.PlayerPlayingSeconds("m6", 480)
	if got != 380 {
		t.Fatalf("expected 380s for m6, got %d", got)
	}
}

func TestMatch_CurrentLineup(t *testing.T) {
	m := testMatch()
	m.Events = []MatchEvent{
		{Type: EventSubOut, PlayerID: "m1", Timestamp: 200, Period: 1},
		{Type: EventSubIn, PlayerID: "m6", Timestamp: 200, Period: 1},
	}
	lineup := m.CurrentLineup()
	if len(lineup) != 5 {
		t.Fatalf("expected 5 on court, got %d", len(lineup))
	}
	// m1 should be out, m6 should be in
	lineupMap := make(map[string]bool)
	for _, id := range lineup {
		lineupMap[id] = true
	}
	if lineupMap["m1"] {
		t.Fatal("m1 should not be on court after sub out")
	}
	if !lineupMap["m6"] {
		t.Fatal("m6 should be on court after sub in")
	}
}

func TestMatch_Substitute(t *testing.T) {
	m := testMatch()
	m.Substitute("m6", "m1", 200, 1)
	if len(m.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(m.Events))
	}
	if m.Events[0].Type != EventSubOut || m.Events[0].PlayerID != "m1" {
		t.Fatalf("expected sub_out for m1, got %+v", m.Events[0])
	}
	if m.Events[1].Type != EventSubIn || m.Events[1].PlayerID != "m6" {
		t.Fatalf("expected sub_in for m6, got %+v", m.Events[1])
	}
}

func TestMatch_PeriodDurationMinutes(t *testing.T) {
	tests := []struct {
		format PeriodFormat
		want   int
	}{
		{PeriodFormat4x8, 8},
		{PeriodFormat4x10, 10},
		{PeriodFormat2x20, 20},
		{PeriodFormat("unknown"), 10},
	}
	for _, tt := range tests {
		m := &Match{PeriodFormat: tt.format}
		if got := m.PeriodDurationMinutes(); got != tt.want {
			t.Errorf("PeriodDurationMinutes(%s) = %d, want %d", tt.format, got, tt.want)
		}
	}
}

func TestMatch_TotalPeriods(t *testing.T) {
	tests := []struct {
		format PeriodFormat
		want   int
	}{
		{PeriodFormat4x8, 4},
		{PeriodFormat4x10, 4},
		{PeriodFormat2x20, 2},
		{PeriodFormat("unknown"), 4},
	}
	for _, tt := range tests {
		m := &Match{PeriodFormat: tt.format}
		if got := m.TotalPeriods(); got != tt.want {
			t.Errorf("TotalPeriods(%s) = %d, want %d", tt.format, got, tt.want)
		}
	}
}
