# Task: Validate Hubble Observability

> **Context**: Read `agents/context.md` first.

## Objective

Confirm Hubble is correctly installed, all components are healthy, and the Hubble UI
is accessible from a browser on the host Mac through the Envoy Gateway.

## Prerequisites

- Lima VM running (`limactl list` shows `k8s-lab` as `Running`)
- SSH tunnel active: `ssh -F ~/.lima/k8s-lab/ssh.config -N -L 6443:127.0.0.1:6443 lima-k8s-lab`
- `export KUBECONFIG=~/.kube/lima-k8s-lab`

---

## Tasks

### 1. Validate ArgoCD sync

```bash
kubectl get application hubble-k8s-lab -n argocd \
  -o custom-columns="NAME:.metadata.name,SYNC:.status.sync.status,HEALTH:.status.health.status"
```

**Expected:** `Synced   Healthy`

If `OutOfSync`, force a refresh:
```bash
kubectl annotate application hubble-k8s-lab -n argocd \
  argocd.argoproj.io/refresh=normal --overwrite
```

### 2. Validate Hubble pods

```bash
kubectl get pods -n kube-system -l app.kubernetes.io/name=hubble-relay
kubectl get pods -n kube-system -l app.kubernetes.io/name=hubble-ui
```

**Expected:**
- `hubble-relay-*`: `1/1 Running`
- `hubble-ui-*`: `2/2 Running` (frontend nginx + backend containers)

### 3. Validate the Gateway listener

```bash
kubectl get gateway eg -n envoy-gateway-system \
  -o jsonpath='{range .spec.listeners[*]}{.name}: {.port}{"\n"}{end}'
```

**Expected:**
```
http: 80
hubble: 8080
```

### 4. Validate the HTTPRoute

```bash
kubectl get httproute hubble-ui -n kube-system
kubectl get httproute hubble-ui -n kube-system \
  -o jsonpath='{.status.parents[0].conditions[*].type}: {.status.parents[0].conditions[*].status}{"\n"}'
```

**Expected:** `Accepted ResolvedRefs: True True`

### 5. Smoke test the route (from inside the VM)

The Gateway IP (`192.168.5.15`) is on the Lima vmnet interface and is **not directly
reachable from the Mac**. Test from inside the VM:

```bash
# Get the NodePort for the hubble listener (port 8080)
HUBBLE_PORT=$(kubectl get svc -n envoy-gateway-system \
  -l gateway.envoyproxy.io/owning-gateway-name=eg \
  -o jsonpath='{.items[0].spec.ports[?(@.port==8080)].nodePort}')
echo "Hubble NodePort: $HUBBLE_PORT"

limactl shell k8s-lab curl -s -o /dev/null -w "hubble: %{http_code}\n" \
  http://192.168.5.15:$HUBBLE_PORT/
# Expected: hubble: 200
```

### 6. Open SSH tunnels for browser access

The Mac browser **cannot** reach `192.168.5.15` directly. Use the SSH tunnel to
reach the Gateway NodePort. This is part of the standard UI tunnel command:

```bash
# Run in a dedicated terminal (tunnels all three UI ports at once)
ssh -F ~/.lima/k8s-lab/ssh.config -N \
  -L 8080:localhost:32305 \
  -L 9080:localhost:30080 \
  -L 12000:localhost:32091 \
  lima-k8s-lab
```

> `32091` is the NodePort for the `hubble:8080` Gateway listener. Verify with:
> ```bash
> kubectl get svc -n envoy-gateway-system \
>   -l gateway.envoyproxy.io/owning-gateway-name=eg \
>   -o jsonpath='{.items[0].spec.ports[?(@.port==8080)].nodePort}'
> ```

Verify from the Mac:
```bash
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:12000/
# Expected: 200
```

### 7. Browser validation

Open in a Mac browser:

```
http://localhost:12000
```

**Expected:**
- Hubble UI loads with a namespace dropdown in the top-left
- Select a namespace (e.g. `mesh-demo` or `kube-system`) to see the service map
- Network flows between pods appear as animated lines on the graph

> **Why a dedicated listener, not a subpath?**
> Hubble UI uses `<base href="/">` so all JS/CSS asset requests go to `/`, which would
> conflict with other routes on the main `http:80` listener. A dedicated `hubble:8080`
> listener lets Hubble UI own the entire port with no path conflict.
> 
> **Why SSH tunnel, not direct NodePort?**
> The Lima vmnet IP (`192.168.5.15`) is only routable from inside the VM. SSH tunnels
> forward the NodePort through the SSH connection to `localhost`, making it reachable
> from the Mac. SSH tunnels are preferred over `kubectl port-forward` because they don't
> die on Envoy idle connection timeout.

---

## Expected Final State

| Check | Expected |
|-------|----------|
| ArgoCD app `hubble-k8s-lab` | `Synced Healthy` |
| `hubble-relay` pod | `1/1 Running` |
| `hubble-ui` pod | `2/2 Running` |
| Gateway listener `hubble` | port `8080` present |
| HTTPRoute `hubble-ui` | `Accepted` |
| `curl localhost:12000/` | `200` |
| Browser | Hubble service map renders |
