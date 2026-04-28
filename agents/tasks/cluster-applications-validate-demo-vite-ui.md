# Task: Validate demo-vite-ui Application

> **Context**: Read `agents/context.md` first.

## Objective

Confirm the `demo-vite-ui` app is correctly deployed via ArgoCD, reachable through Envoy
Gateway, and accessible from a browser on the host Mac.

## Prerequisites

- Lima VM running (`limactl list` shows `k8s-lab` as `Running`)
- SSH tunnel active: `ssh -F ~/.lima/k8s-lab/ssh.config -N -L 6443:127.0.0.1:6443 lima-k8s-lab`
- `export KUBECONFIG=~/.kube/lima-k8s-lab`

---

## Tasks

### 1. Validate ArgoCD sync

```bash
kubectl get application demo-vite-ui-k8s-lab -n argocd \
  -o custom-columns="NAME:.metadata.name,SYNC:.status.sync.status,HEALTH:.status.health.status"
```

**Expected:** `Synced   Healthy`

If `OutOfSync`, force a refresh:
```bash
kubectl annotate application demo-vite-ui-k8s-lab -n argocd \
  argocd.argoproj.io/refresh=normal --overwrite
```

### 2. Validate pod and image

```bash
kubectl get pods -n envoy-gateway-system -l app=demo-vite-ui
kubectl get deploy demo-vite-ui -n envoy-gateway-system \
  -o jsonpath='image: {.spec.template.spec.containers[0].image}{"\n"}'
```

**Expected:**
- Pod: `1/1 Running`
- Image: `ghcr.io/rayabueg/demo-vite-ui:0.2.0` (or the current pinned tag in `cluster-applications/apps/demo-vite-ui.yaml`)

### 3. Validate HTTPRoute

```bash
kubectl get httproute demo-vite-ui -n envoy-gateway-system
```

**Expected:** Shows `AGE` with no errors. To see full accepted status:
```bash
kubectl get httproute demo-vite-ui -n envoy-gateway-system \
  -o jsonpath='{.status.parents[0].conditions[*].type}: {.status.parents[0].conditions[*].status}{"\n"}'
# Expected: Accepted ResolvedRefs: True True
```

### 4. Smoke test the route (from inside the VM)

The Gateway IP (`192.168.5.15`) is on the Lima vmnet interface and is **not directly
reachable from the Mac**. Always test routes via `limactl shell` or the port-forward
(Step 5).

```bash
limactl shell k8s-lab curl -s -o /dev/null -w "/vite/: %{http_code}\n" \
  http://192.168.5.15:30080/vite/
# Expected: /vite/: 200
```

### 5. Open a port-forward for browser access

The Mac browser **cannot** reach `192.168.5.15` directly. Port-forward the Envoy Gateway
listener service to `localhost:9080`:

```bash
# Detect the Envoy listener service name (it includes a hash)
ENVOY_SVC=$(kubectl get svc -n envoy-gateway-system \
  -l gateway.envoyproxy.io/owning-gateway-name=eg -o name | head -1)
echo "Service: $ENVOY_SVC"

# Start the port-forward (run in background or a dedicated terminal)
kubectl port-forward $ENVOY_SVC -n envoy-gateway-system 9080:80
```

Verify the port-forward is working from the Mac:
```bash
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:9080/vite/
# Expected: 200
```

### 6. Browser validation

Open in a Mac browser:

```
http://localhost:9080/vite/
```

**Expected:** A white page with:
- Title: **k8s-lab UI (Vite + React)**
- A "Loaded at" timestamp
- A "Clicks" counter with a working button

> **Gray screen / assets pending?**  The browser must access the app via `localhost:9080`
> (the port-forward), not `192.168.5.15:30080`. The Lima vmnet IP is only routable from
> inside the VM. If you use the VM IP directly in the browser, the HTML loads (returned
> by the first GET) but subsequent asset requests to `192.168.5.15` hang, producing a
> gray screen with pending requests in DevTools.

---

## Expected Final State

| Check | Expected |
|-------|----------|
| ArgoCD app | `Synced Healthy` |
| Pod | `1/1 Running` |
| HTTPRoute | `Accepted` |
| `/vite/` via `limactl` curl | HTTP 200 |
| `/vite/` via `localhost:9080` | HTTP 200 |
| Browser | Renders UI with counter |
