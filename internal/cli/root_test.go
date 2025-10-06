package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewRootCommandMetadata(t *testing.T) {
	cmd := NewRootCommand()

	if cmd.Use != "koz" {
		t.Fatalf("expected Use to be 'koz', got %q", cmd.Use)
	}

	if !strings.Contains(cmd.Short, "Git worktree") {
		t.Fatalf("expected Short to mention Git worktree, got %q", cmd.Short)
	}

	if !strings.Contains(cmd.Long, "worktrees") {
		t.Fatalf("expected Long to mention worktrees, got %q", cmd.Long)
	}
}

func TestNewRootCommandHelpExecution(t *testing.T) {
	cmd := NewRootCommand()

	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected help execution to succeed, got error %v", err)
	}

	text := output.String()
	if !strings.Contains(text, "koz is a helper tool") {
		t.Fatalf("expected help text to describe koz, got %q", text)
	}
}
