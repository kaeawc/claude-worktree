package gitlab

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var (
	// ErrNotGitLabRepo is returned when the repository is not a GitLab repository
	ErrNotGitLabRepo = errors.New("not a GitLab repository")
	// ErrNoRemote is returned when no git remote is configured
	ErrNoRemote = errors.New("no git remote configured")
)

// RepositoryInfo contains detected repository information
type RepositoryInfo struct {
	Owner string // Group/owner (may include nested groups like "group/subgroup")
	Project string // Project name
	Host  string // GitLab host (e.g., "gitlab.com" or "gitlab.example.com")
	URL   string // Remote URL
}

// DetectRepository auto-detects GitLab owner/project/host from git remote
// Tries 'origin' remote first, falls back to first available remote
// Supports both HTTPS and SSH URLs
// Supports both gitlab.com and self-hosted instances
func DetectRepository(gitRoot string) (*RepositoryInfo, error) {
	// Try origin remote first
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	cmd.Dir = gitRoot
	output, err := cmd.Output()

	if err != nil {
		// Origin not found, try to get first remote
		cmd = exec.Command("git", "remote")
		cmd.Dir = gitRoot
		remotesOutput, remotesErr := cmd.Output()
		if remotesErr != nil {
			return nil, ErrNoRemote
		}

		remotes := strings.Split(strings.TrimSpace(string(remotesOutput)), "\n")
		if len(remotes) == 0 || remotes[0] == "" {
			return nil, ErrNoRemote
		}

		// Get URL for first remote
		cmd = exec.Command("git", "config", "--get", fmt.Sprintf("remote.%s.url", remotes[0]))
		cmd.Dir = gitRoot
		output, err = cmd.Output()
		if err != nil {
			return nil, ErrNoRemote
		}
	}

	url := strings.TrimSpace(string(output))
	if url == "" {
		return nil, ErrNoRemote
	}

	owner, project, host, err := parseGitLabURL(url)
	if err != nil {
		return nil, err
	}

	return &RepositoryInfo{
		Owner:   owner,
		Project: project,
		Host:    host,
		URL:     url,
	}, nil
}

// parseGitLabURL extracts owner/project/host from a GitLab remote URL
// Handles:
//   - https://gitlab.com/owner/project.git
//   - https://gitlab.com/owner/project
//   - git@gitlab.com:owner/project.git
//   - https://gitlab.example.com/owner/project.git (self-hosted)
//   - git@gitlab.example.com:owner/project.git (self-hosted)
//   - https://gitlab.com/group/subgroup/project.git (nested groups)
//   - git@gitlab.com:group/subgroup/project.git (nested groups)
func parseGitLabURL(url string) (owner, project, host string, err error) {
	// HTTPS pattern: https://<host>/owner/project(.git)?
	// Captures owner (may contain nested groups like "group/subgroup"), project, and host
	httpsPattern := regexp.MustCompile(`^https://([^/]+)/(.+)/([^/]+?)(\.git)?$`)
	if matches := httpsPattern.FindStringSubmatch(url); matches != nil {
		host := matches[1]
		ownerPath := matches[2]
		project := matches[3]

		// Validate it's a GitLab host
		if isGitLabHost(host) {
			return ownerPath, project, host, nil
		}
	}

	// SSH pattern: git@<host>:owner/project(.git)?
	// Captures owner (may contain nested groups), project, and host
	sshPattern := regexp.MustCompile(`^git@([^:]+):(.+)/([^/]+?)(\.git)?$`)
	if matches := sshPattern.FindStringSubmatch(url); matches != nil {
		host := matches[1]
		ownerPath := matches[2]
		project := matches[3]

		// Validate it's a GitLab host
		if isGitLabHost(host) {
			return ownerPath, project, host, nil
		}
	}

	return "", "", "", ErrNotGitLabRepo
}

// isGitLabHost checks if a host is a GitLab instance
// Returns true for gitlab.com and self-hosted instances
func isGitLabHost(host string) bool {
	// gitlab.com
	if host == "gitlab.com" {
		return true
	}

	// Self-hosted GitLab (heuristic: contains "gitlab" in the hostname)
	if strings.Contains(strings.ToLower(host), "gitlab") {
		return true
	}

	return false
}
