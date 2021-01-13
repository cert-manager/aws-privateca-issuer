module github.com/jniebuhr/aws-pca-issuer

go 1.15

require (
	github.com/aws/aws-sdk-go v1.36.25
	github.com/go-logr/logr v0.3.0
	github.com/jetstack/cert-manager v1.1.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	k8s.io/utils v0.0.0-20200912215256-4140de9c8800
	sigs.k8s.io/controller-runtime v0.7.0
)
