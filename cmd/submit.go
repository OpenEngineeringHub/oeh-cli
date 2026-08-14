package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// OehSubmission is the payload sent to the OEH platform engine.
type OehSubmission struct {
	TaskID     string      `json:"task_id"`
	Token      string      `json:"token"`
	WorkspaceID string     `json:"workspace_id"`
	Evidence   OehEvidence `json:"evidence"`
	GitInfo    *GitInfo    `json:"git_info,omitempty"`
	SubmittedAt string     `json:"submitted_at"`
	CLIVersion  string     `json:"cli_version"`
}

// OehEvidence contains local verification results.
type OehEvidence struct {
	Results     []EvidenceResult      `json:"verification_results"`
	Environment map[string]string     `json:"environment,omitempty"`
	Metrics     map[string]interface{} `json:"metrics,omitempty"`
}

// EvidenceResult is the result of one verification step.
type EvidenceResult struct {
	StepID     string `json:"step_id"`
	Passed     bool   `json:"passed"`
	Evidence   string `json:"evidence,omitempty"`
	Error      string `json:"error_message,omitempty"`
	DurationMs int64  `json:"duration_ms"`
}

// GitInfo contains Git context for the submission.
type GitInfo struct {
	Remote    string `json:"remote,omitempty"`
	Branch    string `json:"branch,omitempty"`
	Commit    string `json:"commit,omitempty"`
	RepoURL   string `json:"repo_url,omitempty"`
}

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit current task to the OEH platform for authoritative verification",
	Long: `Runs all local checks, collects evidence, and submits to the OEH platform.

The platform will run authoritative verification (including hidden tests)
and update your progress and XP.

Example:
  oeh verify   # check locally first
  oeh submit   # submit when ready`,
	RunE: runSubmit,
}

func runSubmit(cmd *cobra.Command, args []string) error {
	state, spec, err := loadWorkspace()
	if err != nil {
		return fmt.Errorf("no task found — run: oeh task start <TASK-ID>")
	}

	token := getToken()
	if token == "" {
		return fmt.Errorf("not authenticated — run: oeh login --token <TOKEN>")
	}

	fmt.Println()
	printHeader()
	fmt.Printf("  Submitting: %s\n", state.TaskID)
	fmt.Println()

	// 1. Run local checks for evidence
	printStep("Running local verification...")
	results := make([]EvidenceResult, 0, len(spec.Steps))
	passed := 0
	for _, step := range spec.Steps {
		r := runStep(step)
		er := EvidenceResult{
			StepID:     r.StepID,
			Passed:     r.Passed,
			Evidence:   r.Evidence,
			Error:      r.Error,
			DurationMs: r.DurationMs,
		}
		results = append(results, er)
		if r.Passed {
			passed++
			printCheckPass(step.Label, r.Evidence)
		} else {
			printCheckFail(step.Label, r.Error)
		}
	}

	total := len(spec.Steps)
	fmt.Printf("\n  Local: %d/%d checks passed\n\n", passed, total)

	if passed == 0 && total > 0 {
		printWarn("No checks passed locally. Submit anyway? (Ctrl+C to cancel)")
		time.Sleep(2 * time.Second)
	}

	// 2. Collect Git info
	printStep("Collecting workspace info...")
	gitInfo := collectGitInfo()
	if gitInfo != nil {
		fmt.Printf("  Repository: %s\n", gitInfo.Remote)
		fmt.Printf("  Branch:     %s\n", gitInfo.Branch)
		fmt.Printf("  Commit:     %s\n", gitInfo.Commit)
	} else {
		printWarn("Not a Git repository. Consider: git init && git add . && git commit -m 'initial'")
	}

	// 3. Build submission payload
	submission := OehSubmission{
		TaskID:      state.TaskID,
		Token:       token,
		WorkspaceID: state.WorkspaceID,
		Evidence: OehEvidence{
			Results: results,
			Environment: map[string]string{
				"cli_version": cliVersion,
				"os":          getOS(),
			},
		},
		GitInfo:     gitInfo,
		SubmittedAt: time.Now().UTC().Format(time.RFC3339),
		CLIVersion:  cliVersion,
	}

	// 4. Submit
	printStep("Submitting to OEH platform...")
	submissionID, err := postSubmission(submission)
	if err != nil {
		// Offline mode: save locally and show instructions
		printWarn("Could not reach OEH platform — saving submission locally")
		saveLocalSubmission(submission)
		fmt.Println()
		fmt.Println("  When you're online, run:")
		fmt.Println("  oeh submit")
		fmt.Println()
		return nil
	}

	fmt.Println()
	printBanner("SUBMISSION RECEIVED")
	fmt.Println()
	fmt.Printf("  Submission ID:  %s\n", submissionID)
	fmt.Printf("  Task:           %s\n", state.TaskID)
	fmt.Printf("  Local:          %d/%d checks\n", passed, total)
	fmt.Println()
	fmt.Println("  Platform is running authoritative verification.")
	fmt.Println("  Check results on the OEH platform or run:")
	fmt.Println("  oeh submission status")
	fmt.Println()
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func collectGitInfo() *GitInfo {
	commit, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return nil
	}
	branch, _ := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	remote, _ := exec.Command("git", "remote", "get-url", "origin").Output()

	return &GitInfo{
		Commit:  strings.TrimSpace(string(commit)),
		Branch:  strings.TrimSpace(string(branch)),
		Remote:  strings.TrimSpace(string(remote)),
		RepoURL: strings.TrimSpace(string(remote)),
	}
}

func postSubmission(sub OehSubmission) (string, error) {
	platformURL := getPlatformURL()
	url := platformURL + "/api/submissions"

	body, _ := json.Marshal(sub)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		SubmissionID string `json:"submission_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Sprintf("sub_%d", time.Now().Unix()), nil
	}
	return result.SubmissionID, nil
}

func saveLocalSubmission(sub OehSubmission) {
	data, _ := json.MarshalIndent(sub, "", "  ")
	filename := fmt.Sprintf(".oeh/submission_%d.json", time.Now().Unix())
	_ = os.WriteFile(filename, data, 0600)
	fmt.Printf("  Saved to %s\n", filename)
}

func getOS() string {
	out, _ := exec.Command("uname", "-s").Output()
	return strings.TrimSpace(string(out))
}
