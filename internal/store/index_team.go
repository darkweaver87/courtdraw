package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/darkweaver87/courtdraw/internal/model"
)

// TeamIndexEntry holds metadata for a single team in the index.
type TeamIndexEntry struct {
	File     string    `yaml:"file"`
	Name     string    `yaml:"name"`
	Club     string    `yaml:"club,omitempty"`
	Season   string    `yaml:"season,omitempty"`
	Members  int       `yaml:"members"`
	Modified time.Time `yaml:"modified"`
}

// TeamIndex is the on-disk index of all teams in a directory.
type TeamIndex struct {
	Version int              `yaml:"version"`
	Entries []TeamIndexEntry `yaml:"entries"`
}

// teamEntryFromTeam extracts index metadata from a team.
func teamEntryFromTeam(file string, team *model.Team, modTime time.Time) TeamIndexEntry {
	return TeamIndexEntry{
		File:     file,
		Name:     team.Name,
		Club:     team.Club,
		Season:   team.Season,
		Members:  len(team.Members),
		Modified: modTime,
	}
}

// loadTeamIndex reads index.yaml from dir. Returns an empty index if absent or corrupt.
func loadTeamIndex(dir string) *TeamIndex {
	path := filepath.Join(dir, indexFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return &TeamIndex{Version: 1}
	}
	var idx TeamIndex
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return &TeamIndex{Version: 1}
	}
	if idx.Version == 0 {
		idx.Version = 1
	}
	return &idx
}

// saveTeamIndex writes the team index atomically (write .tmp then rename).
func saveTeamIndex(dir string, idx *TeamIndex) error {
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

// rebuildTeamIndex scans the directory and builds a fresh index.
func rebuildTeamIndex(dir string) *TeamIndex {
	idx := &TeamIndex{Version: 1}
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
		var team model.Team
		if unmarshalErr := yaml.Unmarshal(data, &team); unmarshalErr != nil {
			continue
		}
		info, err := e.Info()
		modTime := time.Now()
		if err == nil {
			modTime = info.ModTime()
		}
		idx.Entries = append(idx.Entries, teamEntryFromTeam(base, &team, modTime))
	}
	return idx
}

// upsertTeamEntry adds or updates a team entry in the index.
func (s *YAMLStore) upsertTeamEntry(entry TeamIndexEntry) {
	for i, e := range s.teamIndex.Entries {
		if e.File == entry.File {
			s.teamIndex.Entries[i] = entry
			return
		}
	}
	s.teamIndex.Entries = append(s.teamIndex.Entries, entry)
}

// removeTeamEntry removes a team entry from the index by file name.
func (s *YAMLStore) removeTeamEntry(file string) {
	entries := s.teamIndex.Entries
	for i, e := range entries {
		if e.File == file {
			s.teamIndex.Entries = append(entries[:i], entries[i+1:]...)
			return
		}
	}
}

// ensureTeamIndex loads the team index, rebuilding if absent or stale.
func (s *YAMLStore) ensureTeamIndex() {
	path := filepath.Join(s.teamsDir, indexFileName)
	if _, err := os.Stat(path); err != nil {
		s.teamIndex = rebuildTeamIndex(s.teamsDir)
		_ = saveTeamIndex(s.teamsDir, s.teamIndex)
		return
	}
	s.teamIndex = loadTeamIndex(s.teamsDir)
	if countYAMLFiles(s.teamsDir) != len(s.teamIndex.Entries) {
		s.teamIndex = rebuildTeamIndex(s.teamsDir)
		_ = saveTeamIndex(s.teamsDir, s.teamIndex)
	}
}

// TeamFileName returns the kebab-case filename (without extension) for a team.
func TeamFileName(team *model.Team) string {
	name := ToKebab(team.Name)
	if name == "" {
		return ""
	}
	season := strings.TrimSpace(team.Season)
	if season == "" {
		return name
	}
	return fmt.Sprintf("%s-%s", name, ToKebab(season))
}
