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

Expected: node `k8s-lab` in `Ready` state, Cilium pods running.

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

## Step 6: Verify Envoy Gateway

```bash
kubectl get pods -n envoy-gateway-system
kubectl get gateway eg -n envoy-gateway-system

GATEWAY_IP=$(kubectl get gateway eg -n envoy-gateway-system \
  -o jsonpath='{.status.addresses[0].value}')
echo "Gateway IP: $GATEWAY_IP"

curl -s http://$GATEWAY_IP/hello    # should return whoami response
curl -s http://$GATEWAY_IP/vite/    # should return Vite UI HTML
```

---

## Full startup checklist

- [ ] `limactl list` shows `k8s-lab   Running`
- [ ] Root ArgoCD apps applied (fresh rebuild only)
- [ ] SSH tunnel running in background terminal
- [ ] `kubectl get nodes` → node `Ready`
- [ ] `kubectl get pods -n kube-system` → all `Running`
- [ ] `kubectl get applications -n argocd` → all `Synced` + `Healthy`
- [ ] `curl http://$GATEWAY_IP/hello` → HTTP 200
- [ ] Istio: `kubectl get pods -n istio-system` → see `validate-istio-ambient.md`

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
