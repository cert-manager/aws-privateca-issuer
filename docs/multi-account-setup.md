# Multi-Account AWS Private CA Setup

This document explains how to use the AWS Private CA Issuer with AWS Private Certificate Authorities (PCAs) hosted in different AWS accounts.

## Overview

Organizations often have AWS PCAs located in different AWS accounts while their Kubernetes clusters exist in a separate centralized account. The AWS Private CA Issuer now supports assuming an IAM role per Issuer, enabling seamless integration with AWS PCAs across different accounts.

## Prerequisites

Before setting up cross-account access, ensure you have:

1. An AWS Private CA created in a source AWS account
2. A Kubernetes cluster with cert-manager and the AWS PCA Issuer installed
3. Proper IAM roles and permissions configured across accounts

## IAM Role Setup

In the AWS account hosting the PCA, create an IAM role that the AWS PCA Issuer can assume:

1. Create a new IAM role with a descriptive name (e.g., `pca-issuer-role`)
2. Configure a trust policy allowing the Kubernetes cluster's AWS account to assume this role:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::<CLUSTER_AWS_ACCOUNT_ID>:role/<CLUSTER_NODE_ROLE>"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
```

3. Attach a policy to the role with the necessary permissions to use the AWS Private CA:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "acm-pca:DescribeCertificateAuthority",
        "acm-pca:IssueCertificate",
        "acm-pca:GetCertificate"
      ],
      "Resource": "arn:aws:acm-pca:<REGION>:<PCA_AWS_ACCOUNT_ID>:certificate-authority/<CA_ID>"
    }
  ]
}
```

## Configuring the Issuer

To configure an issuer to use a PCA in a different account, specify the ARN of the PCA and the role to assume:

```yaml
apiVersion: awspca.cert-manager.io/v1beta1
kind: AWSPCAClusterIssuer
metadata:
  name: cross-account-issuer
spec:
  arn: arn:aws:acm-pca:<REGION>:<PCA_AWS_ACCOUNT_ID>:certificate-authority/<CA_ID>
  region: <REGION>
  role: arn:aws:iam::<PCA_AWS_ACCOUNT_ID>:role/pca-issuer-role
```

For a namespace-scoped issuer:

```yaml
apiVersion: awspca.cert-manager.io/v1beta1
kind: AWSPCAIssuer
metadata:
  name: cross-account-issuer
  namespace: cert-manager
spec:
  arn: arn:aws:acm-pca:<REGION>:<PCA_AWS_ACCOUNT_ID>:certificate-authority/<CA_ID>
  region: <REGION>
  role: arn:aws:iam::<PCA_AWS_ACCOUNT_ID>:role/pca-issuer-role
```

## Authentication Flow

When the AWS PCA Issuer reconciles an issuer with a role specified:

1. The issuer first authenticates using the cluster's AWS credentials (instance profile, EKS Pod Identity, or provided static credentials)
2. It then assumes the specified IAM role in the target account
3. The temporary credentials from the assumed role are used for all operations with the AWS Private CA

## Multiple Issuers for Different Accounts

You can create multiple issuers, each assuming a different role to access PCAs in different accounts:

```yaml
---
apiVersion: awspca.cert-manager.io/v1beta1
kind: AWSPCAClusterIssuer
metadata:
  name: account-a-issuer
spec:
  arn: arn:aws:acm-pca:us-west-2:111111111111:certificate-authority/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa
  region: us-west-2
  role: arn:aws:iam::111111111111:role/pca-issuer-role-a
---
apiVersion: awspca.cert-manager.io/v1beta1
kind: AWSPCAClusterIssuer
metadata:
  name: account-b-issuer
spec:
  arn: arn:aws:acm-pca:us-east-1:222222222222:certificate-authority/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb
  region: us-east-1
  role: arn:aws:iam::222222222222:role/pca-issuer-role-b
```

## Troubleshooting

If you encounter issues with cross-account setup:

1. Verify IAM role permissions and trust relationships
2. Check that the Kubernetes cluster has proper permissions to assume the role
3. Look for error messages in the AWS PCA Issuer logs: `kubectl logs -n cert-manager deploy/aws-privateca-issuer`
4. Test the IAM role assumption directly using the AWS CLI before troubleshooting the issuer
