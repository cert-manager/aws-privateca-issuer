package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	cmclientv1 "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"
	"github.com/stretchr/testify/assert"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/cert-manager/aws-privateca-issuer/pkg/api/v1beta1"
	clientV1beta1 "github.com/cert-manager/aws-privateca-issuer/pkg/clientset/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	iclient          *clientV1beta1.Client
	cmClient         *cmclientv1.CertmanagerV1Client
	clientset        *kubernetes.Clientset
	xaCfg            aws.Config
	issuerSpecs      []issuerTemplate
	certificateSpecs []certTemplate

	rsaCaArn, ecCaArn, xaCAArn, accessKey, secretKey, policyArn, endEntityResourceShareArn, subordinateCAResourceShareArn string

	region = "us-east-1"
	ctx    = context.TODO()
)

type issuerTemplate struct {
	spec       v1beta1.AWSPCAIssuerSpec
	issuerName string
}

type certTemplate struct {
	spec     cmv1.CertificateSpec
	certName string
}

func TestMain(m *testing.M) {
	//setup k8 client
	//kubeconfig files will be gathered from the home directory
	// tmp/pca_kubeconfig is auto populated if creating cluster from makefile
	kubeconfig := "/tmp/pca_kubeconfig"

	clientConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic("Ensure that that the kubeconfig for the cluster that is being tested is placed in /tmp/pca_kubeconfig")
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

		endEntityResourcePermission := "arn:aws:ram::aws:permission/AWSRAMDefaultPermissionCertificateAuthority"
		subordinateCAResourcePermission := "arn:aws:ram::aws:permission/AWSRAMSubordinateCACertificatePathLen0IssuanceCertificateAuthority"

		endEntityResourceShareArn = shareCA(ctx, cfg, xaCfg, xaCAArn, endEntityResourcePermission)
		subordinateCAResourceShareArn = shareCA(ctx, cfg, xaCfg, xaCAArn, subordinateCAResourcePermission)

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

	// We wait for 10 seconds to ensure the AccessKey is availabile
	time.Sleep(10 * time.Second)

	/*
	* Create a shared suite of Issuers and Certificates Specs to be used in
	* validing Cluster and Namepsace issuers
	 */
	issuerSpecs = []issuerTemplate{
		//Basic RSA Issuer
		{
			issuerName: "rsa-issuer",
			spec: v1beta1.AWSPCAIssuerSpec{
				Arn:    rsaCaArn,
				Region: region,
			},
		},
		//Basic EC Issuer
		{
			issuerName: "ec-issuer",
			spec: v1beta1.AWSPCAIssuerSpec{
				Arn:    ecCaArn,
				Region: region,
			},
		},
	}

	if xaRoleExists {
		//XA CA Issuer
		xaCASpec := issuerTemplate{
			issuerName: "crossaccount-issuer",
			spec: v1beta1.AWSPCAIssuerSpec{
				Arn:    xaCAArn,
				Region: region,
			},
		}
		issuerSpecs = append(issuerSpecs, xaCASpec)
	}

	certificateSpecs = []certTemplate{
		//Basic EC Certificate
		{
			certName: "ec-cert",
			spec: cmv1.CertificateSpec{
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
				Usages: []cmv1.KeyUsage{cmv1.UsageClientAuth, cmv1.UsageServerAuth},
			},
		},
		//Basic RSA Certificate
		{
			certName: "rsa-cert",
			spec: cmv1.CertificateSpec{
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
				Usages: []cmv1.KeyUsage{cmv1.UsageClientAuth, cmv1.UsageServerAuth},
			},
		},
		//Basic RSA Certificate with Validity < 24h
		{
			certName: "rsa-cert-low-validity",
			spec: cmv1.CertificateSpec{
				Subject: &cmv1.X509Subject{
					Organizations: []string{"aws"},
				},
				DNSNames: []string{"rsa-cert.aws.com"},
				PrivateKey: &cmv1.CertificatePrivateKey{
					Algorithm: cmv1.RSAKeyAlgorithm,
					Size:      2048,
				},
				Duration: &metav1.Duration{
					Duration: 20 * time.Hour,
				},
				RenewBefore: &metav1.Duration{
					Duration: 5 * time.Hour,
				},
				Usages: []cmv1.KeyUsage{cmv1.UsageClientAuth, cmv1.UsageServerAuth},
			},
		},
		//Default isCA certificate
		{
			certName: "sub-ca-certificate",
			spec: cmv1.CertificateSpec{
				Subject: &cmv1.X509Subject{
					Organizations: []string{"aws"},
				},
				DNSNames: []string{"rsa-cert.aws.com"},
				PrivateKey: &cmv1.CertificatePrivateKey{
					Algorithm: cmv1.RSAKeyAlgorithm,
					Size:      2048,
				},
				Duration: &metav1.Duration{
					Duration: 20 * time.Hour,
				},
				RenewBefore: &metav1.Duration{
					Duration: 5 * time.Hour,
				},
				IsCA: true,
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
		deleteResourceShare(ctx, xaCfg, endEntityResourceShareArn)
		deleteResourceShare(ctx, xaCfg, subordinateCAResourceShareArn)
		log.Printf("Deleted resource shares associated with XA CA")

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

	selectorSecretName := "pca-cluster-issuer-secret-selector-" + currentTime

	selectorKeyData := make(map[string][]byte)
	selectorKeyData["accessKeyId"] = []byte(accessKey)
	selectorKeyData["secretAccessKey"] = []byte(secretKey)

	selectorSecret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: selectorSecretName},
		Data:       selectorKeyData,
	}

	_, err = clientset.CoreV1().Secrets("default").Create(ctx, &selectorSecret, metav1.CreateOptions{})

	if err != nil {
		assert.FailNow(t, "Failed to create cluster issuer selector secret"+err.Error())
	}

	clusterIssuers := []v1beta1.AWSPCAClusterIssuer{}

	//compose issuers
	for _, template := range issuerSpecs {

		//Create issuer without secret (IRSA/EC2 instance profiles)
		issuerName := currentTime + "--" + template.issuerName

		issuer := v1beta1.AWSPCAClusterIssuer{
			ObjectMeta: metav1.ObjectMeta{Name: issuerName},
			Spec:       template.spec,
		}

		clusterIssuers = append(clusterIssuers, issuer)

		//Create issuer with secret
		template.spec.SecretRef = v1beta1.AWSCredentialsSecretReference{
			SecretReference: v1.SecretReference{
				Name:      secretName,
				Namespace: "default",
			},
		}

		issuerName = issuerName + "-secret"

		issuer = v1beta1.AWSPCAClusterIssuer{
			ObjectMeta: metav1.ObjectMeta{Name: issuerName},
			Spec:       template.spec,
		}

		clusterIssuers = append(clusterIssuers, issuer)

		//Create issuer with selector keys for secret
		template.spec.SecretRef = v1beta1.AWSCredentialsSecretReference{
			SecretReference: v1.SecretReference{
				Name:      selectorSecretName,
				Namespace: "default",
			},
			AccessKeyIDSelector: v1.SecretKeySelector{
				Key: "accessKeyId",
			},
			SecretAccessKeySelector: v1.SecretKeySelector{
				Key: "secretAccessKey",
			},
		}

		issuerName = issuerName + "-selectors"

		issuer = v1beta1.AWSPCAClusterIssuer{
			ObjectMeta: metav1.ObjectMeta{Name: issuerName},
			Spec:       template.spec,
		}

		clusterIssuers = append(clusterIssuers, issuer)

	}

	for _, clusterIssuer := range clusterIssuers {

		issuerName := clusterIssuer.ObjectMeta.Name

		log.Print("----------")

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

		for _, template := range certificateSpecs {
			certificateName := issuerName + "-" + template.certName

			secretName := certificateName + "-cert-secret"

			template.spec.SecretName = secretName

			template.spec.IssuerRef = cmmeta.ObjectReference{
				Kind:  "AWSPCAClusterIssuer",
				Group: "awspca.cert-manager.io",
				Name:  issuerName,
			}

			certificate := cmv1.Certificate{
				ObjectMeta: metav1.ObjectMeta{Name: certificateName},
				Spec:       template.spec,
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

	selectorSecretName := "pca-ns-issuer-secret-selector-" + currentTime

	selectorKeyData := make(map[string][]byte)
	selectorKeyData["accessKeyId"] = []byte(accessKey)
	selectorKeyData["secretAccessKey"] = []byte(secretKey)

	selectorSecret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: selectorSecretName},
		Data:       selectorKeyData,
	}

	_, err = clientset.CoreV1().Secrets(namespaceName).Create(ctx, &selectorSecret, metav1.CreateOptions{})

	if err != nil {
		assert.FailNow(t, "Failed to create namespace issuer selector secret"+err.Error())
	}

	namespaceIssuers := []v1beta1.AWSPCAIssuer{}

	//compose issuers
	for _, template := range issuerSpecs {
		issuerName := currentTime + "--" + template.issuerName

		issuer := v1beta1.AWSPCAIssuer{
			ObjectMeta: metav1.ObjectMeta{Name: issuerName},
			Spec:       template.spec,
		}

		namespaceIssuers = append(namespaceIssuers, issuer)

		//Create issuer with secret
		template.spec.SecretRef = v1beta1.AWSCredentialsSecretReference{
			SecretReference: v1.SecretReference{
				Name:      secretName,
				Namespace: namespaceName,
			},
		}

		issuerName = issuerName + "-secret"

		issuer = v1beta1.AWSPCAIssuer{
			ObjectMeta: metav1.ObjectMeta{Name: issuerName},
			Spec:       template.spec,
		}

		namespaceIssuers = append(namespaceIssuers, issuer)

		template.spec.SecretRef = v1beta1.AWSCredentialsSecretReference{
			SecretReference: v1.SecretReference{
				Name:      selectorSecretName,
				Namespace: namespaceName,
			},
			AccessKeyIDSelector: v1.SecretKeySelector{
				Key: "accessKeyId",
			},
			SecretAccessKeySelector: v1.SecretKeySelector{
				Key: "secretAccessKey",
			},
		}

		issuerName = issuerName + "-selectors"

		issuer = v1beta1.AWSPCAIssuer{
			ObjectMeta: metav1.ObjectMeta{Name: issuerName},
			Spec:       template.spec,
		}

		namespaceIssuers = append(namespaceIssuers, issuer)

	}

	for _, namespaceIssuer := range namespaceIssuers {

		issuerName := namespaceIssuer.ObjectMeta.Name

		log.Print("----------")

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

		for _, template := range certificateSpecs {
			certificateName := issuerName + "-" + template.certName

			secretName := certificateName + "-cert-secret"

			template.spec.SecretName = secretName

			template.spec.IssuerRef = cmmeta.ObjectReference{
				Kind:  "AWSPCAIssuer",
				Group: "awspca.cert-manager.io",
				Name:  issuerName,
			}

			certificate := cmv1.Certificate{
				ObjectMeta: metav1.ObjectMeta{Name: certificateName},
				Spec:       template.spec,
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

// This tests the case where the Issuer is not in a Ready state when the CertificateRequest is made.
// In this instance, the CertificateRequest should be put into Pending and then properly Issued once the
// Issuer is Ready.
func TestCertificateRecoveryWhenIssuerNotReady(t *testing.T) {
	currentTime := strconv.FormatInt(time.Now().Unix(), 10)

	template := issuerSpecs[0]
	issuerName := currentTime + "--" + template.issuerName

	issuer := v1beta1.AWSPCAClusterIssuer{
		ObjectMeta: metav1.ObjectMeta{Name: issuerName},
		Spec:       template.spec,
	}

	log.Print("----------")
	log.Printf("Testing certificate recovery with issuer: %s", issuerName)

	certificateName := issuerName + "-rsa-cert"
	certificate := cmv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: certificateName},
		Spec: cmv1.CertificateSpec{
			IssuerRef: cmmeta.ObjectReference{
				Kind:  "AWSPCAClusterIssuer",
				Group: "awspca.cert-manager.io",
				Name:  issuerName,
			},
			SecretName: certificateName + "-cert-secret",
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
			Usages: []cmv1.KeyUsage{cmv1.UsageClientAuth, cmv1.UsageServerAuth},
		},
	}

	log.Printf("Testing Certificate %s", certificateName)

	cert, err := cmClient.Certificates("default").Create(ctx, &certificate, metav1.CreateOptions{})

	if err != nil {
		assert.FailNow(t, "Could not create certificate: "+err.Error())
	}

	revision := cert.Status.Revision
	rev := 1
	if revision != nil {
		rev = *revision
	}
	crName := fmt.Sprintf("%s-%d", certificateName, rev)

	err = waitForCertificateRequestToBeCreated(ctx, cmClient, crName, "default")

	if err != nil {
		assert.FailNow(t, "CertificateRequest not found: "+err.Error())
	}

	err = waitForCertificateRequestPending(ctx, cmClient, crName, "default")

	if err != nil {
		assert.FailNow(t, "CertificateRequest did not reach a pending state: "+err.Error())
	}

	_, err = iclient.AWSPCAClusterIssuers().Create(ctx, &issuer, metav1.CreateOptions{})

	if err != nil {
		assert.FailNow(t, "Could not create Cluster Issuer: "+err.Error())
	}

	err = waitForClusterIssuerReady(ctx, iclient, issuerName)

	if err != nil {
		assert.FailNow(t, "Cluster issuer did not reach a ready state: "+err.Error())
	}

	err = waitForCertificateReady(ctx, cmClient, certificateName, "default")

	if err != nil {
		assert.FailNow(t, "Certificate did not reach a ready state: "+err.Error())
	}

	err = cmClient.Certificates("default").Delete(ctx, certificateName, metav1.DeleteOptions{})

	if err != nil {
		assert.FailNow(t, "Certificate was not succesfully deleted: "+err.Error())
	}

	err = iclient.AWSPCAClusterIssuers().Delete(ctx, issuerName, metav1.DeleteOptions{})

	if err != nil {
		assert.FailNow(t, "Issuer was not successfully deleted: "+err.Error())
	}
}
