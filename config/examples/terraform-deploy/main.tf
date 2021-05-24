resource "helm_release" "aws-pca-issuer" {
  count            = var.enabled ? 1 : 0
  chart            = var.helm_chart_name
  namespace        = var.k8s_namespace
  create_namespace = var.create_namespace
  name             = var.helm_release_name
  version          = var.helm_chart_version
  repository       = var.helm_repo_url
  force_update     = var.force_update
  recreate_pods    = var.recreate_pods

  set {
    name  = "serviceAccount.annotation"
    value = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/${var.cluster_name}-aws_pca_issuer_role"
  }
}
