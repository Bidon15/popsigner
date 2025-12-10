package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Bidon15/banhbaoring"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	baoAddr   string
	baoToken  string
	storePath string
	jsonOut   bool
)

var rootCmd = &cobra.Command{
	Use:   "banhbao",
	Short: "BanhBao - OpenBao keyring management for Celestia",
	Long: `BanhBao provides secure key management using OpenBao Transit engine.

Keys are stored in OpenBao and never leave the secure boundary.
Only signatures are returned to the client.

Environment variables:
  BAO_ADDR   - OpenBao server address (e.g., http://127.0.0.1:8200)
  BAO_TOKEN  - OpenBao authentication token`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("banhbao v0.1.0")
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&baoAddr, "bao-addr", "", "OpenBao address (or BAO_ADDR env)")
	rootCmd.PersistentFlags().StringVar(&baoToken, "bao-token", "", "OpenBao token (or BAO_TOKEN env)")
	rootCmd.PersistentFlags().StringVar(&storePath, "store-path", "./keyring-metadata.json", "Local metadata store path")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output in JSON format")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(keysCmd)
	rootCmd.AddCommand(migrateCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// getKeyring creates a BaoKeyring from CLI flags or environment variables.
func getKeyring() (*banhbaoring.BaoKeyring, error) {
	addr := baoAddr
	if addr == "" {
		addr = os.Getenv("BAO_ADDR")
	}

	token := baoToken
	if token == "" {
		token = os.Getenv("BAO_TOKEN")
	}

	if addr == "" || token == "" {
		return nil, fmt.Errorf("BAO_ADDR and BAO_TOKEN are required (set via flags or environment)")
	}

	cfg := banhbaoring.Config{
		BaoAddr:   addr,
		BaoToken:  token,
		StorePath: storePath,
	}

	return banhbaoring.New(context.Background(), cfg)
}

// colorRed returns text in red color for terminal output.
func colorRed(s string) string {
	return "\033[31m" + s + "\033[0m"
}

// colorGreen returns text in green color for terminal output.
func colorGreen(s string) string {
	return "\033[32m" + s + "\033[0m"
}

// colorYellow returns text in yellow color for terminal output.
func colorYellow(s string) string {
	return "\033[33m" + s + "\033[0m"
}

// colorBold returns text in bold for terminal output.
func colorBold(s string) string {
	return "\033[1m" + s + "\033[0m"
}
