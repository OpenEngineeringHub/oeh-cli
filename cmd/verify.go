package cmd

import (
	"fmt"
	"net/http"
	"os"
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
	activeContainer := getActiveContainer(state.TaskID)
	if activeContainer != "" {
		fmt.Printf("  Runtime:   Linux Container [%s]\n", activeContainer)
	} else {
		fmt.Printf("  Runtime:   Host Machine [%s/%s]\n", getOS(), "local")
	}
	fmt.Println()

	if len(spec.Steps) == 0 {
		printWarn("No verification steps defined for this task.")
		fmt.Println()
		return nil
	}

	results := make([]StepResult, 0, len(spec.Steps))

	for _, step := range spec.Steps {
		fmt.Printf("  Checking: %s...\n", step.Label)
		result := runStep(step, activeContainer)
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

func runStep(step VerificationStep, activeContainer string) StepResult {
	start := time.Now()
	result := StepResult{StepID: step.ID, Label: step.Label}

	switch step.Type {
	case "http":
		result = runHTTPCheck(step)
	case "process":
		result = runProcessCheck(step, activeContainer)
	case "file":
		result = runFileCheck(step, activeContainer)
	case "command":
		result = runCommandCheck(step, activeContainer)
	default:
		result.Passed = false
		result.Error = "unknown check type: " + step.Type
	}

	result.DurationMs = time.Since(start).Milliseconds()
	return result
}

func getActiveContainer(taskID string) string {
	containerName := "oeh-ws-" + taskID
	out, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", containerName).Output()
	if err == nil && strings.TrimSpace(string(out)) == "true" {
		return containerName
	}
	return ""
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

func runProcessCheck(step VerificationStep, container string) StepResult {
	r := StepResult{StepID: step.ID, Label: step.Label}
	name := step.ProcessName
	if name == "" {
		r.Error = "no process_name configured"
		return r
	}

	if container != "" {
		out, err := exec.Command("docker", "exec", container, name, "--version").Output()
		if err == nil {
			r.Passed = true
			r.Evidence = fmt.Sprintf("%s in container: %s", name, strings.Split(strings.TrimSpace(string(out)), "\n")[0])
			return r
		}
		r.Error = fmt.Sprintf("%s not found inside container '%s'", name, container)
		return r
	}

	// Try running locally
	for _, flag := range []string{"--version", "version", "-v"} {
		out, err := exec.Command(name, flag).Output()
		if err == nil {
			r.Passed = true
			r.Evidence = name + " detected: " + strings.Split(strings.TrimSpace(string(out)), "\n")[0]
			return r
		}
	}

	_, err := exec.LookPath(name)
	if err == nil {
		r.Passed = true
		r.Evidence = name + " found in host PATH"
		return r
	}

	r.Error = name + " not found in PATH"
	return r
}

func runFileCheck(step VerificationStep, container string) StepResult {
	r := StepResult{StepID: step.ID, Label: step.Label}
	filePath := step.FilePath
	if filePath == "" {
		r.Error = "no file_path configured"
		return r
	}

	if container != "" {
		err := exec.Command("docker", "exec", container, "test", "-f", filePath).Run()
		if err == nil {
			r.Passed = true
			r.Evidence = "file exists in container: " + filePath
			return r
		}
		r.Error = "file missing in container: " + filePath
		return r
	}

	// Check local filesystem
	if _, err := os.Stat(filePath); err == nil {
		r.Passed = true
		r.Evidence = "file exists: " + filePath
	} else {
		r.Error = "file not found: " + filePath
	}
	return r
}

func runCommandCheck(step VerificationStep, container string) StepResult {
	r := StepResult{StepID: step.ID, Label: step.Label}
	if step.Command == "" {
		r.Error = "no command configured"
		return r
	}

	var out []byte
	var err error

	if container != "" {
		out, err = exec.Command("docker", "exec", container, "sh", "-c", step.Command).CombinedOutput()
	} else {
		parts := strings.Fields(step.Command)
		out, err = exec.Command(parts[0], parts[1:]...).CombinedOutput()
	}

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
	if container != "" {
		r.Evidence = fmt.Sprintf("[container] %s", firstLine(output))
	} else if output != "" {
		r.Evidence = firstLine(output)
	} else {
		r.Evidence = "command exited 0"
	}
	return r
}

func firstLine(s string) string {
	if s == "" {
		return ""
	}
	return strings.Split(s, "\n")[0]
}
