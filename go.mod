module github.com/jniebuhr/aws-pca-issuer

go 1.15

require (
	github.com/aws/aws-sdk-go v1.36.25
	github.com/go-logr/logr v0.3.0
	github.com/jetstack/cert-manager v1.3.1
	github.com/stretchr/testify v1.6.1 // indirect
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	sigs.k8s.io/controller-runtime v0.8.3
)
