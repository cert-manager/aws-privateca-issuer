module github.com/cert-manager/aws-pca-issuer

go 1.15

require (
	github.com/aws/aws-sdk-go-v2 v1.5.0
	github.com/aws/aws-sdk-go-v2/credentials v1.2.0
	github.com/aws/aws-sdk-go-v2/service/acmpca v1.4.0
	github.com/aws/aws-sdk-go-v2/service/sts v1.4.0
	github.com/go-logr/logr v0.3.0
	github.com/jetstack/cert-manager v1.3.1
	github.com/stretchr/testify v1.6.1
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	sigs.k8s.io/controller-runtime v0.8.3
)
