# Task: Start Up the Lab

> **Context**: Read `agents/context.md` first.

## Objective

Start the k8s-lab from whatever state it's in — either resuming a stopped VM or
rebuilding from scratch — and get the cluster fully operational with ArgoCD synced.

---

## Step 0: Determine current state

```bash
limactl list
```

| Output | What to do |
|---|---|
| `k8s-lab   Running` | Skip to **Step 2** (tunnel + verify) |
| `k8s-lab   Stopped` | **Step 1a** — resume |
| No instance listed | **Step 1b** — full rebuild |

---

## Step 1a: Resume a stopped VM

```bash
cd ~/code/k8s-lab/lima
./bootstrap-cluster.sh
```

`bootstrap-cluster.sh` is idempotent — it starts the VM, skips kubeadm init if already
initialized, and re-exports the kubeconfig. Safe to run every time.

Expected duration: ~1-2 minutes.

After resume, continue through Steps 2 → 3, then open the ArgoCD UI (**Step 6**).

---

## Step 1b: Full rebuild (VM does not exist)

```bash
cd ~/code/k8s-lab/lima

# 1. Provision VM + containerd + kubeadm (~5-10 min)
./rebuild-lab.sh

# 2. Bootstrap cluster: kubeadm init + Cilium + ArgoCD (~5 min)
./bootstrap-cluster.sh
```

Expected total duration: ~10-15 minutes.

**Override defaults if needed** (e.g., fewer resources):
```bash
VM_NAME=k8s-lab CPUS=4 MEMORY=8 DISK=40 ./rebuild-lab.sh
```

After rebuild, continue through Steps 2 → 3 → 4 → 5, then open the ArgoCD UI (**Step 6**).

---

## Step 2: Open the API tunnel (required for host kubectl)

In a **dedicated terminal** (leave it running):

```bash
ssh -F ~/.lima/k8s-lab/ssh.config -N -L 6443:127.0.0.1:6443 lima-k8s-lab
```

> This tunnel forwards `127.0.0.1:6443` on your Mac to the cluster API inside the VM.
> Without it, all `kubectl` commands will fail with `connection refused`.

---

## Step 3: Set KUBECONFIG and verify cluster

In a **new terminal**:

```bash
export KUBECONFIG=~/.kube/lima-k8s-lab

# Node should be Ready
kubectl get nodes

# Core pods should be Running
kubectl get pods -n kube-system
```

Expected: node `lima-k8s-lab` in `Ready` state, Cilium pods running.

---

## Step 4: Apply root ArgoCD apps (fresh rebuild only)

> Skip this step if resuming a stopped VM — the apps will already be in the cluster.

The bootstrap script installs ArgoCD but does **not** apply the root Application.
You must apply it once to kick off GitOps:

```bash
# cluster-addons root app (discovers all addons via ApplicationSet)
kubectl apply -f ~/code/k8s-lab/cluster-addons/bootstrap/argocd/root-app.yaml

# cluster-applications root app (discovers team apps)
kubectl apply -f ~/code/k8s-lab/cluster-applications/bootstrap/argocd/root-app.yaml
```

ArgoCD will immediately begin syncing all addon folders (`cert-manager`, `envoy-gateway`,
`istio`, `mesh-demo`, etc.) in sync-wave order. Allow 2-3 minutes for the first full sync.

---

## Step 5: Verify ArgoCD is up

```bash
kubectl get pods -n argocd
kubectl get applications -n argocd
```

Expected: all ArgoCD pods `Running`, all Applications `Synced` + `Healthy`.

If any Application is `OutOfSync` after sync completes:
```bash
# Force a refresh on a specific app
kubectl annotate application <name> -n argocd \
  argocd.argoproj.io/refresh=normal --overwrite
```

---

## Step 6: Open port-forwards for browser access

All browser-accessible UIs are routed through the Envoy Gateway service. Use
`kubectl port-forward` directly to the Envoy service — this avoids NodePort
instability across rebuilds (NodePorts are reassigned each time).

First, look up the Envoy service name (it includes a hash that changes on rebuild):

```bash
export KUBECONFIG=~/.kube/lima-k8s-lab
kubectl get svc -n envoy-gateway-system
# Look for the service named: envoy-envoy-gateway-system-eg-<hash>
```

**Run in a dedicated terminal (leave it running):**

> **Note on port 9080 (ArgoCD):** ArgoCD runs in TLS mode and is accessed
> directly via `svc/argocd-server:443` — bypassing Envoy Gateway entirely for
> this port. This avoids the TCP reset caused by ArgoCD's HTTPS redirect killing
> `kubectl port-forward`. Accept the self-signed cert warning in the browser.

