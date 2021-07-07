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
	"crypto/md5"
	"encoding/pem"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acmpca"
	acmpcatypes "github.com/aws/aws-sdk-go-v2/service/acmpca/types"
	cmapi "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"k8s.io/apimachinery/pkg/types"
)

var collection = new(sync.Map)

// GenericProvisioner abstracts over the Provisioner type for mocking purposes
type GenericProvisioner interface {
	Sign(ctx context.Context, cr *cmapi.CertificateRequest) ([]byte, []byte, error)
}

// acmPCAClient abstracts over the methods used from acmpca.Client
type acmPCAClient interface {
	acmpca.GetCertificateAPIClient
	DescribeCertificateAuthority(ctx context.Context, params *acmpca.DescribeCertificateAuthorityInput, optFns ...func(*acmpca.Options)) (*acmpca.DescribeCertificateAuthorityOutput, error)
	IssueCertificate(ctx context.Context, params *acmpca.IssueCertificateInput, optFns ...func(*acmpca.Options)) (*acmpca.IssueCertificateOutput, error)
}

// PCAProvisioner contains logic for issuing PCA certificates
type PCAProvisioner struct {
	pcaClient        acmPCAClient
	arn              string
	signingAlgorithm *acmpcatypes.SigningAlgorithm
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
func NewProvisioner(config aws.Config, arn string) (p *PCAProvisioner) {
	return &PCAProvisioner{
		pcaClient: acmpca.NewFromConfig(config),
		arn:       arn,
	}
}

// idempotencyToken is limited to 64 ASCII characters, so make a fixed length hash.
// @see: https://docs.aws.amazon.com/AWSEC2/latest/APIReference/Run_Instance_Idempotency.html
func idempotencyToken(cr *cmapi.CertificateRequest) string {
	token := []byte(cr.ObjectMeta.Namespace + "/" + cr.ObjectMeta.Name)
	return fmt.Sprintf("%x", md5.Sum(token))
}

// Sign takes a certificate request and signs it using PCA
func (p *PCAProvisioner) Sign(ctx context.Context, cr *cmapi.CertificateRequest) ([]byte, []byte, error) {
	block, _ := pem.Decode(cr.Spec.Request)
	if block == nil {
		return nil, nil, fmt.Errorf("failed to decode CSR")
	}

	validityDays := int64(30)
	if cr.Spec.Duration != nil {
		validityDays = int64(cr.Spec.Duration.Hours() / 24)
	}

	tempArn := templateArn(p.arn, cr.Spec)

	// Consider it a "retry" if we try to re-create a cert with the same name in the same namespace
	token := idempotencyToken(cr)

	err := getSigningAlgorithm(ctx, p)
	if err != nil {
		return nil, nil, err
	}

	issueParams := acmpca.IssueCertificateInput{
		CertificateAuthorityArn: aws.String(p.arn),
		SigningAlgorithm:        *p.signingAlgorithm,
		TemplateArn:             aws.String(tempArn),
		Csr:                     cr.Spec.Request,
		Validity: &acmpcatypes.Validity{
			Type:  acmpcatypes.ValidityPeriodTypeDays,
			Value: &validityDays,
		},
		IdempotencyToken: aws.String(token),
	}

	issueOutput, err := p.pcaClient.IssueCertificate(ctx, &issueParams)

	if err != nil {
		return nil, nil, err
	}

	getParams := acmpca.GetCertificateInput{
		CertificateArn:          aws.String(*issueOutput.CertificateArn),
		CertificateAuthorityArn: aws.String(p.arn),
	}

	waiter := acmpca.NewCertificateIssuedWaiter(p.pcaClient)
	err = waiter.Wait(ctx, &getParams, 5*time.Minute)
	if err != nil {
		return nil, nil, err
	}

	getOutput, err := p.pcaClient.GetCertificate(ctx, &getParams)
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

func getSigningAlgorithm(ctx context.Context, p *PCAProvisioner) error {
	if p.signingAlgorithm != nil {
		return nil
	}

	describeParams := acmpca.DescribeCertificateAuthorityInput{
		CertificateAuthorityArn: aws.String(p.arn),
	}
	describeOutput, err := p.pcaClient.DescribeCertificateAuthority(ctx, &describeParams)

	if err != nil {
		return err
	}

	p.signingAlgorithm = &describeOutput.CertificateAuthority.CertificateAuthorityConfiguration.SigningAlgorithm
	return nil
}

func templateArn(caArn string, spec cmapi.CertificateRequestSpec) string {
	arn := strings.SplitAfterN(caArn, ":", 3)
	prefix := arn[0] + arn[1]

	if len(spec.Usages) == 1 {
		switch spec.Usages[0] {
		case cmapi.UsageCodeSigning:
			return prefix + "acm-pca:::template/CodeSigningCertificate/V1"
		case cmapi.UsageClientAuth:
			return prefix + "acm-pca:::template/EndEntityClientAuthCertificate/V1"
		case cmapi.UsageServerAuth:
			return prefix + "acm-pca:::template/EndEntityServerAuthCertificate/V1"
		case cmapi.UsageOCSPSigning:
			return prefix + "acm-pca:::template/OCSPSigningCertificate/V1"
		}
	} else if len(spec.Usages) == 2 {
		clientServer := (spec.Usages[0] == cmapi.UsageClientAuth && spec.Usages[1] == cmapi.UsageServerAuth)
		serverClient := (spec.Usages[0] == cmapi.UsageServerAuth && spec.Usages[1] == cmapi.UsageClientAuth)
		if clientServer || serverClient {
			return prefix + "acm-pca:::template/EndEntityCertificate/V1"
		}
	}

	return prefix + "acm-pca:::template/BlankEndEntityCertificate_CSRPassthrough/V1"
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
