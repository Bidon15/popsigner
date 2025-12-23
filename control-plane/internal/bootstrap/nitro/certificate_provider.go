// Package nitro provides Nitro/Orbit chain deployment orchestration.
package nitro

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Bidon15/popsigner/control-plane/internal/models"
	"github.com/Bidon15/popsigner/control-plane/internal/service"
)

// CertificateServiceProvider adapts the CertificateService to the CertificateProvider interface.
// It auto-issues a new certificate for each deployment, since private keys are not stored.
type CertificateServiceProvider struct {
	certService service.CertificateService
}

// NewCertificateServiceProvider creates a new certificate provider backed by CertificateService.
func NewCertificateServiceProvider(certService service.CertificateService) *CertificateServiceProvider {
	return &CertificateServiceProvider{
		certService: certService,
	}
}

// GetCertificates issues a new deployment certificate for the organization.
// Since private keys are not stored (for security), we issue a fresh certificate
// for each deployment. This certificate is short-lived (24h for deployments).
func (p *CertificateServiceProvider) GetCertificates(ctx context.Context, orgID uuid.UUID) (*CertificateBundle, error) {
	if p.certService == nil {
		return nil, fmt.Errorf("certificate service not configured")
	}

	// Issue a short-lived certificate for deployment
	// Name includes timestamp to allow multiple deployments
	certName := fmt.Sprintf("deployment-%s", time.Now().Format("20060102-150405"))

	req := &models.CreateCertificateRequest{
		OrgID:          orgID,
		Name:           certName,
		ValidityPeriod: 24 * time.Hour, // Short-lived for deployments
	}

	bundle, err := p.certService.Issue(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("issue deployment certificate: %w", err)
	}

	return &CertificateBundle{
		ClientCert: string(bundle.ClientCert),
		ClientKey:  string(bundle.ClientKey),
		CaCert:     string(bundle.CACert),
	}, nil
}

// Verify CertificateServiceProvider implements CertificateProvider
var _ CertificateProvider = (*CertificateServiceProvider)(nil)

