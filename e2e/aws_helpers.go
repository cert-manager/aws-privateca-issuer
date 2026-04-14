package main

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/acmpca"
	"github.com/aws/aws-sdk-go-v2/service/acmpca/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/ram"
	ramtypes "github.com/aws/aws-sdk-go-v2/service/ram/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type statementEntry struct {
	Effect   string
	Action   []string
	Resource string
}

type policyDocument struct {
	Version   string
	Statement []statementEntry
}

type caParams struct {
	signingAlgorithm types.SigningAlgorithm
	keyAlgorithm     types.KeyAlgorithm
	caType           types.CertificateAuthorityType
	commonName       string
	issuerCAArn      *string
	templateArn      string
	validity         types.Validity
}

func createUser(ctx context.Context, cfg aws.Config) (string, string) {
	iamClient := iam.NewFromConfig(cfg)

	policy := policyDocument{
		Version: "2012-10-17",
		Statement: []statementEntry{
			{
				Effect: "Allow",
				Action: []string{
					"acm-pca:DescribeCertificateAuthority",
					"acm-pca:GetCertificate",
					"acm-pca:IssueCertificate",
				},
				Resource: "*",
			},
		},
	}

	policyJSON, err := json.Marshal(&policy)
	if err != nil {
		panic(err.Error())
	}

	policyName := "CMPolicy" + strconv.FormatInt(time.Now().Unix(), 10)

	policyParams := iam.CreatePolicyInput{
		PolicyName:     aws.String(policyName),
		PolicyDocument: aws.String(string(policyJSON)),
	}

	policyOutput, policyErr := iamClient.CreatePolicy(ctx, &policyParams)

	if policyErr != nil {
		panic(policyErr.Error())
	}

	policyArn := policyOutput.Policy.Arn

	userName := "CMUser" + strconv.FormatInt(time.Now().Unix(), 10)

	userParams := iam.CreateUserInput{
		UserName:            aws.String(userName),
		PermissionsBoundary: policyArn,
	}

	_, userErr := iamClient.CreateUser(ctx, &userParams)

	if userErr != nil {
		panic(userErr.Error())
	}

	attachParams := iam.AttachUserPolicyInput{
		UserName:  aws.String(userName),
		PolicyArn: policyOutput.Policy.Arn,
	}

	_, attachErr := iamClient.AttachUserPolicy(ctx, &attachParams)

	if attachErr != nil {
		panic(attachErr.Error())
	}

	return userName, *policyArn
}

func createAccessKey(ctx context.Context, cfg aws.Config, userName string) (string, string) {
	iamClient := iam.NewFromConfig(cfg)

	createKeyParams := iam.CreateAccessKeyInput{
		UserName: aws.String(userName),
	}

	createKeyOutput, createKeyErr := iamClient.CreateAccessKey(ctx, &createKeyParams)

	if createKeyErr != nil {
		panic(createKeyErr.Error())
	}

	return *createKeyOutput.AccessKey.AccessKeyId, *createKeyOutput.AccessKey.SecretAccessKey
}

func deleteUser(ctx context.Context, cfg aws.Config, userName string, policyArn string) {
	iamClient := iam.NewFromConfig(cfg)

	detachParams := iam.DetachUserPolicyInput{
		UserName:  aws.String(userName),
		PolicyArn: aws.String(policyArn),
	}

	_, detachErr := iamClient.DetachUserPolicy(ctx, &detachParams)

	if detachErr != nil {
		panic(detachErr.Error())
	}

	deleteParams := iam.DeleteUserInput{
		UserName: aws.String(userName),
	}

	_, deleteErr := iamClient.DeleteUser(ctx, &deleteParams)

	if deleteErr != nil {
		panic(deleteErr.Error())
	}
}

func deleteAccessKey(ctx context.Context, cfg aws.Config, userName string, accessKey string) {
	iamClient := iam.NewFromConfig(cfg)

	deleteKeyParams := iam.DeleteAccessKeyInput{
		AccessKeyId: aws.String(accessKey),
		UserName:    aws.String(userName),
	}

	_, deleteKeyErr := iamClient.DeleteAccessKey(ctx, &deleteKeyParams)

	if deleteKeyErr != nil {
		panic(deleteKeyErr.Error())
	}
}

func deleteCertificateAuthority(ctx context.Context, cfg aws.Config, caArn string) {
	pcaClient := acmpca.NewFromConfig(cfg)

	updateCAParams := acmpca.UpdateCertificateAuthorityInput{
		CertificateAuthorityArn: &caArn,
		Status:                  types.CertificateAuthorityStatusDisabled,
	}

	_, updateErr := pcaClient.UpdateCertificateAuthority(ctx, &updateCAParams)

	if updateErr != nil {
		panic(updateErr.Error())
	}

	deleteCAParams := acmpca.DeleteCertificateAuthorityInput{
		CertificateAuthorityArn:     &caArn,
		PermanentDeletionTimeInDays: aws.Int32(7),
	}

	_, deleteErr := pcaClient.DeleteCertificateAuthority(ctx, &deleteCAParams)

	if deleteErr != nil {
		panic(deleteErr.Error())
	}

}

