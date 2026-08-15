package cmd

import (
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type envCheck struct {
	name    string
	check   func() (string, bool)
	require bool // if true, failing this marks environment NOT READY
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check your engineering environment",
	Long: `Detects installed tools, runtime versions, Docker engine status, internet connectivity, and OEH authentication status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println()
		printHeader()
		fmt.Printf("  Open Engineering Hub Doctor  ·  OEH CLI v%s\n", cliVersion)
		fmt.Printf("  Host OS: %s  ·  Arch: %s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Println()

		checks := []envCheck{
			{"OEH CLI", checkCLI, true},
			{"Internet", checkInternet, true},
			{"Git", checkTool("git", "--version"), true},
			{"Docker Client", checkTool("docker", "version"), false},
			{"Docker Engine", checkDockerEngine, false},
			{"GitHub CLI", checkTool("gh", "--version"), false},
			{"Python", checkPython, false},
			{"Node.js", checkTool("node", "--version"), false},
			{"Go Runtime", checkTool("go", "version"), false},
			{"Ollama", checkOllama, false},
			{"OEH Auth", checkAuth, true},
		}

		allRequired := true
		dockerAvailable := false
		start := time.Now()

		for _, c := range checks {
			version, ok := c.check()
			if c.name == "Docker Engine" && ok {
				dockerAvailable = true
			}
			if ok {
				printCheckPass(c.name, version)
			} else {
				printCheckFail(c.name, version)
				if c.require {
					allRequired = false
				}
			}
		}

		fmt.Printf("\n  Checked in %dms\n\n", time.Since(start).Milliseconds())

		if allRequired {
			printBanner("ENVIRONMENT READY")
		} else {
			printBanner("ACTION REQUIRED — fix required issues above")
		}

		if !dockerAvailable {
			fmt.Println()
			printWarn("Docker is not running locally.")
			fmt.Println("  You have two choices for running OEH labs & projects:")
			fmt.Println("    1. Install Docker Desktop (local containers): https://www.docker.com/products/docker-desktop/")
			fmt.Println("    2. Use Cloud Workspace (GitHub Codespaces): Click 'Open in Codespace' on the OEH platform")
			fmt.Println()
		}

		fmt.Println()
		return nil
	},
}

// ─── Individual checks ────────────────────────────────────────────────────────

func checkCLI() (string, bool) {
	return "v" + cliVersion, true
}

func checkInternet() (string, bool) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("https://1.1.1.1")
	if err != nil {
		return "offline — check internet connection", false
	}
	defer resp.Body.Close()
	return "connected", true
}

func checkTool(binary, versionFlag string) func() (string, bool) {
	return func() (string, bool) {
		out, err := exec.Command(binary, versionFlag).Output()
		if err != nil {
			return "not found", false
		}
		line := strings.Split(strings.TrimSpace(string(out)), "\n")[0]
		for _, prefix := range []string{"git version ", "go version ", "node ", "gh version "} {
			line = strings.TrimPrefix(line, prefix)
		}
		return line, true
	}
}

func checkDockerEngine() (string, bool) {
	out, err := exec.Command("docker", "info", "--format", "{{.ServerVersion}}").Output()
	if err != nil {
		return "engine not running — start Docker Desktop", false
	}
	return "v" + strings.TrimSpace(string(out)), true
}

func checkPython() (string, bool) {
	for _, bin := range []string{"python3", "python"} {
		out, err := exec.Command(bin, "--version").Output()
		if err == nil {
			return strings.TrimSpace(string(out)), true
		}
	}
	return "not found", false
}

func checkOllama() (string, bool) {
	out, err := exec.Command("ollama", "--version").Output()
	if err != nil {
		return "not found (optional for LLM labs)", false
	}
	return strings.TrimSpace(string(out)), true
}

func checkAuth() (string, bool) {
	token := getToken()
	if token == "" {
		return "not authenticated — run: oeh login --token <TOKEN>", false
	}
	return "authenticated", true
}
