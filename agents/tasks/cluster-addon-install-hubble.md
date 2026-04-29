# Task: Install Hubble Observability

> **Context**: Read `agents/context.md` first.

## Objective

Enable Hubble on the existing Cilium installation to get a real-time network flow
visualizer (service map + flow explorer) using ArgoCD GitOps. Hubble is enabled by
upgrading the Cilium Helm release with additional values — it does **not** require a
separate Helm chart or a new ArgoCD Application project.

## Background

Cilium includes Hubble as an optional sub-component. Enabling it upgrades the existing
`cilium` Helm release in-place. Three components are added:

| Component | Description |
|-----------|-------------|
| `hubble-relay` | Deployment — aggregates flow data from all nodes via gRPC |
| `hubble-ui` | Deployment (2 containers: nginx frontend + Go backend) |
| `hubble-ui` Service | ClusterIP, port 80 → targetPort 8081 (nginx) |

## Architecture: Gateway Access

Hubble UI's HTML contains `<base href="/">` — all JS/CSS assets are loaded from `/`
(root). This prevents subpath routing via URL rewrite (e.g. `/hubble`) because asset
requests bypass the route prefix entirely and hit the wrong backend.

**Solution**: Add a dedicated listener on the `eg` Gateway (port 8080). The HTTPRoute
matches `/` on that listener, so Hubble UI owns the entire port with no path conflict.

```
Browser (Mac) → localhost:12000
  → SSH tunnel → VM NodePort 32091 → envoy-gateway-system / envoy-envoy-gateway-system-eg-...  :8080
    → hubble listener → HTTPRoute (/ → hubble-ui:80)
      → hubble-ui pod (nginx :8081)
```

---

## Prerequisites

- Lima VM running (`limactl list` shows `k8s-lab` as `Running`)
- SSH tunnel active: `ssh -F ~/.lima/k8s-lab/ssh.config -N -L 6443:127.0.0.1:6443 lima-k8s-lab`
- `export KUBECONFIG=~/.kube/lima-k8s-lab`
- Cilium already installed and healthy (check: `kubectl get pods -n kube-system -l k8s-app=cilium`)
- ArgoCD logged in: `argocd login localhost:8080 --username admin --password <password> --insecure`

---

## Files Created

### `cluster-addons/clusters/k8s-lab/addons/hubble/application.yaml`

ArgoCD Application that upgrades the `cilium` Helm release to enable Hubble. Key points:
- `releaseName: cilium` — targets the existing release, not a new one
- All original Cilium values are preserved to prevent drift
- Hubble values are appended: `hubble.enabled`, `hubble.relay.enabled`, `hubble.ui.enabled`
- `ServerSideApply: true` — required for Cilium CRD management
- `ignoreDifferences` on DaemonSet/Deployment — Cilium operator injects runtime fields

### `cluster-addons/clusters/k8s-lab/addons/hubble/kustomization.yaml`

References `application.yaml` and `httproute.yaml`.

### `cluster-addons/clusters/k8s-lab/addons/hubble/httproute.yaml`

HTTPRoute that binds to the `hubble` listener (`sectionName: hubble`) on the `eg` Gateway.
Matches `/` with no URL rewrite.

### `cluster-addons/clusters/k8s-lab/addons/envoy-gateway/gateway/eg-gateway.yaml`

Added a second listener:
```yaml
- name: hubble
  protocol: HTTP
  port: 8080
  allowedRoutes:
    namespaces:
      from: All
```

---

## Steps

### 1. Verify the addon directory is discovered by ArgoCD

The `cluster-addons-k8s-lab` ApplicationSet auto-discovers `clusters/k8s-lab/addons/*`.
After pushing the `hubble/` directory, ArgoCD may cache the directory listing.
If the `hubble` Application does not appear within ~2 minutes, restart the repo-server:

```bash
kubectl rollout restart deployment argocd-repo-server -n argocd
kubectl rollout status deployment argocd-repo-server -n argocd
```

Then check:
```bash
kubectl get application hubble-k8s-lab -n argocd
```

### 2. Sync the hubble addon

```bash
argocd app sync hubble-k8s-lab --insecure
```

Wait for the Cilium pods to restart (the DaemonSet rolls out new pods):
```bash
kubectl rollout status daemonset cilium -n kube-system --timeout=120s
kubectl rollout status deployment hubble-relay -n kube-system --timeout=60s
kubectl rollout status deployment hubble-ui -n kube-system --timeout=60s
```

### 3. Sync the envoy-gateway addon (for the new listener)

```bash
argocd app sync envoy-gateway-k8s-lab --insecure
```

Verify the Gateway has both listeners:
```bash
kubectl get gateway eg -n envoy-gateway-system \
  -o jsonpath='{range .spec.listeners[*]}{.name}: {.port}{"\n"}{end}'
# Expected:
# http: 80
# hubble: 8080
```

Verify the NodePort for 8080:
```bash
kubectl get svc -n envoy-gateway-system | grep envoy-envoy
# Expected: 80:30080/TCP,8080:<nodeport>/TCP
```

### 4. Sync the hubble HTTPRoute

```bash
argocd app sync hubble-k8s-lab --insecure
```

Verify the route is accepted:
```bash
kubectl get httproute hubble-ui -n kube-system \
  -o jsonpath='{.status.parents[0].conditions[*].type}: {.status.parents[0].conditions[*].status}{"\n"}'
# Expected: Accepted ResolvedRefs: True True
```

---

## Expected Final State

| Check | Expected |
|-------|----------|
| `hubble-k8s-lab` ArgoCD app | `Synced Healthy` |
| `hubble-relay` pod | `1/1 Running` |
| `hubble-ui` pod | `2/2 Running` |
| Gateway listeners | `http: 80`, `hubble: 8080` |
| HTTPRoute `hubble-ui` | `Accepted` |

Validate with the companion task: `cluster-addon-validate-hubble.md`
