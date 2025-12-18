package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Bidon15/popsigner/popctl/internal/api"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Manage chain deployments",
	Long: `Create and manage OP Stack or Nitro chain deployments using POPSigner.

Commands:
  create    Create a new deployment from a YAML config
  status    Get deployment status
  list      List all deployments
  artifacts Download deployment artifacts
  resume    Resume a paused deployment

Examples:
  popctl bootstrap create --config my-chain.yaml
  popctl bootstrap status abc123-def456
  popctl bootstrap artifacts abc123-def456 --output ./artifacts/`,
}

var bootstrapCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new chain deployment",
	Long: `Create a new chain deployment from a YAML configuration file.

The config file must specify:
  - chain_id: The unique chain ID for your L2/L3
  - stack: Either "opstack" or "nitro"
  - Additional stack-specific configuration

Example config (opstack):
  chain_id: 42069
  stack: opstack
  l1_chain_id: 11155111
  l1_rpc_url: https://sepolia.infura.io/v3/YOUR_KEY

Example:
  popctl bootstrap create --config my-chain.yaml
  popctl bootstrap create --config my-chain.yaml --start`,
	RunE: runBootstrapCreate,
}

var bootstrapStatusCmd = &cobra.Command{
	Use:   "status <deployment-id>",
	Short: "Get deployment status",
	Long: `Get the current status of a deployment.

Use --watch to continuously poll for updates.

Examples:
  popctl bootstrap status abc123-def456
  popctl bootstrap status abc123-def456 --watch`,
	Args: cobra.ExactArgs(1),
	RunE: runBootstrapStatus,
}

var bootstrapListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all deployments",
	Long: `List all chain deployments.

Use --status to filter by status (pending, running, completed, failed, paused).

Examples:
  popctl bootstrap list
  popctl bootstrap list --status running`,
	RunE: runBootstrapList,
}

var bootstrapArtifactsCmd = &cobra.Command{
	Use:   "artifacts <deployment-id>",
	Short: "Download deployment artifacts",
	Long: `Download deployment artifacts (genesis.json, rollup.json, etc.) to a directory.

For OP Stack deployments: genesis.json, rollup.json, state.json
For Nitro deployments: chain-info.json, node-config.json, core-contracts.json

Examples:
  popctl bootstrap artifacts abc123-def456
  popctl bootstrap artifacts abc123-def456 --output ./my-chain/`,
	Args: cobra.ExactArgs(1),
	RunE: runBootstrapArtifacts,
}

var bootstrapResumeCmd = &cobra.Command{
	Use:   "resume <deployment-id>",
	Short: "Resume a paused or pending deployment",
	Long: `Resume a deployment that is in 'pending' or 'paused' status.

This will start or restart the deployment process.

Examples:
  popctl bootstrap resume abc123-def456`,
	Args: cobra.ExactArgs(1),
	RunE: runBootstrapResume,
}

func init() {
	// Create flags
	bootstrapCreateCmd.Flags().StringP("config", "c", "", "Path to config YAML file (required)")
	bootstrapCreateCmd.Flags().Bool("start", false, "Start deployment immediately after creation")
	_ = bootstrapCreateCmd.MarkFlagRequired("config")

	// Status flags
	bootstrapStatusCmd.Flags().BoolP("watch", "w", false, "Watch for status updates (polls every 2s)")

	// List flags
	bootstrapListCmd.Flags().String("status", "", "Filter by status (pending, running, completed, failed, paused)")

	// Artifacts flags
	bootstrapArtifactsCmd.Flags().StringP("output", "o", ".", "Output directory for artifacts")

	// Add subcommands
	bootstrapCmd.AddCommand(bootstrapCreateCmd)
	bootstrapCmd.AddCommand(bootstrapStatusCmd)
	bootstrapCmd.AddCommand(bootstrapListCmd)
	bootstrapCmd.AddCommand(bootstrapArtifactsCmd)
	bootstrapCmd.AddCommand(bootstrapResumeCmd)

	rootCmd.AddCommand(bootstrapCmd)
}

func runBootstrapCreate(cmd *cobra.Command, args []string) error {
	configFile, _ := cmd.Flags().GetString("config")
	startNow, _ := cmd.Flags().GetBool("start")

	// Read config file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("invalid YAML config: %w", err)
	}

	// Extract required fields
	chainIDRaw, ok := config["chain_id"]
	if !ok {
		return fmt.Errorf("config missing required field: chain_id")
	}
	chainID, ok := toInt64(chainIDRaw)
	if !ok {
		return fmt.Errorf("chain_id must be a number")
	}

	stack, ok := config["stack"].(string)
	if !ok {
		return fmt.Errorf("config missing required field: stack (must be 'opstack' or 'nitro')")
	}

	if stack != "opstack" && stack != "nitro" {
		return fmt.Errorf("stack must be 'opstack' or 'nitro', got '%s'", stack)
	}

	client, err := getClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	deployment, err := client.CreateDeployment(ctx, api.CreateDeploymentRequest{
		ChainID: chainID,
		Stack:   stack,
		Config:  config,
	})
	if err != nil {
		printError(err)
		return err
	}

	if jsonOut {
		return printJSON(deployment)
	}

	fmt.Printf("%s Deployment created\n", colorGreen("✓"))
	fmt.Printf("  ID:       %s\n", deployment.ID)
	fmt.Printf("  Chain ID: %d\n", deployment.ChainID)
	fmt.Printf("  Stack:    %s\n", deployment.Stack)
	fmt.Printf("  Status:   %s\n", statusEmoji(deployment.Status))

	if startNow {
		fmt.Printf("\n%s Starting deployment...\n", colorYellow("→"))
		if err := client.StartDeployment(ctx, deployment.ID); err != nil {
			printError(err)
			return err
		}
		fmt.Printf("%s Deployment started!\n", colorGreen("✓"))
		fmt.Printf("\n💡 Monitor progress: popctl bootstrap status %s --watch\n", deployment.ID)
	} else {
		fmt.Printf("\n💡 To start deployment:\n")
		fmt.Printf("   popctl bootstrap resume %s\n", deployment.ID)
	}

	return nil
}

