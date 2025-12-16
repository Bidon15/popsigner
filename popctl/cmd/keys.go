package cmd

import (
	"context"
	"fmt"

	"github.com/Bidon15/popsigner/popctl/internal/api"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage cryptographic keys",
	Long: `Key management commands for the POPSigner control plane.

Examples:
  popctl keys list
  popctl keys create sequencer-main --exportable
  popctl keys create-batch blob-worker --count 4
  popctl keys get 01HXYZ...
  popctl keys delete 01HXYZ... --force
  popctl keys export 01HXYZ...`,
}

var keysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all keys",
	RunE:  runKeysList,
}

var keysCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new key",
	Long: `Create a new cryptographic key in the specified namespace.

Examples:
  popctl keys create my-sequencer
  popctl keys create backup-key --exportable
  popctl keys create dev-key --namespace 01HXYZ...`,
	Args: cobra.ExactArgs(1),
	RunE: runKeysCreate,
}

var keysCreateBatchCmd = &cobra.Command{
	Use:   "create-batch <prefix>",
	Short: "Create multiple keys at once",
	Long: `Create multiple keys in parallel with a common prefix.

Keys are named "{prefix}-1", "{prefix}-2", etc.

Examples:
  popctl keys create-batch blob-worker --count 4
  # Creates: blob-worker-1, blob-worker-2, blob-worker-3, blob-worker-4`,
	Args: cobra.ExactArgs(1),
	RunE: runKeysCreateBatch,
}

var keysGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get key details",
	Args:  cobra.ExactArgs(1),
	RunE:  runKeysGet,
}

var keysDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a key",
	Long: `Permanently delete a key. This action cannot be undone.

WARNING: Deleting a key will make any assets controlled by that key
permanently inaccessible.`,
	Args: cobra.ExactArgs(1),
	RunE: runKeysDelete,
}

var keysImportCmd = &cobra.Command{
	Use:   "import <name>",
	Short: "Import a private key",
	Long: `Import an existing private key into POPSigner.

The private key must be provided as a base64-encoded string.

Examples:
  popctl keys import imported-key --private-key <base64>`,
	Args: cobra.ExactArgs(1),
	RunE: runKeysImport,
}

var keysExportCmd = &cobra.Command{
	Use:   "export <id>",
	Short: "Export a key's private key",
	Long: `Export the private key material from an exportable key.

Only keys created with --exportable can be exported.

WARNING: Handle exported private keys with extreme care.`,
	Args: cobra.ExactArgs(1),
	RunE: runKeysExport,
}

func init() {
	// List flags
	keysListCmd.Flags().String("namespace", "", "filter by namespace ID")

	// Create flags
	keysCreateCmd.Flags().String("namespace", "", "namespace ID (uses default if not set)")
	keysCreateCmd.Flags().Bool("exportable", false, "allow key to be exported")
	keysCreateCmd.Flags().StringToString("metadata", nil, "key metadata (key=value pairs)")

	// Create batch flags
	keysCreateBatchCmd.Flags().String("namespace", "", "namespace ID (uses default if not set)")
	keysCreateBatchCmd.Flags().Int("count", 0, "number of keys to create (required, max 100)")
	keysCreateBatchCmd.Flags().Bool("exportable", false, "allow keys to be exported")
	_ = keysCreateBatchCmd.MarkFlagRequired("count")

	// Delete flags
	keysDeleteCmd.Flags().BoolP("force", "f", false, "skip confirmation prompt")
	keysDeleteCmd.Flags().BoolP("yes", "y", false, "skip confirmation prompt (alias for --force)")

	// Import flags
	keysImportCmd.Flags().String("namespace", "", "namespace ID (uses default if not set)")
	keysImportCmd.Flags().String("private-key", "", "base64-encoded private key (required)")
	keysImportCmd.Flags().Bool("exportable", false, "allow key to be exported")
	_ = keysImportCmd.MarkFlagRequired("private-key")

	// Add subcommands
	keysCmd.AddCommand(keysListCmd)
	keysCmd.AddCommand(keysCreateCmd)
	keysCmd.AddCommand(keysCreateBatchCmd)
	keysCmd.AddCommand(keysGetCmd)
	keysCmd.AddCommand(keysDeleteCmd)
	keysCmd.AddCommand(keysImportCmd)
	keysCmd.AddCommand(keysExportCmd)

	rootCmd.AddCommand(keysCmd)
}

