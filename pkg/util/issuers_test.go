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

	api "github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetIssuer(t *testing.T) {
	namespacedIssuer := &api.AWSPCAIssuer{
		ObjectMeta: metav1.ObjectMeta{Name: "shared", Namespace: "ns1"},
	}
	clusterIssuer := &api.AWSPCAClusterIssuer{
		ObjectMeta: metav1.ObjectMeta{Name: "shared"},
	}

	tests := map[string]struct {
		objects      []client.Object
		kind         string
		name         types.NamespacedName
		expectedType api.GenericIssuer
		expectedErr  bool
		notFound     bool
	}{
		"namespaced-issuer-found": {
			objects:      []client.Object{namespacedIssuer, clusterIssuer},
			kind:         IssuerKind,
			name:         types.NamespacedName{Namespace: "ns1", Name: "shared"},
			expectedType: &api.AWSPCAIssuer{},
		},
		"cluster-issuer-found": {
			objects:      []client.Object{namespacedIssuer, clusterIssuer},
			kind:         ClusterIssuerKind,
			name:         types.NamespacedName{Name: "shared"},
			expectedType: &api.AWSPCAClusterIssuer{},
		},
		"namespaced-issuer-missing-does-not-use-cluster-issuer": {
			objects:     []client.Object{clusterIssuer},
			kind:        IssuerKind,
			name:        types.NamespacedName{Namespace: "ns1", Name: "shared"},
			expectedErr: true,
			notFound:    true,
		},
		"cluster-issuer-missing-does-not-use-namespaced-issuer": {
			objects:     []client.Object{namespacedIssuer},
			kind:        ClusterIssuerKind,
			name:        types.NamespacedName{Name: "shared"},
			expectedErr: true,
			notFound:    true,
		},
		"unsupported-kind": {
			objects:     []client.Object{namespacedIssuer, clusterIssuer},
			kind:        "Issuer",
			name:        types.NamespacedName{Namespace: "ns1", Name: "shared"},
			expectedErr: true,
		},
	}

	scheme := runtime.NewScheme()
	require.NoError(t, api.AddToScheme(scheme))

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tc.objects...).
				Build()

			iss, err := GetIssuer(context.TODO(), fakeClient, tc.kind, tc.name)
			if tc.expectedErr {
				require.Error(t, err)
				assert.Nil(t, iss)
				assert.Equal(t, tc.notFound, apierrors.IsNotFound(err))
				return
			}
			require.NoError(t, err)
			assert.IsType(t, tc.expectedType, iss)
			assert.Equal(t, tc.name.Name, iss.GetName())
		})
	}
}

func TestGetIssuerSameNameBothKinds(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, api.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			&api.AWSPCAIssuer{
				ObjectMeta: metav1.ObjectMeta{Name: "shared", Namespace: "ns1"},
				Spec:       api.AWSPCAIssuerSpec{Arn: "namespaced-arn"},
			},
			&api.AWSPCAClusterIssuer{
				ObjectMeta: metav1.ObjectMeta{Name: "shared"},
				Spec:       api.AWSPCAIssuerSpec{Arn: "cluster-arn"},
			},
		).
		Build()

	iss, err := GetIssuer(context.TODO(), fakeClient, IssuerKind, types.NamespacedName{Namespace: "ns1", Name: "shared"})
	require.NoError(t, err)
	assert.IsType(t, &api.AWSPCAIssuer{}, iss)
	assert.Equal(t, "ns1", iss.GetNamespace())
	assert.Equal(t, "namespaced-arn", iss.GetSpec().Arn)

	ciss, err := GetIssuer(context.TODO(), fakeClient, ClusterIssuerKind, types.NamespacedName{Name: "shared"})
	require.NoError(t, err)
	assert.IsType(t, &api.AWSPCAClusterIssuer{}, ciss)
	assert.Equal(t, "", ciss.GetNamespace())
	assert.Equal(t, "cluster-arn", ciss.GetSpec().Arn)
}

func TestIsSupportedIssuerKind(t *testing.T) {
	assert.True(t, IsSupportedIssuerKind(IssuerKind))
	assert.True(t, IsSupportedIssuerKind(ClusterIssuerKind))
	assert.False(t, IsSupportedIssuerKind("Issuer"))
	assert.False(t, IsSupportedIssuerKind("ClusterIssuer"))
	assert.False(t, IsSupportedIssuerKind(""))
}
