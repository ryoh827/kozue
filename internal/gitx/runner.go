package gitx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

const defaultTimeout = 3 * time.Second

// Runner encapsulates git command execution with timeout and diagnostics.
type Runner struct {
	Timeout time.Duration
	Verbose bool
	Stderr  io.Writer
}

// NewRunner builds a Runner with timeout sourced from KOZ_GIT_TIMEOUT.
func NewRunner(verbose bool, stderr io.Writer) *Runner {
	return &Runner{
		Timeout: readTimeoutEnv(),
		Verbose: verbose,
		Stderr:  stderr,
	}
}

// Run executes git with the provided working directory.
func (r *Runner) Run(ctx context.Context, dir string, args ...string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	timeout := r.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if r.Verbose {
		echo := fmt.Sprintf("[git] dir=%s args=%s\n", dir, strings.Join(args, " "))
		if r.Stderr != nil {
			io.WriteString(r.Stderr, echo)
		} else {
			fmt.Fprint(os.Stderr, echo)
		}
	}

	err := cmd.Run()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("git %v: timeout after %s", args, timeout)
		}
		return "", fmt.Errorf("git %v: %w\n%s", args, err, stderr.String())
	}

	return stdout.String(), nil
}

func readTimeoutEnv() time.Duration {
	val := os.Getenv("KOZ_GIT_TIMEOUT")
	if val == "" {
		return defaultTimeout
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return defaultTimeout
	}
	return d
}