func (testCtx *TestContext) createCertificateAuthority(ctx context.Context, cfg aws.Config, isRSA bool) string {
	var caParams caParams
	caParams.caType = types.CertificateAuthorityTypeRoot
	caParams.commonName = "CMTest-" + strconv.FormatInt(time.Now().Unix(), 10)
	caParams.issuerCAArn = nil // will be self-signed later
	caParams.templateArn = "arn:aws:acm-pca:::template/RootCACertificate/V1"
	caParams.validity = types.Validity{
		Type:  types.ValidityPeriodTypeYears,
		Value: aws.Int64(365),
	}

	return createCertificateAuthority(ctx, cfg, caParams, isRSA)
}

func createSubCertificateAuthority(ctx context.Context, cfg aws.Config, isRSA bool, parentCAArn string) string {
	var caParams caParams

	caParams.caType = types.CertificateAuthorityTypeSubordinate
	caParams.commonName = "CMSubordinate-" + strconv.FormatInt(time.Now().Unix(), 10)
	caParams.issuerCAArn = &parentCAArn
	caParams.templateArn = "arn:aws:acm-pca:::template/SubordinateCACertificate_PathLen1/V1"
	caParams.validity = types.Validity{
		Type:  types.ValidityPeriodTypeYears,
		Value: aws.Int64(30),
	}

	return createCertificateAuthority(ctx, cfg, caParams, isRSA)
}

func createCertificateAuthority(ctx context.Context, cfg aws.Config, caParams caParams, isRSA bool) string {
	pcaClient := acmpca.NewFromConfig(cfg)

	if isRSA {
		caParams.signingAlgorithm = types.SigningAlgorithmSha256withrsa
		caParams.keyAlgorithm = types.KeyAlgorithmRsa2048
	} else {
		caParams.signingAlgorithm = types.SigningAlgorithmSha256withecdsa
		caParams.keyAlgorithm = types.KeyAlgorithmEcPrime256v1
	}

	createCertificateAuthorityParams := acmpca.CreateCertificateAuthorityInput{
		CertificateAuthorityType: caParams.caType,
		CertificateAuthorityConfiguration: &types.CertificateAuthorityConfiguration{
			KeyAlgorithm:     caParams.keyAlgorithm,
			SigningAlgorithm: caParams.signingAlgorithm,
			Subject: &types.ASN1Subject{
				CommonName: aws.String(caParams.commonName),
			},
		},
	}

	createOutput, createErr := pcaClient.CreateCertificateAuthority(ctx, &createCertificateAuthorityParams)

	if createErr != nil {
		panic(createErr.Error())
	}

	caArn := createOutput.CertificateAuthorityArn

	// For root CAs, the CA signs its own certificate
	if caParams.issuerCAArn == nil {
		caParams.issuerCAArn = caArn
	}

	getCsrParams := acmpca.GetCertificateAuthorityCsrInput{
		CertificateAuthorityArn: caArn,
	}

	csrWaiter := acmpca.NewCertificateAuthorityCSRCreatedWaiter(pcaClient)
	csrWaiterErr := csrWaiter.Wait(ctx, &getCsrParams, 1*time.Minute)

	if csrWaiterErr != nil {
		panic(csrWaiterErr.Error())
	}

	csrOutput, csrErr := pcaClient.GetCertificateAuthorityCsr(ctx, &getCsrParams)

	if csrErr != nil {
		panic(csrErr.Error())
	}

	issuerCertificateParams := acmpca.IssueCertificateInput{
		CertificateAuthorityArn: caParams.issuerCAArn,
		Csr:                     []byte(*csrOutput.Csr),
		SigningAlgorithm:        caParams.signingAlgorithm,
		TemplateArn:             aws.String(caParams.templateArn),
		Validity:                &caParams.validity,
	}

	issueOutput, issueErr := pcaClient.IssueCertificate(ctx, &issuerCertificateParams)

	if issueErr != nil {
		panic(issueErr.Error())
	}

	getCertParams := acmpca.GetCertificateInput{
		CertificateArn:          issueOutput.CertificateArn,
		CertificateAuthorityArn: caParams.issuerCAArn,
	}

	certWaiter := acmpca.NewCertificateIssuedWaiter(pcaClient)
	certWaiterErr := certWaiter.Wait(ctx, &getCertParams, 2*time.Minute)

	if certWaiterErr != nil {
		panic(certWaiterErr.Error())
	}

	getCertOutput, getCertErr := pcaClient.GetCertificate(ctx, &getCertParams)
	if getCertErr != nil {
		panic(getCertErr.Error())
	}

	importCertParams := acmpca.ImportCertificateAuthorityCertificateInput{
		Certificate:             []byte(*getCertOutput.Certificate),
		CertificateAuthorityArn: caArn,
	}

	// For subordinate CAs, include the certificate chain
	if getCertOutput.CertificateChain != nil {
		importCertParams.CertificateChain = []byte(*getCertOutput.CertificateChain)
	}

	_, importCertErr := pcaClient.ImportCertificateAuthorityCertificate(ctx, &importCertParams)

	if importCertErr != nil {
		panic(importCertErr.Error())
	}

	return *caArn
}

