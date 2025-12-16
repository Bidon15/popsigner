package cmd

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
	Long:  `Commands for managing the popctl configuration file.`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Long:  `Create a new configuration file at ~/.popsigner.yaml with interactive prompts.`,
	RunE:  runConfigInit,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE:  runConfigShow,
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	configPath := configFilePath()

	// Check if file exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("%s Config file already exists at %s\n", colorYellow("⚠"), configPath)
		fmt.Print("Overwrite? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Aborted")
			return nil
		}
	}

	// Prompt for values
	fmt.Print("API Key (psk_...): ")
	var apiKeyInput string
	fmt.Scanln(&apiKeyInput)

	fmt.Print("API URL (press Enter for default): ")
	var apiURLInput string
	fmt.Scanln(&apiURLInput)
	if apiURLInput == "" {
		apiURLInput = "https://api.popsigner.com"
	}

	fmt.Print("Default Namespace ID (optional, press Enter to skip): ")
	var nsIDInput string
	fmt.Scanln(&nsIDInput)

	// Validate namespace ID if provided
	if nsIDInput != "" {
		if _, err := uuid.Parse(nsIDInput); err != nil {
			return fmt.Errorf("invalid namespace ID: %w", err)
		}
	}

	// Write config file
	configContent := fmt.Sprintf(`# POPSigner CLI Configuration
api_key: %s
api_url: %s
`, apiKeyInput, apiURLInput)

	if nsIDInput != "" {
		configContent += fmt.Sprintf("namespace_id: %s\n", nsIDInput)
	}

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("%s Config file created at %s\n", colorGreen("✓"), configPath)
	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	key := viper.GetString("api_key")
	if apiKey != "" {
		key = apiKey
	}

	url := getAPIURL()
	nsID := viper.GetString("namespace_id")
	if namespaceID != "" {
		nsID = namespaceID
	}

	if jsonOut {
		return printJSON(map[string]interface{}{
			"api_key":      maskAPIKey(key),
			"api_url":      url,
			"namespace_id": nsID,
			"config_file":  viper.ConfigFileUsed(),
		})
	}

	fmt.Printf("API Key:      %s\n", maskAPIKey(key))
	fmt.Printf("API URL:      %s\n", url)
	if nsID != "" {
		fmt.Printf("Namespace ID: %s\n", nsID)
	} else {
		fmt.Printf("Namespace ID: %s\n", colorYellow("(not set)"))
	}

	if configFile := viper.ConfigFileUsed(); configFile != "" {
		fmt.Printf("Config File:  %s\n", configFile)
	}

	return nil
}

// maskAPIKey masks the API key for display.
func maskAPIKey(key string) string {
	if key == "" {
		return colorYellow("(not set)")
	}
	if len(key) <= 12 {
		return "****"
	}
	return key[:8] + "..." + key[len(key)-4:]
}

