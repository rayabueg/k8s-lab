# Task: Validate Istio Ambient + mesh-demo After Cluster Boot

> **Context**: Read `agents/context.md` first.

## Objective

After the lab is started and ArgoCD has synced the new `istio` and `mesh-demo` addon
folders, validate that Istio Ambient is running correctly and the sample app is reachable
through Envoy Gateway with ambient-mode mTLS active.

## Prerequisites

- Lima VM running (`limactl list` shows `k8s-lab` as `Running`)
- SSH tunnel active: `ssh -F ~/.lima/k8s-lab/ssh.config -N -L 6443:127.0.0.1:6443 lima-k8s-lab`
- `export KUBECONFIG=~/.kube/lima-k8s-lab`

## Tasks

### 1. Wait for ArgoCD to sync

```bash
kubectl get applications -n argocd
```

Expected Applications (all `Synced` + `Healthy`):
- `istio-k8s-lab` (the umbrella ApplicationSet-generated app)
- `istio-base`, `istiod`, `istio-cni`, `ztunnel` (child apps)
- `mesh-demo-k8s-lab`

If any app is `OutOfSync`, force a refresh:
```bash
kubectl annotate application <name> -n argocd argocd.argoproj.io/refresh=normal --overwrite
```

### 2. Validate Istio control plane

```bash
# All pods in istio-system should be Running
kubectl get pods -n istio-system

# ztunnel DaemonSet: DESIRED == READY
kubectl get ds ztunnel -n istio-system

# istio-cni DaemonSet
kubectl get ds istio-cni-node -n istio-system
```

**Expected:**
- `istiod-*` pod `Running`
- `ztunnel-*` pod `Running` on every node (1 node in this lab)
- `istio-cni-node-*` pod `Running`

### 3. Validate mesh-demo workload

```bash
# Deployment should show 2/2 AVAILABLE
kubectl get deploy mesh-demo -n mesh-demo

# Pods should be Running (no istio-proxy sidecar container)
kubectl get pods -n mesh-demo
kubectl get pods -n mesh-demo -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{range .spec.containers[*]}{.name}{" "}{end}{"\n"}{end}'
# ^^^ should NOT include "istio-proxy"
```

### 4. Validate ambient enrollment

```bash
# Check ztunnel access logs — mesh-demo pods should appear as inbound connections
kubectl logs -n istio-system ds/ztunnel --tail=20 | grep mesh-demo
# Expected: lines like: direction="inbound" dst.workload="mesh-demo-..." dst.namespace="mesh-demo"
```

> Note: `ztunnel-cli` is not available in Istio 1.24.x images. Use log inspection instead.

If nothing appears, check the namespace label is set:
```bash
kubectl get namespace mesh-demo --show-labels
# Should include: istio.io/dataplane-mode=ambient
```

### 5. Validate HTTPRoute + Envoy Gateway

```bash
# Get the Gateway address and detect the correct HTTP port
GATEWAY_IP=$(kubectl get gateway eg -n envoy-gateway-system -o jsonpath='{.status.addresses[0].value}')
GATEWAY_SVC=$(kubectl get svc -n envoy-gateway-system -l gateway.envoyproxy.io/owning-gateway-name=eg -o name | head -1)
GATEWAY_PORT=$(kubectl get $GATEWAY_SVC -n envoy-gateway-system -o jsonpath='{.spec.ports[?(@.name=="http-80")].nodePort}')
# Fall back to port 80 if not NodePort
GATEWAY_PORT=${GATEWAY_PORT:-80}
echo "Gateway: http://$GATEWAY_IP:$GATEWAY_PORT"

# Gateway IP is VM-internal — must curl from inside the Lima VM
limactl shell k8s-lab curl -s http://$GATEWAY_IP:$GATEWAY_PORT/mesh-demo

# Expected: traefik/whoami response showing request headers + IP info
```

> The gateway service type may be `NodePort` (address = node IP, port = 30080) or `ClusterIP`
> (address = cluster IP, port = 80). Use the detection snippet above to handle both cases.

### 6. Validate existing routes not broken (regression)

> Requires `GATEWAY_IP` and `GATEWAY_PORT` to be set — run Step 5 first or re-export them.

```bash
limactl shell k8s-lab curl -s http://$GATEWAY_IP:$GATEWAY_PORT/hello    # demo-hello (whoami)
limactl shell k8s-lab curl -s -o /dev/null -w "%{http_code}" http://$GATEWAY_IP:$GATEWAY_PORT/vite/
```

Both should return HTTP 200.

### 7. Optional: Confirm mTLS is active between pods

The `whoami` image has no curl/wget, so use ztunnel logs as evidence of active mTLS:

```bash
# Confirm ztunnel is logging inbound connections for mesh-demo (direction=inbound = ztunnel-proxied)
kubectl logs -n istio-system ds/ztunnel --tail=10 | grep "mesh-demo.*inbound"
# Expected: lines with direction="inbound" bytes_sent=... confirming ztunnel is handling traffic
```

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `istiod` pod in `CrashLoopBackOff` | Helm values schema error | Check `kubectl logs -n istio-system deploy/istiod` |
| `istio-cni-node` not starting | Cilium CNI chain conflict | Verify `cni.exclusive: false` in application-cni.yaml |
| `ztunnel` pod `Pending` | Node taint/toleration issue | `kubectl describe pod -n istio-system -l app=ztunnel` |
| `mesh-demo` pods not in ztunnel workloads | Namespace label missing | `kubectl label ns mesh-demo istio.io/dataplane-mode=ambient` |
| `/mesh-demo` returns 404 | HTTPRoute not accepted | `kubectl get httproute mesh-demo -n mesh-demo -o yaml` |
| ArgoCD app stuck `OutOfSync` | CRD replace not working | Check `Replace=true` on `istio-base` Application |

## Expected final state

- [ ] All ArgoCD apps `Synced` + `Healthy`
- [ ] `ztunnel` DaemonSet `DESIRED == READY`
- [ ] `mesh-demo` pods running, no `istio-proxy` sidecar
- [ ] `curl http://$GATEWAY_IP/mesh-demo` returns 200 (via `limactl shell k8s-lab curl ...`)
- [ ] `curl http://$GATEWAY_IP/hello` still returns 200 (no regression)
- [ ] mesh-demo pods visible in `ztunnel-cli workloads`
