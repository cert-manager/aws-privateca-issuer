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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acmpca"
	acmpcatypes "github.com/aws/aws-sdk-go-v2/service/acmpca/types"
	issuerapi "github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
	cert    = `-----BEGIN CERTIFICATE-----
MIIFnTCCA4WgAwIBAgICEjQwDQYJKoZIhvcNAQENBQAwXjELMAkGA1UEBhMCVVMx
EzARBgNVBAgMClNvbWUtU3RhdGUxFDASBgNVBAoMC0NvbXBhbnkgTHRkMQswCQYD
VQQLDAJJVDEXMBUGA1UEAwwOaW50LmRvbWFpbi5jb20wHhcNMjEwNTIwMjE1NTIw
WhcNMjEwODE4MjE1NTIwWjBMMQswCQYDVQQGEwJVUzETMBEGA1UECAwKU29tZS1T
dGF0ZTEPMA0GA1UEBwwGTXlDaXR5MRcwFQYDVQQDDA5mb28uZG9tYWluLmNvbTCC
ASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAM9Xy+lnHabROa0xfFt4waOG
hPYVUmi9tBKzqvDM4FTNrfH5eBvxrpLZIjQ+laILB3VGquvxiWhaR6FlMcP2eSHh
XnR5PfWuH3NWfPH1kx5IH8a0StlqpjtP2ARc2bUKl61nwgZPQqFyWVvyeLStxkUZ
JwKjInZITWj3iE1JT+ZVSbg0V1BC5kunbbUjAMh1j0Tc4p+k+Xn+TspmUwV0cjFr
VzcdFmZG7wXid4BIoWY2zQoRzRZLfNjB9FQ2UbxIRPT+ZHXHneUynRHcNG4dHv0S
vcXbdmHooibmz1Me6RdRgW1FnE02oz6sgELCBpewE+SZgOxHqAJkAzIardojHNMC
AwEAAaOCAXUwggFxMAkGA1UdEwQCMAAwEQYJYIZIAYb4QgEBBAQDAgZAMDMGCWCG
SAGG+EIBDQQmFiRPcGVuU1NMIEdlbmVyYXRlZCBTZXJ2ZXIgQ2VydGlmaWNhdGUw
HQYDVR0OBBYEFLSX84yrEzUl4xtGvnLTR7hZtC+xMIGWBgNVHSMEgY4wgYuAFHWX
FFPGAdFLUsDTZgJE+3xMfjv1oW+kbTBrMQswCQYDVQQGEwJVUzETMBEGA1UECAwK
U29tZS1TdGF0ZTEPMA0GA1UEBwwGTXlDaXR5MRQwEgYDVQQKDAtDb21wYW55IEx0
ZDELMAkGA1UECwwCSVQxEzARBgNVBAMMCmRvbWFpbi5jb22CAhAAMA4GA1UdDwEB
/wQEAwIFoDATBgNVHSUEDDAKBggrBgEFBQcDATA/BgNVHREEODA2ghBjZXJ0MS5k
b21haW4uY29tghBjZXJ0Mi5kb21haW4uY29tghBjZXJ0My5kb21haW4uY29tMA0G
CSqGSIb3DQEBDQUAA4ICAQAj4dnj91HVFA0y2CmMrkjWzlvEkHwUNpM+lAQvX1ma
/5tVgTYfV4+JlUSYfQWJBsLMtJMnCSfR+FCJ2cCLKE9lpVh1NW37+mlxZ3s0AvMa
gde/Ybh/DcR5QWaeDg9vtRDa8/zhE+wTuEofMlPNSd5Q/+xhStm646pl2KhJiBLx
O/tXKLvxcazynDkN8q23anQ2exFOoBjyG6zL65WNUWUjjAbcWwYXiG2jn5O+6Gl3
j7ZXYsykuRtMyGlC7+FTA3mcWrzz/1/Ve1Udnmk96gZUrkzIRtRXcExvwEASGrhD
f+5+OL+gHSoqDdQCs/yAXQHkSSyXaYNLGFDfDh8ytncBl4shPy+6mV2O2ypCPZtD
viawEqgtIcYV2wkC1ZsmwFHHS7b1xHmi8m3JaVtxakaiXE8exz7Aw2GvxrvkGXpo
vH9nmkQMcyOyPEyooFtKrRPqaI7jbTB/NSyKQFE1tj2MO/YSmeVVYHO1fIjy9lxV
PE7sxcw1dkADmzKpb+tXGd5rDRVAaEeNVSm9npsPcyhB/TTQvctGVKHqhPVrhz3B
CQeOhFhKIB/zKfyyTSguHQWXg5YyqgwwlPIU4IxP1TT5yKjpjJLGYOgfDzBoi3wm
acZaNrl7uzcGWWSOzarX1CnOibTGZJq7bL92f6NyyIXC32HaryIKXvVJW/rPUEUu
hA==
-----END CERTIFICATE-----`
	intermediate = `-----BEGIN CERTIFICATE-----
MIIFqzCCA5OgAwIBAgICEAAwDQYJKoZIhvcNAQENBQAwazELMAkGA1UEBhMCVVMx
EzARBgNVBAgMClNvbWUtU3RhdGUxDzANBgNVBAcMBk15Q2l0eTEUMBIGA1UECgwL
Q29tcGFueSBMdGQxCzAJBgNVBAsMAklUMRMwEQYDVQQDDApkb21haW4uY29tMB4X
DTIxMDUyMDIxNDg0MFoXDTMxMDUxODIxNDg0MFowXjELMAkGA1UEBhMCVVMxEzAR
BgNVBAgMClNvbWUtU3RhdGUxFDASBgNVBAoMC0NvbXBhbnkgTHRkMQswCQYDVQQL
DAJJVDEXMBUGA1UEAwwOaW50LmRvbWFpbi5jb20wggIiMA0GCSqGSIb3DQEBAQUA
A4ICDwAwggIKAoICAQDy3dUOQ0nt3k7fvcDRxBMZdYg5JCWlmx90ZCQpCfJoAq4H
l79y1wCJP8HwhnTYZfohKFWWtU53DnpF08yMAs4Eru9F8oZ433zF4Duu1Ln4wnrD
mZ6ElD7e+SmtqCKZra7/BJ5NR3oxL7GobzX/fKTWiiS/lkcPNlcLBxr1Bx1yyBGs
xALhPkSsli9wI1DNk6Ep2FhhsQbsga4ncTvvaKDyGVHclIOgEiNawYEJ2aWYcHaB
Us1ZocJ+6OM17YaISY2jhy7BS3/WT0oANYdom7of4USL8qHbL1h8ZNiLuVxt4hAW
CCFLmBJ6Lum1diu0UwHbn5i7c7Rxivstjz7Rw25jRqW7RtVZ4MsbaVdLCrTJpQoI
m7qgn2j1IxuWwjVm/PQ273vRUIH7yNU80RG+Cn7uUj2AnaDTOlyytLU/rxjRPUvL
Pl2ZtbZ4ijTGFoI6cnguIacGyU0X3qU5W2EV0T/QkjAomfAumqwFEKMNAwKGiBcc
Bps5AaOWfox3oxM5MWJip6lGuWm//e8Xks1L2jaMIDMjpwHGat+Q4BLrURVsq+VQ
fDjZr2P7lvj6XYE4Ln+WSScXOa6/PAaZoPYinAhlBg72JxQ/s7xQXk1zalk+O4Zw
4jSMckPgS9BGOVNYzq6ZcIjF4GqZkGZMgraZcicNEoPQYdscYzzZHk/jwFdCnwID
AQABo2YwZDAdBgNVHQ4EFgQUdZcUU8YB0UtSwNNmAkT7fEx+O/UwHwYDVR0jBBgw
FoAUDVtiq2sjBzXtbYEt0M6AlfSvgnowEgYDVR0TAQH/BAgwBgEB/wIBADAOBgNV
HQ8BAf8EBAMCAYYwDQYJKoZIhvcNAQENBQADggIBABeQh5HakEXlI39AqSrZWYja
/y6b+IrYQM00UbsBb81rr+xSUGyyRVv/F8qS9WC6FR/mnyOzMWo3nZn6N/cYgY8p
UnTDq4sjl/9drhN7G4LB7GWijKb77b+Z1V9JJImfGbShe+9NbEgol8U2+JVqBCzg
cp7neuPsDoe2uybcee1v/BZAMTzVkOAiIvZvwcacFNyqI6jYIQHDaHYMqs3r6gek
c6TccS1EIhzxacQHn6gSJHSspCWf+7Kpv/Ef9K68ZKb8xE3MwP8/ja8R7+KuEvJO
M927x4833GTBPoZ6LvqNHpKlTBQAQjSZF+RjLIVquHnQnBsxgfk+lfsamLAOi3tm
wjtOl1tv/pn+fIHfMVMqFb2l6LIL6oEEhjjOwUcw2UDgs8oHbnK3VYLZOdsO2aJo
5oBSM+aLgiETQ5LEl/F/OK3nCstswSBvGnQOKKaBlAx06P2PGPNe7MhO+PPS+AWf
6uqXDa5p/pd+CRnN3eYCvTcgVy5+c1Xa2xZdk+eKX2lM7IitFhZqkE/9HizsTouI
B2F0YnOrxJGdJk0gfCuzIcPRTU+RnGrw5KpNA2T6NuNFiHARmBIN5m5hgU+kCFFf
ABPPTMt3xoqP7F5Hbo/DFyM+OiORsmVvHdA0aPEioUW6Yd+2/1gJZD3a0J15qbG/
h80UhX6VVzshLC8PecHj
-----END CERTIFICATE-----`
	root = `-----BEGIN CERTIFICATE-----
MIIFtDCCA5ygAwIBAgIBADANBgkqhkiG9w0BAQ0FADBrMQswCQYDVQQGEwJVUzET
MBEGA1UECAwKU29tZS1TdGF0ZTEPMA0GA1UEBwwGTXlDaXR5MRQwEgYDVQQKDAtD
b21wYW55IEx0ZDELMAkGA1UECwwCSVQxEzARBgNVBAMMCmRvbWFpbi5jb20wHhcN
MjEwNTIwMjE0MDEzWhcNMzEwNTE4MjE0MDEzWjBrMQswCQYDVQQGEwJVUzETMBEG
A1UECAwKU29tZS1TdGF0ZTEPMA0GA1UEBwwGTXlDaXR5MRQwEgYDVQQKDAtDb21w
YW55IEx0ZDELMAkGA1UECwwCSVQxEzARBgNVBAMMCmRvbWFpbi5jb20wggIiMA0G
CSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCtG/KGvLrRSSW9wB1UCThXeqf18MTH
7mcrJhE2wrTXlnzpyjzPW8plwvcn1DIpW8WrUQ5pB8ylzW+oIKNJXnyQMjZg9/aM
zcNmVCkYMTDZpCS9L2fCq2PXP2WHVfoFObegb3P26puEAWGIqleg3Aq0lG2QdVpp
Dvxjf74eZreRUzXTGkCuBgGE55NGLEwyWpqQpx9eOhJlKhcdv55t+tzjdeXUNt+k
x5yKXPvjjNqwMpvNecgrHGWa3ZAJZk4VPuwhDPT4PYEuWR8aKPyRIr8ySugDHKBz
tv/HELFcpBZRfj5Angd7i5UdeN3SNOfk1fEDKWuSKNKvK1B2qhsjbiQl4rXTSE4d
ryW8+Kj48CkxqTATYCy/a3kTIJpJiM+aIlM70NleniYYXO+vR898/YOKmmcG42TL
wGmeiDECtHMPgTj0ypjRrySDGl9957Xb3/3UFt39sSQuOdCxIuDG99q49NHw/os6
AJQmRZkv3+3fvrUModoAjjwBus2LBJ+2ddZV8Mt5KFiMOPEeW1IyGUGi2oaI/sN/
u0A1v/ylnmJmqZBFZCMVw9i2IoIi91eXOWxs6L+5/n9T6AJTUvMznaH37ho5mcOE
SicXpvLqjldVo1jTaxmiBRx9m5nI/ES/4eATzsispq3+SqI2ZyhSqqLg63//sgIp
nMYAbXwiuGg6FQIDAQABo2MwYTAdBgNVHQ4EFgQUDVtiq2sjBzXtbYEt0M6AlfSv
gnowHwYDVR0jBBgwFoAUDVtiq2sjBzXtbYEt0M6AlfSvgnowDwYDVR0TAQH/BAUw
AwEB/zAOBgNVHQ8BAf8EBAMCAYYwDQYJKoZIhvcNAQENBQADggIBAKvxfodn7mtZ
/3MzzkRWI5bmI4+NutbRSGpMWridUb2Clnr94IjNwdx5z9YPyRaIezzT2kkX2VzO
225ZguitHqXtSsCF/dIshe5kSbeCKx0PKjs+OsgSocXJV0vJsVdfStBmZiwa4N7F
OUCkOESvRoFRT8QZHIYfdVI5C1fejvlXCcaC2iXJt1rSCepS29/mzTjgUplruRZ6
Tk46MoHDfhtO56DF91YfZahcSd43251hw3Dkcnhpojhu56p7gVcARag0THPOm9BM
VLFDqScvVYo9SR7cxTY+XqhCWvRBekSj1dy/YAV9lEMxWxuoxlOHDiFSmfvNH7bW
ceUvW7Y8fcsZNH6Aa4qNv13g1sj4aMD8VIHfyt8CfTOBAjpyJv5LDvAzLBZ53fHq
k+WVhaSLdNNK4CiJRdTosoWKLCvT50MqF0IYn0rWwGFxMzqbRi4UXY+UNQWUjmWN
vzkbxR4hQoSJyPft+meKbuYkw+Z2wo0I1KmcDEKJxUa6CmGYDg7xUyNsrfqa+2Rq
ZovD4aDHhDYpXWsBrP3GgwWmFAjgvu3NNo+Q5vcQzDFasZ4B8eRJHjwsbTY1KavX
grbfme8ArsvvcZFsy7SN2jS/Wiqw7PQSML65c4Fhs6jE8wiVsvV43ZrqR+auUOTk
vWJet5qO7W0LkKp4DeQWA0KkAtmgR3ZQ
-----END CERTIFICATE-----`
	chain = intermediate + "\n" + root
)

