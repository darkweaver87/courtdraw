package store

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// YAMLStore implements Store backed by YAML files on disk.
type YAMLStore struct {
	exercisesDir string
	sessionsDir  string
}

// NewYAMLStore creates a YAMLStore rooted at baseDir.
// Creates subdirectories if they don't exist.
func NewYAMLStore(baseDir string) (*YAMLStore, error) {
	exDir := filepath.Join(baseDir, "exercises")
	sesDir := filepath.Join(baseDir, "sessions")
	for _, d := range []string{exDir, sesDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("create dir %s: %w", d, err)
		}
	}
	return &YAMLStore{exercisesDir: exDir, sessionsDir: sesDir}, nil
}

var kebabRe = regexp.MustCompile(`[^a-z0-9]+`)

// ToKebab converts a name to a kebab-case filename (without extension).
func ToKebab(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = kebabRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func (s *YAMLStore) ListExercises() ([]string, error) {
	return listYAML(s.exercisesDir)
}

func (s *YAMLStore) LoadExercise(name string) (*model.Exercise, error) {
	path := filepath.Join(s.exercisesDir, name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load exercise %s: %w", name, err)
	}
	var ex model.Exercise
	if err := yaml.Unmarshal(data, &ex); err != nil {
		return nil, fmt.Errorf("parse exercise %s: %w", name, err)
	}
	return &ex, nil
}

func (s *YAMLStore) SaveExercise(exercise *model.Exercise) error {
	name := ToKebab(exercise.Name)
	if name == "" {
		return fmt.Errorf("exercise name is empty")
	}
	path := filepath.Join(s.exercisesDir, name+".yaml")
	data, err := yaml.Marshal(exercise)
	if err != nil {
		return fmt.Errorf("marshal exercise: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func (s *YAMLStore) DeleteExercise(name string) error {
	path := filepath.Join(s.exercisesDir, name+".yaml")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete exercise %s: %w", name, err)
	}
	return nil
}

func (s *YAMLStore) ListSessions() ([]string, error) {
	return listYAML(s.sessionsDir)
}

func (s *YAMLStore) LoadSession(name string) (*model.Session, error) {
	path := filepath.Join(s.sessionsDir, name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load session %s: %w", name, err)
	}
	var ses model.Session
	if err := yaml.Unmarshal(data, &ses); err != nil {
		return nil, fmt.Errorf("parse session %s: %w", name, err)
	}
	return &ses, nil
}

func (s *YAMLStore) SaveSession(session *model.Session) error {
	name := ToKebab(session.Title)
	if name == "" {
		return fmt.Errorf("session title is empty")
	}
	path := filepath.Join(s.sessionsDir, name+".yaml")
	data, err := yaml.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// listYAML returns the base names (without .yaml) of all YAML files in a directory.
func listYAML(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", dir, err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			names = append(names, strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml"))
		}
	}
	return names, nil
}
