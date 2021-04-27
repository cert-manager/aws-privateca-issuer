<p align="center"><img src="assets/logo.png" alt="Logo" width="250px" /></p>
<p align="center">
<a href="https://github.com/jniebuhr/aws-pca-issuer/actions">
<img alt="Build Status" src="https://github.com/jniebuhr/aws-pca-issuer/workflows/CI/badge.svg" />
</a>
<a href="https://goreportcard.com/report/github.com/jniebuhr/aws-pca-issuer">
<img alt="Build Status" src="https://goreportcard.com/badge/github.com/jniebuhr/aws-pca-issuer" />
</a>
<img alt="Latest version" src="https://img.shields.io/github/v/release/jniebuhr/aws-pca-issuer?color=success&sort=semver" />
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
helm repo add awspca https://jniebuhr.github.io/aws-pca-issuer/
helm install awspca/aws-pca-issuer --generate-name
```

You can check the chart configuration in the default [values](https://github.com/jniebuhr/aws-pca-issuer/blob/master/charts/aws-pca-issuer/values.yaml) file.


## Configuration

As of now, the only configurable settings are access to AWS. So you can use `AWS_REGION`, `AWS_ACCESS_KEY_ID` or `AWS_SECRET_ACCESS_KEY`.

Access to AWS can also be configured using an EC2 instance role.

A minimal policy to use the issuer with an authority would look like follows:

```
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "aws-pca-issuer",
      "Action": [
        "acm-pca:GetCertificate",
        "acm-pca:GetCertificateAuthorityCertificate",
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

Examples can be found in the [examples](https://github.com/jniebuhr/aws-pca-issuer/tree/master/config/examples/) directory.

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