type errorACMPCAClient struct{}

func (m *errorACMPCAClient) DescribeCertificateAuthority(_ context.Context, input *acmpca.DescribeCertificateAuthorityInput, _ ...func(*acmpca.Options)) (*acmpca.DescribeCertificateAuthorityOutput, error) {
	return &acmpca.DescribeCertificateAuthorityOutput{
		CertificateAuthority: &acmpcatypes.CertificateAuthority{
			CertificateAuthorityConfiguration: &acmpcatypes.CertificateAuthorityConfiguration{
				SigningAlgorithm: acmpcatypes.SigningAlgorithmSha256withecdsa,
			},
		},
	}, nil
}

func (m *errorACMPCAClient) IssueCertificate(_ context.Context, input *acmpca.IssueCertificateInput, _ ...func(*acmpca.Options)) (*acmpca.IssueCertificateOutput, error) {
	return nil, errors.New("Cannot issue certificate")
}

func (m *errorACMPCAClient) GetCertificate(_ context.Context, input *acmpca.GetCertificateInput, _ ...func(*acmpca.Options)) (*acmpca.GetCertificateOutput, error) {
	return nil, errors.New("Cannot get certificate")
}

type workingACMPCAClient struct {
	issueCertInput *acmpca.IssueCertificateInput
}

