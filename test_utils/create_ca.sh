# This script helps create AWS Private CAs to be used for running the e2e tests

if [ "$#" -lt 3 ]; then
    echo "Usage: `basename $0` <region> <ca_config> <ca_cert_signing_algo>"
    exit 1
fi

region=$1

CREATE_CA_RESPONSE=$(aws acm-pca create-certificate-authority \
--certificate-authority-configuration "$2" \
--certificate-authority-type "ROOT" \
--idempotency-token $RANDOM \
--region $region \
--output json \
--endpoint https://acm-pca.$region.amazonaws.com | jq -r '.CertificateAuthorityArn')

CAARN=$CREATE_CA_RESPONSE

sleep 15 # wait for CA activation to finish before proceeding

aws acm-pca get-certificate-authority-csr \
    --certificate-authority-arn $CAARN \
    --output text \
    --region $region \
    --endpoint https://acm-pca.$region.amazonaws.com > ca.csr

ISSUE_CERTIFICATE_RESPONSE=$(aws acm-pca issue-certificate \
    --certificate-authority-arn $CAARN \
    --csr fileb://ca.csr \
    --signing-algorithm "$3" \
    --template-arn arn:aws:acm-pca:::template/RootCACertificate/V1 \
    --validity Value=365,Type="DAYS" \
    --idempotency-token $RANDOM \
    --region $region \
    --output json \
    --endpoint https://acm-pca.$region.amazonaws.com | jq -r '.CertificateArn')

CERTARN=$ISSUE_CERTIFICATE_RESPONSE

sleep 1 #Wait to ensure certificate if issued before proceeding

aws acm-pca get-certificate \
    --certificate-authority-arn $CAARN \
    --certificate-arn $CERTARN \
    --output text \
    --endpoint https://acm-pca.$region.amazonaws.com > cert.pem

aws acm-pca import-certificate-authority-certificate \
    --certificate-authority-arn $CAARN \
    --certificate fileb://cert.pem \
    --endpoint https://acm-pca.$region.amazonaws.com

#output the active CA's ARN
echo $CAARN

#Clean up activation materials
rm ca.csr
rm cert.pem



