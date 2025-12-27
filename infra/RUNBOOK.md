# Infrastructure Runbook

Complete guide to deploy and operate OCI Kubernetes infrastructure from scratch.

## Quick Reference

| Access | Method |
|--------|--------|
| K8s API (kubectl) | Via Twingate → 10.0.1.104:6443 |
| Grafana | Via Twingate → grafana.observability.svc.cluster.local:3000 |
| Prometheus | Via Twingate → prometheus.observability.svc.cluster.local:9090 |

---

## Prerequisites

| Tool | Install | Verify |
|------|---------|--------|
| OCI CLI | `brew install oci-cli` | `oci --version` |
| Terraform | `brew install terraform` | `terraform version` |
| kubectl | `brew install kubectl` | `kubectl version --client` |
| Twingate Client | [twingate.com/download](https://twingate.com/download) | - |

---

## Phase 1: Terraform Infrastructure

```bash
cd infra/terraform/environments/dev

# Copy and configure credentials
cp terraform.tfvars.example terraform.tfvars
# Fill in: tenancy_ocid, user_ocid, fingerprint, private_key_path, 
# region, compartment_ocid, ssh_public_key, twingate_network, twingate_api_token

# Deploy
terraform init
terraform apply
```

---

## Phase 2: Bootstrap - Temporary Public K8s Access

> [!IMPORTANT]
> Initially, the K8s API is not publicly accessible. To deploy Twingate connectors,
> you need temporary public access, then remove it after connectors are online.

### Step 2.1: Add Temporary 6443 Rule

Add to `modules/network/main.tf` security list:
```hcl
# TEMPORARY - REMOVE AFTER TWINGATE DEPLOYED
ingress_security_rules {
  protocol = "6"
  source   = "0.0.0.0/0"
  tcp_options { min = 6443; max = 6443 }
}
```

Apply: `terraform apply`

### Step 2.2: Configure kubeconfig with Public IP

```bash
# Get public IP from terraform output or OCI console
oci ce cluster create-kubeconfig --cluster-id <CLUSTER_ID> ...

# Edit kubeconfig to use public IP temporarily
sed -i '' 's|https://10.0.1.104:6443|https://PUBLIC_IP:6443|g' ~/.kube/config

# Test
kubectl get nodes
```

---

## Phase 3: Deploy Twingate Connector

### Step 3.1: Create K8s Secrets from Terraform Tokens

```bash
cd infra/terraform/environments/dev

TW_NETWORK=$(grep twingate_network terraform.tfvars | sed "s/.*= *\"//" | sed "s/\".*//")

# Primary connector secret
TW_ACCESS=$(terraform output -json twingate_primary_tokens | jq -r '.access_token')
TW_REFRESH=$(terraform output -json twingate_primary_tokens | jq -r '.refresh_token')

kubectl create secret generic twingate-primary-tokens \
  --namespace twingate \
  --from-literal=TWINGATE_NETWORK="$TW_NETWORK" \
  --from-literal=TWINGATE_ACCESS_TOKEN="$TW_ACCESS" \
  --from-literal=TWINGATE_REFRESH_TOKEN="$TW_REFRESH"

# Secondary connector secret
TW_ACCESS=$(terraform output -json twingate_secondary_tokens | jq -r '.access_token')
TW_REFRESH=$(terraform output -json twingate_secondary_tokens | jq -r '.refresh_token')

kubectl create secret generic twingate-secondary-tokens \
  --namespace twingate \
  --from-literal=TWINGATE_NETWORK="$TW_NETWORK" \
  --from-literal=TWINGATE_ACCESS_TOKEN="$TW_ACCESS" \
  --from-literal=TWINGATE_REFRESH_TOKEN="$TW_REFRESH"
```

### Step 3.2: Deploy Connector Pods (Zero-Downtime)

> [!CAUTION]
> Deploy secondary FIRST, wait for it to come online, then deploy primary.

```bash
# Deploy secondary first
kubectl apply -f infra/k8s/twingate/secondary-connector.yaml
kubectl wait --for=condition=available deployment/twingate-connector-secondary -n twingate --timeout=120s

# Verify secondary is online
kubectl logs -n twingate -l app=twingate-connector-secondary --tail=5
# Should show: State: Online

# Now safe to deploy primary
kubectl apply -f infra/k8s/twingate/primary-connector.yaml

# Clean up old deployment if exists
kubectl delete deployment twingate-connector -n twingate --ignore-not-found

# Clean up old namespace if exists
kubectl delete namespace twingate-system --ignore-not-found

# Verify both pods
kubectl get pods -n twingate
kubectl logs -n twingate -l connector=primary --tail=3
kubectl logs -n twingate -l connector=secondary --tail=3
# Both should show: State: Online
```

---

## Phase 4: Secure - Remove Public Access

### Step 4.1: Switch kubeconfig to Private IP

```bash
sed -i '' 's|https://PUBLIC_IP:6443|https://10.0.1.104:6443|g' ~/.kube/config
```

### Step 4.2: Remove Temporary 6443 Rule

Remove the temp rule from `modules/network/main.tf` and apply:
```bash
terraform apply
```

### Step 4.3: Connect via Twingate

1. Open Twingate Client
2. Connect to your network
3. Test: `kubectl get nodes`

---

## Phase 5: Deploy Observability

```bash
# Create Grafana secret
kubectl create secret generic grafana-credentials \
  --namespace observability \
  --from-literal=admin-user='admin' \
  --from-literal=admin-password="$(openssl rand -base64 16)"

# Deploy
kubectl apply -k infra/k8s/observability/
```

---

## Access Services (via Twingate)

| Service | Address |
|---------|---------|
| Grafana | http://grafana.observability.svc.cluster.local:3000 |
| Prometheus | http://prometheus.observability.svc.cluster.local:9090 |
| K8s API | https://10.0.1.104:6443 |

**Grafana credentials:**
```bash
echo "User: admin"
kubectl get secret grafana-credentials -n observability -o jsonpath='{.data.admin-password}' | base64 -d
```

---

## Verification

```bash
kubectl get nodes                    # Should work via Twingate
kubectl get pods -n observability    # All Running
kubectl get pods -n twingate         # 2 pods: connector-primary, connector-secondary
kubectl logs -n twingate -l connector=primary --tail=2   # State: Online
kubectl logs -n twingate -l connector=secondary --tail=2 # State: Online
```

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| kubectl timeout on private IP | Connect Twingate client first |
| ErrImagePull in OCI | Add `docker.io/` prefix to image |
| CreateContainerConfigError | Check secret exists, use `runAsUser: 65532` not `runAsNonRoot` |
| Twingate State: Offline | Check connector tokens, verify network name |
