package aws

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/service/acmpca"
	"github.com/aws/aws-sdk-go/service/acmpca/acmpcaiface"
	v1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
)

var (
	arn      = "arn:aws:acm-pca:us-east-1:account:certificate-authority/12345678-1234-1234-1234-123456789012"
	template = x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:         "domain.com",
			Country:            []string{"US"},
			Province:           []string{"Some-State"},
			Locality:           []string{"MyCity"},
			Organization:       []string{"Company Ltd"},
			OrganizationalUnit: []string{"IT"},
		},
	}
	certArn = "arn"
	cert    = "cert"
	chain   = "chain"
	cacert  = "cacert"
)

type errorACMPCAClient struct {
	acmpcaiface.ACMPCAAPI
}

func (m *errorACMPCAClient) IssueCertificate(input *acmpca.IssueCertificateInput) (*acmpca.IssueCertificateOutput, error) {
	return nil, errors.New("Cannot issue certificate")
}

type workingACMPCAClient struct {
	acmpcaiface.ACMPCAAPI
}

func (m *workingACMPCAClient) IssueCertificate(input *acmpca.IssueCertificateInput) (*acmpca.IssueCertificateOutput, error) {
	return &acmpca.IssueCertificateOutput{CertificateArn: &certArn}, nil
}

func (m *workingACMPCAClient) WaitUntilCertificateIssued(input *acmpca.GetCertificateInput) error {
	return nil
}

func (m *workingACMPCAClient) GetCertificate(input *acmpca.GetCertificateInput) (*acmpca.GetCertificateOutput, error) {
	return &acmpca.GetCertificateOutput{Certificate: &cert, CertificateChain: &chain}, nil
}

func (m *workingACMPCAClient) GetCertificateAuthorityCertificate(input *acmpca.GetCertificateAuthorityCertificateInput) (*acmpca.GetCertificateAuthorityCertificateOutput, error) {
	return &acmpca.GetCertificateAuthorityCertificateOutput{Certificate: &cacert}, nil
}
func TestPCASignatureAlgorithm(t *testing.T) {
	type createKey func() (priv interface{})

	type testCase struct {
		expectedAlgorithm string
		createKeyFun      createKey
	}
	tests := map[string]testCase{
		"success-RSA-2048": {
			expectedAlgorithm: acmpca.SigningAlgorithmSha256withrsa,
			createKeyFun: func() (priv interface{}) {
				keyBytes, _ := rsa.GenerateKey(rand.Reader, 2048)
				return keyBytes
			},
		},
		"success-RSA-3072": {
			expectedAlgorithm: acmpca.SigningAlgorithmSha384withrsa,
			createKeyFun: func() (priv interface{}) {
				keyBytes, _ := rsa.GenerateKey(rand.Reader, 3072)
				return keyBytes
			},
		},
		"success-RSA-4096": {
			expectedAlgorithm: acmpca.SigningAlgorithmSha512withrsa,
			createKeyFun: func() (priv interface{}) {
				keyBytes, _ := rsa.GenerateKey(rand.Reader, 4096)
				return keyBytes
			},
		},
		"success-ECDSA-521": {
			expectedAlgorithm: acmpca.SigningAlgorithmSha512withecdsa,
			createKeyFun: func() (priv interface{}) {
				keyBytes, _ := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
				return keyBytes
			},
		},
		"success-ECDSA-384": {
			expectedAlgorithm: acmpca.SigningAlgorithmSha384withecdsa,
			createKeyFun: func() (priv interface{}) {
				keyBytes, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
				return keyBytes
			},
		},
		"success-ECDSA-256": {
			expectedAlgorithm: acmpca.SigningAlgorithmSha256withecdsa,
			createKeyFun: func() (priv interface{}) {
				keyBytes, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
				return keyBytes
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			csrBytes, _ := x509.CreateCertificateRequest(rand.Reader, &template, tc.createKeyFun())
			csr, _ := x509.ParseCertificateRequest(csrBytes)
			response, _ := signatureAlgorithm(csr)

			if tc.expectedAlgorithm != response {
				assert.Fail(t, "Expected type "+tc.expectedAlgorithm+" but got "+response)
			}
		})
	}
}

func TestPCASign(t *testing.T) {

	type testCase struct {
		provisioner    PCAProvisioner
		expectFailure  bool
		expectedCaCert string
		expectedCert   string
	}

	tests := map[string]testCase{
		"success": {
			provisioner:    PCAProvisioner{arn: arn, pcaClient: &workingACMPCAClient{}},
			expectFailure:  false,
			expectedCaCert: cacert,
			expectedCert:   string(append([]byte(cert+"\n"), []byte(chain)...)),
		},
		"failure-error-issueCertificate": {
			provisioner:   PCAProvisioner{arn: arn, pcaClient: &errorACMPCAClient{}},
			expectFailure: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			key, _ := rsa.GenerateKey(rand.Reader, 2048)
			csrBytes, _ := x509.CreateCertificateRequest(rand.Reader, &template, key)

			cr := &v1.CertificateRequest{
				Spec: v1.CertificateRequestSpec{
					Request: pem.EncodeToMemory(&pem.Block{
						Bytes: csrBytes,
						Type:  "CERTIFICATE REQUEST",
					}),
				},
			}

			leaf, ca, err := tc.provisioner.Sign(context.TODO(), cr)

			if tc.expectFailure && err == nil {
				fmt.Print(err.Error())
				assert.Fail(t, "Expected an error but received none")
			}

			if tc.expectedCaCert != "" && tc.expectedCert != "" {
				assert.Equal(t, []byte(tc.expectedCert), leaf)
				assert.Equal(t, []byte(tc.expectedCaCert), ca)
			}

		})
	}
}
