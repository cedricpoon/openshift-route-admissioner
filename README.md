# [Openshift Route Admissioner](https://github.com/cedricpoon/openshift-route-admissioner)
![Release Charts](https://github.com/cedricpoon/openshift-route-admissioner/workflows/Release%20Charts/badge.svg)
[![](https://img.shields.io/docker/cloud/build/cedricpoon/route-admissioner)](https://hub.docker.com/repository/docker/cedricpoon/route-admissioner)

Openshift operator for host whitelisting and label assignment on Route.

## Environment
- OpenShift 4.5.4 (Kubernetes v1.18.3+012b3ec)
- OKD 4.5.0-0.okd-2020-07-14-153706 (Kubernetes v1.18.3)

## Installation
This operator is distributed using **Helm 3**
```sh
helm repo add cedio https://cedricpoon.github.io/openshift-route-admissioner
helm repo update
helm search repo cedio/route-admissioner

helm install route-admissioner cedio/route-admissioner --namespace route-admissioner-operator
```

### High Availability
You can set `Pod Count` for `Deployment Configs` to the size of nodes in cluster.

## Usage
### Domain Whitelisting
The whitelisting guard for `Route` host is applied based on `Namespace` annotation.
```yaml
kind: Namespace
metadata:
  labels:
    route-admissioner/enabled: ''
  annotations:
    route-admissioner/allowed-domain: 'hidden.hk,hidden.now'
```
### Route Labeling
Route admissioner uses `Configmap/route-admissioner-label-map` for labelling `Route` which matches the rule set.
```yaml
data:
  key: "route-admissioner/hide"
  map: |-
    [
      {
        "domain": "hidden.hk",
        "value": "True"
      },
      {
        "domain": "hidden.now",
        "value": "True"
      },
      {
        "domain": "hidden.com",
        "value": "True"
      }
    ]
```
Resulting object with host `hidden.hidden.hk` will be
```yaml
kind: Route
metadata:
  labels:
    route-admissioner/hide: True
```

## Reference
- banzaicloud/admission-webhook-example, https://github.com/banzaicloud/admission-webhook-example
