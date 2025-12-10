package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Bidon15/banhbaoring"
	"github.com/spf13/cobra"
)

// Version information - set via ldflags during build
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// Global flag variables
var (
	baoAddr   string
	baoToken  string
	storePath string
	verbose   bool
)

// Environment variable names
const (
	EnvBaoAddr      = "BAO_ADDR"
	EnvBaoToken     = "BAO_TOKEN"
	EnvBaoStorePath = "BAO_STORE_PATH"
)

// Default values
const (
	DefaultStorePathSuffix = ".baokey/keyring.json"
)

// rootCmd is the base command for the CLI
var rootCmd *cobra.Command

// versionCmd prints version information
var versionCmd *cobra.Command

func init() {
	rootCmd = &cobra.Command{
		Use:   "baokey",
		Short: "BaoKey - OpenBao keyring management for Celestia",
		Long: `BaoKey provides secure key management using OpenBao Transit engine.

Keys are stored in OpenBao and never leave the secure boundary.
Only signatures are returned to the client.

Configuration can be provided via flags or environment variables:
  --bao-addr    or BAO_ADDR      OpenBao server address (required)
  --bao-token   or BAO_TOKEN     OpenBao authentication token (required)
  --store-path  or BAO_STORE_PATH Local metadata store path`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Print the version, commit hash, and build date of baokey",
		Run: func(cmd *cobra.Command, args []string) {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "baokey %s\n", Version)
			if verbose {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  commit:  %s\n", Commit)
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  built:   %s\n", BuildDate)
			}
		},
	}

	// Add persistent flags for all commands
	rootCmd.PersistentFlags().StringVar(&baoAddr, "bao-addr", "", "OpenBao server address (or BAO_ADDR env)")
	rootCmd.PersistentFlags().StringVar(&baoToken, "bao-token", "", "OpenBao authentication token (or BAO_TOKEN env)")
	rootCmd.PersistentFlags().StringVar(&storePath, "store-path", "", "Local metadata store path (or BAO_STORE_PATH env)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// ExecuteWithArgs runs the root command with the provided arguments (for testing)
func ExecuteWithArgs(args []string) error {
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

// SetOutput sets the output writer for the root command (for testing)
func SetOutput(w io.Writer) {
	rootCmd.SetOut(w)
	rootCmd.SetErr(w)
}

// ResetFlags resets all global flags to their defaults (for testing)
func ResetFlags() {
	baoAddr = ""
	baoToken = ""
	storePath = ""
	verbose = false
}

// CLIConfig holds resolved configuration from flags and environment
type CLIConfig struct {
	BaoAddr   string
	BaoToken  string
	StorePath string
	Verbose   bool
}

// GetConfig resolves configuration from flags and environment variables.
// Flags take precedence over environment variables.
func GetConfig() (*CLIConfig, error) {
	cfg := &CLIConfig{
		BaoAddr:   baoAddr,
		BaoToken:  baoToken,
		StorePath: storePath,
		Verbose:   verbose,
	}

	// Resolve from environment if not set via flags
	if cfg.BaoAddr == "" {
		cfg.BaoAddr = os.Getenv(EnvBaoAddr)
	}
	if cfg.BaoToken == "" {
		cfg.BaoToken = os.Getenv(EnvBaoToken)
	}
	if cfg.StorePath == "" {
		cfg.StorePath = os.Getenv(EnvBaoStorePath)
	}

	// Apply default store path
	if cfg.StorePath == "" {
		cfg.StorePath = DefaultStorePath()
	}

	return cfg, nil
}

// ValidateConfig checks that all required configuration is present
func (c *CLIConfig) Validate() error {
	if c.BaoAddr == "" {
		return fmt.Errorf("BAO_ADDR is required (use --bao-addr flag or BAO_ADDR environment variable)")
	}
	if c.BaoToken == "" {
		return fmt.Errorf("BAO_TOKEN is required (use --bao-token flag or BAO_TOKEN environment variable)")
	}
	return nil
}

// ToBanhbaoConfig converts CLIConfig to banhbaoring.Config
func (c *CLIConfig) ToBanhbaoConfig() banhbaoring.Config {
	return banhbaoring.Config{
		BaoAddr:   c.BaoAddr,
		BaoToken:  c.BaoToken,
		StorePath: c.StorePath,
	}
}

// DefaultStorePath returns the default store path (~/.baokey/keyring.json)
func DefaultStorePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return "./keyring.json"
	}
	return filepath.Join(home, DefaultStorePathSuffix)
}

// GetKeyring creates and returns a configured BaoKeyring.
// This is a helper for command implementations.
func GetKeyring() (*banhbaoring.BaoKeyring, error) {
	cfg, err := GetConfig()
	if err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return banhbaoring.New(context.Background(), cfg.ToBanhbaoConfig())
}

// VerbosePrintf prints to stdout if verbose mode is enabled
func VerbosePrintf(format string, args ...interface{}) {
	if verbose {
		fmt.Printf(format, args...)
	}
}

// VerbosePrintln prints to stdout if verbose mode is enabled
func VerbosePrintln(args ...interface{}) {
	if verbose {
		fmt.Println(args...)
	}
}

// EnsureStoreDir ensures the store directory exists
func EnsureStoreDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0700)
}

// FormatAddress truncates an address for display if it's too long
func FormatAddress(addr string, maxLen int) string {
	if len(addr) <= maxLen {
		return addr
	}
	if maxLen < 10 {
		return addr[:maxLen]
	}
	// Show first and last parts with ellipsis
	half := (maxLen - 3) / 2
	return addr[:half] + "..." + addr[len(addr)-half:]
}