func runKeysList(cmd *cobra.Command, args []string) error {
	client, err := getClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	var nsID *uuid.UUID
	if nsStr, _ := cmd.Flags().GetString("namespace"); nsStr != "" {
		id, err := uuid.Parse(nsStr)
		if err != nil {
			return fmt.Errorf("invalid namespace ID: %w", err)
		}
		nsID = &id
	}

	keys, err := client.ListKeys(ctx, nsID)
	if err != nil {
		printError(err)
		return err
	}

	if jsonOut {
		return printJSON(map[string]interface{}{
			"keys":  keys,
			"count": len(keys),
		})
	}

	if len(keys) == 0 {
		fmt.Println("No keys found")
		return nil
	}

	w := newTable()
	printTableHeader(w, "ID", "NAME", "ADDRESS", "ALGORITHM", "EXPORTABLE")
	for _, k := range keys {
		exportable := "-"
		if k.Exportable {
			exportable = colorGreen("yes")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			truncate(k.ID.String(), 12),
			k.Name,
			truncate(k.Address, 16),
			k.Algorithm,
			exportable,
		)
	}
	return w.Flush()
}

func runKeysCreate(cmd *cobra.Command, args []string) error {
	keyName := args[0]

	client, err := getClient()
	if err != nil {
		return err
	}

	nsStr, _ := cmd.Flags().GetString("namespace")
	nsID, err := getNamespaceID(nsStr)
	if err != nil {
		return err
	}

	exportable, _ := cmd.Flags().GetBool("exportable")
	metadata, _ := cmd.Flags().GetStringToString("metadata")

	ctx := context.Background()

	key, err := client.CreateKey(ctx, api.CreateKeyRequest{
		Name:        keyName,
		NamespaceID: nsID,
		Algorithm:   "secp256k1",
		Exportable:  exportable,
		Metadata:    metadata,
	})
	if err != nil {
		printError(err)
		return err
	}

	if jsonOut {
		return printJSON(key)
	}

	fmt.Printf("%s Created key: %s\n", colorGreen("✓"), keyName)
	fmt.Printf("  ID:        %s\n", key.ID)
	fmt.Printf("  Address:   %s\n", key.Address)
	fmt.Printf("  PublicKey: %s\n", truncate(key.PublicKey, 40))
	if exportable {
		fmt.Printf("  %s Key is marked as exportable\n", colorYellow("⚠"))
	}
	return nil
}

func runKeysCreateBatch(cmd *cobra.Command, args []string) error {
	prefix := args[0]

	client, err := getClient()
	if err != nil {
		return err
	}

	nsStr, _ := cmd.Flags().GetString("namespace")
	nsID, err := getNamespaceID(nsStr)
	if err != nil {
		return err
	}

	count, _ := cmd.Flags().GetInt("count")
	if count <= 0 || count > 100 {
		return fmt.Errorf("count must be between 1 and 100")
	}

	exportable, _ := cmd.Flags().GetBool("exportable")

	ctx := context.Background()

	keys, err := client.CreateKeysBatch(ctx, api.CreateBatchRequest{
		Prefix:      prefix,
		Count:       count,
		NamespaceID: nsID,
		Exportable:  exportable,
	})
	if err != nil {
		printError(err)
		return err
	}

	if jsonOut {
		return printJSON(map[string]interface{}{
			"keys":  keys,
			"count": len(keys),
		})
	}

	fmt.Printf("%s Created %d keys with prefix '%s'\n", colorGreen("✓"), len(keys), prefix)
	w := newTable()
	printTableHeader(w, "NAME", "ADDRESS", "ID")
	for _, k := range keys {
		fmt.Fprintf(w, "%s\t%s\t%s\n", k.Name, truncate(k.Address, 16), truncate(k.ID.String(), 12))
	}
	return w.Flush()
}

