package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/Bidon15/banhbaoring/migration"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/spf13/cobra"
)

// migrateCmd is the parent command for migration operations.
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate keys between keyrings",
	Long: `Migration commands for transferring keys between local keyring and OpenBao.

Available subcommands:
  import - Import key from local keyring to OpenBao
  export - Export key from OpenBao to local keyring`,
}

// migrateImportCmd imports keys from local keyring.
var migrateImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import key from local keyring to OpenBao",
	Long: `Import a key from a local Cosmos SDK keyring to OpenBao.

The source keyring can be any Cosmos SDK compatible keyring backend
(file, os, test). After import, the key will be stored in OpenBao
and signing operations will be performed there.

Examples:
  # Import a single key
  banhbao migrate import --from ~/.celestia-app --key-name mykey

  # Import all keys
  banhbao migrate import --from ~/.celestia-app --all

  # Import and delete from source
  banhbao migrate import --from ~/.celestia-app --key-name mykey --delete-after-import`,
	RunE: runMigrateImport,
}

// migrateExportCmd exports keys to local keyring.
var migrateExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export key from OpenBao to local keyring",
	Long: `Export a key from OpenBao to a local Cosmos SDK keyring.

⚠️  SECURITY WARNING: Exporting keys from OpenBao compromises the
security model. The private key will be exposed to the local filesystem.
Only export keys when absolutely necessary.

The key must be marked as exportable when created.

Examples:
  # Export a key (requires confirmation)
  banhbao migrate export --to ~/.celestia-app --key-name mykey \
    --confirm "I understand this compromises key security"`,
	RunE: runMigrateExport,
}

func init() {
	// Import flags
	migrateImportCmd.Flags().String("from", "", "Source keyring path (required)")
	migrateImportCmd.Flags().String("backend", "file", "Source keyring backend (file, os, test)")
	migrateImportCmd.Flags().String("key-name", "", "Key name to import (required unless --all)")
	migrateImportCmd.Flags().String("new-name", "", "New name for imported key (optional)")
	migrateImportCmd.Flags().Bool("delete-after-import", false, "Delete key from source after import")
	migrateImportCmd.Flags().Bool("all", false, "Import all keys from source keyring")
	migrateImportCmd.Flags().Bool("exportable", false, "Allow key to be exported from OpenBao")
	migrateImportCmd.Flags().Bool("verify", true, "Verify import by signing test data")

	// Export flags
	migrateExportCmd.Flags().String("to", "", "Destination keyring path (required)")
	migrateExportCmd.Flags().String("backend", "file", "Destination keyring backend (file, os, test)")
	migrateExportCmd.Flags().String("key-name", "", "Key name to export (required)")
	migrateExportCmd.Flags().String("new-name", "", "New name for exported key (optional)")
	migrateExportCmd.Flags().String("confirm", "", "Confirmation string (required for export)")
	migrateExportCmd.Flags().Bool("verify", true, "Verify export by signing test data")

	migrateCmd.AddCommand(migrateImportCmd)
	migrateCmd.AddCommand(migrateExportCmd)
}

// MigrateImportOutput represents the import command output.
type MigrateImportOutput struct {
	KeyName  string `json:"key_name"`
	Address  string `json:"address"`
	Verified bool   `json:"verified"`
	Status   string `json:"status"`
}

// MigrateBatchImportOutput represents the batch import output.
type MigrateBatchImportOutput struct {
	Successful []MigrateImportOutput `json:"successful"`
	Failed     []MigrateFailedOutput `json:"failed"`
}

// MigrateFailedOutput represents a failed import.
type MigrateFailedOutput struct {
	KeyName string `json:"key_name"`
	Error   string `json:"error"`
}

