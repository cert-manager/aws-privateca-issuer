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
	"k8s.io/client-go/kubernetes"
)

func waitForIssuerReady(ctx context.Context, client *clientV1beta1.Client, name string, namespace string) error {
	return wait.PollImmediate(250*time.Millisecond, 2*time.Minute,
		func() (bool, error) {

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
	return wait.PollImmediate(250*time.Millisecond, 2*time.Minute,
		func() (bool, error) {

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
	return wait.PollImmediate(250*time.Millisecond, 2*time.Minute,
		func() (bool, error) {

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
	return wait.PollImmediate(250*time.Millisecond, 2*time.Minute,
		func() (bool, error) {

			_, err := client.CertificateRequests(namespace).Get(ctx, name, metav1.GetOptions{})

			if err != nil {
				return false, nil
			}

			return true, nil
		})
}

func waitForCertificateState(ctx context.Context, client *cmclientv1.CertmanagerV1Client, name string, namespace string, reason string, status string) error {
	return wait.PollImmediate(250*time.Millisecond, 2*time.Minute,
		func() (bool, error) {

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

func getCertificateData(ctx context.Context, clientset *kubernetes.Clientset, namespace string, secretName string) (string, error) {
	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error getting certificate secret %q: %v", secretName, err)
	}

	// The certificate is stored in the 'tls.crt' key of the secret data
	certBytes, exists := secret.Data["tls.crt"]
	if !exists {
		return "", fmt.Errorf("certificate data not found in secret %q", secretName)
	}

	return string(certBytes), nil
}
