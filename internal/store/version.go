package store

import (
	"context"
	"fmt"
)

// ReleaseInfo holds tag and URL for a GitHub release.
type ReleaseInfo struct {
	Tag string
	URL string
}

// CheckLatestVersion returns the latest release tag and URL from GitHub.
func CheckLatestVersion(ctx context.Context, token string) (*ReleaseInfo, error) {
	client := newGitHubClient(token)
	release, _, err := client.Repositories.GetLatestRelease(ctx, githubOwner, githubRepo)
	if err != nil {
		return nil, fmt.Errorf("get latest release: %w", err)
	}
	return &ReleaseInfo{
		Tag: release.GetTagName(),
		URL: release.GetHTMLURL(),
	}, nil
}
