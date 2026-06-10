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
cd ~/code/personal/k8s-lab/bootstrap/lima
./bootstrap-cluster.sh
```

`bootstrap-cluster.sh` is idempotent — it starts the VM, skips kubeadm init if already
initialized, and re-exports the kubeconfig. Safe to run every time.

Expected duration: ~1-2 minutes.

After resume, continue through Steps 2 → 3, then open the ArgoCD UI (**Step 6**).

---

## Step 1b: Full rebuild (VM does not exist)

```bash
cd ~/code/personal/k8s-lab/bootstrap/lima

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

## Step 2: Verify the API is reachable (Lima auto-forwards 6443)

Lima's hostagent automatically forwards the cluster API to `127.0.0.1:6443` — no manual
SSH tunnel is needed. Verify the forward exists:

```bash
lsof -nP -iTCP:6443 -sTCP:LISTEN
# expected: a `limactl` process listening on 127.0.0.1:6443
```

**Fallback** — only if nothing is listening on 6443 (in a dedicated terminal, leave it running):

```bash
ssh -F ~/.lima/k8s-lab/ssh.config -N -L 6443:127.0.0.1:6443 lima-k8s-lab
```

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
kubectl apply -f ~/code/personal/k8s-lab/cluster-addons/bootstrap/argocd/root-app.yaml

# cluster-applications root app (discovers team apps)
kubectl apply -f ~/code/personal/k8s-lab/cluster-applications/bootstrap/argocd/root-app.yaml
```

ArgoCD will immediately begin syncing all addon folders (`cert-manager`, `envoy-gateway`,
`istio`, `mesh-demo`, etc.) in sync-wave order. Allow 2-3 minutes for the first full sync.

---

## Step 5: Verify ArgoCD is up

```bash
kubectl get pods -n argocd
kubectl get applications -n argocd
```

Expected: all ArgoCD pods `Running`, all **16** Applications `Synced` + `Healthy`
(11 addon apps from the `cluster-addons-k8s-lab-k8s-lab` ApplicationSet, plus
`k8s-lab-root`, `cluster-applications`, and the 3 team apps).

> **After a VM resume**, apps may show `Unknown`/`Unknown` for a couple of minutes while
> the restarted application controller re-reconciles. Annotate a refresh (below) and wait.
> If `Unknown` persists, see the `argocd-cm` row in Troubleshooting.

If any Application is `OutOfSync` after sync completes:
```bash
# Force a refresh on a specific app
kubectl annotate application <name> -n argocd \
  argocd.argoproj.io/refresh=normal --overwrite
```

---

## Step 6: Open port-forwards for browser access

All browser-accessible UIs — **including ArgoCD** — are routed through the Envoy
Gateway service, so a single `kubectl port-forward` covers everything. (Since the
Gen2 migration, the `argocd-config` addon runs argocd-server with
`server.insecure: "true"` and routes it via an `HTTPRoute` on the gateway's `argocd`
listener — the old direct-TLS forward to `svc/argocd-server:443` is obsolete.)

The Envoy service name includes a hash that changes on rebuild; it is a **NodePort**
service (port 80 pinned to NodePort 30080 by the EnvoyProxy patch; the other
NodePorts are reassigned on each service recreation — hence the port-forward).

**Run in a dedicated terminal (leave it running):**

```bash
export KUBECONFIG=~/.kube/lima-k8s-lab
SVC=$(kubectl get svc -n envoy-gateway-system -o name | grep 'envoy-envoy-gateway' | cut -d/ -f2)

# one forward for all three UIs
kubectl port-forward -n envoy-gateway-system svc/$SVC 8080:8080 12000:12000 9080:9080 &
```

Port mappings (all via the Gateway):
- `8080` → Gateway `http:8080` (demo apps: `/vite/`, `/hello`, …)
- `9080` → Gateway `argocd:9080` (ArgoCD UI — **http://localhost:9080**, plain HTTP)
- `12000` → Gateway `hubble:12000` (Hubble UI)

Verify all three are up:
```bash
curl -s -o /dev/null -w "Demo-vite (8080): %{http_code}\n" http://localhost:8080/vite/
curl -s -o /dev/null -w "ArgoCD    (9080): %{http_code}\n" http://localhost:9080/
curl -s -o /dev/null -w "Hubble   (12000): %{http_code}\n" http://localhost:12000/
# expected: 200, 200, 200
```

### ArgoCD UI — http://localhost:9080

Browse to **http://localhost:9080** (plain HTTP, no cert warning).

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
- [ ] API reachable: `lsof -nP -iTCP:6443 -sTCP:LISTEN` shows a `limactl` listener (Lima auto-forward)
- [ ] `kubectl get nodes` → node `Ready`
- [ ] `kubectl get pods -n kube-system` → all `Running`
- [ ] `kubectl get applications -n argocd` → all 16 `Synced` + `Healthy`
- [ ] Gateway listeners: `http: 8080`, `hubble: 12000`, `argocd: 9080`; Gateway `Accepted=True` and `Programmed=True`
- [ ] Single port-forward running on the Envoy svc: `8080:8080 12000:12000 9080:9080`
- [ ] `curl http://localhost:8080/vite/` → `200` (demo-vite)
- [ ] `curl http://localhost:9080/` → `200` (ArgoCD, plain HTTP)
- [ ] `curl http://localhost:12000/` → `200` (Hubble)
- [ ] Istio: `kubectl get pods -n istio-system` → see `cluster-addon-validate-istio.md`

