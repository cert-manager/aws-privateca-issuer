module github.com/cert-manager/aws-privateca-issuer

go 1.15

require (
	github.com/aws/aws-sdk-go v1.40.54 // indirect
	github.com/aws/aws-sdk-go-v2 v1.9.1
	github.com/aws/aws-sdk-go-v2/config v1.5.0
	github.com/aws/aws-sdk-go-v2/credentials v1.3.1
	github.com/aws/aws-sdk-go-v2/service/acmpca v1.4.0
	github.com/aws/aws-sdk-go-v2/service/iam v1.10.0
	github.com/aws/aws-sdk-go-v2/service/ram v1.7.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.6.0
	github.com/go-logr/logr v0.3.0
	github.com/jetstack/cert-manager v1.3.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e // indirect
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/utils v0.0.0-20210111153108-fddb29f9d009
	sigs.k8s.io/controller-runtime v0.8.3
)
