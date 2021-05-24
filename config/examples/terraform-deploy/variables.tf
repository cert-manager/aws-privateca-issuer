variable "cluster_name" {
  type        = string
  description = "The name of the cluster"
}

variable "cluster_identity_oidc_issuer" {
  type        = string
  description = "The OIDC Identity issuer for the cluster"
}

variable "cluster_identity_oidc_issuer_arn" {
  type        = string
  description = "The OIDC Identity issuer ARN for the cluster that can be used to associate IAM roles with a service account"
}

# cluster-autoscaler

variable "enabled" {
  type        = bool
  default     = true
  description = "Variable indicating whether deployment is enabled"
}

# Helm

variable "helm_chart_name" {
  type        = string
  default     = "aws-pca-issuer"
  description = "Helm chart name to be installed"
}

variable "helm_chart_version" {
  type        = string
  default     = "0.1.0"
  description = "Version of the Helm chart"
}

variable "helm_release_name" {
  type        = string
  default     = "aws-pca-issuer"
  description = "Helm release name"
}

variable "helm_repo_url" {
  type        = string
  default     = "https://jniebuhr.github.io/aws-pca-issuer/"
  description = "Helm repository"
}

variable "create_namespace" {
  type        = bool
  default     = true
  description = "Have helm_resource create the namespace, default true"
}

variable "force_update" {
  type        = bool
  default     = false
  description = "(Optional) Force resource update through delete/recreate if needed. Defaults to false"
}

variable "recreate_pods" {
  type        = bool
  default     = false
  description = "(Optional) Perform pods restart during upgrade/rollback. Defaults to false."
}

# K8s

variable "k8s_namespace" {
  type        = string
  default     = "default"
  description = "The K8s namespace in which to install the Helm chart, default: 'default'"
}

variable "k8s_serviceaccount" {
  type        = string
  default     = "default"
  description = "The K8s service account to be created"
}
