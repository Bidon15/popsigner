package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/Bidon15/popsigner/popctl/internal/api"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Version is set at build time.
	Version = "dev"

	// Global flags
	cfgFile     string
	apiKey      string
	apiURL      string
	namespaceID string
	jsonOut     bool
)

// rootCmd is the base command.
var rootCmd = &cobra.Command{
	Use:   "popctl",
	Short: "POPSigner CLI - manage keys via the control plane API",
	Long: `popctl is the command-line interface for POPSigner's control plane.

Use it to manage cryptographic keys, sign data, and organize namespaces
remotely using your API key.

Configuration (in order of priority):
  1. Command-line flags (--api-key, --api-url, --namespace)
  2. Environment variables (POPSIGNER_API_KEY, POPSIGNER_API_URL, POPSIGNER_NAMESPACE_ID)
  3. Config file (~/.popsigner.yaml)

Get started:
  $ popctl config init        # Interactive setup
  $ popctl keys list          # List your keys
  $ popctl keys create mykey  # Create a new key`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("popctl version %s\n", Version)
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ~/.popsigner.yaml)")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key (or POPSIGNER_API_KEY)")
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "", "API URL (or POPSIGNER_API_URL)")
	rootCmd.PersistentFlags().StringVar(&namespaceID, "namespace", "", "default namespace ID (or POPSIGNER_NAMESPACE_ID)")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "output in JSON format")

	// Add commands
	rootCmd.AddCommand(versionCmd)
}

// initConfig initializes viper configuration.
func initConfig() {
	// Set defaults
	viper.SetDefault("api_url", api.DefaultBaseURL)

	// Config file
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(home)
			viper.SetConfigType("yaml")
			viper.SetConfigName(".popsigner")
		}
	}

	// Environment variables
	viper.SetEnvPrefix("POPSIGNER")
	viper.AutomaticEnv()
	_ = viper.BindEnv("api_key", "POPSIGNER_API_KEY")
	_ = viper.BindEnv("api_url", "POPSIGNER_API_URL")
	_ = viper.BindEnv("namespace_id", "POPSIGNER_NAMESPACE_ID")

	// Read config file (ignore error if not found)
	_ = viper.ReadInConfig()
}

// getAPIKey returns the API key from flags, env, or config.
func getAPIKey() (string, error) {
	if apiKey != "" {
		return apiKey, nil
	}
	key := viper.GetString("api_key")
	if key == "" {
		return "", fmt.Errorf("API key required. Set via --api-key, POPSIGNER_API_KEY, or ~/.popsigner.yaml")
	}
	return key, nil
}

// getAPIURL returns the API URL from flags, env, or config.
func getAPIURL() string {
	if apiURL != "" {
		return apiURL
	}
	return viper.GetString("api_url")
}

// getNamespaceID returns the namespace ID from flags, env, or config.
func getNamespaceID(flagValue string) (uuid.UUID, error) {
	nsStr := flagValue
	if nsStr == "" {
		nsStr = namespaceID
	}
	if nsStr == "" {
		nsStr = viper.GetString("namespace_id")
	}
	if nsStr == "" {
		return uuid.Nil, fmt.Errorf("namespace ID required. Set via --namespace, POPSIGNER_NAMESPACE_ID, or ~/.popsigner.yaml")
	}
	return uuid.Parse(nsStr)
}

// getClient creates an API client from current configuration.
func getClient() (*api.Client, error) {
	key, err := getAPIKey()
	if err != nil {
		return nil, err
	}

	opts := []api.Option{}
	if url := getAPIURL(); url != "" && url != api.DefaultBaseURL {
		opts = append(opts, api.WithBaseURL(url))
	}

	return api.NewClient(key, opts...), nil
}

// configFilePath returns the default config file path.
func configFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".popsigner.yaml"
	}
	return filepath.Join(home, ".popsigner.yaml")
}

// Output helpers

// printJSON outputs data as formatted JSON.
func printJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// printError prints an error message.
func printError(err error) {
	if apiErr, ok := err.(*api.APIError); ok {
		fmt.Fprintf(os.Stderr, "%s %s\n", colorRed("Error:"), apiErr.Message)
		if apiErr.Code != "" {
			fmt.Fprintf(os.Stderr, "  Code: %s\n", apiErr.Code)
		}
	} else {
		fmt.Fprintf(os.Stderr, "%s %s\n", colorRed("Error:"), err.Error())
	}
}

// newTable creates a new tabwriter for formatted output.
func newTable() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}

// printTableHeader prints a bold header row.
func printTableHeader(w *tabwriter.Writer, columns ...string) {
	for i, col := range columns {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, colorBold(col))
	}
	fmt.Fprintln(w)
}

// Terminal colors

func colorRed(s string) string {
	if !isTTY() {
		return s
	}
	return "\033[31m" + s + "\033[0m"
}

func colorGreen(s string) string {
	if !isTTY() {
		return s
	}
	return "\033[32m" + s + "\033[0m"
}

func colorYellow(s string) string {
	if !isTTY() {
		return s
	}
	return "\033[33m" + s + "\033[0m"
}

func colorBold(s string) string {
	if !isTTY() {
		return s
	}
	return "\033[1m" + s + "\033[0m"
}

func isTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// formatTime formats a time for display.
func formatTime(t interface{}) string {
	switch v := t.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", t)
	}
}

