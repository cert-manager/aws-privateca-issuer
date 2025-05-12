package main

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	issuerapi "github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	clientV1beta1 "github.com/cert-manager/aws-privateca-issuer/pkg/clientset/v1beta1"
	cmclientv1 "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"

	"k8s.io/apimachinery/pkg/util/wait"
)

func waitForIssuerReady(ctx context.Context, client *clientV1beta1.Client, name string, namespace string) error {
	return wait.PollUntilContextTimeout(ctx, 250*time.Millisecond, 2*time.Minute, true,
		func(ctx context.Context) (bool, error) {
			issuer, err := client.AWSPCAIssuers(namespace).Get(ctx, name, metav1.GetOptions{})

			if err != nil {
				return false, fmt.Errorf("error getting Issuer %q: %v", name, err)
			}
			for _, condition := range issuer.Status.Conditions {
				if condition.Type == issuerapi.ConditionTypeReady && condition.Status == metav1.ConditionTrue {
					return true, nil
				}
			}
			return false, nil
		})
}

func waitForClusterIssuerReady(ctx context.Context, client *clientV1beta1.Client, name string) error {
	return wait.PollUntilContextTimeout(ctx, 250*time.Millisecond, 2*time.Minute, true,
		func(ctx context.Context) (bool, error) {
			issuer, err := client.AWSPCAClusterIssuers().Get(ctx, name, metav1.GetOptions{})

			if err != nil {
				return false, fmt.Errorf("error getting Cluster Issuer %q: %v", name, err)
			}

			for _, condition := range issuer.Status.Conditions {
				if condition.Type == issuerapi.ConditionTypeReady && condition.Status == metav1.ConditionTrue {
					return true, nil
				}
			}

			return false, nil
		})
}

func waitForCertificateRequestState(ctx context.Context, client *cmclientv1.CertmanagerV1Client, name string, namespace string, reason string, status string) error {
	return wait.PollUntilContextTimeout(ctx, 250*time.Millisecond, 2*time.Minute, true,
		func(ctx context.Context) (bool, error) {
			cr, err := client.CertificateRequests(namespace).Get(ctx, name, metav1.GetOptions{})

			if err != nil {
				return false, fmt.Errorf("error getting CertificateRequest %q: %v", name, err)
			}

			for _, condition := range cr.Status.Conditions {
				if condition.Reason == reason && string(condition.Status) == status {
					return true, nil
				}
			}
			return false, nil
		})
}

func waitForCertificateRequestToBeCreated(ctx context.Context, client *cmclientv1.CertmanagerV1Client, name string, namespace string) error {
	return wait.PollUntilContextTimeout(ctx, 250*time.Millisecond, 2*time.Minute, true,
		func(ctx context.Context) (bool, error) {
			_, err := client.CertificateRequests(namespace).Get(ctx, name, metav1.GetOptions{})

			if err != nil {
				return false, nil
			}

			return true, nil
		})
}

func waitForCertificateState(ctx context.Context, client *cmclientv1.CertmanagerV1Client, name string, namespace string, reason string, status string) error {
	return wait.PollUntilContextTimeout(ctx, 250*time.Millisecond, 2*time.Minute, true,
		func(ctx context.Context) (bool, error) {
			certificate, err := client.Certificates(namespace).Get(ctx, name, metav1.GetOptions{})

			if err != nil {
				return false, fmt.Errorf("error getting Certificate %q: %v", name, err)
			}

			for _, condition := range certificate.Status.Conditions {
				if condition.Reason == reason && string(condition.Status) == status {
					return true, nil
				}
			}
			return false, nil
		})
}
