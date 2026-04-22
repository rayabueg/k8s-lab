# Kubernetes lab (Lima + Argo CD)

This lab is intentionally split into two parts:

- **Bootstrap (VM + Kubernetes)**: the [lima](lima) repo boots an Ubuntu VM with `kubeadm`, installs Cilium, and (optionally) installs Argo CD.
- **GitOps state (cluster config)**: the [gitops-lab](gitops-lab) repo is the source-of-truth that Argo CD syncs (addons, gateway resources, demo apps, etc.).

Keeping these separate makes it easy to share: your colleague can bring up their own cluster from `lima/` and point Argo CD at a fork of `gitops-lab/`.

## Quick start (for a colleague)

This repo uses git submodules so you can clone everything in one shot.

```bash
git clone --recurse-submodules https://github.com/rayabueg/k8s-lab.git
cd k8s-lab

# If you forgot --recurse-submodules:
git submodule update --init --recursive
```

### 1) Bootstrap the cluster with Lima

```bash
cd lima

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
ssh -F "$HOME/.lima/k8s-lab/ssh.config" -N -L 6443:127.0.0.1:6443 lima-k8s-lab
```

Then:

```bash
export KUBECONFIG="$HOME/.kube/lima-k8s-lab"
kubectl get nodes
```

### 3) (Optional) Argo CD UI

```bash
export KUBECONFIG="$HOME/.kube/lima-k8s-lab"
kubectl -n argocd port-forward svc/argocd-server 8080:443
```

Open `https://localhost:8080`.

Get the initial admin password:

```bash
export KUBECONFIG="$HOME/.kube/lima-k8s-lab"
kubectl -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath="{.data.password}" | base64 --decode

echo
```

### 4) Point Argo CD at the GitOps repo

The GitOps repo is included as the `gitops-lab/` submodule.

If you’re using a fork, update `gitops-lab/bootstrap/argocd/root-app.yaml` so `spec.source.repoURL` points at your fork.

From this repo root:

```bash
cd gitops-lab
```

Edit `bootstrap/argocd/root-app.yaml` and set `spec.source.repoURL` to your repo URL:

```bash
# Quick one-liner (replace with your actual fork URL)
sed -i '' 's|https://github.com/rayabueg/gitops-lab.git|https://github.com/<you>/gitops-lab.git|' \
  bootstrap/argocd/root-app.yaml
```

Then apply:

```bash
export KUBECONFIG="$HOME/.kube/lima-k8s-lab"
kubectl apply -f bootstrap/argocd/root-app.yaml
kubectl -n argocd get applications
```

## Notes

- If you change the GitOps repo URL, make sure Argo CD’s root `Application` points at the right fork.
- Some addons (for example `external-dns`) require provider credentials/config before they should be synced.

## Maintaining this repo (submodules)

This repo is a **parent** that pins exact commits of its submodules (especially `gitops-lab/`).
That means updates are a 2-step process:

This repo currently pins two submodules:

- GitOps state: [gitops-lab/README.md](gitops-lab/README.md) (see the “Contributing” section)
- Bootstrap scripts: [lima/README.md](lima/README.md) (see the “Contributing” section)

### Why submodules (for this repo)

We use submodules to get **tight coupling of versions** without turning everything into a monorepo:

- **Tight coupling (pinned versions):** the parent repo records the exact `gitops-lab/` commit SHA that the lab expects.
  Everyone who checks out the parent + runs `git submodule update` gets the same GitOps state.
- **Separation of concerns:** bootstrap tooling (Lima VM + kubeadm) and GitOps state evolve at different speeds and often have different reviewers/owners.
  Keeping them as separate repos reduces noise and keeps changes focused.
- **History tracking:** content changes land as normal commits in the submodule; the parent repo only records explicit “pointer bump” commits.
  This makes it easy to audit “what changed” vs “which version we pinned” and to roll back by resetting the submodule pointer.

1) **Commit + push changes in the submodule repo first**
2) **Commit + push the parent repo “pointer bump”** (the gitlink change)

### Pull the latest pinned submodule revisions

When you pull parent changes, also refresh submodules to the pinned commits:

```bash
git pull
git submodule update --init --recursive
```

### Update the `gitops-lab/` submodule (recommended workflow)

Make changes inside the submodule and push them:

```bash
cd gitops-lab

# If you plan to commit, work on a branch (submodules are often checked out detached)
git switch -c <branch-name> || git switch <branch-name>

git status
git add -A
git commit -m "<scope>: <summary>"
git push -u origin <branch-name>
```

Then bump the parent repo pointer and push it:

```bash
cd ..
git status
git add gitops-lab

# Commit records the new submodule SHA in the parent
git commit -m "submodule(gitops-lab): bump to $(cd gitops-lab && git rev-parse --short HEAD)"
git push
```

### Optional: move submodules to latest remote commits

Use this only if you explicitly want to advance the submodule to whatever its remote branch currently points at:

```bash
git submodule update --remote --recursive
git add gitops-lab
git commit -m "submodule(gitops-lab): update to latest remote"
git push
```

## Commit message standard (recommended)

Keep messages short and scannable; use a simple `scope: summary` format:

- In `gitops-lab/`: use a real scope like `addons: ...`, `k8s-lab: ...`, `gateway: ...`, `bootstrap: ...`
  - Example: `addons: add descheduler and core-dns overrides`
- In the parent repo: only use submodule-pointer commits
  - Format: `submodule(gitops-lab): bump to <sha> (reason)`
  - Example: `submodule(gitops-lab): bump to 1994eb1 (add descheduler + core-dns)`

This makes it obvious which commits changed **content** (submodule) vs which commits just changed the **pinned version** (parent).
