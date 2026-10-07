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
	"fmt"

	api "github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var realtimeClock clock.Clock = clock.RealClock{}

const (
	// IssuerKind is the kind of the namespaced issuer resource
	IssuerKind = "AWSPCAIssuer"
	// ClusterIssuerKind is the kind of the cluster-scoped issuer resource
	ClusterIssuerKind = "AWSPCAClusterIssuer"
)

// IsSupportedIssuerKind reports whether kind names an issuer type this controller handles
func IsSupportedIssuerKind(kind string) bool {
	return kind == IssuerKind || kind == ClusterIssuerKind
}

// GetIssuer returns the AWSPCAIssuer or AWSPCAClusterIssuer named by kind and name.
// Only the resource of the requested kind is looked up.
func GetIssuer(ctx context.Context, client client.Client, kind string, name types.NamespacedName) (api.GenericIssuer, error) {
	switch kind {
	case IssuerKind:
		iss := new(api.AWSPCAIssuer)
		if err := client.Get(ctx, name, iss); err != nil {
			return nil, err
		}
		return iss, nil
	case ClusterIssuerKind:
		ciss := new(api.AWSPCAClusterIssuer)
		cname := types.NamespacedName{Name: name.Name}
		if err := client.Get(ctx, cname, ciss); err != nil {
			return nil, err
		}
		return ciss, nil
	default:
		return nil, fmt.Errorf("unsupported issuer kind %q", kind)
	}
}

// SetIssuerCondition sets the ready state of an issuer and updates it in the cluster
func SetIssuerCondition(log logr.Logger, issuer api.GenericIssuer, conditionType string, status metav1.ConditionStatus, reason, message string) {
	newCondition := metav1.Condition{
		Type:    conditionType,
		Status:  status,
		Reason:  reason,
		Message: message,
	}

	now := metav1.NewTime(realtimeClock.Now())
	newCondition.LastTransitionTime = now

	for idx, cond := range issuer.GetStatus().Conditions {
		if cond.Type != conditionType {
			continue
		}

		if cond.Status == status {
			newCondition.LastTransitionTime = cond.LastTransitionTime
		}

		issuer.GetStatus().Conditions[idx] = newCondition
		return
	}

	issuer.GetStatus().Conditions = append(issuer.GetStatus().Conditions, newCondition)
}
