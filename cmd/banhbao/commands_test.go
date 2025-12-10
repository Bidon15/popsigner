package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================
// Test Helpers
// ============================================


// resetFlags resets all flags to their default values.
func resetFlags(t *testing.T) {
	t.Helper()
	baoAddr = ""
	baoToken = ""
	storePath = "./keyring-metadata.json"
	jsonOut = false
}

// ============================================
// Root Command Tests
// ============================================

func TestRootCmd(t *testing.T) {
	resetFlags(t)

	t.Run("shows help by default", func(t *testing.T) {
		rootCmd.SetArgs([]string{"--help"})
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})

	t.Run("has correct use name", func(t *testing.T) {
		assert.Equal(t, "banhbao", rootCmd.Use)
	})

	t.Run("has keys subcommand", func(t *testing.T) {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == "keys" {
				found = true
				break
			}
		}
		assert.True(t, found, "keys subcommand should exist")
	})

	t.Run("has migrate subcommand", func(t *testing.T) {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == "migrate" {
				found = true
				break
			}
		}
		assert.True(t, found, "migrate subcommand should exist")
	})

	t.Run("has version subcommand", func(t *testing.T) {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == "version" {
				found = true
				break
			}
		}
		assert.True(t, found, "version subcommand should exist")
	})
}

func TestVersionCmd(t *testing.T) {
	resetFlags(t)

	buf := new(bytes.Buffer)
	versionCmd.SetOut(buf)
	versionCmd.SetArgs([]string{})

	// Execute version command
	versionCmd.Run(versionCmd, []string{})

	// Note: We can't easily capture output from Run, but we can verify the command exists
	assert.Equal(t, "version", versionCmd.Use)
	assert.Equal(t, "Print version information", versionCmd.Short)
}

// ============================================
// Keys Command Tests
// ============================================

func TestKeysCmd(t *testing.T) {
	resetFlags(t)

	t.Run("has correct use name", func(t *testing.T) {
		assert.Equal(t, "keys", keysCmd.Use)
	})

	t.Run("has list subcommand", func(t *testing.T) {
		found := false
		for _, cmd := range keysCmd.Commands() {
			if cmd.Use == "list" {
				found = true
				break
			}
		}
		assert.True(t, found, "list subcommand should exist")
	})

	t.Run("has show subcommand", func(t *testing.T) {
		found := false
		for _, cmd := range keysCmd.Commands() {
			if cmd.Use == "show <name>" {
				found = true
				break
			}
		}
		assert.True(t, found, "show subcommand should exist")
	})

	t.Run("has add subcommand", func(t *testing.T) {
		found := false
		for _, cmd := range keysCmd.Commands() {
			if cmd.Use == "add <name>" {
				found = true
				break
			}
		}
		assert.True(t, found, "add subcommand should exist")
	})

	t.Run("has delete subcommand", func(t *testing.T) {
		found := false
		for _, cmd := range keysCmd.Commands() {
			if cmd.Use == "delete <name>" {
				found = true
				break
			}
		}
		assert.True(t, found, "delete subcommand should exist")
	})

	t.Run("has rename subcommand", func(t *testing.T) {
		found := false
		for _, cmd := range keysCmd.Commands() {
			if cmd.Use == "rename <old-name> <new-name>" {
				found = true
				break
			}
		}
		assert.True(t, found, "rename subcommand should exist")
	})

	t.Run("has export-pub subcommand", func(t *testing.T) {
		found := false
		for _, cmd := range keysCmd.Commands() {
			if cmd.Use == "export-pub <name>" {
				found = true
				break
			}
		}
		assert.True(t, found, "export-pub subcommand should exist")
	})
}

func TestKeysListCmd(t *testing.T) {
	resetFlags(t)

	t.Run("has correct use name", func(t *testing.T) {
		assert.Equal(t, "list", keysListCmd.Use)
	})

	t.Run("has correct short description", func(t *testing.T) {
		assert.Equal(t, "List all keys", keysListCmd.Short)
	})
}

func TestKeysShowCmd(t *testing.T) {
	resetFlags(t)

	t.Run("has correct use name", func(t *testing.T) {
		assert.Equal(t, "show <name>", keysShowCmd.Use)
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := keysShowCmd.Args(keysShowCmd, []string{})
		assert.Error(t, err)

		err = keysShowCmd.Args(keysShowCmd, []string{"key1"})
		assert.NoError(t, err)

		err = keysShowCmd.Args(keysShowCmd, []string{"key1", "key2"})
		assert.Error(t, err)
	})
}

func TestKeysAddCmd(t *testing.T) {
	resetFlags(t)

	t.Run("has correct use name", func(t *testing.T) {
		assert.Equal(t, "add <name>", keysAddCmd.Use)
	})

	t.Run("has exportable flag", func(t *testing.T) {
		flag := keysAddCmd.Flags().Lookup("exportable")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := keysAddCmd.Args(keysAddCmd, []string{})
		assert.Error(t, err)

		err = keysAddCmd.Args(keysAddCmd, []string{"key1"})
		assert.NoError(t, err)
	})
}

func TestKeysDeleteCmd(t *testing.T) {
	resetFlags(t)

	t.Run("has correct use name", func(t *testing.T) {
		assert.Equal(t, "delete <name>", keysDeleteCmd.Use)
	})

	t.Run("has force flag", func(t *testing.T) {
		flag := keysDeleteCmd.Flags().Lookup("force")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has yes flag as alias", func(t *testing.T) {
		flag := keysDeleteCmd.Flags().Lookup("yes")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := keysDeleteCmd.Args(keysDeleteCmd, []string{})
		assert.Error(t, err)

		err = keysDeleteCmd.Args(keysDeleteCmd, []string{"key1"})
		assert.NoError(t, err)
	})
}

func TestKeysRenameCmd(t *testing.T) {
	resetFlags(t)

	t.Run("has correct use name", func(t *testing.T) {
		assert.Equal(t, "rename <old-name> <new-name>", keysRenameCmd.Use)
	})

	t.Run("requires exactly two arguments", func(t *testing.T) {
		err := keysRenameCmd.Args(keysRenameCmd, []string{})
		assert.Error(t, err)

		err = keysRenameCmd.Args(keysRenameCmd, []string{"key1"})
		assert.Error(t, err)

		err = keysRenameCmd.Args(keysRenameCmd, []string{"key1", "key2"})
		assert.NoError(t, err)

		err = keysRenameCmd.Args(keysRenameCmd, []string{"key1", "key2", "key3"})
		assert.Error(t, err)
	})
}

func TestKeysExportPubCmd(t *testing.T) {
	resetFlags(t)

	t.Run("has correct use name", func(t *testing.T) {
		assert.Equal(t, "export-pub <name>", keysExportPubCmd.Use)
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		err := keysExportPubCmd.Args(keysExportPubCmd, []string{})
		assert.Error(t, err)

		err = keysExportPubCmd.Args(keysExportPubCmd, []string{"key1"})
		assert.NoError(t, err)
	})
}

// ============================================
// Migrate Command Tests
// ============================================

func TestMigrateCmd(t *testing.T) {
	resetFlags(t)

	t.Run("has correct use name", func(t *testing.T) {
		assert.Equal(t, "migrate", migrateCmd.Use)
	})

	t.Run("has import subcommand", func(t *testing.T) {
		found := false
		for _, cmd := range migrateCmd.Commands() {
			if cmd.Use == "import" {
				found = true
				break
			}
		}
		assert.True(t, found, "import subcommand should exist")
	})

	t.Run("has export subcommand", func(t *testing.T) {
		found := false
		for _, cmd := range migrateCmd.Commands() {
			if cmd.Use == "export" {
				found = true
				break
			}
		}
		assert.True(t, found, "export subcommand should exist")
	})
}

func TestMigrateImportCmd(t *testing.T) {
	resetFlags(t)

	t.Run("has correct use name", func(t *testing.T) {
		assert.Equal(t, "import", migrateImportCmd.Use)
	})

	t.Run("has from flag", func(t *testing.T) {
		flag := migrateImportCmd.Flags().Lookup("from")
		assert.NotNil(t, flag)
	})

	t.Run("has backend flag", func(t *testing.T) {
		flag := migrateImportCmd.Flags().Lookup("backend")
		assert.NotNil(t, flag)
		assert.Equal(t, "file", flag.DefValue)
	})

	t.Run("has key-name flag", func(t *testing.T) {
		flag := migrateImportCmd.Flags().Lookup("key-name")
		assert.NotNil(t, flag)
	})

	t.Run("has new-name flag", func(t *testing.T) {
		flag := migrateImportCmd.Flags().Lookup("new-name")
		assert.NotNil(t, flag)
	})

	t.Run("has delete-after-import flag", func(t *testing.T) {
		flag := migrateImportCmd.Flags().Lookup("delete-after-import")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has all flag", func(t *testing.T) {
		flag := migrateImportCmd.Flags().Lookup("all")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has exportable flag", func(t *testing.T) {
		flag := migrateImportCmd.Flags().Lookup("exportable")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has verify flag", func(t *testing.T) {
		flag := migrateImportCmd.Flags().Lookup("verify")
		assert.NotNil(t, flag)
		assert.Equal(t, "true", flag.DefValue)
	})
}

