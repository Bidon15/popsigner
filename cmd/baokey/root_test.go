package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		verbose     bool
		wantContain []string
	}{
		{
			name:        "basic version",
			args:        []string{"version"},
			verbose:     false,
			wantContain: []string{"baokey"},
		},
		{
			name:        "verbose version",
			args:        []string{"--verbose", "version"},
			verbose:     true,
			wantContain: []string{"baokey", "commit:", "built:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetFlags()
			var buf bytes.Buffer
			SetOutput(&buf)

			err := ExecuteWithArgs(tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := buf.String()
			for _, want := range tt.wantContain {
				if !strings.Contains(output, want) {
					t.Errorf("output %q does not contain %q", output, want)
				}
			}
		})
	}
}

func TestRootCommand_Help(t *testing.T) {
	ResetFlags()
	var buf bytes.Buffer
	SetOutput(&buf)

	err := ExecuteWithArgs([]string{"--help"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	expectedStrings := []string{
		"BaoKey",
		"OpenBao",
		"--bao-addr",
		"--bao-token",
		"--store-path",
		"--verbose",
		"BAO_ADDR",
		"BAO_TOKEN",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("help output missing %q", expected)
		}
	}
}

func TestGetConfig_FromFlags(t *testing.T) {
	ResetFlags()

	// Set flags directly
	baoAddr = "http://localhost:8200"
	baoToken = "test-token"
	storePath = "/custom/path.json"
	verbose = true

	cfg, err := GetConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.BaoAddr != "http://localhost:8200" {
		t.Errorf("BaoAddr = %q, want %q", cfg.BaoAddr, "http://localhost:8200")
	}
	if cfg.BaoToken != "test-token" {
		t.Errorf("BaoToken = %q, want %q", cfg.BaoToken, "test-token")
	}
	if cfg.StorePath != "/custom/path.json" {
		t.Errorf("StorePath = %q, want %q", cfg.StorePath, "/custom/path.json")
	}
	if !cfg.Verbose {
		t.Error("Verbose should be true")
	}
}

func TestGetConfig_FromEnv(t *testing.T) {
	ResetFlags()

	// Set environment variables
	t.Setenv(EnvBaoAddr, "http://env-addr:8200")
	t.Setenv(EnvBaoToken, "env-token")
	t.Setenv(EnvBaoStorePath, "/env/store.json")

	cfg, err := GetConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.BaoAddr != "http://env-addr:8200" {
		t.Errorf("BaoAddr = %q, want %q", cfg.BaoAddr, "http://env-addr:8200")
	}
	if cfg.BaoToken != "env-token" {
		t.Errorf("BaoToken = %q, want %q", cfg.BaoToken, "env-token")
	}
	if cfg.StorePath != "/env/store.json" {
		t.Errorf("StorePath = %q, want %q", cfg.StorePath, "/env/store.json")
	}
}

func TestGetConfig_FlagsPrecedence(t *testing.T) {
	ResetFlags()

	// Set environment variables
	t.Setenv(EnvBaoAddr, "http://env-addr:8200")
	t.Setenv(EnvBaoToken, "env-token")

	// Set flags (should take precedence)
	baoAddr = "http://flag-addr:8200"
	baoToken = "flag-token"

	cfg, err := GetConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.BaoAddr != "http://flag-addr:8200" {
		t.Errorf("BaoAddr = %q, want %q (flags should take precedence)", cfg.BaoAddr, "http://flag-addr:8200")
	}
	if cfg.BaoToken != "flag-token" {
		t.Errorf("BaoToken = %q, want %q (flags should take precedence)", cfg.BaoToken, "flag-token")
	}
}

func TestGetConfig_DefaultStorePath(t *testing.T) {
	ResetFlags()

	// Clear environment
	t.Setenv(EnvBaoStorePath, "")

	cfg, err := GetConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should use default store path
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, DefaultStorePathSuffix)
	if cfg.StorePath != expected {
		t.Errorf("StorePath = %q, want %q", cfg.StorePath, expected)
	}
}

func TestCLIConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     CLIConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			cfg: CLIConfig{
				BaoAddr:   "http://localhost:8200",
				BaoToken:  "test-token",
				StorePath: "/path/to/store.json",
			},
			wantErr: false,
		},
		{
			name: "missing bao addr",
			cfg: CLIConfig{
				BaoToken:  "test-token",
				StorePath: "/path/to/store.json",
			},
			wantErr: true,
			errMsg:  "BAO_ADDR",
		},
		{
			name: "missing bao token",
			cfg: CLIConfig{
				BaoAddr:   "http://localhost:8200",
				StorePath: "/path/to/store.json",
			},
			wantErr: true,
			errMsg:  "BAO_TOKEN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCLIConfig_ToBanhbaoConfig(t *testing.T) {
	cfg := CLIConfig{
		BaoAddr:   "http://localhost:8200",
		BaoToken:  "test-token",
		StorePath: "/path/to/store.json",
		Verbose:   true,
	}

	baoCfg := cfg.ToBanhbaoConfig()

	if baoCfg.BaoAddr != cfg.BaoAddr {
		t.Errorf("BaoAddr = %q, want %q", baoCfg.BaoAddr, cfg.BaoAddr)
	}
	if baoCfg.BaoToken != cfg.BaoToken {
		t.Errorf("BaoToken = %q, want %q", baoCfg.BaoToken, cfg.BaoToken)
	}
	if baoCfg.StorePath != cfg.StorePath {
		t.Errorf("StorePath = %q, want %q", baoCfg.StorePath, cfg.StorePath)
	}
}

