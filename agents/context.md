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
| Host | macOS (Apple Silicon, ARM64) — primary; Ubuntu / Linux also supported |
| VM | Lima (`k8s-lab`, macOS) or Multipass (`k8s-lab`, Linux) — Ubuntu 24.04 LTS guest in both |
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

- The cluster API runs inside the VM; on Lima the hostagent **auto-forwards** it to `127.0.0.1:6443` (no manual SSH tunnel needed — verify with `lsof -nP -iTCP:6443 -sTCP:LISTEN`, the listener is the `limactl` process)
- `KUBECONFIG=$HOME/.kube/lima-k8s-lab` (Lima) or `$HOME/.kube/multipass-k8s-lab` (Multipass)
- All browser UIs go through a single port-forward of the Envoy Gateway service:
  `kubectl port-forward -n envoy-gateway-system svc/envoy-envoy-gateway-system-eg-<hash> 8080:8080 12000:12000 9080:9080`
  — `8080` apps (`/vite/`, `/hello`, …), `12000` Hubble, `9080` **ArgoCD** (plain HTTP; argocd-server runs `server.insecure=true` and is routed via an `HTTPRoute` on the gateway's `argocd` listener)
- The Envoy service is **NodePort** (pinned `80:30080` via the EnvoyProxy patch; other NodePorts are random per service recreation)
- Container images **must** be multi-arch or ARM64-native when running on Apple Silicon (the node is ARM64); x86_64 hosts under Multipass are also supported

## Repo structure

```
k8s-lab/                                  ← monorepo root (git submodules)
  bootstrap/                              ← VM provisioning (kubeadm, Cilium, ArgoCD bootstrap)
    lima/                                 ← macOS / Apple Silicon host
    multipass/                            ← Ubuntu / Linux host
    shared/                               ← in-VM scripts shared by both
  platform-addons/                        ← addon LIBRARY (manifests, versioned by git tag) — Gen2 shape
    <name>/manifests/                     ← addon's kustomize-buildable manifests (rendered Helm / raw YAML)
    <name>/hack/                          ← render.sh + values (Helm addons); regenerates manifests/
    <name>/metadata.yaml                  ← spec.clusters tag eligibility
    # tags: platform-vX.Y.Z freeze every addon together; the uprev unit
  cluster-addons/                         ← cluster-config (GitOps source of truth) — Gen2 shape
    base/<name>/                          ← remote ref → platform-addons//<name>/manifests?ref=platform-vX.Y.Z (uprev surface)
    clusters/_template/                   ← copyable per-cluster skeleton (__CLUSTER_NAME__/__BRANCH__)
    clusters/k8s-lab/
      applicationset.yaml                 ← ArgoCD auto-discovers clusters/k8s-lab/addons/*
      kustomization.yaml                  ← namesuffix: -k8s-lab
      addons/<name>/                      ← subscription → ../../../../base/<name> + per-cluster patches
    # addon set: argocd-config, cert-manager, core-dns, crds, descheduler,
    # envoy-gateway (incl. gateway/ HTTPRoute & Gateway objects), external-dns,
    # external-secrets, hubble, istio, namespaces
    # NOTE: Gen1 waves/ + in-repo addons/ retired; manifests moved to platform-addons
  cluster-applications/                   ← team app ArgoCD Applications
    apps/                                 ← ArgoCD Application CRDs
    apps-envs/                            ← per-app manifests (demo-vite-ui, gateway-demos, mesh-demo)
  apps/                                   ← demo workload source code
    demo-vite-ui/
  agents/                                 ← this directory
```

## Design decisions

- **No manual `kubectl apply`** — everything is GitOps via ArgoCD unless experimenting
- **Kustomize** is the default; Helm is used when upstream charts are pulled in via ArgoCD
- **Envoy Gateway** is the sole ingress; no Istio ingress gateway will be installed
- **Istio Ambient** (sidecar-less) is preferred over Istio sidecar mode to keep resource usage low on a single-node VM
- **Self-signed certs** are fine for the lab; a real issuer would be swapped in for production

## Current state (as of June 2026)

- [x] VM running (Lima on macOS, Multipass on Linux), kubeadm cluster healthy
- [x] Cilium CNI installed; Hubble UI exposed at `:12000`
- [x] ArgoCD managing cluster-addons via ApplicationSet
- [x] Envoy Gateway + Gateway API routes working
- [x] cert-manager, ExternalDNS, External Secrets installed
- [x] demo-vite-ui deployed and accessible
- [~] Istio Ambient — manifests in `platform-addons/istio/`; validation in progress (see `tasks/cluster-addon-validate-istio.md`)
- [ ] mTLS enforcement between workloads — pending Istio validation
- [x] **Gen2 shape migration** — live on the cluster as of 2026-06-09: `platform-addons` pushed + tagged `platform-v0.1.0`, cluster-addons PR #1 (`gen2-shape`) merged to main, all 16 ArgoCD apps Synced/Healthy on the new shape. Plan: `gen2-shape-migration-plan.md`
  - Known cosmetic issue: the live ApplicationSet is named `cluster-addons-k8s-lab-k8s-lab` (file kept the old `-k8s-lab` name AND the new `namesuffix: -k8s-lab` applies — the §2 naming note in the plan). Renaming it will cascade-prune all addon apps; if done, strip the apps' `resources-finalizer` first (see `tasks/cluster-start-up.md` troubleshooting)
