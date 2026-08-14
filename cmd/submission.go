package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var submissionCmd = &cobra.Command{
	Use:   "submission",
	Short: "Submission management commands",
}

var submissionStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the status of your latest submission",
	RunE:  runSubmissionStatus,
}

var progressCmd = &cobra.Command{
	Use:   "progress",
	Short: "Show your OEH learning progress",
	RunE:  runProgress,
}

func init() {
	submissionCmd.AddCommand(submissionStatusCmd)
}

func runSubmissionStatus(cmd *cobra.Command, args []string) error {
	state, _, err := loadWorkspace()
	if err != nil {
		return fmt.Errorf("no task found — run: oeh task start <TASK-ID>")
	}

	// Try to read the latest local submission if offline
	sub := readLatestLocalSubmission()
	fmt.Println()
	printHeader()
	fmt.Printf("  Submission Status\n\n")
	fmt.Printf("  %-15s %s\n", "Task ID:", state.TaskID)

	if sub != nil {
		fmt.Printf("  %-15s %s\n", "Submitted:", sub.SubmittedAt)
		fmt.Printf("  %-15s %s\n", "Workspace:", sub.WorkspaceID)
		local := 0
		for _, r := range sub.Evidence.Results {
			if r.Passed {
				local++
			}
		}
		fmt.Printf("  %-15s %d/%d local checks\n", "Local:", local, len(sub.Evidence.Results))
		if sub.GitInfo != nil && sub.GitInfo.Commit != "" {
			fmt.Printf("  %-15s %s (%s)\n", "Commit:", sub.GitInfo.Commit, sub.GitInfo.Branch)
		}
	}

	fmt.Println()
	fmt.Println("  Check the OEH platform for authoritative results.")
	fmt.Println()
	return nil
}

func runProgress(cmd *cobra.Command, args []string) error {
	fmt.Println()
	printHeader()
	fmt.Println("  Progress tracking is available on the OEH platform.")
	fmt.Println()
	fmt.Println("  Visit: https://open-engineering-hub.dev/progress")
	fmt.Println()
	return nil
}

// ─── Local submission persistence ─────────────────────────────────────────────

type localSub struct {
	TaskID      string      `json:"task_id"`
	WorkspaceID string      `json:"workspace_id"`
	SubmittedAt string      `json:"submitted_at"`
	Evidence    OehEvidence `json:"evidence"`
	GitInfo     *GitInfo    `json:"git_info,omitempty"`
}

func readLatestLocalSubmission() *localSub {
	oehDir := ".oeh"
	entries, err := os.ReadDir(oehDir)
	if err != nil {
		return nil
	}

	var latest os.DirEntry
	var latestTime time.Time
	for _, e := range entries {
		if !e.IsDir() && len(e.Name()) > 11 && e.Name()[:10] == "submission" {
			info, err := e.Info()
			if err == nil && info.ModTime().After(latestTime) {
				latestTime = info.ModTime()
				latest = e
			}
		}
	}

	if latest == nil {
		return nil
	}

	data, err := os.ReadFile(filepath.Join(oehDir, latest.Name()))
	if err != nil {
		return nil
	}

	var sub localSub
	if err := json.Unmarshal(data, &sub); err != nil {
		return nil
	}
	return &sub
}
