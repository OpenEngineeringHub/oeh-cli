// Package cmd wires all CLI subcommands.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const cliVersion = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "oeh",
	Short: "Open Engineering Hub CLI",
	Long: `oeh — The Open Engineering Hub command-line tool.

Verify tasks, submit labs and projects, and connect your local
workspace to the OEH learning platform.

Get started:
  oeh login       Authenticate with your OEH account
  oeh doctor      Check your environment
  oeh task start  Start a task
  oeh verify      Run local verification
  oeh submit      Submit to the platform`,
	Version: cliVersion,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(
		loginCmd,
		logoutCmd,
		doctorCmd,
		taskCmd,
		workspaceCmd,
		verifyCmd,
		submitCmd,
		submissionCmd,
		mentorCmd,
		progressCmd,
	)
}
