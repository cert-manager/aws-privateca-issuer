package main

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"slices"
	"strings"
	"time"

	util "github.com/cert-manager/cert-manager/pkg/api/util"
	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/cucumber/godog"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var usageMap = map[string]cmv1.KeyUsage{
	"client_auth":  cmv1.UsageClientAuth,
	"server_auth":  cmv1.UsageServerAuth,
	"code_signing": cmv1.UsageCodeSigning,
	"ocsp_signing": cmv1.UsageOCSPSigning,
}

func getCaArn(caType string) string {
	caArn, exists := testContext.caArns[caType]

	if !exists {
		panic(fmt.Sprintf("Unknown CA Type: %s", caType))
	}

	return caArn
}

func getIssuerSpec(caType string) v1beta1.AWSPCAIssuerSpec {
	return v1beta1.AWSPCAIssuerSpec{
		Arn:    getCaArn(caType),
		Region: testContext.region,
	}
}

func getIssuerSpecWithRole(caType string) v1beta1.AWSPCAIssuerSpec {
	spec := getIssuerSpec(caType)
	spec.Role = testContext.roleToAssume
	return spec
}

func (issCtx *IssuerContext) createNamespace(ctx context.Context) error {
	namespaceName := "pca-issuer-ns-" + uuid.New().String()
	namespace := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespaceName},
	}

	_, err := testContext.clientset.CoreV1().Namespaces().Create(ctx, &namespace, metav1.CreateOptions{})

	if err != nil {
		assert.FailNow(godog.T(ctx), "Failed to create namespace "+err.Error())
	}

	issCtx.namespace = namespaceName

	return nil
}

func (issCtx *IssuerContext) createSecret(ctx context.Context, accessKey string, secretKey string) error {
	secretName := "pca-issuer-secret-" + uuid.New().String()

	data := make(map[string][]byte)
	data[accessKey] = []byte(testContext.accessKey)
	data[secretKey] = []byte(testContext.secretKey)

	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName},
		Data:       data,
	}

	_, err := testContext.clientset.CoreV1().Secrets(issCtx.namespace).Create(ctx, &secret, metav1.CreateOptions{})

	if err != nil {
		assert.FailNow(godog.T(ctx), "Failed to create issuer secret "+err.Error())
	}

	awsSecretRef := v1beta1.AWSCredentialsSecretReference{
		SecretReference: v1.SecretReference{
			Name:      secretName,
			Namespace: issCtx.namespace,
		},
	}

	if accessKey != "AWS_ACCESS_KEY_ID" {
		awsSecretRef.AccessKeyIDSelector = v1.SecretKeySelector{
			Key: accessKey,
		}
	}

	if secretKey != "AWS_SECRET_ACCESS_KEY" {
		awsSecretRef.SecretAccessKeySelector = v1.SecretKeySelector{
			Key: secretKey,
		}
	}

	issCtx.secretRef = awsSecretRef
	return nil
}

func getBaseCertSpec(certType string) cmv1.CertificateSpec {
	sanitizedCertType := strings.Replace(strings.ToLower(certType), "_", "-", -1)
	certSpec := cmv1.CertificateSpec{
		Subject: &cmv1.X509Subject{
			Organizations: []string{"aws"},
		},
		DNSNames: []string{sanitizedCertType + "-cert.aws.com"},
		Duration: &metav1.Duration{
			Duration: 721 * time.Hour,
		},
		Usages: []cmv1.KeyUsage{cmv1.UsageClientAuth, cmv1.UsageServerAuth},
	}

	if certType == "RSA" {
		certSpec.PrivateKey = &cmv1.CertificatePrivateKey{
			Algorithm: cmv1.RSAKeyAlgorithm,
			Size:      2048,
		}
	}

	if certType == "ECDSA" {
		certSpec.PrivateKey = &cmv1.CertificatePrivateKey{
			Algorithm: cmv1.ECDSAKeyAlgorithm,
			Size:      256,
		}
	}

	return certSpec
}

func getCertSpec(certType string) cmv1.CertificateSpec {
	switch certType {
	case "RSA":
		return getBaseCertSpec(certType)
	case "ECDSA":
		return getBaseCertSpec(certType)

	// For simplicity, we use RSA as the base for these. This can be further generalized if desired.
	case "SHORT_VALIDITY":
		return getCertSpecWithValidity(getBaseCertSpec("RSA"), 20, 5)
	case "CA":
		return getCaCertSpec(getBaseCertSpec("RSA"))
	default:
		panic(fmt.Sprintf("Unknown Certificate Type: %s", certType))
	}
}

func getCertSpecWithValidity(certSpec cmv1.CertificateSpec, duration time.Duration, renewBefore time.Duration) cmv1.CertificateSpec {
	certSpec.Duration = &metav1.Duration{
		Duration: duration * time.Hour,
	}
	certSpec.RenewBefore = &metav1.Duration{
		Duration: renewBefore * time.Hour,
	}

	return certSpec
}

func getCaCertSpec(certSpec cmv1.CertificateSpec) cmv1.CertificateSpec {
	certSpec.IsCA = true
	return getCertSpecWithValidity(certSpec, 20, 5)
}

func (issCtx *IssuerContext) issueCertificate(ctx context.Context, certType string) error {
	return issCtx.issueCertificateWithUsage(ctx, certType, "")
}

