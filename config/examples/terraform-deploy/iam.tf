data "aws_caller_identity" "current" {}

locals {
  oidc_fully_qualified_subjects          = format("system:serviceaccount:%s:%s", var.k8s_namespace, var.k8s_serviceaccount)
  assume_role_policy_Principal_federated = format("%s/%s", "arn:aws:iam::${data.aws_caller_identity.current.account_id}:oidc-provider", trimprefix(var.cluster_identity_oidc_issuer, "https://"))
}

# Policy
data "aws_iam_policy_document" "aws_pca_issuer" {
  statement {
    sid = "awspcaissuer"

    actions = [
      "acm-pca:*",
    ]

    resources = [
      "*", # TODO: replace with ca arn
    ]

    effect = "Allow"
  }

}

resource "aws_iam_policy" "aws_pca_issuer" {
  name        = "${var.cluster_name}-aws-pca-issuer"
  path        = "/"
  description = "Policy for aws-pca-issuer"

  policy = data.aws_iam_policy_document.aws_pca_issuer.json
}

# Role
resource "aws_iam_role" "aws_pca_issuer_role" {
  name = "${var.cluster_name}-aws_pca_issuer_role"
  assume_role_policy = jsonencode({
    Statement = [{
      Action = "sts:AssumeRoleWithWebIdentity"
      Effect = "Allow"
      Principal = {
        Federated = local.assume_role_policy_Principal_federated
      }
      Condition = {
        StringEquals = {
          format("%s:sub", trimprefix(var.cluster_identity_oidc_issuer, "https://")) = local.oidc_fully_qualified_subjects
        }
      }
    }]
    Version = "2012-10-17"
  })

  force_detach_policies = true
  max_session_duration  = 3600
}

resource "aws_iam_role_policy_attachment" "aws_pca_issuer" {
  role       = aws_iam_role.aws_pca_issuer_role.name
  policy_arn = aws_iam_policy.aws_pca_issuer.arn
}
