<!--- app-name: Coredge Cloud Suite-->

# Coredge Cloud Suite (FLUID)
This chart bootstraps the Coredge Cloud Suite on a K8s cluster.
As part of this platform, user will get access to three portals : Auth Portal, Admin Portal and Self-Service Portal

## Prerequisites
- Kubernetes 1.28+
- Helm 3.8.0+
- Storage Class configured

## Basic installation (assuming above pre-requisites are met)
Prepare the helm chart package
```console
git clone https://github.com/coredgeio/ccs-helm-charts.git
cd ccs-helm-charts
make clean
make
```

To install the chart with the release name `ccs`:
```
ReleaseTag=<release build> 
Ex. 2024-04-01

export NODE_IP=$(kubectl get nodes --namespace default -o jsonpath="{.items[0].status.addresses[0].address}")
ExternalIP=$NODE_IP
```
```
kubectl create ns ccs
helm install -f example/example-values.yaml ccs dist/ccs-0.1.0.tgz -n ccs --set=global.releaseTag=$ReleaseTag --set=global.externalIP=$ExternalIP 
```
Note: All Services may take 2-3 mins to come up and running after deployment

## Verify Deployment
```sh
kubectl get pods,svc -n ccs
```

## How to access portal
### Auth Portal:
```sh
export AUTH_HTTP_NODE_PORT=$(kubectl get --namespace ccs -o jsonpath="{.spec.ports[?(@.name=='http')].nodePort}" services keycloak)
echo "Auth Portal can be accessed at - http://${ExternalIP}:${AUTH_HTTP_NODE_PORT}/auth"
```
**Default credentials**:\
*username*: admin\
*password*: admin

### Admin Portal:
```sh
export ADMIN_HTTP_NODE_PORT=$(kubectl get svc admin-portal -n ccs -o jsonpath="{.spec.ports[0].nodePort}")
echo "Admin Portal can be accessed at - http://${ExternalIP}:${ADMIN_HTTP_NODE_PORT}/"
```
**Super user credentials**:\
*username*: ccsadmin\
*password*: Welcome@123

### User Portal:
```sh
export USER_HTTP_NODE_PORT=$(kubectl get --namespace ccs -o jsonpath="{.spec.ports[?(@.name=='http')].nodePort}" services frontend)
echo "User Portal can be accessed at - http://${ExternalIP}:${USER_HTTP_NODE_PORT}/"
```

## Uninstalling the Chart
To uninstall/delete `ccs`:
```console
helm uninstall ccs -n ccs
```
The command removes all the Kubernetes components associated with the chart and deletes the release.
