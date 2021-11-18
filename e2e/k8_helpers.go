package main

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmv1 "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmclientv1 "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"

	clientV1beta1 "github.com/cert-manager/aws-privateca-issuer/pkg/clientset/v1beta1"

	issuerapi "github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"

	"k8s.io/apimachinery/pkg/util/wait"
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

func waitForCertificateReady(ctx context.Context, client *cmclientv1.CertmanagerV1Client, name string, namespace string) error {
	return wait.PollImmediate(250*time.Millisecond, 2*time.Minute,
		func() (bool, error) {

			certificate, err := client.Certificates(namespace).Get(ctx, name, metav1.GetOptions{})

			if err != nil {
				return false, fmt.Errorf("error getting Certificate %q: %v", name, err)
			}

			for _, condition := range certificate.Status.Conditions {
				if condition.Type == v1.CertificateConditionReady && condition.Status == cmv1.ConditionTrue {
					return true, nil
				}
			}
			return false, nil
		})
}
