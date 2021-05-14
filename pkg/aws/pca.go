/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package aws

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acmpca"
	"github.com/aws/aws-sdk-go/service/acmpca/acmpcaiface"
	cmapi "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"k8s.io/apimachinery/pkg/types"
)

var collection = new(sync.Map)

type GenericProvisioner interface {
	Sign(ctx context.Context, cr *cmapi.CertificateRequest) ([]byte, []byte, error)
}

// PCAProvisioner contains logic for issuing PCA certificates
type PCAProvisioner struct {
	pcaClient acmpcaiface.ACMPCAAPI
	arn       string
}

// GetProvisioner gets a provisioner that has previously been stored
func GetProvisioner(name types.NamespacedName) (GenericProvisioner, bool) {
	value, exists := collection.Load(name)
	if !exists {
		return nil, exists
	}

	p, exists := value.(GenericProvisioner)
	return p, exists
}

// StoreProvisioner stores a provisioner in the cache
func StoreProvisioner(name types.NamespacedName, provisioner GenericProvisioner) {
	collection.Store(name, provisioner)
}

// NewProvisioner returns a new PCAProvisioner
func NewProvisioner(session *session.Session, arn string) (p *PCAProvisioner) {
	return &PCAProvisioner{
		pcaClient: acmpca.New(session, &aws.Config{}),
		arn:       arn,
	}
}

// Sign takes a certificate request and signs it using PCA
func (p *PCAProvisioner) Sign(ctx context.Context, cr *cmapi.CertificateRequest) ([]byte, []byte, error) {
	block, _ := pem.Decode(cr.Spec.Request)
	if block == nil {
		return nil, nil, fmt.Errorf("failed to decode CSR")
	}

	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, nil, err
	}

	sigAlgorithm, err := signatureAlgorithm(csr)
	if err != nil {
		return nil, nil, err
	}

	validityDays := int64(30)
	if cr.Spec.Duration != nil {
		validityDays = int64(cr.Spec.Duration.Hours() / 24)
	}

	issueParams := acmpca.IssueCertificateInput{
		CertificateAuthorityArn: aws.String(p.arn),
		SigningAlgorithm:        aws.String(sigAlgorithm),
		Csr:                     cr.Spec.Request,
		Validity: &acmpca.Validity{
			Type:  aws.String(acmpca.ValidityPeriodTypeDays),
			Value: aws.Int64(validityDays),
		},
		IdempotencyToken: aws.String("awspca"),
	}

	issueOutput, err := p.pcaClient.IssueCertificate(&issueParams)

	if err != nil {
		return nil, nil, err
	}

	getParams := acmpca.GetCertificateInput{
		CertificateArn:          aws.String(*issueOutput.CertificateArn),
		CertificateAuthorityArn: aws.String(p.arn),
	}

	err = p.pcaClient.WaitUntilCertificateIssued(&getParams)
	if err != nil {
		return nil, nil, err
	}

	getOutput, err := p.pcaClient.GetCertificate(&getParams)
	if err != nil {
		return nil, nil, err
	}

	certPem := []byte(*getOutput.Certificate + "\n")
	chainPem := []byte(*getOutput.CertificateChain)
	chainIntCAs, rootCA, err := splitRootCACertificate(chainPem)
	if err != nil {
		return nil, nil, err
	}
	certPem = append(certPem, chainIntCAs...)

	return certPem, rootCA, nil
}

func splitRootCACertificate(caCertChainPem []byte) ([]byte, []byte, error) {
	var caChainCerts []byte
	var rootCACert []byte
	for {
		block, rest := pem.Decode(caCertChainPem)
		if block == nil || block.Type != "CERTIFICATE" {
			return nil, nil, fmt.Errorf("failed to read certificate")
		}
		var encBuf bytes.Buffer
		if err := pem.Encode(&encBuf, block); err != nil {
			return nil, nil, err
		}
		if len(rest) > 0 {
			caChainCerts = append(caChainCerts, encBuf.Bytes()...)
			caCertChainPem = rest
		} else {
			rootCACert = append(rootCACert, encBuf.Bytes()...)
			break
		}
	}
	return caChainCerts, rootCACert, nil
}

func signatureAlgorithm(cr *x509.CertificateRequest) (string, error) {
	switch cr.PublicKeyAlgorithm {
	case x509.RSA:
		pubKey, ok := cr.PublicKey.(*rsa.PublicKey)
		if !ok {
			return "", fmt.Errorf("failed to read public key")
		}

		switch {
		case pubKey.N.BitLen() >= 4096:
			return acmpca.SigningAlgorithmSha512withrsa, nil
		case pubKey.N.BitLen() >= 3072:
			return acmpca.SigningAlgorithmSha384withrsa, nil
		case pubKey.N.BitLen() >= 2048:
			return acmpca.SigningAlgorithmSha256withrsa, nil
		case pubKey.N.BitLen() == 0:
			return acmpca.SigningAlgorithmSha256withrsa, nil
		default:
			return "", fmt.Errorf("unsupported rsa keysize specified: %d", pubKey.N.BitLen())
		}
	case x509.ECDSA:
		pubKey, ok := cr.PublicKey.(*ecdsa.PublicKey)
		if !ok {
			return "", fmt.Errorf("failed to read public key")
		}

		switch pubKey.Curve.Params().BitSize {
		case 521:
			return acmpca.SigningAlgorithmSha512withecdsa, nil
		case 384:
			return acmpca.SigningAlgorithmSha384withecdsa, nil
		case 256:
			return acmpca.SigningAlgorithmSha256withecdsa, nil
		case 0:
			return acmpca.SigningAlgorithmSha256withecdsa, nil
		default:
			return "", fmt.Errorf("unsupported ecdsa keysize specified: %d", pubKey.Curve.Params().BitSize)
		}

	default:
		return "", fmt.Errorf("unsupported public key algorithm: %v", cr.PublicKeyAlgorithm)
	}
}
