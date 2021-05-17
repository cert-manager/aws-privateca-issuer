#Currently test run in us-east-1, this can be changed if needed
REGION=us-east-1

cleanup() {
    rm catemplate
    #rm secret
    ./test_utils/delete_ca.sh $REGION $ECARN
    echo "Deleted CA $ECARN"
    ./test_utils/delete_ca.sh $REGION $RSAARN
    echo "Deleted CA $RSAARN" 
}

report_err() {
    echo "Exited with error on line $1 of e2e_test.sh"
    cleanup
    exit 1
}

decode () {
  echo "$1" | base64 ; echo
}

trap 'report_err $LINENO' ERR

# If CAs arns are passed in via env variables then use those, otherwise create some CAs!
if [[ -z "${RSA_CM_CA_ARN}" ]]; then
  RSAARN=$(./test_utils/create_ca.sh $REGION KeyAlgorithm=RSA_2048,SigningAlgorithm=SHA256WITHRSA,Subject={CommonName=RSA_CM_CA} SHA256WITHRSA)
else
  RSAARN=${RSA_CM_CA_ARN}
fi

if [[ -z "${EC_CM_CA_ARN}" ]]; then
  ECARN=$(./test_utils/create_ca.sh $REGION KeyAlgorithm=EC_prime256v1,SigningAlgorithm=SHA256WITHECDSA,Subject={CommonName=EC_CM_CA} SHA256WITHECDSA)
else
  ECARN=${EC_CM_CA_ARN}
fi

echo "Created RSA CA: $RSAARN"
echo "Created EC CA: $ECARN"

fillInCATemplate() {
    #Escape forward slashes in incoming ARNs
    ESCAPED_ARN=$(printf '%s\n' "$3" | sed -e 's/\//\\&/g');
    cat $1 | sed "s/$2/$ESCAPED_ARN/g" | sed "s/{{REGION}}/$REGION/g" > catemplate
}

#Ensure that the cluster is clean
kubectl delete --all secrets
kubectl delete --all certificates.cert-manager.io
kubectl delete --all certificaterequests.cert-manager.io
kubectl delete --all awspcaclusterissuer.awspca.cert-manager.io
kubectl delete --all awspcaissuer.awspca.cert-manager.io

#Takes AWS creds from env variables
ACCESS_KEY_64=$(printf $AWS_ACCESS_KEY_ID | base64)
SECRET_KEY_64=$(printf $AWS_SECRET_ACCESS_KEY | base64)
cat config/samples/secret.yaml | sed "s/{{BASE64_ACCESSKEY}}/$ACCESS_KEY_64/g" | sed "s/{{BASE64_SECRETKEY}}/$SECRET_KEY_64/g" > secret

kubectl apply --filename secret

#Test EC cert from Cluster issuer
fillInCATemplate config/samples/awspcaclusterissuer_ec/_v1beta1_awspcaclusterissuer_ec.yaml {{EC_CA_ARN}} $ECARN
kubectl apply --filename catemplate
kubectl apply --filename config/samples/awspcaclusterissuer_ec/ec_certificate_awspcaclusterissuer.yaml

kubectl wait --for=condition=Ready --timeout=15s awspcaclusterissuer.awspca.cert-manager.io pca-cluster-issuer-ec
kubectl wait --for=condition=Ready --timeout=15s  certificates.cert-manager.io pca-cluster-issuer-ec-cert

kubectl delete --filename catemplate
kubectl delete --filename config/samples/awspcaclusterissuer_ec/ec_certificate_awspcaclusterissuer.yaml

echo "EC cert created from Cluster issuer"

#Test RSA cert from Cluster issuer
fillInCATemplate config/samples/awspcaclusterissuer_rsa/_v1beta1_awspcaclusterissuer_rsa.yaml {{RSA_CA_ARN}} $RSAARN
kubectl apply --filename catemplate
kubectl apply --filename config/samples/awspcaclusterissuer_rsa/rsa_certificate_awspcaclusterissuer.yaml

kubectl wait --for=condition=Ready --timeout=15s awspcaclusterissuer.awspca.cert-manager.io pca-cluster-issuer-rsa
kubectl wait --for=condition=Ready --timeout=15s  certificates.cert-manager.io pca-cluster-issuer-rsa-cert

kubectl delete --filename catemplate
kubectl delete --filename config/samples/awspcaclusterissuer_rsa/rsa_certificate_awspcaclusterissuer.yaml

echo "RSA cert created from Cluster issuer"

#Test EC cert from Namespaced Issuer
fillInCATemplate config/samples/awspcaissuer_ec/_v1beta1_awspcaissuer_ec.yaml {{EC_CA_ARN}} $ECARN
kubectl apply --filename catemplate
kubectl apply --filename config/samples/awspcaissuer_ec/ec_certificate_awspcaissuer.yaml

kubectl wait --for=condition=Ready --timeout=15s awspcaissuer.awspca.cert-manager.io pca-issuer-ec
kubectl wait --for=condition=Ready --timeout=15s  certificates.cert-manager.io pca-issuer-ec-cert

kubectl delete --filename catemplate
kubectl delete --filename config/samples/awspcaissuer_ec/ec_certificate_awspcaissuer.yaml

echo "EC cert created from Namespaced issuer"

#Test RSA cert from Namespaced Issuer
fillInCATemplate config/samples/awspcaissuer_rsa/_v1beta1_awspcaissuer_rsa.yaml {{RSA_CA_ARN}} $RSAARN
kubectl apply --filename catemplate
kubectl apply --filename config/samples/awspcaissuer_rsa/rsa_certificate_awspcaissuer.yaml

kubectl wait --for=condition=Ready --timeout=15s awspcaissuer.awspca.cert-manager.io pca-issuer-rsa
kubectl wait --for=condition=Ready --timeout=15s  certificates.cert-manager.io pca-issuer-rsa-cert

kubectl delete --filename catemplate
kubectl delete --filename config/samples/awspcaissuer_rsa/rsa_certificate_awspcaissuer.yaml

echo "RSA cert created from Namepsaced issuer"

cleanup