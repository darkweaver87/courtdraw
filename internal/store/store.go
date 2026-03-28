package store

import "github.com/darkweaver87/courtdraw/internal/model"

// Store defines the persistence interface for exercises, sessions, teams, and matches.
type Store interface {
	ListExercises() ([]string, error)
	LoadExercise(name string) (*model.Exercise, error)
	SaveExercise(exercise *model.Exercise) error
	SaveExerciseAs(name string, exercise *model.Exercise) error
	DeleteExercise(name string) error

	ListSessions() ([]string, error)
	LoadSession(name string) (*model.Session, error)
	SaveSession(session *model.Session) error
	DeleteSession(name string) error

	ListTeams() ([]TeamIndexEntry, error)
	LoadTeam(name string) (*model.Team, error)
	SaveTeam(team *model.Team) error
	DeleteTeam(name string) error

	ListMatches() ([]MatchIndexEntry, error)
	LoadMatch(name string) (*model.Match, error)
	SaveMatch(match *model.Match) error
	DeleteMatch(name string) error
}
