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

1. **Create namespace** — add to `cluster-addons/clusters/k8s-lab/addons/namespaces/namespaces.yaml`
   ```yaml
   - name: <app-namespace>
   ```

2. **Create app manifests** under `apps/<app-name>/` at repo root (or a new `cluster-addons` path):
   - `deployment.yaml` — use `traefik/whoami:v1.10.3` (ARM64 safe)
   - `service.yaml` — ClusterIP

3. **Expose via Envoy Gateway** — add to `cluster-addons/clusters/k8s-lab/addons/envoy-gateway/gateway/`:
   - `<app-name>.yaml` — `HTTPRoute` pointing at the service
   - Add to `kustomization.yaml` resources list

4. **Create ArgoCD Application** in `cluster-applications/apps/<app-name>.yaml`
   - Source: this repo, path to app manifests
   - Destination: cluster, target namespace

5. **Commit and push** — ArgoCD will sync automatically

6. **Verify**:
   ```bash
   # ArgoCD synced
   kubectl get application <app-name>-k8s-lab -n argocd

   # Pods running
   kubectl get pods -n <app-namespace>

   # Route works
   GATEWAY_IP=$(kubectl get gateway eg -n envoy-gateway-system \
     -o jsonpath='{.status.addresses[0].value}')
   curl -s http://$GATEWAY_IP/<path>
   ```
   **Done when:** pods `Running`, HTTPRoute `Accepted`, curl returns a response body.

## Example HTTPRoute (adapt host/service)

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: <app-name>
  namespace: <app-namespace>
spec:
  parentRefs:
    - name: eg
      namespace: envoy-gateway-system
  hostnames:
    - "<app-name>.local"
  rules:
    - matches:
        - path:
            type: PathPrefix
            value: /
      backendRefs:
        - name: <app-name>
          port: 80
```

## Optional: Enroll in Istio Ambient

After Istio Ambient is installed (see `tasks/install-istio-ambient.md`), enable ambient mode
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
