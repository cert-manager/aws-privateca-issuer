#!/usr/bin/env bash

set_variables() {
    HOME_DIR=$(pwd)
    export E2E_DIR="$HOME_DIR/e2e"
    K8S_NAMESPACE="aws-privateca-issuer"
    HELM_CHART_NAME="awspca/aws-privateca-issuer"
    CLUSTER_NAME=pca-external-issuer
    AWS_REGION="us-east-1"
    INTERFACE=$(curl --silent http://169.254.169.254/latest/meta-data/network/interfaces/macs/)
    export SUBNET=$(curl --silent http://169.254.169.254/latest/meta-data/network/interfaces/macs/${INTERFACE}/subnet-id)
    export SECURITY_GROUP_ID=$(curl --silent http://169.254.169.254/latest/meta-data/network/interfaces/macs/${INTERFACE}/security-group-ids)
    export VPC_ID=$(curl --silent http://169.254.169.254/latest/meta-data/network/interfaces/macs/${INTERFACE}/vpc-id)
    export PORT=6443
    tag_subnet
    add_inbound_rule
    create_ca
}

tag_subnet() {
    aws ec2 create-tags --resources $SUBNET --tags Key=kubernetes.io/cluster/$CLUSTER_NAME,Value=shared Key=kubernetes.io/role/elb,Value=1
}

add_inbound_rule() {
    aws ec2 authorize-security-group-ingress --group-id $SECURITY_GROUP_ID --protocol tcp --port $PORT --cidr "0.0.0.0/0" >/dev/null 2>&1
}

create_target_group() {
    TARGET_GROUP_ARN=$(aws elbv2 create-target-group --name blog-test --target-type instance --protocol TCP --port $PORT --vpc-id $VPC_ID | jq -r ".TargetGroups[0].TargetGroupArn")

    aws elbv2 register-targets --target-group-arn $TARGET_GROUP_ARN --targets Id=$(curl --silent http://169.254.169.254/latest/meta-data/instance-id),Port=$PORT

    export LOAD_BALANCER_HOSTNAME=$(kubectl get service nlb-tls-app -ojson | jq -r ".status.loadBalancer.ingress[0].hostname")

    LOAD_BALANCER_NAME=$(cut -d'.' -f1 <<<"$LOAD_BALANCER_HOSTNAME" | sed 's/\(.*\)-/\1\//')

    LOAD_BALANCER_ARN=arn:aws:elasticloadbalancing:$AWS_REGION:$(aws sts get-caller-identity | jq -r ".Account"):loadbalancer/net/$LOAD_BALANCER_NAME

    LISTENER_ARN=$(aws elbv2 describe-listeners --load-balancer-arn $LOAD_BALANCER_ARN | jq -r ".Listeners[0].ListenerArn")

    aws elbv2 modify-listener --listener-arn $LISTENER_ARN --protocol TCP --port $PORT --default-actions Type=forward,TargetGroupArn=$TARGET_GROUP_ARN >/dev/null 2>&1

}

create_ca() {

    export CA_ARN=$(aws acm-pca create-certificate-authority --certificate-authority-configuration file://$E2E_DIR/blog-test/ca_config.json --certificate-authority-type "ROOT" --query 'CertificateAuthorityArn' --output text)

    aws acm-pca wait certificate-authority-csr-created --certificate-authority-arn $CA_ARN

    aws acm-pca get-certificate-authority-csr --certificate-authority-arn $CA_ARN --output text --region us-east-1 >$E2E_DIR/blog-test/ca.csr

    CERTIFICATE_ARN=$(aws acm-pca issue-certificate --certificate-authority-arn $CA_ARN --csr fileb://$E2E_DIR/blog-test/ca.csr --signing-algorithm SHA256WITHRSA --template-arn arn:aws:acm-pca:::template/RootCACertificate/V1 --validity Value=365,Type=DAYS --query 'CertificateArn' --output text)

    aws acm-pca wait certificate-issued --certificate-authority-arn $CA_ARN --certificate-arn $CERTIFICATE_ARN

    aws acm-pca get-certificate --certificate-authority-arn $CA_ARN --certificate-arn $CERTIFICATE_ARN --output text >$E2E_DIR/blog-test/cert.pem

    aws acm-pca import-certificate-authority-certificate --certificate-authority-arn $CA_ARN --certificate fileb://$E2E_DIR/blog-test/cert.pem

}

delete_ca() {

    aws acm-pca update-certificate-authority --certificate-authority-arn $CA_ARN --status "DISABLED"

    aws acm-pca delete-certificate-authority --certificate-authority-arn $CA_ARN --permanent-deletion-time-in-days 7

}

clean_up() {
    set +e

    echo "Cleaning up test resources"

    kubectl delete -f $E2E_DIR/blog-test/test-nlb-tls-app.yaml >/dev/null 2>&1

    kubectl delete -f $E2E_DIR/blog-test/nlb-lab-tls.yaml >/dev/null 2>&1

    kubectl delete -f $E2E_DIR/blog-test/test-cluster-issuer.yaml >/dev/null 2>&1

    helm uninstall aws-load-balancer-controller -n kube-system >/dev/null 2>&1

    delete_ca
}

install_aws_load_balancer() {
    helm repo add eks https://aws.github.io/eks-charts >/dev/null 2>&1
    kubectl apply -k "github.com/aws/eks-charts/stable/aws-load-balancer-controller/crds?ref=master" >/dev/null 2>&1
    helm install aws-load-balancer-controller eks/aws-load-balancer-controller -n kube-system --set clusterName=$CLUSTER_NAME --set env.AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY_ID" --set env.AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY" --set env.AWS_SESSION_TOKEN="$AWS_SESSION_TOKEN" --set env.AWS_REGION="$AWS_REGION" --set enableServiceMutatorWebhook=false>/dev/null 2>&1
    kubectl wait --for=condition=Available --timeout=60s deployments -n kube-system aws-load-balancer-controller 1>/dev/null || exit 1
    echo "AWS Load Balancer installed."
}

main() {

    set_variables

    kubectl wait --for=condition=Available --timeout 60s deployments issuer-aws-privateca-issuer -n $K8S_NAMESPACE 1>/dev/null || exit 1

    echo "issuer-aws-privateca-issuer deployment found."

    POD_NAME=$(kubectl get pods -n $K8S_NAMESPACE -ojson | jq -r ".items[0].metadata.name")

    if [ -z "$POD_NAME" ]; then
        echo "[ERROR] Found empty ACK controller pod name. Exiting ..."
        exit 1
    fi

    echo "$POD_NAME pod found."

    install_aws_load_balancer

    envsubst <$E2E_DIR/blog-test/cluster-issuer.yaml >$E2E_DIR/blog-test/test-cluster-issuer.yaml

    kubectl apply -f $E2E_DIR/blog-test/test-cluster-issuer.yaml 1>/dev/null || exit 1

    kubectl wait --for=condition=Ready --timeout=60s awspcaclusterissuer.awspca.cert-manager.io demo-test-root-ca 1>/dev/null || exit 1

    echo "demo-test-root-ca awspcaclusterissuer found."

    kubectl apply -f $E2E_DIR/blog-test/nlb-lab-tls.yaml 1>/dev/null || exit 1

    kubectl wait --for=condition=Ready --timeout=15s certificates.cert-manager.io nlb-lab-tls-cert 1>/dev/null || exit 1

    echo "nlb-lab-tls-cert certificate found."

    envsubst <$E2E_DIR/blog-test/nlb-tls-app.yaml >$E2E_DIR/blog-test/test-nlb-tls-app.yaml

    kubectl apply -f $E2E_DIR/blog-test/test-nlb-tls-app.yaml 1>/dev/null || exit 1

    kubectl wait --for=condition=Available --timeout 60s deployments nlb-tls-app 1>/dev/null || exit 1

    echo "nlb-tls-app deployment found."

    timeout 60s bash -c 'until kubectl get service/nlb-tls-app --output=jsonpath='{.status.loadBalancer}' | grep "ingress"; do : ; done' 1>/dev/null || exit 1

    echo "Creating target groups"
    
    create_target_group

    echo "Waiting for target groups"

    timeout 600s bash -c 'until echo | openssl s_client -connect $LOAD_BALANCER_HOSTNAME:$PORT; do : ; done' || exit 1

    echo "Blog Test Finished Successfully"
}

trap clean_up EXIT

main