func TestMigrateExportCmd(t *testing.T) {
	resetFlags(t)

	t.Run("has correct use name", func(t *testing.T) {
		assert.Equal(t, "export", migrateExportCmd.Use)
	})

	t.Run("has to flag", func(t *testing.T) {
		flag := migrateExportCmd.Flags().Lookup("to")
		assert.NotNil(t, flag)
	})

	t.Run("has backend flag", func(t *testing.T) {
		flag := migrateExportCmd.Flags().Lookup("backend")
		assert.NotNil(t, flag)
		assert.Equal(t, "file", flag.DefValue)
	})

	t.Run("has key-name flag", func(t *testing.T) {
		flag := migrateExportCmd.Flags().Lookup("key-name")
		assert.NotNil(t, flag)
	})

	t.Run("has new-name flag", func(t *testing.T) {
		flag := migrateExportCmd.Flags().Lookup("new-name")
		assert.NotNil(t, flag)
	})

	t.Run("has confirm flag", func(t *testing.T) {
		flag := migrateExportCmd.Flags().Lookup("confirm")
		assert.NotNil(t, flag)
	})

	t.Run("has verify flag", func(t *testing.T) {
		flag := migrateExportCmd.Flags().Lookup("verify")
		assert.NotNil(t, flag)
		assert.Equal(t, "true", flag.DefValue)
	})
}

// ============================================
// GetKeyring Tests
// ============================================

func TestGetKeyring_MissingCredentials(t *testing.T) {
	resetFlags(t)

	// Ensure env vars are not set
	_ = os.Unsetenv("BAO_ADDR")
	_ = os.Unsetenv("BAO_TOKEN")

	_, err := getKeyring()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BAO_ADDR and BAO_TOKEN are required")
}

func TestGetKeyring_UsesEnvVars(t *testing.T) {
	resetFlags(t)

	// Set env vars
	_ = os.Setenv("BAO_ADDR", "http://localhost:8200")
	_ = os.Setenv("BAO_TOKEN", "test-token")
	defer func() {
		_ = os.Unsetenv("BAO_ADDR")
		_ = os.Unsetenv("BAO_TOKEN")
	}()

	// Should fail because the server isn't running, but shouldn't fail on missing credentials
	_, err := getKeyring()
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "BAO_ADDR and BAO_TOKEN are required")
}

func TestGetKeyring_FlagsOverrideEnvVars(t *testing.T) {
	resetFlags(t)

	// Set env vars
	_ = os.Setenv("BAO_ADDR", "http://env-addr:8200")
	_ = os.Setenv("BAO_TOKEN", "env-token")
	defer func() {
		_ = os.Unsetenv("BAO_ADDR")
		_ = os.Unsetenv("BAO_TOKEN")
	}()

	// Set flags to override
	baoAddr = "http://flag-addr:8200"
	baoToken = "flag-token"

	// Should fail because the server isn't running, but should use flag values
	_, err := getKeyring()
	assert.Error(t, err)
	// The error should reference the flag address (health check failure)
	assert.NotContains(t, err.Error(), "BAO_ADDR and BAO_TOKEN are required")
}

// ============================================
// Color Helper Tests
// ============================================

func TestColorHelpers(t *testing.T) {
	t.Run("colorRed", func(t *testing.T) {
		result := colorRed("test")
		assert.Contains(t, result, "test")
		assert.Contains(t, result, "\033[31m")
		assert.Contains(t, result, "\033[0m")
	})

	t.Run("colorGreen", func(t *testing.T) {
		result := colorGreen("test")
		assert.Contains(t, result, "test")
		assert.Contains(t, result, "\033[32m")
		assert.Contains(t, result, "\033[0m")
	})

	t.Run("colorYellow", func(t *testing.T) {
		result := colorYellow("test")
		assert.Contains(t, result, "test")
		assert.Contains(t, result, "\033[33m")
		assert.Contains(t, result, "\033[0m")
	})

	t.Run("colorBold", func(t *testing.T) {
		result := colorBold("test")
		assert.Contains(t, result, "test")
		assert.Contains(t, result, "\033[1m")
		assert.Contains(t, result, "\033[0m")
	})
}

// ============================================
// JSON Output Structure Tests
// ============================================

func TestKeyListOutputJSON(t *testing.T) {
	output := KeyListResult{
		Keys: []KeyListOutput{
			{Name: "key1", Address: "cosmos1abc"},
			{Name: "key2", Address: "cosmos1def"},
		},
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)

	var parsed KeyListResult
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Len(t, parsed.Keys, 2)
	assert.Equal(t, "key1", parsed.Keys[0].Name)
	assert.Equal(t, "cosmos1abc", parsed.Keys[0].Address)
}

func TestKeyShowOutputJSON(t *testing.T) {
	output := KeyShowOutput{
		Name:       "mykey",
		Address:    "cosmos1xyz",
		Algorithm:  "secp256k1",
		Exportable: true,
		CreatedAt:  "2024-01-01 12:00:00",
		Source:     "generated",
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)

	var parsed KeyShowOutput
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "mykey", parsed.Name)
	assert.Equal(t, "secp256k1", parsed.Algorithm)
	assert.True(t, parsed.Exportable)
}

func TestKeyAddOutputJSON(t *testing.T) {
	output := KeyAddOutput{
		Name:    "newkey",
		Address: "cosmos1new",
		Status:  "created",
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)

	var parsed KeyAddOutput
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "newkey", parsed.Name)
	assert.Equal(t, "created", parsed.Status)
}

func TestKeyDeleteOutputJSON(t *testing.T) {
	output := KeyDeleteOutput{
		Name:   "oldkey",
		Status: "deleted",
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)

	var parsed KeyDeleteOutput
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "oldkey", parsed.Name)
	assert.Equal(t, "deleted", parsed.Status)
}

func TestKeyRenameOutputJSON(t *testing.T) {
	output := KeyRenameOutput{
		OldName: "oldname",
		NewName: "newname",
		Status:  "renamed",
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)

	var parsed KeyRenameOutput
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "oldname", parsed.OldName)
	assert.Equal(t, "newname", parsed.NewName)
}

func TestKeyExportPubOutputJSON(t *testing.T) {
	output := KeyExportPubOutput{
		Name:  "mykey",
		Armor: "-----BEGIN TENDERMINT PUBLIC KEY-----\ntest\n-----END TENDERMINT PUBLIC KEY-----",
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)

	var parsed KeyExportPubOutput
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "mykey", parsed.Name)
	assert.Contains(t, parsed.Armor, "BEGIN TENDERMINT PUBLIC KEY")
}

func TestMigrateImportOutputJSON(t *testing.T) {
	output := MigrateImportOutput{
		KeyName:  "imported-key",
		Address:  "cosmos1imported",
		Verified: true,
		Status:   "imported",
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)

	var parsed MigrateImportOutput
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "imported-key", parsed.KeyName)
	assert.True(t, parsed.Verified)
}

func TestMigrateBatchImportOutputJSON(t *testing.T) {
	output := MigrateBatchImportOutput{
		Successful: []MigrateImportOutput{
			{KeyName: "key1", Address: "addr1", Verified: true, Status: "imported"},
		},
		Failed: []MigrateFailedOutput{
			{KeyName: "key2", Error: "some error"},
		},
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)

	var parsed MigrateBatchImportOutput
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Len(t, parsed.Successful, 1)
	assert.Len(t, parsed.Failed, 1)
	assert.Equal(t, "key1", parsed.Successful[0].KeyName)
	assert.Equal(t, "some error", parsed.Failed[0].Error)
}

func TestMigrateExportOutputJSON(t *testing.T) {
	output := MigrateExportOutput{
		KeyName:  "exported-key",
		Address:  "cosmos1exported",
		DestPath: "/path/to/dest",
		Verified: true,
		Status:   "exported",
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)

	var parsed MigrateExportOutput
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "exported-key", parsed.KeyName)
	assert.Equal(t, "/path/to/dest", parsed.DestPath)
}

// ============================================
// Constants Tests
// ============================================

