package compose

import (
	"strings"
	"testing"
)

func TestGenerateOPStack(t *testing.T) {
	gen := NewGenerator()

	cfg := &ComposeConfig{
		ChainID:              42069,
		ChainName:            "my-test-chain",
		L1RPC:                "${L1_RPC_URL}",
		DAType:               "celestia",
		CelestiaRPC:          "https://celestia-rpc.popsigner.com",
		POPSignerRPCEndpoint: "https://rpc.popsigner.com",
		BatcherAddress:       "0x1111111111111111111111111111111111111111",
		ProposerAddress:      "0x2222222222222222222222222222222222222222",
		Contracts: map[string]string{
			"l2_output_oracle": "0x3333333333333333333333333333333333333333",
		},
	}

	result, err := gen.Generate(StackOPStack, cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify OP Stack compose contains expected services
	if !strings.Contains(result.ComposeYAML, "op-node:") {
		t.Error("Missing op-node service")
	}
	if !strings.Contains(result.ComposeYAML, "op-geth:") {
		t.Error("Missing op-geth service")
	}
	if !strings.Contains(result.ComposeYAML, "op-batcher:") {
		t.Error("Missing op-batcher service")
	}
	if !strings.Contains(result.ComposeYAML, "op-proposer:") {
		t.Error("Missing op-proposer service")
	}
	if !strings.Contains(result.ComposeYAML, "op-alt-da:") {
		t.Error("Missing op-alt-da service for Celestia DA")
	}

	// Verify API key auth is used (not mTLS)
	if !strings.Contains(result.ComposeYAML, "--signer.header=X-API-Key") {
		t.Error("Missing API key auth configuration")
	}
	if strings.Contains(result.ComposeYAML, "external-signer") {
		t.Error("Should not contain mTLS external-signer for OP Stack")
	}

	// Verify network name contains chain name
	if !strings.Contains(result.ComposeYAML, "my-test-chain-opstack-network") {
		t.Error("Network name should contain sanitized chain name")
	}

	// Verify health checks are present
	if !strings.Contains(result.ComposeYAML, "healthcheck:") {
		t.Error("Missing health checks")
	}

	// Verify logging configuration
	if !strings.Contains(result.ComposeYAML, "x-logging:") {
		t.Error("Missing logging anchor configuration")
	}
}

func TestGenerateOPStackWithoutCelestia(t *testing.T) {
	gen := NewGenerator()

	cfg := &ComposeConfig{
		ChainID:              42069,
		ChainName:            "my-l1-da-chain",
		DAType:               "", // No alt-DA
		POPSignerRPCEndpoint: "https://rpc.popsigner.com",
		BatcherAddress:       "0x1111111111111111111111111111111111111111",
		ProposerAddress:      "0x2222222222222222222222222222222222222222",
	}

	result, err := gen.Generate(StackOPStack, cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should NOT contain alt-da service
	if strings.Contains(result.ComposeYAML, "op-alt-da:") {
		t.Error("Should not contain op-alt-da service when DAType is empty")
	}

	// Should NOT contain altda config flags
	if strings.Contains(result.ComposeYAML, "--altda.enabled=true") {
		t.Error("Should not contain altda flags when DAType is empty")
	}
}

func TestGenerateNitro(t *testing.T) {
	gen := NewGenerator()

	cfg := &ComposeConfig{
		ChainID:               42170,
		ChainName:             "my-orbit-chain",
		L1RPC:                 "${L1_RPC_URL}",
		DAType:                "celestia",
		CelestiaRPC:           "https://celestia-rpc.popsigner.com",
		POPSignerMTLSEndpoint: "https://mtls.popsigner.com:8546",
		ValidatorAddress:      "0x4444444444444444444444444444444444444444",
		RollupAddress:         "0x5555555555555555555555555555555555555555",
		InboxAddress:          "0x6666666666666666666666666666666666666666",
		SequencerInboxAddr:    "0x7777777777777777777777777777777777777777",
	}

	result, err := gen.Generate(StackNitro, cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify Nitro compose contains expected services
	if !strings.Contains(result.ComposeYAML, "nitro:") {
		t.Error("Missing nitro service")
	}
	if !strings.Contains(result.ComposeYAML, "batch-poster:") {
		t.Error("Missing batch-poster service")
	}
	if !strings.Contains(result.ComposeYAML, "validator:") {
		t.Error("Missing validator service")
	}
	if !strings.Contains(result.ComposeYAML, "celestia-server:") {
		t.Error("Missing celestia-server service for Celestia DA")
	}

	// Verify mTLS auth is used (not API key)
	if !strings.Contains(result.ComposeYAML, "external-signer.client-cert=/certs/client.crt") {
		t.Error("Missing mTLS client cert configuration")
	}
	if !strings.Contains(result.ComposeYAML, "external-signer.client-private-key=/certs/client.key") {
		t.Error("Missing mTLS client key configuration")
	}
	if strings.Contains(result.ComposeYAML, "--signer.header") {
		t.Error("Should not contain API key auth for Nitro")
	}

	// Verify certificate volume mount
	if !strings.Contains(result.ComposeYAML, "./certs:/certs:ro") {
		t.Error("Missing certificate volume mount")
	}

	// Verify network name
	if !strings.Contains(result.ComposeYAML, "my-orbit-chain-nitro-network") {
		t.Error("Network name should contain sanitized chain name")
	}
}

func TestGenerateNitroWithoutCelestia(t *testing.T) {
	gen := NewGenerator()

	cfg := &ComposeConfig{
		ChainID:               42170,
		ChainName:             "my-anytrust-chain",
		DAType:                "", // No Celestia
		POPSignerMTLSEndpoint: "https://mtls.popsigner.com:8546",
	}

	result, err := gen.Generate(StackNitro, cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should NOT contain celestia-server service
	if strings.Contains(result.ComposeYAML, "celestia-server:") {
		t.Error("Should not contain celestia-server when DAType is empty")
	}

	// Core services should still be present
	if !strings.Contains(result.ComposeYAML, "nitro:") {
		t.Error("Missing nitro service")
	}
	if !strings.Contains(result.ComposeYAML, "batch-poster:") {
		t.Error("Missing batch-poster service")
	}
	if !strings.Contains(result.ComposeYAML, "validator:") {
		t.Error("Missing validator service")
	}
}

func TestEnvFileGenerationOPStack(t *testing.T) {
	gen := NewGenerator()

	cfg := &ComposeConfig{
		ChainID:              42069,
		ChainName:            "test-chain",
		DAType:               "celestia",
		POPSignerRPCEndpoint: "https://rpc.popsigner.com",
		BatcherAddress:       "0x1111111111111111111111111111111111111111",
		ProposerAddress:      "0x2222222222222222222222222222222222222222",
		CelestiaRPC:          "https://celestia.popsigner.com",
		Contracts: map[string]string{
			"l2_output_oracle": "0x3333333333333333333333333333333333333333",
		},
	}

	result, err := gen.Generate(StackOPStack, cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify OP Stack env contains expected variables
	if !strings.Contains(result.EnvExample, "POPSIGNER_RPC_URL") {
		t.Error("OP Stack env missing RPC URL")
	}
	if !strings.Contains(result.EnvExample, "POPSIGNER_API_KEY") {
		t.Error("OP Stack env missing API key")
	}
	if !strings.Contains(result.EnvExample, "BATCHER_ADDRESS=0x1111111111111111111111111111111111111111") {
		t.Error("OP Stack env missing batcher address")
	}
	if !strings.Contains(result.EnvExample, "PROPOSER_ADDRESS=0x2222222222222222222222222222222222222222") {
		t.Error("OP Stack env missing proposer address")
	}
	if !strings.Contains(result.EnvExample, "L2_OUTPUT_ORACLE_ADDRESS") {
		t.Error("OP Stack env missing L2 output oracle address")
	}
	if !strings.Contains(result.EnvExample, "CELESTIA_RPC_URL") {
		t.Error("OP Stack env missing Celestia RPC URL")
	}
	if !strings.Contains(result.EnvExample, "Chain ID: 42069") {
		t.Error("OP Stack env missing chain ID in header")
	}
}

func TestEnvFileGenerationNitro(t *testing.T) {
	gen := NewGenerator()

	cfg := &ComposeConfig{
		ChainID:               42170,
		ChainName:             "orbit-chain",
		DAType:                "celestia",
		POPSignerMTLSEndpoint: "https://mtls.popsigner.com:8546",
		CelestiaRPC:           "https://celestia.popsigner.com",
		RollupAddress:         "0x5555555555555555555555555555555555555555",
	}

	result, err := gen.Generate(StackNitro, cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify Nitro env contains expected variables
	if !strings.Contains(result.EnvExample, "POPSIGNER_MTLS_URL") {
		t.Error("Nitro env missing mTLS URL")
	}
	if !strings.Contains(result.EnvExample, "mTLS Certificate") {
		t.Error("Nitro env missing cert instructions")
	}
	if !strings.Contains(result.EnvExample, "client.crt") {
		t.Error("Nitro env missing client cert reference")
	}
	if !strings.Contains(result.EnvExample, "client.key") {
		t.Error("Nitro env missing client key reference")
	}
	if !strings.Contains(result.EnvExample, "CELESTIA_RPC_URL") {
		t.Error("Nitro env missing Celestia RPC URL")
	}
	if !strings.Contains(result.EnvExample, "Chain ID: 42170") {
		t.Error("Nitro env missing chain ID in header")
	}
}

func TestEnvFileNoCelestia(t *testing.T) {
	gen := NewGenerator()

	cfg := &ComposeConfig{
		ChainID:              42069,
		ChainName:            "test-chain",
		DAType:               "", // No Celestia
		POPSignerRPCEndpoint: "https://rpc.popsigner.com",
	}

	result, err := gen.Generate(StackOPStack, cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should NOT contain Celestia config
	if strings.Contains(result.EnvExample, "CELESTIA_RPC_URL") {
		t.Error("Env should not contain Celestia config when DAType is empty")
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-chain", "my-chain"},
		{"My Chain", "my-chain"},
		{"My_Chain_123", "my_chain_123"},
		{"UPPERCASE", "uppercase"},
		{"special!@#$chars", "specialchars"},
		{"", "rollup"},
		{"   ", "rollup"},
		{"chain-name-with-dashes", "chain-name-with-dashes"},
		{"chain_name_with_underscores", "chain_name_with_underscores"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConvenienceFunctions(t *testing.T) {
	cfg := &ComposeConfig{
		ChainID:              42069,
		ChainName:            "test-chain",
		POPSignerRPCEndpoint: "https://rpc.popsigner.com",
	}

	// Test GenerateOPStack convenience function
	opResult, err := GenerateOPStack(cfg)
	if err != nil {
		t.Fatalf("GenerateOPStack failed: %v", err)
	}
	if !strings.Contains(opResult.ComposeYAML, "op-node:") {
		t.Error("GenerateOPStack should generate OP Stack compose")
	}

	// Test GenerateNitro convenience function
	cfg.POPSignerMTLSEndpoint = "https://mtls.popsigner.com:8546"
	nitroResult, err := GenerateNitro(cfg)
	if err != nil {
		t.Fatalf("GenerateNitro failed: %v", err)
	}
	if !strings.Contains(nitroResult.ComposeYAML, "nitro:") {
		t.Error("GenerateNitro should generate Nitro compose")
	}
}

func TestNilConfig(t *testing.T) {
	gen := NewGenerator()

	_, err := gen.Generate(StackOPStack, nil)
	if err == nil {
		t.Error("Generate should return error for nil config")
	}
}

func TestEmptyChainName(t *testing.T) {
	gen := NewGenerator()

	cfg := &ComposeConfig{
		ChainID:              42069,
		ChainName:            "", // Empty chain name
		POPSignerRPCEndpoint: "https://rpc.popsigner.com",
	}

	result, err := gen.Generate(StackOPStack, cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Should use default "rollup" name
	if !strings.Contains(result.ComposeYAML, "rollup-opstack-network") {
		t.Error("Should use default 'rollup' name when ChainName is empty")
	}
}

func TestVolumesMounts(t *testing.T) {
	gen := NewGenerator()

	// Test OP Stack volumes
	opCfg := &ComposeConfig{
		ChainID:              42069,
		ChainName:            "test",
		POPSignerRPCEndpoint: "https://rpc.popsigner.com",
	}

	opResult, _ := gen.Generate(StackOPStack, opCfg)

	// Check OP Stack volume mounts
	if !strings.Contains(opResult.ComposeYAML, "./config:/config:ro") {
		t.Error("OP Stack should have config volume mount")
	}
	if !strings.Contains(opResult.ComposeYAML, "./secrets:/secrets:ro") {
		t.Error("OP Stack should have secrets volume mount")
	}
	if !strings.Contains(opResult.ComposeYAML, "./genesis/genesis.json:/genesis.json:ro") {
		t.Error("OP Stack should have genesis volume mount")
	}

	// Test Nitro volumes
	nitroCfg := &ComposeConfig{
		ChainID:               42170,
		ChainName:             "test",
		POPSignerMTLSEndpoint: "https://mtls.popsigner.com:8546",
	}

	nitroResult, _ := gen.Generate(StackNitro, nitroCfg)

	// Check Nitro volume mounts
	if !strings.Contains(nitroResult.ComposeYAML, "./config/node-config.json:/config/node-config.json:ro") {
		t.Error("Nitro should have node-config volume mount")
	}
	if !strings.Contains(nitroResult.ComposeYAML, "./config/chain-info.json:/config/chain-info.json:ro") {
		t.Error("Nitro should have chain-info volume mount")
	}
	if !strings.Contains(nitroResult.ComposeYAML, "./certs:/certs:ro") {
		t.Error("Nitro should have certs volume mount")
	}
}

func TestPortExposure(t *testing.T) {
	gen := NewGenerator()

	// Test OP Stack ports
	opCfg := &ComposeConfig{
		ChainID:              42069,
		ChainName:            "test",
		POPSignerRPCEndpoint: "https://rpc.popsigner.com",
	}

	opResult, _ := gen.Generate(StackOPStack, opCfg)

	// Check OP Stack exposed ports
	if !strings.Contains(opResult.ComposeYAML, "8545:8545") {
		t.Error("OP Stack should expose geth HTTP port 8545")
	}
	if !strings.Contains(opResult.ComposeYAML, "8546:8546") {
		t.Error("OP Stack should expose geth WS port 8546")
	}
	if !strings.Contains(opResult.ComposeYAML, "9545:8545") {
		t.Error("OP Stack should expose op-node port 9545")
	}

	// Test Nitro ports
	nitroCfg := &ComposeConfig{
		ChainID:               42170,
		ChainName:             "test",
		POPSignerMTLSEndpoint: "https://mtls.popsigner.com:8546",
	}

	nitroResult, _ := gen.Generate(StackNitro, nitroCfg)

	// Check Nitro exposed ports
	if !strings.Contains(nitroResult.ComposeYAML, "8547:8547") {
		t.Error("Nitro should expose HTTP port 8547")
	}
	if !strings.Contains(nitroResult.ComposeYAML, "8548:8548") {
		t.Error("Nitro should expose WS port 8548")
	}
	if !strings.Contains(nitroResult.ComposeYAML, "9642:9642") {
		t.Error("Nitro should expose metrics port 9642")
	}
}

func TestDockerImageVersions(t *testing.T) {
	gen := NewGenerator()

	// Test OP Stack images
	opCfg := &ComposeConfig{
		ChainID:              42069,
		ChainName:            "test",
		POPSignerRPCEndpoint: "https://rpc.popsigner.com",
	}

	opResult, _ := gen.Generate(StackOPStack, opCfg)

	// Check OP Stack uses versioned images
	if !strings.Contains(opResult.ComposeYAML, "op-node:v1.9.0") {
		t.Error("OP Stack should use versioned op-node image")
	}
	if !strings.Contains(opResult.ComposeYAML, "op-geth:v1.101408.0") {
		t.Error("OP Stack should use versioned op-geth image")
	}
	if !strings.Contains(opResult.ComposeYAML, "op-batcher:v1.9.0") {
		t.Error("OP Stack should use versioned op-batcher image")
	}
	if !strings.Contains(opResult.ComposeYAML, "op-proposer:v1.9.0") {
		t.Error("OP Stack should use versioned op-proposer image")
	}

	// Test Nitro images
	nitroCfg := &ComposeConfig{
		ChainID:               42170,
		ChainName:             "test",
		POPSignerMTLSEndpoint: "https://mtls.popsigner.com:8546",
	}

	nitroResult, _ := gen.Generate(StackNitro, nitroCfg)

	// Check Nitro uses versioned images
	if !strings.Contains(nitroResult.ComposeYAML, "nitro-node:v3.1.0") {
		t.Error("Nitro should use versioned nitro-node image")
	}
}

