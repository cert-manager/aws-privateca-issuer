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

package util

import (
	"context"
	"github.com/go-logr/logr"
	api "github.com/jniebuhr/aws-pca-issuer/pkg/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var Clock clock.Clock = clock.RealClock{}

func GetIssuer(ctx context.Context, client client.Client, name types.NamespacedName) (api.GenericIssuer, error) {
	iss := new(api.AWSPCAIssuer)
	err := client.Get(ctx, name, iss)
	if err != nil {
		ciss := new(api.AWSPCAClusterIssuer)
		cname := types.NamespacedName{
			Name: name.Name,
		}
		err = client.Get(ctx, cname, ciss)
		if err != nil {
			return nil, err
		}
		return ciss, nil
	}
	return iss, nil
}

func SetIssuerCondition(log logr.Logger, issuer api.GenericIssuer, conditionType string, status metav1.ConditionStatus, reason, message string) {
	newCondition := metav1.Condition{
		Type:    conditionType,
		Status:  status,
		Reason:  reason,
		Message: message,
	}

	now := metav1.NewTime(Clock.Now())
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
