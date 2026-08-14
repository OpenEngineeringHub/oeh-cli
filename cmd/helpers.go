// Package cmd — shared helpers for pretty printing and config.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
)

// ─── Simple config (replaces viper to avoid network issues) ───────────────────

type cliConfig struct {
	Auth struct {
		Token       string `json:"token"`
		PlatformURL string `json:"platform_url"`
	} `json:"auth"`
}

func loadConfig() cliConfig {
	var cfg cliConfig
	data, err := os.ReadFile(configFile())
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(data, &cfg)
	return cfg
}

func saveConfig(cfg cliConfig) error {
	if err := ensureConfigDir(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile(), data, 0600)
}

func getToken() string {
	return loadConfig().Auth.Token
}

func getPlatformURL() string {
	url := loadConfig().Auth.PlatformURL
	if url == "" {
		return "https://open-engineering-hub.dev"
	}
	return url
}

// ─── Print helpers ─────────────────────────────────────────────────────────────

var (
	bold    = color.New(color.Bold)
	green   = color.New(color.FgGreen)
	red     = color.New(color.FgRed)
	yellow  = color.New(color.FgYellow)
	cyan    = color.New(color.FgCyan)
	dimGray = color.New(color.FgHiBlack)
)

func printHeader() {
	bold.Println("  ⚡ Open Engineering Hub")
}

func printSuccess(msg string) {
	green.Printf("  ✓ %s\n", msg)
}

func printWarn(msg string) {
	yellow.Printf("  ⚠ %s\n", msg)
}

func printStep(msg string) {
	dimGray.Printf("  → %s\n", msg)
}

func printCheckPass(label, detail string) {
	green.Printf("  ✓ %-32s", label)
	if detail != "" {
		dimGray.Printf("  %s", detail)
	}
	fmt.Println()
}

func printCheckFail(label, detail string) {
	red.Printf("  ✗ %-32s", label)
	if detail != "" {
		dimGray.Printf("  %s", detail)
	}
	fmt.Println()
}

func printCheckWarn(label, detail string) {
	yellow.Printf("  ○ %-32s", label)
	if detail != "" {
		dimGray.Printf("  %s", detail)
	}
	fmt.Println()
}

func printBanner(msg string) {
	fmt.Println()
	cyan.Printf("  ╔══════════════════════════════════╗\n")
	cyan.Printf("  ║  %-32s  ║\n", msg)
	cyan.Printf("  ╚══════════════════════════════════╝\n")
}
