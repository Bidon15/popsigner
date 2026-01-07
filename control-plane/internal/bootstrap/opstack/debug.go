// Package opstack provides OP Stack chain deployment infrastructure.
// This file contains debug and verification helper methods for the OPDeployer.
package opstack

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
)

// logBytecodeSizes logs the bytecode sizes of key contracts for debugging.
// This helps identify which contracts might exceed the 24KB EIP-170 limit.
func (d *OPDeployer) logBytecodeSizes(artifactDir string) {
	contracts := []string{
		"OPContractsManager",
		"OPContractsManagerInterop",
		"OPContractsManagerStandardValidator",
		"OPContractsManagerGameTypeAdder",
		"OPContractsManagerDeployer",
		"OPContractsManagerUpgrader",
		"OptimismPortal2",
		"SystemConfig",
		"L1CrossDomainMessenger",
		"MIPS",
		// FaultDisputeGame contracts - often cause issues
		"FaultDisputeGame",
		"PermissionedDisputeGame",
		"FaultDisputeGameV2",
		"PermissionedDisputeGameV2",
		"PreimageOracle",
	}

	forgeDir := artifactDir + "/forge-artifacts"

	for _, name := range contracts {
		path := fmt.Sprintf("%s/%s.sol/%s.json", forgeDir, name, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue // Skip if not found
		}

		// Parse bytecode from JSON
		type artifact struct {
			Bytecode struct {
				Object string `json:"object"`
			} `json:"bytecode"`
			DeployedBytecode struct {
				Object string `json:"object"`
			} `json:"deployedBytecode"`
		}

		var a artifact
		if err := json.Unmarshal(data, &a); err != nil {
			continue
		}

		// Calculate sizes (hex string, so /2 for bytes, -2 for "0x" prefix)
		initSize := 0
		if len(a.Bytecode.Object) > 2 {
			initSize = (len(a.Bytecode.Object) - 2) / 2
		}
		deployedSize := 0
		if len(a.DeployedBytecode.Object) > 2 {
			deployedSize = (len(a.DeployedBytecode.Object) - 2) / 2
		}

		// Flag contracts that might cause issues
		status := "✓"
		if initSize > 24576 {
			status = "⚠️ INIT CODE EXCEEDS 24KB"
		} else if deployedSize > 24576 {
			status = "⚠️ DEPLOYED CODE EXCEEDS 24KB"
		} else if initSize > 20000 || deployedSize > 20000 {
			status = "⚠ CLOSE TO LIMIT"
		}

		d.logger.Info("contract bytecode size",
			slog.String("contract", name),
			slog.Int("init_bytes", initSize),
			slog.Int("deployed_bytes", deployedSize),
			slog.String("status", status),
		)
	}
}

// checkL1Code checks if a contract exists on L1 (actual chain, not simulation)
func (d *OPDeployer) checkL1Code(ctx context.Context, client *ethclient.Client, name string, addr common.Address) {
	if addr == (common.Address{}) {
		d.logger.Warn("L1 check: address is zero", slog.String("name", name))
		return
	}

	code, err := client.CodeAt(ctx, addr, nil)
	if err != nil {
		d.logger.Error("L1 check: failed to get code",
			slog.String("name", name),
			slog.String("address", addr.Hex()),
			slog.String("error", err.Error()),
		)
		return
	}

	// Check for ERC-5202 blueprint preamble
	hasValidPreamble := len(code) >= 2 && code[0] == 0xFE && code[1] == 0x71
	preambleHex := ""
	if len(code) >= 4 {
		preambleHex = fmt.Sprintf("0x%02x%02x%02x%02x", code[0], code[1], code[2], code[3])
	} else if len(code) > 0 {
		preambleHex = fmt.Sprintf("0x%x", code)
	}

	d.logger.Info("L1 code check",
		slog.String("name", name),
		slog.String("address", addr.Hex()),
		slog.Int("codeLen", len(code)),
		slog.Bool("existsOnL1", len(code) > 0),
		slog.Bool("isBlueprintPreamble", hasValidPreamble),
		slog.String("firstBytes", preambleHex),
	)
}

