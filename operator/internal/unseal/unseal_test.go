package unseal

import (
	"context"
	"strings"
	"testing"

	popsignerv1 "github.com/Bidon15/popsigner/operator/api/v1"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		providerType string
		wantName     string
		wantErr      bool
	}{
		{"awskms", "awskms", false},
		{"gcpkms", "gcpkms", false},
		{"azurekv", "azurekv", false},
		{"transit", "transit", false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.providerType, func(t *testing.T) {
			provider, err := NewProvider(tt.providerType)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if provider.Name() != tt.wantName {
				t.Errorf("Name() = %v, want %v", provider.Name(), tt.wantName)
			}
		})
	}
}

func TestAWSKMSProvider(t *testing.T) {
	provider := &AWSKMSProvider{}

	t.Run("Name", func(t *testing.T) {
		if provider.Name() != "awskms" {
			t.Errorf("Name() = %v, want awskms", provider.Name())
		}
	})

	t.Run("Validate_missing_config", func(t *testing.T) {
		spec := &popsignerv1.AutoUnsealSpec{
			Provider: "awskms",
		}
		if err := provider.Validate(spec); err == nil {
			t.Error("Expected validation error for missing config")
		}
	})

	t.Run("Validate_missing_keyId", func(t *testing.T) {
		spec := &popsignerv1.AutoUnsealSpec{
			Provider: "awskms",
			AWSKMS: &popsignerv1.AWSKMSSpec{
				Region: "us-west-2",
			},
		}
		if err := provider.Validate(spec); err == nil {
			t.Error("Expected validation error for missing keyId")
		}
	})

	t.Run("Validate_valid", func(t *testing.T) {
		spec := &popsignerv1.AutoUnsealSpec{
			Provider: "awskms",
			AWSKMS: &popsignerv1.AWSKMSSpec{
				KeyID:  "alias/my-key",
				Region: "us-west-2",
			},
		}
		if err := provider.Validate(spec); err != nil {
			t.Errorf("Unexpected validation error: %v", err)
		}
	})

	t.Run("GetConfig", func(t *testing.T) {
		spec := &popsignerv1.AutoUnsealSpec{
			Provider: "awskms",
			AWSKMS: &popsignerv1.AWSKMSSpec{
				KeyID:  "alias/my-key",
				Region: "eu-west-1",
			},
		}
		config := provider.GetConfig(spec)
		if !strings.Contains(config, "seal \"awskms\"") {
			t.Error("Config should contain seal type")
		}
		if !strings.Contains(config, "eu-west-1") {
			t.Error("Config should contain region")
		}
		if !strings.Contains(config, "alias/my-key") {
			t.Error("Config should contain key ID")
		}
	})

	t.Run("GetConfig_default_region", func(t *testing.T) {
		spec := &popsignerv1.AutoUnsealSpec{
			Provider: "awskms",
			AWSKMS: &popsignerv1.AWSKMSSpec{
				KeyID: "alias/my-key",
			},
		}
		config := provider.GetConfig(spec)
		if !strings.Contains(config, "us-east-1") {
			t.Error("Config should default to us-east-1 region")
		}
	})

	t.Run("GetEnvVars_with_credentials", func(t *testing.T) {
		spec := &popsignerv1.AutoUnsealSpec{
			Provider: "awskms",
			AWSKMS: &popsignerv1.AWSKMSSpec{
				KeyID:  "alias/my-key",
				Region: "us-west-2",
				Credentials: &popsignerv1.SecretKeyRef{
					Name: "aws-creds",
					Key:  "credentials",
				},
			},
		}
		envVars, err := provider.GetEnvVars(context.Background(), spec, "default")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		envNames := make(map[string]bool)
		for _, ev := range envVars {
			envNames[ev.Name] = true
		}

		if !envNames["AWS_REGION"] {
			t.Error("Should have AWS_REGION env var")
		}
		if !envNames["AWS_ACCESS_KEY_ID"] {
			t.Error("Should have AWS_ACCESS_KEY_ID env var")
		}
		if !envNames["AWS_SECRET_ACCESS_KEY"] {
			t.Error("Should have AWS_SECRET_ACCESS_KEY env var")
		}
	})

	t.Run("GetVolumes", func(t *testing.T) {
		spec := &popsignerv1.AutoUnsealSpec{
			Provider: "awskms",
			AWSKMS:   &popsignerv1.AWSKMSSpec{},
		}
		volumes := provider.GetVolumes(spec)
		if len(volumes) != 0 {
			t.Error("AWS KMS should not require additional volumes")
		}
	})
}

