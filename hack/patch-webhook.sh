#!/bin/bash

CA_BUNDLE=$(kubectl get configmap client-ca -n openshift-kube-apiserver -o json | jq -r '.data."ca-bundle.crt"' | base64 | tr -d '\n')

kubectl patch mutatingwebhookconfiguration webhook.route-admissioner.k8s.io --patch '{"webhooks": [{"name": "route-admissioner.k8s.io", "clientConfig": {"caBundle": "'$CA_BUNDLE'"}}]}'