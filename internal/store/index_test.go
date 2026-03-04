package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/darkweaver87/courtdraw/internal/model"
)

func TestExerciseEntryFromExercise(t *testing.T) {
	ex := &model.Exercise{
		Name:      "Test Drill",
		Category:  model.Category("offense"),
		AgeGroup:  model.AgeGroup("u13"),
		CourtType: model.HalfCourt,
		Duration:  "15m",
		Tags:      []string{"passing", "3v2"},
	}
	now := time.Now()
	entry := exerciseEntryFromExercise("test-drill", ex, now)

	if entry.File != "test-drill" {
		t.Errorf("File = %q, want test-drill", entry.File)
	}
	if entry.Name != "Test Drill" {
		t.Errorf("Name = %q, want Test Drill", entry.Name)
	}
	if entry.Category != "offense" {
		t.Errorf("Category = %q, want offense", entry.Category)
	}
	if entry.AgeGroup != "u13" {
		t.Errorf("AgeGroup = %q, want u13", entry.AgeGroup)
	}
	if entry.CourtType != "half_court" {
		t.Errorf("CourtType = %q, want half_court", entry.CourtType)
	}
	if entry.Duration != "15m" {
		t.Errorf("Duration = %q, want 15m", entry.Duration)
	}
	if len(entry.Tags) != 2 {
		t.Errorf("Tags len = %d, want 2", len(entry.Tags))
	}
	if !entry.Modified.Equal(now) {
		t.Errorf("Modified = %v, want %v", entry.Modified, now)
	}
}

func TestSessionEntryFromSession(t *testing.T) {
	ses := &model.Session{
		Title: "Training U15",
		Date:  "2026-03-03",
	}
	now := time.Now()
	entry := sessionEntryFromSession("training-u15-2026-03-03", ses, now)

	if entry.File != "training-u15-2026-03-03" {
		t.Errorf("File = %q, want training-u15-2026-03-03", entry.File)
	}
	if entry.Title != "Training U15" {
		t.Errorf("Title = %q, want Training U15", entry.Title)
	}
	if entry.Date != "2026-03-03" {
		t.Errorf("Date = %q, want 2026-03-03", entry.Date)
	}
}

func TestRebuildExerciseIndex(t *testing.T) {
	dir := t.TempDir()

	// Write two exercise files.
	for _, name := range []string{"drill-a", "drill-b"} {
		data := []byte("name: " + name + "\ncourt_type: half_court\ncourt_standard: fiba\nsequences:\n  - label: Setup\n")
		if err := os.WriteFile(filepath.Join(dir, name+".yaml"), data, 0644); err != nil {
			t.Fatal(err)
		}
	}

	idx := rebuildExerciseIndex(dir)
	if len(idx.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(idx.Entries))
	}
	if idx.Version != 1 {
		t.Errorf("Version = %d, want 1", idx.Version)
	}
}

func TestRebuildSessionIndex(t *testing.T) {
	dir := t.TempDir()

	data := []byte("title: Test Session\ndate: 2026-03-03\nexercises: []\n")
	if err := os.WriteFile(filepath.Join(dir, "test-session.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}

	idx := rebuildSessionIndex(dir)
	if len(idx.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(idx.Entries))
	}
	if idx.Entries[0].Title != "Test Session" {
		t.Errorf("Title = %q, want Test Session", idx.Entries[0].Title)
	}
}

func TestRebuildIndex_SkipsIndexFile(t *testing.T) {
	dir := t.TempDir()

	// Write a real exercise and an index file.
	data := []byte("name: Drill\ncourt_type: half_court\ncourt_standard: fiba\nsequences:\n  - label: S\n")
	os.WriteFile(filepath.Join(dir, "drill.yaml"), data, 0644)
	os.WriteFile(filepath.Join(dir, "index.yaml"), []byte("version: 1\nentries: []\n"), 0644)

	idx := rebuildExerciseIndex(dir)
	if len(idx.Entries) != 1 {
		t.Fatalf("expected 1 entry (index.yaml skipped), got %d", len(idx.Entries))
	}
}

func TestSaveAndLoadExerciseIndex(t *testing.T) {
	dir := t.TempDir()

	original := &ExerciseIndex{
		Version: 1,
		Entries: []ExerciseIndexEntry{
			{File: "drill-a", Name: "Drill A", Category: "offense", Modified: time.Now().Truncate(time.Second)},
			{File: "drill-b", Name: "Drill B", Duration: "10m", Modified: time.Now().Truncate(time.Second)},
		},
	}
	if err := saveExerciseIndex(dir, original); err != nil {
		t.Fatal(err)
	}

	loaded := loadExerciseIndex(dir)
	if len(loaded.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(loaded.Entries))
	}
	if loaded.Entries[0].File != "drill-a" || loaded.Entries[1].File != "drill-b" {
		t.Errorf("entries mismatch: %+v", loaded.Entries)
	}
}

