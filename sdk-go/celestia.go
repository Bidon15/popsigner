// Celestia integration for POPSigner SDK.
//
// This package provides a drop-in keyring implementation for the Celestia Node client,
// allowing you to use POPSigner as a secure remote signer for blob submission.
//
// Example usage:
//
//	// Create a Celestia-compatible keyring backed by POPSigner
//	kr, err := popsigner.NewCelestiaKeyring("your-api-key", "your-key-id")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Use with Celestia client
//	cfg := client.Config{
//	    ReadConfig: client.ReadConfig{
//	        BridgeDAAddr: "http://localhost:26658",
//	        DAAuthToken:  "your_token",
//	    },
//	    SubmitConfig: client.SubmitConfig{
//	        DefaultKeyName: kr.KeyName(),
//	        Network:        "mocha-4",
//	    },
//	}
//
//	celestiaClient, err := client.New(ctx, cfg, kr)
package popsigner

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
)

// CelestiaKeyring implements keyring functionality for the Celestia Node client.
// It uses POPSigner as the backend for secure remote signing.
//
// This keyring can be passed directly to the Celestia client.New() function.
type CelestiaKeyring struct {
	client   *Client
	keyID    string
	keyName  string
	pubKey   []byte
	address  string
	celestia string // bech32 celestia1... address
}

// CelestiaKeyringOption configures the CelestiaKeyring.
type CelestiaKeyringOption func(*celestiaKeyringConfig)

type celestiaKeyringConfig struct {
	baseURL string
}

// WithCelestiaBaseURL sets a custom API base URL for the Celestia keyring.
func WithCelestiaBaseURL(url string) CelestiaKeyringOption {
	return func(cfg *celestiaKeyringConfig) {
		cfg.baseURL = url
	}
}

// NewCelestiaKeyring creates a new Celestia-compatible keyring backed by POPSigner.
//
// The apiKey is your POPSigner API key.
// The keyID is the UUID of the signing key to use.
//
// Example:
//
//	kr, err := popsigner.NewCelestiaKeyring("psk_live_xxx", "344399b0-1234-5678-9abc-def012345678")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Now use kr with celestia client.New(ctx, cfg, kr)
func NewCelestiaKeyring(apiKey, keyID string, opts ...CelestiaKeyringOption) (*CelestiaKeyring, error) {
	cfg := &celestiaKeyringConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Create POPSigner client
	clientOpts := []Option{}
	if cfg.baseURL != "" {
		clientOpts = append(clientOpts, WithBaseURL(cfg.baseURL))
	}
	client := NewClient(apiKey, clientOpts...)

	// Validate key ID format
	keyUUID, err := uuid.Parse(keyID)
	if err != nil {
		return nil, fmt.Errorf("invalid key ID format: %w", err)
	}

	// Fetch key info to validate it exists and get public key
	key, err := client.Keys.Get(context.Background(), keyUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch key: %w", err)
	}

	// Decode public key
	pubKey, err := hex.DecodeString(key.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	// Derive Celestia address from hex address
	celestiaAddr := deriveCelestiaAddress(key.Address)

	return &CelestiaKeyring{
		client:   client,
		keyID:    keyID,
		keyName:  key.Name,
		pubKey:   pubKey,
		address:  key.Address,
		celestia: celestiaAddr,
	}, nil
}

// KeyName returns the name of the key being used.
func (k *CelestiaKeyring) KeyName() string {
	return k.keyName
}

// KeyID returns the POPSigner key ID.
func (k *CelestiaKeyring) KeyID() string {
	return k.keyID
}

// Address returns the hex-encoded address.
func (k *CelestiaKeyring) Address() string {
	return k.address
}

// CelestiaAddress returns the bech32 Celestia address (celestia1...).
func (k *CelestiaKeyring) CelestiaAddress() string {
	return k.celestia
}

// PublicKey returns the compressed secp256k1 public key.
func (k *CelestiaKeyring) PublicKey() []byte {
	return k.pubKey
}

// Sign signs a message using POPSigner.
// This implements the signing interface expected by the Celestia client.
func (k *CelestiaKeyring) Sign(name string, msg []byte) ([]byte, []byte, error) {
	keyUUID, err := uuid.Parse(k.keyID)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid key ID: %w", err)
	}

	resp, err := k.client.Sign.Sign(context.Background(), keyUUID, msg, false)
	if err != nil {
		return nil, nil, fmt.Errorf("signing failed: %w", err)
	}

	return resp.Signature, k.pubKey, nil
}

// SignByAddress signs with the key matching the given address.
func (k *CelestiaKeyring) SignByAddress(address []byte, msg []byte) ([]byte, []byte, error) {
	// For simplicity, we only support the single key configured
	return k.Sign(k.keyName, msg)
}

// deriveCelestiaAddress converts a hex address to bech32 celestia format.
func deriveCelestiaAddress(hexAddr string) string {
	addrBytes, err := hex.DecodeString(hexAddr)
	if err != nil || len(addrBytes) != 20 {
		return ""
	}

	// Bech32 encode with "celestia" prefix
	result, err := bech32Encode("celestia", addrBytes)
	if err != nil {
		return ""
	}
	return result
}

// bech32Encode encodes data to bech32 format.
func bech32Encode(hrp string, data []byte) (string, error) {
	converted := make([]byte, 0, len(data)*8/5+1)
	acc := 0
	bits := 0
	for _, b := range data {
		acc = (acc << 8) | int(b)
		bits += 8
		for bits >= 5 {
			bits -= 5
			converted = append(converted, byte((acc>>bits)&0x1f))
		}
	}
	if bits > 0 {
		converted = append(converted, byte((acc<<(5-bits))&0x1f))
	}

	values := append(expandHRP(hrp), converted...)
	values = append(values, 0, 0, 0, 0, 0, 0)
	polymod := bech32Polymod(values) ^ 1
	checksum := make([]byte, 6)
	for i := 0; i < 6; i++ {
		checksum[i] = byte((polymod >> (5 * (5 - i))) & 0x1f)
	}

	charset := "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
	result := hrp + "1"
	for _, b := range converted {
		result += string(charset[b])
	}
	for _, b := range checksum {
		result += string(charset[b])
	}

	return result, nil
}

func expandHRP(hrp string) []byte {
	result := make([]byte, len(hrp)*2+1)
	for i, c := range hrp {
		result[i] = byte(c >> 5)
		result[i+len(hrp)+1] = byte(c & 0x1f)
	}
	result[len(hrp)] = 0
	return result
}

func bech32Polymod(values []byte) int {
	gen := []int{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}
	chk := 1
	for _, v := range values {
		b := chk >> 25
		chk = ((chk & 0x1ffffff) << 5) ^ int(v)
		for i := 0; i < 5; i++ {
			if (b>>i)&1 == 1 {
				chk ^= gen[i]
			}
		}
	}
	return chk
}
