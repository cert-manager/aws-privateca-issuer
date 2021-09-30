package main

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cmv1 "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmclientv1 "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"

	clientV1beta1 "github.com/cert-manager/aws-privateca-issuer/pkg/clientset/v1beta1"

	"k8s.io/apimachinery/pkg/util/wait"
)

func waitForIssuerReady(ctx context.Context, client *clientV1beta1.Client, name string, namespace string) error {
	return wait.PollImmediate(500*time.Millisecond, time.Minute,
		func() (bool, error) {

			issuer, err := client.AWSPCAIssuers(namespace).Get(ctx, name, metav1.GetOptions{})

			if err != nil {
				return false, fmt.Errorf("error getting Issuer %q: %v", name, err)
			}
			if len(issuer.Status.Conditions) > 0 && issuer.Status.Conditions[0].Status == metav1.ConditionTrue {
				return true, nil
			}
			return false, nil
		})
}

func waitForClusterIssuerReady(ctx context.Context, client *clientV1beta1.Client, name string) error {
	return wait.PollImmediate(500*time.Millisecond, time.Minute,
		func() (bool, error) {

			issuer, err := client.AWSPCAClusterIssuers().Get(ctx, name, metav1.GetOptions{})

			if err != nil {
				return false, fmt.Errorf("error getting Cluster Issuer %q: %v", name, err)
			}

			if len(issuer.Status.Conditions) > 0 && issuer.Status.Conditions[0].Status == metav1.ConditionTrue {
				return true, nil
			}
			return false, nil
		})
}

func waitForCertificateReady(ctx context.Context, client *cmclientv1.CertmanagerV1Client, name string, namespace string) error {
	return wait.PollImmediate(500*time.Millisecond, time.Minute,
		func() (bool, error) {

			certificate, err := client.Certificates(namespace).Get(ctx, name, metav1.GetOptions{})

			if err != nil {
				return false, fmt.Errorf("error getting Certificate %q: %v", name, err)
			}
			if len(certificate.Status.Conditions) > 0 && certificate.Status.Conditions[0].Status == cmv1.ConditionTrue {
				return true, nil
			}
			return false, nil
		})

}
