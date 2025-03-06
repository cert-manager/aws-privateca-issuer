package controllers

import (
	"testing"

	api "github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
)

// TestValidateIssuer ensures the issuer validation works correctly
func TestValidateIssuer(t *testing.T) {
	tests := []struct {
		name      string
		spec      *api.AWSPCAIssuerSpec
		expectErr bool
	}{
		{
			name: "valid spec with role",
			spec: &api.AWSPCAIssuerSpec{
				Arn:    "arn:aws:acm-pca:us-west-2:123456789012:certificate-authority/12345678-1234-1234-1234-123456789012",
				Region: "us-west-2",
				Role:   "arn:aws:iam::123456789012:role/test-role",
			},
			expectErr: false,
		},
		{
			name: "valid spec without role",
			spec: &api.AWSPCAIssuerSpec{
				Arn:    "arn:aws:acm-pca:us-west-2:123456789012:certificate-authority/12345678-1234-1234-1234-123456789012",
				Region: "us-west-2",
			},
			expectErr: false,
		},
		{
			name: "missing ARN",
			spec: &api.AWSPCAIssuerSpec{
				Region: "us-west-2",
				Role:   "arn:aws:iam::123456789012:role/test-role",
			},
			expectErr: true,
		},
		{
			name: "missing region",
			spec: &api.AWSPCAIssuerSpec{
				Arn:  "arn:aws:acm-pca:us-west-2:123456789012:certificate-authority/12345678-1234-1234-1234-123456789012",
				Role: "arn:aws:iam::123456789012:role/test-role",
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIssuer(tt.spec)
			if (err != nil) != tt.expectErr {
				t.Errorf("validateIssuer() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}