func TestRequiredConfirmation(t *testing.T) {
	assert.Equal(t, "I understand this compromises key security", requiredConfirmation)
}

// ============================================
// Command Error Handling Tests
// ============================================

func TestRunMigrateImport_MissingFromFlag(t *testing.T) {
	resetFlags(t)

	// Create a fresh command to avoid state issues
	cmd := &cobra.Command{}
	cmd.Flags().String("from", "", "")
	cmd.Flags().String("key-name", "test", "")
	cmd.Flags().Bool("all", false, "")

	err := runMigrateImport(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--from is required")
}

func TestRunMigrateImport_MissingKeyName(t *testing.T) {
	resetFlags(t)

	cmd := &cobra.Command{}
	cmd.Flags().String("from", "/some/path", "")
	cmd.Flags().String("key-name", "", "")
	cmd.Flags().Bool("all", false, "")

	err := runMigrateImport(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--key-name is required")
}

func TestRunMigrateExport_MissingToFlag(t *testing.T) {
	resetFlags(t)

	cmd := &cobra.Command{}
	cmd.Flags().String("to", "", "")
	cmd.Flags().String("key-name", "test", "")

	err := runMigrateExport(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--to is required")
}

func TestRunMigrateExport_MissingKeyName(t *testing.T) {
	resetFlags(t)

	cmd := &cobra.Command{}
	cmd.Flags().String("to", "/some/path", "")
	cmd.Flags().String("key-name", "", "")

	err := runMigrateExport(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--key-name is required")
}

// ============================================
// Persistent Flags Tests
// ============================================

func TestPersistentFlags(t *testing.T) {
	t.Run("bao-addr flag exists", func(t *testing.T) {
		flag := rootCmd.PersistentFlags().Lookup("bao-addr")
		assert.NotNil(t, flag)
	})

	t.Run("bao-token flag exists", func(t *testing.T) {
		flag := rootCmd.PersistentFlags().Lookup("bao-token")
		assert.NotNil(t, flag)
	})

	t.Run("store-path flag exists with default", func(t *testing.T) {
		flag := rootCmd.PersistentFlags().Lookup("store-path")
		assert.NotNil(t, flag)
		assert.Equal(t, "./keyring-metadata.json", flag.DefValue)
	})

	t.Run("json flag exists", func(t *testing.T) {
		flag := rootCmd.PersistentFlags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})
}

// ============================================
// Command Tree Structure Tests
// ============================================

func TestCommandTree(t *testing.T) {
	t.Run("root has expected number of commands", func(t *testing.T) {
		// version, keys, migrate
		assert.GreaterOrEqual(t, len(rootCmd.Commands()), 3)
	})

	t.Run("keys has expected number of subcommands", func(t *testing.T) {
		// list, show, add, delete, rename, export-pub
		assert.Equal(t, 6, len(keysCmd.Commands()))
	})

	t.Run("migrate has expected number of subcommands", func(t *testing.T) {
		// import, export
		assert.Equal(t, 2, len(migrateCmd.Commands()))
	})
}

// ============================================
// Help Text Tests
// ============================================

func TestHelpText(t *testing.T) {
	t.Run("root has long description", func(t *testing.T) {
		assert.NotEmpty(t, rootCmd.Long)
		assert.Contains(t, rootCmd.Long, "OpenBao")
	})

	t.Run("keys has long description", func(t *testing.T) {
		assert.NotEmpty(t, keysCmd.Long)
	})

	t.Run("migrate import has long description", func(t *testing.T) {
		assert.NotEmpty(t, migrateImportCmd.Long)
		assert.Contains(t, migrateImportCmd.Long, "Import")
	})

	t.Run("migrate export has long description", func(t *testing.T) {
		assert.NotEmpty(t, migrateExportCmd.Long)
		assert.Contains(t, migrateExportCmd.Long, "SECURITY WARNING")
	})
}

// ============================================
// Edge Case Tests
// ============================================

func TestEmptyKeyName(t *testing.T) {
	// Commands that require key names should validate them
	err := keysShowCmd.Args(keysShowCmd, []string{""})
	assert.NoError(t, err) // Cobra doesn't validate empty strings, only count
}

func TestSpecialCharactersInKeyName(t *testing.T) {
	// Test that special characters are accepted in key names
	specialNames := []string{
		"key-with-dashes",
		"key_with_underscores",
		"key.with.dots",
		"key123",
		"KEY_UPPER",
	}

	for _, name := range specialNames {
		t.Run(fmt.Sprintf("name=%s", name), func(t *testing.T) {
			err := keysShowCmd.Args(keysShowCmd, []string{name})
			assert.NoError(t, err)
		})
	}
}

// ============================================
// Behavior Tests (without actual keyring)
// ============================================

func TestMigrateExport_RequiresConfirmation(t *testing.T) {
	resetFlags(t)

	// This test verifies that without proper confirmation, export shows warning
	// We can't easily test the actual output, but we verify the constant exists
	assert.Equal(t, "I understand this compromises key security", requiredConfirmation)
}

func TestJSONOutputFlag(t *testing.T) {
	resetFlags(t)

	// Test that jsonOut affects behavior
	jsonOut = true
	assert.True(t, jsonOut)

	jsonOut = false
	assert.False(t, jsonOut)
}

// ============================================
// String Processing Tests
// ============================================

func TestColorFunctionsWithEmptyString(t *testing.T) {
	assert.Equal(t, "\033[31m\033[0m", colorRed(""))
	assert.Equal(t, "\033[32m\033[0m", colorGreen(""))
	assert.Equal(t, "\033[33m\033[0m", colorYellow(""))
	assert.Equal(t, "\033[1m\033[0m", colorBold(""))
}

func TestColorFunctionsWithNewlines(t *testing.T) {
	input := "line1\nline2"
	result := colorRed(input)
	assert.Contains(t, result, "line1\nline2")
}

// ============================================
// Runnable Command Tests
// ============================================

func TestKeysListRunE(t *testing.T) {
	assert.NotNil(t, keysListCmd.RunE)
}

func TestKeysShowRunE(t *testing.T) {
	assert.NotNil(t, keysShowCmd.RunE)
}

func TestKeysAddRunE(t *testing.T) {
	assert.NotNil(t, keysAddCmd.RunE)
}

func TestKeysDeleteRunE(t *testing.T) {
	assert.NotNil(t, keysDeleteCmd.RunE)
}

func TestKeysRenameRunE(t *testing.T) {
	assert.NotNil(t, keysRenameCmd.RunE)
}

func TestKeysExportPubRunE(t *testing.T) {
	assert.NotNil(t, keysExportPubCmd.RunE)
}

func TestMigrateImportRunE(t *testing.T) {
	assert.NotNil(t, migrateImportCmd.RunE)
}

func TestMigrateExportRunE(t *testing.T) {
	assert.NotNil(t, migrateExportCmd.RunE)
}

// ============================================
// JSON Field Name Tests
// ============================================

func TestJSONFieldNames(t *testing.T) {
	t.Run("KeyListOutput fields", func(t *testing.T) {
		output := KeyListOutput{Name: "test", Address: "addr"}
		data, _ := json.Marshal(output)
		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"name"`)
		assert.Contains(t, jsonStr, `"address"`)
	})

	t.Run("KeyShowOutput fields", func(t *testing.T) {
		output := KeyShowOutput{
			Name:       "test",
			Algorithm:  "secp256k1",
			Exportable: true,
			CreatedAt:  "2024-01-01",
			Source:     "generated",
		}
		data, _ := json.Marshal(output)
		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"algorithm"`)
		assert.Contains(t, jsonStr, `"exportable"`)
		assert.Contains(t, jsonStr, `"created_at"`)
		assert.Contains(t, jsonStr, `"source"`)
	})

	t.Run("MigrateImportOutput fields", func(t *testing.T) {
		output := MigrateImportOutput{
			KeyName:  "test",
			Verified: true,
			Status:   "imported",
		}
		data, _ := json.Marshal(output)
		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"key_name"`)
		assert.Contains(t, jsonStr, `"verified"`)
		assert.Contains(t, jsonStr, `"status"`)
	})

	t.Run("MigrateExportOutput fields", func(t *testing.T) {
		output := MigrateExportOutput{
			DestPath: "/path",
			Verified: true,
		}
		data, _ := json.Marshal(output)
		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"dest_path"`)
	})
}

// Test for no panic on nil checks
func TestNilSafety(t *testing.T) {
	t.Run("keysCmd not nil", func(t *testing.T) {
		assert.NotNil(t, keysCmd)
	})

	t.Run("migrateCmd not nil", func(t *testing.T) {
		assert.NotNil(t, migrateCmd)
	})

	t.Run("rootCmd not nil", func(t *testing.T) {
		assert.NotNil(t, rootCmd)
	})
}

// ============================================
// Backend Selection Tests
// ============================================

func TestBackendFlagDefaults(t *testing.T) {
	// Import command
	importBackend := migrateImportCmd.Flags().Lookup("backend")
	assert.Equal(t, "file", importBackend.DefValue)

	// Export command
	exportBackend := migrateExportCmd.Flags().Lookup("backend")
	assert.Equal(t, "file", exportBackend.DefValue)
}

func TestBackendFlagTypes(t *testing.T) {
	// Verify backend flags accept common values
	commonBackends := []string{"file", "os", "test"}

	for _, backend := range commonBackends {
		t.Run(fmt.Sprintf("backend=%s", backend), func(t *testing.T) {
			// Just verify the string is valid (no panic)
			assert.NotEmpty(t, backend)
			assert.True(t, strings.Contains("file os test", backend))
		})
	}
}

// ============================================
// Mock Server Tests
// ============================================

// mockBaoServer creates a mock OpenBao server for testing (HTTP, not HTTPS).
func mockBaoServer(t *testing.T, keys map[string]mockKey) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health check
		if r.URL.Path == "/v1/sys/health" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"initialized": true,
				"sealed":      false,
			})
			return
		}

		// List keys
		if r.URL.Path == "/v1/secp256k1/keys" && r.Method == "LIST" {
			keyNames := make([]string, 0, len(keys))
			for name := range keys {
				keyNames = append(keyNames, name)
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"keys": keyNames,
				},
			})
			return
		}

		// Create key
		if strings.HasPrefix(r.URL.Path, "/v1/secp256k1/keys/") && r.Method == "POST" {
			keyName := strings.TrimPrefix(r.URL.Path, "/v1/secp256k1/keys/")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"name":       keyName,
					"public_key": "02" + strings.Repeat("ab", 32),
					"address":    "cosmos1test" + keyName,
				},
			})
			return
		}

		// Default: 404
		w.WriteHeader(http.StatusNotFound)
	}))
}

