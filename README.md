# pihole-operator

<img src="docs/pihole.svg" width="256px"/>

## Description

PiHole Operator is the solution to the problem of wanting a highly available 
replicated pihole deployment. It does this by making configuration of pihole
done exclusively through kubernetes resource definitions. 

## Installation

This operator, similar to many others, relies on OLM for installation, and 
distribution. https://sdk.operatorframework.io/docs/olm-integration/tutorial-bundle/#enabling-olm

```
# install operator-sdk
wget https://github.com/operator-framework/operator-sdk/releases/download/v1.33.0/operator-sdk_linux_amd64
mv ./operator-sdk_linux_amd64 ./operator-sdk
sudo install ./operator-sdk /usr/local/bin


# install the pihole-operator
operator-sdk olm install
operator-sdk olm status
operator-sdk run bundle ghcr.io/robbert229/pihole-operator/pihole-operator-bundle:v0.0.1
```

### Updating

```
# update the pihole-operator
operator-sdk run bundle-upgrade ghcr.io/robbert229/pihole-operator/pihole-operator-bundle:v0.0.X
```

## Usage

```yaml
apiVersion: pihole.lab.johnrowley.co/v1alpha1
kind: PiHole
metadata:
  name: pihole-sample
spec:
  # how many replicas should exist in the replica set.
  replicas: 3 
  dns:
    # what upstream servers should the pihole use.
    upstreamServers:
    - "8.8.8.8"
    - "8.8.4.4"
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
  # dnsRecordSelector is a label selector that is used to ensure that the 
  # pihole instance will only pickup configuration that match the given 
  # selector
  dnsRecordSelector: {}

  # dnsRecordNamespaceSelector is a label selector that is used to filter
  # which namespaces the pihole-operator should operate in.
  dnsRecordNamespaceSelector: {}
---
# here we configure the pihole to have a dns record installed.
apiVersion: pihole.lab.johnrowley.co/v1alpha1
kind: DnsRecord
metadata:
  name: dnsrecord-cname-sample
spec:
  # cname allows us to set a cname record.
  cname:
    domain: "pihole.lab.johnrowley.co"
    target: "pihole.infra.lab.johnrowley.co"
---
# here we configure the pihole to have a dns record installed.
apiVersion: pihole.lab.johnrowley.co/v1alpha1
kind: DnsRecord
metadata:
  name: dnsrecord-a-sample
spec:
  # a allows us to set an a record.
  a:
    domain: "pihole.lab.johnrowley.co"
    ip: "123.123.123.123"
```

## Attribution

The code for interacting with the pihole api was sourced from [Ryan Wholey's excellent terraform provider](https://github.com/ryanwholey/terraform-provider-pihole).