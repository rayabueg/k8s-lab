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
  kustomization.yaml          ← lists all four Application manifests
  application-base.yaml       ← ArgoCD Application for istio-base (CRDs)
  application-istiod.yaml     ← ArgoCD Application for istiod (ambient profile)
  application-cni.yaml        ← ArgoCD Application for istio-cni (ambient)
  application-ztunnel.yaml    ← ArgoCD Application for ztunnel DaemonSet
```

> The ApplicationSet at `clusters/k8s-lab/applicationset.yaml` auto-discovers
> any folder under `addons/*` — no changes to root kustomization needed.

## Tasks

1. Confirm the Istio version that supports Ambient + ARM64 (1.22+ recommended)
2. Create the folder structure above under `cluster-addons/`
3. Write ArgoCD `Application` manifests (Helm source) for each component in install order:
   - `application-base.yaml` — chart `base`, `syncPolicy.syncOptions: [Replace=true]`, wave `-4`
   - `application-istiod.yaml` — chart `istiod`, values: `profile: ambient`, wave `-3`
   - `application-cni.yaml` — chart `cni`, values: `profile: ambient`, `cni.exclusive: false`, wave `-2`
   - `application-ztunnel.yaml` — chart `ztunnel`, wave `-1`
4. Write `kustomization.yaml` listing all four Application manifests as resources
5. Commit and push `cluster-addons` — ApplicationSet auto-discovers `addons/istio/`
6. Verify Cilium config compatibility:
   - `kubeProxyReplacement: true` + Istio Ambient requires `socketLB` awareness
   - `cni.exclusive: false` is the key flag for Cilium + Istio CNI coexistence

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