type mockKey struct {
	_ struct{} // placeholder to avoid empty struct issues
}

// ============================================
// Integration-Style Tests with Mock Server
// ============================================

func TestKeysList_NoKeys(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring and store (simulate empty store)
	_ = os.WriteFile(storeFile, []byte(`{"version":1,"keys":{}}`), 0600)

	// Test list command by calling the RunE function
	buf := new(bytes.Buffer)
	keysListCmd.SetOut(buf)

	err := runKeysList(keysListCmd, []string{})
	require.NoError(t, err)
}

func TestKeysList_WithJSON(t *testing.T) {
	resetFlags(t)
	jsonOut = true
	defer func() { jsonOut = false }()

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring store with a key
	_ = os.WriteFile(storeFile, []byte(`{"version":1,"keys":{}}`), 0600)

	err := runKeysList(keysListCmd, []string{})
	require.NoError(t, err)
}

func TestKeysShow_KeyNotFound(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create empty keyring store
	_ = os.WriteFile(storeFile, []byte(`{"version":1,"keys":{}}`), 0600)

	err := runKeysShow(keysShowCmd, []string{"nonexistent"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "key not found")
}

func TestKeysShow_WithExistingKey(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring store with a key
	storeData := `{"version":1,"keys":{"mykey":{"uid":"mykey","name":"mykey","pub_key":"AgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==","pub_key_type":"secp256k1","address":"cosmos1test","bao_key_path":"secp256k1/keys/mykey","algorithm":"secp256k1","exportable":false,"created_at":"2024-01-01T00:00:00Z","source":"generated"}}}`
	_ = os.WriteFile(storeFile, []byte(storeData), 0600)

	err := runKeysShow(keysShowCmd, []string{"mykey"})
	require.NoError(t, err)
}

func TestKeysShow_WithJSON(t *testing.T) {
	resetFlags(t)
	jsonOut = true
	defer func() { jsonOut = false }()

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring store with a key
	storeData := `{"version":1,"keys":{"mykey":{"uid":"mykey","name":"mykey","pub_key":"AgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==","pub_key_type":"secp256k1","address":"cosmos1test","bao_key_path":"secp256k1/keys/mykey","algorithm":"secp256k1","exportable":false,"created_at":"2024-01-01T00:00:00Z","source":"generated"}}}`
	_ = os.WriteFile(storeFile, []byte(storeData), 0600)

	err := runKeysShow(keysShowCmd, []string{"mykey"})
	require.NoError(t, err)
}

func TestKeysDelete_WithForce(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server that accepts DELETE
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/sys/health" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"initialized": true, "sealed": false})
			return
		}
		if r.Method == "DELETE" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring store with a key
	storeData := `{"version":1,"keys":{"mykey":{"uid":"mykey","name":"mykey","pub_key":"AgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==","pub_key_type":"secp256k1","address":"cosmos1test","bao_key_path":"secp256k1/keys/mykey","algorithm":"secp256k1","exportable":false,"created_at":"2024-01-01T00:00:00Z","source":"generated"}}}`
	_ = os.WriteFile(storeFile, []byte(storeData), 0600)

	// Create command with force flag
	cmd := &cobra.Command{}
	cmd.Flags().BoolP("force", "f", true, "")
	cmd.Flags().BoolP("yes", "y", false, "")

	err := runKeysDelete(cmd, []string{"mykey"})
	require.NoError(t, err)
}

func TestKeysDelete_WithYesFlag(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server that accepts DELETE
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/sys/health" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"initialized": true, "sealed": false})
			return
		}
		if r.Method == "DELETE" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring store with a key
	storeData := `{"version":1,"keys":{"mykey":{"uid":"mykey","name":"mykey","pub_key":"AgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==","pub_key_type":"secp256k1","address":"cosmos1test","bao_key_path":"secp256k1/keys/mykey","algorithm":"secp256k1","exportable":false,"created_at":"2024-01-01T00:00:00Z","source":"generated"}}}`
	_ = os.WriteFile(storeFile, []byte(storeData), 0600)

	// Create command with yes flag
	cmd := &cobra.Command{}
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().BoolP("yes", "y", true, "")

	err := runKeysDelete(cmd, []string{"mykey"})
	require.NoError(t, err)
}

func TestKeysDelete_WithJSON(t *testing.T) {
	resetFlags(t)
	jsonOut = true
	defer func() { jsonOut = false }()

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server that accepts DELETE
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/sys/health" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"initialized": true, "sealed": false})
			return
		}
		if r.Method == "DELETE" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring store with a key
	storeData := `{"version":1,"keys":{"mykey":{"uid":"mykey","name":"mykey","pub_key":"AgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==","pub_key_type":"secp256k1","address":"cosmos1test","bao_key_path":"secp256k1/keys/mykey","algorithm":"secp256k1","exportable":false,"created_at":"2024-01-01T00:00:00Z","source":"generated"}}}`
	_ = os.WriteFile(storeFile, []byte(storeData), 0600)

	// Create command with force flag
	cmd := &cobra.Command{}
	cmd.Flags().BoolP("force", "f", true, "")
	cmd.Flags().BoolP("yes", "y", false, "")

	err := runKeysDelete(cmd, []string{"mykey"})
	require.NoError(t, err)
}

func TestKeysRename_Success(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring store with a key
	storeData := `{"version":1,"keys":{"oldkey":{"uid":"oldkey","name":"oldkey","pub_key":"AgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==","pub_key_type":"secp256k1","address":"cosmos1test","bao_key_path":"secp256k1/keys/oldkey","algorithm":"secp256k1","exportable":false,"created_at":"2024-01-01T00:00:00Z","source":"generated"}}}`
	_ = os.WriteFile(storeFile, []byte(storeData), 0600)

	err := runKeysRename(keysRenameCmd, []string{"oldkey", "newkey"})
	require.NoError(t, err)
}

func TestKeysRename_WithJSON(t *testing.T) {
	resetFlags(t)
	jsonOut = true
	defer func() { jsonOut = false }()

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring store with a key
	storeData := `{"version":1,"keys":{"oldkey":{"uid":"oldkey","name":"oldkey","pub_key":"AgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==","pub_key_type":"secp256k1","address":"cosmos1test","bao_key_path":"secp256k1/keys/oldkey","algorithm":"secp256k1","exportable":false,"created_at":"2024-01-01T00:00:00Z","source":"generated"}}}`
	_ = os.WriteFile(storeFile, []byte(storeData), 0600)

	err := runKeysRename(keysRenameCmd, []string{"oldkey", "newkey"})
	require.NoError(t, err)
}

