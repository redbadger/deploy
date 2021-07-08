package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

const completionDesc = `
Generate autocompletions script for deploy for the specified shell (bash or zsh).

This command can generate shell autocompletions. e.g.

	$ deploy completion bash

Can be sourced as such

	$ source <(deploy completion bash)
`

var (
	completionShells = map[string]func(out io.Writer, cmd *cobra.Command) error{
		"bash": runCompletionBash,
		"zsh":  runCompletionZsh,
	}
)

func init() {
	rootCmd.AddCommand(newCompletionCmd(os.Stdout))
}

func newCompletionCmd(out io.Writer) *cobra.Command {
	shells := []string{}
	for s := range completionShells {
		shells = append(shells, s)
	}

	cmd := &cobra.Command{
		Use:   "completion SHELL",
		Short: "Generate autocompletions script for the specified shell (bash or zsh)",
		Long:  completionDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCompletion(out, cmd, args)
		},
		ValidArgs: shells,
	}

	return cmd
}

func runCompletion(out io.Writer, cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("shell not specified")
	}
	if len(args) > 1 {
		return fmt.Errorf("too many arguments, expected only the shell type")
	}
	run, found := completionShells[args[0]]
	if !found {
		return fmt.Errorf("unsupported shell type %q", args[0])
	}

	return run(out, cmd)
}

func runCompletionBash(out io.Writer, cmd *cobra.Command) error {
	return cmd.Root().GenBashCompletion(out)
}

func runCompletionZsh(out io.Writer, cmd *cobra.Command) error {
	return cmd.Root().GenZshCompletion(out)
}
