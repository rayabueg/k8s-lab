# demo-vite-ui

Minimal Vite + React UI meant to be containerized and deployed to the k8s-lab cluster.

## Local dev

```bash
cd apps/demo-vite-ui
npm install
npm run dev
```

## Build & push to GHCR (public)

Pick an image name:

- `IMAGE=ghcr.io/<your-gh-username-or-org>/demo-vite-ui:0.1.0`

Login (requires a GitHub token with `write:packages`):

```bash
echo "$GHCR_TOKEN" | podman login ghcr.io -u "<your-gh-username>" --password-stdin
```

Build and push multi-arch (use absolute path — Podman on macOS requires it):

```bash
# Set IMAGE to your actual path
IMAGE=ghcr.io/<your-gh-username-or-org>/demo-vite-ui:0.1.0

podman manifest rm "$IMAGE" 2>/dev/null
podman build \
  --platform linux/amd64,linux/arm64 \
  --manifest "$IMAGE" \
  /path/to/k8s-lab/apps/demo-vite-ui
podman manifest push "$IMAGE"
```

> **Note:** The build stage uses `--platform=$BUILDPLATFORM` so the Node.js/esbuild compile step always runs on your Mac's native architecture. Only the final Nginx image is cross-compiled.

## Deploy to the cluster

1) Edit the image in the Kubernetes manifest:

- [gitops-lab/clusters/k8s-lab/gateway/demo-vite-ui.yaml](../../gitops-lab/clusters/k8s-lab/gateway/demo-vite-ui.yaml)

2) Apply it:

```bash
export KUBECONFIG="$HOME/.kube/lima-k8s-lab"
kubectl apply -f gitops-lab/clusters/k8s-lab/gateway/demo-vite-ui.yaml
```

3) Test via Envoy Gateway:

```bash
curl -i http://127.0.0.1:30080/vite/
```
