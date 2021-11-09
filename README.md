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

## Known Issues

1. STS GetCallerIdentity failing because of a region not specified bug

    There is currently a known issue with the plugin that is preventing certificate issuance due to STS GetCallerIdentity failing because of a region not specified bug, regardless of whether a region was specified or not (https://github.com/cert-manager/aws-privateca-issuer/issues/54). There is an existing pull request to fix this (https://github.com/cert-manager/aws-privateca-issuer/pull/53), but we are holding off on accepting any pull requests until our testing is redesigned. To fix this issue until then, please checkout the cleanup branch by running

        git fetch -a
        git checkout cleanup

    Also, please be sure you are using the plugin with an IAM user, as that is the most reliable workflow https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html#Using_CreateAccessKey
    This user must have minimum permissions listed in the [Configuration](#configuration) section.

        export AWS_SECRET_ACCESS_KEY=<Secret Access Key you generated>
        export AWS_ACCESS_KEY_ID=<Access Key you generated>

2. Validity durations under 24 hours causing failures

    There is currently a known issue that is preventing issuance of certificates with validity durations under 24h due to a typecasting issue from float64 to int64 (https://github.com/cert-manager/aws-privateca-issuer/issues/69). There is an existing pull request to fix this (https://github.com/cert-manager/aws-privateca-issuer/pull/70), but we are holding off on accepting any pull requests until we merge in https://github.com/cert-manager/aws-privateca-issuer/pull/65. To fix this issue until then, please use validity durations that are greater than 24h.

## Setup

Install cert-manager first (https://cert-manager.io/docs/installation/kubernetes/).

Then install AWS PCA Issuer using Helm:

```shell
helm repo add awspca https://cert-manager.github.io/aws-privateca-issuer
helm install awspca/aws-privateca-issuer --generate-name
```

You can check the chart configuration in the default [values](charts/aws-pca-issuer/values.yaml) file.


## Configuration

As of now, the only configurable settings are access to AWS. So you can use `AWS_REGION`, `AWS_ACCESS_KEY_ID` or `AWS_SECRET_ACCESS_KEY`.

Alternatively, you can supply arbitrary secrets for the access and secret keys with the `accessKeyIDSelector` and `secretAccessKeySelector` fields in the clusterissuer and/or issuer manifests.

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

Examples can be found in the [examples](config/examples/) directory.

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

### Authentication

Please note that if you are using [KIAM](https://github.com/uswitch/kiam) for authentication, this plugin has been tested on KIAM v4.0. [IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) is also tested and supported.

There is a custom AWS authentication method we have coded into our plugin that allows a user to define a [Kubernetes secret](https://kubernetes.io/docs/concepts/configuration/secret/) with AWS Creds passed in, example [here](config/samples/secret.yaml). The user applies that file with their creds and then references the secret in their Issuer CRD when running the plugin, example [here](config/samples/awspcaclusterissuer_ec/_v1beta1_awspcaclusterissuer_ec.yaml#L8-L10).

## Understanding/Running the tests

### Running the Unit Tests
Running ```make test``` will run the written unit test

### Running the End-To-End Tests

NOTE: Running these tests **will incur charges in your AWS Account**. 

Running ```make e2etest``` will take the current code artifacts and transform them into a Docker image that will run on a kind cluster and ensure that the current version of the code still works with the [Supported Workflows](#Supported-Workflows)

The easiest way to get the test to run would be to use the follow make targets:
```make cluster && make install-eks-webhook && make e2etest```

### Getting ```make cluster``` to run
```make cluster``` will create a kind cluster on your machine that has Cert-Manager installed as well as the aws-pca-issuer plugin (using the HEAD of the current branch)

Before running ```make cluster``` we will need to do the following:

\- Have the following tools on your machine:
* [Git](https://git-scm.com/)
* [Golang v1.13+](https://golang.org/)
* [Docker v17.03+](https://docs.docker.com/install/)
* [Kind v0.9.0+](https://kind.sigs.k8s.io/docs/user/quick-start/) -> This will be installed via running the test
* [Kubectl v1.11.3+](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* [AWS CLI v2](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html)
* [Helm](https://helm.sh/docs/intro/install/)

\- (Optional) You will need a AWS IAM User to test authentication via K8 secrets. You can provide an already existing user into the test via ```export PLUGIN_USER_NAME_OVERRIDE=<IAM User Name>```.  This IAM User should have a policy attached to it that follows with the policy listed in [Configuration](#configuration). This user will be used to test authentication in the plugin via K8 secrets.

\- An S3 Bucket with [BPA disabled](https://docs.aws.amazon.com/AmazonS3/latest/userguide/access-control-block-public-access.html) in us-east-1. After creating the bucket run ```export OIDC_S3_BUCKET_NAME=<Name of bucket you just created```

\- You will need AWS credentials loaded into your terminal that, via the CLI, minimally allow the following actions via an IAM policy:
- ```acm-pca:*``` : This is so that Private CA's maybe be created and deleted via the appropriate APIs for testing
- If you did not provider a user via PLUGIN_USER_NAME_OVERRIDE, the test suite can create a user for you. This will require the following permissions: ```iam:CreatePolicy```,```iam:CreateUser```, and ```iam:AttachUserPolicy```
- ```iam:CreateAccessKey``` and ```iam:DeleteAccessKey```: This allow us to create and delete access keys to be used to validate that authentication via K8 secrets is functional. If the user was set via $PLUGIN_USER_NAME_OVERRIDE
- ```s3:PutObject``` and ```s3::PutObjectAcl``` these can be scoped down to the s3 bucket you created above

\- [An AWS IAM OIDC Provider](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_create_oidc.html). Before creating the OIDC provider, set a temporary value for $OIDC_IAM_ROLE (```export OIDC_IAM_ROLE=arn:aws:iam::000000000000:role/oidc-kind-cluster-role``` and run ```make cluster && make install-eks-cluster && make kind-cluster-delete```). This needs to be done otherwise you may see an error complaining about the absense of a file .well-known/openid-configuration. Running these commands helps bootstrap the S3 bucket so that the OIDC provider can be created. Set the provider url of the OIDC provider to be ```$OIDC_S3_BUCKET_NAME.s3.us-east-1.amazonaws.com/cluster/my-oidc-cluster]```. Set the audience to be ```sts.amazonaws.com```.

\- An IAM role that has a trust relationship with the IAM OIDC Provider that was just created. An inline policy for this role can be grabbed from [Configuration](#configuration) except you can't scope it to a particular CA since those will be created during the test run. This role will be used to test authentication in the plugin via IRSA. The trust relationship should look something like:
```
{  
  "Version": "2012-10-17",  
  "Statement": [  
	{  
      "Effect": "Allow",  
	  "Principal": {  
	    "Federated": "${OIDC_ARN}"  
	   },  
	   "Action": "sts:AssumeRoleWithWebIdentity",  
	   "Condition": {  
	     "StringEquals": {  
	       "${OIDC_URL}:sub:system:serviceaccount:aws-privateca-issuer:aws-privateca-issuer-sa"  
	     }  
	   }  
	 }  
   ]  
}
```
After creating this role run ```export OIDC_IAM_ROLE=<IAM role arn you created above>```

\- ```make install-eks-webhook``` will install a webhook in that kind cluster that will enable the use of IRSA

\- ```make e2etest``` will run end-to-end test against the kind cluster created via ```make cluster```.

Getting IRSA to work on Kind was heavily inspired by the following blog: https://reece.tech/posts/oidc-k8s-to-aws/

If you want to also test that cross account issuers are working, you will need:

\- A seperate AWS account that has a role that trust the caller who kicks off the end-to-end test via the CLI, the role will need a policy with the following permissions
- ```acm-pca:*```: This is so the test can create a Private CA is the other account
- ```ram:GetResourceShareAssociations```, ```ram:CreateResourceShare```, and ```ram:DeleteResourceShare```: These allow the creation of a CA that can be shared with the source (caller) account
- After creating this role you will need to run ```export PLUGIN_CROSS_ACCOUNT_ROLE=<name of the role you created above>```. If you do not do this, you will see a message about cross account testing being skipped due to this enviornment variable not being set.

Soon these test should be automatically run on each PR, but for the time being each PR will have a core-collaborator for the project run the tests manually to ensure no regressions on the supported workflows


### Contributing to the End-to-End test

The test are fairly straightforward, they will take a set of "issuer templates" (Base name for a aws-pca-issuer as well as a AWSIssuerSpec) and a set of "certificate templates" (Base name for type of certificate as well as a certificate spec). The tests will then take every certificate spec and apply it to each issuer spec. The test will ensure all issuers made from issuer specs reach a ready state as well as ensure that each certificate issued off a issuer reaches a ready state. The issuers with the different certificates is verified to be working for both cluster and namespace issuers.

For the most part, updating end-to-end will be updating these "issuer specs" and "certificate specs" which reside within e2e/e2e_test.go. If the test need updating beyond that, the core logic for the test is also embedded in e2e/e2e_test.go. The other files within the e2e folder are mainly utilities that shouldn't require frequent update

## Supported workflows

AWS Private Certificate Authority(PCA) Issuer Plugin supports the following integrations and use cases:

* Integration with [cert-manager 1.4+](https://cert-manager.io/docs/installation/supported-releases/) and corresponding Kubernetes versions.

* Authentication methods:
    * [KIAM v4.0](https://github.com/uswitch/kiam)
    * [IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) - IAM roles for service accounts
    * [Kubernetes Secrets](#authentication)
    * [EC2 Instance Profiles](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use_switch-role-ec2_instance-profiles.html)

* AWS Private CA features:
    * [End-to-End TLS encryption on Amazon Elastic Kubernetes Service](https://aws.amazon.com/blogs/containers/setting-up-end-to-end-tls-encryption-on-amazon-eks-with-the-new-aws-load-balancer-controller/)(Amazon EKS).
    * [TLS-enabled Kubernetes clusters with AWS Private CA and Amazon EKS](https://aws.amazon.com/blogs/security/tls-enabled-kubernetes-clusters-with-acm-private-ca-and-amazon-eks-2/)
    * Cross Account CA sharing with supported Cross Account templates
    * [Supported PCA Certificate Templates](https://docs.aws.amazon.com/acm-pca/latest/userguide/UsingTemplates.html#template-varieties): CodeSigningCertificate/V1; EndEntityClientAuthCertificate/V1; EndEntityServerAuthCertificate/V1; OCSPSigningCertificate/V1; EndEntityCertificate/V1; BlankEndEntityCertificate_CSRPassthrough/V1

## Troubleshooting

1. Check the secret with the AWS credentials: AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY values have to be base64 encoded.

2. If the generated CertificateRequest shows no events, it is very likely that you're using an older version of cert-manager which doesn't support approval check. Disable approval check at the issuer deployment.

## Help & Feedback

For help, please consider the following venues (in order):

* Check the [Troubleshooting section](#troubleshooting)
* [Search open issues](https://github.com/cert-manager/aws-privateca-issuer/issues)
* [File an issue](https://github.com/cert-manager/aws-privateca-issuer/issues/new)
* Please ask questions in the slack channel [#cert-manager-aws-privateca-issuer](https://kubernetes.slack.com/archives/C02FEDR3FN2)


## Contributing

We welcome community contributions and pull requests.

See our [contribution guide](CONTRIBUTING.md) for more information on how to report issues, set up a development environment, and submit code.

We adhere to the [Amazon Open Source Code of Conduct](https://aws.github.io/code-of-conduct).
