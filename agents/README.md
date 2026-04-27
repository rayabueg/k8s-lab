# Agent Instructions

## How to work in this repo

- This repo uses GitOps via ArgoCD — do not apply manifests manually unless explicitly told to
- All persistent changes must go through the appropriate gitops repo (`cluster-addons` or `cluster-applications`)
- This is a monorepo with git submodules; the top-level `k8s-lab/` ties together three concerns:
  - `lima/` — VM provisioning (kubeadm bootstrap, Cilium, ArgoCD install)
  - `cluster-addons/` — cluster infrastructure: addons, CRDs, gateways, namespaces
  - `cluster-applications/` — team-facing ArgoCD `Application` CRDs

## Architecture

| Layer | Tool |
|---|---|
| VM | Lima (Ubuntu, ARM64 / Apple Silicon) |
| Kubernetes | kubeadm (single-node) |
| CNI | Cilium |
| GitOps | ArgoCD (ApplicationSet per cluster) |
| Ingress | Envoy Gateway |
| Service Mesh | Istio Ambient (in progress) |
| DNS | CoreDNS + ExternalDNS |
| Certificates | cert-manager (self-signed ClusterIssuer) |
| Secrets | External Secrets Operator |

## Repo layout conventions

- **Addons** (cluster infra): `cluster-addons/clusters/k8s-lab/addons/<addon-name>/`
  - Each addon folder contains a `kustomization.yaml` listing its resources
  - The `applicationset.yaml` at `clusters/k8s-lab/` auto-discovers every folder under `addons/*`
  - Addon folders can contain: raw manifests, ArgoCD `Application` CRDs (for Helm charts), or both
- **New workloads**: deploy as an addon folder containing `namespace.yaml`, `deployment.yaml`, `service.yaml`, `httproute.yaml`, and `kustomization.yaml` — see `addons/mesh-demo/` as the reference pattern
- **Image-built apps** source lives in `apps/<app-name>/` at repo root (e.g. `demo-vite-ui`); each has an `ApplicationSet` in `cluster-applications/apps/<app-name>.yaml` — the list generator controls which clusters it runs on
- **Shared gateway objects** (GatewayClass, Gateway, EnvoyProxy) live under `cluster-addons/clusters/k8s-lab/addons/envoy-gateway/gateway/`; per-app `HTTPRoute`s live in the app's own addon folder

## Patterns

- Prefer Kustomize overlays over raw manifests for anything that may differ per environment
- Use `traefik/whoami` (not `hashicorp/http-echo`) for test workloads — multi-arch safe on ARM64
- New app `HTTPRoute`s belong in the app's own addon folder, not in `envoy-gateway/gateway/`
- Self-signed TLS is issued by `cert-manager`; reference the `selfsigned-cluster-issuer` ClusterIssuer
- Gateway IP is VM-internal (not reachable from Mac host); use `limactl shell k8s-lab curl ...` to test routes

## Files in this directory

| File | Purpose |
|---|---|
| `context.md` | Source-of-truth lab description — paste into any agent session to orient it |
| `validations.md` | Baseline cluster health checks (nodes, pods, ArgoCD, gateway smoke test) |
| `tasks/` | One file per task; paste into agent to drive a focused work session |
