#This script helps to delete CAs that were created during the e2e tests

if [ "$#" -lt 2 ]; then
    echo "Usage: `basename $0` <region> <ca_arn_to_delete>"
    exit 1
fi

region=$1

aws acm-pca update-certificate-authority \
    --certificate-authority-arn $2 \
    --status "DISABLED" \
    --region $region \
    --endpoint https://acm-pca.$region.amazonaws.com

sleep 2 #ensure CA is in a disabled state before proceeding

aws acm-pca delete-certificate-authority \
    --certificate-authority-arn $2 \
    --region $region \
    --endpoint https://acm-pca.$region.amazonaws.com
