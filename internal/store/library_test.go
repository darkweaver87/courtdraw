package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewCachedLibrary(t *testing.T) {
	tmp := t.TempDir()
	lib := NewCachedLibrary(tmp)

	want := filepath.Join(tmp, "library")
	if lib.Dir() != want {
		t.Errorf("Dir() = %q, want %q", lib.Dir(), want)
	}

	// Directory should have been created.
	info, err := os.Stat(want)
	if err != nil {
		t.Fatalf("stat library dir: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected library path to be a directory")
	}
}

func TestIsCacheEmpty(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "library")
	os.MkdirAll(dir, 0755)

	if !IsCacheEmpty(dir) {
		t.Error("expected empty cache")
	}

	// Write a non-yaml file — should still be empty.
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0644)
	if !IsCacheEmpty(dir) {
		t.Error("expected empty cache with non-yaml file")
	}

	// Write a yaml file — no longer empty.
	os.WriteFile(filepath.Join(dir, "test.yaml"), []byte("name: test"), 0644)
	if IsCacheEmpty(dir) {
		t.Error("expected non-empty cache with yaml file")
	}
}

func TestIsCacheEmpty_NonexistentDir(t *testing.T) {
	if !IsCacheEmpty("/nonexistent/path") {
		t.Error("expected empty for nonexistent dir")
	}
}

func TestManifestRoundTrip(t *testing.T) {
	tmp := t.TempDir()

	m := &Manifest{
		Files:    map[string]string{"foo.yaml": "abc123", "bar.yaml": "def456"},
		LastSync: "2026-03-07T12:00:00Z",
	}
	if err := saveManifest(tmp, m); err != nil {
		t.Fatalf("save manifest: %v", err)
	}

	loaded := loadManifest(tmp)
	if len(loaded.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(loaded.Files))
	}
	if loaded.Files["foo.yaml"] != "abc123" {
		t.Errorf("foo.yaml SHA = %q, want abc123", loaded.Files["foo.yaml"])
	}
	if loaded.LastSync != "2026-03-07T12:00:00Z" {
		t.Errorf("LastSync = %q, want 2026-03-07T12:00:00Z", loaded.LastSync)
	}
}

func TestLoadManifest_Missing(t *testing.T) {
	tmp := t.TempDir()
	m := loadManifest(tmp)
	if m == nil {
		t.Fatal("expected non-nil manifest")
	}
	if len(m.Files) != 0 {
		t.Errorf("expected empty files map, got %d entries", len(m.Files))
	}
}
