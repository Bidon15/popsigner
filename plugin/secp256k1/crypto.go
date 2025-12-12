package secp256k1

import (
	"crypto/sha256"
	"fmt"
	"runtime"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"golang.org/x/crypto/ripemd160" //nolint:staticcheck // Required for Cosmos address derivation
	"golang.org/x/crypto/sha3"
)

// GenerateKey creates a new secp256k1 keypair using btcec.
// Returns the private key and public key (compressed 33-byte format).
func GenerateKey() (*btcec.PrivateKey, *btcec.PublicKey, error) {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}
	return privKey, privKey.PubKey(), nil
}

// SignMessage signs a message hash using ECDSA with low-S normalization (BIP-62).
// The hash should be 32 bytes (e.g., SHA-256 of the message).
// Returns the signature in R||S format (64 bytes).
func SignMessage(privKey *btcec.PrivateKey, hash []byte) ([]byte, error) {
	if len(hash) != 32 {
		return nil, fmt.Errorf("hash must be 32 bytes, got %d", len(hash))
	}
	if privKey == nil {
		return nil, fmt.Errorf("private key cannot be nil")
	}

	sig := ecdsa.Sign(privKey, hash)
	return formatCosmosSignature(sig), nil
}

// VerifySignature verifies an ECDSA signature against a message hash and public key.
// The signature should be in R||S format (64 bytes).
// The hash should be 32 bytes.
func VerifySignature(pubKey *btcec.PublicKey, hash, sigBytes []byte) (bool, error) {
	if pubKey == nil {
		return false, fmt.Errorf("public key cannot be nil")
	}
	if len(hash) != 32 {
		return false, fmt.Errorf("hash must be 32 bytes, got %d", len(hash))
	}

	sig, err := parseCosmosSignature(sigBytes)
	if err != nil {
		return false, err
	}

	return sig.Verify(hash, pubKey), nil
}

// SerializePublicKey serializes a public key to compressed 33-byte format.
func SerializePublicKey(pubKey *btcec.PublicKey) []byte {
	if pubKey == nil {
		return nil
	}
	return pubKey.SerializeCompressed()
}

// ParsePublicKey deserializes a public key from compressed or uncompressed format.
func ParsePublicKey(data []byte) (*btcec.PublicKey, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("public key data cannot be empty")
	}

	pubKey, err := btcec.ParsePubKey(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}
	return pubKey, nil
}

// ParsePrivateKey deserializes a private key from raw 32-byte format.
func ParsePrivateKey(data []byte) (*btcec.PrivateKey, error) {
	if len(data) != 32 {
		return nil, fmt.Errorf("private key must be 32 bytes, got %d", len(data))
	}

	privKey, _ := btcec.PrivKeyFromBytes(data)
	if privKey == nil {
		return nil, fmt.Errorf("failed to parse private key")
	}
	return privKey, nil
}

// SerializePrivateKey serializes a private key to raw 32-byte format.
func SerializePrivateKey(privKey *btcec.PrivateKey) []byte {
	if privKey == nil {
		return nil
	}
	return privKey.Serialize()
}

// hashSHA256 computes SHA-256.
func hashSHA256(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// hashKeccak256 computes Keccak-256 (Ethereum).
func hashKeccak256(data []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	return h.Sum(nil)
}

// deriveCosmosAddress derives address from compressed public key.
// Formula: RIPEMD160(SHA256(pubkey))
func deriveCosmosAddress(pubKey []byte) []byte {
	sha := sha256.Sum256(pubKey)
	rip := ripemd160.New()
	rip.Write(sha[:])
	return rip.Sum(nil)
}

// formatCosmosSignature formats as R||S (64 bytes) with low-S normalization.
// It takes a DER-encoded signature and converts it to the R||S format used by Cosmos.
func formatCosmosSignature(sig *ecdsa.Signature) []byte {
	// Serialize to DER format and parse R, S
	derBytes := sig.Serialize()

	// DER format: 0x30 [length] 0x02 [r_len] [r] 0x02 [s_len] [s]
	// Extract R and S from DER encoding
	r, s := extractRSFromDER(derBytes)

	// Normalize to low-S (BIP-62)
	// If S > half order, negate it to get the low-S form
	if s.IsOverHalfOrder() {
		s.Negate()
	}

	result := make([]byte, 64)
	r.PutBytesUnchecked(result[:32])
	s.PutBytesUnchecked(result[32:])
	return result
}

// extractRSFromDER extracts R and S values from a DER-encoded ECDSA signature.
func extractRSFromDER(der []byte) (*btcec.ModNScalar, *btcec.ModNScalar) {
	// DER format: 0x30 [total_len] 0x02 [r_len] [r_bytes] 0x02 [s_len] [s_bytes]
	// Skip the sequence tag (0x30) and length byte
	offset := 2

	// Skip R integer tag (0x02)
	offset++
	rLen := int(der[offset])
	offset++

	// Extract R bytes (may have leading zero for positive numbers)
	rBytes := der[offset : offset+rLen]
	offset += rLen

	// Skip S integer tag (0x02)
	offset++
	sLen := int(der[offset])
	offset++

	// Extract S bytes
	sBytes := der[offset : offset+sLen]

	// Convert to ModNScalar (handles leading zeros)
	r := new(btcec.ModNScalar)
	s := new(btcec.ModNScalar)

	// Remove leading zero if present (DER encoding adds 0x00 for positive numbers with high bit set)
	if len(rBytes) == 33 && rBytes[0] == 0 {
		rBytes = rBytes[1:]
	}
	if len(sBytes) == 33 && sBytes[0] == 0 {
		sBytes = sBytes[1:]
	}

	// Pad to 32 bytes if necessary
	rPadded := make([]byte, 32)
	sPadded := make([]byte, 32)
	copy(rPadded[32-len(rBytes):], rBytes)
	copy(sPadded[32-len(sBytes):], sBytes)

	r.SetByteSlice(rPadded)
	s.SetByteSlice(sPadded)

	return r, s
}

// parseCosmosSignature parses R||S format.
func parseCosmosSignature(sigBytes []byte) (*ecdsa.Signature, error) {
	if len(sigBytes) != 64 {
		return nil, fmt.Errorf("signature must be 64 bytes, got %d", len(sigBytes))
	}

	r := new(btcec.ModNScalar)
	s := new(btcec.ModNScalar)

	overflow := r.SetByteSlice(sigBytes[:32])
	if overflow {
		return nil, fmt.Errorf("r value overflows")
	}

	overflow = s.SetByteSlice(sigBytes[32:])
	if overflow {
		return nil, fmt.Errorf("s value overflows")
	}

	// Verify that R and S are not zero
	if r.IsZero() || s.IsZero() {
		return nil, fmt.Errorf("invalid signature: R or S is zero")
	}

	return ecdsa.NewSignature(r, s), nil
}

// secureZero wipes sensitive data from memory.
func secureZero(b []byte) {
	for i := range b {
		b[i] = 0
	}
	runtime.KeepAlive(b)
}
