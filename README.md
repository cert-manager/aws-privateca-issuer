<p align="center"><img src="assets/logo.png" alt="Logo" width="250px" /></p>
<p align="center">
<a href="https://github.com/cert-manager/aws-privateca-issuer/actions">
<img alt="Build Status" src="https://github.com/cert-manager/aws-privateca-issuer/workflows/CI/badge.svg" />
</a>
<a href="https://goreportcard.com/report/github.com/cert-manager/aws-privateca-issuer">
<img alt="Build Status" src="https://goreportcard.com/badge/github.com/cert-manager/aws-privateca-issuer" />
</a>
<img alt="Latest version" src="https://img.shields.io/github/v/release/cert-manager/aws-privateca-issuer?color=success&sort=semver" />
</p>

# AWS Private CA Issuer

AWS ACM Private CA is a module of the AWS Certificate Manager that can setup and manage private CAs.

cert-manager is a Kubernetes add-on to automate the management and issuance of TLS certificates from various issuing sources.
It will ensure certificates are valid and up to date periodically, and attempt to renew certificates at an appropriate time before expiry.

This project acts as an addon (see https://cert-manager.io/docs/configuration/external/) to cert-manager that signs off certificate requests using AWS PCA.

## Setup

Install cert-manager first (https://cert-manager.io/docs/installation/kubernetes/).

Then install AWS PCA Issuer using Helm:

```shell
helm repo add awspca https://cert-manager.github.io/aws-privateca-issuer
helm install awspca/aws-pca-issuer --generate-name
```

You can check the chart configuration in the default [values](https://github.com/cert-manager/aws-privateca-issuer/blob/master/charts/aws-pca-issuer/values.yaml) file.


## Configuration

As of now, the only configurable settings are access to AWS. So you can use `AWS_REGION`, `AWS_ACCESS_KEY_ID` or `AWS_SECRET_ACCESS_KEY`.

Access to AWS can also be configured using an EC2 instance role.

A minimal policy to use the issuer with an authority would look like follows:

```
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "awspcaissuer",
      "Action": [
        "acm-pca:DescribeCertificateAuthority",
        "acm-pca:GetCertificate",
        "acm-pca:IssueCertificate"
      ],
      "Effect": "Allow",
      "Resource": "arn:aws:acm-pca:<region>:<account_id>:certificate-authority/<resource_id>"
    }
  ]
}
```

## Usage

This operator provides two custom resources that you can use.

Examples can be found in the [examples](https://github.com/cert-manager/aws-privateca-issuer/tree/master/config/examples/) directory.

### AWSPCAIssuer

This is a regular namespaced issuer that can be used as a reference in your Certificate CRs.

### AWSPCAClusterIssuer

This CR is identical to the AWSPCAIssuer. The only difference being that it's not namespaced and can be referenced from anywhere.

### Disable Approval Check

The AWSPCA Issuer will wait for CertificateRequests to have an [approved condition
set](https://cert-manager.io/docs/concepts/certificaterequest/#approval) before
signing. If using an older version of cert-manager (pre v1.3), you can disable
this check by supplying the command line flag `-disable-approved-check` to the
Issuer Deployment.

## Running the tests

Running ```make test``` will run the written unit test

Running ```make runtests``` will take the current code artifacts and transform them into a Docker image that will run on a kind cluster and ensure that the current version of the code still enables EC/RSA certs to be issued by an AWS Private CA. It will also verify the unit tests pass.

### Requirements for running the integration testing

NOTE: Running these tests **will incur charges in your AWS Account**. Please refer to [AWS PCA pricing](https://aws.amazon.com/certificate-manager/pricing/).


For running the integration tests you will need a few things:
* 2 AWS Private CAs - One Private CA that is backed by an RSA key and another Private CA that is backed by an EC key (Currently the PCA external issuer only supports issuing RSA certs from an RSA key backed CA and issuing EC certs from an EC key backed CA).
* Access to an AWS Account (Via an IAM User) where you will have permission to create, update, and delete the Private CAs needed to run the integration tests
* [Git](https://git-scm.com/)
* [Golang v1.13+](https://golang.org/)
* [Docker v17.03+](https://docs.docker.com/install/)
* [Kind v0.9.0+](https://kind.sigs.k8s.io/docs/user/quick-start/)
* [Kubectl v1.11.3+](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* [Kubebuilder v2.3.1+](https://book.kubebuilder.io/quick-start.html#installation)
* [Kustomize v3.8.1+](https://kustomize.io/)
* [AWS CLI v2](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html)

### How the integration tests work / How to run them

As mentioned before, running the tests is as easy as ```make runtests```

The code for the integration tests live in test_utils/e2e_tests.sh. The test first begins by creating an RSA backed CA and an EC backed CA (If the Enviornment variables ```RSA_CM_CA_ARN``` and/or ```EC_CM_CA_ARN``` are set, the test will skip creating that kind of CA and just use the ARN supplied). 

The tests will take your AWS CLI creds via the enviornment variables ```AWS_ACCESS_KEY_ID``` and ```AWS_SECRET_ACCESS_KEY``` and use those to not only create/update/delete the CAs used for the test, but also use these as the secret to pass to the AWSPCAClusterIssuer/AWSPCAIssuer for allowing the Issuers to issue certificates.

The tests will then spin up a kind cluster and create various Issuer/ClusterIssuer resources along with various certificate resources. The test will verify that using the Cluster or Namespace Issuer, the PCA external issuer is able to issue both EC and RSA certificates and the Cert Manager certificate resources reach a ready state.

After the test, the resources created with the kind cluster are cleaned up, the kind cluster is deleted, and the CAs used during the test are deleted.

The Private CAs created during this test run are cleaned up on a **best-effort basis**. To ensure no runaway costs, verify via the AWS CLI or Console that the Private CAs created during the test run are in a deleted state. If you need to delete the Private CAs created during the test run yourself you may use the script test_utils/delete_ca.sh or refer to the [AWS Private CA documentation](https://docs.aws.amazon.com/acm-pca/latest/userguide/PCADeleteCA.html)

If at any point, ```make runtests``` encounters an error, the integration tests should be considered a failure.

## Troubleshooting

1. Check the secret with the AWS credentials: AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY values have to be base64 encoded.

2. If the generated CertificateRequest shows no events, it is very likely that you're using an older version of cert-manager which doesn't support approval check. Disable approval check at the issuer deployment.
