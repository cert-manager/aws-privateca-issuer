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
	return issCtx.createNamespaceIssuerWithSpec(ctx, caType, getIssuerSpec(caType))
}

func (issCtx *IssuerContext) createNamespaceIssuerWithRole(ctx context.Context) error {
	return issCtx.createNamespaceIssuerWithSpec(ctx, "RSA", getIssuerSpecWithRole("RSA"))
}

func (issCtx *IssuerContext) verifyNamespaceIssuerFails(ctx context.Context, message string) error {
	err := waitForIssuerState(ctx, testContext.iclient, issCtx.issuerName, issCtx.namespace, metav1.ConditionFalse, message)

	if err != nil {
		assert.FailNow(godog.T(ctx), "Namespace issuer did not fail with the expected message: "+err.Error())
	}

	return nil
}

func (issCtx *IssuerContext) createNamespaceIssuerWithSpec(ctx context.Context, caType string, spec v1beta1.AWSPCAIssuerSpec) error {
	issCtx.issuerName = uuid.New().String() + "--namespace-issuer--" + strings.ToLower(caType)
	issCtx.issuerType = "AWSPCAIssuer"
	issSpec := v1beta1.AWSPCAIssuer{
		ObjectMeta: metav1.ObjectMeta{Name: issCtx.issuerName},
		Spec:       spec,
	}

	if issCtx.secretRef != (v1beta1.AWSCredentialsSecretReference{}) {
		issSpec.Spec.SecretRef = issCtx.secretRef
	}

	_, err := testContext.iclient.AWSPCAIssuers(issCtx.namespace).Create(ctx, &issSpec, metav1.CreateOptions{})

	if err != nil {
		assert.FailNow(godog.T(ctx), "Could not create namespace issuer: "+err.Error())
	}

	return nil
}
