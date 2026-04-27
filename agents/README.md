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
  - Each addon folder contains a `kustomization.yaml` and optionally an ArgoCD `application.yaml`
  - The root `applicationset.yaml` at `clusters/k8s-lab/` auto-discovers addons
- **Apps** (workloads): `cluster-applications/apps/<app-name>.yaml`
  - One ArgoCD `Application` manifest per app
- **Demo workloads** source lives in `apps/<app-name>/` at repo root

## Patterns

- Prefer Kustomize overlays over raw manifests for anything that may differ per environment
- Use `traefik/whoami` (not `hashicorp/http-echo`) for test workloads — multi-arch safe on ARM64
- Gateway API (`HTTPRoute`, `Gateway`) objects live under `cluster-addons/clusters/k8s-lab/addons/envoy-gateway/gateway/`
- Self-signed TLS is issued by `cert-manager`; reference the `selfsigned-cluster-issuer` ClusterIssuer

## Files in this directory

| File | Purpose |
|---|---|
| `context.md` | Source-of-truth lab description — paste into any agent session to orient it |
| `validations.md` | Definition-of-done contracts per feature area |
| `tasks/` | One file per task; paste into agent to drive a focused work session |
