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

func (issCtx *IssuerContext) createClusterIssuer(ctx context.Context, caType string) error {
	return issCtx.createClusterIssuerWithSpec(ctx, caType, getIssuerSpec(caType))
}

func (issCtx *IssuerContext) createClusterIssuerWithRole(ctx context.Context) error {
	return issCtx.createClusterIssuerWithSpec(ctx, "RSA", getIssuerSpecWithRole("RSA"))
}

func (issCtx *IssuerContext) createClusterIssuerWithSpec(ctx context.Context, caType string, spec v1beta1.AWSPCAIssuerSpec) error {
	if issCtx.issuerName == "" {
		issCtx.issuerName = uuid.New().String() + "--cluster-issuer--" + strings.ToLower(caType)
	}

	issCtx.issuerType = "AWSPCAClusterIssuer"
	issSpec := v1beta1.AWSPCAClusterIssuer{
		ObjectMeta: metav1.ObjectMeta{Name: issCtx.issuerName},
		Spec:       spec,
	}

	if issCtx.secretRef != (v1beta1.AWSCredentialsSecretReference{}) {
		issSpec.Spec.SecretRef = issCtx.secretRef
	}

	_, err := testContext.iclient.AWSPCAClusterIssuers().Create(ctx, &issSpec, metav1.CreateOptions{})

	if err != nil {
		assert.FailNow(godog.T(ctx), "Could not create Cluster Issuer: "+err.Error())
	}

	err = waitForClusterIssuerReady(ctx, testContext.iclient, issCtx.issuerName)

	if err != nil {
		assert.FailNow(godog.T(ctx), "Cluster issuer did not reach a ready state: "+err.Error())
	}

	return nil
}

func (issCtx *IssuerContext) deleteClusterIssuer(ctx context.Context) error {
	err := testContext.iclient.AWSPCAClusterIssuers().Delete(ctx, issCtx.issuerName, metav1.DeleteOptions{})

	if err != nil {
		assert.FailNow(godog.T(ctx), "Issuer was not successfully deleted: "+err.Error())
	}

	return nil
}
