package store

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/v74/github"
	"gopkg.in/yaml.v3"
)

const (
	githubOwner = "darkweaver87"
	githubRepo  = "courtdraw"
	libraryPath = "library"
)

// Manifest tracks SHA of each cached file for incremental sync.
type Manifest struct {
	Files    map[string]string `yaml:"files"`     // filename → git blob SHA
	LastSync string            `yaml:"last_sync"` // RFC3339
}

// LibrarySyncResult reports what changed during a sync.
type LibrarySyncResult struct {
	Added   []string
	Updated []string
	Removed []string
}

// SyncLibrary fetches the library/ directory from GitHub, compares SHA hashes
// against the local manifest, and downloads only changed files.
// token is optional — unauthenticated requests work for public repos (60 req/h).
func SyncLibrary(ctx context.Context, cacheDir, token string) (*LibrarySyncResult, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	client := newGitHubClient(token)

	// List files in library/ on GitHub.
	_, dirContents, _, err := client.Repositories.GetContents(
		ctx, githubOwner, githubRepo, libraryPath,
		&github.RepositoryContentGetOptions{Ref: "main"},
	)
	if err != nil {
		return nil, fmt.Errorf("list remote library: %w", err)
	}

	// Build remote file map: filename → SHA.
	remoteFiles := make(map[string]string)
	for _, entry := range dirContents {
		name := entry.GetName()
		if entry.GetType() != "file" {
			continue
		}
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		remoteFiles[name] = entry.GetSHA()
	}

	// Load local manifest.
	manifest := loadManifest(cacheDir)

	result := &LibrarySyncResult{}

	// Download new or updated files.
	for name, remoteSHA := range remoteFiles {
		localSHA := manifest.Files[name]
		if localSHA == remoteSHA {
			continue
		}

		content, err := downloadFile(ctx, client, name)
		if err != nil {
			return nil, fmt.Errorf("download %s: %w", name, err)
		}

		if err := os.WriteFile(filepath.Join(cacheDir, name), content, 0600); err != nil {
			return nil, fmt.Errorf("write %s: %w", name, err)
		}

		if localSHA == "" {
			result.Added = append(result.Added, name)
		} else {
			result.Updated = append(result.Updated, name)
		}
		manifest.Files[name] = remoteSHA
	}

	// Remove local files that are no longer on remote.
	for name := range manifest.Files {
		if _, ok := remoteFiles[name]; !ok {
			os.Remove(filepath.Join(cacheDir, name))
			result.Removed = append(result.Removed, name)
			delete(manifest.Files, name)
		}
	}

	manifest.LastSync = time.Now().UTC().Format(time.RFC3339)
	if err := saveManifest(cacheDir, manifest); err != nil {
		return result, fmt.Errorf("save manifest: %w", err)
	}

	return result, nil
}

// IsCacheEmpty returns true if the library cache directory has no .yaml files.
func IsCacheEmpty(cacheDir string) bool {
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return true
	}
	for _, e := range entries {
		if !e.IsDir() && (strings.HasSuffix(e.Name(), ".yaml") || strings.HasSuffix(e.Name(), ".yml")) {
			return false
		}
	}
	return true
}

func newGitHubClient(token string) *github.Client {
	if token == "" {
		return github.NewClient(nil)
	}
	httpClient := &http.Client{
		Transport: &syncTokenTransport{token: token, base: http.DefaultTransport},
	}
	return github.NewClient(httpClient)
}

// syncTokenTransport adds a Bearer token to requests.
type syncTokenTransport struct {
	token string
	base  http.RoundTripper
}

func (t *syncTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(req2)
}

func downloadFile(ctx context.Context, client *github.Client, name string) ([]byte, error) {
	fileContent, _, _, err := client.Repositories.GetContents(
		ctx, githubOwner, githubRepo, libraryPath+"/"+name,
		&github.RepositoryContentGetOptions{Ref: "main"},
	)
	if err != nil {
		return nil, err
	}
	content, err := fileContent.GetContent()
	if err != nil {
		return nil, err
	}
	return []byte(content), nil
}

const manifestFile = ".manifest.yaml"

func loadManifest(cacheDir string) *Manifest {
	path := filepath.Join(cacheDir, manifestFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return &Manifest{Files: make(map[string]string)}
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return &Manifest{Files: make(map[string]string)}
	}
	if m.Files == nil {
		m.Files = make(map[string]string)
	}
	return &m
}

func saveManifest(cacheDir string, m *Manifest) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cacheDir, manifestFile), data, 0600)
}
