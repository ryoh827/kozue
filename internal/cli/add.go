package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ryoh827/kozue/internal/gitx"
	"github.com/ryoh827/kozue/internal/worktree"
)

type addOptions struct {
	create  bool
	from    string
	path    string
	track   bool
	noTrack bool
	stash   bool
}

func newAddCommand(rootOpts *rootOptions) *cobra.Command {
	opts := &addOptions{}

	cmd := &cobra.Command{
		Use:   "add <branch>",
		Short: "Create a new git worktree (optionally creating a branch)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			branch := args[0]
			return runAdd(cmd, rootOpts, opts, branch)
		},
	}

	cmd.Flags().BoolVarP(&opts.create, "create", "b", false, "create a new branch for the worktree")
	cmd.Flags().StringVar(&opts.from, "from", "", "base ref when creating a new branch (default origin/main)")
	cmd.Flags().StringVar(&opts.path, "path", "", "explicit filesystem path for the worktree")
	cmd.Flags().BoolVar(&opts.track, "track", false, "force the branch to track origin/<branch>")
	cmd.Flags().BoolVar(&opts.noTrack, "no-track", false, "skip upstream configuration even if a remote branch exists")
	cmd.Flags().BoolVar(&opts.stash, "stash", false, "stash current changes before creating the worktree")

	return cmd
}

func runAdd(cmd *cobra.Command, rootOpts *rootOptions, opts *addOptions, branch string) error {
	runner := newRunner(cmd.ErrOrStderr(), rootOpts.verbose)
	ctx := cmd.Context()

	if opts.track && opts.noTrack {
		return exitErrorf(2, "cannot use --track together with --no-track")
	}

	repoRoot, err := detectRepoRoot(ctx, runner)
	if err != nil {
		return err
	}

	if err := ensureCleanIfRequired(ctx, runner, repoRoot, branch, opts.stash); err != nil {
		return err
	}

	entries, err := worktree.ListEntries(ctx, runner, repoRoot)
	if err != nil {
		return combineErrors(1, err)
	}

	targetPath := opts.path
	if targetPath == "" {
		existsMap := make(map[string]struct{}, len(entries))
		for _, e := range entries {
			existsMap[filepath.Clean(e.Path)] = struct{}{}
		}

		targetPath, err = worktree.DefaultWorktreePath(repoRoot, branch, func(path string) bool {
			if _, ok := existsMap[filepath.Clean(path)]; ok {
				return true
			}
			_, statErr := os.Stat(path)
			return statErr == nil
		})
		if err != nil {
			return combineErrors(1, err)
		}
	}

	if err := gitFetch(ctx, runner, repoRoot); err != nil {
		return combineErrors(1, err)
	}

	if opts.create {
		if err := gitWorktreeAddCreate(ctx, runner, repoRoot, branch, targetPath, opts.from); err != nil {
			return combineErrors(1, err)
		}
	} else {
		if err := gitWorktreeAddExisting(ctx, runner, repoRoot, branch, targetPath); err != nil {
			return combineErrors(1, err)
		}
	}

	if shouldConfigureUpstream(ctx, runner, repoRoot, branch, opts) {
		if err := gitSetUpstream(ctx, runner, targetPath, branch); err != nil {
			return combineErrors(1, err)
		}
	}

	fmt.Fprintln(cmd.OutOrStdout(), targetPath)
	return nil
}

func ensureCleanIfRequired(ctx context.Context, runner *gitx.Runner, repoRoot, branch string, autoStash bool) error {
	status, err := runner.Run(ctx, repoRoot, "status", "--porcelain", "-uno")
	if err != nil {
		return combineErrors(1, err)
	}

	if strings.TrimSpace(status) == "" {
		return nil
	}

	if !autoStash {
		return exitErrorf(2, "uncommitted changes present; commit/stash or pass --stash before add")
	}

	message := fmt.Sprintf("koz/%s", branch)
	if _, err := runner.Run(ctx, repoRoot, "stash", "push", "-u", "-m", message); err != nil {
		return combineErrors(1, err)
	}

	return nil
}

func gitFetch(ctx context.Context, runner *gitx.Runner, repoRoot string) error {
	_, err := runner.Run(ctx, repoRoot, "fetch", "--prune")
	return err
}

func gitWorktreeAddCreate(ctx context.Context, runner *gitx.Runner, repoRoot, branch, path, start string) error {
	if start == "" {
		start = "origin/main"
	}

	_, err := runner.Run(ctx, repoRoot, "worktree", "add", "-b", branch, path, start)
	return err
}

func gitWorktreeAddExisting(ctx context.Context, runner *gitx.Runner, repoRoot, branch, path string) error {
	_, err := runner.Run(ctx, repoRoot, "worktree", "add", path, branch)
	return err
}

func gitSetUpstream(ctx context.Context, runner *gitx.Runner, path, branch string) error {
	remoteRef := fmt.Sprintf("origin/%s", branch)
	_, err := runner.Run(ctx, path, "branch", "--set-upstream-to", remoteRef)
	return err
}

func shouldConfigureUpstream(ctx context.Context, runner *gitx.Runner, repoRoot, branch string, opts *addOptions) bool {
	if opts.track {
		return true
	}
	if opts.noTrack {
		return false
	}
	return remoteBranchExists(ctx, runner, repoRoot, branch)
}

func remoteBranchExists(ctx context.Context, runner *gitx.Runner, repoRoot, branch string) bool {
	ref := fmt.Sprintf("origin/%s", branch)
	if _, err := runner.Run(ctx, repoRoot, "rev-parse", "--verify", "--quiet", ref); err != nil {
		return false
	}
	return true
}
