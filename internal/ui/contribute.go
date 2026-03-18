package ui

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v74/github"
)

const (
	upstreamOwner = "darkweaver87"
	upstreamRepo  = "courtdraw"
)

// tokenTransport is an http.RoundTripper that adds a Bearer token header.
type tokenTransport struct {
	token string
	base  http.RoundTripper
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(req2)
}

// createContributionPR uses the GitHub API (go-github) to fork the repo,
// push the exercise file, and open a pull request. Returns the PR URL.
func createContributionPR(token, name string, yamlData []byte) (string, error) {
	if token == "" {
		return "", errors.New("no GitHub token configured")
	}

	ctx := context.Background()
	httpClient := &http.Client{
		Transport: &tokenTransport{token: token, base: http.DefaultTransport},
	}
	client := github.NewClient(httpClient)

	// 1. Get authenticated user login.
	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return "", fmt.Errorf("get user: %w", err)
	}
	login := user.GetLogin()

	// 2. Fork the repo (no-op if already forked).
	// CreateFork returns 202 (job scheduled) or 422 (already exists) — both are fine.
	_, _, _ = client.Repositories.CreateFork(ctx, upstreamOwner, upstreamRepo, &github.RepositoryCreateForkOptions{})

	// 3. Get upstream main branch SHA.
	ref, _, err := client.Git.GetRef(ctx, upstreamOwner, upstreamRepo, "refs/heads/main")
	if err != nil {
		return "", fmt.Errorf("get main ref: %w", err)
	}
	sha := ref.Object.GetSHA()

	branch := "contribute-" + sanitizeBranch(name)
	filePath := "library/" + name + ".yaml"

	// 4. Create branch on the fork.
	_, _, err = client.Git.CreateRef(ctx, login, upstreamRepo, &github.Reference{
		Ref:    github.Ptr("refs/heads/" + branch),
		Object: &github.GitObject{SHA: github.Ptr(sha)},
	})
	if err != nil {
		// Branch may already exist — continue.
		if !strings.Contains(err.Error(), "Reference already exists") {
			return "", fmt.Errorf("create branch: %w", err)
		}
	}

	// 5. Upload the exercise file.
	commitMsg := "Add exercise: " + name
	_, _, err = client.Repositories.CreateFile(ctx, login, upstreamRepo, filePath, &github.RepositoryContentFileOptions{
		Message: github.Ptr(commitMsg),
		Content: yamlData,
		Branch:  github.Ptr(branch),
	})
	if err != nil {
		return "", fmt.Errorf("upload file: %w", err)
	}

	// 6. Create pull request.
	pr, _, err := client.PullRequests.Create(ctx, upstreamOwner, upstreamRepo, &github.NewPullRequest{
		Title: github.Ptr("Add exercise: " + name),
		Body:  github.Ptr("Community exercise contribution: **" + name + "**"),
		Head:  github.Ptr(login + ":" + branch),
		Base:  github.Ptr("main"),
	})
	if err != nil {
		return "", fmt.Errorf("create PR: %w", err)
	}

	return pr.GetHTMLURL(), nil
}

// sanitizeBranch removes characters not allowed in git branch names.
func sanitizeBranch(s string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(s) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			b.WriteRune(c)
		} else if c == ' ' || c == '_' {
			b.WriteRune('-')
		}
	}
	result := b.String()
	if len(result) > 40 {
		result = result[:40]
	}
	return result
}
