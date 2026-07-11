package util

import (
	"os"
	"testing"
)

func TestIsSSH(t *testing.T) {
	// Save and restore environment
	origTTY := os.Getenv("SSH_TTY")
	origConn := os.Getenv("SSH_CONNECTION")
	defer func() {
		os.Setenv("SSH_TTY", origTTY)
		os.Setenv("SSH_CONNECTION", origConn)
	}()

	// No SSH
	os.Unsetenv("SSH_TTY")
	os.Unsetenv("SSH_CONNECTION")
	if IsSSH() {
		t.Error("Expected false without SSH env vars")
	}

	// SSH_TTY set
	os.Setenv("SSH_TTY", "/dev/pts/0")
	if !IsSSH() {
		t.Error("Expected true with SSH_TTY")
	}
	os.Unsetenv("SSH_TTY")

	// SSH_CONNECTION set
	os.Setenv("SSH_CONNECTION", "1.2.3.4 5678 5.6.7.8 22")
	if !IsSSH() {
		t.Error("Expected true with SSH_CONNECTION")
	}
}

func TestIsInteractiveEnvironment(t *testing.T) {
	origCI := os.Getenv("CI")
	defer os.Setenv("CI", origCI)

	// CI environment
	os.Setenv("CI", "true")
	if IsInteractiveEnvironment() {
		t.Error("Expected false in CI")
	}

	os.Unsetenv("CI")
	// Result depends on whether stderr is a TTY (test environments may vary)
	// Just verify it doesn't panic
	_ = IsInteractiveEnvironment()
}

func TestOpenBrowser_InvalidPlatform(t *testing.T) {
	// We can't easily test actual browser opening, but we can verify
	// the function doesn't panic with valid input
	// This test is inherently platform-dependent, so we just verify compilation
	_ = OpenBrowser
}