---

## Image-pull / bootstrap timing notes

> **Applicable to fresh rebuilds and first-resume after a rebuild.**

After `bootstrap-cluster.sh` completes, Cilium and ArgoCD images may still be pulling
from the internet inside the VM. This is normal — the script exits as soon as the
manifests are applied, not after the images are ready.

**Do not immediately run the ArgoCD / pod-health checks.** Instead:

1. Wait for Cilium to be fully ready (all `cilium-*` pods `Running`):

   ```bash
   export KUBECONFIG=~/.kube/lima-k8s-lab
   kubectl rollout status daemonset/cilium -n kube-system --timeout=10m
   ```

   Expected duration after a cold start: **2–5 minutes** (image pull + startup).

2. Then wait for ArgoCD:

   ```bash
   kubectl rollout status deployment/argocd-server -n argocd --timeout=10m
   ```

   Expected duration: **1–3 minutes** after Cilium is healthy.

3. Only after both are `Running`, proceed to **Step 5** (verify ArgoCD applications).

If `kubectl rollout status` times out, check image pull progress:

```bash
kubectl describe pod -n kube-system -l k8s-app=cilium | grep -A5 'Events:'
kubectl describe pod -n argocd -l app.kubernetes.io/name=argocd-server | grep -A5 'Events:'
```

`Pulling image` in events is normal — wait for `Pulled` and `Started`.

---

## Troubleshooting

| Symptom | Fix |
|---|---|
| `connection refused` to `127.0.0.1:6443` | Lima auto-forward missing — check `limactl list` shows Running; fallback SSH tunnel in Step 2 |
| `Can't open user config file ~/.lima/k8s-lab/ssh.config` | VM not started — run Step 1a/1b |
| ALL apps `Unknown`/`Unknown`, controller logs spam `configmap "argocd-cm" not found` | Recreate `argocd-cm` from `platform-addons/argocd-config/manifests/argocd-cm-health.yaml` **and** label it `app.kubernetes.io/part-of=argocd` (ArgoCD ignores the cm without the label) |
| Apps stuck deleting (deletionTimestamp set) after an AppSet rename | The old AppSet was pruned and is cascade-deleting its apps; strip `resources-finalizer.argocd.argoproj.io` from each deleting app (`kubectl patch application <n> -n argocd --type merge -p '{"metadata":{"finalizers":null}}'`) so they delete WITHOUT destroying cluster resources; the new AppSet recreates them and adopts the live resources |
| envoy-gateway app `Progressing` forever; Envoy svc is `LoadBalancer` `<pending>` despite EnvoyProxy `type: NodePort` | GatewayClass is zombie-deleting: check `kubectl get gatewayclass eg -o jsonpath='{.metadata.deletionTimestamp}'`. EG silently ignores `parametersRef` on a deleting GatewayClass. Fix: remove its finalizer, let it delete, immediately re-apply from `platform-addons/envoy-gateway/manifests/gateway/eg-gateway.yaml`. Deleting just the svc does NOT help — EG recreates it wrong until the GatewayClass is clean |
| Node `NotReady` | Cilium not yet ready: `kubectl get pods -n kube-system -l k8s-app=cilium` |
| ArgoCD pod `Pending` | Resources constrained — check `kubectl describe pod -n argocd` |
| No Applications in ArgoCD after fresh rebuild | Root apps not applied — run Step 4 |
| ArgoCD apps stuck `OutOfSync` | Annotate app with `argocd.argoproj.io/refresh=normal` (see Step 5) |
| Gateway IP unreachable from host (`HTTP 000`) | Gateway IP is a VM-internal address; curl from inside VM: `limactl shell k8s-lab curl http://<ip>/hello` |
| `kubectl` returns stale data after VM suspend/resume | Re-run `bootstrap-cluster.sh` to refresh kubeconfig |
| Cilium / ArgoCD pods stuck in `ContainerCreating` or `Init` | Images still pulling — wait and monitor with `kubectl rollout status` (see **Image-pull / bootstrap timing notes**) |
| ArgoCD Applications not appearing after fresh rebuild | ArgoCD server not ready yet — wait for `argocd-server` rollout before applying root apps |
| ArgoCD UI returns `502 Bad Gateway` | Port-forward dropped — re-run Step 6 |
| ArgoCD `admin` password unknown | `kubectl get secret argocd-initial-admin-secret -n argocd -o jsonpath='{.data.password}' \| base64 -d` |
