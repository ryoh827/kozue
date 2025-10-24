package cli

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ryoh827/kozue/internal/worktree"
)

type removeOptions struct {
	force       bool
	iUnderstand bool
	archivePath string
}

func newRemoveCommand(rootOpts *rootOptions) *cobra.Command {
	opts := &removeOptions{}

	cmd := &cobra.Command{
		Use:   "rm <name|path>",
		Short: "Remove a git worktree safely",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			return runRemove(cmd, rootOpts, opts, target)
		},
	}

	cmd.Flags().BoolVar(&opts.force, "force", false, "allow removal despite safety checks")
	cmd.Flags().BoolVar(&opts.iUnderstand, "i-understand", false, "acknowledge the destructive action when used with --force")
	cmd.Flags().StringVar(&opts.archivePath, "archive", "", "zip the worktree to the provided path before deletion")

	return cmd
}

func runRemove(cmd *cobra.Command, rootOpts *rootOptions, opts *removeOptions, target string) error {
	runner := newRunner(cmd.ErrOrStderr(), rootOpts.verbose)
	ctx := cmd.Context()

	repoRoot, err := detectRepoRoot(ctx, runner)
	if err != nil {
		return err
	}

	entries, err := worktree.ListEntries(ctx, runner, repoRoot)
	if err != nil {
		return combineErrors(1, err)
	}

	targetEntry, err := resolveWorktree(target, entries)
	if err != nil {
		return err
	}

	if filepath.Clean(targetEntry.Path) == filepath.Clean(repoRoot) {
		return exitErrorf(2, "refusing to remove the main worktree (%s)", repoRoot)
	}

	dirtyCount, err := worktree.DirtyCount(ctx, runner, targetEntry.Path)
	if err != nil {
		return combineErrors(1, err)
	}

	ahead, _, err := worktree.AheadBehind(ctx, runner, targetEntry.Path)
	if err != nil {
		return combineErrors(1, err)
	}

	hazards := hazardsFor(targetEntry, dirtyCount, ahead)
	if len(hazards) > 0 && !(opts.force && opts.iUnderstand) {
		return exitErrorf(2, "cannot remove: %s. use --force --i-understand to override.", strings.Join(hazards, "; "))
	}

	if opts.archivePath != "" {
		if err := archiveWorktree(targetEntry.Path, opts.archivePath); err != nil {
			return combineErrors(1, err)
		}
	}

	args := []string{"worktree", "remove"}
	if opts.force {
		args = append(args, "--force")
	}
	args = append(args, targetEntry.Path)

	if _, err := runner.Run(ctx, repoRoot, args...); err != nil {
		return combineErrors(1, err)
	}

	summary := fmt.Sprintf("removed %s", targetEntry.Path)
	if opts.archivePath != "" {
		summary = fmt.Sprintf("%s (archived to %s)", summary, opts.archivePath)
	}

	fmt.Fprintln(cmd.OutOrStdout(), summary)
	return nil
}

func resolveWorktree(target string, entries []worktree.Entry) (worktree.Entry, error) {
	cleanTarget := filepath.Clean(target)
	if stat, err := os.Stat(cleanTarget); err == nil && stat.IsDir() {
		for _, entry := range entries {
			if filepath.Clean(entry.Path) == cleanTarget {
				return entry, nil
			}
		}
	}

	matches := make([]worktree.Entry, 0)
	for _, entry := range entries {
		if entry.Branch == target {
			matches = append(matches, entry)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return worktree.Entry{}, exitErrorf(2, "multiple worktrees match branch %q", target)
	}

	matches = matches[:0]
	for _, entry := range entries {
		if entry.Basename() == target {
			matches = append(matches, entry)
		}
	}

	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return worktree.Entry{}, exitErrorf(2, "no worktree matches %q", target)
	default:
		names := make([]string, len(matches))
		for i, entry := range matches {
			names[i] = entry.Path
		}
		return worktree.Entry{}, exitErrorf(2, "ambiguous worktree reference %q (%s)", target, strings.Join(names, ", "))
	}
}

func hazardsFor(entry worktree.Entry, dirty int, ahead *int) []string {
	var hazards []string
	if entry.Locked {
		reason := entry.LockReason
		if reason != "" {
			hazards = append(hazards, fmt.Sprintf("worktree is locked (reason: %s)", reason))
		} else {
			hazards = append(hazards, "worktree is locked")
		}
	}
	if dirty > 0 {
		hazards = append(hazards, "uncommitted changes present")
	}
	if ahead != nil && *ahead > 0 {
		hazards = append(hazards, fmt.Sprintf("ahead of upstream by %d", *ahead))
	}
	return hazards
}

func archiveWorktree(source, destination string) error {
	if source == "" {
		return errors.New("cannot archive empty source path")
	}

	if _, err := os.Stat(destination); err == nil {
		return fmt.Errorf("archive path %s already exists", destination)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if _, err := os.Stat(source); err != nil {
		return err
	}

	file, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	defer zw.Close()

	baseParent := filepath.Dir(source)

	walkFn := func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(baseParent, path)
		if err != nil {
			return err
		}

		rel = filepath.ToSlash(rel)
		info, err := d.Info()
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = rel

		if d.IsDir() {
			header.Name += "/"
			_, err = zw.CreateHeader(header)
			return err
		}

		if info.Mode()&os.ModeSymlink != 0 {
			header.SetMode(info.Mode())
			header.Method = zip.Store
			w, err := zw.CreateHeader(header)
			if err != nil {
				return err
			}
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			_, err = w.Write([]byte(target))
			return err
		}

		header.Method = zip.Deflate
		w, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(w, f)
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	}

	if err := filepath.WalkDir(source, walkFn); err != nil {
		return err
	}

	return nil
}
