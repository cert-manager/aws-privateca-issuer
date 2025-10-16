package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	clientV1beta1 "github.com/cert-manager/aws-privateca-issuer/pkg/clientset/v1beta1"
	cmclientv1 "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"
	"github.com/cucumber/godog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var testContext *TestContext
var errDetails []string

// These are variables shared by all of the tests
type TestContext struct {
	iclient   *clientV1beta1.Client
	cmClient  *cmclientv1.CertmanagerV1Client
	clientset *kubernetes.Clientset
	xaCfg     aws.Config
	caArns    map[string]string

	region, partition, accessKey, secretKey, endEntityResourceShareArn, subordinateCaResourceShareArn, userName, policyArn, roleToAssume string
}

// These are variables specific to each test
type IssuerContext struct {
	certName   string
	issuerName string
	issuerType string
	namespace  string
	secretRef  v1beta1.AWSCredentialsSecretReference
}

var opts = godog.Options{
	Concurrency: 8,
	Format:      "pretty",
	Paths:       []string{"features"},
	Strict:      true,
}

const (
	KubeConfigPath      = "/tmp/pca_kubeconfig"
	CrossAccountRoleKey = "PLUGIN_CROSS_ACCOUNT_ROLE"
	DefaultRegion       = "us-east-1"
	UserNameOverrideKey = "PLUGIN_USER_NAME_OVERRIDE"
)

func TestMain(m *testing.M) {
	o := opts

	roleName, xaRoleExists := os.LookupEnv(CrossAccountRoleKey)
	if !xaRoleExists {
		log.Printf("Skipping CrossAccount tests")
		o.Tags = "~@CrossAccount"
	} else {
		log.Printf("Using CrossAccount role: " + roleName)
	}

	log.Printf(fmt.Sprintf("Running tests with the following tags: %s", o.Tags))
	status := godog.TestSuite{
		Name:                 "AWSPrivateCAIssuer",
		Options:              &o,
		ScenarioInitializer:  InitializeScenario,
		TestSuiteInitializer: InitializeTestSuite,
	}.Run()

	os.Exit(status)
}

