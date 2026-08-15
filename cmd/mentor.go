package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var mentorCmd = &cobra.Command{
	Use:   "mentor",
	Short: "Ask AI Mentor for hints and diagnostic help on current task",
	Long: `Analyze local task progress, test verification results, and git diff to provide context-aware debugging guidance.

Example:
  oeh mentor
  oeh mentor --prompt "Why is TTFT check failing?"`,
	RunE: runMentor,
}

var mentorPromptFlag string

func init() {
	mentorCmd.Flags().StringVarP(&mentorPromptFlag, "prompt", "p", "", "Specific question or error message to ask the mentor")
}

type MentorContextPayload struct {
	TaskID      string           `json:"task_id"`
	TaskTitle   string           `json:"task_title"`
	Objective   string           `json:"objective"`
	TestResults []EvidenceResult `json:"test_results"`
	GitDiff     string           `json:"git_diff,omitempty"`
	UserPrompt  string           `json:"user_prompt,omitempty"`
	CLIVersion  string           `json:"cli_version"`
}

func runMentor(cmd *cobra.Command, args []string) error {
	state, spec, err := loadWorkspace()
	if err != nil {
		return fmt.Errorf("no task active — run: oeh task start <TASK-ID>")
	}

	fmt.Println()
	printHeader()
	fmt.Printf("  AI Engineering Mentor  ·  Task: %s\n", state.TaskID)
	fmt.Println()

	// 1. Gather test results
	printStep("Collecting local verification context...")
	results := make([]EvidenceResult, 0, len(spec.Steps))
	for _, step := range spec.Steps {
		r := runStep(step)
		results = append(results, EvidenceResult{
			StepID:     r.StepID,
			Passed:     r.Passed,
			Evidence:   r.Evidence,
			Error:      r.Error,
			DurationMs: r.DurationMs,
		})
	}

	// 2. Gather git diff snippet
	printStep("Gathering workspace changes...")
	gitDiff := getGitDiffSnippet()

	payload := MentorContextPayload{
		TaskID:      state.TaskID,
		TaskTitle:   spec.Title,
		Objective:   spec.Objective,
		TestResults: results,
		GitDiff:     gitDiff,
		UserPrompt:  mentorPromptFlag,
		CLIVersion:  cliVersion,
	}

	// 3. Try local Ollama first, fallback to OEH Engine
	printStep("Consulting AI Mentor...")
	fmt.Println()

	advice, err := queryLocalOllama(payload)
	if err != nil {
		advice, err = queryEngineMentor(payload)
		if err != nil {
			advice = generateStaticDiagnosticHint(spec, results)
		}
	}

	printBanner("MENTOR GUIDANCE")
	fmt.Println()
	fmt.Println(advice)
	fmt.Println()

	return nil
}

func getGitDiffSnippet() string {
	out, err := exec.Command("git", "diff", "HEAD", "--stat").Output()
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(out))
	if len(s) > 1000 {
		return s[:1000] + "\n...[truncated]"
	}
	return s
}

func queryLocalOllama(payload MentorContextPayload) (string, error) {
	promptText := fmt.Sprintf(`Task: %s - %s
Objective: %s
Failing tests: %v
User question: %s

Provide 2-3 concise diagnostic hints without giving away the full code solution. Focus on engineering principles.`,
		payload.TaskID, payload.TaskTitle, payload.Objective, getFailingTests(payload.TestResults), payload.UserPrompt)

	reqBody := map[string]interface{}{
		"model":  "llama3.2:3b",
		"prompt": promptText,
		"stream": false,
	}
	data, _ := json.Marshal(reqBody)

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Post("http://localhost:11434/api/generate", "application/json", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.Response == "" {
		return "", fmt.Errorf("invalid response from ollama")
	}
	return strings.TrimSpace(result.Response), nil
}

func queryEngineMentor(payload MentorContextPayload) (string, error) {
	url := getPlatformURL() + "/api/mentor/help"
	data, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Guidance string `json:"guidance"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.Guidance == "" {
		return "", fmt.Errorf("invalid response from engine mentor")
	}
	return result.Guidance, nil
}

func getFailingTests(results []EvidenceResult) []string {
	failing := []string{}
	for _, r := range results {
		if !r.Passed {
			failing = append(failing, fmt.Sprintf("%s: %s", r.StepID, r.Error))
		}
	}
	return failing
}

func generateStaticDiagnosticHint(spec *TaskSpec, results []EvidenceResult) string {
	failing := getFailingTests(results)
	if len(failing) == 0 {
		return "  🎉 All local checks are passing! Run 'oeh submit' to submit your work for authoritative scoring."
	}

	var sb strings.Builder
	sb.WriteString("  💡 Diagnostic Analysis:\n")
	for _, f := range failing {
		sb.WriteString(fmt.Sprintf("  · Issue: %s\n", f))
	}
	sb.WriteString("\n  Next Steps:\n")
	sb.WriteString("  1. Verify the process or endpoint service is active on localhost.\n")
	sb.WriteString("  2. Check logs for non-200 HTTP response codes or missing headers.\n")
	sb.WriteString("  3. Run 'oeh verify' after fixing issues.")
	return sb.String()
}