func (m *workingACMPCAClient) DescribeCertificateAuthority(_ context.Context, input *acmpca.DescribeCertificateAuthorityInput, _ ...func(*acmpca.Options)) (*acmpca.DescribeCertificateAuthorityOutput, error) {
	return &acmpca.DescribeCertificateAuthorityOutput{
		CertificateAuthority: &acmpcatypes.CertificateAuthority{
			CertificateAuthorityConfiguration: &acmpcatypes.CertificateAuthorityConfiguration{
				SigningAlgorithm: acmpcatypes.SigningAlgorithmSha256withecdsa,
			},
		},
	}, nil
}

func (m *workingACMPCAClient) IssueCertificate(_ context.Context, input *acmpca.IssueCertificateInput, _ ...func(*acmpca.Options)) (*acmpca.IssueCertificateOutput, error) {
	m.issueCertInput = input
	return &acmpca.IssueCertificateOutput{CertificateArn: &certArn}, nil
}

func (m *workingACMPCAClient) GetCertificate(_ context.Context, input *acmpca.GetCertificateInput, _ ...func(*acmpca.Options)) (*acmpca.GetCertificateOutput, error) {
	return &acmpca.GetCertificateOutput{Certificate: &cert, CertificateChain: &chain}, nil
}

func TestProvisonerOperation(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, issuerapi.AddToScheme(scheme))
	require.NoError(t, cmapi.AddToScheme(scheme))
	require.NoError(t, v1.AddToScheme(scheme))

	objects := []client.Object{
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "issuer1-credentials",
				Namespace: "ns1",
			},
			Data: map[string][]byte{
				"AWS_ACCESS_KEY_ID":     []byte("ZXhhbXBsZQ=="),
				"AWS_SECRET_ACCESS_KEY": []byte("ZXhhbXBsZQ=="),
			},
		},
	}
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		WithStatusSubresource(objects...).
		Build()

	issSpec := &issuerapi.AWSPCAIssuerSpec{
		SecretRef: issuerapi.AWSCredentialsSecretReference{
			SecretReference: v1.SecretReference{
				Name:      "issuer1-credentials",
				Namespace: "ns1",
			},
		},
		Region: "us-east-1",
		Arn:    "arn:aws:acm-pca:us-east-1:account:certificate-authority/12345678-1234-1234-1234-123456789012",
	}

	ClearProvisioners()
	provisioner, err := GetProvisioner(context.TODO(), fakeClient, types.NamespacedName{Namespace: "ns1", Name: "issuer1"}, issSpec)
	assert.Equal(t, err, nil)
	output, err := GetProvisioner(context.TODO(), fakeClient, types.NamespacedName{Namespace: "ns1", Name: "issuer1"}, issSpec)
	assert.Equal(t, output, provisioner)
	assert.Equal(t, err, nil)

	DeleteProvisioner(context.TODO(), fakeClient, types.NamespacedName{Namespace: "ns1", Name: "issuer1"})
	output, err = GetProvisioner(context.TODO(), fakeClient, types.NamespacedName{Namespace: "ns1", Name: "issuer1"}, issSpec)
	assert.NotEqual(t, output, provisioner)
	assert.Equal(t, err, nil)
}

