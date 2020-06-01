#!/bin/bash

set -e

usage() {
    cat <<EOF
Prepare CA certificates and server key/certificate with creating secrets

usage: ${0} [OPTIONS]

The following flags are required.

       --service          Service name of webhook.
       --namespace        Namespace where webhook service and secret reside.
       --secret           Secret name for CA certificate and server certificate/key pair.
       --days             Duration for days of CA keystore and server certificates
EOF
    exit 1
}

while [[ $# -gt 0 ]]; do
    case ${1} in
        --service)
            service="$2"
            shift
            ;;
        --secret)
            secret="$2"
            shift
            ;;
        --namespace)
            namespace="$2"
            shift
            ;;
        --days)
            days="$2"
            shift
            ;;
        *)
            usage
            ;;
    esac
    shift
done

[ -z ${service} ] && service=route-admissioner-svc
[ -z ${secret} ] && secret=route-admissioner-certs
[ -z ${namespace} ] && namespace=default
[ -z ${days} ] && days=365

if [ ! -x "$(command -v openssl)" ]; then
    echo "openssl not found"
    exit 1
fi

tmpdir=$(mktemp -d)
echo "creating certs in tmpdir ${tmpdir} "

# CA
openssl genrsa -out ${tmpdir}/ca.key 2048
openssl req -x509 -new -nodes -key ${tmpdir}/ca.key -sha256 -days ${days} -out ${tmpdir}/ca.crt -subj "/C=HK/CN=route-admissioner"

# Server
openssl genrsa -out ${tmpdir}/server.key 2048
openssl req -new -sha256 -key ${tmpdir}/server.key -subj "/C=HK/CN=${service}.${namespace}.svc" -out ${tmpdir}/server.csr
openssl x509 -req -in ${tmpdir}/server.csr -CA ${tmpdir}/ca.crt -CAkey ${tmpdir}/ca.key -CAcreateserial -out ${tmpdir}/server.crt -days ${days} -sha256

# create the secret with CA cert and server cert/key
kubectl create secret generic ${secret} \
        --from-file=key.pem=${tmpdir}/server.key \
        --from-file=cert.pem=${tmpdir}/server.crt \
        --from-file=ca-key.pem=${tmpdir}/ca.key \
        --from-file=ca-cert.pem=${tmpdir}/ca.crt \
        --dry-run -o yaml |
    kubectl -n ${namespace} apply -f -