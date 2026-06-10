# Agent Instructions

## How to work in this repo

- This repo uses GitOps via ArgoCD — do not apply manifests manually unless explicitly told to
- All persistent changes must go through the appropriate gitops repo (`cluster-addons` or `cluster-applications`)
- This is a monorepo with git submodules; the top-level `k8s-lab/` ties together four concerns:
  - `bootstrap/` — VM provisioning (kubeadm bootstrap, Cilium, ArgoCD install); supports Lima (macOS) and Multipass (Ubuntu/Linux), with in-VM scripts shared under `bootstrap/shared/`
  - `platform-addons/` — addon manifest **library** (Gen2): one folder per addon, versioned by `platform-vX.Y.Z` git tags
  - `cluster-addons/` — cluster **config**: thin `base/` refs into platform-addons + per-cluster subscriptions
  - `cluster-applications/` — team-facing ArgoCD `Application` CRDs

## Architecture

| Layer | Tool |
|---|---|
| VM | Lima (macOS / Apple Silicon, ARM64) or Multipass (Ubuntu / Linux) — Ubuntu 24.04 guest in both |
| Kubernetes | kubeadm (single-node) |
| CNI | Cilium |
| GitOps | ArgoCD (ApplicationSet per cluster) |
| Ingress | Envoy Gateway |
| Service Mesh | Istio Ambient (in progress) |
| DNS | CoreDNS + ExternalDNS |
| Certificates | cert-manager (self-signed ClusterIssuer) |
| Secrets | External Secrets Operator |

## Repo layout conventions

- **Addons** (cluster infra) follow the Gen2 shape — library → base ref → per-cluster subscription:
  - `platform-addons/<name>/manifests/` — the actual manifests (rendered Helm output or raw YAML), in a separate library repo tagged `platform-vX.Y.Z`
  - `cluster-addons/base/<name>/kustomization.yaml` — thin remote ref: `https://github.com/rayabueg/platform-addons.git//<name>/manifests?ref=platform-vX.Y.Z`; the `?ref=` pin is the **single uprev surface**
  - `cluster-addons/clusters/k8s-lab/addons/<name>/kustomization.yaml` — subscription: `../../../../base/<name>` + per-cluster patches
  - `cluster-addons/clusters/k8s-lab/applicationset.yaml` — auto-discovers every folder under `clusters/k8s-lab/addons/*`; `clusters/k8s-lab/kustomization.yaml` applies `namesuffix: -k8s-lab`
  - `cluster-addons/clusters/_template/` — copyable skeleton for onboarding a new cluster
  - To change manifests, edit `platform-addons/<name>/manifests/` and cut a new tag; to roll a cluster forward, bump the `?ref=` in `base/<name>`
  - (The Gen1 `waves/` + in-repo `addons/` layout is retired — see `gen2-shape-migration-plan.md`)
- **New workloads**: deploy as a folder containing `namespace.yaml`, `deployment.yaml`, `service.yaml`, `httproute.yaml`, and `kustomization.yaml` — see `cluster-applications/apps-envs/mesh-demo/` as the reference pattern
- **Image-built apps** source lives in `apps/<app-name>/` at repo root (e.g. `demo-vite-ui`); each has an `ApplicationSet` in `cluster-applications/apps/<app-name>.yaml` — the list generator controls which clusters it runs on
- **Shared gateway objects** (GatewayClass, Gateway, EnvoyProxy) live in `platform-addons/envoy-gateway/manifests/gateway/`; per-app `HTTPRoute`s live with the app (or in `platform-addons/argocd-config` for the ArgoCD route)

## Patterns

- Prefer Kustomize overlays over raw manifests for anything that may differ per environment
- Use `traefik/whoami` (not `hashicorp/http-echo`) for test workloads — multi-arch safe on ARM64
- New app `HTTPRoute`s belong in the app's own addon folder, not in `envoy-gateway/gateway/`
- Self-signed TLS is issued by `cert-manager`; reference the `selfsigned-cluster-issuer` ClusterIssuer
- Gateway IP is VM-internal (not reachable from the host); test routes from inside the VM via `limactl shell k8s-lab curl ...` (Lima) or `multipass exec k8s-lab -- curl ...` (Multipass)

## Files in this directory

| File | Purpose |
|---|---|
| `context.md` | Source-of-truth lab description — paste into any agent session to orient it |
| `validations.md` | Baseline cluster health checks (nodes, pods, ArgoCD, gateway smoke test) |
| `tasks/` | One file per task; paste into agent to drive a focused work session |
