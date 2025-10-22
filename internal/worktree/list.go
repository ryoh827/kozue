package worktree

import (
	"context"
	"fmt"
	"strings"

	"github.com/ryoh827/kozue/internal/gitx"
)

// ListEntries fetches the worktree entries for the repository root.
func ListEntries(ctx context.Context, runner *gitx.Runner, repoRoot string) ([]Entry, error) {
	out, err := runner.Run(ctx, repoRoot, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	return ParsePorcelainList(out)
}

// DirtyCount returns the number of dirty files (`git status --porcelain -uno`).
func DirtyCount(ctx context.Context, runner *gitx.Runner, path string) (int, error) {
	out, err := runner.Run(ctx, path, "status", "--porcelain", "-uno")
	if err != nil {
		return 0, err
	}

	out = strings.TrimSpace(out)
	if out == "" {
		return 0, nil
	}

	return len(strings.Split(out, "\n")), nil
}

// AheadBehind returns counts for ahead/behind relative to upstream.
func AheadBehind(ctx context.Context, runner *gitx.Runner, path string) (ahead *int, behind *int, err error) {
	out, err := runner.Run(ctx, path, "rev-list", "--left-right", "--count", "@{u}...HEAD")
	if err != nil {
		if strings.Contains(err.Error(), "no upstream") {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	parts := strings.Fields(strings.TrimSpace(out))
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("unexpected rev-list output: %q", out)
	}

	behindCount, err := parseInt(parts[0])
	if err != nil {
		return nil, nil, err
	}
	aheadCount, err := parseInt(parts[1])
	if err != nil {
		return nil, nil, err
	}

	return &aheadCount, &behindCount, nil
}

func parseInt(value string) (int, error) {
	var n int
	_, err := fmt.Sscanf(value, "%d", &n)
	return n, err
}