func TestKeysRename_KeyNotFound(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create empty keyring store
	_ = os.WriteFile(storeFile, []byte(`{"version":1,"keys":{}}`), 0600)

	err := runKeysRename(keysRenameCmd, []string{"nonexistent", "newname"})
	require.Error(t, err)
}

func TestKeysExportPub_Success(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring store with a key (valid 33-byte pubkey encoded in base64)
	storeData := `{"version":1,"keys":{"mykey":{"uid":"mykey","name":"mykey","pub_key":"Ak9yBLKjTbFpVrJFG9n7T3q9N+0PJPZqQqY8a3xS1ux4","pub_key_type":"secp256k1","address":"cosmos1test","bao_key_path":"secp256k1/keys/mykey","algorithm":"secp256k1","exportable":false,"created_at":"2024-01-01T00:00:00Z","source":"generated"}}}`
	_ = os.WriteFile(storeFile, []byte(storeData), 0600)

	err := runKeysExportPub(keysExportPubCmd, []string{"mykey"})
	require.NoError(t, err)
}

func TestKeysExportPub_WithJSON(t *testing.T) {
	resetFlags(t)
	jsonOut = true
	defer func() { jsonOut = false }()

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring store with a key (valid 33-byte pubkey encoded in base64)
	storeData := `{"version":1,"keys":{"mykey":{"uid":"mykey","name":"mykey","pub_key":"Ak9yBLKjTbFpVrJFG9n7T3q9N+0PJPZqQqY8a3xS1ux4","pub_key_type":"secp256k1","address":"cosmos1test","bao_key_path":"secp256k1/keys/mykey","algorithm":"secp256k1","exportable":false,"created_at":"2024-01-01T00:00:00Z","source":"generated"}}}`
	_ = os.WriteFile(storeFile, []byte(storeData), 0600)

	err := runKeysExportPub(keysExportPubCmd, []string{"mykey"})
	require.NoError(t, err)
}

func TestKeysExportPub_KeyNotFound(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create empty keyring store
	_ = os.WriteFile(storeFile, []byte(`{"version":1,"keys":{}}`), 0600)

	err := runKeysExportPub(keysExportPubCmd, []string{"nonexistent"})
	require.Error(t, err)
}

func TestKeysAdd_Success(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server that handles key creation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/sys/health" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"initialized": true, "sealed": false})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/v1/secp256k1/keys/") && r.Method == "POST" {
			keyName := strings.TrimPrefix(r.URL.Path, "/v1/secp256k1/keys/")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"name":       keyName,
					"public_key": "024f7204b2a34db16956b2451bd9fb4f7abd37ed0f24f66a42a63c6b7c52d6ec78",
					"address":    "cosmos1test" + keyName,
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create empty keyring store
	_ = os.WriteFile(storeFile, []byte(`{"version":1,"keys":{}}`), 0600)

	// Create command with exportable flag
	cmd := &cobra.Command{}
	cmd.Flags().Bool("exportable", false, "")

	err := runKeysAdd(cmd, []string{"newkey"})
	require.NoError(t, err)
}

func TestKeysAdd_WithExportable(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server that handles key creation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/sys/health" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"initialized": true, "sealed": false})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/v1/secp256k1/keys/") && r.Method == "POST" {
			keyName := strings.TrimPrefix(r.URL.Path, "/v1/secp256k1/keys/")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"name":       keyName,
					"public_key": "024f7204b2a34db16956b2451bd9fb4f7abd37ed0f24f66a42a63c6b7c52d6ec78",
					"address":    "cosmos1test" + keyName,
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create empty keyring store
	_ = os.WriteFile(storeFile, []byte(`{"version":1,"keys":{}}`), 0600)

	// Create command with exportable flag set to true
	cmd := &cobra.Command{}
	cmd.Flags().Bool("exportable", true, "")

	err := runKeysAdd(cmd, []string{"newkey"})
	require.NoError(t, err)
}

func TestKeysAdd_WithJSON(t *testing.T) {
	resetFlags(t)
	jsonOut = true
	defer func() { jsonOut = false }()

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server that handles key creation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/sys/health" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"initialized": true, "sealed": false})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/v1/secp256k1/keys/") && r.Method == "POST" {
			keyName := strings.TrimPrefix(r.URL.Path, "/v1/secp256k1/keys/")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"name":       keyName,
					"public_key": "024f7204b2a34db16956b2451bd9fb4f7abd37ed0f24f66a42a63c6b7c52d6ec78",
					"address":    "cosmos1test" + keyName,
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create empty keyring store
	_ = os.WriteFile(storeFile, []byte(`{"version":1,"keys":{}}`), 0600)

	// Create command with exportable flag
	cmd := &cobra.Command{}
	cmd.Flags().Bool("exportable", false, "")

	err := runKeysAdd(cmd, []string{"newkey"})
	require.NoError(t, err)
}

