# Cluster Health (Baseline)

A quick "is the cluster healthy?" check. Paste this into any agent session after a
lab start or a major change to confirm the foundation is solid before going deeper.

Feature-specific validation lives in the relevant task file (e.g. `tasks/validate-istio-ambient.md`).

---

## 1. Nodes

```bash
kubectl get nodes
```

**Done when:** all nodes `Ready`.

---

## 2. System pods

```bash
kubectl get pods -A | grep -Ev 'Running|Completed'
```

**Done when:** no pods in `CrashLoopBackOff`, `Error`, or `Pending` (excluding expected init states).

---

## 3. ArgoCD applications

```bash
kubectl get applications -n argocd
```

**Done when:** all Applications show `Synced` + `Healthy`. If any are `OutOfSync`:

```bash
kubectl annotate application <name> -n argocd argocd.argoproj.io/refresh=normal --overwrite
```

---

## 4. Envoy Gateway smoke test

```bash
GATEWAY_IP=$(kubectl get gateway eg -n envoy-gateway-system \
  -o jsonpath='{.status.addresses[0].value}')

curl -s http://$GATEWAY_IP/hello   # expect: whoami response body
curl -s http://$GATEWAY_IP/vite/   # expect: HTML page
```

**Done when:** both return HTTP 200.
