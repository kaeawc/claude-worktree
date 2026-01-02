package gitlab

import (
	"testing"
)

func TestParseGitLabURLHTTPS(t *testing.T) {
	tests := []struct {
		url       string
		owner     string
		project   string
		host      string
		shouldErr bool
	}{
		{
			url:       "https://gitlab.com/owner/project.git",
			owner:     "owner",
			project:   "project",
			host:      "gitlab.com",
			shouldErr: false,
		},
		{
			url:       "https://gitlab.com/owner/project",
			owner:     "owner",
			project:   "project",
			host:      "gitlab.com",
			shouldErr: false,
		},
		{
			url:       "https://gitlab.example.com/owner/project.git",
			owner:     "owner",
			project:   "project",
			host:      "gitlab.example.com",
			shouldErr: false,
		},
		{
			url:       "https://gitlab.com/group/subgroup/project.git",
			owner:     "group/subgroup",
			project:   "project",
			host:      "gitlab.com",
			shouldErr: false,
		},
		{
			url:       "https://gitlab.example.com/group/subgroup/project",
			owner:     "group/subgroup",
			project:   "project",
			host:      "gitlab.example.com",
			shouldErr: false,
		},
		{
			url:       "https://github.com/owner/repo.git",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		owner, project, host, err := parseGitLabURL(tt.url)
		if (err != nil) != tt.shouldErr {
			t.Errorf("parseGitLabURL(%s): expected error=%v, got error=%v", tt.url, tt.shouldErr, err != nil)
			continue
		}

		if !tt.shouldErr {
			if owner != tt.owner {
				t.Errorf("parseGitLabURL(%s): expected owner=%s, got %s", tt.url, tt.owner, owner)
			}
			if project != tt.project {
				t.Errorf("parseGitLabURL(%s): expected project=%s, got %s", tt.url, tt.project, project)
			}
			if host != tt.host {
				t.Errorf("parseGitLabURL(%s): expected host=%s, got %s", tt.url, tt.host, host)
			}
		}
	}
}

func TestParseGitLabURLSSH(t *testing.T) {
	tests := []struct {
		url       string
		owner     string
		project   string
		host      string
		shouldErr bool
	}{
		{
			url:       "git@gitlab.com:owner/project.git",
			owner:     "owner",
			project:   "project",
			host:      "gitlab.com",
			shouldErr: false,
		},
		{
			url:       "git@gitlab.com:owner/project",
			owner:     "owner",
			project:   "project",
			host:      "gitlab.com",
			shouldErr: false,
		},
		{
			url:       "git@gitlab.example.com:owner/project.git",
			owner:     "owner",
			project:   "project",
			host:      "gitlab.example.com",
			shouldErr: false,
		},
		{
			url:       "git@gitlab.com:group/subgroup/project.git",
			owner:     "group/subgroup",
			project:   "project",
			host:      "gitlab.com",
			shouldErr: false,
		},
		{
			url:       "git@github.com:owner/repo.git",
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		owner, project, host, err := parseGitLabURL(tt.url)
		if (err != nil) != tt.shouldErr {
			t.Errorf("parseGitLabURL(%s): expected error=%v, got error=%v", tt.url, tt.shouldErr, err != nil)
			continue
		}

		if !tt.shouldErr {
			if owner != tt.owner {
				t.Errorf("parseGitLabURL(%s): expected owner=%s, got %s", tt.url, tt.owner, owner)
			}
			if project != tt.project {
				t.Errorf("parseGitLabURL(%s): expected project=%s, got %s", tt.url, tt.project, project)
			}
			if host != tt.host {
				t.Errorf("parseGitLabURL(%s): expected host=%s, got %s", tt.url, tt.host, host)
			}
		}
	}
}

func TestIsGitLabHost(t *testing.T) {
	tests := []struct {
		host     string
		isGitLab bool
	}{
		{"gitlab.com", true},
		{"gitlab.example.com", true},
		{"my.gitlab.example.com", true},
		{"gitlab-instance.io", true},
		{"github.com", false},
		{"bitbucket.org", false},
		{"example.com", false},
	}

	for _, tt := range tests {
		result := isGitLabHost(tt.host)
		if result != tt.isGitLab {
			t.Errorf("isGitLabHost(%s): expected %v, got %v", tt.host, tt.isGitLab, result)
		}
	}
}
