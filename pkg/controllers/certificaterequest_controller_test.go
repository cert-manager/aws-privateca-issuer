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
package controllers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	acmpcatypes "github.com/aws/aws-sdk-go-v2/service/acmpca/types"
	cmutil "github.com/cert-manager/cert-manager/pkg/api/util"
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	cmgen "github.com/cert-manager/cert-manager/test/unit/gen"
	"github.com/go-logr/logr"
	logrtesting "github.com/go-logr/logr/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	issuerapi "github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	awspca "github.com/cert-manager/aws-privateca-issuer/pkg/aws"
)

type fakeProvisioner struct {
	cert    []byte
	caCert  []byte
	getErr  error
	signErr error
}

func (p *fakeProvisioner) Sign(ctx context.Context, cr *cmapi.CertificateRequest, log logr.Logger) error {
	metav1.SetMetaDataAnnotation(&cr.ObjectMeta, "aws-privateca-issuer/certificate-arn", "arn")
	return p.signErr
}

func (p *fakeProvisioner) Get(ctx context.Context, cr *cmapi.CertificateRequest, certArn string, log logr.Logger) ([]byte, []byte, error) {
	return p.cert, p.caCert, p.getErr
}

type createMockProvisioner func()

type fakeRequeueItter struct {
}

func (r *fakeRequeueItter) RequeueAfter() time.Duration {
	return 1 * time.Hour
}

func TestProvisonerOperation(t *testing.T) {
	provisioner := awspca.NewProvisioner(aws.Config{}, "arn")
	awspca.StoreProvisioner(types.NamespacedName{Namespace: "ns1", Name: "issuer1"}, provisioner)
	output, exists := awspca.GetProvisioner(types.NamespacedName{Namespace: "ns1", Name: "issuer1"})
	assert.Equal(t, output, provisioner)
	assert.Equal(t, exists, true)
}

