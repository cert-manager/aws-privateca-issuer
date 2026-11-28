module github.com/cert-manager/aws-privateca-issuer

go 1.27.3

require (
	github.com/aws/aws-sdk-go-v2 v1.48.2
	github.com/aws/aws-sdk-go-v2/config v1.33.23
	github.com/aws/aws-sdk-go-v2/credentials v1.21.5
	github.com/aws/aws-sdk-go-v2/service/acmpca v1.57.2
	github.com/aws/aws-sdk-go-v2/service/iam v1.68.2
	github.com/aws/aws-sdk-go-v2/service/ram v1.46.3
	github.com/aws/aws-sdk-go-v2/service/sts v1.53.2
	github.com/aws/smithy-go v1.28.11
	github.com/cert-manager/cert-manager v1.21.3
	github.com/cucumber/godog v0.16.0
	github.com/go-logr/logr v1.4.4
	github.com/google/uuid v1.6.0
	github.com/stretchr/testify v1.12.1
	k8s.io/api v0.37.3
	k8s.io/apimachinery v0.37.3
	k8s.io/client-go v0.37.3
	k8s.io/utils v0.0.0-20261023123148-ca04ba58ecab
	sigs.k8s.io/controller-runtime v0.25.3
)

require (
	github.com/Azure/go-ntlmssp v0.1.1 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.20.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.5.14 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.8.14 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.5.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.27 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.14.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.14.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.39.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.44.7 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cucumber/gherkin/go/v42 v42.0.1 // indirect
	github.com/cucumber/messages/go/v34 v34.2.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emicklei/go-restful/v3 v3.14.0 // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/fxamacker/cbor/v2 v2.10.0 // indirect
	github.com/go-asn1-ber/asn1-ber v1.5.8 // indirect
	github.com/go-ldap/ldap/v3 v3.4.15 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-openapi/jsonpointer v1.0.3 // indirect
	github.com/go-openapi/jsonreference v1.0.3 // indirect
	github.com/go-openapi/swag v0.29.2 // indirect
	github.com/go-openapi/swag/cmdutils v0.29.2 // indirect
	github.com/go-openapi/swag/conv v0.29.2 // indirect
	github.com/go-openapi/swag/fileutils v0.29.2 // indirect
	github.com/go-openapi/swag/jsonutils v0.29.2 // indirect
	github.com/go-openapi/swag/loading v0.29.2 // indirect
	github.com/go-openapi/swag/mangling v0.29.2 // indirect
	github.com/go-openapi/swag/netutils v0.29.2 // indirect
	github.com/go-openapi/swag/pools v0.29.2 // indirect
	github.com/go-openapi/swag/stringutils v0.29.2 // indirect
	github.com/go-openapi/swag/typeutils v0.29.2 // indirect
	github.com/go-openapi/swag/yamlutils v0.29.2 // indirect
	github.com/google/gnostic-models v0.7.1 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-memdb v1.3.5 // indirect
	github.com/hashicorp/golang-lru v0.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pandatix/go-abnf v0.5.5 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.25.0 // indirect
	github.com/prometheus/client_model v0.6.3 // indirect
	github.com/prometheus/common v0.74.0 // indirect
	github.com/prometheus/procfs v0.23.0 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.1.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/otel v1.47.0 // indirect
	go.opentelemetry.io/otel/trace v1.47.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	go.yaml.in/yaml/v2 v2.4.5 // indirect
	go.yaml.in/yaml/v3 v3.0.6 // indirect
	golang.org/x/crypto v0.59.0 // indirect
	golang.org/x/net v0.63.0 // indirect
	golang.org/x/oauth2 v0.38.0 // indirect
	golang.org/x/sync v0.25.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/term v0.48.0 // indirect
	golang.org/x/text v0.44.0 // indirect
	golang.org/x/time v0.16.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.5.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/apiextensions-apiserver v0.37.3 // indirect
	k8s.io/component-base v0.37.3 // indirect
	k8s.io/klog/v2 v2.140.0 // indirect
	k8s.io/kube-openapi v0.0.0-20261124193638-7c5f465a2be7 // indirect
	sigs.k8s.io/gateway-api v1.7.0 // indirect
	sigs.k8s.io/json v0.0.0-20260909141634-11ed52e25bc5 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.4.2 // indirect
	sigs.k8s.io/structured-merge-diff/v7 v7.0.0 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)
