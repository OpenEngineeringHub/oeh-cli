package cmd

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// VerificationStep is a single check defined in the task spec.
type VerificationStep struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	Type           string `json:"type"` // http | process | file | command | benchmark
	URL            string `json:"url,omitempty"`
	Method         string `json:"method,omitempty"`
	ExpectedStatus int    `json:"expected_status,omitempty"`
	ProcessName    string `json:"process_name,omitempty"`
	FilePath       string `json:"file_path,omitempty"`
	Command        string `json:"command,omitempty"`
	OutputPattern  string `json:"output_pattern,omitempty"`
	Optional       bool   `json:"optional,omitempty"`
}

// StepResult is the result of running one verification step.
type StepResult struct {
	StepID    string
	Label     string
	Passed    bool
	Evidence  string
	Error     string
	DurationMs int64
}

var verifyCmd = &cobra.Command{
	Use:     "verify",
	Aliases: []string{"test"},
	Short:   "Run local verification checks for the current task",
	Long: `Run all verification steps defined in the current task spec locally.

This gives you fast feedback before submitting to the platform.
All checks run on your machine — nothing is sent to OEH.

Example:
  oeh task start ie-ch03-lab-001
  # do the work...
  oeh test
  # or
  oeh verify`,
	RunE: runVerify,
}

func runVerify(cmd *cobra.Command, args []string) error {
	state, spec, err := loadWorkspace()
	if err != nil {
		return fmt.Errorf("no task found — run: oeh task start <TASK-ID>")
	}

	fmt.Println()
	printHeader()
	fmt.Printf("  Verifying: %s\n", state.TaskID)
	fmt.Printf("  Task:      %s\n", spec.Title)
	fmt.Println()

	if len(spec.Steps) == 0 {
		printWarn("No verification steps defined for this task.")
		fmt.Println()
		return nil
	}

	results := make([]StepResult, 0, len(spec.Steps))

	for _, step := range spec.Steps {
		fmt.Printf("  Checking: %s...\n", step.Label)
		result := runStep(step)
		results = append(results, result)

		if result.Passed {
			printCheckPass(step.Label, result.Evidence)
		} else {
			if step.Optional {
				printCheckWarn(step.Label, result.Error+" (optional)")
			} else {
				printCheckFail(step.Label, result.Error)
			}
		}
	}

	fmt.Println()

	// Summary
	passed := 0
	failed := 0
	for _, r := range results {
		if r.Passed {
			passed++
		} else {
			failed++
		}
	}

	allRequired := true
	for i, r := range results {
		if !r.Passed && !spec.Steps[i].Optional {
			allRequired = false
			break
		}
	}

	fmt.Printf("  Results: %d/%d checks passed\n\n", passed, len(results))

	if allRequired {
		printBanner(fmt.Sprintf("VERIFICATION PASSED  (%d/%d)", passed, len(results)))
		fmt.Println()
		fmt.Println("  Ready to submit! Run:")
		fmt.Println("  oeh submit")
	} else {
		printBanner(fmt.Sprintf("VERIFICATION FAILED  (%d/%d)", passed, len(results)))
		fmt.Println()
		fmt.Println("  Fix the failing checks above and run oeh verify again.")
	}

	fmt.Println()
	return nil
}

// ─── Step runners ─────────────────────────────────────────────────────────────

func runStep(step VerificationStep) StepResult {
	start := time.Now()
	result := StepResult{StepID: step.ID, Label: step.Label}

	switch step.Type {
	case "http":
		result = runHTTPCheck(step)
	case "process":
		result = runProcessCheck(step)
	case "file":
		result = runFileCheck(step)
	case "command":
		result = runCommandCheck(step)
	default:
		result.Passed = false
		result.Error = "unknown check type: " + step.Type
	}

	result.DurationMs = time.Since(start).Milliseconds()
	return result
}

func runHTTPCheck(step VerificationStep) StepResult {
	r := StepResult{StepID: step.ID, Label: step.Label}

	method := step.Method
	if method == "" {
		method = "GET"
	}
	url := step.URL
	if url == "" {
		r.Error = "no URL configured for http check"
		return r
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		r.Error = fmt.Sprintf("request failed: %v", err)
		return r
	}
	defer resp.Body.Close()

	expected := step.ExpectedStatus
	if expected == 0 {
		expected = 200
	}

	if resp.StatusCode == expected {
		r.Passed = true
		r.Evidence = fmt.Sprintf("HTTP %d from %s", resp.StatusCode, url)
	} else {
		r.Error = fmt.Sprintf("expected HTTP %d, got %d from %s", expected, resp.StatusCode, url)
	}
	return r
}

func runProcessCheck(step VerificationStep) StepResult {
	r := StepResult{StepID: step.ID, Label: step.Label}

	name := step.ProcessName
	if name == "" {
		r.Error = "no process_name configured"
		return r
	}

	// Try running the binary with --version or version
	for _, flag := range []string{"--version", "version", "-v"} {
		out, err := exec.Command(name, flag).Output()
		if err == nil {
			r.Passed = true
			r.Evidence = name + " detected: " + strings.Split(strings.TrimSpace(string(out)), "\n")[0]
			return r
		}
	}

	// Try plain execution
	_, err := exec.LookPath(name)
	if err == nil {
		r.Passed = true
		r.Evidence = name + " found in PATH"
		return r
	}

	r.Error = name + " not found in PATH"
	return r
}

func runFileCheck(step VerificationStep) StepResult {
	r := StepResult{StepID: step.ID, Label: step.Label}
	// Basic file existence check would use os.Stat; simplified here
	r.Error = "file check: specify file_path in task spec"
	return r
}

func runCommandCheck(step VerificationStep) StepResult {
	r := StepResult{StepID: step.ID, Label: step.Label}
	if step.Command == "" {
		r.Error = "no command configured"
		return r
	}

	parts := strings.Fields(step.Command)
	out, err := exec.Command(parts[0], parts[1:]...).CombinedOutput()
	output := strings.TrimSpace(string(out))

	if err != nil {
		r.Error = fmt.Sprintf("command failed: %v — %s", err, output)
		return r
	}

	if step.OutputPattern != "" && !strings.Contains(output, step.OutputPattern) {
		r.Error = fmt.Sprintf("output does not contain expected pattern: %q", step.OutputPattern)
		return r
	}

	r.Passed = true
	if output != "" {
		lines := strings.Split(output, "\n")
		r.Evidence = lines[0]
	} else {
		r.Evidence = "command exited 0"
	}
	return r
}
