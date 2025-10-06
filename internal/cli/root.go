package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewRootCommand constructs the top-level command for koz.
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "koz",
		Short: "Helper for comfortable Git worktree operations",
		Long:  "koz is a helper tool designed to make working with Git worktrees more comfortable.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	return cmd
}

// Execute runs the root command and returns an exit code suited for os.Exit.
func Execute() int {
	if err := NewRootCommand().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	return 0
}
