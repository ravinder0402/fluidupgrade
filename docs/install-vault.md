<!--- app-name: Vault --->

# Install Vault to Enable SSL certificates management
This chart will install Vault in Non-HA mode  

## Prerequisites
- Kubernetes 1.28+
- Helm 3.8.0+
- Storage Class configured

## Installation Procedure
1. Create below values override `vault-values.yaml`
```console
ui:
  enabled: true
  serviceType: "NodePort"
  serviceNodePort: 31333
ha:
  enabled: false
```

2. Install Vault helm with release name `vault`:
```
helm repo add hashicorp https://helm.releases.hashicorp.com 
helm install -f vault-values.yaml vault hashicorp/vault -n vault --create-namespace
kubectl get all -n vault
```

3. Init Vault and Unseal
Store below generated Unseal Keys and Root Token safely
```
kubectl exec -it vault-0  sh -n vault
vault operator init
```

Run 3 times with distinct Unseal Keys from above 
```
vault operator unseal
```


