package model

import "testing"

func TestTeam_Validate_Valid(t *testing.T) {
	team := &Team{
		Name: "U15 Boys",
		Members: []Member{
			{ID: "m1", FirstName: "Jean", LastName: "Dupont", Role: MemberRolePlayer},
			{ID: "m2", FirstName: "Pierre", LastName: "Martin", Role: MemberRolePlayer},
		},
	}
	if err := team.Validate(); err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}
}

func TestTeam_Validate_NoName(t *testing.T) {
	team := &Team{Name: ""}
	if err := team.Validate(); err != ErrTeamNoName {
		t.Fatalf("expected ErrTeamNoName, got: %v", err)
	}
}

func TestTeam_Validate_DuplicateID(t *testing.T) {
	team := &Team{
		Name: "U15",
		Members: []Member{
			{ID: "m1", FirstName: "Jean", Role: MemberRolePlayer},
			{ID: "m1", FirstName: "Pierre", Role: MemberRolePlayer},
		},
	}
	err := team.Validate()
	if err == nil {
		t.Fatal("expected error for duplicate member ID")
	}
}

func TestTeam_Players(t *testing.T) {
	team := &Team{
		Name: "Test",
		Members: []Member{
			{ID: "m1", FirstName: "Coach", Role: MemberRoleCoach},
			{ID: "m2", FirstName: "B", Number: 7, Role: MemberRolePlayer},
			{ID: "m3", FirstName: "A", Number: 3, Role: MemberRolePlayer},
			{ID: "m4", FirstName: "Asst", Role: MemberRoleAssistant},
		},
	}
	players := team.Players()
	if len(players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(players))
	}
	// Should be sorted by number: 3, 7
	if players[0].Number != 3 {
		t.Fatalf("expected first player #3, got #%d", players[0].Number)
	}
	if players[1].Number != 7 {
		t.Fatalf("expected second player #7, got #%d", players[1].Number)
	}
}

func TestTeam_Staff(t *testing.T) {
	team := &Team{
		Name: "Test",
		Members: []Member{
			{ID: "m1", FirstName: "Coach", Role: MemberRoleCoach},
			{ID: "m2", FirstName: "Player", Role: MemberRolePlayer},
			{ID: "m3", FirstName: "Asst", Role: MemberRoleAssistant},
		},
	}
	staff := team.Staff()
	if len(staff) != 2 {
		t.Fatalf("expected 2 staff, got %d", len(staff))
	}
}

func TestMember_FullName(t *testing.T) {
	m := Member{FirstName: "Jean", LastName: "Dupont"}
	if got := m.FullName(); got != "Jean Dupont" {
		t.Fatalf("expected 'Jean Dupont', got %q", got)
	}
}

func TestMember_FullName_FirstOnly(t *testing.T) {
	m := Member{FirstName: "Jean"}
	if got := m.FullName(); got != "Jean" {
		t.Fatalf("expected 'Jean', got %q", got)
	}
}

func TestMember_DisplayLabel(t *testing.T) {
	m := Member{FirstName: "Jean", Number: 7}
	if got := m.DisplayLabel(); got != "#7 Jean" {
		t.Fatalf("expected '#7 Jean', got %q", got)
	}
}

func TestMember_DisplayLabel_NoNumber(t *testing.T) {
	m := Member{FirstName: "Jean"}
	if got := m.DisplayLabel(); got != "Jean" {
		t.Fatalf("expected 'Jean', got %q", got)
	}
}

func TestMember_AgeCategory(t *testing.T) {
	tests := []struct {
		birthYear   int
		currentYear int
		want        string
	}{
		{2018, 2026, "U9"},
		{2016, 2026, "U11"},
		{2014, 2026, "U13"},
		{2012, 2026, "U15"},
		{2010, 2026, "U17"},
		{2008, 2026, "U19"},
		{2000, 2026, "Senior"},
		{0, 2026, ""},
	}
	for _, tt := range tests {
		m := Member{BirthYear: tt.birthYear}
		got := m.AgeCategory(tt.currentYear)
		if got != tt.want {
			t.Errorf("AgeCategory(birth=%d, current=%d) = %q, want %q", tt.birthYear, tt.currentYear, got, tt.want)
		}
	}
}

func TestTeam_NextMemberID(t *testing.T) {
	team := &Team{Name: "Test"}
	if got := team.NextMemberID(); got != "m1" {
		t.Fatalf("expected m1, got %s", got)
	}

	team.Members = []Member{
		{ID: "m1"}, {ID: "m3"}, {ID: "m2"},
	}
	if got := team.NextMemberID(); got != "m4" {
		t.Fatalf("expected m4, got %s", got)
	}
}
