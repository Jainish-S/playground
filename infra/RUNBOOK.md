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

## Phase 6: Container Registry (OCI)

OCI Container Registry is **free** with 500GB storage and unlimited pulls.

### Step 6.1: Generate Auth Token

1. OCI Console → Profile → User Settings → Auth Tokens
2. Generate Token, name: `docker-registry`
3. **Copy immediately** (shown only once)

### Step 6.2: Docker Login

```bash
# Format: docker login <region>.ocir.io
docker login bom.ocir.io
# Username: <namespace>/oracleidentitycloudservice/<email>
# Password: <auth-token>
```

### Step 6.3: Build & Push Images

```bash
# Standard workflow using the push script
./scripts/oci-push-images.sh

# With specific version tag
./scripts/oci-push-images.sh v1.0.0
```

**Image Naming Convention:**
```
<region>.ocir.io/<namespace>/guardrail/<service>:<tag>

Services:
  - guardrail-server
  - model-prompt-guard
  - model-pii-detect
  - model-hate-detect
  - model-content-class

Tags:
  - latest (default)
  - <git-sha> (auto-added)
  - v1.0.0 (version, optional)
```

> [!NOTE]
> OCI auto-creates public repositories on first push. No terraform needed.

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| kubectl timeout on private IP | Connect Twingate client first |
| ErrImagePull in OCI | Add `docker.io/` prefix to image |
| CreateContainerConfigError | Check secret exists, use `runAsUser: 65532` not `runAsNonRoot` |
| Twingate State: Offline | Check connector tokens, verify network name |
| ImageInspectError | Use full `docker.io/library/image:tag` path |

---

## Phase 7: Deploy Contour Ingress Controller

> [!NOTE]
> Contour uses Envoy proxy with HTTPProxy CRDs for type-safe, validated routing.
> See `docs/CONTOUR_ARCHITECTURE.md` for detailed design rationale.

### Step 7.1: Install Contour (Official Manifests)

```bash
# Install Contour (creates projectcontour namespace)
kubectl apply -f https://projectcontour.io/quickstart/contour.yaml

# Wait for pods
kubectl get pods -n projectcontour -w
# Expected: contour (2 replicas), envoy (DaemonSet)

# Patch Envoy to NodePort (use alternate ports if nginx exists)
kubectl patch svc envoy -n projectcontour -p '{"spec": {"type": "NodePort", "ports": [{"port": 80, "targetPort": 8080, "nodePort": 30081, "name": "http"}, {"port": 443, "targetPort": 8443, "nodePort": 30444, "name": "https"}]}}'
```

### Step 7.2: Verify Contour

```bash
kubectl get pods -n projectcontour
# contour-xxx (2/2 Running)
# envoy-xxx (2/2 Running on each node)

kubectl get svc -n projectcontour
# envoy NodePort 30081/30444
```

---

## Phase 8: Deploy Guardrails Platform

### Step 8.1: Create Namespace and Secrets

```bash
# Apply namespace
kubectl apply -f infra/k8s/guardrails/namespace.yaml

# Create OCI registry secret from docker config
kubectl create secret generic oci-registry-secret \
  --from-file=.dockerconfigjson=$HOME/.docker/config.json \
  --type=kubernetes.io/dockerconfigjson \
  -n guardrails-platform

# Apply configs
kubectl apply -f infra/k8s/guardrails/configs/secrets.yaml
kubectl apply -f infra/k8s/guardrails/configs/configmap.yaml
```

### Step 8.2: Deploy Databases

```bash
kubectl apply -f infra/k8s/guardrails/databases.yaml

# Wait for ready
kubectl wait --for=condition=ready pod -l app=postgres -n guardrails-platform --timeout=120s
kubectl wait --for=condition=ready pod -l app=redis -n guardrails-platform --timeout=60s
```

### Step 8.3: Deploy Application

```bash
# Deploy ML models
kubectl apply -f infra/k8s/guardrails/models/

# Deploy guardrail server
kubectl apply -f infra/k8s/guardrails/guardrail-server/

# Apply HTTPProxy for Contour routing
kubectl apply -f infra/k8s/contour/httpproxy-guardrail.yaml
```

### Step 8.4: Verify Deployment

```bash
# All pods should be Running
kubectl get pods -n guardrails-platform
# Expected: 8 pods (2 guardrail-server, 4 models, 1 postgres, 1 redis)

# Test health endpoint
kubectl port-forward svc/guardrail-server 8000:8000 -n guardrails-platform &
curl http://localhost:8000/v1/health
# {"status":"healthy"}

# Test validation API
curl -X POST http://localhost:8000/v1/validate \
  -H "X-API-Key: test" -H "Content-Type: application/json" \
  -d '{"project_id":"test","text":"Hello","type":"input"}'
```

---

## Access Summary

| Service | Internal Address | NodePort |
|---------|-----------------|----------|
| Contour Envoy HTTP | envoy.projectcontour:80 | 30081 |
| Contour Envoy HTTPS | envoy.projectcontour:443 | 30444 |
| Guardrail API | guardrail-server.guardrails-platform:8000 | via Contour |
| Grafana | grafana.observability:3000 | 30300 |
| Prometheus | prometheus.observability:9090 | - |