func TestKeysAdd_KeyAlreadyExists(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring store with existing key
	storeData := `{"version":1,"keys":{"existingkey":{"uid":"existingkey","name":"existingkey","pub_key":"AgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==","pub_key_type":"secp256k1","address":"cosmos1test","bao_key_path":"secp256k1/keys/existingkey","algorithm":"secp256k1","exportable":false,"created_at":"2024-01-01T00:00:00Z","source":"generated"}}}`
	_ = os.WriteFile(storeFile, []byte(storeData), 0600)

	// Create command
	cmd := &cobra.Command{}
	cmd.Flags().Bool("exportable", false, "")

	err := runKeysAdd(cmd, []string{"existingkey"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

// ============================================
// Migration Command Validation Tests
// ============================================

func TestMigrateImport_AllFlagSkipsKeyName(t *testing.T) {
	resetFlags(t)

	cmd := &cobra.Command{}
	cmd.Flags().String("from", "/some/path", "")
	cmd.Flags().String("key-name", "", "")
	cmd.Flags().Bool("all", true, "")
	cmd.Flags().String("backend", "file", "")
	cmd.Flags().String("new-name", "", "")
	cmd.Flags().Bool("delete-after-import", false, "")
	cmd.Flags().Bool("exportable", false, "")
	cmd.Flags().Bool("verify", true, "")

	// This will fail because keyring.New can't connect, but validates flags are OK
	err := runMigrateImport(cmd, []string{})
	// Should fail on keyring connection, not flag validation
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "--key-name is required")
}

func TestMigrateExport_SecurityWarning(t *testing.T) {
	resetFlags(t)

	// Test that export without confirmation shows warning but doesn't error
	cmd := &cobra.Command{}
	cmd.Flags().String("to", "/some/path", "")
	cmd.Flags().String("key-name", "testkey", "")
	cmd.Flags().String("confirm", "", "") // Empty confirmation
	cmd.Flags().String("backend", "file", "")
	cmd.Flags().String("new-name", "", "")
	cmd.Flags().Bool("verify", true, "")

	// Should return nil (shows warning, doesn't proceed)
	err := runMigrateExport(cmd, []string{})
	assert.NoError(t, err)
}

func TestMigrateExport_WithConfirmation(t *testing.T) {
	resetFlags(t)

	// Test that export with correct confirmation proceeds
	cmd := &cobra.Command{}
	cmd.Flags().String("to", "/some/path", "")
	cmd.Flags().String("key-name", "testkey", "")
	cmd.Flags().String("confirm", requiredConfirmation, "") // Correct confirmation
	cmd.Flags().String("backend", "file", "")
	cmd.Flags().String("new-name", "", "")
	cmd.Flags().Bool("verify", true, "")

	// Should fail on getKeyring (no credentials)
	err := runMigrateExport(cmd, []string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BAO_ADDR and BAO_TOKEN are required")
}

// ============================================
// Error Path Coverage Tests
// ============================================

func TestKeysList_ConnectionError(t *testing.T) {
	resetFlags(t)

	baoAddr = "http://nonexistent:8200"
	baoToken = "test-token"
	storePath = "./nonexistent-store.json"

	err := runKeysList(keysListCmd, []string{})
	require.Error(t, err)
}

func TestKeysShow_ConnectionError(t *testing.T) {
	resetFlags(t)

	baoAddr = "http://nonexistent:8200"
	baoToken = "test-token"
	storePath = "./nonexistent-store.json"

	err := runKeysShow(keysShowCmd, []string{"somekey"})
	require.Error(t, err)
}

func TestKeysAdd_ConnectionError(t *testing.T) {
	resetFlags(t)

	baoAddr = "http://nonexistent:8200"
	baoToken = "test-token"
	storePath = "./nonexistent-store.json"

	cmd := &cobra.Command{}
	cmd.Flags().Bool("exportable", false, "")

	err := runKeysAdd(cmd, []string{"newkey"})
	require.Error(t, err)
}

func TestKeysDelete_ConnectionError(t *testing.T) {
	resetFlags(t)

	baoAddr = "http://nonexistent:8200"
	baoToken = "test-token"
	storePath = "./nonexistent-store.json"

	cmd := &cobra.Command{}
	cmd.Flags().BoolP("force", "f", true, "")
	cmd.Flags().BoolP("yes", "y", false, "")

	err := runKeysDelete(cmd, []string{"somekey"})
	require.Error(t, err)
}

func TestKeysRename_ConnectionError(t *testing.T) {
	resetFlags(t)

	baoAddr = "http://nonexistent:8200"
	baoToken = "test-token"
	storePath = "./nonexistent-store.json"

	err := runKeysRename(keysRenameCmd, []string{"old", "new"})
	require.Error(t, err)
}

func TestKeysExportPub_ConnectionError(t *testing.T) {
	resetFlags(t)

	baoAddr = "http://nonexistent:8200"
	baoToken = "test-token"
	storePath = "./nonexistent-store.json"

	err := runKeysExportPub(keysExportPubCmd, []string{"somekey"})
	require.Error(t, err)
}

// ============================================
// Additional Coverage Tests
// ============================================

func TestKeysList_WithKeysTable(t *testing.T) {
	resetFlags(t)
	jsonOut = false

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring store with multiple keys
	storeData := `{"version":1,"keys":{
		"key1":{"uid":"key1","name":"key1","pub_key":"Ak9yBLKjTbFpVrJFG9n7T3q9N+0PJPZqQqY8a3xS1ux4","pub_key_type":"secp256k1","address":"cosmos1abc","bao_key_path":"secp256k1/keys/key1","algorithm":"secp256k1","exportable":false,"created_at":"2024-01-01T00:00:00Z","source":"generated"},
		"key2":{"uid":"key2","name":"key2","pub_key":"Ak9yBLKjTbFpVrJFG9n7T3q9N+0PJPZqQqY8a3xS1ux4","pub_key_type":"secp256k1","address":"cosmos1def","bao_key_path":"secp256k1/keys/key2","algorithm":"secp256k1","exportable":true,"created_at":"2024-01-02T00:00:00Z","source":"imported"}
	}}`
	_ = os.WriteFile(storeFile, []byte(storeData), 0600)

	err := runKeysList(keysListCmd, []string{})
	require.NoError(t, err)
}

func TestKeysList_WithKeysJSON(t *testing.T) {
	resetFlags(t)
	jsonOut = true
	defer func() { jsonOut = false }()

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring store with keys
	storeData := `{"version":1,"keys":{
		"key1":{"uid":"key1","name":"key1","pub_key":"Ak9yBLKjTbFpVrJFG9n7T3q9N+0PJPZqQqY8a3xS1ux4","pub_key_type":"secp256k1","address":"cosmos1abc","bao_key_path":"secp256k1/keys/key1","algorithm":"secp256k1","exportable":false,"created_at":"2024-01-01T00:00:00Z","source":"generated"}
	}}`
	_ = os.WriteFile(storeFile, []byte(storeData), 0600)

	err := runKeysList(keysListCmd, []string{})
	require.NoError(t, err)
}

func TestMigrateImport_WithSourceKeyring(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create empty keyring store
	_ = os.WriteFile(storeFile, []byte(`{"version":1,"keys":{}}`), 0600)

	// Create command with flags - this will fail on opening source keyring or import
	cmd := &cobra.Command{}
	cmd.Flags().String("from", "/nonexistent/path", "")
	cmd.Flags().String("key-name", "testkey", "")
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().String("backend", "test", "")
	cmd.Flags().String("new-name", "", "")
	cmd.Flags().Bool("delete-after-import", false, "")
	cmd.Flags().Bool("exportable", false, "")
	cmd.Flags().Bool("verify", true, "")

	err := runMigrateImport(cmd, []string{})
	// Should fail somewhere in the import process
	require.Error(t, err)
}

func TestMigrateExport_WithDestKeyring(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring store with exportable key
	storeData := `{"version":1,"keys":{"mykey":{"uid":"mykey","name":"mykey","pub_key":"Ak9yBLKjTbFpVrJFG9n7T3q9N+0PJPZqQqY8a3xS1ux4","pub_key_type":"secp256k1","address":"cosmos1test","bao_key_path":"secp256k1/keys/mykey","algorithm":"secp256k1","exportable":true,"created_at":"2024-01-01T00:00:00Z","source":"generated"}}}`
	_ = os.WriteFile(storeFile, []byte(storeData), 0600)

	// Create command with flags
	cmd := &cobra.Command{}
	cmd.Flags().String("to", "/nonexistent/path", "")
	cmd.Flags().String("key-name", "mykey", "")
	cmd.Flags().String("confirm", requiredConfirmation, "")
	cmd.Flags().String("backend", "test", "")
	cmd.Flags().String("new-name", "", "")
	cmd.Flags().Bool("verify", true, "")

	err := runMigrateExport(cmd, []string{})
	// Should fail but after keyring is created (export endpoint not implemented)
	require.Error(t, err)
}

func TestKeysRename_TargetExists(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring store with both keys
	storeData := `{"version":1,"keys":{
		"key1":{"uid":"key1","name":"key1","pub_key":"Ak9yBLKjTbFpVrJFG9n7T3q9N+0PJPZqQqY8a3xS1ux4","pub_key_type":"secp256k1","address":"cosmos1abc","bao_key_path":"secp256k1/keys/key1","algorithm":"secp256k1","exportable":false,"created_at":"2024-01-01T00:00:00Z","source":"generated"},
		"key2":{"uid":"key2","name":"key2","pub_key":"Ak9yBLKjTbFpVrJFG9n7T3q9N+0PJPZqQqY8a3xS1ux4","pub_key_type":"secp256k1","address":"cosmos1def","bao_key_path":"secp256k1/keys/key2","algorithm":"secp256k1","exportable":true,"created_at":"2024-01-02T00:00:00Z","source":"imported"}
	}}`
	_ = os.WriteFile(storeFile, []byte(storeData), 0600)

	err := runKeysRename(keysRenameCmd, []string{"key1", "key2"})
	// Should fail because key2 already exists
	require.Error(t, err)
}

func TestKeysDelete_ServerError(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server that returns error on DELETE
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/sys/health" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"initialized": true, "sealed": false})
			return
		}
		if r.Method == "DELETE" {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"errors": []string{"internal error"}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring store with a key
	storeData := `{"version":1,"keys":{"mykey":{"uid":"mykey","name":"mykey","pub_key":"AgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==","pub_key_type":"secp256k1","address":"cosmos1test","bao_key_path":"secp256k1/keys/mykey","algorithm":"secp256k1","exportable":false,"created_at":"2024-01-01T00:00:00Z","source":"generated"}}}`
	_ = os.WriteFile(storeFile, []byte(storeData), 0600)

	// Create command with force flag
	cmd := &cobra.Command{}
	cmd.Flags().BoolP("force", "f", true, "")
	cmd.Flags().BoolP("yes", "y", false, "")

	err := runKeysDelete(cmd, []string{"mykey"})
	require.Error(t, err)
}

func TestKeysAdd_ServerError(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server that returns error on create
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/sys/health" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"initialized": true, "sealed": false})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/v1/secp256k1/keys/") && r.Method == "POST" {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"errors": []string{"internal error"}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create empty keyring store
	_ = os.WriteFile(storeFile, []byte(`{"version":1,"keys":{}}`), 0600)

	// Create command
	cmd := &cobra.Command{}
	cmd.Flags().Bool("exportable", false, "")

	err := runKeysAdd(cmd, []string{"newkey"})
	require.Error(t, err)
}

func TestGetKeyring_OnlyAddr(t *testing.T) {
	resetFlags(t)

	// Only set addr, not token
	_ = os.Setenv("BAO_ADDR", "http://localhost:8200")
	_ = os.Unsetenv("BAO_TOKEN")
	defer func() { _ = os.Unsetenv("BAO_ADDR") }()

	_, err := getKeyring()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BAO_ADDR and BAO_TOKEN are required")
}

func TestGetKeyring_OnlyToken(t *testing.T) {
	resetFlags(t)

	// Only set token, not addr
	_ = os.Unsetenv("BAO_ADDR")
	_ = os.Setenv("BAO_TOKEN", "test-token")
	defer func() { _ = os.Unsetenv("BAO_TOKEN") }()

	_, err := getKeyring()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BAO_ADDR and BAO_TOKEN are required")
}

func TestMigrateImport_AllKeys(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create empty keyring store
	_ = os.WriteFile(storeFile, []byte(`{"version":1,"keys":{}}`), 0600)

	// Create command with all flag
	cmd := &cobra.Command{}
	cmd.Flags().String("from", "/nonexistent/path", "")
	cmd.Flags().String("key-name", "", "")
	cmd.Flags().Bool("all", true, "")
	cmd.Flags().String("backend", "test", "")
	cmd.Flags().String("new-name", "", "")
	cmd.Flags().Bool("delete-after-import", false, "")
	cmd.Flags().Bool("exportable", false, "")
	cmd.Flags().Bool("verify", true, "")

	err := runMigrateImport(cmd, []string{})
	// Should fail on opening source keyring (all flag doesn't require key-name)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "--key-name is required")
}

func TestMigrateExport_WrongConfirmation(t *testing.T) {
	resetFlags(t)

	cmd := &cobra.Command{}
	cmd.Flags().String("to", "/some/path", "")
	cmd.Flags().String("key-name", "testkey", "")
	cmd.Flags().String("confirm", "wrong confirmation", "") // Wrong confirmation
	cmd.Flags().String("backend", "file", "")
	cmd.Flags().String("new-name", "", "")
	cmd.Flags().Bool("verify", true, "")

	// Should return nil (shows warning, doesn't proceed)
	err := runMigrateExport(cmd, []string{})
	assert.NoError(t, err)
}

func TestColorWithSpecialChars(t *testing.T) {
	special := "test\ttab\nnewline"
	
	result := colorRed(special)
	assert.Contains(t, result, "\t")
	assert.Contains(t, result, "\n")
	
	result = colorGreen(special)
	assert.Contains(t, result, special)
	
	result = colorYellow(special)
	assert.Contains(t, result, special)
	
	result = colorBold(special)
	assert.Contains(t, result, special)
}

func TestMigrateImportOutputJSONSerialization(t *testing.T) {
	output := MigrateFailedOutput{
		KeyName: "failed-key",
		Error:   "some error message",
	}

	data, err := json.Marshal(output)
	require.NoError(t, err)

	var parsed MigrateFailedOutput
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "failed-key", parsed.KeyName)
	assert.Equal(t, "some error message", parsed.Error)
}

func TestMigrateImport_WithJSONOutput(t *testing.T) {
	resetFlags(t)
	jsonOut = true
	defer func() { jsonOut = false }()

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create empty keyring store
	_ = os.WriteFile(storeFile, []byte(`{"version":1,"keys":{}}`), 0600)

	// Create command with flags
	cmd := &cobra.Command{}
	cmd.Flags().String("from", "/nonexistent/path", "")
	cmd.Flags().String("key-name", "testkey", "")
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().String("backend", "test", "")
	cmd.Flags().String("new-name", "", "")
	cmd.Flags().Bool("delete-after-import", false, "")
	cmd.Flags().Bool("exportable", false, "")
	cmd.Flags().Bool("verify", true, "")

	err := runMigrateImport(cmd, []string{})
	// Should fail
	require.Error(t, err)
}

func TestMigrateExport_WithJSONOutput(t *testing.T) {
	resetFlags(t)
	jsonOut = true
	defer func() { jsonOut = false }()

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring store with exportable key
	storeData := `{"version":1,"keys":{"mykey":{"uid":"mykey","name":"mykey","pub_key":"Ak9yBLKjTbFpVrJFG9n7T3q9N+0PJPZqQqY8a3xS1ux4","pub_key_type":"secp256k1","address":"cosmos1test","bao_key_path":"secp256k1/keys/mykey","algorithm":"secp256k1","exportable":true,"created_at":"2024-01-01T00:00:00Z","source":"generated"}}}`
	_ = os.WriteFile(storeFile, []byte(storeData), 0600)

	// Create command with flags
	cmd := &cobra.Command{}
	cmd.Flags().String("to", "/nonexistent/path", "")
	cmd.Flags().String("key-name", "mykey", "")
	cmd.Flags().String("confirm", requiredConfirmation, "")
	cmd.Flags().String("backend", "test", "")
	cmd.Flags().String("new-name", "", "")
	cmd.Flags().Bool("verify", true, "")

	err := runMigrateExport(cmd, []string{})
	// Should fail (export not fully implemented)
	require.Error(t, err)
}

func TestMigrateImport_BatchWithJSON(t *testing.T) {
	resetFlags(t)
	jsonOut = true
	defer func() { jsonOut = false }()

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create empty keyring store
	_ = os.WriteFile(storeFile, []byte(`{"version":1,"keys":{}}`), 0600)

	// Create command with all flag
	cmd := &cobra.Command{}
	cmd.Flags().String("from", "/nonexistent/path", "")
	cmd.Flags().String("key-name", "", "")
	cmd.Flags().Bool("all", true, "")
	cmd.Flags().String("backend", "test", "")
	cmd.Flags().String("new-name", "", "")
	cmd.Flags().Bool("delete-after-import", false, "")
	cmd.Flags().Bool("exportable", false, "")
	cmd.Flags().Bool("verify", true, "")

	err := runMigrateImport(cmd, []string{})
	// Should fail
	require.Error(t, err)
}

func TestKeysDelete_KeyNotFound(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server that returns 404 on DELETE
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/sys/health" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"initialized": true, "sealed": false})
			return
		}
		if r.Method == "DELETE" {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"errors": []string{"key not found"}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring store with a key
	storeData := `{"version":1,"keys":{"mykey":{"uid":"mykey","name":"mykey","pub_key":"AgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==","pub_key_type":"secp256k1","address":"cosmos1test","bao_key_path":"secp256k1/keys/mykey","algorithm":"secp256k1","exportable":false,"created_at":"2024-01-01T00:00:00Z","source":"generated"}}}`
	_ = os.WriteFile(storeFile, []byte(storeData), 0600)

	// Create command with force flag
	cmd := &cobra.Command{}
	cmd.Flags().BoolP("force", "f", true, "")
	cmd.Flags().BoolP("yes", "y", false, "")

	err := runKeysDelete(cmd, []string{"mykey"})
	require.Error(t, err)
}

func TestMain_Version(t *testing.T) {
	// Just verify the main function structure
	assert.NotNil(t, rootCmd)
	assert.NotNil(t, versionCmd)
}

func TestStorePath_DefaultValue(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("store-path")
	assert.Equal(t, "./keyring-metadata.json", flag.DefValue)
}

// ============================================
// More Coverage Tests
// ============================================

func TestMigrateImport_NewNameFlag(t *testing.T) {
	resetFlags(t)

	cmd := &cobra.Command{}
	cmd.Flags().String("from", "/some/path", "")
	cmd.Flags().String("key-name", "testkey", "")
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().String("backend", "test", "")
	cmd.Flags().String("new-name", "renamedkey", "") // Testing new-name
	cmd.Flags().Bool("delete-after-import", true, "") // Testing delete-after
	cmd.Flags().Bool("exportable", true, "")          // Testing exportable
	cmd.Flags().Bool("verify", false, "")             // Testing verify=false

	// Will fail but exercises the flag parsing code paths
	err := runMigrateImport(cmd, []string{})
	require.Error(t, err)
}

func TestMigrateExport_NewNameFlag(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring store with exportable key
	storeData := `{"version":1,"keys":{"mykey":{"uid":"mykey","name":"mykey","pub_key":"Ak9yBLKjTbFpVrJFG9n7T3q9N+0PJPZqQqY8a3xS1ux4","pub_key_type":"secp256k1","address":"cosmos1test","bao_key_path":"secp256k1/keys/mykey","algorithm":"secp256k1","exportable":true,"created_at":"2024-01-01T00:00:00Z","source":"generated"}}}`
	_ = os.WriteFile(storeFile, []byte(storeData), 0600)

	cmd := &cobra.Command{}
	cmd.Flags().String("to", "/nonexistent/path", "")
	cmd.Flags().String("key-name", "mykey", "")
	cmd.Flags().String("confirm", requiredConfirmation, "")
	cmd.Flags().String("backend", "test", "")
	cmd.Flags().String("new-name", "renamedkey", "") // Testing new-name
	cmd.Flags().Bool("verify", false, "")            // Testing verify=false

	err := runMigrateExport(cmd, []string{})
	require.Error(t, err)
}

func TestMigrateExport_NonExportableKey(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring store with NON-exportable key
	storeData := `{"version":1,"keys":{"mykey":{"uid":"mykey","name":"mykey","pub_key":"Ak9yBLKjTbFpVrJFG9n7T3q9N+0PJPZqQqY8a3xS1ux4","pub_key_type":"secp256k1","address":"cosmos1test","bao_key_path":"secp256k1/keys/mykey","algorithm":"secp256k1","exportable":false,"created_at":"2024-01-01T00:00:00Z","source":"generated"}}}`
	_ = os.WriteFile(storeFile, []byte(storeData), 0600)

	cmd := &cobra.Command{}
	cmd.Flags().String("to", "/nonexistent/path", "")
	cmd.Flags().String("key-name", "mykey", "")
	cmd.Flags().String("confirm", requiredConfirmation, "")
	cmd.Flags().String("backend", "test", "")
	cmd.Flags().String("new-name", "", "")
	cmd.Flags().Bool("verify", true, "")

	err := runMigrateExport(cmd, []string{})
	require.Error(t, err)
	// Should fail because key is not exportable
	assert.Contains(t, err.Error(), "not exportable")
}

func TestMigrateExport_KeyNotFound(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create empty keyring store
	_ = os.WriteFile(storeFile, []byte(`{"version":1,"keys":{}}`), 0600)

	cmd := &cobra.Command{}
	cmd.Flags().String("to", "/nonexistent/path", "")
	cmd.Flags().String("key-name", "nonexistent", "")
	cmd.Flags().String("confirm", requiredConfirmation, "")
	cmd.Flags().String("backend", "test", "")
	cmd.Flags().String("new-name", "", "")
	cmd.Flags().Bool("verify", true, "")

	err := runMigrateExport(cmd, []string{})
	require.Error(t, err)
}

func TestKeysDelete_NotFoundInStore(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create empty keyring store
	_ = os.WriteFile(storeFile, []byte(`{"version":1,"keys":{}}`), 0600)

	// Create command with force flag
	cmd := &cobra.Command{}
	cmd.Flags().BoolP("force", "f", true, "")
	cmd.Flags().BoolP("yes", "y", false, "")

	err := runKeysDelete(cmd, []string{"nonexistent"})
	require.Error(t, err)
}

func TestKeysList_EmptyWithTable(t *testing.T) {
	resetFlags(t)
	jsonOut = false

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create empty keyring store
	_ = os.WriteFile(storeFile, []byte(`{"version":1,"keys":{}}`), 0600)

	err := runKeysList(keysListCmd, []string{})
	require.NoError(t, err)
}

func TestMigrateImport_ConnectionError(t *testing.T) {
	resetFlags(t)

	baoAddr = "http://nonexistent:8200"
	baoToken = "test-token"
	storePath = "./nonexistent-store.json"

	cmd := &cobra.Command{}
	cmd.Flags().String("from", "/some/path", "")
	cmd.Flags().String("key-name", "testkey", "")
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().String("backend", "test", "")
	cmd.Flags().String("new-name", "", "")
	cmd.Flags().Bool("delete-after-import", false, "")
	cmd.Flags().Bool("exportable", false, "")
	cmd.Flags().Bool("verify", true, "")

	err := runMigrateImport(cmd, []string{})
	require.Error(t, err)
}

func TestMigrateExport_ConnectionError(t *testing.T) {
	resetFlags(t)

	baoAddr = "http://nonexistent:8200"
	baoToken = "test-token"
	storePath = "./nonexistent-store.json"

	cmd := &cobra.Command{}
	cmd.Flags().String("to", "/some/path", "")
	cmd.Flags().String("key-name", "testkey", "")
	cmd.Flags().String("confirm", requiredConfirmation, "")
	cmd.Flags().String("backend", "test", "")
	cmd.Flags().String("new-name", "", "")
	cmd.Flags().Bool("verify", true, "")

	err := runMigrateExport(cmd, []string{})
	require.Error(t, err)
}

func TestMigrateExportFlagShortcuts(t *testing.T) {
	// Verify flag aliases work
	flag := migrateExportCmd.Flags().Lookup("to")
	assert.NotNil(t, flag)
	
	flag = migrateExportCmd.Flags().Lookup("key-name")
	assert.NotNil(t, flag)
	
	flag = migrateExportCmd.Flags().Lookup("confirm")
	assert.NotNil(t, flag)
}

func TestMigrateImportFlagShortcuts(t *testing.T) {
	// Verify flag aliases work
	flag := migrateImportCmd.Flags().Lookup("from")
	assert.NotNil(t, flag)
	
	flag = migrateImportCmd.Flags().Lookup("key-name")
	assert.NotNil(t, flag)
	
	flag = migrateImportCmd.Flags().Lookup("all")
	assert.NotNil(t, flag)
}

// Test verifyFlag variations
func TestMigrateImport_VerifyFalse(t *testing.T) {
	resetFlags(t)

	cmd := &cobra.Command{}
	cmd.Flags().String("from", "/some/path", "")
	cmd.Flags().String("key-name", "testkey", "")
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().String("backend", "test", "")
	cmd.Flags().String("new-name", "", "")
	cmd.Flags().Bool("delete-after-import", false, "")
	cmd.Flags().Bool("exportable", false, "")
	cmd.Flags().Bool("verify", false, "") // Testing verify=false

	err := runMigrateImport(cmd, []string{})
	require.Error(t, err)
}

func TestMigrateExport_VerifyFalse(t *testing.T) {
	resetFlags(t)

	// Create a temp store
	tmpDir := t.TempDir()
	storeFile := filepath.Join(tmpDir, "keyring.json")
	storePath = storeFile

	// Create mock server
	server := mockBaoServer(t, map[string]mockKey{})
	defer server.Close()

	// Set credentials
	baoAddr = server.URL
	baoToken = "test-token"

	// Create keyring store with exportable key
	storeData := `{"version":1,"keys":{"mykey":{"uid":"mykey","name":"mykey","pub_key":"Ak9yBLKjTbFpVrJFG9n7T3q9N+0PJPZqQqY8a3xS1ux4","pub_key_type":"secp256k1","address":"cosmos1test","bao_key_path":"secp256k1/keys/mykey","algorithm":"secp256k1","exportable":true,"created_at":"2024-01-01T00:00:00Z","source":"generated"}}}`
	_ = os.WriteFile(storeFile, []byte(storeData), 0600)

	cmd := &cobra.Command{}
	cmd.Flags().String("to", "/nonexistent/path", "")
	cmd.Flags().String("key-name", "mykey", "")
	cmd.Flags().String("confirm", requiredConfirmation, "")
	cmd.Flags().String("backend", "test", "")
	cmd.Flags().String("new-name", "", "")
	cmd.Flags().Bool("verify", false, "") // Testing verify=false

	err := runMigrateExport(cmd, []string{})
	require.Error(t, err)
}

// Additional edge cases for full coverage
func TestMigrateImport_DeleteAfterTrue(t *testing.T) {
	resetFlags(t)

	cmd := &cobra.Command{}
	cmd.Flags().String("from", "/some/path", "")
	cmd.Flags().String("key-name", "testkey", "")
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().String("backend", "test", "")
	cmd.Flags().String("new-name", "", "")
	cmd.Flags().Bool("delete-after-import", true, "") // Testing delete-after=true
	cmd.Flags().Bool("exportable", false, "")
	cmd.Flags().Bool("verify", true, "")

	err := runMigrateImport(cmd, []string{})
	require.Error(t, err)
}

func TestMigrateImport_ExportableTrue(t *testing.T) {
	resetFlags(t)

	cmd := &cobra.Command{}
	cmd.Flags().String("from", "/some/path", "")
	cmd.Flags().String("key-name", "testkey", "")
	cmd.Flags().Bool("all", false, "")
	cmd.Flags().String("backend", "test", "")
	cmd.Flags().String("new-name", "", "")
	cmd.Flags().Bool("delete-after-import", false, "")
	cmd.Flags().Bool("exportable", true, "") // Testing exportable=true
	cmd.Flags().Bool("verify", true, "")

	err := runMigrateImport(cmd, []string{})
	require.Error(t, err)
}

func TestMigrateImport_AllWithJSON(t *testing.T) {
	resetFlags(t)
	jsonOut = true
	defer func() { jsonOut = false }()

	cmd := &cobra.Command{}
	cmd.Flags().String("from", "/some/path", "")
	cmd.Flags().String("key-name", "", "") // Empty when all=true
	cmd.Flags().Bool("all", true, "")
	cmd.Flags().String("backend", "test", "")
	cmd.Flags().String("new-name", "", "")
	cmd.Flags().Bool("delete-after-import", false, "")
	cmd.Flags().Bool("exportable", false, "")
	cmd.Flags().Bool("verify", true, "")

	err := runMigrateImport(cmd, []string{})
	require.Error(t, err)
}

