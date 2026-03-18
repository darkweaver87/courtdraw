package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
	"gopkg.in/yaml.v3"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// YAMLStore implements Store backed by YAML files on disk.
type YAMLStore struct {
	exercisesDir  string
	sessionsDir   string
	exerciseIndex *ExerciseIndex
	sessionIndex  *SessionIndex
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
	ys := &YAMLStore{exercisesDir: exDir, sessionsDir: sesDir}
	ys.ensureExerciseIndex()
	ys.ensureSessionIndex()
	ys.migrateRecentFiles()
	return ys, nil
}

var kebabRe = regexp.MustCompile(`[^a-z0-9]+`)

// ToKebab converts a name to a kebab-case filename (without extension).
// Accented characters are transliterated to ASCII (é→e, ç→c, etc.).
func ToKebab(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = stripAccents(s)
	s = kebabRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// stripAccents removes combining marks from NFD-decomposed text,
// transliterating accented characters to their ASCII base (e.g. é→e).
func stripAccents(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range norm.NFD.String(s) {
		if !unicode.Is(unicode.Mn, r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (s *YAMLStore) ListExercises() ([]string, error) {
	names := make([]string, len(s.exerciseIndex.Entries))
	for i, e := range s.exerciseIndex.Entries {
		names[i] = e.File
	}
	return names, nil
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
	return s.SaveExerciseAs(name, exercise)
}

// SaveExerciseAs saves an exercise with an explicit file name (without extension).
func (s *YAMLStore) SaveExerciseAs(name string, exercise *model.Exercise) error {
	if name == "" {
		return errors.New("exercise name is empty")
	}
	path := filepath.Join(s.exercisesDir, name+".yaml")
	data, err := yaml.Marshal(exercise)
	if err != nil {
		return fmt.Errorf("marshal exercise: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}
	now := time.Now()
	entry := exerciseEntryFromExercise(name, exercise, now)
	s.upsertExerciseEntry(entry)
	_ = saveExerciseIndex(s.exercisesDir, s.exerciseIndex)
	return nil
}

func (s *YAMLStore) DeleteExercise(name string) error {
	path := filepath.Join(s.exercisesDir, name+".yaml")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete exercise %s: %w", name, err)
	}
	s.removeExerciseEntry(name)
	_ = saveExerciseIndex(s.exercisesDir, s.exerciseIndex)
	return nil
}

func (s *YAMLStore) ListSessions() ([]string, error) {
	names := make([]string, len(s.sessionIndex.Entries))
	for i, e := range s.sessionIndex.Entries {
		names[i] = e.File
	}
	return names, nil
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
	name := SessionFileName(session)
	if name == "" {
		return errors.New("session title is empty")
	}
	path := filepath.Join(s.sessionsDir, name+".yaml")
	data, err := yaml.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}
	now := time.Now()
	entry := sessionEntryFromSession(name, session, now)
	s.upsertSessionEntry(entry)
	_ = saveSessionIndex(s.sessionsDir, s.sessionIndex)
	return nil
}

// DeleteSession removes a session file and its index entry.
func (s *YAMLStore) DeleteSession(name string) error {
	path := filepath.Join(s.sessionsDir, name+".yaml")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete session %s: %w", name, err)
	}
	s.removeSessionEntry(name)
	_ = saveSessionIndex(s.sessionsDir, s.sessionIndex)
	return nil
}

// SessionFileName returns the kebab-case filename (without extension) for a session,
// combining the sanitized title and date (e.g. "seance-2026-03-03").
func SessionFileName(session *model.Session) string {
	title := ToKebab(session.Title)
	if title == "" {
		return ""
	}
	date := strings.TrimSpace(session.Date)
	if date == "" {
		return title
	}
	return title + "-" + date
}

// ClearRecentFile removes an exercise from the recent files list by resetting LastOpened.
func (s *YAMLStore) ClearRecentFile(name string) {
	for i := range s.exerciseIndex.Entries {
		if s.exerciseIndex.Entries[i].File == name {
			s.exerciseIndex.Entries[i].LastOpened = time.Time{}
			_ = saveExerciseIndex(s.exercisesDir, s.exerciseIndex)
			return
		}
	}
}

// RecordRecentFile marks an exercise as recently opened by setting LastOpened.
func (s *YAMLStore) RecordRecentFile(name string) {
	now := time.Now()
	for i := range s.exerciseIndex.Entries {
		if s.exerciseIndex.Entries[i].File == name {
			s.exerciseIndex.Entries[i].LastOpened = now
			_ = saveExerciseIndex(s.exercisesDir, s.exerciseIndex)
			return
		}
	}
}

// RecentFiles returns the top N exercise file names sorted by LastOpened descending.
func (s *YAMLStore) RecentFiles(maxN int) []string {
	type entry struct {
		file       string
		lastOpened time.Time
	}
	var recent []entry
	for _, e := range s.exerciseIndex.Entries {
		if !e.LastOpened.IsZero() {
			recent = append(recent, entry{file: e.File, lastOpened: e.LastOpened})
		}
	}
	sort.Slice(recent, func(i, j int) bool {
		return recent[i].lastOpened.After(recent[j].lastOpened)
	})
	if len(recent) > maxN {
		recent = recent[:maxN]
	}
	names := make([]string, len(recent))
	for i, e := range recent {
		names[i] = e.file
	}
	return names
}

// RecordRecentSession marks a session as recently opened by setting LastOpened.
func (s *YAMLStore) RecordRecentSession(name string) {
	now := time.Now()
	for i := range s.sessionIndex.Entries {
		if s.sessionIndex.Entries[i].File == name {
			s.sessionIndex.Entries[i].LastOpened = now
			_ = saveSessionIndex(s.sessionsDir, s.sessionIndex)
			return
		}
	}
}

// RecentSessions returns the top N session file names sorted by LastOpened descending.
func (s *YAMLStore) RecentSessions(maxN int) []string {
	type entry struct {
		file       string
		lastOpened time.Time
	}
	var recent []entry
	for _, e := range s.sessionIndex.Entries {
		if !e.LastOpened.IsZero() {
			recent = append(recent, entry{file: e.File, lastOpened: e.LastOpened})
		}
	}
	sort.Slice(recent, func(i, j int) bool {
		return recent[i].lastOpened.After(recent[j].lastOpened)
	})
	if len(recent) > maxN {
		recent = recent[:maxN]
	}
	names := make([]string, len(recent))
	for i, e := range recent {
		names[i] = e.file
	}
	return names
}

// ClearRecentSession removes a session from the recent list by resetting LastOpened.
func (s *YAMLStore) ClearRecentSession(name string) {
	for i := range s.sessionIndex.Entries {
		if s.sessionIndex.Entries[i].File == name {
			s.sessionIndex.Entries[i].LastOpened = time.Time{}
			_ = saveSessionIndex(s.sessionsDir, s.sessionIndex)
			return
		}
	}
}

// ExerciseIndexEntries returns a copy of the exercise index entries.
func (s *YAMLStore) ExerciseIndexEntries() []ExerciseIndexEntry {
	out := make([]ExerciseIndexEntry, len(s.exerciseIndex.Entries))
	copy(out, s.exerciseIndex.Entries)
	return out
}

// RebuildExerciseIndex rebuilds the exercise index from disk.
// Returns file names that failed to parse (nil if all succeeded).
func (s *YAMLStore) RebuildExerciseIndex() []string {
	idx, errs := rebuildExerciseIndex(s.exercisesDir)
	s.exerciseIndex = idx
	_ = saveExerciseIndex(s.exercisesDir, s.exerciseIndex)
	return errs
}

// RebuildSessionIndex rebuilds the session index from disk.
func (s *YAMLStore) RebuildSessionIndex() {
	s.sessionIndex = rebuildSessionIndex(s.sessionsDir)
	_ = saveSessionIndex(s.sessionsDir, s.sessionIndex)
}

// ensureExerciseIndex loads the exercise index, rebuilding if absent or stale.
func (s *YAMLStore) ensureExerciseIndex() {
	path := filepath.Join(s.exercisesDir, indexFileName)
	if _, err := os.Stat(path); err != nil {
		idx, _ := rebuildExerciseIndex(s.exercisesDir)
		s.exerciseIndex = idx
		_ = saveExerciseIndex(s.exercisesDir, s.exerciseIndex)
		return
	}
	s.exerciseIndex = loadExerciseIndex(s.exercisesDir)
	// Rebuild if the number of index entries doesn't match the YAML files on disk
	// (handles externally added or removed files).
	if countYAMLFiles(s.exercisesDir) != len(s.exerciseIndex.Entries) {
		oldIndex := s.exerciseIndex
		idx, _ := rebuildExerciseIndex(s.exercisesDir)
		// Preserve LastOpened from the old index.
		oldMap := make(map[string]time.Time, len(oldIndex.Entries))
		for _, e := range oldIndex.Entries {
			if !e.LastOpened.IsZero() {
				oldMap[e.File] = e.LastOpened
			}
		}
		for i := range idx.Entries {
			if t, ok := oldMap[idx.Entries[i].File]; ok {
				idx.Entries[i].LastOpened = t
			}
		}
		s.exerciseIndex = idx
		_ = saveExerciseIndex(s.exercisesDir, s.exerciseIndex)
	}
}

// ensureSessionIndex loads the session index, rebuilding if absent or stale.
func (s *YAMLStore) ensureSessionIndex() {
	path := filepath.Join(s.sessionsDir, indexFileName)
	if _, err := os.Stat(path); err != nil {
		s.sessionIndex = rebuildSessionIndex(s.sessionsDir)
		_ = saveSessionIndex(s.sessionsDir, s.sessionIndex)
		return
	}
	s.sessionIndex = loadSessionIndex(s.sessionsDir)
	// Rebuild if the number of index entries doesn't match the YAML files on disk.
	if countYAMLFiles(s.sessionsDir) != len(s.sessionIndex.Entries) {
		oldIndex := s.sessionIndex
		s.sessionIndex = rebuildSessionIndex(s.sessionsDir)
		// Preserve LastOpened from the old index.
		oldMap := make(map[string]time.Time, len(oldIndex.Entries))
		for _, e := range oldIndex.Entries {
			if !e.LastOpened.IsZero() {
				oldMap[e.File] = e.LastOpened
			}
		}
		for i := range s.sessionIndex.Entries {
			if t, ok := oldMap[s.sessionIndex.Entries[i].File]; ok {
				s.sessionIndex.Entries[i].LastOpened = t
			}
		}
		_ = saveSessionIndex(s.sessionsDir, s.sessionIndex)
	}
}

// countYAMLFiles counts non-index YAML files in a directory.
func countYAMLFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if isIndexFile(name) {
			continue
		}
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			n++
		}
	}
	return n
}

