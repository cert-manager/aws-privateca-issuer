package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	cmv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmclientv1 "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"
	"github.com/stretchr/testify/assert"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	clientV1beta1 "github.com/cert-manager/aws-privateca-issuer/pkg/clientset/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	iclient          *clientV1beta1.Client
	cmClient         *cmclientv1.CertmanagerV1Client
	clientset        *kubernetes.Clientset
	xaCfg            aws.Config
	issuerSpecs      []v1beta1.AWSPCAIssuerSpec
	certificateSpecs []cmv1.CertificateSpec

	rsaCaArn, ecCaArn, xaCAArn, accessKey, secretKey, policyArn, resourceShareArn string

	region = "us-east-1"
	ctx    = context.TODO()
)

func TestMain(m *testing.M) {
	/*
	* Setup clients to interact with Kubernetes cluster
	 */

	//setup k8 client
	//kubeconfig files will be gathered from the home directory
	home := homedir.HomeDir()
	kubeconfig := filepath.Join(home, ".kube", "config")

	clientConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	clientset, err = kubernetes.NewForConfig(clientConfig)
	if err != nil {
		panic(err.Error())
	}

	var cfg, cfgErr = config.LoadDefaultConfig(ctx, config.WithRegion(region))

	if cfgErr != nil {
		panic(cfgErr.Error())
	}

	iclient, err = clientV1beta1.NewForConfig(clientConfig)

	if err != nil {
		panic(err.Error())
	}

	cmClient, err = cmclientv1.NewForConfig(clientConfig)

	if err != nil {
		panic(err.Error())
	}

	/*
	* Create CAs to be used in testing
	 */

	rsaCaArn = createCertificateAuthority(ctx, cfg, true)
	log.Printf("Created RSA CA with arn %s", rsaCaArn)

	ecCaArn = createCertificateAuthority(ctx, cfg, false)
	log.Printf("Created EC CA with arn %s", ecCaArn)

	xaRole, xaRoleExists := os.LookupEnv("PLUGIN_CROSS_ACCOUNT_ROLE")
	if xaRoleExists {
		xaCfg = assumeRole(ctx, cfg, xaRole, region)

		xaCAArn = createCertificateAuthority(ctx, xaCfg, true)

		log.Printf("Created XA CA with arn %s", xaCAArn)

		resourceShareArn = shareCA(ctx, cfg, xaCfg, xaCAArn)
	} else {
		log.Print("Cross account role not present in PLUGIN_CROSS_ACCOUNT_ROLE, skipping cross account testing")
	}

	/*
	*Create an Access Key to be used for validiting auth via secret for an Issuer
	 */

	userName, envUserExists := os.LookupEnv("PLUGIN_USER_NAME_OVERRIDE")

	if !envUserExists {
		userName, policyArn = createUser(ctx, cfg)
		log.Printf("Created User %s with policy arn %s", userName, policyArn)
	} else {
		log.Printf("Using User %s from PLUGIN_USER_NAME_OVERRIDE", userName)
	}

	accessKey, secretKey = createAccessKey(ctx, cfg, userName)

	/*
	* Create a shared suite of Issuers and Certificates Specs to be used in
	* validing Cluster and Namepsace issuers
	 */

	issuerSpecs = []v1beta1.AWSPCAIssuerSpec{
		//Basic RSA Issuer
		{
			Arn:    rsaCaArn,
			Region: region,
		},
		//Basic EC Issuer
		{
			Arn:    ecCaArn,
			Region: region,
		},
	}

	if xaRoleExists {
		//XA CA Issuer
		xaCASpec := v1beta1.AWSPCAIssuerSpec{
			Arn:    xaCAArn,
			Region: region,
		}
		issuerSpecs = append(issuerSpecs, xaCASpec)
	}

	certificateSpecs = []cmv1.CertificateSpec{
		//Basic EC Certificate
		{
			Subject: &cmv1.X509Subject{
				Organizations: []string{"aws"},
			},
			DNSNames: []string{"ec-cert.aws.com"},
			PrivateKey: &cmv1.CertificatePrivateKey{
				Algorithm: cmv1.ECDSAKeyAlgorithm,
				Size:      256,
			},
			Duration: &metav1.Duration{
				Duration: 721 * time.Hour,
			},
		},
		//Basic RSA Certificate
		{
			Subject: &cmv1.X509Subject{
				Organizations: []string{"aws"},
			},
			DNSNames: []string{"rsa-cert.aws.com"},
			PrivateKey: &cmv1.CertificatePrivateKey{
				Algorithm: cmv1.RSAKeyAlgorithm,
				Size:      2048,
			},
			Duration: &metav1.Duration{
				Duration: 721 * time.Hour,
			},
		},
	}

	/*
	* Run the test!
	 */
	exitVal := m.Run()

	/*
	* Clean up testing resources
	 */
	deleteCertificateAuthority(ctx, cfg, rsaCaArn)
	log.Printf("Deleted the RSA CA")

	deleteCertificateAuthority(ctx, cfg, ecCaArn)
	log.Printf("Deleted the EC CA")

	//Delete IAM User and policy
	deleteAccessKey(ctx, cfg, userName, accessKey)
	log.Printf("Deleted the Access Key")

	if !envUserExists {
		deleteUser(ctx, cfg, userName, policyArn)
		log.Printf("Deleted the User and associated policy")
	} else {
		log.Printf("User %s was not deleted", userName)
	}

	//Delete XA testing resources
	if xaRoleExists {
		deleteResourceShare(ctx, xaCfg, resourceShareArn)
		log.Printf("Deleted resource share associated with XA CA")

		deleteCertificateAuthority(ctx, xaCfg, xaCAArn)
		log.Printf("Deleted the XA CA")
	}

	//Exit
	os.Exit(exitVal)
}

