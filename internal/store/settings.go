package store

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Settings holds application-level preferences.
type Settings struct {
	Language           string   `yaml:"language"`
	PdfExportDir       string   `yaml:"pdf_export_dir,omitempty"`
	GithubToken        string   `yaml:"github_token,omitempty"`
	ExerciseDir        string   `yaml:"exercise_dir,omitempty"`
	RecentFiles        []string `yaml:"recent_files,omitempty"`
	DismissedVersion   string   `yaml:"dismissed_version,omitempty"` // version update already seen/dismissed by the user
	DefaultCourtStandard string `yaml:"default_court_standard,omitempty"` // "fiba" or "nba"
	DefaultCourtType   string   `yaml:"default_court_type,omitempty"`   // "half_court" or "full_court"
	DefaultOrientation string   `yaml:"default_orientation,omitempty"` // "portrait" or "landscape"
	ShowApron          *bool    `yaml:"show_apron,omitempty"`          // nil = default (true)
}

// ApronVisible returns whether the apron bands should be shown (default: true).
func (s *Settings) ApronVisible() bool {
	if s.ShowApron == nil {
		return true
	}
	return *s.ShowApron
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

	// Decode base64-encoded token.
	if settings.GithubToken != "" {
		if decoded, err := base64.StdEncoding.DecodeString(settings.GithubToken); err == nil {
			settings.GithubToken = string(decoded)
		}
	}

	return &settings, nil
}

// SaveSettings writes settings to baseDir/settings.yaml.
func (s *YAMLStore) SaveSettings(settings *Settings) error {
	baseDir := filepath.Dir(s.exercisesDir)
	path := filepath.Join(baseDir, "settings.yaml")

	// Encode token as base64 before marshaling.
	toSave := *settings
	if toSave.GithubToken != "" {
		toSave.GithubToken = base64.StdEncoding.EncodeToString([]byte(toSave.GithubToken))
	}

	data, err := yaml.Marshal(&toSave)
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

// ExercisesDir returns the exercises directory path.
func (s *YAMLStore) ExercisesDir() string {
	return s.exercisesDir
}
