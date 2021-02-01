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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/go-logr/logr"
	api "github.com/jniebuhr/aws-pca-issuer/pkg/api/v1beta1"
	awspca "github.com/jniebuhr/aws-pca-issuer/pkg/aws"
	"github.com/jniebuhr/aws-pca-issuer/pkg/util"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var awsDefaultRegion = os.Getenv("AWS_REGION")

// GenericIssuerReconciler reconciles both AWSPCAIssuer and AWSPCAClusterIssuer objects
type GenericIssuerReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
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

	var region *string
	if spec.Region != "" {
		region = aws.String(spec.Region)
	} else {
		if awsDefaultRegion != "" {
			region = aws.String(awsDefaultRegion)
		} else {
			err := fmt.Errorf("unable to determine an AWS region to use")
			log.Error(err, "unable to determine an AWS region to use")
			_ = r.setStatus(ctx, issuer, metav1.ConditionFalse, "Error", "unable to determine an AWS region to use")
			return ctrl.Result{}, err
		}
	}

	var clientCredentials *credentials.Credentials
	if spec.SecretRef.Name != "" {
		secretNamespaceName := types.NamespacedName{
			Namespace: spec.SecretRef.Namespace,
			Name:      spec.SecretRef.Name,
		}

		secret := new(core.Secret)
		if err := r.Client.Get(ctx, secretNamespaceName, secret); err != nil {
			log.Error(err, "failed to retrieve AWS secret")
			_ = r.setStatus(ctx, issuer, metav1.ConditionFalse, "Error", "Failed to retrieve secret: %v", err)
			return ctrl.Result{}, err
		}

		accessKey, ok := secret.Data["AWS_ACCESS_KEY_ID"]
		if !ok {
			err := fmt.Errorf("secret does not contain AWS_ACCESS_KEY_ID")
			log.Error(err, "secret value AWS_ACCESS_KEY_ID was not found")
			_ = r.setStatus(ctx, issuer, metav1.ConditionFalse, "Error", "secret value AWS_ACCESS_KEY_ID was not found")
			return ctrl.Result{}, err
		}

		secretKey, ok := secret.Data["AWS_SECRET_ACCESS_KEY"]
		if !ok {
			err := fmt.Errorf("secret does not contain AWS_SECRET_ACCESS_KEY")
			log.Error(err, "secret value AWS_SECRET_ACCESS_KEY was not found")
			_ = r.setStatus(ctx, issuer, metav1.ConditionFalse, "Error", "secret value AWS_SECRET_ACCESS_KEY was not found")
			return ctrl.Result{}, err
		}

		clientCredentials = credentials.NewStaticCredentials(string(accessKey), string(secretKey), "")
	}

	sess, err := session.NewSession(&aws.Config{
		Region:      region,
		Credentials: clientCredentials,
	})
	if err != nil {
		log.Error(err, "failed to create AWS session")
		_ = r.setStatus(ctx, issuer, metav1.ConditionFalse, "Error", "Failed to create AWS session")
		return ctrl.Result{}, err
	}

	awspca.StoreProvisioner(req.NamespacedName, awspca.NewProvisioner(sess, spec.Arn))

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
		return fmt.Errorf("spec.arn cannot be empty")
	case spec.Region == "" && awsDefaultRegion == "":
		return fmt.Errorf("spec.region cannot be empty if no default region is specified")
	}
	return nil
}
