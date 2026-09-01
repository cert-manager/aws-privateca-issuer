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

package util

import (
	"context"
	"testing"
	"time"

	issuerapi "github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	logrtesting "github.com/go-logr/logr/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clocktesting "k8s.io/utils/clock/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetIssuer(t *testing.T) {
	type testCase struct {
		objects      []client.Object
		name         types.NamespacedName
		expectError  bool
		expectType   string // "AWSPCAIssuer" or "AWSPCAClusterIssuer"
	}

	tests := map[string]testCase{
		"success-issuer-found": {
			name: types.NamespacedName{Namespace: "ns1", Name: "issuer1"},
			objects: []client.Object{
				&issuerapi.AWSPCAIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer1",
						Namespace: "ns1",
					},
					Spec: issuerapi.AWSPCAIssuerSpec{
						Arn:    "arn:aws:acm-pca:us-east-1:account:certificate-authority/12345678",
						Region: "us-east-1",
					},
				},
			},
			expectError: false,
			expectType:  "AWSPCAIssuer",
		},
		"success-cluster-issuer-fallback": {
			name: types.NamespacedName{Namespace: "ns1", Name: "issuer1"},
			objects: []client.Object{
				&issuerapi.AWSPCAClusterIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "issuer1",
					},
					Spec: issuerapi.AWSPCAIssuerSpec{
						Arn:    "arn:aws:acm-pca:us-east-1:account:certificate-authority/12345678",
						Region: "us-east-1",
					},
				},
			},
			expectError: false,
			expectType:  "AWSPCAClusterIssuer",
		},
		"failure-neither-exists": {
			name:        types.NamespacedName{Namespace: "ns1", Name: "issuer1"},
			objects:     []client.Object{},
			expectError: true,
		},
		"success-issuer-preferred-over-cluster-issuer": {
			name: types.NamespacedName{Namespace: "ns1", Name: "issuer1"},
			objects: []client.Object{
				&issuerapi.AWSPCAIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "issuer1",
						Namespace: "ns1",
					},
					Spec: issuerapi.AWSPCAIssuerSpec{
						Arn:    "arn:aws:acm-pca:us-east-1:account:certificate-authority/issuer",
						Region: "us-east-1",
					},
				},
				&issuerapi.AWSPCAClusterIssuer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "issuer1",
					},
					Spec: issuerapi.AWSPCAIssuerSpec{
						Arn:    "arn:aws:acm-pca:us-east-1:account:certificate-authority/cluster",
						Region: "us-east-1",
					},
				},
			},
			expectError: false,
			expectType:  "AWSPCAIssuer",
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, issuerapi.AddToScheme(scheme))

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.TODO()
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tc.objects...).
				Build()

			result, err := GetIssuer(ctx, fakeClient, tc.name)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				switch tc.expectType {
				case "AWSPCAIssuer":
					assert.IsType(t, &issuerapi.AWSPCAIssuer{}, result)
				case "AWSPCAClusterIssuer":
					assert.IsType(t, &issuerapi.AWSPCAClusterIssuer{}, result)
				}
			}
		})
	}
}

