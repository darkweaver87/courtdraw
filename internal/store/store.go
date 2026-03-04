package store

import "github.com/darkweaver87/courtdraw/internal/model"

// Store defines the persistence interface for exercises and sessions.
type Store interface {
	ListExercises() ([]string, error)
	LoadExercise(name string) (*model.Exercise, error)
	SaveExercise(exercise *model.Exercise) error
	DeleteExercise(name string) error

	ListSessions() ([]string, error)
	LoadSession(name string) (*model.Session, error)
	SaveSession(session *model.Session) error
	DeleteSession(name string) error
}