func InitializeTestSuite(suiteCtx *godog.TestSuiteContext) {
	// This is a BeforeAll hook that initializes the variables in the TestContext struct
	suiteCtx.BeforeSuite(func() {
		ctx := context.TODO()
		testContext = &TestContext{}
		testContext.caArns = make(map[string]string)

		//setup k8 client
		//kubeconfig files will be gathered from the home directory
		// tmp/pca_kubeconfig is auto populated if creating cluster from makefile
		clientConfig, err := clientcmd.BuildConfigFromFlags("", KubeConfigPath)
		if err != nil {
			panic(fmt.Sprintf("Ensure that that the kubeconfig for the cluster that is being tested is placed in %s", KubeConfigPath))
		}

		testContext.clientset, err = kubernetes.NewForConfig(clientConfig)
		if err != nil {
			panic(err.Error())
		}

		// We set the region to the AWS_REGION environment variable or us-east-1 by default
		testContext.region = DefaultRegion
		region, regionExists := os.LookupEnv("AWS_REGION")
		if regionExists {
			testContext.region = region
		}

		cfg, cfgErr := config.LoadDefaultConfig(ctx, config.WithRegion(testContext.region))
		if cfgErr != nil {
			panic(cfgErr.Error())
		}

		callerID := getCallerIdentity(ctx, cfg)

		parsedArn, parseErr := arn.Parse(*callerID.Arn)
		if parseErr != nil {
			panic("Failed to parse caller identity ARN: " + parseErr.Error())
		}

		testContext.partition = parsedArn.Partition

		testContext.roleToAssume = fmt.Sprintf("arn:%s:iam::%s:role/IssuerTestRole-test-us-east-1", testContext.partition, *callerID.Account)
		if roleToAssumeOverride, exists := os.LookupEnv("ROLE_TO_ASSUME_OVERRIDE"); exists {
			testContext.roleToAssume = roleToAssumeOverride
		}

		testContext.iclient, err = clientV1beta1.NewForConfig(clientConfig)

		if err != nil {
			panic(err.Error())
		}

		testContext.cmClient, err = cmclientv1.NewForConfig(clientConfig)

		if err != nil {
			panic(err.Error())
		}

		// Create CAs to be used in testing
		testContext.caArns["RSA"] = testContext.createCertificateAuthority(ctx, cfg, true)
		log.Printf("Created RSA CA with arn %s", testContext.caArns["RSA"])

		testContext.caArns["ECDSA"] = testContext.createCertificateAuthority(ctx, cfg, false)
		log.Printf("Created EC CA with arn %s", testContext.caArns["ECDSA"])

		xaRole, xaRoleExists := os.LookupEnv(CrossAccountRoleKey)
		if xaRoleExists {
			testContext.xaCfg = assumeRole(ctx, cfg, xaRole, testContext.region)

			testContext.caArns["XA"] = testContext.createCertificateAuthority(ctx, testContext.xaCfg, true)

			log.Printf("Created XA CA with arn %s", testContext.caArns["XA"])

			endEntityResourcePermission := "arn:" + testContext.partition + ":ram::aws:permission/AWSRAMDefaultPermissionCertificateAuthority"
			subordinateCaResourcePermission := "arn:" + testContext.partition + ":ram::aws:permission/AWSRAMSubordinateCACertificatePathLen0IssuanceCertificateAuthority"

			testContext.endEntityResourceShareArn = shareCA(ctx, cfg, testContext.xaCfg, testContext.caArns["XA"], endEntityResourcePermission)
			testContext.subordinateCaResourceShareArn = shareCA(ctx, cfg, testContext.xaCfg, testContext.caArns["XA"], subordinateCaResourcePermission)
		} else {
			log.Print("Cross account role not present in PLUGIN_CROSS_ACCOUNT_ROLE, skipping cross account testing")
		}

		// Create an Access Key to be used for validiting auth via secret for an Issuer
		userName, envUserExists := os.LookupEnv(UserNameOverrideKey)

		if !envUserExists {
			testContext.userName, testContext.policyArn = createUser(ctx, cfg)
			log.Printf("Created User %s with policy arn %s", testContext.userName, testContext.policyArn)
		} else {
			testContext.userName = userName
			log.Printf("Using User %s from PLUGIN_USER_NAME_OVERRIDE", testContext.userName)
		}

		testContext.accessKey, testContext.secretKey = createAccessKey(ctx, cfg, testContext.userName)

		// We wait for 10 seconds to ensure the AccessKey is availabile
		time.Sleep(10 * time.Second)
	})

	// This is an AfterAll hook that will clean up the variables in the TestContext struct
	suiteCtx.AfterSuite(func() {
		ctx := context.TODO()

		log.Printf(strings.Join(errDetails, "\n"))
		cfg, cfgErr := config.LoadDefaultConfig(ctx, config.WithRegion(testContext.region))
		if cfgErr != nil {
			panic(cfgErr.Error())
		}

		deleteCertificateAuthority(ctx, cfg, testContext.caArns["RSA"])
		log.Printf("Deleted the RSA CA")

		deleteCertificateAuthority(ctx, cfg, testContext.caArns["ECDSA"])
		log.Printf("Deleted the EC CA")

		deleteAccessKey(ctx, cfg, testContext.userName, testContext.accessKey)
		log.Printf("Deleted the Access Key")

		_, envUserExists := os.LookupEnv(UserNameOverrideKey)
		if !envUserExists {
			deleteUser(ctx, cfg, testContext.userName, testContext.policyArn)
			log.Printf("Deleted the User and associated policy")
		} else {
			log.Printf("User %s was not deleted", testContext.userName)
		}

		//Delete XA testing resources
		_, xaRoleExists := os.LookupEnv(CrossAccountRoleKey)
		if xaRoleExists {
			deleteResourceShare(ctx, testContext.xaCfg, testContext.endEntityResourceShareArn)
			deleteResourceShare(ctx, testContext.xaCfg, testContext.subordinateCaResourceShareArn)
			log.Printf("Deleted resource shares associated with XA CA")

			deleteCertificateAuthority(ctx, testContext.xaCfg, testContext.caArns["XA"])
			log.Printf("Deleted the XA CA")
		}
	})
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	// This initializes the IssuerContext struct for each test
	issuerContext := &IssuerContext{
		namespace: "default",
		secretRef: v1beta1.AWSCredentialsSecretReference{},
	}

	// This defines the mapping of steps --> functions
	ctx.Step(`^I create a namespace`, issuerContext.createNamespace)
	ctx.Step(`^I create a Secret with keys ([A-Za-z_]+) and ([A-Za-z_]+) for my AWS credentials$`, issuerContext.createSecret)
	ctx.Step(`^I create an AWSPCAClusterIssuer using a (RSA|ECDSA|XA) CA$`, issuerContext.createClusterIssuer)
	ctx.Step(`^I create an AWSPCAClusterIssuer with role assumption$`, issuerContext.createClusterIssuerWithRole)
	ctx.Step(`^I delete the AWSPCAClusterIssuer$`, issuerContext.deleteClusterIssuer)
	ctx.Step(`^I create an AWSPCAIssuer using a (RSA|ECDSA|XA) CA$`, issuerContext.createNamespaceIssuer)
	ctx.Step(`^I create an AWSPCAIssuer with role assumption$`, issuerContext.createNamespaceIssuerWithRole)
	ctx.Step(`^I issue a (SHORT_VALIDITY|RSA|ECDSA|CA) certificate$`, issuerContext.issueCertificate)
	ctx.Step(`^the certificate should be issued successfully$`, issuerContext.verifyCertificateIssued)
	ctx.Step(`^the certificate request has been created$`, issuerContext.verifyCertificateRequestIsCreated)
	ctx.Step(`^the certificate request has reason (Pending|Failed|Issued|Denied) and status (True|False|Unknown)$`, issuerContext.verifyCertificateRequestState)

	// This cleans up all of the resources after a test
	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		// Collect details if scenario failed
		if err != nil {
			errDetails = append(errDetails, GetErrorDetails(ctx, sc, issuerContext))
		}

		// Delete created Issuers
		switch issuerContext.issuerType {
		case "AWSPCAClusterIssuer":
			testContext.iclient.AWSPCAClusterIssuers().Delete(ctx, issuerContext.issuerName, metav1.DeleteOptions{})
		case "AWSPCAIssuer":
			testContext.iclient.AWSPCAIssuers(issuerContext.namespace).Delete(ctx, issuerContext.issuerName, metav1.DeleteOptions{})
		}

		// Delete created Secrets
		if issuerContext.secretRef != (v1beta1.AWSCredentialsSecretReference{}) {
			testContext.clientset.CoreV1().Secrets(issuerContext.namespace).Delete(ctx, issuerContext.secretRef.SecretReference.Name, metav1.DeleteOptions{})
		}

		// Delete created Certificates
		testContext.cmClient.Certificates(issuerContext.namespace).Delete(ctx, issuerContext.certName, metav1.DeleteOptions{})

		// Delete left over certificate secrets
		testContext.clientset.CoreV1().Secrets(issuerContext.namespace).Delete(ctx, issuerContext.certName+"-cert-secret", metav1.DeleteOptions{})

		// Delete created namespace
		if issuerContext.namespace != "default" {
			testContext.clientset.CoreV1().Namespaces().Delete(ctx, issuerContext.namespace, metav1.DeleteOptions{})
		}

		return ctx, nil
	})
}