func runBootstrapStatus(cmd *cobra.Command, args []string) error {
	deploymentID := args[0]
	watch, _ := cmd.Flags().GetBool("watch")

	client, err := getClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	for {
		deployment, err := client.GetDeployment(ctx, deploymentID)
		if err != nil {
			printError(err)
			return err
		}

		if jsonOut {
			return printJSON(deployment)
		}

		// Clear screen if watching
		if watch {
			fmt.Print("\033[H\033[2J")
		}

		fmt.Printf("Deployment: %s\n", deployment.ID)
		fmt.Printf("Chain ID:   %d\n", deployment.ChainID)
		fmt.Printf("Stack:      %s\n", deployment.Stack)
		fmt.Printf("Status:     %s\n", statusEmoji(deployment.Status))

		if deployment.CurrentStage != nil && *deployment.CurrentStage != "" {
			fmt.Printf("Stage:      %s\n", *deployment.CurrentStage)
		}

		if deployment.Error != nil && *deployment.Error != "" {
			fmt.Printf("Error:      %s\n", colorRed(*deployment.Error))
		}

		fmt.Printf("Created:    %s\n", deployment.CreatedAt)
		fmt.Printf("Updated:    %s\n", deployment.UpdatedAt)

		// Stop watching if deployment is terminal
		if !watch || deployment.Status == "completed" || deployment.Status == "failed" {
			break
		}

		if watch {
			fmt.Printf("\n%s Watching for updates (Ctrl+C to stop)...\n", colorYellow("⏳"))
		}

		time.Sleep(2 * time.Second)
	}

	return nil
}

func runBootstrapList(cmd *cobra.Command, args []string) error {
	statusFilter, _ := cmd.Flags().GetString("status")

	client, err := getClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	deployments, err := client.ListDeployments(ctx, statusFilter)
	if err != nil {
		printError(err)
		return err
	}

	if jsonOut {
		return printJSON(map[string]interface{}{
			"deployments": deployments,
			"count":       len(deployments),
		})
	}

	if len(deployments) == 0 {
		fmt.Println("No deployments found")
		return nil
	}

	w := newTable()
	printTableHeader(w, "ID", "CHAIN ID", "STACK", "STATUS", "CREATED")
	for _, d := range deployments {
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n",
			truncate(d.ID, 12),
			d.ChainID,
			d.Stack,
			statusEmoji(d.Status),
			d.CreatedAt,
		)
	}
	return w.Flush()
}

func runBootstrapArtifacts(cmd *cobra.Command, args []string) error {
	deploymentID := args[0]
	outputDir, _ := cmd.Flags().GetString("output")

	client, err := getClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	artifacts, err := client.GetArtifacts(ctx, deploymentID)
	if err != nil {
		printError(err)
		return err
	}

	if len(artifacts) == 0 {
		fmt.Println("No artifacts available yet")
		fmt.Println("The deployment may still be in progress or pending.")
		return nil
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for _, a := range artifacts {
		filename := getArtifactFilename(a.Type)
		filepath := filepath.Join(outputDir, filename)

		data, err := json.MarshalIndent(a.Content, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal %s: %w", a.Type, err)
		}

		if err := os.WriteFile(filepath, data, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filepath, err)
		}

		fmt.Printf("%s %s\n", colorGreen("✓"), filepath)
	}

	fmt.Printf("\n%s Artifacts saved to %s\n", colorGreen("✓"), outputDir)
	return nil
}

func runBootstrapResume(cmd *cobra.Command, args []string) error {
	deploymentID := args[0]

	client, err := getClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// First get the current status
	deployment, err := client.GetDeployment(ctx, deploymentID)
	if err != nil {
		printError(err)
		return err
	}

	if deployment.Status != "pending" && deployment.Status != "paused" {
		return fmt.Errorf("cannot resume deployment with status '%s' (must be pending or paused)", deployment.Status)
	}

	if err := client.StartDeployment(ctx, deploymentID); err != nil {
		printError(err)
		return err
	}

	if jsonOut {
		return printJSON(map[string]string{
			"status":  "started",
			"message": fmt.Sprintf("Deployment %s started", deploymentID),
		})
	}

	fmt.Printf("%s Deployment resumed\n", colorGreen("✓"))
	fmt.Printf("\n💡 Monitor progress: popctl bootstrap status %s --watch\n", deploymentID)

	return nil
}

// Helper functions

func statusEmoji(status string) string {
	switch status {
	case "pending":
		return "⏳ pending"
	case "running":
		return colorYellow("🔄 running")
	case "completed":
		return colorGreen("✅ completed")
	case "failed":
		return colorRed("❌ failed")
	case "paused":
		return "⏸️  paused"
	default:
		return status
	}
}

func getArtifactFilename(artifactType string) string {
	switch artifactType {
	case "genesis":
		return "genesis.json"
	case "rollup_config":
		return "rollup.json"
	case "state":
		return "state.json"
	case "chain_info":
		return "chain-info.json"
	case "node_config":
		return "node-config.json"
	case "core_contracts":
		return "core-contracts.json"
	default:
		return artifactType + ".json"
	}
}

func toInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		return int64(n), true
	default:
		return 0, false
	}
}