func TestSaveAndLoadSessionIndex(t *testing.T) {
	dir := t.TempDir()

	original := &SessionIndex{
		Version: 1,
		Entries: []SessionIndexEntry{
			{File: "ses-a", Title: "Session A", Date: "2026-01-01", Modified: time.Now().Truncate(time.Second)},
		},
	}
	if err := saveSessionIndex(dir, original); err != nil {
		t.Fatal(err)
	}

	loaded := loadSessionIndex(dir)
	if len(loaded.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded.Entries))
	}
	if loaded.Entries[0].Title != "Session A" {
		t.Errorf("Title = %q, want Session A", loaded.Entries[0].Title)
	}
}

func TestYAMLStore_DeleteSession(t *testing.T) {
	dir := t.TempDir()
	s, err := NewYAMLStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	ses := &model.Session{
		Title:    "Test Session",
		AgeGroup: "U15",
		Exercises: []model.ExerciseEntry{
			{Exercise: "test-drill"},
		},
	}
	if err := s.SaveSession(ses); err != nil {
		t.Fatal(err)
	}

	names, _ := s.ListSessions()
	if len(names) != 1 {
		t.Fatalf("expected 1 session, got %d", len(names))
	}

	if err := s.DeleteSession("test-session"); err != nil {
		t.Fatal(err)
	}

	names, _ = s.ListSessions()
	if len(names) != 0 {
		t.Fatalf("expected 0 sessions after delete, got %d", len(names))
	}

	// File should be gone.
	path := filepath.Join(dir, "sessions", "test-session.yaml")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("session file still exists after delete")
	}
}

func TestYAMLStore_SaveExercise_UpdatesIndex(t *testing.T) {
	dir := t.TempDir()
	s, err := NewYAMLStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	ex := testExercise()
	if err := s.SaveExercise(ex); err != nil {
		t.Fatal(err)
	}

	entries := s.ExerciseIndexEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 index entry, got %d", len(entries))
	}
	if entries[0].File != "test-drill" {
		t.Errorf("File = %q, want test-drill", entries[0].File)
	}
	if entries[0].Name != "Test Drill" {
		t.Errorf("Name = %q, want Test Drill", entries[0].Name)
	}
}

func TestYAMLStore_DeleteExercise_UpdatesIndex(t *testing.T) {
	dir := t.TempDir()
	s, err := NewYAMLStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.SaveExercise(testExercise()); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteExercise("test-drill"); err != nil {
		t.Fatal(err)
	}

	entries := s.ExerciseIndexEntries()
	if len(entries) != 0 {
		t.Fatalf("expected 0 index entries after delete, got %d", len(entries))
	}
}

func TestYAMLStore_RecordRecentFile(t *testing.T) {
	dir := t.TempDir()
	s, err := NewYAMLStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Save an exercise first so it exists in the index.
	if err := s.SaveExercise(testExercise()); err != nil {
		t.Fatal(err)
	}

	s.RecordRecentFile("test-drill")

	entries := s.ExerciseIndexEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].LastOpened.IsZero() {
		t.Error("LastOpened should not be zero after RecordRecentFile")
	}
}

func TestYAMLStore_RecentFiles_Order(t *testing.T) {
	dir := t.TempDir()
	s, err := NewYAMLStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Save two exercises.
	ex1 := &model.Exercise{
		Name: "Alpha", CourtType: model.HalfCourt, CourtStandard: model.FIBA,
		Sequences: []model.Sequence{{Label: "S"}},
	}
	ex2 := &model.Exercise{
		Name: "Beta", CourtType: model.HalfCourt, CourtStandard: model.FIBA,
		Sequences: []model.Sequence{{Label: "S"}},
	}
	s.SaveExercise(ex1)
	s.SaveExercise(ex2)

	// Record alpha first, then beta.
	s.RecordRecentFile("alpha")
	time.Sleep(10 * time.Millisecond)
	s.RecordRecentFile("beta")

	recent := s.RecentFiles(10)
	if len(recent) != 2 {
		t.Fatalf("expected 2 recent, got %d", len(recent))
	}
	if recent[0] != "beta" {
		t.Errorf("most recent should be beta, got %q", recent[0])
	}
	if recent[1] != "alpha" {
		t.Errorf("second should be alpha, got %q", recent[1])
	}

	// Test maxN limit.
	capped := s.RecentFiles(1)
	if len(capped) != 1 {
		t.Fatalf("expected 1 capped, got %d", len(capped))
	}
}

