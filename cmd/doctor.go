package cmd

import (
	"fmt"
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
	Long:  `Detects installed tools, runtime versions, and OEH authentication status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println()
		printHeader()
		fmt.Printf("  Environment Doctor  ·  OEH CLI v%s\n", cliVersion)
		fmt.Println()

		checks := []envCheck{
			{"OS", checkOS, true},
			{"Git", checkTool("git", "--version"), true},
			{"Go", checkTool("go", "version"), false},
			{"Python", checkPython, false},
			{"Node.js", checkTool("node", "--version"), false},
			{"Docker", checkDocker, false},
			{"Ollama", checkOllama, false},
			{"OEH Auth", checkAuth, true},
		}

		allRequired := true
		start := time.Now()

		for _, c := range checks {
			version, ok := c.check()
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
			printBanner("ACTION REQUIRED — fix issues above")
		}

		fmt.Println()
		return nil
	},
}

// ─── Individual checks ────────────────────────────────────────────────────────

func checkOS() (string, bool) {
	return runtime.GOOS + "/" + runtime.GOARCH, true
}

func checkTool(binary, versionFlag string) func() (string, bool) {
	return func() (string, bool) {
		out, err := exec.Command(binary, versionFlag).Output()
		if err != nil {
			return "not found — install " + binary, false
		}
		line := strings.Split(strings.TrimSpace(string(out)), "\n")[0]
		// Trim verbose prefix: "git version 2.x" → "2.x"
		for _, prefix := range []string{"git version ", "go version ", "node "} {
			line = strings.TrimPrefix(line, prefix)
		}
		return line, true
	}
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

func checkDocker() (string, bool) {
	out, err := exec.Command("docker", "version", "--format", "{{.Client.Version}}").Output()
	if err != nil {
		return "not running or not installed", false
	}
	return "v" + strings.TrimSpace(string(out)), true
}

func checkOllama() (string, bool) {
	out, err := exec.Command("ollama", "--version").Output()
	if err != nil {
		return "not found — install from ollama.ai", false
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
