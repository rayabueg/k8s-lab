# Task: Deploy a Sample App

> **Context**: Read `agents/context.md` first.

## Objective

Deploy a simple demo workload to the cluster via GitOps, expose it through Envoy Gateway,
and (optionally) enroll it in Istio Ambient for mTLS verification.

## Constraints

- All manifests committed to `cluster-addons/` (infra objects) or `cluster-applications/` (ArgoCD Application)
- No manual `kubectl apply`
- Container image must be ARM64-compatible (use `traefik/whoami` for echo-style tests)
- Expose via Gateway API `HTTPRoute`, not Ingress

## Tasks

1. **Create an addon folder** for the app under `cluster-addons/clusters/k8s-lab/addons/<app-name>/`:
   - `namespace.yaml` — `kind: Namespace` with optional `istio.io/dataplane-mode: ambient` label
   - `deployment.yaml` — use `traefik/whoami:v1.10.3` (ARM64-safe)
   - `service.yaml` — ClusterIP
   - `httproute.yaml` — `HTTPRoute` pointing at `eg` Gateway in `envoy-gateway-system`
   - `kustomization.yaml` — lists all four resources

   > This is the established pattern (see `cluster-addons/clusters/k8s-lab/addons/mesh-demo/`).
   > The ApplicationSet auto-discovers the folder — no other files need to be edited.

2. **Commit and push** `cluster-addons` — ArgoCD will sync automatically

3. **Verify**:
   ```bash
   # ArgoCD synced
   kubectl get application <app-name>-k8s-lab -n argocd

   # Pods running
   kubectl get pods -n <app-namespace>

   # Route works — Gateway IP is VM-internal, curl from inside the VM
   GATEWAY_IP=$(kubectl get gateway eg -n envoy-gateway-system \
     -o jsonpath='{.status.addresses[0].value}')
   limactl shell k8s-lab curl -s http://$GATEWAY_IP/<path>
   ```
   **Done when:** pods `Running`, HTTPRoute `Accepted`, curl returns a response body.

## Example HTTPRoute

Route by path prefix (no `hostnames` needed — the gateway handles all hosts):

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: <app-name>
  namespace: <app-name>
spec:
  parentRefs:
    - group: gateway.networking.k8s.io
      kind: Gateway
      name: eg
      namespace: envoy-gateway-system
  rules:
    - matches:
        - path:
            type: PathPrefix
            value: /<app-name>
      backendRefs:
        - name: <app-name>
          port: 80
```

## Optional: Enroll in Istio Ambient

After Istio Ambient is installed (see `tasks/cluster-addon-istio-ambient.md`), enable ambient mode
for the namespace:

```bash
kubectl label namespace <app-namespace> istio.io/dataplane-mode=ambient
```

Verify enrollment:
```bash
# Pods should appear in ztunnel's workload table
kubectl exec -n istio-system ds/ztunnel -- ztunnel-cli workloads 2>/dev/null | grep <app-namespace>

# No istio-proxy sidecar in pods
kubectl get pods -n <app-namespace> \
  -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{range .spec.containers[*]}{.name}{" "}{end}{"\n"}{end}'
# ^^^ should NOT include "istio-proxy"
```