func GetIssuerConditions(ctx context.Context, issuerContext *IssuerContext) ([]metav1.Condition, error) {
	switch issuerContext.issuerType {
	case "AWSPCAClusterIssuer":
		issuer, err := testContext.iclient.AWSPCAClusterIssuers().Get(ctx, issuerContext.issuerName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return issuer.Status.Conditions, nil
	case "AWSPCAIssuer":
		issuer, err := testContext.iclient.AWSPCAIssuers(issuerContext.namespace).Get(ctx, issuerContext.issuerName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return issuer.Status.Conditions, nil
	}

	return nil, fmt.Errorf("Unknown issuer type: %s", issuerContext.issuerType)
}

func GetErrorDetails(ctx context.Context, sc *godog.Scenario, issuerContext *IssuerContext) string {
	errMsg := "------------------------------\n"
	errMsg += fmt.Sprintf("❌ SCENARIO FAILED: %s\n", sc.Name)

	errMsg += "\nSCENARIO STEPS:\n"
	for _, step := range sc.Steps {
		errMsg += fmt.Sprintf("%s\n", step.Text)
	}

	errMsg += "\nTEST RESOURCE DETAILS:\n"
	issuerConditions, err := GetIssuerConditions(ctx, issuerContext)
	if err != nil {
		errMsg += fmt.Sprintf("Error getting Issuer: %v\n", err)
		return errMsg
	}

	errMsg += fmt.Sprintf("\nLogging Issuer conditions:\n")
	for _, condition := range issuerConditions {
		errMsg += fmt.Sprintf("Reason: %s, Status: %s, Message: %s\n", condition.Reason, condition.Status, condition.Message)
	}

	crName := fmt.Sprintf("%s-%d", issuerContext.certName, 1)
	cr, err := testContext.cmClient.CertificateRequests(issuerContext.namespace).Get(ctx, crName, metav1.GetOptions{})
	if err != nil {
		errMsg += fmt.Sprintf("Error getting CertificateRequest: %v\n", err)
		return errMsg
	}

	errMsg += fmt.Sprintf("\nLogging CertificateRequest conditions:\n")
	for _, condition := range cr.Status.Conditions {
		errMsg += fmt.Sprintf("Reason: %s, Status: %s, Message: %s\n", condition.Reason, condition.Status, condition.Message)
	}

	return errMsg
}
