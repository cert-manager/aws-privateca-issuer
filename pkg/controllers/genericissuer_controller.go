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

package controllers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
	api "github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	awspca "github.com/cert-manager/aws-privateca-issuer/pkg/aws"
	"github.com/cert-manager/aws-privateca-issuer/pkg/util"
	"github.com/go-logr/logr"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const maxNotBeforeOffset = 24 * time.Hour

var (
	errNoArnInSpec             = errors.New("no Arn found in Issuer Spec")
	errNoRegionInSpec          = errors.New("no Region found in Issuer Spec")
	errNegativeNotBeforeOffset = errors.New("notBeforeOffset in Issuer Spec must not be negative")
	errLargeNotBeforeOffset    = fmt.Errorf("notBeforeOffset in Issuer Spec must not exceed %dh", int(maxNotBeforeOffset.Hours()))
	errServiceAccountNoName    = errors.New("serviceAccountRef.name is required")
	errServiceAccountNoRole    = errors.New("role must be set when using serviceAccountRef")
	errServiceAccountAndSecret = errors.New("set only one of secretRef and serviceAccountRef")
)

var awsDefaultRegion = os.Getenv("AWS_REGION")

// GenericIssuerReconciler reconciles both AWSPCAIssuer and AWSPCAClusterIssuer objects
type GenericIssuerReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	// GetCallerIdentitty should be set to true if you want to call and log the
	// result of sts.GetCallerIdentity.
	// This is useful to verify what AWS user is being authenticated by the Issuer,
	// but can be skipped during unit tests to avoid having a dependency on a
	// live STS service.
	GetCallerIdentity bool

	CredentialOptions awspca.CredentialOptions
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *GenericIssuerReconciler) Reconcile(ctx context.Context, req ctrl.Request, issuer api.GenericIssuer) (ctrl.Result, error) {
	log := r.Log.WithValues("genericissuer", req.NamespacedName)
	spec := issuer.GetSpec()
	err := validateIssuer(spec)
	if err != nil {
		log.Error(err, "failed to validate issuer")
		_ = r.setStatus(ctx, issuer, metav1.ConditionFalse, "Validation", fmt.Sprintf("Failed to validate resource: %v", err))
		return ctrl.Result{}, err
	}

	awspca.DeleteProvisioner(ctx, r.Client, req.NamespacedName)
	cfg, err := awspca.GetConfig(ctx, r.Client, issuer, r.CredentialOptions)
	if err != nil {
		log.Error(err, "Error loading config")
		_ = r.setStatus(ctx, issuer, metav1.ConditionFalse, "Error", err.Error())
		return ctrl.Result{}, err
	}

	if r.GetCallerIdentity {
		id, err := sts.NewFromConfig(cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			log.Error(err, "failed to sts.GetCallerIdentity")
			_ = r.setStatus(ctx, issuer, metav1.ConditionFalse, "Error", stableErrorMessage(err))
			return ctrl.Result{}, err
		}
		log.Info("sts.GetCallerIdentity", "arn", id.Arn, "account", id.Account, "user_id", id.UserId)
	}

	return ctrl.Result{}, r.setStatus(ctx, issuer, metav1.ConditionTrue, "Verified", "Issuer verified")
}

func (r *GenericIssuerReconciler) setStatus(ctx context.Context, issuer api.GenericIssuer, status metav1.ConditionStatus, reason, message string) error {
	log := r.Log.WithValues("genericissuer", issuer.GetName())
	util.SetIssuerCondition(log, issuer, api.ConditionTypeReady, status, reason, message)

	eventType := core.EventTypeNormal
	if status == metav1.ConditionFalse {
		eventType = core.EventTypeWarning
	}
	r.Recorder.Event(issuer, eventType, reason, message)

	return r.Client.Status().Update(ctx, issuer)
}

// AWS errors embed a per-request ID. Dropping it keeps the status unchanged while
// the cause is, since every status change re-triggers a reconcile and so an STS call.
func stableErrorMessage(err error) string {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return fmt.Sprintf("%s: %s", apiErr.ErrorCode(), apiErr.ErrorMessage())
	}
	return err.Error()
}

func validateIssuer(spec *api.AWSPCAIssuerSpec) error {
	switch {
	case spec.ServiceAccountRef != nil && spec.ServiceAccountRef.Name == "":
		return errServiceAccountNoName
	case spec.ServiceAccountRef != nil && spec.Role == "":
		return errServiceAccountNoRole
	case spec.ServiceAccountRef != nil && spec.SecretRef.Name != "":
		return errServiceAccountAndSecret
	case spec.Arn == "":
		return errNoArnInSpec
	case spec.Region == "" && awsDefaultRegion == "":
		return errNoRegionInSpec
	case spec.NotBeforeOffset != nil && spec.NotBeforeOffset.Duration < 0:
		return errNegativeNotBeforeOffset
	case spec.NotBeforeOffset != nil && spec.NotBeforeOffset.Duration > maxNotBeforeOffset:
		return errLargeNotBeforeOffset
	}
	return nil
}
