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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	api "github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	awspca "github.com/cert-manager/aws-privateca-issuer/pkg/aws"
	"github.com/cert-manager/aws-privateca-issuer/pkg/util"
	"github.com/go-logr/logr"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	errNoSecretAccessKey = errors.New("no AWS Secret Access Key Found")
	errNoAccessKeyID     = errors.New("no AWS Access Key ID Found")
	errNoArnInSpec       = errors.New("no Arn found in Issuer Spec")
	errNoRegionInSpec    = errors.New("no Region found in Issuer Spec")
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
		_ = r.setStatus(ctx, issuer, metav1.ConditionFalse, "Validation", "Failed to validate resource: %v", err)
		return ctrl.Result{}, err
	}

	var cfg, cfgErr = r.getConfig(ctx, spec)

	if cfgErr != nil {
		log.Error(cfgErr, "Error loading config")
		_ = r.setStatus(ctx, issuer, metav1.ConditionFalse, "Error", cfgErr.Error())
		return ctrl.Result{}, err
	}

	if r.GetCallerIdentity {
		id, err := sts.NewFromConfig(cfg).GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			log.Error(err, "failed to sts.GetCallerIdentity")
			return ctrl.Result{}, err
		}
		log.Info("sts.GetCallerIdentity", "arn", id.Arn, "account", id.Account, "user_id", id.UserId)
	}

	log.Info("Calling StoreProvisioner")
	awspca.StoreProvisioner(req.NamespacedName, awspca.NewProvisioner(cfg, spec.Arn))

	return ctrl.Result{}, r.setStatus(ctx, issuer, metav1.ConditionTrue, "Verified", "Issuer verified")
}

func (r *GenericIssuerReconciler) setStatus(ctx context.Context, issuer api.GenericIssuer, status metav1.ConditionStatus, reason, message string, args ...interface{}) error {
	log := r.Log.WithValues("genericissuer", issuer.GetName())
	completeMessage := fmt.Sprintf(message, args...)
	util.SetIssuerCondition(log, issuer, api.ConditionTypeReady, status, reason, completeMessage)

	eventType := core.EventTypeNormal
	if status == metav1.ConditionFalse {
		eventType = core.EventTypeWarning
	}
	r.Recorder.Event(issuer, eventType, reason, completeMessage)

	return r.Client.Status().Update(ctx, issuer)
}

func validateIssuer(spec *api.AWSPCAIssuerSpec) error {
	switch {
	case spec.Arn == "":
		return fmt.Errorf(errNoArnInSpec.Error())
	case spec.Region == "" && awsDefaultRegion == "":
		return fmt.Errorf(errNoRegionInSpec.Error())
	}
	return nil
}

func (r *GenericIssuerReconciler) getConfig(ctx context.Context, spec *api.AWSPCAIssuerSpec) (aws.Config, error) {
	if spec.Region != "" {
		return config.LoadDefaultConfig(ctx,
			config.WithRegion(spec.Region),
		)
	}

	if spec.SecretRef.Name != "" {
		secretNamespaceName := types.NamespacedName{
			Namespace: spec.SecretRef.Namespace,
			Name:      spec.SecretRef.Name,
		}

		secret := new(core.Secret)
		if err := r.Client.Get(ctx, secretNamespaceName, secret); err != nil {
			return aws.Config{}, fmt.Errorf("failed to retrieve secret: %v", err)
		}

		accessKey, ok := secret.Data["AWS_ACCESS_KEY_ID"]
		if !ok {
			return aws.Config{}, errNoAccessKeyID
		}

		secretKey, ok := secret.Data["AWS_SECRET_ACCESS_KEY"]
		if !ok {
			return aws.Config{}, errNoSecretAccessKey
		}

		return config.LoadDefaultConfig(ctx,
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(string(accessKey), string(secretKey), "")),
		)
	}

	return aws.Config{}, nil
}