func TestYAMLStore_ClearRecentFile(t *testing.T) {
	dir := t.TempDir()
	s, err := NewYAMLStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.SaveExercise(testExercise()); err != nil {
		t.Fatal(err)
	}
	s.RecordRecentFile("test-drill")

	// Verify it's in recent.
	recent := s.RecentFiles(10)
	if len(recent) != 1 {
		t.Fatalf("expected 1 recent, got %d", len(recent))
	}

	// Clear it.
	s.ClearRecentFile("test-drill")

	recent = s.RecentFiles(10)
	if len(recent) != 0 {
		t.Fatalf("expected 0 recent after clear, got %d", len(recent))
	}

	// Exercise should still exist in the index.
	entries := s.ExerciseIndexEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 index entry (not deleted), got %d", len(entries))
	}
}

func TestYAMLStore_MigrateRecentFiles(t *testing.T) {
	dir := t.TempDir()

	// First, create a store and save an exercise so the index has entries.
	s1, err := NewYAMLStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s1.SaveExercise(testExercise()); err != nil {
		t.Fatal(err)
	}

	// Write recent files to settings.
	settings := &Settings{RecentFiles: []string{"test-drill"}}
	if err := s1.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}

	// Create a new store — migration should happen in NewYAMLStore.
	s2, err := NewYAMLStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Recent files should be in index now.
	recent := s2.RecentFiles(10)
	if len(recent) != 1 || recent[0] != "test-drill" {
		t.Errorf("expected [test-drill] in recent, got %v", recent)
	}

	// Settings should no longer have recent files.
	settings2, _ := s2.LoadSettings()
	if len(settings2.RecentFiles) != 0 {
		t.Errorf("settings.RecentFiles should be empty after migration, got %v", settings2.RecentFiles)
	}
}

func testSession(title, date string) *model.Session {
	return &model.Session{Title: title, Date: date}
}

func TestYAMLStore_RecordRecentSession(t *testing.T) {
	dir := t.TempDir()
	s, err := NewYAMLStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	ses := testSession("My Session", "2026-03-04")
	if err := s.SaveSession(ses); err != nil {
		t.Fatal(err)
	}
	name := SessionFileName(ses)
	s.RecordRecentSession(name)

	recent := s.RecentSessions(10)
	if len(recent) != 1 || recent[0] != name {
		t.Errorf("expected [%s] in recent, got %v", name, recent)
	}
}

func TestYAMLStore_RecentSessions_Order(t *testing.T) {
	dir := t.TempDir()
	s, err := NewYAMLStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	s1 := testSession("Alpha", "2026-01-01")
	s2ses := testSession("Beta", "2026-02-02")
	s.SaveSession(s1)
	s.SaveSession(s2ses)

	s.RecordRecentSession(SessionFileName(s1))
	time.Sleep(10 * time.Millisecond)
	s.RecordRecentSession(SessionFileName(s2ses))

	recent := s.RecentSessions(10)
	if len(recent) != 2 {
		t.Fatalf("expected 2 recent, got %d", len(recent))
	}
	if recent[0] != SessionFileName(s2ses) {
		t.Errorf("most recent should be %s, got %q", SessionFileName(s2ses), recent[0])
	}
	if recent[1] != SessionFileName(s1) {
		t.Errorf("second should be %s, got %q", SessionFileName(s1), recent[1])
	}

	capped := s.RecentSessions(1)
	if len(capped) != 1 {
		t.Fatalf("expected 1 capped, got %d", len(capped))
	}
}

func TestYAMLStore_ClearRecentSession(t *testing.T) {
	dir := t.TempDir()
	s, err := NewYAMLStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	ses := testSession("My Session", "2026-03-04")
	if err := s.SaveSession(ses); err != nil {
		t.Fatal(err)
	}
	name := SessionFileName(ses)
	s.RecordRecentSession(name)

	recent := s.RecentSessions(10)
	if len(recent) != 1 {
		t.Fatalf("expected 1 recent, got %d", len(recent))
	}

	s.ClearRecentSession(name)
	recent = s.RecentSessions(10)
	if len(recent) != 0 {
		t.Fatalf("expected 0 recent after clear, got %d", len(recent))
	}

	// Session should still be listed (not deleted).
	names, _ := s.ListSessions()
	if len(names) != 1 {
		t.Fatalf("expected 1 session (not deleted), got %d", len(names))
	}
}
