package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/Bidon15/banhbaoring"
	"github.com/spf13/cobra"
)

// keysCmd is the parent command for key operations.
var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage keys in OpenBao",
	Long: `Key management commands for OpenBao keyring.

Available subcommands:
  list    - List all keys
  show    - Show key details by name
  add     - Create a new key
  delete  - Delete a key
  rename  - Rename a key
  export-pub - Export public key (armored)`,
}

// keysListCmd lists all keys.
var keysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all keys",
	RunE:  runKeysList,
}

// keysShowCmd shows key details by UID.
var keysShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show key details",
	Args:  cobra.ExactArgs(1),
	RunE:  runKeysShow,
}

// keysAddCmd creates a new key.
var keysAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new key",
	Args:  cobra.ExactArgs(1),
	RunE:  runKeysAdd,
}

// keysDeleteCmd deletes a key.
var keysDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a key",
	Args:  cobra.ExactArgs(1),
	RunE:  runKeysDelete,
}

// keysRenameCmd renames a key.
var keysRenameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Rename a key",
	Args:  cobra.ExactArgs(2),
	RunE:  runKeysRename,
}

// keysExportPubCmd exports a public key in armored format.
var keysExportPubCmd = &cobra.Command{
	Use:   "export-pub <name>",
	Short: "Export public key (armored)",
	Args:  cobra.ExactArgs(1),
	RunE:  runKeysExportPub,
}

func init() {
	// Add flags to subcommands
	keysAddCmd.Flags().Bool("exportable", false, "Allow key export from OpenBao")
	keysDeleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	keysDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt (alias for --force)")

	// Add subcommands to keys command
	keysCmd.AddCommand(keysListCmd)
	keysCmd.AddCommand(keysShowCmd)
	keysCmd.AddCommand(keysAddCmd)
	keysCmd.AddCommand(keysDeleteCmd)
	keysCmd.AddCommand(keysRenameCmd)
	keysCmd.AddCommand(keysExportPubCmd)
}

// KeyListOutput represents a key in the list output.
type KeyListOutput struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// KeyListResult represents the list output.
type KeyListResult struct {
	Keys []KeyListOutput `json:"keys"`
}

