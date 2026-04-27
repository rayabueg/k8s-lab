# Task: Tear Down the Lab

> **Context**: Read `agents/context.md` first.

## Objective

Cleanly stop or fully destroy the k8s-lab. Choose the level of teardown based on intent.

---

## Option A: Suspend (stop VM, keep state)

Use this when you want to free up Mac resources but resume the cluster later without rebuilding.

```bash
limactl stop k8s-lab
```

- VM is stopped, disk image is preserved
- Resume with `./bootstrap-cluster.sh` (see `tasks/cluster-start-up.md` Step 1a)
- SSH tunnel process dies automatically; you'll need to restart it on resume

---

## Option B: Full destroy (delete VM + disk)

Use this to reclaim disk space or start fresh. **This is irreversible.**

```bash
limactl delete -f k8s-lab
```

- Deletes the VM and its 60 GiB disk image
- All cluster state is gone (no etcd, no PVs, no secrets)
- Next start requires full rebuild (`tasks/cluster-start-up.md` Step 1b, ~15 min)

---

## Cleanup: kill the SSH tunnel

The API tunnel (`ssh -N -L 6443:...`) is a background process. Kill it after teardown:

```bash
# Find and kill the tunnel
pkill -f "6443:127.0.0.1:6443" && echo "Tunnel killed" || echo "No tunnel running"
```

---

## Cleanup: stale kubeconfig (optional)

After a full destroy, the exported kubeconfig is stale. Remove it to avoid confusion:

```bash
rm -f ~/.kube/lima-k8s-lab
```

---

## Full teardown checklist

- [ ] Decide: suspend (Option A) or destroy (Option B)
- [ ] Run `limactl stop k8s-lab` or `limactl delete -f k8s-lab`
- [ ] Kill SSH tunnel: `pkill -f "6443:127.0.0.1:6443"`
- [ ] (Destroy only) Remove kubeconfig: `rm -f ~/.kube/lima-k8s-lab`
- [ ] Verify: `limactl list` shows `Stopped` or no instance

---

## Troubleshooting

| Symptom | Fix |
|---|---|
| `limactl delete` hangs | Force: `limactl delete -f k8s-lab` |
| VM stuck in `Stopping` state | `limactl stop --force k8s-lab` |
| Tunnel process not found | Already dead — nothing to do |
| Disk space not reclaimed after delete | Check `~/.lima/k8s-lab/` is gone; `rm -rf ~/.lima/k8s-lab` if residue remains |
