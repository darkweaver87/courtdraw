package model

import (
	"image/color"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestValidate_Valid(t *testing.T) {
	e := &Exercise{
		Name:          "Test",
		CourtType:     HalfCourt,
		CourtStandard: FIBA,
		Sequences: []Sequence{
			{Players: []Player{{ID: "p1", Role: RoleAttacker, Position: Position{0.5, 0.5}}}},
		},
	}
	if err := e.Validate(); err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}
}

func TestValidate_NoName(t *testing.T) {
	e := &Exercise{
		CourtType:     HalfCourt,
		CourtStandard: FIBA,
		Sequences:     []Sequence{{}},
	}
	if err := e.Validate(); err != ErrNoName {
		t.Fatalf("expected ErrNoName, got: %v", err)
	}
}

func TestValidate_NoSequences(t *testing.T) {
	e := &Exercise{
		Name:          "Test",
		CourtType:     HalfCourt,
		CourtStandard: FIBA,
	}
	if err := e.Validate(); err != ErrNoSequences {
		t.Fatalf("expected ErrNoSequences, got: %v", err)
	}
}

func TestValidate_NoCourtType(t *testing.T) {
	e := &Exercise{
		Name:          "Test",
		CourtStandard: FIBA,
		Sequences:     []Sequence{{}},
	}
	if err := e.Validate(); err != ErrNoCourtType {
		t.Fatalf("expected ErrNoCourtType, got: %v", err)
	}
}

func TestValidate_NoStandard(t *testing.T) {
	e := &Exercise{
		Name:      "Test",
		CourtType: HalfCourt,
		Sequences: []Sequence{{}},
	}
	if err := e.Validate(); err != ErrNoStandard {
		t.Fatalf("expected ErrNoStandard, got: %v", err)
	}
}

func TestValidate_PlayerNoID(t *testing.T) {
	e := &Exercise{
		Name:          "Test",
		CourtType:     HalfCourt,
		CourtStandard: FIBA,
		Sequences: []Sequence{
			{Players: []Player{{Role: RoleAttacker, Position: Position{0.5, 0.5}}}},
		},
	}
	err := e.Validate()
	if err == nil {
		t.Fatal("expected error for player with no ID")
	}
}

func TestActionRef_UnmarshalString(t *testing.T) {
	data := `"player1"`
	var ref ActionRef
	if err := yaml.Unmarshal([]byte(data), &ref); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !ref.IsPlayer || ref.PlayerID != "player1" {
		t.Fatalf("expected player ref 'player1', got: %+v", ref)
	}
}

func TestActionRef_UnmarshalPosition(t *testing.T) {
	data := `[0.3, 0.7]`
	var ref ActionRef
	if err := yaml.Unmarshal([]byte(data), &ref); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ref.IsPlayer {
		t.Fatal("expected position ref, got player ref")
	}
	if ref.Position[0] != 0.3 || ref.Position[1] != 0.7 {
		t.Fatalf("expected [0.3, 0.7], got: %v", ref.Position)
	}
}

func TestActionRef_MarshalPlayer(t *testing.T) {
	ref := ActionRef{PlayerID: "p1", IsPlayer: true}
	data, err := yaml.Marshal(ref)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	expected := "p1\n"
	if string(data) != expected {
		t.Fatalf("expected %q, got %q", expected, string(data))
	}
}

func TestActionRef_MarshalPosition(t *testing.T) {
	ref := ActionRef{Position: Position{0.5, 0.8}, IsPlayer: false}
	data, err := yaml.Marshal(ref)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got []float64
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal roundtrip: %v", err)
	}
	if len(got) != 2 || got[0] != 0.5 || got[1] != 0.8 {
		t.Fatalf("expected [0.5, 0.8], got: %v", got)
	}
}

func TestActionRef_RoundTrip(t *testing.T) {
	action := Action{
		Type: ActionPass,
		From: ActionRef{PlayerID: "a1", IsPlayer: true},
		To:   ActionRef{Position: Position{0.5, 0.9}, IsPlayer: false},
	}
	data, err := yaml.Marshal(action)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Action
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Type != ActionPass {
		t.Fatalf("expected pass, got %s", got.Type)
	}
	if !got.From.IsPlayer || got.From.PlayerID != "a1" {
		t.Fatalf("expected player ref a1, got: %+v", got.From)
	}
	if got.To.IsPlayer || got.To.Position[0] != 0.5 || got.To.Position[1] != 0.9 {
		t.Fatalf("expected position [0.5, 0.9], got: %+v", got.To)
	}
}

func TestRoleColor(t *testing.T) {
	tests := []struct {
		role PlayerRole
		want color.NRGBA
	}{
		{RoleAttacker, ColorAttack},
		{RoleDefender, ColorDefense},
		{RoleCoach, ColorCoach},
		{RolePointGuard, ColorAttack},
		{RoleCenter, ColorAttack},
		{PlayerRole("unknown"), ColorNeutral},
	}
	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			got := RoleColor(tt.role)
			if got != tt.want {
				t.Fatalf("RoleColor(%s) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestRoleLabel(t *testing.T) {
	if l := RoleLabel(RoleAttacker); l != "A" {
		t.Fatalf("expected A, got %s", l)
	}
	if l := RoleLabel(RoleDefender); l != "D" {
		t.Fatalf("expected D, got %s", l)
	}
	if l := RoleLabel(RolePointGuard); l != "PG" {
		t.Fatalf("expected PG, got %s", l)
	}
}

func TestExercise_FullRoundTrip(t *testing.T) {
	ex := Exercise{
		Name:          "Test Drill",
		CourtType:     HalfCourt,
		CourtStandard: FIBA,
		Duration:      "10m",
		Intensity:     IntensityMedium,
		Category:      CategoryOffense,
		Tags:          []string{"shooting", "passing"},
		Sequences: []Sequence{
			{
				Label:        "Setup",
				Instructions: []string{"Line up at free throw line"},
				Players: []Player{
					{ID: "a1", Label: "A1", Role: RoleAttacker, Position: Position{0.3, 0.5}},
					{ID: "d1", Role: RoleDefender, Position: Position{0.7, 0.5}},
				},
				Actions: []Action{
					{Type: ActionPass, From: ActionRef{PlayerID: "a1", IsPlayer: true}, To: ActionRef{PlayerID: "d1", IsPlayer: true}},
				},
				Accessories: []Accessory{
					{Type: AccessoryCone, ID: "c1", Position: Position{0.5, 0.3}},
				},
			},
		},
	}

	data, err := yaml.Marshal(ex)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Exercise
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Name != ex.Name {
		t.Fatalf("name: %s != %s", got.Name, ex.Name)
	}
	if len(got.Sequences) != 1 {
		t.Fatalf("sequences: %d != 1", len(got.Sequences))
	}
	if len(got.Sequences[0].Players) != 2 {
		t.Fatalf("players: %d != 2", len(got.Sequences[0].Players))
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("validate round-tripped: %v", err)
	}
}
