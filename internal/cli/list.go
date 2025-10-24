package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"text/tabwriter"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/ryoh827/kozue/internal/gitx"
	"github.com/ryoh827/kozue/internal/worktree"
)

type listOptions struct {
	json bool
}

func newListCommand(rootOpts *rootOptions) *cobra.Command {
	opts := &listOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List git worktrees managed alongside the repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, rootOpts, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.json, "json", false, "emit machine-readable JSON")

	return cmd
}

func runList(cmd *cobra.Command, rootOpts *rootOptions, opts *listOptions) error {
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

	details := measureWorktrees(ctx, cmd, runner, entries)

	if opts.json {
		return renderListJSON(cmd, repoRoot, details)
	}

	renderListTable(cmd, repoRoot, details)
	return nil
}

func measureWorktrees(ctx context.Context, cmd *cobra.Command, runner *gitx.Runner, entries []worktree.Entry) []worktreeDetail {
	details := make([]worktreeDetail, len(entries))

	var wg sync.WaitGroup
	jobs := make(chan int)

	workers := recommendedWorkers(len(entries))
	if workers == 0 {
		return details
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				detail := worktreeDetail{Entry: entries[idx]}

				if dirty, err := worktree.DirtyCount(ctx, runner, entries[idx].Path); err != nil {
					warn(cmd, "dirty", entries[idx].Path, err)
				} else {
					detail.DirtyFiles = &dirty
				}

				if ahead, behind, err := worktree.AheadBehind(ctx, runner, entries[idx].Path); err != nil {
					warn(cmd, "ahead/behind", entries[idx].Path, err)
				} else {
					detail.Ahead = ahead
					detail.Behind = behind
				}

				details[idx] = detail
			}
		}()
	}

	for i := range entries {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	return details
}

func recommendedWorkers(n int) int {
	if n <= 0 {
		return 0
	}
	limit := runtime.GOMAXPROCS(0)
	if limit < 2 {
		limit = 2
	}
	if limit > n {
		limit = n
	}
	return limit
}

func warn(cmd *cobra.Command, subject, path string, err error) {
	fmt.Fprintf(cmd.ErrOrStderr(), "warning: unable to capture %s for %s: %v\n", subject, path, err)
}

type worktreeDetail struct {
	worktree.Entry
	DirtyFiles *int
	Ahead      *int
	Behind     *int
}

func renderListTable(cmd *cobra.Command, repoRoot string, details []worktreeDetail) {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "PATH\tBRANCH\tHEAD\tDIRTY\tAHEAD/BEHIND\tFLAGS")

	for _, detail := range details {
		flags := buildFlags(detail.Entry)
		branch := detail.Branch
		if branch == "" && detail.Detached {
			branch = "(detached)"
		}

		aheadBehind := "-"
		if detail.Ahead != nil || detail.Behind != nil {
			a := 0
			b := 0
			if detail.Ahead != nil {
				a = *detail.Ahead
			}
			if detail.Behind != nil {
				b = *detail.Behind
			}
			aheadBehind = fmt.Sprintf("%d/%d", a, b)
		}

		dirty := "-"
		if detail.DirtyFiles != nil {
			dirty = fmt.Sprintf("%d", *detail.DirtyFiles)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			truncatePath(repoRoot, detail.Path, 60),
			branch,
			detail.HeadShort(),
			dirty,
			aheadBehind,
			flags,
		)
	}
	w.Flush()
}

func buildFlags(entry worktree.Entry) string {
	var b strings.Builder
	if entry.Detached {
		b.WriteString("D")
	}
	if entry.Locked {
		b.WriteString("L")
	}
	return b.String()
}

func truncatePath(repoRoot, path string, width int) string {
	rel := path
	sep := string(os.PathSeparator)
	if repoRoot != "" {
		prefix := repoRoot + sep
		if strings.HasPrefix(path, prefix) {
			rel = path[len(prefix):]
		}
	}

	if width <= 0 {
		return rel
	}

	if utf8.RuneCountInString(rel) <= width {
		return rel
	}

	runes := []rune(rel)
	return "…" + string(runes[len(runes)-width+1:])
}

func renderListJSON(cmd *cobra.Command, repoRoot string, details []worktreeDetail) error {
	payload := struct {
		SchemaVersion int               `json:"schema_version"`
		GeneratedAt   string            `json:"generated_at"`
		RepoRoot      string            `json:"repo_root"`
		Worktrees     []jsonWorktreeRow `json:"worktrees"`
	}{
		SchemaVersion: 1,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		RepoRoot:      repoRoot,
		Worktrees:     make([]jsonWorktreeRow, len(details)),
	}

	for i, detail := range details {
		row := jsonWorktreeRow{
			Path:       detail.Path,
			Head:       detail.Head,
			HeadShort:  detail.HeadShort(),
			Branch:     detail.Branch,
			Detached:   detail.Detached,
			Locked:     detail.Locked,
			LockReason: detail.LockReason,
			Bare:       detail.Bare,
		}
		if detail.DirtyFiles != nil {
			value := *detail.DirtyFiles
			row.DirtyFiles = &value
		}
		if detail.Ahead != nil {
			value := *detail.Ahead
			row.Ahead = &value
		}
		if detail.Behind != nil {
			value := *detail.Behind
			row.Behind = &value
		}
		payload.Worktrees[i] = row
	}

	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

type jsonWorktreeRow struct {
	Path       string `json:"path"`
	Head       string `json:"head"`
	HeadShort  string `json:"head_short"`
	Branch     string `json:"branch"`
	Detached   bool   `json:"detached"`
	Locked     bool   `json:"locked"`
	LockReason string `json:"lock_reason"`
	Bare       bool   `json:"bare"`
	DirtyFiles *int   `json:"dirty_files"`
	Ahead      *int   `json:"ahead"`
	Behind     *int   `json:"behind"`
}
