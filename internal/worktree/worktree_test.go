package worktree

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePorcelainList(t *testing.T) {
	output := strings.TrimSpace(`worktree /repo
HEAD 0123456789abcdef
branch refs/heads/main

worktree /repo/wt-1
HEAD fedcba9876543210
branch refs/heads/feature
locked reason: release freeze
`)

	trees, err := ParsePorcelainList(output)
	if err != nil {
		t.Fatalf("expected parse to succeed: %v", err)
	}

	if len(trees) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(trees))
	}

	if !trees[1].Locked || trees[1].LockReason != "release freeze" {
		t.Fatalf("expected locked worktree with reason, got %+v", trees[1])
	}
}

func TestSanitizeBranch(t *testing.T) {
	if got := SanitizeBranch("refs/heads/feature/cart cleanup"); got != "feature-cart-cleanup" {
		t.Fatalf("unexpected sanitize result: %q", got)
	}

	longName := "refs/heads/" + strings.Repeat("a", 100)
	if got := SanitizeBranch(longName); len(got) > 64 {
		t.Fatalf("sanitize should truncate to 64 chars, got %d", len(got))
	}
}

func TestDefaultWorktreePath(t *testing.T) {
	repo := filepath.Join(string(filepath.Separator), "tmp", "project")
	existing := map[string]struct{}{}

	exists := func(path string) bool {
		_, ok := existing[filepath.Clean(path)]
		return ok
	}

	first, err := DefaultWorktreePath(repo, "feature/login", exists)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	existing[filepath.Clean(first)] = struct{}{}

	second, err := DefaultWorktreePath(repo, "feature/login", exists)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first == second {
		t.Fatalf("expected unique fallback path, got %q", second)
	}

	if !strings.Contains(second, "-2") {
		t.Fatalf("expected suffix -2 in %q", second)
	}
}