func runMigrateImport(cmd *cobra.Command, args []string) error {
	fromPath, _ := cmd.Flags().GetString("from")
	backend, _ := cmd.Flags().GetString("backend")
	keyName, _ := cmd.Flags().GetString("key-name")
	newName, _ := cmd.Flags().GetString("new-name")
	deleteAfter, _ := cmd.Flags().GetBool("delete-after-import")
	all, _ := cmd.Flags().GetBool("all")
	exportable, _ := cmd.Flags().GetBool("exportable")
	verify, _ := cmd.Flags().GetBool("verify")

	// Validate flags
	if fromPath == "" {
		return fmt.Errorf("--from is required")
	}
	if !all && keyName == "" {
		return fmt.Errorf("--key-name is required (or use --all to import all keys)")
	}

	// Open source keyring
	sourceKr, err := keyring.New("celestia", backend, fromPath, os.Stdin, nil)
	if err != nil {
		return fmt.Errorf("failed to open source keyring: %w", err)
	}

	// Open destination (BaoKeyring)
	destKr, err := getKeyring()
	if err != nil {
		return err
	}
	defer func() { _ = destKr.Close() }()

	ctx := context.Background()

	if all {
		// Batch import
		result, err := migration.BatchImport(ctx, migration.BatchImportConfig{
			SourceKeyring:     sourceKr,
			DestKeyring:       destKr,
			DeleteAfterImport: deleteAfter,
			Exportable:        exportable,
			VerifyAfterImport: verify,
		})
		if err != nil {
			return fmt.Errorf("batch import failed: %w", err)
		}

		if jsonOut {
			output := MigrateBatchImportOutput{
				Successful: make([]MigrateImportOutput, 0, len(result.Successful)),
				Failed:     make([]MigrateFailedOutput, 0, len(result.Failed)),
			}
			for _, s := range result.Successful {
				output.Successful = append(output.Successful, MigrateImportOutput{
					KeyName:  s.KeyName,
					Address:  s.Address,
					Verified: s.Verified,
					Status:   "imported",
				})
			}
			for _, f := range result.Failed {
				output.Failed = append(output.Failed, MigrateFailedOutput{
					KeyName: f.KeyName,
					Error:   f.Error.Error(),
				})
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(output)
		}

		// Table output
		fmt.Printf("%s Imported %d keys successfully\n", colorGreen("✓"), len(result.Successful))

		if len(result.Successful) > 0 {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", colorBold("NAME"), colorBold("ADDRESS"), colorBold("VERIFIED"))
			for _, s := range result.Successful {
				verifiedStr := colorGreen("yes")
				if !s.Verified {
					verifiedStr = colorRed("no")
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", s.KeyName, s.Address, verifiedStr)
			}
			_ = w.Flush()
		}

		if len(result.Failed) > 0 {
			fmt.Printf("\n%s Failed to import %d keys:\n", colorRed("✗"), len(result.Failed))
			for _, f := range result.Failed {
				fmt.Printf("  %s %s: %v\n", colorRed("✗"), f.KeyName, f.Error)
			}
		}

		return nil
	}

	// Single import
	result, err := migration.Import(ctx, migration.ImportConfig{
		SourceKeyring:     sourceKr,
		DestKeyring:       destKr,
		KeyName:           keyName,
		NewKeyName:        newName,
		DeleteAfterImport: deleteAfter,
		Exportable:        exportable,
		VerifyAfterImport: verify,
	})
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	if jsonOut {
		output := MigrateImportOutput{
			KeyName:  result.KeyName,
			Address:  result.Address,
			Verified: result.Verified,
			Status:   "imported",
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	fmt.Printf("%s Imported key: %s\n", colorGreen("✓"), result.KeyName)
	fmt.Printf("Address: %s\n", result.Address)
	if result.Verified {
		fmt.Printf("Verified: %s\n", colorGreen("yes"))
	} else {
		fmt.Printf("Verified: %s\n", colorRed("no"))
	}

	return nil
}

// MigrateExportOutput represents the export command output.
type MigrateExportOutput struct {
	KeyName  string `json:"key_name"`
	Address  string `json:"address"`
	DestPath string `json:"dest_path"`
	Verified bool   `json:"verified"`
	Status   string `json:"status"`
}

const requiredConfirmation = "I understand this compromises key security"

func runMigrateExport(cmd *cobra.Command, args []string) error {
	toPath, _ := cmd.Flags().GetString("to")
	backend, _ := cmd.Flags().GetString("backend")
	keyName, _ := cmd.Flags().GetString("key-name")
	newName, _ := cmd.Flags().GetString("new-name")
	confirm, _ := cmd.Flags().GetString("confirm")
	verify, _ := cmd.Flags().GetBool("verify")

	// Validate flags
	if toPath == "" {
		return fmt.Errorf("--to is required")
	}
	if keyName == "" {
		return fmt.Errorf("--key-name is required")
	}

	// Check confirmation
	if confirm != requiredConfirmation {
		fmt.Println(colorRed("⚠️  SECURITY WARNING"))
		fmt.Println()
		fmt.Println(migration.SecurityWarning(keyName, "", toPath))
		fmt.Println()
		fmt.Printf("To proceed, use:\n")
		fmt.Printf("  --confirm '%s'\n", requiredConfirmation)
		return nil
	}

	// Open source (BaoKeyring)
	sourceKr, err := getKeyring()
	if err != nil {
		return err
	}
	defer func() { _ = sourceKr.Close() }()

	// Open destination keyring
	destKr, err := keyring.New("celestia", backend, toPath, os.Stdin, nil)
	if err != nil {
		return fmt.Errorf("failed to open destination keyring: %w", err)
	}

	ctx := context.Background()

	result, err := migration.Export(ctx, migration.ExportConfig{
		SourceKeyring:     sourceKr,
		DestKeyring:       destKr,
		KeyName:           keyName,
		NewKeyName:        newName,
		VerifyAfterExport: verify,
		Confirmed:         true,
	})
	if err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	if jsonOut {
		output := MigrateExportOutput{
			KeyName:  result.KeyName,
			Address:  result.Address,
			DestPath: result.DestPath,
			Verified: result.Verified,
			Status:   "exported",
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	fmt.Printf("%s Exported key: %s\n", colorYellow("⚠"), result.KeyName)
	fmt.Printf("Address: %s\n", result.Address)
	fmt.Printf("Destination: %s\n", result.DestPath)
	if result.Verified {
		fmt.Printf("Verified: %s\n", colorGreen("yes"))
	} else {
		fmt.Printf("Verified: %s\n", colorRed("no"))
	}

	return nil
}
