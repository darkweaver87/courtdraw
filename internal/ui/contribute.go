package ui

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

const upstreamRepo = "darkweaver87/courtdraw"

// createContributionPR uses the gh CLI to fork the repo, push the exercise
// file, and open a pull request. Returns the PR URL on success.
func createContributionPR(name string, yamlData []byte) (string, error) {
	// Verify gh CLI is available and authenticated.
	if _, err := ghExec("auth", "status"); err != nil {
		return "", fmt.Errorf("gh CLI not authenticated — run 'gh auth login' first")
	}

	// Get authenticated user login.
	userJSON, err := ghExec("api", "user", "-q", ".login")
	if err != nil {
		return "", fmt.Errorf("get user: %w", err)
	}
	user := strings.TrimSpace(userJSON)
	if user == "" {
		return "", fmt.Errorf("could not determine GitHub username")
	}

	// Fork the repo (no-op if already forked).
	ghExec("repo", "fork", upstreamRepo, "--clone=false")

	// Get upstream main branch SHA.
	sha, err := ghExec("api", fmt.Sprintf("repos/%s/git/ref/heads/main", upstreamRepo), "-q", ".object.sha")
	if err != nil {
		return "", fmt.Errorf("get main SHA: %w", err)
	}
	sha = strings.TrimSpace(sha)

	branch := "contribute-" + sanitizeBranch(name)
	forkRepo := user + "/courtdraw"
	filePath := "library/" + name + ".yaml"

	// Create branch on the fork.
	_, err = ghExec("api", fmt.Sprintf("repos/%s/git/refs", forkRepo),
		"-X", "POST",
		"-f", "ref=refs/heads/"+branch,
		"-f", "sha="+sha,
	)
	if err != nil {
		// Branch may already exist — try to continue.
		if !strings.Contains(err.Error(), "Reference already exists") {
			return "", fmt.Errorf("create branch: %w", err)
		}
	}

	// Upload the file via Contents API.
	content := base64.StdEncoding.EncodeToString(yamlData)
	_, err = ghExec("api", fmt.Sprintf("repos/%s/contents/%s", forkRepo, filePath),
		"-X", "PUT",
		"-f", fmt.Sprintf("message=Add exercise: %s", name),
		"-f", "content="+content,
		"-f", "branch="+branch,
	)
	if err != nil {
		return "", fmt.Errorf("upload file: %w", err)
	}

	// Create pull request.
	prJSON, err := ghExec("api", fmt.Sprintf("repos/%s/pulls", upstreamRepo),
		"-X", "POST",
		"-f", fmt.Sprintf("title=Add exercise: %s", name),
		"-f", fmt.Sprintf("body=Community exercise contribution: **%s**", name),
		"-f", fmt.Sprintf("head=%s:%s", user, branch),
		"-f", "base=main",
	)
	if err != nil {
		return "", fmt.Errorf("create PR: %w", err)
	}

	// Extract PR URL from response.
	var pr struct {
		HTMLURL string `json:"html_url"`
	}
	if json.Unmarshal([]byte(prJSON), &pr) == nil && pr.HTMLURL != "" {
		return pr.HTMLURL, nil
	}
	return "", nil
}

// ghExec runs a gh CLI command and returns stdout.
func ghExec(args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s: %s", err, string(out))
	}
	return string(out), nil
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
