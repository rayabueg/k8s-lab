# Lab Context

> Paste this file into any agent session to orient it before starting a task.

## What this lab is

A personal Kubernetes lab running on a MacBook (Apple Silicon / ARM64) used to simulate
an internal developer platform and study service mesh / platform engineering patterns.
The long-term goal is to prototype an architecture similar to a Software Delivery Platform (SDP)
using open-source components before applying the same patterns at scale.

## Infrastructure

| Component | Detail |
|---|---|
| Host | macOS (Apple Silicon, ARM64) |
| VM | Lima (`k8s-lab`) — Ubuntu 24.04 LTS |
| Kubernetes | kubeadm, single-node (control-plane acts as worker) |
| Container runtime | containerd |
| CNI | Cilium (eBPF, kube-proxy replacement) |
| DNS | CoreDNS with custom local entries |
| TLS | cert-manager with self-signed ClusterIssuer |
| Secrets | External Secrets Operator |
| GitOps | ArgoCD (ApplicationSet auto-discovers addons) |
| Ingress | Envoy Gateway (Gateway API) |
| Service Mesh | Istio Ambient — **in progress** |
| Node scheduling | Descheduler |

## Network / access

- The cluster API runs inside the Lima VM; a local SSH tunnel exposes it on `127.0.0.1:6443`
- `KUBECONFIG=$HOME/.kube/lima-k8s-lab`
- Container images **must** be multi-arch or ARM64-native (the node is ARM64)

## Repo structure

```
k8s-lab/                        ← monorepo root (git submodules)
  lima/                         ← VM provisioning (kubeadm, Cilium, ArgoCD bootstrap)
  cluster-addons/               ← cluster infrastructure (GitOps source of truth)
    clusters/k8s-lab/
      applicationset.yaml       ← ArgoCD auto-discovers addons here
      addons/
        cert-manager/
        cilium/
        core-dns/
        crds/
        descheduler/
        envoy-gateway/
          gateway/              ← HTTPRoute / Gateway objects
        external-dns/
        external-secrets/
        namespaces/
        istio/                  ← (planned)
  cluster-applications/         ← team app ArgoCD Applications
    apps/
  apps/                         ← demo workload source code
    demo-vite-ui/
  agents/                       ← this directory
```

## Design decisions

- **No manual `kubectl apply`** — everything is GitOps via ArgoCD unless experimenting
- **Kustomize** is the default; Helm is used when upstream charts are pulled in via ArgoCD
- **Envoy Gateway** is the sole ingress; no Istio ingress gateway will be installed
- **Istio Ambient** (sidecar-less) is preferred over Istio sidecar mode to keep resource usage low on a single-node VM
- **Self-signed certs** are fine for the lab; a real issuer would be swapped in for production

## Current state (as of April 2026)

- [x] Lima VM running, kubeadm cluster healthy
- [x] Cilium CNI installed
- [x] ArgoCD managing cluster-addons via ApplicationSet
- [x] Envoy Gateway + Gateway API routes working
- [x] cert-manager, ExternalDNS, External Secrets installed
- [x] demo-vite-ui deployed and accessible
- [ ] Istio Ambient — not yet installed
- [ ] mTLS enforcement between workloads — blocked on Istio