func TestClusterIssuers(t *testing.T) {

	currentTime := strconv.FormatInt(time.Now().Unix(), 10)

	secretName := "pca-cluster-issuer-secret-" + currentTime

	data := make(map[string][]byte)
	data["AWS_ACCESS_KEY_ID"] = []byte(accessKey)
	data["AWS_SECRET_ACCESS_KEY"] = []byte(secretKey)

	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName},
		Data:       data,
	}

	_, err := clientset.CoreV1().Secrets("default").Create(ctx, &secret, metav1.CreateOptions{})

	if err != nil {
		assert.FailNow(t, "Failed to create cluster issuer secret"+err.Error())
	}

	clusterIssuers := []v1beta1.AWSPCAClusterIssuer{}

	//compose issuers
	for index, specs := range issuerSpecs {
		specs.SecretRef = v1beta1.AWSCredentialsSecretReference{
			SecretReference: v1.SecretReference{
				Name:      secretName,
				Namespace: "default",
			},
		}

		issuerName := "cluster-issuer-" + strconv.Itoa(index) + "-" + currentTime

		issuer := v1beta1.AWSPCAClusterIssuer{
			ObjectMeta: metav1.ObjectMeta{Name: issuerName},
			Spec:       specs,
		}

		clusterIssuers = append(clusterIssuers, issuer)
	}

	for _, clusterIssuer := range clusterIssuers {

		issuerName := clusterIssuer.ObjectMeta.Name

		log.Printf("Testing issuer: %s", issuerName)

		_, err := iclient.AWSPCAClusterIssuers().Create(ctx, &clusterIssuer, metav1.CreateOptions{})

		if err != nil {
			assert.FailNow(t, "Could not create Cluster Issuer: "+err.Error())
		}

		err = waitForClusterIssuerReady(ctx, iclient, issuerName)

		if err != nil {
			assert.FailNow(t, "Cluster issuer did not reach a ready state: "+err.Error())
		}

		//compose certificates
		certificates := []cmv1.Certificate{}

		for index, specs := range certificateSpecs {
			certificateName := "cluster-cert-" + strconv.Itoa(index) + "-" + currentTime

			secretName := certificateName + "-" + "secret"

			specs.SecretName = secretName

			specs.IssuerRef = cmmeta.ObjectReference{
				Kind:  "AWSPCAClusterIssuer",
				Group: "awspca.cert-manager.io",
				Name:  issuerName,
			}

			certificate := cmv1.Certificate{
				ObjectMeta: metav1.ObjectMeta{Name: certificateName},
				Spec:       specs,
			}

			certificates = append(certificates, certificate)
		}

		for _, certificate := range certificates {

			certName := certificate.ObjectMeta.Name

			log.Printf("Testing Certificate %s", certName)

			_, err = cmClient.Certificates("default").Create(ctx, &certificate, metav1.CreateOptions{})

			if err != nil {
				assert.FailNow(t, "Could not create certificate: "+err.Error())
			}

			err = waitForCertificateReady(ctx, cmClient, certName, "default")

			if err != nil {
				assert.FailNow(t, "Certificate did not reach a ready state: "+err.Error())
			}

			err = cmClient.Certificates("default").Delete(ctx, certName, metav1.DeleteOptions{})

			if err != nil {
				assert.FailNow(t, "Certificate was not succesfully deleted: "+err.Error())
			}
		}

		err = iclient.AWSPCAClusterIssuers().Delete(ctx, issuerName, metav1.DeleteOptions{})

		if err != nil {
			assert.FailNow(t, "Issuer was not successfully deleted: "+err.Error())
		}
	}
}

