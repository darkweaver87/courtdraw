package store

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// MatchIndexEntry holds metadata for a single match in the index.
type MatchIndexEntry struct {
	File     string    `yaml:"file"`
	TeamName string    `yaml:"team_name"`
	Opponent string    `yaml:"opponent"`
	Date     string    `yaml:"date"`
	HomeAway string    `yaml:"home_away"`
	Status   string    `yaml:"status"`
	Modified time.Time `yaml:"modified"`
}

// MatchIndex is the on-disk index of all matches in a directory.
type MatchIndex struct {
	Version int               `yaml:"version"`
	Entries []MatchIndexEntry `yaml:"entries"`
}

// matchEntryFromMatch extracts index metadata from a match.
func matchEntryFromMatch(file string, match *model.Match, modTime time.Time) MatchIndexEntry {
	return MatchIndexEntry{
		File:     file,
		TeamName: match.TeamName,
		Opponent: match.Opponent,
		Date:     match.Date,
		HomeAway: match.HomeAway,
		Status:   match.Status,
		Modified: modTime,
	}
}

// loadMatchIndex reads index.yaml from dir. Returns an empty index if absent or corrupt.
func loadMatchIndex(dir string) *MatchIndex {
	path := filepath.Join(dir, indexFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return &MatchIndex{Version: 1}
	}
	var idx MatchIndex
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return &MatchIndex{Version: 1}
	}
	if idx.Version == 0 {
		idx.Version = 1
	}
	return &idx
}

// saveMatchIndex writes the match index atomically (write .tmp then rename).
func saveMatchIndex(dir string, idx *MatchIndex) error {
	data, err := yaml.Marshal(idx)
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, indexFileName+".tmp")
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, indexFileName))
}

// rebuildMatchIndex scans the directory and builds a fresh index.
func rebuildMatchIndex(dir string) *MatchIndex {
	idx := &MatchIndex{Version: 1}
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
		var match model.Match
		if unmarshalErr := yaml.Unmarshal(data, &match); unmarshalErr != nil {
			continue
		}
		info, err := e.Info()
		modTime := time.Now()
		if err == nil {
			modTime = info.ModTime()
		}
		idx.Entries = append(idx.Entries, matchEntryFromMatch(base, &match, modTime))
	}
	return idx
}

// upsertMatchEntry adds or updates a match entry in the index.
func (s *YAMLStore) upsertMatchEntry(entry MatchIndexEntry) {
	for i, e := range s.matchIndex.Entries {
		if e.File == entry.File {
			s.matchIndex.Entries[i] = entry
			return
		}
	}
	s.matchIndex.Entries = append(s.matchIndex.Entries, entry)
}

// removeMatchEntry removes a match entry from the index by file name.
func (s *YAMLStore) removeMatchEntry(file string) {
	entries := s.matchIndex.Entries
	for i, e := range entries {
		if e.File == file {
			s.matchIndex.Entries = append(entries[:i], entries[i+1:]...)
			return
		}
	}
}

// ensureMatchIndex loads the match index, rebuilding if absent or stale.
func (s *YAMLStore) ensureMatchIndex() {
	path := filepath.Join(s.matchesDir, indexFileName)
	if _, err := os.Stat(path); err != nil {
		s.matchIndex = rebuildMatchIndex(s.matchesDir)
		_ = saveMatchIndex(s.matchesDir, s.matchIndex)
		return
	}
	s.matchIndex = loadMatchIndex(s.matchesDir)
	if countYAMLFiles(s.matchesDir) != len(s.matchIndex.Entries) {
		s.matchIndex = rebuildMatchIndex(s.matchesDir)
		_ = saveMatchIndex(s.matchesDir, s.matchIndex)
	}
}

// MatchFileName returns the kebab-case filename (without extension) for a match.
func MatchFileName(match *model.Match) string {
	opponent := ToKebab(match.Opponent)
	if opponent == "" {
		return ""
	}
	date := strings.TrimSpace(match.Date)
	if date == "" {
		return opponent
	}
	return opponent + "-" + date
}
