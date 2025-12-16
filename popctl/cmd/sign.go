package cmd

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Bidon15/popsigner/popctl/internal/api"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var signCmd = &cobra.Command{
	Use:   "sign <key-id>",
	Short: "Sign data with a key",
	Long: `Sign data using the specified key.

Data can be provided in several formats:
  --data <base64>        Base64-encoded data
  --data file:/path      Read data from file
  --data-hex <hex>       Hex-encoded data (useful for transaction hashes)

For blockchain transactions, use --prehashed if the data is already hashed.

Examples:
  popctl sign 01HXYZ... --data SGVsbG8gV29ybGQ=
  popctl sign 01HXYZ... --data file:./message.txt
  popctl sign 01HXYZ... --data-hex deadbeef... --prehashed`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSign,
}

var signBatchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Sign multiple messages in parallel",
	Long: `Sign multiple messages in parallel for maximum throughput.

The requests file should be a JSON file with the following format:
{
  "requests": [
    {"key_id": "01HXYZ...", "data": "base64data", "prehashed": false},
    {"key_id": "01HABC...", "data": "base64data", "prehashed": true}
  ]
}

Example:
  popctl sign batch --requests requests.json`,
	RunE: runSignBatch,
}

func init() {
	// Sign flags
	signCmd.Flags().String("data", "", "base64-encoded data to sign (or file:/path)")
	signCmd.Flags().String("data-hex", "", "hex-encoded data to sign")
	signCmd.Flags().Bool("prehashed", false, "data is already hashed")
	signCmd.Flags().String("output", "base64", "output format: base64, hex")

	// Batch flags
	signBatchCmd.Flags().String("requests", "", "JSON file with sign requests (required)")
	_ = signBatchCmd.MarkFlagRequired("requests")

	signCmd.AddCommand(signBatchCmd)
	rootCmd.AddCommand(signCmd)
}

func runSign(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}

	keyID, err := uuid.Parse(args[0])
	if err != nil {
		return fmt.Errorf("invalid key ID: %w", err)
	}

	// Get data from flags
	dataB64, err := getDataFromFlags(cmd)
	if err != nil {
		return err
	}

	prehashed, _ := cmd.Flags().GetBool("prehashed")
	outputFormat, _ := cmd.Flags().GetString("output")

	client, err := getClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	result, err := client.Sign(ctx, keyID, dataB64, prehashed)
	if err != nil {
		printError(err)
		return err
	}

	// Decode signature for output formatting
	sigBytes, _ := base64.StdEncoding.DecodeString(result.Signature)
	sigStr := result.Signature
	if outputFormat == "hex" && len(sigBytes) > 0 {
		sigStr = hex.EncodeToString(sigBytes)
	}

	if jsonOut {
		return printJSON(map[string]interface{}{
			"key_id":      keyID.String(),
			"signature":   sigStr,
			"public_key":  result.PublicKey,
			"key_version": result.KeyVersion,
		})
	}

	fmt.Printf("%s Signed data with key %s\n", colorGreen("✓"), truncate(keyID.String(), 12))
	fmt.Printf("\nSignature (%s):\n%s\n", outputFormat, sigStr)
	fmt.Printf("\nPublic Key: %s\n", result.PublicKey)
	fmt.Printf("Key Version: %d\n", result.KeyVersion)

	return nil
}

func runSignBatch(cmd *cobra.Command, args []string) error {
	requestsFile, _ := cmd.Flags().GetString("requests")

	// Read requests file
	data, err := os.ReadFile(requestsFile)
	if err != nil {
		return fmt.Errorf("failed to read requests file: %w", err)
	}

	var input struct {
		Requests []struct {
			KeyID     string `json:"key_id"`
			Data      string `json:"data"`
			Prehashed bool   `json:"prehashed"`
		} `json:"requests"`
	}

	if err := json.Unmarshal(data, &input); err != nil {
		return fmt.Errorf("failed to parse requests file: %w", err)
	}

	if len(input.Requests) == 0 {
		return fmt.Errorf("no requests in file")
	}

	// Build sign requests
	requests := make([]api.SignRequest, len(input.Requests))
	for i, r := range input.Requests {
		keyID, err := uuid.Parse(r.KeyID)
		if err != nil {
			return fmt.Errorf("request %d: invalid key ID: %w", i, err)
		}

		requests[i] = api.SignRequest{
			KeyID:     keyID,
			Data:      r.Data,
			Prehashed: r.Prehashed,
		}
	}

	client, err := getClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	result, err := client.SignBatch(ctx, requests)
	if err != nil {
		printError(err)
		return err
	}

	if jsonOut {
		return printJSON(map[string]interface{}{
			"results": result.Signatures,
			"count":   result.Count,
		})
	}

	fmt.Printf("%s Signed %d messages in parallel\n\n", colorGreen("✓"), len(result.Signatures))

	w := newTable()
	printTableHeader(w, "KEY ID", "STATUS", "SIGNATURE")
	for _, r := range result.Signatures {
		status := colorGreen("ok")
		sig := truncate(r.Signature, 24)
		if r.Error != "" {
			status = colorRed("error")
			sig = r.Error
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", truncate(r.KeyID.String(), 12), status, sig)
	}
	return w.Flush()
}

// getDataFromFlags extracts data from --data or --data-hex flags.
func getDataFromFlags(cmd *cobra.Command) (string, error) {
	dataStr, _ := cmd.Flags().GetString("data")
	dataHex, _ := cmd.Flags().GetString("data-hex")

	if dataStr == "" && dataHex == "" {
		return "", fmt.Errorf("either --data or --data-hex is required")
	}

	if dataStr != "" && dataHex != "" {
		return "", fmt.Errorf("cannot use both --data and --data-hex")
	}

	if dataHex != "" {
		// Decode hex and re-encode as base64
		decoded, err := hex.DecodeString(dataHex)
		if err != nil {
			return "", fmt.Errorf("invalid hex data: %w", err)
		}
		return base64.StdEncoding.EncodeToString(decoded), nil
	}

	// Check if it's a file reference
	if strings.HasPrefix(dataStr, "file:") {
		filePath := strings.TrimPrefix(dataStr, "file:")
		fileData, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
		return base64.StdEncoding.EncodeToString(fileData), nil
	}

	// Otherwise, assume it's already base64
	return dataStr, nil
}

