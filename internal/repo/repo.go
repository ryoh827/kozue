package repo

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/ryoh827/kozue/internal/gitx"
)

// DetectRoot returns the git repository root from the provided directory.
func DetectRoot(ctx context.Context, runner *gitx.Runner, dir string) (string, error) {
	out, err := runner.Run(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}

	root := strings.TrimSpace(out)
	if root == "" {
		return "", errors.New("not a git repository")
	}
	return filepath.Clean(root), nil
}
