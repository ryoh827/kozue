package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/ryoh827/kozue/internal/gitx"
	"github.com/ryoh827/kozue/internal/repo"
)

func newRunner(writer io.Writer, verbose bool) *gitx.Runner {
	return gitx.NewRunner(verbose, writer)
}

func detectRepoRoot(ctx context.Context, runner *gitx.Runner) (string, error) {
	root, err := repo.DetectRoot(ctx, runner, "")
	if err != nil {
		if strings.Contains(err.Error(), "not a git repository") {
			return "", exitErrorf(2, "not a git repository (run inside a repo root or subdir)")
		}
		return "", exitError(2, err)
	}
	return root, nil
}

func cwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

func combineErrors(code int, err error) error {
	var exit *ExitError
	if errors.As(err, &exit) {
		return err
	}
	return exitError(code, err)
}
