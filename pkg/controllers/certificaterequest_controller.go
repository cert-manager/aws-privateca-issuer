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
	"math/rand"
	"time"

	acmpcatypes "github.com/aws/aws-sdk-go-v2/service/acmpca/types"
	"github.com/cert-manager/aws-privateca-issuer/pkg/aws"
	"github.com/cert-manager/aws-privateca-issuer/pkg/util"
	cmapi "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	"github.com/go-logr/logr"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	cmutil "github.com/cert-manager/cert-manager/pkg/api/util"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
)

type RequeueItter interface {
	RequeueAfter() time.Duration
}

type requeueItter struct {
}

func (r *requeueItter) RequeueAfter() time.Duration {
	return 1*time.Minute + time.Duration(rand.Intn(60))*time.Second
}

func NewRequeueItter() RequeueItter {
	return &requeueItter{}
}

// CertificateRequestReconciler reconciles a AWSPCAIssuer object
type CertificateRequestReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	Clock                  clock.Clock
	RequeueItter           RequeueItter
	CheckApprovedCondition bool
	MaxRetryDuration       time.Duration
}

// +kubebuilder:rbac:groups=cert-manager.io,resources=certificaterequests,verbs=get;list;watch;update
// +kubebuilder:rbac:groups=cert-manager.io,resources=certificaterequests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *CertificateRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("certificaterequest", req.NamespacedName)
	cr := new(cmapi.CertificateRequest)
	if err := r.Client.Get(ctx, req.NamespacedName, cr); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		log.Error(err, "Failed to request CertificateRequest")
		return ctrl.Result{}, err
	}

	if cr.Spec.IssuerRef.Group != api.GroupVersion.Group {
		log.V(4).Info("CertificateRequest does not specify an issuerRef matching our group")
		return ctrl.Result{}, nil
	}

	// Ignore CertificateRequest if it is already Ready
	if cmutil.CertificateRequestHasCondition(cr, cmapi.CertificateRequestCondition{
		Type:   cmapi.CertificateRequestConditionReady,
		Status: cmmeta.ConditionTrue,
	}) {
		log.V(4).Info("CertificateRequest is Ready. Ignoring.")
		return ctrl.Result{}, nil
	}
	// Ignore CertificateRequest if it is already Failed
	if cmutil.CertificateRequestHasCondition(cr, cmapi.CertificateRequestCondition{
		Type:   cmapi.CertificateRequestConditionReady,
		Status: cmmeta.ConditionFalse,
		Reason: cmapi.CertificateRequestReasonFailed,
	}) {
		log.V(4).Info("CertificateRequest is Failed. Ignoring.")
		return ctrl.Result{}, nil
	}
	// Ignore CertificateRequest if it already has a Denied Ready Reason
	if cmutil.CertificateRequestHasCondition(cr, cmapi.CertificateRequestCondition{
		Type:   cmapi.CertificateRequestConditionReady,
		Status: cmmeta.ConditionFalse,
		Reason: cmapi.CertificateRequestReasonDenied,
	}) {
		log.V(4).Info("CertificateRequest already has a Ready condition with Denied Reason. Ignoring.")
		return ctrl.Result{}, nil
	}

	// If CertificateRequest has been denied, mark the CertificateRequest as
	// Ready=Denied and set FailureTime if not already.
	if cmutil.CertificateRequestIsDenied(cr) {
		log.V(4).Info("CertificateRequest has been denied. Marking as failed.")

		if cr.Status.FailureTime == nil {
			nowTime := metav1.NewTime(r.Clock.Now())
			cr.Status.FailureTime = &nowTime
		}

		message := "The CertificateRequest was denied by an approval controller"
		return ctrl.Result{}, r.setPermanentStatus(ctx, cr, cmmeta.ConditionFalse, cmapi.CertificateRequestReasonDenied, message)
	}

	if r.CheckApprovedCondition {
		// If CertificateRequest has not been approved, exit early.
		if !cmutil.CertificateRequestIsApproved(cr) {
			log.V(4).Info("certificate request has not been approved")
			return ctrl.Result{}, nil
		}
	}

	if len(cr.Status.Certificate) > 0 {
		log.V(4).Info("Certificate was already signed")
		return ctrl.Result{}, nil
	}

	issuerName := types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      cr.Spec.IssuerRef.Name,
	}
	if cr.Spec.IssuerRef.Kind == "AWSPCAClusterIssuer" {
		issuerName.Namespace = ""
	}

	retry, requeue, err := r.HandleSignRequest(ctx, log, issuerName, cr)

	if err != nil {
		now := r.Clock.Now()
		creationTime := cr.GetCreationTimestamp()
		pastMaxRetryDuration := now.After(creationTime.Add(r.MaxRetryDuration))

		if pastMaxRetryDuration || !retry {
			return ctrl.Result{Requeue: requeue}, r.setPermanentStatus(ctx, cr, cmmeta.ConditionFalse, cmapi.CertificateRequestReasonFailed, "Permanent error signing certificate: %s", err.Error())
		}

		return ctrl.Result{
			RequeueAfter: r.RequeueItter.RequeueAfter(),
		}, r.setTemporaryStatus(ctx, cr, cmmeta.ConditionFalse, cmapi.CertificateRequestReasonFailed, "Temporary error signing certificate, retry again: %s", err.Error())
	}

	return ctrl.Result{Requeue: requeue}, r.setPermanentStatus(ctx, cr, cmmeta.ConditionTrue, cmapi.CertificateRequestReasonIssued, "Certificate issued")
}

