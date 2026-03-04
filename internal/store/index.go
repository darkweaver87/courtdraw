package store

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/darkweaver87/courtdraw/internal/model"
)

const indexFileName = "index.yaml"

// ExerciseIndexEntry holds metadata for a single exercise in the index.
type ExerciseIndexEntry struct {
	File       string    `yaml:"file"`
	Name       string    `yaml:"name"`
	Category   string    `yaml:"category,omitempty"`
	AgeGroup   string    `yaml:"age_group,omitempty"`
	CourtType  string    `yaml:"court_type,omitempty"`
	Duration   string    `yaml:"duration,omitempty"`
	Tags       []string  `yaml:"tags,omitempty"`
	Modified   time.Time `yaml:"modified"`
	LastOpened time.Time `yaml:"last_opened,omitempty"`
}

// ExerciseIndex is the on-disk index of all exercises in a directory.
type ExerciseIndex struct {
	Version int                  `yaml:"version"`
	Entries []ExerciseIndexEntry `yaml:"entries"`
}

// SessionIndexEntry holds metadata for a single session in the index.
type SessionIndexEntry struct {
	File       string    `yaml:"file"`
	Title      string    `yaml:"title"`
	Date       string    `yaml:"date,omitempty"`
	Modified   time.Time `yaml:"modified"`
	LastOpened time.Time `yaml:"last_opened,omitempty"`
}

// SessionIndex is the on-disk index of all sessions in a directory.
type SessionIndex struct {
	Version int                 `yaml:"version"`
	Entries []SessionIndexEntry `yaml:"entries"`
}

// exerciseEntryFromExercise extracts index metadata from an exercise.
func exerciseEntryFromExercise(file string, ex *model.Exercise, modTime time.Time) ExerciseIndexEntry {
	return ExerciseIndexEntry{
		File:      file,
		Name:      ex.Name,
		Category:  string(ex.Category),
		AgeGroup:  string(ex.AgeGroup),
		CourtType: string(ex.CourtType),
		Duration:  ex.Duration,
		Tags:      ex.Tags,
		Modified:  modTime,
	}
}

// sessionEntryFromSession extracts index metadata from a session.
func sessionEntryFromSession(file string, ses *model.Session, modTime time.Time) SessionIndexEntry {
	return SessionIndexEntry{
		File:     file,
		Title:    ses.Title,
		Date:     ses.Date,
		Modified: modTime,
	}
}

// loadExerciseIndex reads index.yaml from dir. Returns an empty index if absent or corrupt.
func loadExerciseIndex(dir string) *ExerciseIndex {
	path := filepath.Join(dir, indexFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return &ExerciseIndex{Version: 1}
	}
	var idx ExerciseIndex
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return &ExerciseIndex{Version: 1}
	}
	if idx.Version == 0 {
		idx.Version = 1
	}
	return &idx
}

// loadSessionIndex reads index.yaml from dir. Returns an empty index if absent or corrupt.
func loadSessionIndex(dir string) *SessionIndex {
	path := filepath.Join(dir, indexFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return &SessionIndex{Version: 1}
	}
	var idx SessionIndex
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return &SessionIndex{Version: 1}
	}
	if idx.Version == 0 {
		idx.Version = 1
	}
	return &idx
}

// saveExerciseIndex writes the exercise index atomically (write .tmp then rename).
func saveExerciseIndex(dir string, idx *ExerciseIndex) error {
	data, err := yaml.Marshal(idx)
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, indexFileName+".tmp")
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, indexFileName))
}

// saveSessionIndex writes the session index atomically (write .tmp then rename).
func saveSessionIndex(dir string, idx *SessionIndex) error {
	data, err := yaml.Marshal(idx)
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, indexFileName+".tmp")
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, indexFileName))
}

// rebuildExerciseIndex scans the directory and builds a fresh index.
func rebuildExerciseIndex(dir string) *ExerciseIndex {
	idx := &ExerciseIndex{Version: 1}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return idx
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if isIndexFile(name) {
			continue
		}
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		base := strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml")
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var ex model.Exercise
		if err := yaml.Unmarshal(data, &ex); err != nil {
			continue
		}
		info, err := e.Info()
		modTime := time.Now()
		if err == nil {
			modTime = info.ModTime()
		}
		idx.Entries = append(idx.Entries, exerciseEntryFromExercise(base, &ex, modTime))
	}
	return idx
}

// rebuildSessionIndex scans the directory and builds a fresh index.
func rebuildSessionIndex(dir string) *SessionIndex {
	idx := &SessionIndex{Version: 1}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return idx
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if isIndexFile(name) {
			continue
		}
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		base := strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml")
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var ses model.Session
		if err := yaml.Unmarshal(data, &ses); err != nil {
			continue
		}
		info, err := e.Info()
		modTime := time.Now()
		if err == nil {
			modTime = info.ModTime()
		}
		idx.Entries = append(idx.Entries, sessionEntryFromSession(base, &ses, modTime))
	}
	return idx
}

// isIndexFile returns true if the filename is the index file itself.
func isIndexFile(name string) bool {
	return name == indexFileName || name == "index.yml" || name == indexFileName+".tmp"
}
