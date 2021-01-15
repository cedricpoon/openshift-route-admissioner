#!/bin/sh

set -e

usage() {
    cat <<EOF
Patch MutatingWebhookConfiguration with CA Certificate

usage: ${0} [OPTIONS]

The following flags are required.

       --namespace        Namespace where webhook service and secret reside.
       --secret           Secret name for CA certificate and server certificate/key pair.
EOF
    exit 1
}

while [[ $# -gt 0 ]]; do
    case ${1} in
        --secret)
            secret="$2"
            shift
            ;;
        --namespace)
            namespace="$2"
            shift
            ;;
        *)
            usage
            ;;
    esac
    shift
done

[ -z ${secret} ] && secret=route-admissioner-certs
[ -z ${namespace} ] && namespace=default

CA_BUNDLE=$(kubectl get secret ${secret} -n ${namespace} -o json | jq -r '.data."ca-cert.pem"')

kubectl patch mutatingwebhookconfiguration webhook.route-admissioner.k8s.io --patch '{"webhooks": [{"name": "route-admissioner.k8s.io", "clientConfig": {"caBundle": "'$CA_BUNDLE'"}}]}'