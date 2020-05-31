#!/bin/bash

./create-signed-cert.sh --namespace $NAMESPACE

./patch-webhook.sh

./route-admissioner -logtostderr