func TestCertificateRequestReconcile(t *testing.T) {
	type testCase struct {
		name                         types.NamespacedName
		objects                      []client.Object
		expectedSignResult           ctrl.Result
		expectedGetResult            ctrl.Result
		expectedError                bool
		expectedReadyConditionStatus cmmeta.ConditionStatus
		expectedReadyConditionReason string
		expectedCertificate          []byte
		expectedCACertificate        []byte
		retryDuration                time.Duration
		mockProvisioner              createMockProvisioner
	}
	tests := map[string]testCase{
		"success-issuer": {
			name: types.NamespacedName{Namespace: "ns1", Name: "cr1"},
			objects: []client.Object{
				cmgen.CertificateRequest(
					"cr1",
					cmgen.SetCertificateRequestNamespace("ns1"),
					SetCreationTime(time.Now()),
					cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
						Name:  "issuer1",
						Group: issuerapi.GroupVersion.Group,
						Kind:  "Issuer",
					}),
					cmgen.SetCertificateRequestStatusCondition(cmapi.CertificateRequestCondition{
						Type:   cmapi.CertificateRequestConditionReady,
						Status: cmmeta.ConditionUnknown,
					}),
				),
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
			expectedSignResult:           ctrl.Result{Requeue: true},
			expectedGetResult:            ctrl.Result{},
			expectedReadyConditionStatus: cmmeta.ConditionTrue,
			expectedReadyConditionReason: cmapi.CertificateRequestReasonIssued,
			expectedError:                false,
			expectedCertificate:          []byte("cert"),
			expectedCACertificate:        []byte("cacert"),
			mockProvisioner: func() {
				awspca.StoreProvisioner(types.NamespacedName{Namespace: "ns1", Name: "issuer1"}, &fakeProvisioner{caCert: []byte("cacert"), cert: []byte("cert")})
			},
		},
		"success-cluster-issuer": {
			name: types.NamespacedName{Namespace: "ns1", Name: "cr1"},
			objects: []client.Object{
				cmgen.CertificateRequest(
					"cr1",
					cmgen.SetCertificateRequestNamespace("ns1"),
					SetCreationTime(time.Now()),
					cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
						Name:  "clusterissuer1",
						Group: issuerapi.GroupVersion.Group,
						Kind:  "AWSPCAClusterIssuer",
					}),
					cmgen.SetCertificateRequestStatusCondition(cmapi.CertificateRequestCondition{
						Type:   cmapi.CertificateRequestConditionReady,
						Status: cmmeta.ConditionUnknown,
					}),
				),
				&issuerapi.AWSPCAClusterIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "clusterissuer1",
					},
					Spec: issuerapi.AWSPCAIssuerSpec{
						SecretRef: issuerapi.AWSCredentialsSecretReference{
							SecretReference: v1.SecretReference{
								Name: "clusterissuer1-credentials",
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
						Name: "clusterissuer1-credentials",
					},
					Data: map[string][]byte{
						"AWS_ACCESS_KEY_ID":     []byte("ZXhhbXBsZQ=="),
						"AWS_SECRET_ACCESS_KEY": []byte("ZXhhbXBsZQ=="),
					},
				},
			},
			expectedReadyConditionStatus: cmmeta.ConditionTrue,
			expectedReadyConditionReason: cmapi.CertificateRequestReasonIssued,
			expectedError:                false,
			expectedCertificate:          []byte("cert"),
			expectedCACertificate:        []byte("cacert"),
			mockProvisioner: func() {
				awspca.StoreProvisioner(types.NamespacedName{Name: "clusterissuer1"}, &fakeProvisioner{caCert: []byte("cacert"), cert: []byte("cert")})
			},
		},
		"success-previous-failure": {
			name: types.NamespacedName{Namespace: "ns1", Name: "cr1"},
			objects: []client.Object{
				cmgen.CertificateRequest(
					"cr1",
					cmgen.SetCertificateRequestNamespace("ns1"),
					SetCreationTime(time.Now()),
					cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
						Name:  "clusterissuer1",
						Group: issuerapi.GroupVersion.Group,
						Kind:  "AWSPCAClusterIssuer",
					}),
					cmgen.SetCertificateRequestStatusCondition(cmapi.CertificateRequestCondition{
						Type:   cmapi.CertificateRequestConditionReady,
						Status: cmmeta.ConditionFalse,
					}),
				),
				&issuerapi.AWSPCAClusterIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "clusterissuer1",
					},
					Spec: issuerapi.AWSPCAIssuerSpec{
						SecretRef: issuerapi.AWSCredentialsSecretReference{
							SecretReference: v1.SecretReference{
								Name: "clusterissuer1-credentials",
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
						Name: "clusterissuer1-credentials",
					},
					Data: map[string][]byte{
						"AWS_ACCESS_KEY_ID":     []byte("ZXhhbXBsZQ=="),
						"AWS_SECRET_ACCESS_KEY": []byte("ZXhhbXBsZQ=="),
					},
				},
			},
			expectedReadyConditionStatus: cmmeta.ConditionTrue,
			expectedReadyConditionReason: cmapi.CertificateRequestReasonIssued,
			expectedError:                false,
			expectedCertificate:          []byte("cert"),
			expectedCACertificate:        []byte("cacert"),
			mockProvisioner: func() {
				awspca.StoreProvisioner(types.NamespacedName{Name: "clusterissuer1"}, &fakeProvisioner{caCert: []byte("cacert"), cert: []byte("cert")})
			},
		},
		"success-certificate-already-issued": {
			name: types.NamespacedName{Name: "cr1"},
			objects: []client.Object{
				cmgen.CertificateRequest(
					"cr1",
					cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
						Name:  "clusterissuer1",
						Group: issuerapi.GroupVersion.Group,
						Kind:  "ClusterIssuer",
					}),
					cmgen.SetCertificateRequestStatusCondition(cmapi.CertificateRequestCondition{
						Type:   cmapi.CertificateRequestReasonIssued,
						Status: cmmeta.ConditionTrue,
					}),
					cmgen.SetCertificateRequestCertificate([]byte("oldCert")),
					cmgen.SetCertificateRequestCA([]byte("oldCaCert")),
				),
				&issuerapi.AWSPCAClusterIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "clusterissuer1",
					},
					Spec: issuerapi.AWSPCAIssuerSpec{
						SecretRef: issuerapi.AWSCredentialsSecretReference{
							SecretReference: v1.SecretReference{
								Name: "clusterissuer1-credentials",
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
						Name: "clusterissuer1-credentials",
					},
					Data: map[string][]byte{
						"AWS_ACCESS_KEY_ID":     []byte("ZXhhbXBsZQ=="),
						"AWS_SECRET_ACCESS_KEY": []byte("ZXhhbXBsZQ=="),
					},
				},
			},
			expectedReadyConditionStatus: cmmeta.ConditionTrue,
			expectedReadyConditionReason: cmapi.CertificateRequestReasonIssued,
			expectedError:                false,
			expectedCertificate:          []byte("oldCert"),
			expectedCACertificate:        []byte("oldCaCert"),
			mockProvisioner: func() {
				awspca.StoreProvisioner(types.NamespacedName{Name: "clusterissuer1"}, &fakeProvisioner{caCert: []byte("cacert"), cert: []byte("cert")})
			},
		},
		"failure-certificate-not-issued": {
			name: types.NamespacedName{Namespace: "ns1", Name: "cr1"},
			objects: []client.Object{
				cmgen.CertificateRequest(
					"cr1",
					cmgen.SetCertificateRequestNamespace("ns1"),
					cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
						Name:  "issuer1",
						Group: issuerapi.GroupVersion.Group,
						Kind:  "Issuer",
					}),
					cmgen.SetCertificateRequestStatusCondition(cmapi.CertificateRequestCondition{
						Type:   cmapi.CertificateRequestConditionReady,
						Status: cmmeta.ConditionUnknown,
					}),
				),
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
			expectedSignResult:           ctrl.Result{Requeue: true},
			expectedGetResult:            ctrl.Result{Requeue: true},
			expectedReadyConditionStatus: cmmeta.ConditionFalse,
			expectedReadyConditionReason: cmapi.CertificateRequestReasonPending,
			expectedError:                false,
			mockProvisioner: func() {
				awspca.StoreProvisioner(types.NamespacedName{Namespace: "ns1", Name: "issuer1"}, &fakeProvisioner{getErr: &acmpcatypes.RequestInProgressException{}})
			},
		},
		"failure-get-failure": {
			name: types.NamespacedName{Namespace: "ns1", Name: "cr1"},
			objects: []client.Object{
				cmgen.CertificateRequest(
					"cr1",
					cmgen.SetCertificateRequestNamespace("ns1"),
					cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
						Name:  "issuer1",
						Group: issuerapi.GroupVersion.Group,
						Kind:  "Issuer",
					}),
					cmgen.SetCertificateRequestStatusCondition(cmapi.CertificateRequestCondition{
						Type:   cmapi.CertificateRequestConditionReady,
						Status: cmmeta.ConditionUnknown,
					}),
				),
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
			expectedSignResult:           ctrl.Result{Requeue: true},
			expectedReadyConditionStatus: cmmeta.ConditionFalse,
			expectedReadyConditionReason: cmapi.CertificateRequestReasonFailed,
			expectedError:                false,
			mockProvisioner: func() {
				awspca.StoreProvisioner(types.NamespacedName{Namespace: "ns1", Name: "issuer1"}, &fakeProvisioner{getErr: errors.New("Get Failure")})
			},
		},
		"failure-issuer-not-ready": {
			name: types.NamespacedName{Namespace: "ns1", Name: "cr1"},
			objects: []client.Object{
				cmgen.CertificateRequest(
					"cr1",
					cmgen.SetCertificateRequestNamespace("ns1"),
					SetCreationTime(time.Now()),
					cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
						Name:  "issuer1",
						Group: issuerapi.GroupVersion.Group,
						Kind:  "Issuer",
					}),
					cmgen.SetCertificateRequestStatusCondition(cmapi.CertificateRequestCondition{
						Type:   cmapi.CertificateRequestConditionReady,
						Status: cmmeta.ConditionUnknown,
					}),
				),
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
								Status: metav1.ConditionFalse,
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
			expectedError:                true,
			expectedReadyConditionStatus: cmmeta.ConditionFalse,
			expectedReadyConditionReason: cmapi.CertificateRequestReasonFailed,
			mockProvisioner: func() {
				awspca.StoreProvisioner(types.NamespacedName{Namespace: "ns1", Name: "issuer1"}, &fakeProvisioner{caCert: []byte("cacert"), cert: []byte("cert")})
			},
		},
		"failure-issuer-not-found": {
			name: types.NamespacedName{Namespace: "ns1", Name: "cr1"},
			objects: []client.Object{
				cmgen.CertificateRequest(
					"cr1",
					cmgen.SetCertificateRequestNamespace("ns1"),
					SetCreationTime(time.Now()),
					cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
						Name:  "issuer1",
						Group: issuerapi.GroupVersion.Group,
						Kind:  "Issuer",
					}),
					cmgen.SetCertificateRequestStatusCondition(cmapi.CertificateRequestCondition{
						Type:   cmapi.CertificateRequestConditionReady,
						Status: cmmeta.ConditionUnknown,
					}),
				),
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
			expectedReadyConditionStatus: cmmeta.ConditionFalse,
			expectedReadyConditionReason: cmapi.CertificateRequestReasonFailed,
			expectedError:                true,
		},
		"failure-provisioner-not-found": {
			name: types.NamespacedName{Namespace: "ns1", Name: "cr1"},
			objects: []client.Object{
				cmgen.CertificateRequest(
					"cr1",
					cmgen.SetCertificateRequestNamespace("ns1"),
					SetCreationTime(time.Now()),
					cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
						Name:  "issuer1",
						Group: issuerapi.GroupVersion.Group,
						Kind:  "Issuer",
					}),
					cmgen.SetCertificateRequestStatusCondition(cmapi.CertificateRequestCondition{
						Type:   cmapi.CertificateRequestConditionReady,
						Status: cmmeta.ConditionUnknown,
					}),
				),
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
			expectedReadyConditionStatus: cmmeta.ConditionFalse,
			expectedReadyConditionReason: cmapi.CertificateRequestReasonFailed,
			expectedError:                true,
		},
		"failure-provisioner-not-found-temporary": {
			name: types.NamespacedName{Namespace: "ns1", Name: "cr1"},
			objects: []client.Object{
				cmgen.CertificateRequest(
					"cr1",
					cmgen.SetCertificateRequestNamespace("ns1"),
					SetCreationTime(time.Now()),
					cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
						Name:  "issuer1",
						Group: issuerapi.GroupVersion.Group,
						Kind:  "Issuer",
					}),
					cmgen.SetCertificateRequestStatusCondition(cmapi.CertificateRequestCondition{
						Type:   cmapi.CertificateRequestConditionReady,
						Status: cmmeta.ConditionUnknown,
					}),
				),
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
			expectedReadyConditionStatus: cmmeta.ConditionUnknown,
			expectedReadyConditionReason: "",
			expectedError:                true,
			expectedResult:               ctrl.Result{RequeueAfter: 1 * time.Hour},
			retryDuration:                1 * time.Hour,
		},
		"failure-sign-failure": {
			name: types.NamespacedName{Namespace: "ns1", Name: "cr1"},
			objects: []client.Object{
				cmgen.CertificateRequest(
					"cr1",
					cmgen.SetCertificateRequestNamespace("ns1"),
					SetCreationTime(time.Now()),
					cmgen.SetCertificateRequestIssuer(cmmeta.ObjectReference{
						Name:  "issuer1",
						Group: issuerapi.GroupVersion.Group,
						Kind:  "Issuer",
					}),
					cmgen.SetCertificateRequestStatusCondition(cmapi.CertificateRequestCondition{
						Type:   cmapi.CertificateRequestConditionReady,
						Status: cmmeta.ConditionUnknown,
					}),
				),
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
			expectedReadyConditionStatus: cmmeta.ConditionFalse,
			expectedReadyConditionReason: cmapi.CertificateRequestReasonFailed,
			expectedError:                false,
			mockProvisioner: func() {
				awspca.StoreProvisioner(types.NamespacedName{Namespace: "ns1", Name: "issuer1"}, &fakeProvisioner{signErr: errors.New("Sign Failure")})
			},
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
			controller := CertificateRequestReconciler{
				Client:           fakeClient,
				Log:              logrtesting.NewTestLogger(t),
				Scheme:           scheme,
				Recorder:         record.NewFakeRecorder(10),
				Clock:            clock.RealClock{},
				RequeueItter:     &fakeRequeueItter{},
				MaxRetryDuration: tc.retryDuration,
			}

			ctx := context.TODO()

			if tc.mockProvisioner != nil {
				tc.mockProvisioner()
			}

			result, signErr := controller.Reconcile(ctx, reconcile.Request{NamespacedName: tc.name})
			assert.Equal(t, tc.expectedSignResult, result, "Unexpected sign result")

			result, getErr := controller.Reconcile(ctx, reconcile.Request{NamespacedName: tc.name})
			assert.Equal(t, tc.expectedGetResult, result, "Unexpected get result")

			if tc.expectedError && (signErr == nil && getErr == nil) {
				assert.Fail(t, "Expected an error but got none")
			}

			var cr cmapi.CertificateRequest
			err := fakeClient.Get(ctx, tc.name, &cr)
			require.NoError(t, client.IgnoreNotFound(err), "unexpected error from fake client")
			if tc.expectedReadyConditionStatus != "" {
				assertCertificateRequestHasReadyCondition(t, tc.expectedReadyConditionStatus, tc.expectedReadyConditionReason, &cr)
			}

			if tc.expectedCertificate != nil {
				assert.Equal(t, tc.expectedCertificate, cr.Status.Certificate)
			}
			if tc.expectedCACertificate != nil {
				assert.Equal(t, tc.expectedCACertificate, cr.Status.CA)
			}
		})
	}
}

func assertCertificateRequestHasReadyCondition(t *testing.T, status cmmeta.ConditionStatus, reason string, cr *cmapi.CertificateRequest) {
	condition := cmutil.GetCertificateRequestCondition(cr, cmapi.CertificateRequestConditionReady)
	if !assert.NotNil(t, condition, "Ready condition not found") {
		return
	}
	assert.Equal(t, status, condition.Status, "unexpected condition status")
	validReasons := sets.NewString(
		cmapi.CertificateRequestReasonFailed,
		cmapi.CertificateRequestReasonIssued,
		"",
		cmapi.CertificateRequestReasonPending,
	)
	assert.Contains(t, validReasons, reason, "unexpected condition reason")
	assert.Equal(t, reason, condition.Reason, "unexpected condition reason")
}

func SetCreationTime(time time.Time) cmgen.CertificateRequestModifier {
	return func(c *cmapi.CertificateRequest) {
		c.SetCreationTimestamp(metav1.NewTime(time))
	}
}