func TestPCATemplateArn(t *testing.T) {
	var (
		arn     = "arn:aws:acm-pca:us-east-1:account:certificate-authority/12345678-1234-1234-1234-123456789012"
		govArn  = "arn:aws-us-gov:acm-pca:us-east-1:account:certificate-authority/12345678-1234-1234-1234-123456789012"
		fakeArn = "arn:fake:acm-pca:us-east-1:account:certificate-authority/12345678-1234-1234-1234-123456789012"
	)

	type testCase struct {
		expectedSuffix  string
		certificateSpec cmapi.CertificateRequestSpec
	}
	tests := map[string]testCase{
		"client": {
			expectedSuffix: ":acm-pca:::template/EndEntityClientAuthCertificate/V1",
			certificateSpec: cmapi.CertificateRequestSpec{
				Usages: []cmapi.KeyUsage{
					cmapi.UsageClientAuth,
				},
			},
		},
		"server": {
			expectedSuffix: ":acm-pca:::template/EndEntityServerAuthCertificate/V1",
			certificateSpec: cmapi.CertificateRequestSpec{
				Usages: []cmapi.KeyUsage{
					cmapi.UsageServerAuth,
				},
			},
		},
		"client server": {
			expectedSuffix: ":acm-pca:::template/EndEntityCertificate/V1",
			certificateSpec: cmapi.CertificateRequestSpec{
				Usages: []cmapi.KeyUsage{
					cmapi.UsageClientAuth,
					cmapi.UsageServerAuth,
				},
			},
		},
		"server client": {
			expectedSuffix: ":acm-pca:::template/EndEntityCertificate/V1",
			certificateSpec: cmapi.CertificateRequestSpec{
				Usages: []cmapi.KeyUsage{
					cmapi.UsageServerAuth,
					cmapi.UsageClientAuth,
				},
			},
		},
		"code signing": {
			expectedSuffix: ":acm-pca:::template/CodeSigningCertificate/V1",
			certificateSpec: cmapi.CertificateRequestSpec{
				Usages: []cmapi.KeyUsage{
					cmapi.UsageCodeSigning,
				},
			},
		},
		"ocsp signing": {
			expectedSuffix: ":acm-pca:::template/OCSPSigningCertificate/V1",
			certificateSpec: cmapi.CertificateRequestSpec{
				Usages: []cmapi.KeyUsage{
					cmapi.UsageOCSPSigning,
				},
			},
		},
		"other": {
			expectedSuffix: ":acm-pca:::template/BlankEndEntityCertificate_APICSRPassthrough/V1",
			certificateSpec: cmapi.CertificateRequestSpec{
				Usages: []cmapi.KeyUsage{
					cmapi.UsageTimestamping,
				},
			},
		},
		"isCA default": {
			expectedSuffix: ":acm-pca:::template/SubordinateCACertificate_PathLen0/V1",
			certificateSpec: cmapi.CertificateRequestSpec{
				IsCA: true,
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			spec := tc.certificateSpec

			response := templateArn(arn, spec)
			assert.True(t, strings.HasSuffix(response, tc.expectedSuffix), "returns expected template")
			assert.True(t, strings.HasPrefix(response, "arn:aws:"), "returns expected ARN prefix")
		})
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			spec := tc.certificateSpec

			response := templateArn(govArn, spec)
			assert.True(t, strings.HasSuffix(response, tc.expectedSuffix), "us-gov returns expected template")
			assert.True(t, strings.HasPrefix(response, "arn:aws-us-gov:"), "us-gov returns expected ARN prefix")
		})
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			spec := tc.certificateSpec

			response := templateArn(fakeArn, spec)
			assert.True(t, strings.HasSuffix(response, tc.expectedSuffix), "fake arn returns expected template")
			assert.True(t, strings.HasPrefix(response, "arn:fake:"), "fake arn returns expected ARN prefix")
		})
	}
}

