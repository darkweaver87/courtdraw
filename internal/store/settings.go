package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Settings holds application-level preferences.
type Settings struct {
	Language     string `yaml:"language"`
	PdfExportDir string `yaml:"pdf_export_dir,omitempty"`
}

// LoadSettings reads settings from baseDir/settings.yaml.
// Returns default settings if the file doesn't exist.
func (s *YAMLStore) LoadSettings() (*Settings, error) {
	baseDir := filepath.Dir(s.exercisesDir)
	path := filepath.Join(baseDir, "settings.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Settings{}, nil
		}
		return nil, fmt.Errorf("load settings: %w", err)
	}

	var settings Settings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parse settings: %w", err)
	}
	return &settings, nil
}

// SaveSettings writes settings to baseDir/settings.yaml.
func (s *YAMLStore) SaveSettings(settings *Settings) error {
	baseDir := filepath.Dir(s.exercisesDir)
	path := filepath.Join(baseDir, "settings.yaml")

	data, err := yaml.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}
