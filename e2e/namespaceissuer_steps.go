package main

import (
	"context"
	"strings"

	"github.com/cucumber/godog"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (issCtx *IssuerContext) createNamespaceIssuer(ctx context.Context, caType string) error {
	issCtx.issuerName = uuid.New().String() + "--namespace-issuer--" + strings.ToLower(caType)
	issCtx.issuerType = "AWSPCAIssuer"
	issSpec := v1beta1.AWSPCAIssuer{
		ObjectMeta: metav1.ObjectMeta{Name: issCtx.issuerName},
		Spec:       getIssuerSpec(caType),
	}

	if issCtx.secretRef != (v1beta1.AWSCredentialsSecretReference{}) {
		issSpec.Spec.SecretRef = issCtx.secretRef
	}

	_, err := testContext.iclient.AWSPCAIssuers(issCtx.namespace).Create(ctx, &issSpec, metav1.CreateOptions{})

	if err != nil {
		assert.FailNow(godog.T(ctx), "Could not create namespace issuer: "+err.Error())
	}

	err = waitForIssuerReady(ctx, testContext.iclient, issCtx.issuerName, issCtx.namespace)

	if err != nil {
		assert.FailNow(godog.T(ctx), "Namespace issuer did not reach a ready state: "+err.Error())
	}

	return nil
}
