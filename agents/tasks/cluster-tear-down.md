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
- The Lima 6443 auto-forward dies with the VM and comes back on start; kill any
  `kubectl port-forward` processes before stopping (`pkill -f 'kubectl port-forward'`)

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

## Cleanup: kill port-forwards (and any legacy SSH tunnel)

The UI port-forwards are background processes. Kill them after teardown:

```bash
pkill -f "kubectl port-forward" && echo "port-forwards killed" || echo "none running"
# legacy SSH tunnel, if one was started manually:
pkill -f "6443:127.0.0.1:6443" 2>/dev/null || true
```

(The Lima 6443 auto-forward needs no cleanup — it dies with the VM.)

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
- [ ] Kill port-forwards: `pkill -f "kubectl port-forward"`
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
