/*
Copyright 2021.

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
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acmpca"
	cmapi "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"k8s.io/apimachinery/pkg/types"
	"sync"
)

var collection = new(sync.Map)

func GetProvisioner(name types.NamespacedName) (*PCAProvisioner, bool) {
	value, exists := collection.Load(name)
	if !exists {
		return nil, exists
	}
	p, exists := value.(*PCAProvisioner)
	return p, exists
}

func StoreProvisioner(name types.NamespacedName, provisioner *PCAProvisioner) {
	collection.Store(name, provisioner)
}

type PCAProvisioner struct {
	session *session.Session
	arn     string
}

func NewProvisioner(session *session.Session, arn string) (p *PCAProvisioner) {
	return &PCAProvisioner{
		session: session,
		arn:     arn,
	}
}

func (p *PCAProvisioner) Sign(ctx context.Context, cr *cmapi.CertificateRequest) ([]byte, []byte, error) {
	svc := acmpca.New(p.session, &aws.Config{})

	validityDays := int64(30)
	if cr.Spec.Duration != nil {
		validityDays = int64(cr.Spec.Duration.Hours() / 24)
	}

	issueParams := acmpca.IssueCertificateInput{
		CertificateAuthorityArn: aws.String(p.arn),
		SigningAlgorithm:        aws.String(acmpca.SigningAlgorithmSha256withrsa),
		Csr:                     cr.Spec.Request,
		Validity: &acmpca.Validity{
			Type:  aws.String(acmpca.ValidityPeriodTypeDays),
			Value: aws.Int64(validityDays),
		},
		IdempotencyToken: aws.String("awspca"),
	}

	issueOutput, err := svc.IssueCertificate(&issueParams)

	if err != nil {
		return nil, nil, err
	}

	getParams := acmpca.GetCertificateInput{
		CertificateArn:          aws.String(*issueOutput.CertificateArn),
		CertificateAuthorityArn: aws.String(p.arn),
	}

	svc.WaitUntilCertificateIssued(&getParams)

	getOutput, getError := svc.GetCertificate(&getParams)

	if getError != nil {
		return nil, nil, getError
	}

	certPem := []byte(*getOutput.Certificate + "\n")
	chainPem := []byte(*getOutput.CertificateChain)
	certPem = append(certPem, chainPem...)

	caParams := acmpca.GetCertificateAuthorityCertificateInput{
		CertificateAuthorityArn: aws.String(p.arn),
	}
	caOutput, caError := svc.GetCertificateAuthorityCertificate(&caParams)
	if caError != nil {
		return nil, nil, caError
	}

	caPem := []byte(*caOutput.Certificate)

	return certPem, caPem, nil
}
