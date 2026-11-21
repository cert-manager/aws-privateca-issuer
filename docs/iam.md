# IAM for AWS Private CA Issuer

This page covers how the controller gets AWS credentials for each Issuer, and the IAM policies it needs.

- [Recommended setup](#recommended-setup)
- [EKS Pod Identity](#eks-pod-identity) and [the IAM policy](#iam-policy)
- [How an Issuer gets credentials](#how-an-issuer-gets-credentials), including [ambient credentials](#ambient-credentials)
- [IRSA per Issuer](#irsa-per-issuer)
- [CAs in other accounts](#cas-in-other-accounts)
- [IAM Roles Anywhere](#iam-roles-anywhere)
- [Not recommended](#not-recommended)

An Issuer's `spec.arn` is the private CA it signs with, as in the [example Issuer](../README.md#usage).
The controller calls `acm-pca:DescribeCertificateAuthority`, `acm-pca:IssueCertificate` and `acm-pca:GetCertificate` on that CA.
If an Issuer sets `spec.role`, the controller first assumes it with `sts:AssumeRole`, or with `sts:AssumeRoleWithWebIdentity` when the Issuer also sets `serviceAccountRef`.
It also calls `sts:GetCallerIdentity`, which needs no permission, to log which identity each Issuer uses.
It doesn't create or modify AWS resources.

## Recommended setup

Which credentials the controller uses depends on the Issuer's kind:

| | `AWSPCAClusterIssuer` (cluster-scoped, usually created by an administrator) | `AWSPCAIssuer` (namespaced, often created by other teams) |
|---|---|---|
| The controller's own credentials, such as [EKS Pod Identity](#eks-pod-identity) | Used by default. Set the chart's `disableClusterIssuerAmbientCredentials: true` to turn this off. | Not used by default. Set the chart's `enableIssuerAmbientCredentials: true` to allow it, which lets every namespace use the controller's role. |
| [IRSA per Issuer](#irsa-per-issuer), with `serviceAccountRef` and `role` | Supported. `serviceAccountRef.namespace` is required. | Supported, with a `ServiceAccount` in the Issuer's own namespace. |

For a cluster shared by several teams, use EKS Pod Identity for `AWSPCAClusterIssuers`, and IRSA per Issuer for each team's `AWSPCAIssuers`.
Each team's IAM role then decides which CAs its Issuers can use.

## EKS Pod Identity

[EKS Pod Identity](https://docs.aws.amazon.com/eks/latest/userguide/pod-identities.html) is the recommended way to give the controller credentials.
It needs the [EKS Pod Identity Agent](https://docs.aws.amazon.com/eks/latest/userguide/pod-id-agent-setup.html) add-on.
Create an IAM role with the [EKS Pod Identity trust policy](https://docs.aws.amazon.com/eks/latest/userguide/pod-id-role.html), attach [the IAM policy](#iam-policy), and associate it with the controller's `ServiceAccount`.
With the install command in the [README](../README.md#setup), that `ServiceAccount` is `aws-privateca-issuer` in the `aws-privateca-issuer` namespace:

```sh
aws eks create-pod-identity-association --cluster-name my-cluster \
  --namespace aws-privateca-issuer --service-account aws-privateca-issuer \
  --role-arn arn:aws:iam::111122223333:role/my-cluster-pca-issuer
```

Pod Identity is assigned to the controller's `ServiceAccount`, not to individual Issuers.
For a separate role per Issuer, use [IRSA per Issuer](#irsa-per-issuer).

## IAM policy

In the role's policy, list the ARN of each CA the controller's Issuers may use:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "awspcaissuer",
      "Effect": "Allow",
      "Action": [
        "acm-pca:DescribeCertificateAuthority",
        "acm-pca:GetCertificate",
        "acm-pca:IssueCertificate"
      ],
      "Resource": "arn:aws:acm-pca:<region>:<account_id>:certificate-authority/<resource_id>"
    }
  ]
}
```

Every Issuer that uses the role can use all of those CAs.
The AWS managed policy [`AWSPrivateCAConnectorForKubernetesPolicy`](https://docs.aws.amazon.com/aws-managed-policy/latest/reference/AWSPrivateCAConnectorForKubernetesPolicy.html) grants these actions on every CA, so it doesn't limit which CAs the controller's Issuers can use.
Write your own policy to limit which CAs the controller can use.

## How an Issuer gets credentials

The credentials the controller uses for an Issuer depend on its spec:

| The Issuer sets | Credentials |
|---|---|
| Neither `secretRef` nor `serviceAccountRef` | The controller's own credentials, found by the AWS SDK default credential chain: EKS Pod Identity, IRSA, IAM Roles Anywhere or the instance profile. These are [ambient credentials](#ambient-credentials). |
| `serviceAccountRef` and `role` | A token for that `ServiceAccount`, exchanged for `role` with `sts:AssumeRoleWithWebIdentity`. See [IRSA per Issuer](#irsa-per-issuer). |
| `secretRef` | Access keys from a Kubernetes `Secret`. Using access keys from a `Secret` is deprecated. |

An Issuer that sets `role` without `serviceAccountRef` first gets ambient credentials, or the `Secret`'s access keys if it sets `secretRef`.
It then assumes `role` with them, as described in [`spec.role`](#specrole).

### Ambient credentials

Ambient credentials are the controller's own AWS credentials.
The controller flags `--cluster-issuer-ambient-credentials` (default `true`) and `--issuer-ambient-credentials` (default `false`) decide whether `AWSPCAClusterIssuers` and `AWSPCAIssuers` may use them.
The chart sets them with `disableClusterIssuerAmbientCredentials` and `enableIssuerAmbientCredentials`.
For example, to let `AWSPCAIssuers` use ambient credentials:

```yaml
enableIssuerAmbientCredentials: true
```

`AWSPCAIssuers` are usually created in other teams' namespaces, so by default they don't use ambient credentials.
With ambient credentials enabled for them, anyone who can create an `AWSPCAIssuer` can issue certificates from any CA the controller's role allows.
An `AWSPCAIssuer` that sets only `role` also uses ambient credentials, because the controller assumes that role with its own credentials.

## IRSA per Issuer

Set `serviceAccountRef` to give an Issuer its own IAM role through IRSA, instead of the controller's credentials.
The controller requests a short-lived token for that `ServiceAccount`, and uses it to assume the Issuer's `role`.
To set it up:

1. Create an IAM role whose trust policy allows the cluster's OIDC provider for `system:serviceaccount:<namespace>:<name>`, as for any IRSA role.
   `serviceAccountRef` works only through IRSA, so the controller ignores any EKS Pod Identity association on the `ServiceAccount`.
2. Let the controller request tokens for that `ServiceAccount`, in the chart's `serviceAccountTokenRequests`, or with your own `Role` and `RoleBinding` that grant `create` on `serviceaccounts/token`:

   ```yaml
   serviceAccountTokenRequests:
     - namespace: team-a
       name: pca-issuer
   ```

3. Set `role` and `serviceAccountRef`, without `secretRef`, on the Issuer.
   An `AWSPCAIssuer` can only name a `ServiceAccount` in its own namespace; an `AWSPCAClusterIssuer` must set `serviceAccountRef.namespace`.

See [config/examples/issuer-with-service-account](../config/examples/issuer-with-service-account/issuer.yaml) for an example.

By default, the chart also lets Issuers name the controller's own `ServiceAccount` in `serviceAccountRef`.
Using the controller's own `ServiceAccount` there is deprecated, because it gives an Issuer the controller's IAM role even when ambient credentials are disabled for it.
To turn it off, set `enableLegacyControllerServiceAccountToken: false`, and add the controller's `ServiceAccount` to `serviceAccountTokenRequests` if you still need it.

## CAs in other accounts

There are three ways to use a CA in another account: share it with AWS Resource Access Manager (RAM), use an EKS Pod Identity target role, or set `spec.role`.

### AWS RAM

A CA in another account can be shared with yours through RAM.
The RAM permission you choose decides which certificate templates your account can use, so pick one that covers the templates in [Using AWS Private CA templates](../README.md#using-aws-private-ca-templates).

### EKS Pod Identity target role

To use a role in another account, prefer an EKS Pod Identity [target role](https://docs.aws.amazon.com/eks/latest/userguide/pod-id-assign-target-role.html) over `spec.role`.
EKS assumes the target role through the controller's role, and gives the controller the target role's credentials:

```sh
aws eks create-pod-identity-association --cluster-name my-cluster \
  --namespace aws-privateca-issuer --service-account aws-privateca-issuer \
  --role-arn arn:aws:iam::111122223333:role/my-cluster-pca-issuer \
  --target-role-arn arn:aws:iam::444455556666:role/my-cluster-pca-ca
```

The cluster administrator chooses the role in the EKS API, rather than whoever can create an Issuer.
The controller's role needs `sts:AssumeRole` and `sts:TagSession` on only that target role, and the target role's trust policy must allow both.
The EKS guide has both policies.
The target role is the same for every Issuer that uses the controller's credentials, and its policy lists the CAs it may use, as in [IAM policy](#iam-policy).
For a role per Issuer, use `spec.role` or [IRSA per Issuer](#irsa-per-issuer).

### `spec.role`

When an Issuer sets `role`, the controller assumes that role with AWS Security Token Service (STS) before making AWS Private CA API calls:

```yaml
apiVersion: awspca.cert-manager.io/v1beta1
kind: AWSPCAClusterIssuer
metadata:
  name: example
spec:
  arn: <some-pca-arn>
  role: <some-role-arn>
  region: <some-region>
```

Whoever can create an Issuer chooses the role, so limit which roles the controller can assume.
The assumed role's trust policy must allow the controller's role to call `sts:AssumeRole` and `sts:TagSession`.
With EKS Pod Identity, the controller's session tags are transitive, so `sts:TagSession` is required.
Unless the trust policy names the controller's role in the same account, the controller's role policy must also allow both actions, with each role's ARN in `Resource` rather than a wildcard.
If `sts:TagSession` isn't allowed, `sts:AssumeRole` fails with `not authorized to perform: sts:TagSession`.

## IAM Roles Anywhere

If the controller runs outside AWS, use IAM Roles Anywhere instead of long-lived access keys.
Run the [aws_signing_helper](https://github.com/aws/rolesanywhere-credential-helper) in `serve` mode with the chart's `extraContainers`, and point the controller at it with the chart's `env` value.
These are the controller's own credentials, so they are [ambient credentials](#ambient-credentials).
Set these chart values:

```yaml
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
      - "<role-arn>"
      - "--profile-arn"
      - "<profile-arn>"
      - "--trust-anchor-arn"
      - "<trust-anchor-arn>"
    volumeMounts:
      - name: cert
        mountPath: /etc/cert/
        readOnly: true
volumes:
  - name: cert
    secret:
      secretName: cert
```

## Not recommended

EC2 instance roles, which every pod on the node can use, are not recommended.
Using AWS access keys from a Kubernetes `Secret` (`secretRef`) is deprecated: the keys are long-lived, and to support them, the chart gives the controller read access to `Secrets` in every namespace.
`Secret` credentials are enabled by default for backwards compatibility.
Set the chart's `enableLegacySecretCredentials: false` to turn them off.
