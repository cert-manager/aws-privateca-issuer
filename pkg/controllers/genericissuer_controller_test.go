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
	"fmt"
	"testing"

	logrtesting "github.com/go-logr/logr/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	issuerapi "github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
)

const (
	ClusterIssuer = "ClusterIssuer"
)

func TestIssuerReconcile(t *testing.T) {
	type testCase struct {
		kind                         string
		name                         types.NamespacedName
		objects                      []client.Object
		expectedResult               ctrl.Result
		expectedError                error
		expectedReadyConditionStatus metav1.ConditionStatus
	}

	tests := map[string]testCase{
		"success-with-secret-selector": {
			name: types.NamespacedName{Namespace: "ns1", Name: "issuer1"},
			objects: []client.Object{
				&issuerapi.AWSPCAIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer1",
						Namespace: "ns1",
					},
					Spec: issuerapi.AWSPCAIssuerSpec{
						SecretRef: issuerapi.SecretReference{
							Name:      "issuer1-credentials",
							Namespace: "ns1",
							SecretAccessKeySelector: issuerapi.SecretSelector{
								Key: "fake-secret-access-key",
							},
							AccessKeyIDSelector: issuerapi.SecretSelector{
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
								Status: metav1.ConditionUnknown,
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
			expectedReadyConditionStatus: metav1.ConditionTrue,
			expectedResult:               ctrl.Result{},
		},
		"success-issuer": {
			name: types.NamespacedName{Namespace: "ns1", Name: "issuer1"},
			objects: []client.Object{
				&issuerapi.AWSPCAIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer1",
						Namespace: "ns1",
					},
					Spec: issuerapi.AWSPCAIssuerSpec{
						SecretRef: issuerapi.SecretReference{
							Name:      "issuer1-credentials",
							Namespace: "ns1",
						},
						Region: "us-east-1",
						Arn:    "arn:aws:acm-pca:us-east-1:account:certificate-authority/12345678-1234-1234-1234-123456789012",
					},
					Status: issuerapi.AWSPCAIssuerStatus{
						Conditions: []metav1.Condition{
							{
								Type:   issuerapi.ConditionTypeReady,
								Status: metav1.ConditionUnknown,
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
			expectedReadyConditionStatus: metav1.ConditionTrue,
			expectedResult:               ctrl.Result{},
		},
		"success-cluster-issuer": {
			kind: ClusterIssuer,
			name: types.NamespacedName{Name: "clusterissuer1"},
			objects: []client.Object{
				&issuerapi.AWSPCAClusterIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "clusterissuer1",
					},
					Spec: issuerapi.AWSPCAIssuerSpec{
						SecretRef: issuerapi.SecretReference{
							Name: "clusterissuer1-credentials",
						},
						Region: "us-east-1",
						Arn:    "arn:aws:acm-pca:us-east-1:account:certificate-authority/12345678-1234-1234-1234-123456789012",
					},
					Status: issuerapi.AWSPCAIssuerStatus{
						Conditions: []metav1.Condition{
							{
								Type:   issuerapi.ConditionTypeReady,
								Status: metav1.ConditionUnknown,
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
			expectedReadyConditionStatus: metav1.ConditionTrue,
			expectedResult:               ctrl.Result{},
		},
		"failure-issuer-no-region-specified": {
			name: types.NamespacedName{Namespace: "ns1", Name: "issuer1"},
			objects: []client.Object{
				&issuerapi.AWSPCAIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer1",
						Namespace: "ns1",
					},
					Spec: issuerapi.AWSPCAIssuerSpec{
						SecretRef: issuerapi.SecretReference{
							Name:      "issuer1-credentials",
							Namespace: "ns1",
						},
						Arn: "arn:aws:acm-pca:us-east-1:account:certificate-authority/12345678-1234-1234-1234-123456789012",
					},
					Status: issuerapi.AWSPCAIssuerStatus{
						Conditions: []metav1.Condition{
							{
								Type:   issuerapi.ConditionTypeReady,
								Status: metav1.ConditionUnknown,
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
			expectedReadyConditionStatus: metav1.ConditionFalse,
			expectedError:                errNoRegionInSpec,
			expectedResult:               ctrl.Result{},
		},
		"failure-issuer-no-arn-specified": {
			name: types.NamespacedName{Namespace: "ns1", Name: "issuer1"},
			objects: []client.Object{
				&issuerapi.AWSPCAIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer1",
						Namespace: "ns1",
					},
					Spec: issuerapi.AWSPCAIssuerSpec{
						SecretRef: issuerapi.SecretReference{
							Name:      "issuer1-credentials",
							Namespace: "ns1",
						},
						Region: "us-east-1",
					},
					Status: issuerapi.AWSPCAIssuerStatus{
						Conditions: []metav1.Condition{
							{
								Type:   issuerapi.ConditionTypeReady,
								Status: metav1.ConditionUnknown,
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
			expectedReadyConditionStatus: metav1.ConditionFalse,
			expectedError:                errNoArnInSpec,
			expectedResult:               ctrl.Result{},
		},
		"failure-issuer-no-access-key-specified": {
			name: types.NamespacedName{Namespace: "ns1", Name: "issuer1"},
			objects: []client.Object{
				&issuerapi.AWSPCAIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer1",
						Namespace: "ns1",
					},
					Spec: issuerapi.AWSPCAIssuerSpec{
						SecretRef: issuerapi.SecretReference{
							Name:      "issuer1-credentials",
							Namespace: "ns1",
						},
						Region: "us-east-1",
						Arn:    "arn:aws:acm-pca:us-east-1:account:certificate-authority/12345678-1234-1234-1234-123456789012",
					},
					Status: issuerapi.AWSPCAIssuerStatus{
						Conditions: []metav1.Condition{
							{
								Type:   issuerapi.ConditionTypeReady,
								Status: metav1.ConditionUnknown,
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
			expectedReadyConditionStatus: metav1.ConditionFalse,
			expectedError:                errNoAccessKeyID,
			expectedResult:               ctrl.Result{},
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
						SecretRef: issuerapi.SecretReference{
							Name:      "issuer1-credentials",
							Namespace: "ns1",
							AccessKeyIDSelector: issuerapi.SecretSelector{
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
								Status: metav1.ConditionUnknown,
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
			expectedReadyConditionStatus: metav1.ConditionFalse,
			expectedError:                errNoAccessKeyID,
			expectedResult:               ctrl.Result{},
		},
		"failure-issuer-no-secret-access-key-specified": {
			name: types.NamespacedName{Namespace: "ns1", Name: "issuer1"},
			objects: []client.Object{
				&issuerapi.AWSPCAIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer1",
						Namespace: "ns1",
					},
					Spec: issuerapi.AWSPCAIssuerSpec{
						SecretRef: issuerapi.SecretReference{
							Name:      "issuer1-credentials",
							Namespace: "ns1",
						},
						Region: "us-east-1",
						Arn:    "arn:aws:acm-pca:us-east-1:account:certificate-authority/12345678-1234-1234-1234-123456789012",
					},
					Status: issuerapi.AWSPCAIssuerStatus{
						Conditions: []metav1.Condition{
							{
								Type:   issuerapi.ConditionTypeReady,
								Status: metav1.ConditionUnknown,
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
			expectedReadyConditionStatus: metav1.ConditionFalse,
			expectedError:                errNoSecretAccessKey,
			expectedResult:               ctrl.Result{},
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
						SecretRef: issuerapi.SecretReference{
							Name:      "issuer1-credentials",
							Namespace: "ns1",
							SecretAccessKeySelector: issuerapi.SecretSelector{
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
								Status: metav1.ConditionUnknown,
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
			expectedReadyConditionStatus: metav1.ConditionFalse,
			expectedError:                errNoSecretAccessKey,
			expectedResult:               ctrl.Result{},
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, issuerapi.AddToScheme(scheme))
	require.NoError(t, v1.AddToScheme(scheme))

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tc.objects...).
				Build()

			controller := GenericIssuerReconciler{
				Client:   fakeClient,
				Log:      &logrtesting.TestLogger{T: t},
				Scheme:   scheme,
				Recorder: record.NewFakeRecorder(10),
			}

			var (
				result reconcile.Result
				err    error
				status issuerapi.AWSPCAIssuerStatus
			)

			ctx := context.TODO()

			if tc.kind == ClusterIssuer {
				iss := new(issuerapi.AWSPCAClusterIssuer)

				require.NoError(t, controller.Client.Get(ctx, tc.name, iss))

				result, err = controller.Reconcile(ctx, reconcile.Request{NamespacedName: tc.name}, iss)

				status = iss.Status
			} else {
				iss := new(issuerapi.AWSPCAIssuer)

				require.NoError(t, controller.Client.Get(ctx, tc.name, iss))

				result, err = controller.Reconcile(ctx, reconcile.Request{NamespacedName: tc.name}, iss)

				status = iss.Status
			}

			if tc.expectedError != nil {
				assertErrorIs(t, tc.expectedError, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tc.expectedResult, result, "Unexpected result")

			if tc.expectedReadyConditionStatus != "" {
				assertIssuerHasReadyCondition(t, tc.expectedReadyConditionStatus, &status)
			}
		})
	}
}

func assertErrorIs(t *testing.T, expectedError, actualError error) {
	if !assert.Error(t, actualError) {
		return
	}
	assert.Equal(t, actualError, expectedError, "Errors do not match!")
}

func assertIssuerHasReadyCondition(t *testing.T, status metav1.ConditionStatus, issuerStatus *issuerapi.AWSPCAIssuerStatus) {
	fmt.Printf("%v", issuerStatus.Conditions)
	assert.Equal(t, status, issuerStatus.Conditions[0].Status, "unexpected condition status")
}