// migrateRecentFiles migrates settings.RecentFiles to index LastOpened entries.
func (s *YAMLStore) migrateRecentFiles() {
	settings, err := s.LoadSettings()
	if err != nil || len(settings.RecentFiles) == 0 {
		return
	}
	now := time.Now()
	for i, name := range settings.RecentFiles {
		for j := range s.exerciseIndex.Entries {
			if s.exerciseIndex.Entries[j].File == name && s.exerciseIndex.Entries[j].LastOpened.IsZero() {
				// Stagger times so ordering is preserved (most recent first).
				s.exerciseIndex.Entries[j].LastOpened = now.Add(-time.Duration(i) * time.Second)
			}
		}
	}
	_ = saveExerciseIndex(s.exercisesDir, s.exerciseIndex)
	settings.RecentFiles = nil
	_ = s.SaveSettings(settings)
}

// upsertExerciseEntry adds or updates an exercise entry in the index.
func (s *YAMLStore) upsertExerciseEntry(entry ExerciseIndexEntry) {
	for i, e := range s.exerciseIndex.Entries {
		if e.File == entry.File {
			// Preserve LastOpened and Created from existing entry.
			entry.LastOpened = e.LastOpened
			if !e.Created.IsZero() {
				entry.Created = e.Created
			}
			s.exerciseIndex.Entries[i] = entry
			return
		}
	}
	// New entry: set Created to now if not already set.
	if entry.Created.IsZero() {
		entry.Created = entry.Modified
	}
	s.exerciseIndex.Entries = append(s.exerciseIndex.Entries, entry)
}

// removeExerciseEntry removes an exercise entry from the index by file name.
func (s *YAMLStore) removeExerciseEntry(file string) {
	entries := s.exerciseIndex.Entries
	for i, e := range entries {
		if e.File == file {
			s.exerciseIndex.Entries = append(entries[:i], entries[i+1:]...)
			return
		}
	}
}

// upsertSessionEntry adds or updates a session entry in the index.
func (s *YAMLStore) upsertSessionEntry(entry SessionIndexEntry) {
	for i, e := range s.sessionIndex.Entries {
		if e.File == entry.File {
			// Preserve LastOpened from existing entry.
			entry.LastOpened = e.LastOpened
			s.sessionIndex.Entries[i] = entry
			return
		}
	}
	s.sessionIndex.Entries = append(s.sessionIndex.Entries, entry)
}

// removeSessionEntry removes a session entry from the index by file name.
func (s *YAMLStore) removeSessionEntry(file string) {
	entries := s.sessionIndex.Entries
	for i, e := range entries {
		if e.File == file {
			s.sessionIndex.Entries = append(entries[:i], entries[i+1:]...)
			return
		}
	}
}
