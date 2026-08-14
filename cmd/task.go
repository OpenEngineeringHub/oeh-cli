package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// TaskSpec is the structure fetched from the OEH platform for a given task ID.
type TaskSpec struct {
	TaskID    string                `json:"task_id"`
	Version   int                   `json:"version"`
	Type      string                `json:"type"`
	Title     string                `json:"title"`
	Course    string                `json:"course"`
	Chapter   int                   `json:"chapter"`
	Objective string                `json:"objective"`
	Steps     []VerificationStep    `json:"verification_steps"`
	Env       TaskEnv               `json:"environment"`
}

// TaskEnv lists environment requirements.
type TaskEnv struct {
	OS    string   `json:"os,omitempty"`
	Tools []string `json:"tools,omitempty"`
}

// OehState is the .oeh/state.json file.
type OehState struct {
	TaskID      string    `json:"task_id"`
	Version     int       `json:"task_version"`
	WorkspaceID string    `json:"workspace_id"`
	StartedAt   time.Time `json:"started_at"`
	UserID      string    `json:"user_id,omitempty"`
}

// ─── task command ─────────────────────────────────────────────────────────────

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Task management commands",
}

var taskStartCmd = &cobra.Command{
	Use:   "start <TASK-ID>",
	Short: "Initialize workspace for a task",
	Long: `Fetch task specification from OEH and set up your local workspace.

Example:
  oeh task start ie-ch03-lab-001`,
	Args: cobra.ExactArgs(1),
	RunE: runTaskStart,
}

var taskStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current task status and verification progress",
	RunE:  runTaskStatus,
}

func init() {
	taskCmd.AddCommand(taskStartCmd)
	taskCmd.AddCommand(taskStatusCmd)
}

// ─── task start ───────────────────────────────────────────────────────────────

func runTaskStart(cmd *cobra.Command, args []string) error {
	taskID := args[0]

	fmt.Println()
	printHeader()
	fmt.Printf("  Starting task: %s\n\n", taskID)

	// 1. Fetch task spec from platform
	printStep("Fetching task specification...")
	spec, err := fetchTaskSpec(taskID)
	if err != nil {
		// Demo mode: use a mock spec so users can still try the CLI
		printWarn("Could not reach OEH platform — using demo mode")
		spec = mockTaskSpec(taskID)
	} else {
		printSuccess("Task spec loaded: " + spec.Title)
	}

	// 2. Create .oeh directory
	printStep("Initializing workspace...")
	oehDir := ".oeh"
	if err := os.MkdirAll(filepath.Join(oehDir, "evidence"), 0750); err != nil {
		return fmt.Errorf("could not create .oeh directory: %w", err)
	}

	// 3. Write state file
	state := OehState{
		TaskID:      taskID,
		Version:     spec.Version,
		WorkspaceID: genWorkspaceID(),
		StartedAt:   time.Now(),
	}
	stateData, _ := json.MarshalIndent(state, "", "  ")
	if err := os.WriteFile(filepath.Join(oehDir, "state.json"), stateData, 0600); err != nil {
		return fmt.Errorf("could not write state: %w", err)
	}

	// 4. Write task.json
	specData, _ := json.MarshalIndent(spec, "", "  ")
	if err := os.WriteFile(filepath.Join(oehDir, "task.json"), specData, 0600); err != nil {
		return fmt.Errorf("could not write task spec: %w", err)
	}

	printSuccess("Workspace initialized → .oeh/")

	// 5. Show task info
	fmt.Println()
	fmt.Printf("  Task      %s\n", spec.Title)
	fmt.Printf("  ID        %s\n", taskID)
	fmt.Printf("  Type      %s\n", spec.Type)
	fmt.Printf("  Steps     %d verification steps\n", len(spec.Steps))
	fmt.Println()

	// 6. Show environment requirements
	if len(spec.Env.Tools) > 0 {
		fmt.Println("  Required tools:")
		for _, t := range spec.Env.Tools {
			fmt.Printf("    · %s\n", t)
		}
		fmt.Println()
	}

	fmt.Println("  Run oeh doctor to verify your environment.")
	fmt.Println("  When ready: oeh verify")
	fmt.Println()

	return nil
}

// ─── task status ──────────────────────────────────────────────────────────────

func runTaskStatus(cmd *cobra.Command, args []string) error {
	state, spec, err := loadWorkspace()
	if err != nil {
		return fmt.Errorf("no task found in current directory — run: oeh task start <TASK-ID>")
	}

	fmt.Println()
	printHeader()
	fmt.Printf("  Task Status\n\n")
	fmt.Printf("  %-12s %s\n", "Task ID:", state.TaskID)
	fmt.Printf("  %-12s %s\n", "Title:", spec.Title)
	fmt.Printf("  %-12s %s\n", "Started:", state.StartedAt.Format("2006-01-02 15:04"))
	fmt.Println()

	if len(spec.Steps) > 0 {
		fmt.Println("  Verification steps:")
		for i, step := range spec.Steps {
			fmt.Printf("    %d. %-30s ○\n", i+1, step.Label)
		}
	}

	fmt.Println()
	fmt.Println("  Run oeh verify to check your progress.")
	fmt.Println()
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func loadWorkspace() (*OehState, *TaskSpec, error) {
	stateData, err := os.ReadFile(filepath.Join(".oeh", "state.json"))
	if err != nil {
		return nil, nil, err
	}
	var state OehState
	if err := json.Unmarshal(stateData, &state); err != nil {
		return nil, nil, err
	}

	specData, err := os.ReadFile(filepath.Join(".oeh", "task.json"))
	if err != nil {
		return nil, nil, err
	}
	var spec TaskSpec
	if err := json.Unmarshal(specData, &spec); err != nil {
		return nil, nil, err
	}
	return &state, &spec, nil
}

func genWorkspaceID() string {
	return fmt.Sprintf("ws_%d", time.Now().UnixNano()%1_000_000_000)
}

// mockTaskSpec returns a demo spec for offline usage.
func mockTaskSpec(taskID string) *TaskSpec {
	return &TaskSpec{
		TaskID:    taskID,
		Version:   1,
		Type:      "lab",
		Title:     "Demo Task (" + taskID + ")",
		Course:    "ie",
		Chapter:   3,
		Objective: "Demo mode — connect to OEH platform for full spec",
		Env:       TaskEnv{Tools: []string{"ollama", "git"}},
		Steps: []VerificationStep{
			{ID: "step-1", Label: "Ollama installed", Type: "process", ProcessName: "ollama"},
			{ID: "step-2", Label: "API responding", Type: "http", URL: "http://localhost:11434/api/tags", ExpectedStatus: 200},
		},
	}
}
