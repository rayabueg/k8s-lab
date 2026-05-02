# Task: Restart Lab Tunnels

> **Context**: Read `agents/context.md` first.

## Objective

Kill all stale SSH tunnels and `kubectl port-forward` processes, restart them cleanly,
and verify all lab UIs are reachable.

---

## Overview of tunnels

| Purpose | Local port | Target |
|---|---|---|
| Kubernetes API | `6443` | Lima VM `127.0.0.1:6443` (via SSH) |
| Demo apps / Envoy HTTP | `8080` | NodePort `31530` (Envoy Gateway listener `http`) |
| Hubble UI / Envoy | `12000` | NodePort `31328` (Envoy Gateway listener `hubble`) |
| ArgoCD UI (direct) | `9443` | `kubectl port-forward svc/argocd-server 443` |

NodePorts above are for the `envoy-envoy-gateway-system-eg-*` service in `envoy-gateway-system`.
If the Envoy service name or NodePorts have changed, re-query them first (see Step 0).

---

## Step 0 (optional): Re-query current NodePorts

Run this if you suspect the Envoy service name or NodePorts have changed since the table above was written:

```bash
kubectl get svc -n envoy-gateway-system | grep envoy-envoy
```

Look at the `PORT(S)` column:
- `80:XXXXX` → `http` listener NodePort
- `8080:XXXXX` → internal; the Envoy HTTP listener NodePort is the `8080` entry
- `12000:XXXXX` → `hubble` listener NodePort
- `9080:XXXXX` → `argocd` listener NodePort (not used here — direct kubectl pf is preferred)

Update the `Step 2` commands below if the NodePorts differ.

---

## Step 1: Kill all existing tunnels

```bash
pkill -f 'ssh.*k8s-lab' 2>/dev/null
pkill -f 'kubectl port-forward' 2>/dev/null
echo 'killed'
```

Wait 1–2 seconds before continuing.

---

## Step 2: Start SSH tunnels

Run each command individually (not as a single multi-line block):

```bash
# Kubernetes API
nohup ssh -F '/Users/Raymond.Abueg@AlaskaAir.com/.lima/k8s-lab/ssh.config' \
  -N -L 6443:127.0.0.1:6443 lima-k8s-lab \
  >/tmp/ssh-api.log 2>&1 & echo 'api PID:'$!

# Demo apps via Envoy (http listener → NodePort 31530)
nohup ssh -F '/Users/Raymond.Abueg@AlaskaAir.com/.lima/k8s-lab/ssh.config' \
  -N -L 8080:127.0.0.1:31530 lima-k8s-lab \
  >/tmp/ssh-envoy-http.log 2>&1 & echo 'envoy-http PID:'$!

# Hubble UI via Envoy (hubble listener → NodePort 31328)
nohup ssh -F '/Users/Raymond.Abueg@AlaskaAir.com/.lima/k8s-lab/ssh.config' \
  -N -L 12000:127.0.0.1:31328 lima-k8s-lab \
  >/tmp/ssh-envoy-hubble.log 2>&1 & echo 'hubble PID:'$!
```

---

## Step 3: Start ArgoCD port-forward

`kubectl port-forward` needs the KUBECONFIG exported first:

```bash
export KUBECONFIG="$HOME/.kube/lima-k8s-lab"
nohup kubectl port-forward -n argocd svc/argocd-server 9443:443 \
  >/tmp/pf-argocd.log 2>&1 & echo 'argocd PID:'$!
```

---

## Step 4: Verify all tunnels are listening

```bash
for port in 6443 8080 12000 9443; do
  count=$(lsof -i :$port 2>/dev/null | grep -c LISTEN)
  echo "port $port: $count listener(s)"
done
```

Expected output:
```
port 6443: 2 listener(s)
port 8080: 2 listener(s)
port 12000: 2 listener(s)
port 9443: 2 listener(s)
```

---

## Step 5: Smoke-test the UIs

```bash
# Demo Vite UI
curl -s -o /dev/null -w 'demo-vite-ui: %{http_code}\n' http://localhost:8080/vite/

# Hubble UI
curl -s -o /dev/null -w 'hubble-ui: %{http_code}\n' http://localhost:12000/

# ArgoCD API (expect 200 or 307)
curl -sk -o /dev/null -w 'argocd: %{http_code}\n' https://localhost:9443/
```

Expected: `200` for demo-vite-ui and hubble-ui, `200` or `307` for ArgoCD.

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `port XXXX: 0 listener(s)` | SSH tunnel failed to start | Check `/tmp/ssh-*.log` for errors |
| ArgoCD port-forward shows 0 | `nohup` received `KUBECONFIG=...` as command | Export `KUBECONFIG` before running `nohup kubectl...` |
| Envoy returns `502` or hangs | Envoy proxy pod not running | `kubectl get pods -n envoy-gateway-system` |
| `curl` returns `000` (exit 7) | Port not bound — tunnel not running | Re-run Step 2 |
| Envoy NodePort changed | Service re-created after cluster restart | Re-run Step 0 |
| ArgoCD port-forward dies quickly | Pod restarted | Re-run Step 3; check `kubectl get pods -n argocd` |