func TestNamespaceIssuers(t *testing.T) {

	currentTime := strconv.FormatInt(time.Now().Unix(), 10)

	// Create namespace for issuer to live in
	namespaceName := "pca-issuer-ns-" + currentTime

	namespace := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespaceName},
	}

	_, err := clientset.CoreV1().Namespaces().Create(ctx, &namespace, metav1.CreateOptions{})

	if err != nil {
		assert.FailNow(t, "Failed to create namespace"+err.Error())
	}

	secretName := "pca-ns-issuer-secret-" + currentTime

	data := make(map[string][]byte)
	data["AWS_ACCESS_KEY_ID"] = []byte(accessKey)
	data["AWS_SECRET_ACCESS_KEY"] = []byte(secretKey)

	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName},
		Data:       data,
	}

	_, err = clientset.CoreV1().Secrets(namespaceName).Create(ctx, &secret, metav1.CreateOptions{})

	if err != nil {
		assert.FailNow(t, "Failed to create namespace issuer secret"+err.Error())
	}

	namespaceIssuers := []v1beta1.AWSPCAIssuer{}

	//compose issuers
	for index, specs := range issuerSpecs {
		specs.SecretRef = v1beta1.AWSCredentialsSecretReference{
			SecretReference: v1.SecretReference{
				Name:      secretName,
				Namespace: namespaceName,
			},
		}

		issuerName := "ns-issuer-" + strconv.Itoa(index) + "-" + currentTime

		issuer := v1beta1.AWSPCAIssuer{
			ObjectMeta: metav1.ObjectMeta{Name: issuerName},
			Spec:       specs,
		}

		namespaceIssuers = append(namespaceIssuers, issuer)
	}

	for _, namespaceIssuer := range namespaceIssuers {

		issuerName := namespaceIssuer.ObjectMeta.Name

		log.Printf("Testing issuer: %s", issuerName)

		_, err := iclient.AWSPCAIssuers(namespaceName).Create(ctx, &namespaceIssuer, metav1.CreateOptions{})

		if err != nil {
			assert.FailNow(t, "Could not create Namespace Issuer: "+err.Error())
		}

		err = waitForIssuerReady(ctx, iclient, issuerName, namespaceName)

		if err != nil {
			assert.FailNow(t, "Namespace issuer did not reach a ready state: "+err.Error())
		}

		//compose certificates
		certificates := []cmv1.Certificate{}

		for index, specs := range certificateSpecs {
			certificateName := "ns-cert-" + strconv.Itoa(index) + "-" + currentTime

			secretName := certificateName + "-" + "secret"

			specs.SecretName = secretName

			specs.IssuerRef = cmmeta.ObjectReference{
				Kind:  "AWSPCAIssuer",
				Group: "awspca.cert-manager.io",
				Name:  issuerName,
			}

			certificate := cmv1.Certificate{
				ObjectMeta: metav1.ObjectMeta{Name: certificateName},
				Spec:       specs,
			}

			certificates = append(certificates, certificate)
		}

		for _, certificate := range certificates {

			certName := certificate.ObjectMeta.Name

			log.Printf("Testing Certificate %s", certName)

			_, err = cmClient.Certificates(namespaceName).Create(ctx, &certificate, metav1.CreateOptions{})

			if err != nil {
				assert.FailNow(t, "Could not create certificate: "+err.Error())
			}

			err = waitForCertificateReady(ctx, cmClient, certName, namespaceName)

			if err != nil {
				assert.FailNow(t, "Certificate did not reach a ready state: "+err.Error())
			}

			err = cmClient.Certificates(namespaceName).Delete(ctx, certName, metav1.DeleteOptions{})

			if err != nil {
				assert.FailNow(t, "Certificate was not succesfully deleted: "+err.Error())
			}
		}

		err = iclient.AWSPCAIssuers(namespaceName).Delete(ctx, issuerName, metav1.DeleteOptions{})

		if err != nil {
			assert.FailNow(t, "Issuer was not successfully deleted: "+err.Error())
		}
	}
}