func TestDefaultStorePath(t *testing.T) {
	path := DefaultStorePath()

	if path == "" {
		t.Error("DefaultStorePath should not be empty")
	}

	// Should contain the default suffix
	if !strings.HasSuffix(path, "keyring.json") {
		t.Errorf("DefaultStorePath %q should end with 'keyring.json'", path)
	}
}

func TestFormatAddress(t *testing.T) {
	tests := []struct {
		name   string
		addr   string
		maxLen int
		want   string
	}{
		{
			name:   "short address",
			addr:   "celestia1abc",
			maxLen: 20,
			want:   "celestia1abc",
		},
		{
			name:   "exact length",
			addr:   "celestia1abc",
			maxLen: 12,
			want:   "celestia1abc",
		},
		{
			name:   "long address truncated",
			addr:   "celestia1abcdefghijklmnopqrstuvwxyz",
			maxLen: 20,
			want:   "celestia...stuvwxyz",
		},
		{
			name:   "very short max",
			addr:   "celestia1abcdefghij",
			maxLen: 5,
			want:   "celes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatAddress(tt.addr, tt.maxLen)
			if got != tt.want {
				t.Errorf("FormatAddress(%q, %d) = %q, want %q", tt.addr, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestEnsureStoreDir(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "current directory",
			path:    "keyring.json",
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: false,
		},
		{
			name:    "nested directory",
			path:    filepath.Join(t.TempDir(), "subdir", "keyring.json"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnsureStoreDir(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestVerbosePrintf(t *testing.T) {
	// Test when verbose is false
	ResetFlags()
	verbose = false

	// This shouldn't print anything, but we can't easily capture stdout
	// Just verify it doesn't panic
	VerbosePrintf("test %s", "message")
	VerbosePrintln("test", "message")

	// Test when verbose is true
	verbose = true
	VerbosePrintf("test %s", "message")
	VerbosePrintln("test", "message")
}

func TestExecute(t *testing.T) {
	ResetFlags()
	var buf bytes.Buffer
	SetOutput(&buf)

	// Execute with no args should succeed (shows help)
	err := Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResetFlags(t *testing.T) {
	// Set some values
	baoAddr = "test"
	baoToken = "test"
	storePath = "test"
	verbose = true

	// Reset
	ResetFlags()

	// Verify all are empty/false
	if baoAddr != "" {
		t.Error("baoAddr should be empty after reset")
	}
	if baoToken != "" {
		t.Error("baoToken should be empty after reset")
	}
	if storePath != "" {
		t.Error("storePath should be empty after reset")
	}
	if verbose {
		t.Error("verbose should be false after reset")
	}
}

func TestConstants(t *testing.T) {
	// Verify environment variable names
	if EnvBaoAddr != "BAO_ADDR" {
		t.Errorf("EnvBaoAddr = %q, want %q", EnvBaoAddr, "BAO_ADDR")
	}
	if EnvBaoToken != "BAO_TOKEN" {
		t.Errorf("EnvBaoToken = %q, want %q", EnvBaoToken, "BAO_TOKEN")
	}
	if EnvBaoStorePath != "BAO_STORE_PATH" {
		t.Errorf("EnvBaoStorePath = %q, want %q", EnvBaoStorePath, "BAO_STORE_PATH")
	}

	// Verify default store path suffix
	if !strings.Contains(DefaultStorePathSuffix, "baokey") {
		t.Errorf("DefaultStorePathSuffix %q should contain 'baokey'", DefaultStorePathSuffix)
	}
}

func TestVersionVariables(t *testing.T) {
	// These are set via ldflags at build time, default values should be set
	if Version == "" {
		t.Error("Version should not be empty")
	}
	if Commit == "" {
		t.Error("Commit should not be empty")
	}
	if BuildDate == "" {
		t.Error("BuildDate should not be empty")
	}
}

func TestGetKeyring_MissingConfig(t *testing.T) {
	ResetFlags()

	// Clear environment
	t.Setenv(EnvBaoAddr, "")
	t.Setenv(EnvBaoToken, "")

	_, err := GetKeyring()
	if err == nil {
		t.Fatal("expected error for missing config")
	}

	if !strings.Contains(err.Error(), "BAO_ADDR") && !strings.Contains(err.Error(), "BAO_TOKEN") {
		t.Errorf("error should mention missing config: %v", err)
	}
}

func TestRootCmd_SubcommandVersion(t *testing.T) {
	ResetFlags()
	var buf bytes.Buffer
	SetOutput(&buf)

	// Test version subcommand
	err := ExecuteWithArgs([]string{"version"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "baokey") {
		t.Errorf("version output should contain 'baokey', got: %q", output)
	}
}

func TestRootCmd_InvalidSubcommand(t *testing.T) {
	ResetFlags()
	var buf bytes.Buffer
	SetOutput(&buf)

	err := ExecuteWithArgs([]string{"invalid-command"})
	if err == nil {
		t.Fatal("expected error for invalid subcommand")
	}
}

func TestFormatAddress_EdgeCases(t *testing.T) {
	// Empty string
	got := FormatAddress("", 10)
	if got != "" {
		t.Errorf("FormatAddress('', 10) = %q, want ''", got)
	}

	// Max length of 0
	got = FormatAddress("test", 0)
	if got != "" {
		t.Errorf("FormatAddress('test', 0) = %q, want ''", got)
	}
}

func TestEnsureStoreDir_NestedCreation(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "a", "b", "c", "keyring.json")

	err := EnsureStoreDir(nestedPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify directory was created
	dir := filepath.Dir(nestedPath)
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("created path is not a directory")
	}
}

