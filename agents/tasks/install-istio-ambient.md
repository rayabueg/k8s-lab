# Task: Install Istio Ambient

> **Context**: Read `agents/context.md` first.

## Objective

Install Istio Ambient mode into the cluster via ArgoCD — sidecar-less, eBPF-friendly,
coexisting with Cilium CNI and Envoy Gateway.

## Constraints

- Must be GitOps-managed (no manual `kubectl apply` in cluster-addons scope)
- Must use Istio **Ambient** profile — no sidecar injection, no Istio ingress gateway
- Must coexist with Envoy Gateway (which remains the sole ingress)
- Must coexist with Cilium CNI (confirm Cilium + Istio Ambient compatibility flags)
- Images must be multi-arch / ARM64-compatible
- ArgoCD sync order must be respected: CRDs → base → istiod → cni → ztunnel

## Folder structure to create

Under `cluster-addons/clusters/k8s-lab/addons/`:

```
istio/
  kustomization.yaml          ← umbrella kustomization (or split per component)
  base/
    application.yaml          ← ArgoCD Application for istio-base (CRDs)
    kustomization.yaml
  istiod/
    application.yaml          ← ArgoCD Application for istiod (ambient profile)
    kustomization.yaml
  cni/
    application.yaml          ← ArgoCD Application for istio-cni (ambient)
    kustomization.yaml
  ztunnel/
    application.yaml          ← ArgoCD Application for ztunnel DaemonSet
    kustomization.yaml
```

## Tasks

1. Confirm the Istio version that supports Ambient + ARM64 (1.22+ recommended)
2. Create the folder structure above under `cluster-addons/`
3. Write ArgoCD `Application` manifests (Helm source) for each component in install order:
   - `istio-base` — CRDs only, `syncPolicy.syncOptions: [Replace=true]`
   - `istiod` — Helm chart `istiod`, values: `profile: ambient`
   - `istio-cni` — Helm chart `cni`, values: `profile: ambient`
   - `ztunnel` — Helm chart `ztunnel`
4. Add each component to the root `kustomization.yaml` at `clusters/k8s-lab/`
5. Verify Cilium config compatibility:
   - `kubeProxyReplacement: true` + Istio Ambient requires `socketLB` awareness
   - Check if `cilium.io/ambient-compat` annotations are needed
6. Confirm ApplicationSet auto-discovers the new addon folder (check `applicationset.yaml` glob)

## Validation

Run these after ArgoCD syncs all four Istio Applications:

```bash
# All Istio pods Running
kubectl get pods -n istio-system

# DaemonSets: DESIRED == READY
kubectl get ds ztunnel -n istio-system
kubectl get ds istio-cni-node -n istio-system
```

**Done when:**
- `istiod-*` pod `Running`
- `ztunnel` DaemonSet: all nodes `DESIRED == READY`
- `istio-cni-node` DaemonSet: all nodes `DESIRED == READY`
- No existing workload pods restarted unexpectedly
- Envoy Gateway regression check passes:
  ```bash
  GATEWAY_IP=$(kubectl get gateway eg -n envoy-gateway-system \
    -o jsonpath='{.status.addresses[0].value}')
  curl -s http://$GATEWAY_IP/hello   # still returns 200
  ```

## References

- https://istio.io/latest/docs/ambient/install/
- https://istio.io/latest/docs/ambient/install/helm/
- https://docs.cilium.io/en/stable/network/servicemesh/istio/