func runKeysList(cmd *cobra.Command, args []string) error {
	kr, err := getKeyring()
	if err != nil {
		return err
	}
	defer func() { _ = kr.Close() }()

	records, err := kr.List()
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	if jsonOut {
		result := KeyListResult{Keys: make([]KeyListOutput, 0, len(records))}
		for _, r := range records {
			addr, _ := r.GetAddress()
			result.Keys = append(result.Keys, KeyListOutput{
				Name:    r.Name,
				Address: addr.String(),
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	if len(records) == 0 {
		fmt.Println("No keys found")
		return nil
	}

	// Table output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "%s\t%s\n", colorBold("NAME"), colorBold("ADDRESS"))
	for _, r := range records {
		addr, _ := r.GetAddress()
		_, _ = fmt.Fprintf(w, "%s\t%s\n", r.Name, addr.String())
	}
	return w.Flush()
}

// KeyShowOutput represents the show command output.
type KeyShowOutput struct {
	Name       string `json:"name"`
	Address    string `json:"address"`
	Algorithm  string `json:"algorithm"`
	Exportable bool   `json:"exportable"`
	CreatedAt  string `json:"created_at"`
	Source     string `json:"source"`
}

func runKeysShow(cmd *cobra.Command, args []string) error {
	keyName := args[0]

	kr, err := getKeyring()
	if err != nil {
		return err
	}
	defer func() { _ = kr.Close() }()

	meta, err := kr.GetMetadata(keyName)
	if err != nil {
		return fmt.Errorf("key not found: %s", keyName)
	}

	if jsonOut {
		output := KeyShowOutput{
			Name:       meta.Name,
			Address:    meta.Address,
			Algorithm:  meta.Algorithm,
			Exportable: meta.Exportable,
			CreatedAt:  meta.CreatedAt.Format("2006-01-02 15:04:05"),
			Source:     meta.Source,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	fmt.Printf("Name:       %s\n", meta.Name)
	fmt.Printf("Address:    %s\n", meta.Address)
	fmt.Printf("Algorithm:  %s\n", meta.Algorithm)
	fmt.Printf("Exportable: %v\n", meta.Exportable)
	fmt.Printf("Created:    %s\n", meta.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Source:     %s\n", meta.Source)
	return nil
}

// KeyAddOutput represents the add command output.
type KeyAddOutput struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Status  string `json:"status"`
}

func runKeysAdd(cmd *cobra.Command, args []string) error {
	keyName := args[0]
	exportable, _ := cmd.Flags().GetBool("exportable")

	kr, err := getKeyring()
	if err != nil {
		return err
	}
	defer func() { _ = kr.Close() }()

	record, err := kr.NewAccountWithOptions(keyName, banhbaoring.KeyOptions{
		Exportable: exportable,
	})
	if err != nil {
		return fmt.Errorf("failed to create key: %w", err)
	}

	addr, _ := record.GetAddress()

	if jsonOut {
		output := KeyAddOutput{
			Name:    keyName,
			Address: addr.String(),
			Status:  "created",
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	fmt.Printf("%s Created key: %s\n", colorGreen("✓"), keyName)
	fmt.Printf("Address: %s\n", addr.String())
	if exportable {
		fmt.Printf("%s Key is marked as exportable\n", colorYellow("⚠"))
	}
	return nil
}

// KeyDeleteOutput represents the delete command output.
type KeyDeleteOutput struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func runKeysDelete(cmd *cobra.Command, args []string) error {
	keyName := args[0]
	force, _ := cmd.Flags().GetBool("force")
	yes, _ := cmd.Flags().GetBool("yes")

	// --yes is an alias for --force
	if yes {
		force = true
	}

	if !force {
		fmt.Printf("%s Are you sure you want to delete key %q? [y/N]: ", colorYellow("⚠"), keyName)
		var response string
		_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Aborted")
			return nil
		}
	}

	kr, err := getKeyring()
	if err != nil {
		return err
	}
	defer func() { _ = kr.Close() }()

	if err := kr.Delete(keyName); err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}

	if jsonOut {
		output := KeyDeleteOutput{
			Name:   keyName,
			Status: "deleted",
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	fmt.Printf("%s Deleted key: %s\n", colorGreen("✓"), keyName)
	return nil
}

// KeyRenameOutput represents the rename command output.
type KeyRenameOutput struct {
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
	Status  string `json:"status"`
}

func runKeysRename(cmd *cobra.Command, args []string) error {
	oldName := args[0]
	newName := args[1]

	kr, err := getKeyring()
	if err != nil {
		return err
	}
	defer func() { _ = kr.Close() }()

	if err := kr.Rename(oldName, newName); err != nil {
		return fmt.Errorf("failed to rename key: %w", err)
	}

	if jsonOut {
		output := KeyRenameOutput{
			OldName: oldName,
			NewName: newName,
			Status:  "renamed",
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	fmt.Printf("%s Renamed key: %s → %s\n", colorGreen("✓"), oldName, newName)
	return nil
}

// KeyExportPubOutput represents the export-pub command output.
type KeyExportPubOutput struct {
	Name  string `json:"name"`
	Armor string `json:"armor"`
}

func runKeysExportPub(cmd *cobra.Command, args []string) error {
	keyName := args[0]

	kr, err := getKeyring()
	if err != nil {
		return err
	}
	defer func() { _ = kr.Close() }()

	armor, err := kr.ExportPubKeyArmor(keyName)
	if err != nil {
		return fmt.Errorf("failed to export public key: %w", err)
	}

	if jsonOut {
		output := KeyExportPubOutput{
			Name:  keyName,
			Armor: armor,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	fmt.Println(armor)
	return nil
}