func getAccountID(ctx context.Context, cfg aws.Config) string {
	stsClient := sts.NewFromConfig(cfg)

	callerID, callerErr := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})

	if callerErr != nil {
		panic(callerErr.Error())
	}

	return *callerID.Account
}

func getCallerIdentity(ctx context.Context, cfg aws.Config) *sts.GetCallerIdentityOutput {
	stsClient := sts.NewFromConfig(cfg)

	callerID, callerErr := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})

	if callerErr != nil {
		panic(callerErr.Error())
	}

	return callerID
}

func assumeRole(ctx context.Context, cfg aws.Config, roleName string, region string) aws.Config {

	stsClient := sts.NewFromConfig(cfg)

	creds := stscreds.NewAssumeRoleProvider(stsClient, roleName)

	xaConfig, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))

	if err != nil {
		panic(err)
	}

	xaConfig.Credentials = aws.NewCredentialsCache(creds)

	return xaConfig
}

func shareCA(ctx context.Context, cfg aws.Config, xaCfg aws.Config, xaCAArn string, permissionArn string) string {

	callerAccount := getAccountID(ctx, cfg)

	xaRAMClient := ram.NewFromConfig(xaCfg)

	resourceInput := ram.CreateResourceShareInput{
		Name:         aws.String("CM_XA_RESOURCE_SHARE"),
		ResourceArns: []string{xaCAArn},
		Principals:   []string{callerAccount},
		PermissionArns: []string{
			permissionArn,
		},
	}

	resourceOutput, resourceErr := xaRAMClient.CreateResourceShare(ctx, &resourceInput)

	if resourceErr != nil {
		panic(resourceErr.Error())
	}

	resourceShareArn := resourceOutput.ResourceShare.ResourceShareArn

	//Wait for share propogation
	time.Sleep(5 * time.Second)

	ramClient := ram.NewFromConfig(cfg)

	invitesInputs := ram.GetResourceShareInvitationsInput{
		ResourceShareArns: []string{*resourceShareArn},
	}

	invitesOutput, inviteErr := ramClient.GetResourceShareInvitations(ctx, &invitesInputs)

	if inviteErr != nil {
		panic(inviteErr.Error())
	}

	acceptInput := ram.AcceptResourceShareInvitationInput{
		ResourceShareInvitationArn: invitesOutput.ResourceShareInvitations[0].ResourceShareInvitationArn,
	}

	_, acceptErr := ramClient.AcceptResourceShareInvitation(ctx, &acceptInput)

	if acceptErr != nil {
		panic(acceptErr.Error())
	}

	timeout := time.Now().Add(3 * time.Minute)

	shareAssociated := false

	for time.Now().Before(timeout) {
		assocInput := ram.GetResourceShareAssociationsInput{
			AssociationType:   ramtypes.ResourceShareAssociationTypeResource,
			ResourceShareArns: []string{*resourceShareArn},
		}

		assocOutput, assocErr := xaRAMClient.GetResourceShareAssociations(ctx, &assocInput)

		if assocErr != nil {
			log.Printf("GetResourceShareAssociationError: %s", assocErr.Error())
		} else if assocOutput.ResourceShareAssociations[0].Status == ramtypes.ResourceShareAssociationStatusAssociated {
			shareAssociated = true
			break
		}

		log.Printf("Waiting for share to associate...")
		time.Sleep(5 * time.Second)
	}

	if !shareAssociated {
		panic("RAM share failed to associate on XA CA")
	}

	//Wait for policy propogation
	time.Sleep(60 * time.Second)

	return *resourceOutput.ResourceShare.ResourceShareArn
}

func deleteResourceShare(ctx context.Context, cfg aws.Config, resourceShareArn string) {
	ramClient := ram.NewFromConfig(cfg)

	deleteInput := ram.DeleteResourceShareInput{
		ResourceShareArn: aws.String(resourceShareArn),
	}

	_, deleteErr := ramClient.DeleteResourceShare(ctx, &deleteInput)

	if deleteErr != nil {
		panic(deleteErr.Error())
	}
}