func (r *CertificateRequestReconciler) HandleSignRequest(ctx context.Context, log logr.Logger, issuerName types.NamespacedName, cr *cmapi.CertificateRequest) (retry bool, requeue bool, error) {
	iss, err := util.GetIssuer(ctx, r.Client, issuerName)
	if err != nil {
		return true, false, fmt.Errorf("failed to retrieve Issuer resource: %w", err)
	}

	if !isReady(iss) {
		return true, false, fmt.Errorf("issuer %s is not ready", iss.GetName())
	}

	provisioner, ok := aws.GetProvisioner(issuerName)
	if !ok {
		return true, false, fmt.Errorf("provisioner for %s not found (name: %s, namespace: %s)", issuerName, issuerName.Name, issuerName.Namespace)
	}

	certArn, exists := cr.ObjectMeta.GetAnnotations()["aws-privateca-issuer/certificate-arn"]
	if !exists {
		err := provisioner.Sign(ctx, cr, log)
		if err != nil {
			log.Error(err, "failed to request certificate from PCA")
			return false, fmt.Errorf("failed to sign certificat from PCA: %w", err)
		}

		return true, true, nil
	}

	pem, ca, err := provisioner.Get(ctx, cr, certArn, log)
	if err != nil {
		var errorType *acmpcatypes.RequestInProgressException
		if errors.As(err, &errorType) {
			log.Info("certificate is still issuing")
			return false, false,fmt.Errorf("waiting for certificate to be issued: %w", err)
		}

		log.Error(err, "failed to issue certificate from PCA")
		return false,true, fmt.Errorf("failed to issue certificate from PCA: %w", err)
	}

	cr.Status.Certificate = pem
	cr.Status.CA = ca

	return true, false, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CertificateRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cmapi.CertificateRequest{}).
		Complete(r)
}

func isReady(issuer api.GenericIssuer) bool {
	for _, condition := range issuer.GetStatus().Conditions {
		if condition.Type == api.ConditionTypeReady && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

func (r *CertificateRequestReconciler) setStatusInternal(ctx context.Context, cr *cmapi.CertificateRequest, permanent bool, status cmmeta.ConditionStatus, reason, message string, args ...interface{}) error {
	completeMessage := fmt.Sprintf(message, args...)

	eventType := core.EventTypeNormal
	if status == cmmeta.ConditionFalse {
		eventType = core.EventTypeWarning
	}
	r.Recorder.Event(cr, eventType, reason, completeMessage)

	cmutil.SetCertificateRequestCondition(cr, api.ConditionTypeIssuing, status, reason, completeMessage)
	if permanent {
		cmutil.SetCertificateRequestCondition(cr, api.ConditionTypeReady, status, reason, completeMessage)
	}
	r.Client.Status().Update(ctx, cr)

	if reason == cmapi.CertificateRequestReasonFailed {
		return fmt.Errorf(completeMessage)
	}

	return nil
}

func (r *CertificateRequestReconciler) setPermanentStatus(ctx context.Context, cr *cmapi.CertificateRequest, status cmmeta.ConditionStatus, reason, message string, args ...interface{}) error {
	return r.setStatusInternal(ctx, cr, true, status, reason, message, args...)
}

func (r *CertificateRequestReconciler) setTemporaryStatus(ctx context.Context, cr *cmapi.CertificateRequest, status cmmeta.ConditionStatus, reason, message string, args ...interface{}) error {
	return r.setStatusInternal(ctx, cr, false, status, reason, message, args...)
}
