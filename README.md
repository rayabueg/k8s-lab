# SDP Kubernetes lab (Lima + Argo CD)

This lab is intentionally split into two parts:

- **Bootstrap (VM + Kubernetes)**: the [lima](lima) repo boots an Ubuntu VM with `kubeadm`, installs Cilium, and (optionally) installs Argo CD.
- **GitOps state (cluster config)**: the [gitops-lab](gitops-lab) repo is the source-of-truth that Argo CD syncs (addons, gateway resources, demo apps, etc.).

Keeping these separate makes it easy to share: your colleague can bring up their own cluster from `lima/` and point Argo CD at a fork of `gitops-lab/`.

## Quick start (for a colleague)

### 1) Bootstrap the cluster with Lima

```bash
git clone https://github.com/<you>/sdp-lab-lima-bootstrap.git
cd sdp-lab-lima-bootstrap

# Optional: rebuild VM
chmod +x rebuild-lab.sh
./rebuild-lab.sh

# Bootstrap Kubernetes + (optionally) Argo CD
chmod +x bootstrap-cluster.sh
./bootstrap-cluster.sh
```

### 2) Start the API tunnel (required for host kubectl)

In a separate terminal:

```bash
ssh -F "$HOME/.lima/sdp-lab/ssh.config" -N -L 6443:127.0.0.1:6443 lima-sdp-lab
```

Then:

```bash
export KUBECONFIG="$HOME/.kube/lima-sdp-lab"
kubectl get nodes
```

### 3) (Optional) Argo CD UI

```bash
export KUBECONFIG="$HOME/.kube/lima-sdp-lab"
kubectl -n argocd port-forward svc/argocd-server 8080:443
```

Open `https://localhost:8080`.

Get the initial admin password:

```bash
export KUBECONFIG="$HOME/.kube/lima-sdp-lab"
kubectl -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath="{.data.password}" | base64 --decode

echo
```

### 4) Point Argo CD at the GitOps repo

Fork/clone the GitOps repo:

```bash
git clone https://github.com/<you>/gitops-lab.git
cd gitops-lab
```

Edit `bootstrap/argocd/root-app.yaml` and set `spec.source.repoURL` to your repo URL, then apply:

```bash
export KUBECONFIG="$HOME/.kube/lima-sdp-lab"
kubectl apply -f bootstrap/argocd/root-app.yaml
kubectl -n argocd get applications
```

## Notes

- If you change the GitOps repo URL, make sure Argo CD’s root `Application` points at the right fork.
- Some addons (for example `external-dns`) require provider credentials/config before they should be synced.
