<p align="center"><img src="assets/logo.png" alt="Logo" width="250px" /></p>
<p align="center">
<a href="https://goreportcard.com/report/github.com/cert-manager/aws-privateca-issuer">
<img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/cert-manager/aws-privateca-issuer" />
</a>
<img alt="Latest version" src="https://img.shields.io/github/v/release/cert-manager/aws-privateca-issuer?color=success&sort=semver" />
</p>

# AWS Private CA Issuer 12


> [!TIP]
> Amazon Elastic Kubernetes Service (EKS) supports AWS Private CA Issuer as an EKS Add-on named `aws-privateca-connector-for-kubernetes`. This simplifies installation and configuration for Amazon EKS users. See <a href="https://docs.aws.amazon.com/eks/latest/userguide/workloads-add-ons-available-eks.html#add-ons-aws-privateca-connector">AWS add-ons</ulink> for more information.

AWS Private CA is an AWS service that can setup and manage private CAs, as well as issue private certificates.

cert-manager is a Kubernetes add-on to automate the management and issuance of TLS certificates from various issuing sources.
It will ensure certificates are valid, updated periodically and attempt to renew certificates at an appropriate time before expiry.

This project acts as an addon (see https://cert-manager.io/docs/configuration/external/) to cert-manager that signs off certificate requests using AWS Private CA.

## Setup

Install cert-manager first (https://cert-manager.io/docs/installation/kubernetes/).

Then install AWS PCA Issuer using Helm:

```shell
helm repo add awspca https://cert-manager.github.io/aws-privateca-issuer
helm install awspca/aws-privateca-issuer --generate-name
```

You can check the chart configuration in the default [values](charts/aws-pca-issuer/values.yaml) file.

**[AWS PCA Issuer supports ARM starting at version 1.3.0](https://github.com/cert-manager/aws-privateca-issuer/releases/tag/v1.3.0)**

### Accessing the test ECR

AWS PCA Issuer maintains a test ECR that contains versions that correspond to each commit on the main branch. These images can be accessed by setting the image repo to `public.ecr.aws/cert-manager-aws-privateca-issuer/cert-manager-aws-privateca-issuer-test` and the image tag to `latest`. An example of how this is done is shown below:

```shell
helm repo add awspca https://cert-manager.github.io/aws-privateca-issuer
helm install awspca/aws-privateca-issuer --generate-name \
--set image.repository=public.ecr.aws/cert-manager-aws-privateca-issuer/cert-manager-aws-privateca-issuer-test \
--set image.tag=latest
```
## Configuration

As of now, the only configurable settings are access to AWS. So you can use `AWS_REGION`, `AWS_ACCESS_KEY_ID` or `AWS_SECRET_ACCESS_KEY`.

Alternatively, you can supply arbitrary secrets for the access and secret keys with the `accessKeyIDSelector` and `secretAccessKeySelector` fields in the clusterissuer and/or issuer manifests.

Access to AWS can also be configured using an EC2 instance role or [IAM Roles for Service Accounts] (https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html).

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

Examples can be found in the [examples](config/examples/) and [samples](config/samples) directories.

### AWSPCAIssuer

This is a regular namespaced issuer that can be used as a reference in your Certificate CRs.

### AWSPCAClusterIssuer

This CR is identical to the AWSPCAIssuer. The only difference being that it's not namespaced and can be referenced from anywhere.

### Usage with cert-manager Ingress Annotations

The `cert-manager.io/cluster-issuer` annotation cannot be used to point at a `AWSPCAClusterIssuer`. Instead, use `cert-manager.io/issuer:`. Please see [this issue](https://github.com/cert-manager/aws-privateca-issuer/issues/252) for more information.

### Disable Approval Check

The AWSPCA Issuer will wait for CertificateRequests to have an [approved condition
set](https://cert-manager.io/docs/concepts/certificaterequest/#approval) before
signing. If using an older version of cert-manager (pre v1.3), you can disable
this check by supplying the command line flag `-disable-approved-check` to the
Issuer Deployment.

### Disable Kubernetes Client-Side Rate Limiting

The AWSPCA Issuer will throttle the rate of requests to the kubernetes API server to 5 queries per second by [default](https://pkg.go.dev/k8s.io/client-go/rest#pkg-constants). This is not necessary for newer versions of Kubernetes that have implemented [API Priority and Fairness](https://kubernetes.io/docs/concepts/cluster-administration/flow-control/). If using a newer version of Kubernetes, you can disable this client-side rate limiting by supplying the command line flag `-disable-client-side-rate-limiting` to the Issuer Deployment.

### Authentication

Please note that if you are using [KIAM](https://github.com/uswitch/kiam) for authentication, this plugin has been tested on KIAM v4.0. [IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) is also tested and supported.

There is a custom AWS authentication method we have coded into our plugin that allows a user to define a [Kubernetes secret](https://kubernetes.io/docs/concepts/configuration/secret/) with AWS Creds passed in, example [here](config/samples/secret.yaml). The user applies that file with their creds and then references the secret in their Issuer CRD when running the plugin, example [here](config/samples/awspcaclusterissuer_ec/_v1beta1_awspcaclusterissuer_ec.yaml#L8-L10).

#### IAM Roles Anywhere

For use cases where the AWS Private CA issuer needs to run outside of AWS, IAM Roles Anywhere can be used as an alternative to IAM Users.

The helm chart supports `extraContainers` which can be used to deploy the [aws_signing_helper](https://github.com/aws/rolesanywhere-credential-helper) in "serve" mode. Then, we can set `AWS_EC2_METADATA_SERVICE_ENDPOINT="http://127.0.0.1:9911"` on the `aws-privateca-issuer` itself.

A simplified example of what to set for your helm values is as follows:

```
env:
  AWS_EC2_METADATA_SERVICE_ENDPOINT: "http://127.0.0.1:9911"
extraContainers:
  - name: "rolesanywhere-credential-helper"
    image: "public.ecr.aws/rolesanywhere/credential-helper:latest"
    command: ["aws_signing_helper"]
    args:
      - "serve"
      - "--private-key"
      - "/etc/cert/tls.key"
      - "--certificate"
      - "/etc/cert/tls.crt"
      - "--role-arn"
      - "$ROLE_ARN"
      - "--profile-arn"
      - "$PROFILE_ARN"
      - "--trust-anchor-arn"
      - "$TRUST_ANCHOR_ARN"
    volumeMounts:
      - name: cert
        mountPath: /etc/cert/
        readOnly: true
volumes:
  - name: cert
    secret:
      secretName: cert
```

#### Cross Account Assume Role

You can configure the AWS Private CA issuer to assume an IAM role before making AWS Private CA API calls. This method provides flexibility for various authentication scenarios and is an alternative to AWS Resource Access Manager (RAM) sharing for cross-account use cases.

When you specify a role in the `role` field, the issuer assumes that role using AWS Security Token Service (STS) before making AWS Private CA API calls.

Example:
```
apiVersion: awspca.cert-manager.io/v1beta1
kind: AWSPCAClusterIssuer
metadata:
  name: example
spec:
  arn: <some-pca-arn>
  role: <some-role-arn>
  region: <some-region>
```

## Supported workflows

AWS Private Certificate Authority(PCA) Issuer Plugin supports the following integrations and use cases:

* Integration with [cert-manager 1.4+](https://cert-manager.io/docs/installation/supported-releases/) and corresponding Kubernetes versions.

* Authentication methods:
    * [KIAM v4.0](https://github.com/uswitch/kiam)
    * [IRSA](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) - IAM roles for service accounts
    * [Kubernetes Secrets](#authentication)
    * [EC2 Instance Profiles](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use_switch-role-ec2_instance-profiles.html)
    * [IAM Roles Anywhere](https://docs.aws.amazon.com/rolesanywhere/latest/userguide/introduction.html)

* AWS Private CA features:
    * [End-to-End TLS encryption on Amazon Elastic Kubernetes Service](https://aws.amazon.com/blogs/containers/setting-up-end-to-end-tls-encryption-on-amazon-eks-with-the-new-aws-load-balancer-controller/)(Amazon EKS).
    * [TLS-enabled Kubernetes clusters with AWS Private CA and Amazon EKS](https://aws.amazon.com/blogs/security/tls-enabled-kubernetes-clusters-with-acm-private-ca-and-amazon-eks-2/)
    * Cross Account CA sharing with supported Cross Account templates
    * [Supported PCA Certificate Templates](https://docs.aws.amazon.com/acm-pca/latest/userguide/UsingTemplates.html#template-varieties): CodeSigningCertificate/V1; EndEntityClientAuthCertificate/V1; EndEntityServerAuthCertificate/V1; OCSPSigningCertificate/V1; EndEntityCertificate/V1; BlankEndEntityCertificate_APICSRPassthrough/V1


## Mapping Cert-Manager Usage Types to AWS PCA Template Arns

The code for the translation can be found [here](https://github.com/cert-manager/aws-privateca-issuer/blob/main/pkg/aws/pca.go#L177).

Depending on which UsageTypes are set in the Cert-Manager certificate, different AWS PCA templates will be used.
This table shows how the UsageTypes are being translated into which template to use when making an IssueCertificate request:

| Cert-Manager Usage Type(s) | AWS PCA Template ARN                                             |
| -------------------------- | ---------------------------------------------------------------- |
| CodeSigning                | acm-pca:::template/CodeSigningCertificate/V1                     |
| ClientAuth                 | acm-pca:::template/EndEntityClientAuthCertificate/V1             |
| ServerAuth                 | acm-pca:::template/EndEntityServerAuthCertificate/V1             |
| OCSPSigning                | acm-pca:::template/OCSPSigningCertificate/V1                     |
| ClientAuth, ServerAuth     | acm-pca:::template/EndEntityCertificate/V1                       |
| Everything Else            | acm-pca:::template/BlankEndEntityCertificate_APICSRPassthrough/V1   |

## Understanding/Running the tests

### Running the Unit Tests
Running ```make test``` will run the written unit test

If you run into an issue like

```
/home/linuxbrew/.linuxbrew/Cellar/go/1.17/libexec/src/net/cgo_linux.go:13:8: no such package located
```

This can be fixed with a

```
brew install gcc@5
```

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
* [Golang v1.17+](https://golang.org/)
* [Docker v17.03+](https://docs.docker.com/install/)
* [Kind v0.9.0+](https://kind.sigs.k8s.io/docs/user/quick-start/) -> This will be installed via running the test
* [Kubectl v1.13+](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* [AWS CLI v2](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html)
* [Helm](https://helm.sh/docs/intro/install/)
* [Make](https://www.gnu.org/software/make/) Need to have version 3.82+


\- (Optional) You will need a AWS IAM User to test authentication via K8 secrets. You can provide an already existing user into the test via ```export PLUGIN_USER_NAME_OVERRIDE=<IAM User Name>```.  This IAM User should have a policy attached to it that follows with the policy listed in [Configuration](#configuration). This user will be used to test authentication in the plugin via K8 secrets.

\- An S3 Bucket with [BPA disabled](https://docs.aws.amazon.com/AmazonS3/latest/userguide/access-control-block-public-access.html) in us-east-1. After creating the bucket run ```export OIDC_S3_BUCKET_NAME=<Name of bucket you just created>```

\- You will need AWS credentials loaded into your terminal that, via the CLI, minimally allow the following actions via an IAM policy:
- ```acm-pca:*``` : This is so that Private CA's maybe be created and deleted via the appropriate APIs for testing
- If you did not provider a user via PLUGIN_USER_NAME_OVERRIDE, the test suite can create a user for you. This will require the following permissions: ```iam:CreatePolicy```,```iam:CreateUser```, and ```iam:AttachUserPolicy```
- ```iam:CreateAccessKey``` and ```iam:DeleteAccessKey```: This allow us to create and delete access keys to be used to validate that authentication via K8 secrets is functional. If the user was set via $PLUGIN_USER_NAME_OVERRIDE
- ```s3:PutObject``` and ```s3::PutObjectAcl``` these can be scoped down to the s3 bucket you created above

\- [An AWS IAM OIDC Provider](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_create_oidc.html). Before creating the OIDC provider, set a temporary value for $OIDC_IAM_ROLE (```export OIDC_IAM_ROLE=arn:aws:iam::000000000000:role/oidc-kind-cluster-role``` and run ```make cluster && make install-eks-webhook && make kind-cluster-delete```). This needs to be done otherwise you may see an error complaining about the absence of a file .well-known/openid-configuration. Running these commands helps bootstrap the S3 bucket so that the OIDC provider can be created. Set the provider url of the OIDC provider to be ```$OIDC_S3_BUCKET_NAME.s3.us-east-1.amazonaws.com/cluster/my-oidc-cluster```. Set the audience to be ```sts.amazonaws.com```.

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
	       "${OIDC_URL}:sub": "system:serviceaccount:aws-privateca-issuer:aws-privateca-issuer-sa"  
	     }  
	   }  
	 }  
   ]  
}
```
After creating this role run ```export OIDC_IAM_ROLE=<IAM role arn you created above>```

\- ```make cluster``` recreate the cluster with all the appropriate enviornment variables set

\- ```make install-eks-webhook``` will install a webhook in that kind cluster that will enable the use of IRSA

\- ```make e2etest``` will run end-to-end test against the kind cluster created via ```make cluster```.

\- After you update controller code locally, an easy way to redeploy the new controller code and re-run end-to-end test is to run:: ```make upgrade-local && make e2etest```

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

### Other Tests

1. Test to ensure that the workflow laid out in the blog [Setting up end-to-end TLS encryption on Amazon EKS with the new AWS Load Balancer Controller](https://aws.amazon.com/blogs/containers/setting-up-end-to-end-tls-encryption-on-amazon-eks-with-the-new-aws-load-balancer-controller/) is functional. To run the test: ```make cluster && make install-eks-webhook && make blog-test```

2. Test that pulls down the latest release via Helm, checks that the plugin was installed correctly, with the correct version, then gets deleted correctly. To run the test ```make cluster && make install-eks-webhook && make helm-test```

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