func (issCtx *IssuerContext) issueCertificateWithUsage(ctx context.Context, certType string, usage string) error {
	sanitizedCertType := strings.Replace(strings.ToLower(certType), "_", "-", -1)
	issCtx.certName = issCtx.issuerName + "-" + sanitizedCertType + "-cert"
	certSpec := getCertSpec(certType)

	if usage != "" && usage != "any" {
		var usages []cmv1.KeyUsage
		for _, u := range strings.Split(usage, ",") {
			if mapped, ok := usageMap[u]; ok {
				usages = append(usages, mapped)
			}
		}
		certSpec.Usages = usages
	}

	secretName := issCtx.certName + "-cert-secret"
	certSpec.SecretName = secretName
	certSpec.IssuerRef = cmmeta.ObjectReference{
		Kind:  issCtx.issuerType,
		Group: "awspca.cert-manager.io",
		Name:  issCtx.issuerName,
	}

	certificate := cmv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: issCtx.certName},
		Spec:       certSpec,
	}

	_, err := testContext.cmClient.Certificates(issCtx.namespace).Create(ctx, &certificate, metav1.CreateOptions{})

	if err != nil {
		assert.FailNow(godog.T(ctx), "Could not create certificate: "+err.Error())
	}

	return nil
}

func (issCtx *IssuerContext) verifyCertificateIssued(ctx context.Context) error {
	return issCtx.verifyCertificateState(ctx, "Ready", "True")
}

func (issCtx *IssuerContext) verifyCertificateState(ctx context.Context, reason string, status string) error {
	err := waitForCertificateState(ctx, testContext.cmClient, issCtx.certName, issCtx.namespace, reason, status)

	if err != nil {
		assert.FailNow(godog.T(ctx), "Certificate did not reach specified state, Reason = "+reason+", Status = "+status+": "+err.Error())
	}

	return nil
}

func (issCtx *IssuerContext) verifyCertificateRequestIsCreated(ctx context.Context) error {
	certificateRequestName := fmt.Sprintf("%s-%d", issCtx.certName, 1)
	err := waitForCertificateRequestToBeCreated(ctx, testContext.cmClient, certificateRequestName, issCtx.namespace)

	if err != nil {
		assert.FailNow(godog.T(ctx), "Certificate Request did not create successfully: "+err.Error())
	}

	return nil
}

func (issCtx *IssuerContext) verifyCertificateRequestState(ctx context.Context, reason string, status string) error {
	certificateRequestName := fmt.Sprintf("%s-%d", issCtx.certName, 1)
	err := waitForCertificateRequestState(ctx, testContext.cmClient, certificateRequestName, issCtx.namespace, reason, status)

	if err != nil {
		assert.FailNow(godog.T(ctx), "Certificate Request did not reach specified state, Condition = "+reason+", Status = "+status+": "+err.Error())
	}

	return nil
}

func (issCtx *IssuerContext) parseCertificateSecret(ctx context.Context) *x509.Certificate {
	secretName := issCtx.certName + "-cert-secret"
	certData, err := getCertificateData(ctx, testContext.clientset, issCtx.namespace, secretName)
	if err != nil {
		assert.FailNow(godog.T(ctx), "Failed to get certificate data: "+err.Error())
	}

	block, _ := pem.Decode(certData)
	if block == nil {
		assert.FailNow(godog.T(ctx), "Failed to decode PEM block from certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		assert.FailNow(godog.T(ctx), "Failed to parse certificate: "+err.Error())
	}

	return cert
}

func (issCtx *IssuerContext) verifyCertificateUsage(ctx context.Context, usage string) error {
	cert := issCtx.parseCertificateSecret(ctx)

	var expectedX509Usages []x509.ExtKeyUsage
	for _, expectedUsage := range strings.Split(usage, ",") {
		mappedUsage, exists := usageMap[expectedUsage]
		if !exists {
			assert.FailNow(godog.T(ctx), "Expected usage %q not found in usageMap.", expectedUsage)
		}

		x509Usage, _ := util.ExtKeyUsageType(mappedUsage)
		expectedX509Usages = append(expectedX509Usages, x509Usage)
		if !slices.Contains(cert.ExtKeyUsage, x509Usage) {
			assert.FailNow(godog.T(ctx), fmt.Sprintf("Certificate usage mismatch. Found: %v, Expected: %v", cert.ExtKeyUsage, mappedUsage))
		}
	}

	if len(cert.ExtKeyUsage) != len(expectedX509Usages) {
		assert.FailNow(godog.T(ctx), fmt.Sprintf("Certificate key usage mismatch. Found: %v, Expected: %v", cert.ExtKeyUsage, expectedX509Usages))
	}

	return nil
}

func (issCtx *IssuerContext) verifyCertificateAuthorityPathLen(ctx context.Context, pathLen int) error {
	cert := issCtx.parseCertificateSecret(ctx)

	if !cert.IsCA {
		assert.FailNow(godog.T(ctx), "Certificate is not a CA certificate")
	}
	if cert.MaxPathLen != pathLen {
		assert.FailNow(godog.T(ctx), fmt.Sprintf("Expected pathLen %d but got %d", pathLen, cert.MaxPathLen))
	}

	return nil
}