// queryBlueprintsFromContainer calls blueprints() on OpcmContractsContainerImpl to get stored addresses.
// This is crucial for debugging NotABlueprint() - we need to verify the container has correct blueprint refs.
func (d *OPDeployer) queryBlueprintsFromContainer(ctx context.Context, client *ethclient.Client, containerAddr common.Address) {
	if containerAddr == (common.Address{}) {
		d.logger.Error("cannot query blueprints: container address is zero")
		return
	}

	// Function selector for blueprints() - keccak256("blueprints()")[:4]
	// bytes4(keccak256("blueprints()")) = 0x15d38eab
	blueprintsSelector := common.FromHex("0x15d38eab")

	callMsg := ethereum.CallMsg{
		To:   &containerAddr,
		Data: blueprintsSelector,
	}
	result, err := client.CallContract(ctx, callMsg, nil)
	if err != nil {
		d.logger.Error("failed to call blueprints() on container",
			slog.String("address", containerAddr.Hex()),
			slog.String("error", err.Error()),
		)
		return
	}

	d.logger.Info("blueprints() call result from container",
		slog.String("containerAddr", containerAddr.Hex()),
		slog.Int("resultLen", len(result)),
		slog.String("rawResult", fmt.Sprintf("0x%x", result)),
	)

	// The result should contain addresses for each blueprint
	// Parse addresses (each address is 32 bytes, right-padded)
	if len(result) >= 32 {
		// First blueprint address (addressManager)
		addr1 := common.BytesToAddress(result[0:32])
		d.logger.Info("blueprint from container", slog.String("name", "addressManager"), slog.String("addr", addr1.Hex()))
		d.checkL1BlueprintCode(ctx, client, "addressManager", addr1)
	}
	if len(result) >= 64 {
		// Second blueprint address (proxy)
		addr2 := common.BytesToAddress(result[32:64])
		d.logger.Info("blueprint from container", slog.String("name", "proxy"), slog.String("addr", addr2.Hex()))
		d.checkL1BlueprintCode(ctx, client, "proxy", addr2)
	}
	// Log the rest if present
	for i := 2; i*32 <= len(result) && i < 10; i++ {
		addr := common.BytesToAddress(result[i*32 : (i+1)*32])
		d.logger.Info("blueprint from container", slog.Int("idx", i), slog.String("addr", addr.Hex()))
		d.checkL1BlueprintCode(ctx, client, fmt.Sprintf("blueprint[%d]", i), addr)
	}
}

// checkL1BlueprintCode verifies a blueprint has 0xFE71 preamble on L1
func (d *OPDeployer) checkL1BlueprintCode(ctx context.Context, client *ethclient.Client, name string, addr common.Address) {
	if addr == (common.Address{}) {
		d.logger.Warn("blueprint address is zero", slog.String("name", name))
		return
	}

	code, err := client.CodeAt(ctx, addr, nil)
	if err != nil {
		d.logger.Error("failed to get blueprint code from L1",
			slog.String("name", name),
			slog.String("address", addr.Hex()),
			slog.String("error", err.Error()),
		)
		return
	}

	hasValidPreamble := len(code) >= 2 && code[0] == 0xFE && code[1] == 0x71
	preambleHex := ""
	if len(code) >= 4 {
		preambleHex = fmt.Sprintf("0x%02x%02x%02x%02x", code[0], code[1], code[2], code[3])
	}

	if hasValidPreamble {
		d.logger.Info("✓ VALID BLUEPRINT on L1",
			slog.String("name", name),
			slog.String("address", addr.Hex()),
			slog.Int("codeLen", len(code)),
			slog.String("preamble", preambleHex),
		)
	} else {
		d.logger.Error("✗ INVALID BLUEPRINT on L1 - missing 0xFE71 preamble!",
			slog.String("name", name),
			slog.String("address", addr.Hex()),
			slog.Int("codeLen", len(code)),
			slog.String("firstBytes", preambleHex),
		)
	}
}

// verifyBlueprintCode checks if a contract has valid code and logs details for debugging.
// For blueprints, it also checks for the 0xFE71 preamble.
// This is used to debug NotABlueprint() errors by verifying deployments succeeded.
func (d *OPDeployer) verifyBlueprintCode(host *script.Host, name string, addr common.Address) {
	if addr == (common.Address{}) {
		d.logger.Warn("contract address is zero",
			slog.String("name", name),
		)
		return
	}

	code := host.GetCode(addr)
	codeLen := len(code)

	// Check for ERC-5202 blueprint preamble: 0xFE71<length>
	hasValidPreamble := false
	if codeLen >= 2 {
		hasValidPreamble = code[0] == 0xFE && code[1] == 0x71
	}

	// Log first few bytes for debugging
	preambleHex := ""
	if codeLen >= 4 {
		preambleHex = fmt.Sprintf("0x%02x%02x%02x%02x", code[0], code[1], code[2], code[3])
	} else if codeLen > 0 {
		preambleHex = fmt.Sprintf("0x%x", code[:codeLen])
	}

	d.logger.Info("contract code verification",
		slog.String("name", name),
		slog.String("address", addr.Hex()),
		slog.Int("codeLen", codeLen),
		slog.Bool("isBlueprintPreamble", hasValidPreamble),
		slog.String("firstBytes", preambleHex),
	)

	if codeLen == 0 {
		d.logger.Error("contract has NO CODE - deployment may have failed",
			slog.String("name", name),
			slog.String("address", addr.Hex()),
		)
	}
}
