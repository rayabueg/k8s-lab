# Cluster Health (Baseline)

A quick "is the cluster healthy?" check. Paste this into any agent session after a
lab start or a major change to confirm the foundation is solid before going deeper.

Feature-specific validation lives in the relevant task file (e.g. `tasks/cluster-addon-validate-istio.md`).

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

**Done when:** all Applications show `Synced` + `Healthy` (16 apps; the addon AppSet is
`cluster-addons-k8s-lab-k8s-lab`). If any are `OutOfSync`:

```bash
kubectl annotate application <name> -n argocd argocd.argoproj.io/refresh=normal --overwrite
```

If **all** apps show `Unknown`/`Unknown`: verify `argocd-cm` exists **and** carries the
`app.kubernetes.io/part-of: argocd` label — the controller is non-functional without it
(symptom: `configmap "argocd-cm" not found` spam in `argocd-application-controller-0` logs).

---

## 4. ArgoCD UI

**Agents: ensure the port-forward is running before this check.**

ArgoCD is reached **through Envoy Gateway** on the `argocd` listener (port 9080, plain
HTTP). The `argocd-config` addon sets `server.insecure: "true"` and routes
`HTTPRoute argocd → argocd-server svc:80`. (The old direct-TLS method via
`svc/argocd-server:443` is obsolete since the Gen2 migration re-enabled insecure mode.)

```bash
# Start if not already running (background terminal) — one forward covers all UIs
export KUBECONFIG=~/.kube/lima-k8s-lab   # or multipass-k8s-lab
SVC=$(kubectl get svc -n envoy-gateway-system -o name | grep 'envoy-envoy-gateway' | cut -d/ -f2)
kubectl port-forward -n envoy-gateway-system svc/$SVC 8080:8080 12000:12000 9080:9080 &
```

```bash
# Verify it's up (plain HTTP)
curl -s http://localhost:9080 | grep -o '<title>[^<]*</title>'
# expected: <title>Argo CD</title>
```

**Done when:** curl returns `<title>Argo CD</title>` and http://localhost:9080 is browsable.

---

## 5. Envoy Gateway smoke test

Simplest: with the port-forward from step 4 running, smoke-test from the host:

```bash
curl -s -o /dev/null -w "/hello: %{http_code}\n"  http://localhost:8080/hello
curl -s -o /dev/null -w "/vite/: %{http_code}\n"  http://localhost:8080/vite/
curl -s -o /dev/null -w "hubble: %{http_code}\n"  http://localhost:12000/
curl -s -o /dev/null -w "argocd: %{http_code}\n"  http://localhost:9080/
```

Alternative (no port-forward): the Gateway IP is VM-internal — not reachable from the host. Run the curl **inside the VM**:

```bash
# Lima (macOS)
limactl shell k8s-lab -- bash -c '
  GATEWAY_IP=$(kubectl get gateway eg -n envoy-gateway-system \
    -o jsonpath="{.status.addresses[0].value}")
  curl -s -o /dev/null -w "/hello: %{http_code}\n" http://$GATEWAY_IP/hello
  curl -s -o /dev/null -w "/vite/: %{http_code}\n" http://$GATEWAY_IP/vite/
'

# Multipass (Linux)
multipass exec k8s-lab -- bash -c '
  GATEWAY_IP=$(kubectl get gateway eg -n envoy-gateway-system \
    -o jsonpath="{.status.addresses[0].value}")
  curl -s -o /dev/null -w "/hello: %{http_code}\n" http://$GATEWAY_IP/hello
  curl -s -o /dev/null -w "/vite/: %{http_code}\n" http://$GATEWAY_IP/vite/
'
```

**Done when:** both routes return `200`.