```bash
export KUBECONFIG=~/.kube/lima-k8s-lab
SVC=$(kubectl get svc -n envoy-gateway-system -o name | grep 'envoy-envoy-gateway' | cut -d/ -f2)

# 8080 (demo-vite) and 12000 (hubble) — via Envoy Gateway
kubectl port-forward -n envoy-gateway-system svc/$SVC 8080:8080 12000:12000 &

# 9080 (argocd) — directly to argocd-server TLS port
kubectl port-forward -n argocd svc/argocd-server 9080:443 &
```

Port mappings:
- `8080` → Gateway `http:8080` (demo-vite UI)
- `9080` → `argocd-server:443` directly (ArgoCD UI — **https://localhost:9080**)
- `12000` → Gateway `hubble:12000` (Hubble UI)

Verify all three are up:
```bash
curl -s -o /dev/null -w "Demo-vite (8080): %{http_code}\n" http://localhost:8080/vite/
curl -sk -o /dev/null -w "ArgoCD    (9080): %{http_code}\n" https://localhost:9080/
curl -s -o /dev/null -w "Hubble   (12000): %{http_code}\n" http://localhost:12000/
# expected: 200, 200, 200
```

### ArgoCD UI — https://localhost:9080

Browse to **https://localhost:9080** and accept the self-signed certificate warning.

Retrieve credentials:
```bash
# username: admin
kubectl get secret argocd-initial-admin-secret -n argocd \
  -o jsonpath='{.data.password}' | base64 -d && echo
```

### Hubble UI — http://localhost:12000

Browse to **http://localhost:12000** and select a namespace to see the service map.

### App traffic — http://localhost:8080

The `http` Gateway listener (port 8080) serves all application HTTPRoutes (`/vite/`, `/mesh-demo`, etc.).

---

## Step 7: Verify Envoy Gateway listeners

```bash
export KUBECONFIG=~/.kube/lima-k8s-lab
kubectl get pods -n envoy-gateway-system
kubectl get gateway eg -n envoy-gateway-system \
  -o jsonpath='{range .spec.listeners[*]}{.name}: {.port}{"\n"}{end}'
# expected:
# http: 8080
# hubble: 12000
# argocd: 9080
```

---

## Full startup checklist

- [ ] `limactl list` shows `k8s-lab   Running`
- [ ] Root ArgoCD apps applied (fresh rebuild only)
- [ ] API tunnel running: `ssh -F ~/.lima/k8s-lab/ssh.config -N -L 6443:127.0.0.1:6443 lima-k8s-lab`
- [ ] `kubectl get nodes` → node `Ready`
- [ ] `kubectl get pods -n kube-system` → all `Running`
- [ ] `kubectl get applications -n argocd` → all `Synced` + `Healthy`
- [ ] Gateway listeners: `http: 8080`, `hubble: 12000`, `argocd: 9080`
- [ ] Port-forwards running: `8080:8080`, `12000:12000` (via Envoy), `9080:443` (direct to argocd-server)
- [ ] `curl http://localhost:8080/vite/` → `200` (demo-vite)
- [ ] `curl -sk https://localhost:9080/` → `200` (ArgoCD)
- [ ] `curl http://localhost:12000/` → `200` (Hubble)
- [ ] `curl http://localhost:9080/vite/` → `200` (vite UI)
- [ ] Istio: `kubectl get pods -n istio-system` → see `cluster-addon-validate-istio.md`

---

## Troubleshooting

| Symptom | Fix |
|---|---|
| `connection refused` to `127.0.0.1:6443` | SSH tunnel not running — run Step 2 |
| `Can't open user config file ~/.lima/k8s-lab/ssh.config` | VM not started — run Step 1a/1b |
| Node `NotReady` | Cilium not yet ready: `kubectl get pods -n kube-system -l k8s-app=cilium` |
| ArgoCD pod `Pending` | Resources constrained — check `kubectl describe pod -n argocd` |
| No Applications in ArgoCD after fresh rebuild | Root apps not applied — run Step 4 |
| ArgoCD apps stuck `OutOfSync` | Annotate app with `argocd.argoproj.io/refresh=normal` (see Step 5) |
| Gateway IP unreachable from host (`HTTP 000`) | Gateway IP is a VM-internal address; curl from inside VM: `limactl shell k8s-lab curl http://<ip>/hello` |
| `kubectl` returns stale data after VM suspend/resume | Re-run `bootstrap-cluster.sh` to refresh kubeconfig |
| ArgoCD UI returns `502 Bad Gateway` | Port-forward dropped — re-run Step 6 |
| ArgoCD `admin` password unknown | `kubectl get secret argocd-initial-admin-secret -n argocd -o jsonpath='{.data.password}' \| base64 -d` |
