package gitlab

import (
	"errors"
	"testing"
)

func TestIsInstalled(t *testing.T) {
	// Test installed
	fake := NewFakeGitLabExecutor()
	fake.SetResponse("--version", "glab version 1.0.0")
	if !IsInstalled(fake) {
		t.Error("IsInstalled returned false when glab is installed")
	}

	// Test not installed
	fake2 := NewFakeGitLabExecutor()
	fake2.SetError("--version", errors.New("glab not found"))
	if IsInstalled(fake2) {
		t.Error("IsInstalled returned true when glab is not installed")
	}
}

func TestIsAuthenticated(t *testing.T) {
	// Test authenticated
	fake := NewFakeGitLabExecutor()
	fake.SetResponse("auth status", "authenticated")
	if err := IsAuthenticated(fake); err != nil {
		t.Errorf("IsAuthenticated returned error: %v", err)
	}

	// Test not authenticated
	fake2 := NewFakeGitLabExecutor()
	fake2.SetError("auth status", errors.New("not authenticated"))
	if err := IsAuthenticated(fake2); err != ErrGlabNotAuthenticated {
		t.Errorf("IsAuthenticated returned error: %v, expected ErrGlabNotAuthenticated", err)
	}
}

func TestNewClientWithProjectAndExecutor(t *testing.T) {
	fake := NewFakeGitLabExecutor()
	fake.SetResponse("--version", "glab version 1.0.0")
	fake.SetResponse("auth status", "authenticated")

	client, err := NewClientWithProjectAndExecutor("owner", "project", "gitlab.com", fake)
	if err != nil {
		t.Errorf("NewClientWithProjectAndExecutor failed: %v", err)
		return
	}

	if client.Owner != "owner" {
		t.Errorf("expected Owner='owner', got '%s'", client.Owner)
	}
	if client.Project != "project" {
		t.Errorf("expected Project='project', got '%s'", client.Project)
	}
	if client.Host != "gitlab.com" {
		t.Errorf("expected Host='gitlab.com', got '%s'", client.Host)
	}
}

func TestNewClientNotInstalled(t *testing.T) {
	fake := NewFakeGitLabExecutor()
	fake.SetError("--version", errors.New("glab not found"))

	_, err := NewClientWithProjectAndExecutor("owner", "project", "gitlab.com", fake)
	if err != ErrGlabNotInstalled {
		t.Errorf("expected ErrGlabNotInstalled, got %v", err)
	}
}

func TestNewClientNotAuthenticated(t *testing.T) {
	fake := NewFakeGitLabExecutor()
	fake.SetResponse("--version", "glab version 1.0.0")
	fake.SetError("auth status", errors.New("not authenticated"))

	_, err := NewClientWithProjectAndExecutor("owner", "project", "gitlab.com", fake)
	if err != ErrGlabNotAuthenticated {
		t.Errorf("expected ErrGlabNotAuthenticated, got %v", err)
	}
}

func TestNewClientSelfHosted(t *testing.T) {
	fake := NewFakeGitLabExecutor()
	fake.SetResponse("--version", "glab version 1.0.0")
	fake.SetResponse("auth status", "authenticated")

	client, err := NewClientWithProjectAndExecutor("group", "project", "gitlab.example.com", fake)
	if err != nil {
		t.Errorf("NewClientWithProjectAndExecutor failed: %v", err)
		return
	}

	if client.Host != "gitlab.example.com" {
		t.Errorf("expected Host='gitlab.example.com', got '%s'", client.Host)
	}
}

func TestNewClientWithNestedGroups(t *testing.T) {
	fake := NewFakeGitLabExecutor()
	fake.SetResponse("--version", "glab version 1.0.0")
	fake.SetResponse("auth status", "authenticated")

	client, err := NewClientWithProjectAndExecutor("group/subgroup", "project", "gitlab.com", fake)
	if err != nil {
		t.Errorf("NewClientWithProjectAndExecutor failed: %v", err)
		return
	}

	if client.Owner != "group/subgroup" {
		t.Errorf("expected Owner='group/subgroup', got '%s'", client.Owner)
	}
}
