#!/bin/sh

secret=route-admissioner-certs

{ # if cert.pem exists and not expiring soon (i.e. within 1 month)

  kubectl get secret ${secret} -n ${namespace} -o json | \
    jq -r '.data."cert.pem"' | \
    base64 -d | \
    openssl x509 -checkend 2678400

} || { # create certificates

  ./prepare-certificates.sh --namespace $namespace --days 3650 --secret $secret

  ./patch-webhook.sh --namespace $namespace --secret $secret
}