package main

import (
	"context"
	"log"
	"os"
	"testing"

	cmclientv1 "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"
	"github.com/cucumber/godog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	clientV1beta1 "github.com/cert-manager/aws-privateca-issuer/pkg/clientset/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var testContext *TestContext

// These are variables shared by all of the tests
type TestContext struct {
	iclient   *clientV1beta1.Client
	cmClient  *cmclientv1.CertmanagerV1Client
	clientset *kubernetes.Clientset
	xaCfg     aws.Config
	caArns    map[string]string

	region, accessKey, secretKey, endEntityResourceShareArn, subordinateCaResourceShareArn, userName, policyArn string
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
}

func TestMain(m *testing.M) {
	o := opts

	_, xaRoleExists := os.LookupEnv("PLUGIN_CROSS_ACCOUNT_ROLE")
	if !xaRoleExists {
		log.Printf("Skipping CrossAccount tests")
		o.Tags = "@KeyUsage"
	}
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
		kubeconfig := "/tmp/pca_kubeconfig"

		clientConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic("Ensure that that the kubeconfig for the cluster that is being tested is placed in /tmp/pca_kubeconfig")
		}

		testContext.clientset, err = kubernetes.NewForConfig(clientConfig)
		if err != nil {
			panic(err.Error())
		}

		// We set the region to the AWS_REGION environment variable or us-east-1 by default
		testContext.region = "us-east-1"
		region, regionExists := os.LookupEnv("AWS_REGION")
		if regionExists {
			testContext.region = region
		}

		cfg, cfgErr := config.LoadDefaultConfig(ctx, config.WithRegion(testContext.region))
		if cfgErr != nil {
			panic(cfgErr.Error())
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
		testContext.caArns["RSA"] = createCertificateAuthority(ctx, cfg, true)
		log.Printf("Created RSA CA with arn %s", testContext.caArns["RSA"])

		testContext.caArns["ECDSA"] = createCertificateAuthority(ctx, cfg, false)
		log.Printf("Created EC CA with arn %s", testContext.caArns["ECDSA"])

		xaRole, xaRoleExists := os.LookupEnv("PLUGIN_CROSS_ACCOUNT_ROLE")
		if xaRoleExists {
			testContext.xaCfg = assumeRole(ctx, cfg, xaRole, testContext.region)

			testContext.caArns["XA"] = createCertificateAuthority(ctx, testContext.xaCfg, true)

			log.Printf("Created XA CA with arn %s", testContext.caArns["XA"])

			endEntityResourcePermission := "arn:aws:ram::aws:permission/AWSRAMDefaultPermissionCertificateAuthority"
			subordinateCaResourcePermission := "arn:aws:ram::aws:permission/AWSRAMSubordinateCACertificatePathLen0IssuanceCertificateAuthority"

			testContext.endEntityResourceShareArn = shareCA(ctx, cfg, testContext.xaCfg, testContext.caArns["XA"], endEntityResourcePermission)
			testContext.subordinateCaResourceShareArn = shareCA(ctx, cfg, testContext.xaCfg, testContext.caArns["XA"], subordinateCaResourcePermission)
		} else {
			log.Print("Cross account role not present in PLUGIN_CROSS_ACCOUNT_ROLE, skipping cross account testing")
		}

		// Create an Access Key to be used for validiting auth via secret for an Issuer
		userName, envUserExists := os.LookupEnv("PLUGIN_USER_NAME_OVERRIDE")

		if !envUserExists {
			testContext.userName, testContext.policyArn = createUser(ctx, cfg)
			log.Printf("Created User %s with policy arn %s", testContext.userName, testContext.policyArn)
		} else {
			testContext.userName = userName
			log.Printf("Using User %s from PLUGIN_USER_NAME_OVERRIDE", testContext.userName)
		}

		testContext.accessKey, testContext.secretKey = createAccessKey(ctx, cfg, testContext.userName)
	})

	// This is an AfterAll hook that will clean up the variables in the TestContext struct
	suiteCtx.AfterSuite(func() {
		ctx := context.TODO()

		cfg, cfgErr := config.LoadDefaultConfig(ctx, config.WithRegion(testContext.region))

		if cfgErr != nil {
			panic(cfgErr.Error())
		}

		deleteCertificateAuthority(ctx, cfg, testContext.caArns["RSA"])
		log.Printf("Deleted the RSA CA")

		deleteCertificateAuthority(ctx, cfg, testContext.caArns["ECDSA"])
		log.Printf("Deleted the EC CA")

		//Delete IAM User and policy
		deleteAccessKey(ctx, cfg, testContext.userName, testContext.accessKey)
		log.Printf("Deleted the Access Key")

		_, envUserExists := os.LookupEnv("PLUGIN_USER_NAME_OVERRIDE")
		if !envUserExists {
			deleteUser(ctx, cfg, testContext.userName, testContext.policyArn)
			log.Printf("Deleted the User and associated policy")
		} else {
			log.Printf("User %s was not deleted", testContext.userName)
		}

		//Delete XA testing resources
		_, xaRoleExists := os.LookupEnv("PLUGIN_CROSS_ACCOUNT_ROLE")
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
	ctx.Step(`^I delete the AWSPCAClusterIssuer$`, issuerContext.deleteClusterIssuer)
	ctx.Step(`^I create an AWSPCAIssuer using a (RSA|ECDSA|XA) CA$`, issuerContext.createNamespaceIssuer)
	ctx.Step(`^I issue a (SHORT_VALIDITY|RSA|ECDSA|CA) certificate$`, issuerContext.issueCertificateWithoutUsage)
	ctx.Step(`^I issue a (SHORT_VALIDITY|RSA|ECDSA|CA) certificate with usage (.+)$`, issuerContext.issueCertificateWithUsage)

	ctx.Step(`^the certificate should be issued successfully$`, issuerContext.verifyCertificateIssued)
	ctx.Step(`^the certificate request has reason (Pending|Failed|Issued|Denied) and status (True|False|Unknown)$`, issuerContext.verifyCertificateRequestState)

	ctx.Step(`^the certificate should be issued with usage (.+)$`, issuerContext.verifyCertificateContent)

	// This cleans up all of the resources after a test
	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
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
