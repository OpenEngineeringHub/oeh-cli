package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// configDir returns the OEH config directory (~/.oeh).
func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".oeh")
}

// configFile returns the full config file path.
func configFile() string {
	return filepath.Join(configDir(), "config.json")
}

// ensureConfigDir creates ~/.oeh if it doesn't exist.
func ensureConfigDir() error {
	return os.MkdirAll(configDir(), 0700)
}

// ─── login ────────────────────────────────────────────────────────────────────

var loginToken string

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with your OEH account",
	Long: `Authenticate the CLI with your Open Engineering Hub account.

Steps:
  1. Go to Settings → CLI & Workspace on the OEH platform
  2. Click "Generate & Copy" to create a token
  3. Run: oeh login --token <YOUR_TOKEN>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if loginToken == "" {
			fmt.Println()
			printHeader()
			fmt.Println()
			fmt.Println("  How to get your token:")
			fmt.Println("  1. Open the OEH platform")
			fmt.Println("  2. Go to Settings → CLI & Workspace")
			fmt.Println("  3. Click \"Generate & Copy\"")
			fmt.Println("  4. Run: oeh login --token <YOUR_TOKEN>")
			fmt.Println()
			return nil
		}

		cfg := loadConfig()
		cfg.Auth.Token = loginToken
		cfg.Auth.PlatformURL = "https://open-engineering-hub.dev"

		if err := saveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Println()
		printHeader()
		fmt.Println()
		printSuccess("Authenticated successfully!")
		fmt.Println()
		fmt.Println("  Token saved to " + configFile())
		fmt.Println()
		fmt.Println("  Next steps:")
		fmt.Println("  → oeh doctor          Check your environment")
		fmt.Println("  → oeh task start ID   Start a task")
		fmt.Println()
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := loadConfig()
		cfg.Auth.Token = ""
		_ = saveConfig(cfg)
		fmt.Println()
		printSuccess("Logged out. Token removed.")
		fmt.Println()
		return nil
	},
}

func init() {
	loginCmd.Flags().StringVar(&loginToken, "token", "", "Authentication token from the OEH platform")
}
