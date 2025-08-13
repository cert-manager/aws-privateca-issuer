set -euo pipefail

CA_ARN=$(aws ssm  get-parameter --name /iamra/certificate-authority-arn | jq -r '.Parameter.Value')
TRUST_ANCHOR_ARN=$(aws ssm  get-parameter --name /iamra/trust-anchor-arn | jq -r '.Parameter.Value')
PROFILE_ARN=$(aws ssm  get-parameter --name /iamra/profile-arn | jq -r '.Parameter.Value')
ROLE_ARN=$(aws ssm  get-parameter --name /iamra/role-arn | jq -r '.Parameter.Value')

openssl req -out iamra.csr -new -newkey rsa:2048 -nodes -keyout iamra.key -subj "/CN=iamra-issuer"

CERT_ARN=$(aws acm-pca issue-certificate \
      --certificate-authority-arn $CA_ARN \
      --csr fileb://iamra.csr \
      --signing-algorithm "SHA256WITHRSA" \
      --validity Value=1,Type="DAYS" | jq -r .CertificateArn)

aws acm-pca get-certificate \
      --certificate-authority-arn $CA_ARN \
      --certificate-arn $CERT_ARN | \
      jq -r .Certificate > iamra-cert.pem

PROFILE_ARN=$PROFILE_ARN ROLE_ARN=$ROLE_ARN TRUST_ANCHOR_ARN=$TRUST_ANCHOR_ARN envsubst <e2e/iamra-test/iamra-values.yaml >replaced-values.yaml

make manager
make create-local-registry
make kind-cluster
make deploy-cert-manager
make docker-build
make docker-push-local

kubectl create secret tls -n aws-privateca-issuer cert --cert=iamra-cert.pem --key=iamra.key

sleep 15

helm install issuer ./charts/aws-pca-issuer -f replaced-values.yaml -n aws-privateca-issuer
