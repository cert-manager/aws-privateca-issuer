provider "aws" {
  region = var.region
}

data "aws_eks_cluster" "this" {
  name = var.cluster_name
}

data "aws_eks_cluster_auth" "this" {
  name = var.cluster_name
}

provider "kubernetes" {
  host                   = data.aws_eks_cluster.this.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.this.certificate_authority.0.data)
  token                  = data.aws_eks_cluster_auth.this.token
}

provider "helm" {
  kubernetes {
    host                   = data.aws_eks_cluster.this.endpoint
    token                  = data.aws_eks_cluster_auth.this.token
    cluster_ca_certificate = base64decode(data.aws_eks_cluster.this.certificate_authority.0.data)
  }
}

# Example tests here
# Apply your module more than once to ensure it is reusable without collision of resources names
# Apply it with different variable inputs and usage patterns where possible to test

module "aws-pca-issuer" {
  source              = "../../"
  enabled             = true
  k8s_namespace       = "aws-pca-issuer"
  k8s_serviceaccount  = "aws-pca-issuer" # iam-role-for-serviceaccount
  cluster_name        = var.cluster_name
  cluster_identity_oidc_issuer     = data.aws_eks_cluster.this.identity[0].oidc[0].issuer
  cluster_identity_oidc_issuer_arn = data.aws_eks_cluster.this.arn
}
