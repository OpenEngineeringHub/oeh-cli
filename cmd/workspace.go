package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage local Docker container workspace",
	Long: `Orchestrates local Linux DevContainers / Docker sandbox environments for OEH tasks.`,
}

var workspaceStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start containerized environment for current task",
	RunE:  runWorkspaceStart,
}

var workspaceShellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Open an interactive terminal shell inside the workspace container",
	RunE:  runWorkspaceShell,
}

var workspaceStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the workspace container",
	RunE:  runWorkspaceStop,
}

func init() {
	workspaceCmd.AddCommand(workspaceStartCmd)
	workspaceCmd.AddCommand(workspaceShellCmd)
	workspaceCmd.AddCommand(workspaceStopCmd)
}

func runWorkspaceStart(cmd *cobra.Command, args []string) error {
	state, spec, err := loadWorkspace()
	if err != nil {
		return fmt.Errorf("no task active — run: oeh task start <TASK-ID>")
	}

	fmt.Println()
	printHeader()
	fmt.Printf("  Starting Workspace for task: %s (%s)\n\n", state.TaskID, spec.Title)

	// Check if Docker Engine is running
	if _, ok := checkDockerEngine(); !ok {
		printWarn("Docker Engine is not running on host.")
		fmt.Println("  Please start Docker Desktop or work in GitHub Codespaces.")
		fmt.Println()
		return nil
	}

	// Check if docker-compose or devcontainer exists
	if _, err := os.Stat("docker-compose.yml"); err == nil {
		printStep("Starting docker-compose services...")
		c := exec.Command("docker", "compose", "up", "-d")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("failed to start docker compose: %w", err)
		}
		printSuccess("Services started via docker-compose!")
		return nil
	}

	containerName := "oeh-ws-" + state.TaskID
	printStep("Preparing container: " + containerName)

	// Spin up lightweight python/node/dev container image
	runArgs := []string{
		"run", "-d",
		"--name", containerName,
		"-v", fmt.Sprintf("%s:/workspace", getCWD()),
		"-w", "/workspace",
		"mcr.microsoft.com/devcontainers/python:3.12-bullseye",
		"sleep", "infinity",
	}

	// Check if container already exists
	exec.Command("docker", "rm", "-f", containerName).Run()

	out, err := exec.Command("docker", runArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start container: %s", string(out))
	}

	printSuccess("Container running: " + containerName)
	fmt.Println()
	fmt.Println("  To attach interactive shell run:")
	fmt.Println("  oeh workspace shell")
	fmt.Println()

	return nil
}

func runWorkspaceShell(cmd *cobra.Command, args []string) error {
	state, _, err := loadWorkspace()
	if err != nil {
		return fmt.Errorf("no task active — run: oeh task start <TASK-ID>")
	}

	containerName := "oeh-ws-" + state.TaskID

	c := exec.Command("docker", "exec", "-it", containerName, "bash")
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		// Fallback to sh
		c2 := exec.Command("docker", "exec", "-it", containerName, "sh")
		c2.Stdin = os.Stdin
		c2.Stdout = os.Stdout
		c2.Stderr = os.Stderr
		if err2 := c2.Run(); err2 != nil {
			return fmt.Errorf("could not connect to workspace container '%s' — run: oeh workspace start", containerName)
		}
	}

	return nil
}

func runWorkspaceStop(cmd *cobra.Command, args []string) error {
	state, _, err := loadWorkspace()
	if err != nil {
		return fmt.Errorf("no task active — run: oeh task start <TASK-ID>")
	}

	containerName := "oeh-ws-" + state.TaskID
	printStep("Stopping workspace container: " + containerName)
	exec.Command("docker", "rm", "-f", containerName).Run()
	printSuccess("Workspace container stopped.")
	return nil
}

func getCWD() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return filepath.ToSlash(dir)
}
