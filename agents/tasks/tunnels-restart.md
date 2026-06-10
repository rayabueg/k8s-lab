# Task: Restart Lab Tunnels

> **Context**: Read `agents/context.md` first.

## Objective

Kill stale `kubectl port-forward` processes, restart them cleanly, and verify all lab
UIs are reachable.

---

## Overview

| Purpose | Local port | Target |
|---|---|---|
| Kubernetes API | `6443` | **auto-forwarded by Lima's hostagent** — nothing to start |
| Demo apps / Envoy HTTP | `8080` | Envoy Gateway listener `http:8080` |
| ArgoCD UI | `9080` | Envoy Gateway listener `argocd:9080` (plain HTTP — argocd-server runs `server.insecure=true`) |
| Hubble UI | `12000` | Envoy Gateway listener `hubble:12000` |

All three UI ports go through **one** port-forward of the Envoy Gateway service
(`envoy-envoy-gateway-system-eg-<hash>` in `envoy-gateway-system`). No SSH tunnels are
needed: the API is auto-forwarded by Lima, and the UI NodePorts are unstable across
service recreations so `kubectl port-forward` to the service is preferred.

---

## Step 1: Kill existing port-forwards

```bash
pkill -f 'kubectl port-forward' 2>/dev/null
echo 'killed'
```

Wait 1–2 seconds before continuing.

---

## Step 2: Verify the API forward (should already exist)

```bash
lsof -nP -iTCP:6443 -sTCP:LISTEN
# expected: a `limactl` process listening on 127.0.0.1:6443
```

If nothing is listening, the VM is probably stopped — see `tasks/cluster-start-up.md`.

---

## Step 3: Start the UI port-forward

```bash
export KUBECONFIG="$HOME/.kube/lima-k8s-lab"
SVC=$(kubectl get svc -n envoy-gateway-system -o name | grep 'envoy-envoy-gateway' | cut -d/ -f2)
nohup kubectl port-forward -n envoy-gateway-system "svc/$SVC" \
  8080:8080 12000:12000 9080:9080 \
  >/tmp/pf-envoy.log 2>&1 & echo "envoy PID:$!"
```

---

## Step 4: Verify all ports are listening

```bash
for port in 6443 8080 9080 12000; do
  count=$(lsof -i :$port 2>/dev/null | grep -c LISTEN)
  echo "port $port: $count listener(s)"
done
```

Expected: every port shows at least 1 listener.

---

## Step 5: Smoke-test the UIs

```bash
curl -s -o /dev/null -w 'demo-vite-ui: %{http_code}\n' http://localhost:8080/vite/
curl -s -o /dev/null -w 'hubble-ui:    %{http_code}\n' http://localhost:12000/
curl -s -o /dev/null -w 'argocd:       %{http_code}\n' http://localhost:9080/
```

Expected: `200` for all three.

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `port XXXX: 0 listener(s)` | Port-forward failed to start | Check `/tmp/pf-envoy.log` for errors |
| Port-forward shows 0 | `nohup` received `KUBECONFIG=...` as command | Export `KUBECONFIG` before running `nohup kubectl...` |
| Envoy returns `502` or hangs | Envoy proxy pod not running | `kubectl get pods -n envoy-gateway-system` |
| `curl` returns `000` (exit 7) | Port not bound — forward not running | Re-run Step 3 |
| Port-forward dies on its own | Envoy service was recreated (new hash) | Re-run Step 3 — `$SVC` lookup picks up the new name |
| `connection reset` on 9080 | Talking TLS to a plain-HTTP server (or vice versa) | ArgoCD is plain HTTP via the gateway since the Gen2 migration — use `http://localhost:9080` |
| port 6443: 0 listeners | VM stopped (Lima forward dies with it) | `limactl list`; see `tasks/cluster-start-up.md` |
