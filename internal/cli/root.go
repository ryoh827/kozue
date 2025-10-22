package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewRootCommand constructs the top-level command for koz.
func NewRootCommand() *cobra.Command {
	opts := &rootOptions{}

	cmd := &cobra.Command{
		Use:           "koz",
		Short:         "Helper for comfortable Git worktree operations",
		Long:          "koz is a helper tool designed to make working with Git worktrees more comfortable.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.SetContext(context.Background())

	cmd.PersistentFlags().BoolVar(&opts.verbose, "verbose", false, "echo git commands to stderr")

	cmd.AddCommand(newListCommand(opts))
	cmd.AddCommand(newAddCommand(opts))
	cmd.AddCommand(newRemoveCommand(opts))

	return cmd
}

// Execute runs the root command and returns an exit code suited for os.Exit.
func Execute() int {
	err := NewRootCommand().Execute()
	if err == nil {
		return 0
	}

	if exitErr, ok := err.(*ExitError); ok {
		if exitErr.Err != nil {
			fmt.Fprintln(os.Stderr, exitErr.Error())
		}
		return exitErr.Code
	}

	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	return 1
}

type rootOptions struct {
	verbose bool
}
