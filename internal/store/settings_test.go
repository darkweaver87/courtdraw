package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSettings_DefaultsWhenNoFile(t *testing.T) {
	dir := t.TempDir()
	s, err := NewYAMLStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	settings, err := s.LoadSettings()
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if settings.Language != "" {
		t.Fatalf("expected default language '', got %q", settings.Language)
	}
}

func TestSaveAndLoadSettings_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := NewYAMLStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	original := &Settings{Language: "fr"}
	if err := s.SaveSettings(original); err != nil { //nolint:govet // shadow ok in test
		t.Fatalf("save settings: %v", err)
	}

	// Verify the file was created
	path := filepath.Join(dir, "settings.yaml")
	if _, err := os.Stat(path); err != nil { //nolint:govet // shadow ok in test
		t.Fatalf("settings file not created: %v", err)
	}

	loaded, err := s.LoadSettings()
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if loaded.Language != "fr" {
		t.Fatalf("expected language 'fr', got %q", loaded.Language)
	}
}

func TestLoadSettings_SpecificLanguage(t *testing.T) {
	dir := t.TempDir()
	s, err := NewYAMLStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	// Write a settings file manually
	data := []byte("language: es\n")
	path := filepath.Join(dir, "settings.yaml")
	if err := os.WriteFile(path, data, 0600); err != nil { //nolint:govet // shadow ok in test
		t.Fatalf("write settings file: %v", err)
	}

	settings, err := s.LoadSettings()
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if settings.Language != "es" {
		t.Fatalf("expected language 'es', got %q", settings.Language)
	}
}