func TestGCPKMSProvider(t *testing.T) {
	provider := &GCPKMSProvider{}

	t.Run("Name", func(t *testing.T) {
		if provider.Name() != "gcpkms" {
			t.Errorf("Name() = %v, want gcpkms", provider.Name())
		}
	})

	t.Run("Validate_valid", func(t *testing.T) {
		spec := &popsignerv1.AutoUnsealSpec{
			Provider: "gcpkms",
			GCPKMS: &popsignerv1.GCPKMSSpec{
				Project:   "my-project",
				Location:  "global",
				KeyRing:   "my-keyring",
				CryptoKey: "my-key",
			},
		}
		if err := provider.Validate(spec); err != nil {
			t.Errorf("Unexpected validation error: %v", err)
		}
	})

	t.Run("GetConfig", func(t *testing.T) {
		spec := &popsignerv1.AutoUnsealSpec{
			Provider: "gcpkms",
			GCPKMS: &popsignerv1.GCPKMSSpec{
				Project:   "my-project",
				Location:  "global",
				KeyRing:   "my-keyring",
				CryptoKey: "my-key",
			},
		}
		config := provider.GetConfig(spec)
		if !strings.Contains(config, "seal \"gcpckms\"") {
			t.Error("Config should contain seal type")
		}
		if !strings.Contains(config, "my-project") {
			t.Error("Config should contain project")
		}
	})

	t.Run("GetVolumes_with_credentials", func(t *testing.T) {
		spec := &popsignerv1.AutoUnsealSpec{
			Provider: "gcpkms",
			GCPKMS: &popsignerv1.GCPKMSSpec{
				Project:   "my-project",
				Location:  "global",
				KeyRing:   "my-keyring",
				CryptoKey: "my-key",
				Credentials: &popsignerv1.SecretKeyRef{
					Name: "gcp-creds",
					Key:  "key.json",
				},
			},
		}
		volumes := provider.GetVolumes(spec)
		if len(volumes) != 1 {
			t.Errorf("Expected 1 volume, got %d", len(volumes))
		}
	})
}

func TestAzureKVProvider(t *testing.T) {
	provider := &AzureKVProvider{}

	t.Run("Name", func(t *testing.T) {
		if provider.Name() != "azurekv" {
			t.Errorf("Name() = %v, want azurekv", provider.Name())
		}
	})

	t.Run("Validate_valid", func(t *testing.T) {
		spec := &popsignerv1.AutoUnsealSpec{
			Provider: "azurekv",
			AzureKV: &popsignerv1.AzureKVSpec{
				TenantID:  "my-tenant",
				VaultName: "my-vault",
				KeyName:   "my-key",
			},
		}
		if err := provider.Validate(spec); err != nil {
			t.Errorf("Unexpected validation error: %v", err)
		}
	})

	t.Run("GetConfig", func(t *testing.T) {
		spec := &popsignerv1.AutoUnsealSpec{
			Provider: "azurekv",
			AzureKV: &popsignerv1.AzureKVSpec{
				TenantID:  "my-tenant",
				VaultName: "my-vault",
				KeyName:   "my-key",
			},
		}
		config := provider.GetConfig(spec)
		if !strings.Contains(config, "seal \"azurekeyvault\"") {
			t.Error("Config should contain seal type")
		}
		if !strings.Contains(config, "my-tenant") {
			t.Error("Config should contain tenant ID")
		}
	})
}

func TestTransitProvider(t *testing.T) {
	provider := &TransitProvider{}

	t.Run("Name", func(t *testing.T) {
		if provider.Name() != "transit" {
			t.Errorf("Name() = %v, want transit", provider.Name())
		}
	})

	t.Run("Validate_valid", func(t *testing.T) {
		spec := &popsignerv1.AutoUnsealSpec{
			Provider: "transit",
			Transit: &popsignerv1.TransitSpec{
				Address:   "https://vault.example.com:8200",
				MountPath: "transit",
				KeyName:   "autounseal",
			},
		}
		if err := provider.Validate(spec); err != nil {
			t.Errorf("Unexpected validation error: %v", err)
		}
	})

	t.Run("GetConfig", func(t *testing.T) {
		spec := &popsignerv1.AutoUnsealSpec{
			Provider: "transit",
			Transit: &popsignerv1.TransitSpec{
				Address:   "https://vault.example.com:8200",
				MountPath: "transit",
				KeyName:   "autounseal",
			},
		}
		config := provider.GetConfig(spec)
		if !strings.Contains(config, "seal \"transit\"") {
			t.Error("Config should contain seal type")
		}
		if !strings.Contains(config, "vault.example.com") {
			t.Error("Config should contain address")
		}
	})

	t.Run("GetEnvVars", func(t *testing.T) {
		spec := &popsignerv1.AutoUnsealSpec{
			Provider: "transit",
			Transit: &popsignerv1.TransitSpec{
				Address: "https://vault.example.com:8200",
				KeyName: "autounseal",
				Token: popsignerv1.SecretKeyRef{
					Name: "transit-token",
					Key:  "token",
				},
			},
		}
		envVars, err := provider.GetEnvVars(context.Background(), spec, "default")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		hasToken := false
		for _, ev := range envVars {
			if ev.Name == "VAULT_TOKEN" {
				hasToken = true
				break
			}
		}
		if !hasToken {
			t.Error("Should have VAULT_TOKEN env var")
		}
	})
}

func TestGetProviderForCluster(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		cluster := &popsignerv1.POPSignerCluster{
			Spec: popsignerv1.POPSignerClusterSpec{
				OpenBao: popsignerv1.OpenBaoSpec{
					AutoUnseal: popsignerv1.AutoUnsealSpec{
						Enabled: false,
					},
				},
			},
		}
		provider, err := GetProviderForCluster(cluster)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if provider != nil {
			t.Error("Provider should be nil when auto-unseal is disabled")
		}
	})

	t.Run("enabled", func(t *testing.T) {
		cluster := &popsignerv1.POPSignerCluster{
			Spec: popsignerv1.POPSignerClusterSpec{
				OpenBao: popsignerv1.OpenBaoSpec{
					AutoUnseal: popsignerv1.AutoUnsealSpec{
						Enabled:  true,
						Provider: "awskms",
					},
				},
			},
		}
		provider, err := GetProviderForCluster(cluster)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if provider == nil {
			t.Error("Provider should not be nil when auto-unseal is enabled")
		}
		if provider.Name() != "awskms" {
			t.Errorf("Provider name = %v, want awskms", provider.Name())
		}
	})
}