func TestIdempotencyToken(t *testing.T) {
	var (
		idempotencyTokenMaxLength = 36
	)

	type testCase struct {
		request  cmapi.CertificateRequest
		expected string
	}

	tests := map[string]testCase{
		"success": {
			request: cmapi.CertificateRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-name",
					Namespace: "fake-namespace",
				},
			},
			expected: "63e69830270b95081942a3d85034fdc97bb9", // Truncated SHA-256 hash
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			token := idempotencyToken(&tc.request)
			assert.Equal(t, tc.expected, token)
			assert.LessOrEqual(t, len(token), idempotencyTokenMaxLength)
		})
	}
}

func TestPCAGetConfig(t *testing.T) {
	type testCase struct {
		name          types.NamespacedName
		objects       []client.Object
		expectFailure bool
		expectedError error
	}

	tests := map[string]testCase{
		"success-issuer": {
			name: types.NamespacedName{Namespace: "ns1", Name: "issuer1"},
			objects: []client.Object{
				&issuerapi.AWSPCAIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer1",
						Namespace: "ns1",
					},
					Spec: issuerapi.AWSPCAIssuerSpec{
						SecretRef: issuerapi.AWSCredentialsSecretReference{
							SecretReference: v1.SecretReference{
								Name:      "issuer1-credentials",
								Namespace: "ns1",
							},
						},
						Region: "us-east-1",
						Arn:    "arn:aws:acm-pca:us-east-1:account:certificate-authority/12345678-1234-1234-1234-123456789012",
					},
					Status: issuerapi.AWSPCAIssuerStatus{
						Conditions: []metav1.Condition{
							{
								Type:   issuerapi.ConditionTypeReady,
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer1-credentials",
						Namespace: "ns1",
					},
					Data: map[string][]byte{
						"AWS_ACCESS_KEY_ID":     []byte("ZXhhbXBsZQ=="),
						"AWS_SECRET_ACCESS_KEY": []byte("ZXhhbXBsZQ=="),
					},
				},
			},
			expectFailure: false,
		},
		"success-with-secret-selector": {
			name: types.NamespacedName{Namespace: "ns1", Name: "issuer1"},
			objects: []client.Object{
				&issuerapi.AWSPCAIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer1",
						Namespace: "ns1",
					},
					Spec: issuerapi.AWSPCAIssuerSpec{
						SecretRef: issuerapi.AWSCredentialsSecretReference{
							SecretReference: v1.SecretReference{
								Name:      "issuer1-credentials",
								Namespace: "ns1",
							},
							SecretAccessKeySelector: v1.SecretKeySelector{
								Key: "fake-secret-access-key",
							},
							AccessKeyIDSelector: v1.SecretKeySelector{
								Key: "fake-access-key-id",
							},
						},
						Region: "us-east-1",
						Arn:    "arn:aws:acm-pca:us-east-1:account:certificate-authority/12345678-1234-1234-1234-123456789012",
					},
					Status: issuerapi.AWSPCAIssuerStatus{
						Conditions: []metav1.Condition{
							{
								Type:   issuerapi.ConditionTypeReady,
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer1-credentials",
						Namespace: "ns1",
					},
					Data: map[string][]byte{
						"fake-access-key-id":     []byte("ZXhhbXBsZQ=="),
						"fake-secret-access-key": []byte("ZXhhbXBsZQ=="),
					},
				},
			},
			expectFailure: false,
		},
		"failure-secret-not-found": {
			name: types.NamespacedName{Namespace: "ns1", Name: "issuer1"},
			objects: []client.Object{
				&issuerapi.AWSPCAIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer1",
						Namespace: "ns1",
					},
					Spec: issuerapi.AWSPCAIssuerSpec{
						SecretRef: issuerapi.AWSCredentialsSecretReference{
							SecretReference: v1.SecretReference{
								Name:      "issuer1-credentials",
								Namespace: "ns1",
							},
						},
						Region: "us-east-1",
						Arn:    "arn:aws:acm-pca:us-east-1:account:certificate-authority/12345678-1234-1234-1234-123456789012",
					},
					Status: issuerapi.AWSPCAIssuerStatus{
						Conditions: []metav1.Condition{
							{
								Type:   issuerapi.ConditionTypeReady,
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			},
			expectFailure: true,
		},
		"failure-no-access-key-id": {
			name: types.NamespacedName{Namespace: "ns1", Name: "issuer1"},
			objects: []client.Object{
				&issuerapi.AWSPCAIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer1",
						Namespace: "ns1",
					},
					Spec: issuerapi.AWSPCAIssuerSpec{
						SecretRef: issuerapi.AWSCredentialsSecretReference{
							SecretReference: v1.SecretReference{
								Name:      "issuer1-credentials",
								Namespace: "ns1",
							},
						},
						Region: "us-east-1",
						Arn:    "arn:aws:acm-pca:us-east-1:account:certificate-authority/12345678-1234-1234-1234-123456789012",
					},
					Status: issuerapi.AWSPCAIssuerStatus{
						Conditions: []metav1.Condition{
							{
								Type:   issuerapi.ConditionTypeReady,
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer1-credentials",
						Namespace: "ns1",
					},
					Data: map[string][]byte{
						"AWS_SECRET_ACCESS_KEY": []byte("ZXhhbXBsZQ=="),
					},
				},
			},
			expectFailure: true,
			expectedError: ErrNoAccessKeyID,
		},
		"failure-issuer-no-access-key-specified-with-selector": {
			name: types.NamespacedName{Namespace: "ns1", Name: "issuer1"},
			objects: []client.Object{
				&issuerapi.AWSPCAIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer1",
						Namespace: "ns1",
					},
					Spec: issuerapi.AWSPCAIssuerSpec{
						SecretRef: issuerapi.AWSCredentialsSecretReference{
							SecretReference: v1.SecretReference{
								Name:      "issuer1-credentials",
								Namespace: "ns1",
							},
							AccessKeyIDSelector: v1.SecretKeySelector{
								Key: "fake-access-key-id",
							},
						},
						Region: "us-east-1",
						Arn:    "arn:aws:acm-pca:us-east-1:account:certificate-authority/12345678-1234-1234-1234-123456789012",
					},
					Status: issuerapi.AWSPCAIssuerStatus{
						Conditions: []metav1.Condition{
							{
								Type:   issuerapi.ConditionTypeReady,
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer1-credentials",
						Namespace: "ns1",
					},
					Data: map[string][]byte{
						"AWS_ACCESS_KEY_ID":     []byte("ZXhhbXBsZQ=="),
						"AWS_SECRET_ACCESS_KEY": []byte("ZXhhbXBsZQ=="),
					},
				},
			},
			expectFailure: true,
			expectedError: ErrNoAccessKeyID,
		},
		"failure-no-secret-access-key": {
			name: types.NamespacedName{Namespace: "ns1", Name: "issuer1"},
			objects: []client.Object{
				&issuerapi.AWSPCAIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer1",
						Namespace: "ns1",
					},
					Spec: issuerapi.AWSPCAIssuerSpec{
						SecretRef: issuerapi.AWSCredentialsSecretReference{
							SecretReference: v1.SecretReference{
								Name:      "issuer1-credentials",
								Namespace: "ns1",
							},
						},
						Region: "us-east-1",
						Arn:    "arn:aws:acm-pca:us-east-1:account:certificate-authority/12345678-1234-1234-1234-123456789012",
					},
					Status: issuerapi.AWSPCAIssuerStatus{
						Conditions: []metav1.Condition{
							{
								Type:   issuerapi.ConditionTypeReady,
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer1-credentials",
						Namespace: "ns1",
					},
					Data: map[string][]byte{
						"AWS_ACCESS_KEY_ID": []byte("ZXhhbXBsZQ=="),
					},
				},
			},
			expectFailure: true,
			expectedError: ErrNoSecretAccessKey,
		},
		"failure-issuer-no-secret-access-key-specified-with-selector": {
			name: types.NamespacedName{Namespace: "ns1", Name: "issuer1"},
			objects: []client.Object{
				&issuerapi.AWSPCAIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer1",
						Namespace: "ns1",
					},
					Spec: issuerapi.AWSPCAIssuerSpec{
						SecretRef: issuerapi.AWSCredentialsSecretReference{
							SecretReference: v1.SecretReference{
								Name:      "issuer1-credentials",
								Namespace: "ns1",
							},
							SecretAccessKeySelector: v1.SecretKeySelector{
								Key: "fake-secret-access-key",
							},
						},
						Region: "us-east-1",
						Arn:    "arn:aws:acm-pca:us-east-1:account:certificate-authority/12345678-1234-1234-1234-123456789012",
					},
					Status: issuerapi.AWSPCAIssuerStatus{
						Conditions: []metav1.Condition{
							{
								Type:   issuerapi.ConditionTypeReady,
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
				&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer1-credentials",
						Namespace: "ns1",
					},
					Data: map[string][]byte{
						"AWS_ACCESS_KEY_ID":     []byte("ZXhhbXBsZQ=="),
						"AWS_SECRET_ACCESS_KEY": []byte("ZXhhbXBsZQ=="),
					},
				},
			},
			expectFailure: true,
			expectedError: ErrNoSecretAccessKey,
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, issuerapi.AddToScheme(scheme))
	require.NoError(t, cmapi.AddToScheme(scheme))
	require.NoError(t, v1.AddToScheme(scheme))

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tc.objects...).
				WithStatusSubresource(tc.objects...).
				Build()

			ctx := context.TODO()
			iss := new(issuerapi.AWSPCAIssuer)
			require.NoError(t, fakeClient.Get(ctx, tc.name, iss))

			config, err := GetConfig(ctx, fakeClient, iss.GetSpec())

			if tc.expectFailure && err == nil {
				assert.Fail(t, "Expected an error but got none")
			}

			if tc.expectedError != nil {
				assert.Equal(t, tc.expectedError, err, "Unexpected error")
			}

			if err == nil {
				creds, _ := config.Credentials.Retrieve(ctx)
				assert.Equal(t, "ZXhhbXBsZQ==", creds.AccessKeyID)
				assert.Equal(t, "ZXhhbXBsZQ==", creds.SecretAccessKey)
			}
		})
	}
}

