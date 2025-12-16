// Package auth provides authentication middleware for the RPC Gateway.
package auth

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/Bidon15/popsigner/control-plane/internal/models"
	"github.com/Bidon15/popsigner/control-plane/internal/repository"
)

// MTLSAuthenticator validates mTLS client certificates.
type MTLSAuthenticator struct {
	certRepo repository.CertificateRepository
}

// NewMTLSAuthenticator creates a new mTLS authenticator.
func NewMTLSAuthenticator(certRepo repository.CertificateRepository) *MTLSAuthenticator {
	return &MTLSAuthenticator{certRepo: certRepo}
}

// AuthResult contains the result of authentication.
type AuthResult struct {
	OrgID      string
	Method     string // "api_key" or "mtls"
	Identifier string // API key prefix or cert fingerprint (truncated)
}

// Authenticate validates a client certificate and returns the organization ID.
func (a *MTLSAuthenticator) Authenticate(ctx context.Context, r *http.Request) (*AuthResult, error) {
	// Check if TLS connection has peer certificates
	if r.TLS == nil {
		return nil, fmt.Errorf("no TLS connection")
	}

	if len(r.TLS.PeerCertificates) == 0 {
		return nil, fmt.Errorf("no client certificate provided")
	}

	cert := r.TLS.PeerCertificates[0]

	// Calculate fingerprint
	fingerprint := CalculateCertFingerprint(cert)

	// Validate certificate in database
	orgID, err := a.validateCertificate(ctx, fingerprint, cert)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		OrgID:      orgID,
		Method:     "mtls",
		Identifier: fingerprint[:16] + "...", // Truncated for logging
	}, nil
}

// validateCertificate checks if the certificate is valid and returns the org ID.
func (a *MTLSAuthenticator) validateCertificate(ctx context.Context, fingerprint string, cert *x509.Certificate) (string, error) {
	// Look up certificate in database
	dbCert, err := a.certRepo.GetByFingerprint(ctx, fingerprint)
	if err != nil {
		return "", fmt.Errorf("database error: %w", err)
	}

	if dbCert == nil {
		return "", fmt.Errorf("certificate not registered")
	}

	// Check if revoked
	if dbCert.IsRevoked() {
		return "", fmt.Errorf("certificate has been revoked")
	}

	// Check if expired
	if dbCert.IsExpired() {
		return "", fmt.Errorf("certificate has expired")
	}

	// Extract org ID from CN and verify it matches database record
	orgID, err := models.OrgIDFromCN(cert.Subject.CommonName)
	if err != nil {
		return "", fmt.Errorf("invalid certificate CN: %w", err)
	}

	if orgID != dbCert.OrgID.String() {
		return "", fmt.Errorf("certificate CN does not match registered organization")
	}

	return dbCert.OrgID.String(), nil
}

// CalculateCertFingerprint computes SHA256 fingerprint of a certificate.
func CalculateCertFingerprint(cert *x509.Certificate) string {
	hash := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(hash[:])
}

// GetTLSConfig returns TLS config for the server with client certificate verification.
func GetTLSConfig(caCertPEM []byte, clientAuthType tls.ClientAuthType) (*tls.Config, error) {
	// Create CA certificate pool
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return &tls.Config{
		ClientAuth: clientAuthType,
		ClientCAs:  caPool,
		MinVersion: tls.VersionTLS12,
	}, nil
}

// ParseClientAuthType parses a string into tls.ClientAuthType.
func ParseClientAuthType(s string) tls.ClientAuthType {
	switch s {
	case "NoClientCert":
		return tls.NoClientCert
	case "RequestClientCert":
		return tls.RequestClientCert
	case "RequireAnyClientCert":
		return tls.RequireAnyClientCert
	case "VerifyClientCertIfGiven":
		return tls.VerifyClientCertIfGiven
	case "RequireAndVerifyClientCert":
		return tls.RequireAndVerifyClientCert
	default:
		// Default: verify if provided (supports both API key and mTLS)
		return tls.VerifyClientCertIfGiven
	}
}

