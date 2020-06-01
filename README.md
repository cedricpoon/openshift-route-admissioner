# [Openshift Route Admissioner](https://github.com/cedricpoon/openshift-route-admissioner)
![Release Charts](https://github.com/cedricpoon/openshift-route-admissioner/workflows/Release%20Charts/badge.svg)
![](https://img.shields.io/docker/cloud/build/cedricpoon/route-admissioner)

Openshift operator for host whitelisting and label assignment on Route.

## Installation
This operator is distributed using **Helm 3**
```sh
helm repo add cedio https://cedricpoon.github.io/openshift-route-admissioner
helm repo update
helm search repo cedio/route-admissioner

helm install route-admissioner cedio/route-admissioner --namespace route-admissioner-operator
```

## Usage
### Domain Whitelisting
The whitelisting guard for `Route` host is applied based on `Namespace` annotation.
```yaml
kind: Namespace
metadata:
  labels:
    route-admissioner/enabled: ''
  annotations:
    route-admissioner/allowed-domain: 'gongfukheunggong.hk,sidoigakming.now'
```
### Route Labeling
Route admissioner uses `Configmap/route-admissioner-label-map` for labelling `Route` which matches the rule set.
```yaml
data:
  key: "route-admissioner/factcheck"
  map: |-
    [
      {
        "domain": "721.nobody",
        "value": "True"
      },
      {
        "domain": "831.massacre",
        "value": "True"
      },
      {
        "domain": "101.gunshot",
        "value": "True"
      }
    ]
```
Resulting object with host `yuenlong.721.nobody` will be
```yaml
kind: Route
metadata:
  labels:
    route-admissioner/factcheck: True
```

## Reference
- banzaicloud/admission-webhook-example, https://github.com/banzaicloud/admission-webhook-example