func TestPCAGet(t *testing.T) {
	type testCase struct {
		provisioner   PCAProvisioner
		expectFailure bool
		expectedChain string
		expectedCert  string
	}

	tests := map[string]testCase{
		"success": {
			provisioner:   PCAProvisioner{arn: arn, pcaClient: &workingACMPCAClient{}},
			expectFailure: false,
			expectedChain: string([]byte(root + "\n")),
			expectedCert:  string([]byte(cert + "\n" + intermediate + "\n")),
		},
		"failure-error-getCertificate": {
			provisioner:   PCAProvisioner{arn: arn, pcaClient: &errorACMPCAClient{}},
			expectFailure: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			key, _ := rsa.GenerateKey(rand.Reader, 2048)
			csrBytes, _ := x509.CreateCertificateRequest(rand.Reader, &template, key)

			cr := &cmapi.CertificateRequest{
				Spec: cmapi.CertificateRequestSpec{
					Request: pem.EncodeToMemory(&pem.Block{
						Bytes: csrBytes,
						Type:  "CERTIFICATE REQUEST",
					}),
				},
			}

			leaf, chain, err := tc.provisioner.Get(context.TODO(), cr, certArn, logr.Discard())

			if tc.expectFailure && err == nil {
				fmt.Print(err.Error())
				assert.Fail(t, "Expected an error but received none")
			}

			if tc.expectedChain != "" && tc.expectedCert != "" {
				assert.Equal(t, []byte(tc.expectedCert), leaf)
				assert.Equal(t, []byte(tc.expectedChain), chain)
			}
		})
	}
}