func TestSetIssuerCondition(t *testing.T) {
	fixedTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	fakeClock := clocktesting.NewFakeClock(fixedTime)

	// Override the package-level clock for deterministic tests
	originalClock := realtimeClock
	realtimeClock = fakeClock
	t.Cleanup(func() {
		realtimeClock = originalClock
	})

	type testCase struct {
		issuer        issuerapi.GenericIssuer
		conditionType string
		status        metav1.ConditionStatus
		reason        string
		message       string
		validate      func(t *testing.T, issuer issuerapi.GenericIssuer)
	}

	tests := map[string]testCase{
		"add-new-condition-to-empty-list": {
			issuer: &issuerapi.AWSPCAIssuer{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns1"},
				Status:     issuerapi.AWSPCAIssuerStatus{Conditions: []metav1.Condition{}},
			},
			conditionType: issuerapi.ConditionTypeReady,
			status:        metav1.ConditionTrue,
			reason:        "Verified",
			message:       "issuer is ready",
			validate: func(t *testing.T, issuer issuerapi.GenericIssuer) {
				conditions := issuer.GetStatus().Conditions
				require.Len(t, conditions, 1)
				assert.Equal(t, issuerapi.ConditionTypeReady, conditions[0].Type)
				assert.Equal(t, metav1.ConditionTrue, conditions[0].Status)
				assert.Equal(t, "Verified", conditions[0].Reason)
				assert.Equal(t, "issuer is ready", conditions[0].Message)
				assert.Equal(t, metav1.NewTime(fixedTime), conditions[0].LastTransitionTime)
			},
		},
		"update-existing-condition-same-status": {
			issuer: &issuerapi.AWSPCAIssuer{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns1"},
				Status: issuerapi.AWSPCAIssuerStatus{
					Conditions: []metav1.Condition{
						{
							Type:               issuerapi.ConditionTypeReady,
							Status:             metav1.ConditionTrue,
							Reason:             "OldReason",
							Message:            "old message",
							LastTransitionTime: metav1.NewTime(fixedTime.Add(-1 * time.Hour)),
						},
					},
				},
			},
			conditionType: issuerapi.ConditionTypeReady,
			status:        metav1.ConditionTrue,
			reason:        "NewReason",
			message:       "new message",
			validate: func(t *testing.T, issuer issuerapi.GenericIssuer) {
				conditions := issuer.GetStatus().Conditions
				require.Len(t, conditions, 1)
				assert.Equal(t, "NewReason", conditions[0].Reason)
				assert.Equal(t, "new message", conditions[0].Message)
				// LastTransitionTime should be preserved because status didn't change
				assert.Equal(t, metav1.NewTime(fixedTime.Add(-1*time.Hour)), conditions[0].LastTransitionTime)
			},
		},
		"update-existing-condition-different-status": {
			issuer: &issuerapi.AWSPCAIssuer{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns1"},
				Status: issuerapi.AWSPCAIssuerStatus{
					Conditions: []metav1.Condition{
						{
							Type:               issuerapi.ConditionTypeReady,
							Status:             metav1.ConditionTrue,
							Reason:             "OldReason",
							Message:            "old message",
							LastTransitionTime: metav1.NewTime(fixedTime.Add(-1 * time.Hour)),
						},
					},
				},
			},
			conditionType: issuerapi.ConditionTypeReady,
			status:        metav1.ConditionFalse,
			reason:        "Failed",
			message:       "issuer failed",
			validate: func(t *testing.T, issuer issuerapi.GenericIssuer) {
				conditions := issuer.GetStatus().Conditions
				require.Len(t, conditions, 1)
				assert.Equal(t, metav1.ConditionFalse, conditions[0].Status)
				assert.Equal(t, "Failed", conditions[0].Reason)
				assert.Equal(t, "issuer failed", conditions[0].Message)
				// LastTransitionTime should change because status changed
				assert.Equal(t, metav1.NewTime(fixedTime), conditions[0].LastTransitionTime)
			},
		},
		"update-middle-condition": {
			issuer: &issuerapi.AWSPCAIssuer{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns1"},
				Status: issuerapi.AWSPCAIssuerStatus{
					Conditions: []metav1.Condition{
						{
							Type:               "First",
							Status:             metav1.ConditionTrue,
							Reason:             "FirstReason",
							Message:            "first",
							LastTransitionTime: metav1.NewTime(fixedTime.Add(-2 * time.Hour)),
						},
						{
							Type:               "Second",
							Status:             metav1.ConditionTrue,
							Reason:             "SecondReason",
							Message:            "second",
							LastTransitionTime: metav1.NewTime(fixedTime.Add(-2 * time.Hour)),
						},
						{
							Type:               "Third",
							Status:             metav1.ConditionTrue,
							Reason:             "ThirdReason",
							Message:            "third",
							LastTransitionTime: metav1.NewTime(fixedTime.Add(-2 * time.Hour)),
						},
					},
				},
			},
			conditionType: "Second",
			status:        metav1.ConditionFalse,
			reason:        "UpdatedSecond",
			message:       "updated second",
			validate: func(t *testing.T, issuer issuerapi.GenericIssuer) {
				conditions := issuer.GetStatus().Conditions
				require.Len(t, conditions, 3)

				// First condition unchanged
				assert.Equal(t, "First", conditions[0].Type)
				assert.Equal(t, "FirstReason", conditions[0].Reason)
				assert.Equal(t, metav1.NewTime(fixedTime.Add(-2*time.Hour)), conditions[0].LastTransitionTime)

				// Second condition updated
				assert.Equal(t, "Second", conditions[1].Type)
				assert.Equal(t, metav1.ConditionFalse, conditions[1].Status)
				assert.Equal(t, "UpdatedSecond", conditions[1].Reason)
				assert.Equal(t, "updated second", conditions[1].Message)
				assert.Equal(t, metav1.NewTime(fixedTime), conditions[1].LastTransitionTime)

				// Third condition unchanged
				assert.Equal(t, "Third", conditions[2].Type)
				assert.Equal(t, "ThirdReason", conditions[2].Reason)
				assert.Equal(t, metav1.NewTime(fixedTime.Add(-2*time.Hour)), conditions[2].LastTransitionTime)
			},
		},
		"add-new-condition-type": {
			issuer: &issuerapi.AWSPCAIssuer{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns1"},
				Status: issuerapi.AWSPCAIssuerStatus{
					Conditions: []metav1.Condition{
						{
							Type:               issuerapi.ConditionTypeReady,
							Status:             metav1.ConditionTrue,
							Reason:             "Verified",
							Message:            "ready",
							LastTransitionTime: metav1.NewTime(fixedTime.Add(-1 * time.Hour)),
						},
					},
				},
			},
			conditionType: "CustomType",
			status:        metav1.ConditionFalse,
			reason:        "CustomReason",
			message:       "custom message",
			validate: func(t *testing.T, issuer issuerapi.GenericIssuer) {
				conditions := issuer.GetStatus().Conditions
				require.Len(t, conditions, 2)

				// Original condition unchanged
				assert.Equal(t, issuerapi.ConditionTypeReady, conditions[0].Type)
				assert.Equal(t, metav1.NewTime(fixedTime.Add(-1*time.Hour)), conditions[0].LastTransitionTime)

				// New condition appended
				assert.Equal(t, "CustomType", conditions[1].Type)
				assert.Equal(t, metav1.ConditionFalse, conditions[1].Status)
				assert.Equal(t, "CustomReason", conditions[1].Reason)
				assert.Equal(t, "custom message", conditions[1].Message)
				assert.Equal(t, metav1.NewTime(fixedTime), conditions[1].LastTransitionTime)
			},
		},
	}

	log := logrtesting.NewTestLogger(t)

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			SetIssuerCondition(log, tc.issuer, tc.conditionType, tc.status, tc.reason, tc.message)
			tc.validate(t, tc.issuer)
		})
	}
}
