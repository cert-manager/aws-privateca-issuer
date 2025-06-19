package main

import (
	"testing"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
)

func TestGetBaseCertSpec(t *testing.T) {
	// Test default usage
	certSpec := getBaseCertSpec("RSA")
	assert.Equal(t, []cmv1.KeyUsage{cmv1.UsageClientAuth}, certSpec.Usages)

	// Test with custom usage
	certSpec = getBaseCertSpec("RSA", cmv1.UsageServerAuth)
	assert.Equal(t, []cmv1.KeyUsage{cmv1.UsageServerAuth}, certSpec.Usages)

	// Test with multiple usages
	certSpec = getBaseCertSpec("RSA", cmv1.UsageClientAuth, cmv1.UsageServerAuth)
	assert.Equal(t, []cmv1.KeyUsage{cmv1.UsageClientAuth, cmv1.UsageServerAuth}, certSpec.Usages)
}
