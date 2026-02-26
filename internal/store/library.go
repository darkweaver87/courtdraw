package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// Library provides read-only access to the community exercise collection.
type Library struct {
	dir string
}

// NewLibrary creates a Library from a directory path.
func NewLibrary(dir string) *Library {
	return &Library{dir: dir}
}

// ListExercises returns the names of available community exercises.
func (l *Library) ListExercises() ([]string, error) {
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list library: %w", err)
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

// LoadExercise loads a community exercise by name.
func (l *Library) LoadExercise(name string) (*model.Exercise, error) {
	path := filepath.Join(l.dir, name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load library exercise %s: %w", name, err)
	}
	var ex model.Exercise
	if err := yaml.Unmarshal(data, &ex); err != nil {
		return nil, fmt.Errorf("parse library exercise %s: %w", name, err)
	}
	return &ex, nil
}