func TestPCASign(t *testing.T) {
	type testCase struct {
		provisioner     PCAProvisioner
		expectFailure   bool
		expectedCertArn string
	}

	tests := map[string]testCase{
		"success": {
			provisioner:     PCAProvisioner{arn: arn, pcaClient: &workingACMPCAClient{}},
			expectFailure:   false,
			expectedCertArn: "arn",
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

			cr := &cmapi.CertificateRequest{
				Spec: cmapi.CertificateRequestSpec{
					Request: pem.EncodeToMemory(&pem.Block{
						Bytes: csrBytes,
						Type:  "CERTIFICATE REQUEST",
					}),
				},
			}

			err := tc.provisioner.Sign(context.TODO(), cr, logr.Discard())

			if tc.expectFailure && err == nil {
				fmt.Print(err.Error())
				assert.Fail(t, "Expected an error but received none")
			}

			if tc.expectedCertArn != "" {
				assert.Equal(t, cr.GetAnnotations()["aws-privateca-issuer/certificate-arn"], tc.expectedCertArn)
			}
		})
	}
}

func TestPCASignValidity(t *testing.T) {
	now := time.Now()
	client := &workingACMPCAClient{}
	provisioner := PCAProvisioner{arn: arn, pcaClient: client}
	provisioner.clock = func() time.Time { return now }
	type testCase struct {
		duration      *metav1.Duration
		expectedInput *acmpca.IssueCertificateInput
	}

	tests := map[string]testCase{
		"default": {
			duration: nil,
			expectedInput: &acmpca.IssueCertificateInput{
				CertificateAuthorityArn: aws.String(arn),
				Validity: &acmpcatypes.Validity{
					Type:  acmpcatypes.ValidityPeriodTypeAbsolute,
					Value: ptrInt(int64(now.Unix()) + DEFAULT_DURATION),
				},
			},
		},
		"duration specified": {
			duration: ptrDuration(metav1.Duration{Duration: 3 * time.Hour}),
			expectedInput: &acmpca.IssueCertificateInput{
				CertificateAuthorityArn: aws.String(arn),
				Validity: &acmpcatypes.Validity{
					Type:  acmpcatypes.ValidityPeriodTypeAbsolute,
					Value: ptrInt(int64(now.Unix()) + int64(3*time.Hour.Seconds())),
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client.issueCertInput = nil
			key, _ := rsa.GenerateKey(rand.Reader, 2048)
			csrBytes, _ := x509.CreateCertificateRequest(rand.Reader, &template, key)

			cr := &cmapi.CertificateRequest{
				Spec: cmapi.CertificateRequestSpec{
					Request: pem.EncodeToMemory(&pem.Block{
						Bytes: csrBytes,
						Type:  "CERTIFICATE REQUEST",
					}),
					Duration: tc.duration,
				},
			}

			_ = provisioner.Sign(context.TODO(), cr, logr.Discard())
			got := client.issueCertInput
			if got == nil {
				assert.Fail(t, "Expected certificate input, got none")
			} else {
				assert.Equal(t, *got.CertificateAuthorityArn, *tc.expectedInput.CertificateAuthorityArn, name)
				assert.Equal(t, got.Validity.Type, tc.expectedInput.Validity.Type, name)
				assert.Equal(t, *got.Validity.Value, *tc.expectedInput.Validity.Value, name)
			}

		})
	}
}

func ptrInt(i int64) *int64 {
	return &i
}

func ptrDuration(d metav1.Duration) *metav1.Duration {
	return &d
}