func runKeysGet(cmd *cobra.Command, args []string) error {
	keyID, err := uuid.Parse(args[0])
	if err != nil {
		return fmt.Errorf("invalid key ID: %w", err)
	}

	client, err := getClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	key, err := client.GetKey(ctx, keyID)
	if err != nil {
		printError(err)
		return err
	}

	if jsonOut {
		return printJSON(key)
	}

	fmt.Printf("ID:          %s\n", key.ID)
	fmt.Printf("Name:        %s\n", key.Name)
	fmt.Printf("Namespace:   %s\n", key.NamespaceID)
	fmt.Printf("Address:     %s\n", key.Address)
	fmt.Printf("PublicKey:   %s\n", key.PublicKey)
	fmt.Printf("Algorithm:   %s\n", key.Algorithm)
	fmt.Printf("Exportable:  %v\n", key.Exportable)
	fmt.Printf("Version:     %d\n", key.Version)
	fmt.Printf("Created:     %s\n", key.CreatedAt.Format("2006-01-02 15:04:05"))
	if len(key.Metadata) > 0 {
		fmt.Println("Metadata:")
		for k, v := range key.Metadata {
			fmt.Printf("  %s: %s\n", k, v)
		}
	}
	return nil
}

func runKeysDelete(cmd *cobra.Command, args []string) error {
	keyID, err := uuid.Parse(args[0])
	if err != nil {
		return fmt.Errorf("invalid key ID: %w", err)
	}

	force, _ := cmd.Flags().GetBool("force")
	yes, _ := cmd.Flags().GetBool("yes")
	if yes {
		force = true
	}

	if !force {
		fmt.Printf("%s Are you sure you want to delete key %s? [y/N]: ", colorYellow("⚠"), keyID)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Aborted")
			return nil
		}
	}

	client, err := getClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	if err := client.DeleteKey(ctx, keyID); err != nil {
		printError(err)
		return err
	}

	if jsonOut {
		return printJSON(map[string]string{
			"status":  "deleted",
			"message": fmt.Sprintf("Key %s deleted", keyID),
		})
	}

	fmt.Printf("%s Deleted key: %s\n", colorGreen("✓"), keyID)
	return nil
}

func runKeysImport(cmd *cobra.Command, args []string) error {
	keyName := args[0]

	client, err := getClient()
	if err != nil {
		return err
	}

	nsStr, _ := cmd.Flags().GetString("namespace")
	nsID, err := getNamespaceID(nsStr)
	if err != nil {
		return err
	}

	privateKey, _ := cmd.Flags().GetString("private-key")
	exportable, _ := cmd.Flags().GetBool("exportable")

	ctx := context.Background()

	key, err := client.ImportKey(ctx, api.ImportKeyRequest{
		Name:        keyName,
		NamespaceID: nsID,
		PrivateKey:  privateKey,
		Exportable:  exportable,
	})
	if err != nil {
		printError(err)
		return err
	}

	if jsonOut {
		return printJSON(key)
	}

	fmt.Printf("%s Imported key: %s\n", colorGreen("✓"), keyName)
	fmt.Printf("  ID:        %s\n", key.ID)
	fmt.Printf("  Address:   %s\n", key.Address)
	fmt.Printf("  PublicKey: %s\n", truncate(key.PublicKey, 40))
	return nil
}

func runKeysExport(cmd *cobra.Command, args []string) error {
	keyID, err := uuid.Parse(args[0])
	if err != nil {
		return fmt.Errorf("invalid key ID: %w", err)
	}

	client, err := getClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	result, err := client.ExportKey(ctx, keyID)
	if err != nil {
		printError(err)
		return err
	}

	if jsonOut {
		return printJSON(map[string]string{
			"key_id":      keyID.String(),
			"private_key": result.PrivateKey,
			"warning":     result.Warning,
		})
	}

	fmt.Printf("%s Exported private key for %s\n", colorYellow("⚠"), keyID)
	fmt.Printf("\n%s\n\n", result.PrivateKey)
	fmt.Printf("%s %s\n", colorRed("WARNING:"), result.Warning)

	return nil
}

