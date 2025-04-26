<!--- app-name: OpenFGA --->

# Install Openfga
This chart will install Openfga  

## Prerequisites
- Kubernetes 1.28+
- Helm 3.8.0+
- Storage Class configured

## Installation Procedure
1. Create below values override `openfga-values.yaml`
```console
replicaCount: 1
postgresql:
  enabled: true
  auth:
    postgresPassword: password
    database: postgres
datastore:
  engine: postgres
  uri: "postgres://postgres:password@openfga-postgresql:5432/postgres?sslmode=disable"
authn:
  method:
  preshared:
    keys:
    - MYPreSharedToken1
service:
  type: NodePort
```

2. Install Openfga helm with release name `openfga`:
```
helm repo add openfga https://openfga.github.io/helm-charts
helm install openfga openfga/openfga -n openfga -f openfga-values.yaml --create-namespace
kubectl get all -n openfga
```

3. Create store using fga CLI (`Ubuntu`)
```
wget https://github.com/openfga/cli/releases/download/v0.6.5/fga_0.6.5_linux_amd64.deb
dpkg -i fga_0.6.5_linux_amd64.deb

export NODE_IP=$(kubectl get nodes --namespace default -o jsonpath="{.items[0].status.addresses[0].address}")
export OPENFGA_NODEPORT=$(kubectl get --namespace openfga -o jsonpath="{.spec.ports[?(@.name=='http')].nodePort}" services openfga)

fga store create --name "Fluid Auth Store" --api-url=http://${NODE_IP}:${OPENFGA_NODEPORT}
```
