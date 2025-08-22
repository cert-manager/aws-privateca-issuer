package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	util "github.com/cert-manager/cert-manager/pkg/api/util"
	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/cucumber/godog"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"crypto/x509"
	"encoding/pem"
	"slices"

	"github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CertificateConfig struct {
	IsCA         bool
	Duration     time.Duration
	KeyAlgorithm string
	RenewBefore  time.Duration
	Usages       []cmv1.KeyUsage
}

const secretSuffix = "-cert-secret"

var usageMap = map[string]cmv1.KeyUsage{
	"client_auth":       cmv1.UsageClientAuth,
	"server_auth":       cmv1.UsageServerAuth,
	"digital_signature": cmv1.UsageDigitalSignature,
	"code_signing":      cmv1.UsageCodeSigning,
	"ocsp_signing":      cmv1.UsageOCSPSigning,
	"any":               cmv1.UsageAny,
	"email protection":  cmv1.UsageEmailProtection,
	"ipsec user":        cmv1.UsageIPsecUser,
	"ipsec tunnel":      cmv1.UsageIPsecTunnel,
}

func defaultCertConfig() CertificateConfig {
	return CertificateConfig{
		IsCA:         false,
		KeyAlgorithm: "RSA",
		Duration:     721,
		RenewBefore:  240,
		Usages:       []cmv1.KeyUsage{cmv1.UsageClientAuth, cmv1.UsageServerAuth},
	}
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

func getCertSpec(ctx context.Context, certConfig CertificateConfig) cmv1.CertificateSpec {
	sanitizedCertType := strings.Replace(strings.ToLower(certConfig.KeyAlgorithm), "_", "-", -1)

	certSpec := cmv1.CertificateSpec{
		IsCA: certConfig.IsCA,
		Subject: &cmv1.X509Subject{
			Organizations: []string{"aws"},
		},
		DNSNames: []string{sanitizedCertType + "-cert.aws.com"},
		Duration: &metav1.Duration{
			Duration: certConfig.Duration * time.Hour,
		},
		RenewBefore: &metav1.Duration{
			Duration: certConfig.RenewBefore * time.Hour,
		},
		Usages: certConfig.Usages,
	}

	switch certConfig.KeyAlgorithm {
	case "RSA":
		certSpec.PrivateKey = &cmv1.CertificatePrivateKey{
			Algorithm: cmv1.RSAKeyAlgorithm,
			Size:      2048,
		}
	case "ECDSA":
		certSpec.PrivateKey = &cmv1.CertificatePrivateKey{
			Algorithm: cmv1.ECDSAKeyAlgorithm,
			Size:      256,
		}
	default:
		assert.FailNow(godog.T(ctx), "Unknown certificate key algorithm: "+certConfig.KeyAlgorithm)
	}

	return certSpec
}

func (issCtx *IssuerContext) setCAConfig() {
	issCtx.certConfig.IsCA = true
}

func (issCtx *IssuerContext) setKeyAlgorithmConfig(keyAlgorithm string) {
	issCtx.certConfig.KeyAlgorithm = keyAlgorithm
}

func (issCtx *IssuerContext) setDurationConfig(duration int) {
	issCtx.certConfig.Duration = time.Duration(duration)
}

func (issCtx *IssuerContext) setRenewBeforeConfig(renewBefore int) {
	issCtx.certConfig.RenewBefore = time.Duration(renewBefore)
}

func (issCtx *IssuerContext) setUsagesConfig(usageStr string) {
	issCtx.certConfig.Usages = parseUsages(usageStr)
}

func (issCtx *IssuerContext) issueConfiguredCertificate(ctx context.Context) error {
	return issCtx.issueCertificate(ctx, issCtx.certConfig)
}

func (issCtx *IssuerContext) issueCertificate(ctx context.Context, certConfig CertificateConfig) error {
	sanitizedCertType := strings.Replace(strings.ToLower(certConfig.KeyAlgorithm), "_", "-", -1)
	issCtx.certName = issCtx.issuerName + "-" + sanitizedCertType + "-cert"
	certSpec := getCertSpec(ctx, certConfig)

	secretName := issCtx.certName + secretSuffix
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

func parseUsages(usageStr string) []cmv1.KeyUsage {
	parts := strings.Split(usageStr, ",")
	var usages []cmv1.KeyUsage
	for _, part := range parts {
		if usage, exists := usageMap[strings.ToLower(part)]; exists {
			usages = append(usages, usage)
		} else {
			usages = append(usages, cmv1.KeyUsage(part))
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

func (issCtx *IssuerContext) verifyCertificateContent(ctx context.Context, usage string) error {
	secretName := issCtx.certName + secretSuffix

	certBytes, err := getCertificateData(ctx, testContext.clientset, issCtx.namespace, secretName)
	if err != nil {
		assert.FailNow(godog.T(ctx), "Failed to get certificate data: "+err.Error())
	}

	if len(certBytes) == 0 {
		assert.FailNow(godog.T(ctx), "Certificate data is empty")
	}

	decodedData, _ := pem.Decode(certBytes)
	if decodedData == nil {
		assert.FailNow(godog.T(ctx), "Failed to decode certificate data")
	}

	cert, err := x509.ParseCertificate(decodedData.Bytes)
	if err != nil {
		assert.FailNow(godog.T(ctx), "Failed to parse certificate: "+err.Error())
	}

	for _, expectedUsage := range strings.Split(usage, ",") {
		mappedUsage, exists := usageMap[expectedUsage]
		if !exists {
			assert.FailNow(godog.T(ctx), "Expected usage %q not found in usageMap.", expectedUsage)
		}

		x509Usage, _ := util.ExtKeyUsageType(mappedUsage)
		if !slices.Contains(cert.ExtKeyUsage, x509Usage) {
			assert.FailNow(godog.T(ctx), fmt.Sprintf("Certificate usage mismatch. Found: %v, Expected: %v", cert.ExtKeyUsage, mappedUsage))
		}
	}

	return nil
}
