# Contributing

You can find our full contributing guide on [our website](https://cert-manager.io/docs/contributing/).

## DCO Sign off

All contributors to the project retain copyright to their work. However, to ensure
that they are only submitting work that they have rights to, we require
everyone to acknowledge this by signing their work.

Any copyright notices in this repo should specify the authors as
"The cert-manager Authors".

To sign your work, just add a line like this at the end of your commit message:

```text
Signed-off-by: Joe Bloggs <joe@example.com>
```

This can easily be done with the `--signoff` option to `git commit`.
You can also mass sign-off a whole PR with `git rebase --signoff main`, replacing
`main` with the branch you are creating a pull request against if not main.

By doing this you state that you can certify the following (from https://developercertificate.org/):

```text
Developer Certificate of Origin
Version 1.1

Copyright (C) 2004, 2006 The Linux Foundation and its contributors.
1 Letterman Drive
Suite D4700
San Francisco, CA, 94129

Everyone is permitted to copy and distribute verbatim copies of this
license document, but changing it is not allowed.


Developer's Certificate of Origin 1.1

By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or

(b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or

(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.

(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
```

## Accessing the test ECR

Each commit on the main branch is published to `public.ecr.aws/cert-manager-aws-privateca-issuer/cert-manager-aws-privateca-issuer-test` with the tag `latest`.
To install it:

```shell
helm repo add awspca https://cert-manager.github.io/aws-privateca-issuer
helm install aws-privateca-issuer awspca/aws-privateca-issuer \
  --namespace aws-privateca-issuer --create-namespace \
  --set image.repository=public.ecr.aws/cert-manager-aws-privateca-issuer/cert-manager-aws-privateca-issuer-test \
  --set image.tag=latest
```

## Running the tests

### Unit tests

`make test` runs the unit tests.

### End-to-end tests

Running these tests **will incur charges in your AWS account**.

`make e2etest` builds an image from the current code, deploys it to a kind cluster, and issues certificates through `AWSPCAIssuers` and `AWSPCAClusterIssuers` that use IRSA, `serviceAccountRef`, `secretRef` and, optionally, a CA in another account.
To run them, set up [`make cluster`](#setting-up-make-cluster) first, then run `make cluster && make install-eks-webhook && make e2etest`.

In CI, the end-to-end tests run on a pull request when a maintainer adds the `safe to test` label.

### Setting up `make cluster`

`make cluster` creates a kind cluster on your machine, and installs cert-manager and AWS Private CA Issuer built from the current branch.
Before running it, install:

- [Git](https://git-scm.com/)
- [Go](https://go.dev/), at the version in `go.mod`
- [Docker](https://docs.docker.com/install/)
- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/) (optional; the tests install it if it's missing)
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [AWS CLI v2](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html)
- [Helm](https://helm.sh/docs/intro/install/)
- [GNU Make](https://www.gnu.org/software/make/) 3.82 or later

You also need AWS credentials and these resources, in us-east-1:

- Credentials in your terminal whose IAM policy allows:
  - `acm-pca:*`, to create and delete the CAs the tests use.
  - `s3:PutObject` and `s3:PutObjectAcl` on the S3 bucket below.
  - The IAM permissions for the [`secretRef` tests](#secretref-tests).
- An S3 bucket with [Block Public Access disabled](https://docs.aws.amazon.com/AmazonS3/latest/userguide/access-control-block-public-access.html).
  After creating it, run `export OIDC_S3_BUCKET_NAME=<bucket name>`.
- [An AWS IAM OIDC Provider](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_providers_create_oidc.html) with audience `sts.amazonaws.com`.
  Its provider URL is `$OIDC_S3_BUCKET_NAME.s3.us-east-1.amazonaws.com/cluster/$OIDC_CLUSTER_NAME`, where `OIDC_CLUSTER_NAME` defaults to `go env GOARCH`.
  The provider can only be created once the bucket serves `.well-known/openid-configuration`.
  To bootstrap it, run `export OIDC_IAM_ROLE=arn:aws:iam::000000000000:role/oidc-kind-cluster-role`, a placeholder that you replace below, then `make cluster && make install-eks-webhook && make kind-cluster-delete`.
- An IAM role for IRSA, trusted by that OIDC provider for the controller's `ServiceAccount`, `aws-privateca-issuer-${ARCH}-sa`, where `ARCH` is `go env GOARCH`.
  Give it a policy like the one in [IAM policy](docs/iam.md#iam-policy), with `certificate-authority/*` as the resource, because the tests create their CAs.
  The trust policy should look something like:

  ```json
  {
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Principal": { "Federated": "${OIDC_ARN}" },
        "Action": "sts:AssumeRoleWithWebIdentity",
        "Condition": {
          "StringEquals": {
            "${OIDC_URL}:sub": "system:serviceaccount:aws-privateca-issuer:aws-privateca-issuer-${ARCH}-sa"
          }
        }
      }
    ]
  }
  ```

  After creating this role, run `export OIDC_IAM_ROLE=<role ARN>`.

Then:

- `make cluster` recreates the cluster with these environment variables set.
- `make install-eks-webhook` installs a webhook in that kind cluster that enables IRSA, and reinstalls the controller.
- `make e2etest` runs the end-to-end tests against that cluster.
- After you change the controller code, `make upgrade-local && make e2etest` redeploys it and runs the tests again.

IRSA on kind is based on [this blog post](https://reece.tech/posts/oidc-k8s-to-aws/).

#### `secretRef` tests

The suite also tests Issuers that use AWS access keys from a Kubernetes `Secret` (`secretRef`).
Using access keys from a `Secret` is deprecated.
For these tests, it creates an IAM user and an access key, or uses an existing user that you name with `export PLUGIN_USER_NAME_OVERRIDE=<IAM user name>`.
That user needs a policy like the one in [IAM policy](docs/iam.md#iam-policy).
Your credentials need:

- `iam:CreateAccessKey` and `iam:DeleteAccessKey`.
- `iam:CreateUser`, `iam:CreatePolicy` and `iam:AttachUserPolicy`, unless you set `PLUGIN_USER_NAME_OVERRIDE`.

#### Cross-account tests

To also test Issuers that use a CA in another account, you need a role in a separate AWS account that trusts the credentials you run the tests with, and `sts:AssumeRole` on it in your credentials' policy.
The role's policy must allow:

- `acm-pca:*`, to create a CA in that account.
- `ram:GetResourceShareAssociations`, `ram:CreateResourceShare` and `ram:DeleteResourceShare`, to share that CA with your account.

After creating the role, run `export PLUGIN_CROSS_ACCOUNT_ROLE=<role name>`.
Without it, the suite skips the cross-account tests and says so.

### Changing the end-to-end tests

The scenarios are Gherkin feature files in [e2e/features](e2e/features/), run with [godog](https://github.com/cucumber/godog).
Their steps are in `e2e/*_steps.go`, and the suite's setup and shared Issuer and certificate specs are in [e2e/awspcaissuer_test.go](e2e/awspcaissuer_test.go) and [e2e/common_steps.go](e2e/common_steps.go).
The other files in `e2e/` are helpers.

### Other tests

- `make cluster && make install-eks-webhook && make blog-test` checks the workflow in the blog post [Setting up end-to-end TLS encryption on Amazon EKS with the new AWS Load Balancer Controller](https://aws.amazon.com/blogs/containers/setting-up-end-to-end-tls-encryption-on-amazon-eks-with-the-new-aws-load-balancer-controller/).
- `make helm-test` installs the chart the way users do.
  It builds the image, packages the chart as a release would, serves it as a Helm repository, and runs `helm repo add` and `helm install`.
  It then checks that the controller runs the expected image and uninstalls cleanly.
