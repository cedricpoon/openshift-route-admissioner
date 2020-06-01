#!/bin/bash

secret=route-admissioner-certs

{ # if cert.pem exists and not expired

  kubectl get secret ${secret} -n ${namespace} -o json | \
    jq -r '.data."cert.pem"' | \
    base64 -d | \
    openssl x509 -checkend 0

} || { # create certificates

  ./prepare-certificates.sh --namespace $namespace --days 3650 --secret $secret

  ./patch-webhook.sh --namespace $namespace --secret $secret
}