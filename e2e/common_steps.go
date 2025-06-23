package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/cucumber/godog"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"log"

	"crypto/x509"
	"encoding/pem"

	"github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CertificateRequest struct {
	Ctx      context.Context
	CertType string
	Usage    []cmv1.KeyUsage // optional, will be nil if not provided
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

func getBaseCertSpec(certReq CertificateRequest) cmv1.CertificateSpec {
	sanitizedCertType := strings.Replace(strings.ToLower(certReq.CertType), "_", "-", -1)

	if len(certReq.Usage) == 0 {
		certReq.Usage = []cmv1.KeyUsage{cmv1.UsageAny}
	}

	certSpec := cmv1.CertificateSpec{
		Subject: &cmv1.X509Subject{
			Organizations: []string{"aws"},
		},
		DNSNames: []string{sanitizedCertType + "-cert.aws.com"},
		Duration: &metav1.Duration{
			Duration: 721 * time.Hour,
		},
		Usages: certReq.Usage,
	}

	if certReq.CertType == "RSA" {
		certSpec.PrivateKey = &cmv1.CertificatePrivateKey{
			Algorithm: cmv1.RSAKeyAlgorithm,
			Size:      2048,
		}
	}

	if certReq.CertType == "ECDSA" {
		certSpec.PrivateKey = &cmv1.CertificatePrivateKey{
			Algorithm: cmv1.ECDSAKeyAlgorithm,
			Size:      256,
		}
	}

	return certSpec
}

func getCertSpec(certReq CertificateRequest) cmv1.CertificateSpec {
	switch certReq.CertType {
	case "RSA":
		return getBaseCertSpec(certReq)
	case "ECDSA":
		return getBaseCertSpec(certReq)
	case "SHORT_VALIDITY":
		return getCertSpecWithValidity(getBaseCertSpec(certReq), 20, 5, certReq.Usage...)
	case "CA":
		return getCaCertSpec(getBaseCertSpec(certReq))
	default:
		panic(fmt.Sprintf("Unknown Certificate Type: %s", certReq.CertType))
	}
}

func getCertSpecWithValidity(certSpec cmv1.CertificateSpec, duration time.Duration, renewBefore time.Duration, usages ...cmv1.KeyUsage) cmv1.CertificateSpec {
	certSpec.Duration = &metav1.Duration{
		Duration: duration * time.Hour,
	}
	certSpec.RenewBefore = &metav1.Duration{
		Duration: renewBefore * time.Hour,
	}

	certSpec.Usages = usages

	return certSpec
}

func getCaCertSpec(certSpec cmv1.CertificateSpec) cmv1.CertificateSpec {
	certSpec.IsCA = true
	return getCertSpecWithValidity(certSpec, 20, 5)
}

func (issCtx *IssuerContext) issueCertificateWithoutUsage(ctx context.Context, certType string) error {
	certReq := CertificateRequest{
		Ctx:      ctx,
		CertType: certType,
		Usage:    nil,
	}
	return issCtx.issueCertificate(certReq)
}

func (issCtx *IssuerContext) issueCertificateWithUsage(ctx context.Context, certType string, usageStr string) error {
	usages := parseUsages(usageStr)
	certReq := CertificateRequest{
		Ctx:      ctx,
		CertType: certType,
		Usage:    usages,
	}
	return issCtx.issueCertificate(certReq)
}

func (issCtx *IssuerContext) issueCertificate(certReq CertificateRequest) error {
	sanitizedCertType := strings.Replace(strings.ToLower(certReq.CertType), "_", "-", -1)
	issCtx.certName = issCtx.issuerName + "-" + sanitizedCertType + "-cert"
	certSpec := getCertSpec(certReq)

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

	_, err := testContext.cmClient.Certificates(issCtx.namespace).Create(certReq.Ctx, &certificate, metav1.CreateOptions{})

	if err != nil {
		assert.FailNow(godog.T(certReq.Ctx), "Could not create certificate: "+err.Error())
	}

	return nil
}

func parseUsages(usageStr string) []cmv1.KeyUsage {
	usageMap := map[string]cmv1.KeyUsage{
		"client_auth":  cmv1.UsageClientAuth,
		"server_auth":  cmv1.UsageServerAuth,
		"code_signing": cmv1.UsageCodeSigning,
		"ocsp_signing": cmv1.UsageOCSPSigning,
		"any":          cmv1.UsageAny,
	}

	parts := strings.Split(usageStr, ",")
	var usages []cmv1.KeyUsage
	for _, part := range parts {
		if usage, exists := usageMap[strings.ToLower(part)]; exists {
			usages = append(usages, usage)
		} else {
			assert.FailNow(godog.T(context.Background()), "Unknown usage: "+part)
		}
	}

	return usages
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

func (issCtx *IssuerContext) verifyCertificateRequestState(ctx context.Context, reason string, status string) error {
	certificateRequestName := fmt.Sprintf("%s-%d", issCtx.certName, 1)
	waitForCertificateRequestToBeCreated(ctx, testContext.cmClient, certificateRequestName, issCtx.namespace)
	err := waitForCertificateRequestState(ctx, testContext.cmClient, certificateRequestName, issCtx.namespace, reason, status)

	if err != nil {
		assert.FailNow(godog.T(ctx), "Certificate Request did not reach specified state, Condition = "+reason+", Status = "+status+": "+err.Error())
	}

	return nil
}

func (issCtx *IssuerContext) verifyCertificateContent(ctx context.Context, usage string) error {
	// The secret name is typically the same as the certificate name + "-cert-secret"
	secretName := issCtx.certName + "-cert-secret"

	certData, err := getCertificateData(ctx, testContext.clientset, issCtx.namespace, secretName)
	if err != nil {
		assert.FailNow(godog.T(ctx), "Failed to get certificate data: "+err.Error())
	}

	if len(certData) == 0 {
		assert.FailNow(godog.T(ctx), "Certificate data is empty")
	}

	log.Printf("Expected usage: %s", usage)

	decodedData, _ := pem.Decode([]byte(certData))
	if decodedData == nil {
		assert.FailNow(godog.T(ctx), "Failed to decode certificate data")
	}

	cert, err := x509.ParseCertificate(decodedData.Bytes)
	if err != nil {
		assert.FailNow(godog.T(ctx), "Failed to parse certificate: "+err.Error())
	}

	usageLabels := map[x509.ExtKeyUsage]string{
		x509.ExtKeyUsageClientAuth:  "client_auth",
		x509.ExtKeyUsageServerAuth:  "server_auth",
		x509.ExtKeyUsageCodeSigning: "code_signing",
		x509.ExtKeyUsageOCSPSigning: "ocsp_signing",
		x509.ExtKeyUsageAny:         "any",
	}

	expectedUsages := strings.Split(usage, ",")

	// Check if all expected usages are present in the certificate
	for _, expectedUsage := range expectedUsages {
		found := false
		for _, extUsage := range cert.ExtKeyUsage {
			if label, exists := usageLabels[extUsage]; exists {
				if label == expectedUsage {
					log.Printf("Found expected usage type in certificate: %s\n", label)
					found = true
					break
				}
			}
		}
		if !found {
			assert.FailNow(godog.T(ctx), "Certificate did not have expected usage: "+expectedUsage)
		}
	}

	return nil
}
