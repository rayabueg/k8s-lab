# Migration plan — reshape the lab's `cluster-addons` to the Gen2 shape

> Goal: make `k8s-lab/cluster-addons` *structurally* look like
> `Alaska-Airlines-Shared/sdp-cluster-addons` (Gen2), adapted to a single-node
> ARM64 Lima cluster. This is a plan only — no files are changed yet.
> Reference shape lives at `gen2/sdp-cluster-addons/` (read-only mount).

## TL;DR

Gen2 replaces the Gen1 **wave model** with three ideas:

1. `base/<addon>/` is a thin **remote ref** to an external addon library, pinned
   at `?ref=platform-vX.Y.Z`. It is the *single uprev surface*.
2. `clusters/<cluster>/addons/<addon>/` is a per-cluster **subscription** that
   references `../../../../base/<addon>` plus cluster-local patches.
3. Promotion is **branch merge-up** (`development → test → qa → … → main`), not
   wave folders. `clusters/_template/` is the copyable onboarding shape.

The lab already has #2 in embryonic form (it has `clusters/k8s-lab/addons/*`).
The work is: collapse the `waves/` indirection into a `base/` uprev surface,
introduce `_template`, decide on a library model, and prune/rename the addon set
to what a single-node lab actually runs. Branch-per-env is optional at lab scale.

---

## 1. The two shapes side by side

### Current lab (Gen1 waves)

```
cluster-addons/
  addons/<addon>/
    base/{stable,latest,rc}/          # rendered Helm output (bundle.yaml) per channel
    hack/{render.sh,promote.sh,values*.yaml}
  waves/
    wave1/<addon>/kustomization.yaml  # → ../../../addons/<addon>/base/stable
    wave2/<addon>/kustomization.yaml  # → ../../../addons/<addon>/base/latest
  clusters/k8s-lab/
    applicationset.yaml               # git generator over clusters/k8s-lab/addons/*
    kustomization.yaml
    addons/<addon>/kustomization.yaml # → ../../../../waves/wave1/<addon>
  bootstrap/argocd/root-app.yaml
```

Flow today: `render.sh` regenerates `base/latest`; `promote.sh` copies
`latest → stable`; `wave1` pins stable, `wave2` pins latest; each cluster addon
selects a wave. Two indirections (`cluster → wave → channel`) sit in front of the
manifests.

### Target Gen2

```
sdp-cluster-addons/
  base/<addon>/kustomization.yaml     # remote ref to sdp-addons lib at ?ref=platform-vX.Y.Z
  clusters/<cluster>/
    applicationset.yaml               # git generator over clusters/<cluster>/addons/*
    kustomization.yaml                # namesuffix: -<cluster>; wraps the AppSet
    addons/<addon>/kustomization.yaml # → ../../../../base/<addon> + per-cluster patches
  clusters/_template/                 # copyable cluster skeleton
```

Manifests live in a *separate* repo (`sdp-addons`); `base/` only holds the pinned
ref. One indirection (`cluster → base`), and the pin tag is the only thing
automation moves. Promotion is by merging the `base/*` `?ref=` bump up the branch
chain; per-cluster patches stay divergent across branches by design.

### Structural deltas

| Concern | Lab today | Gen2 | Adaptation for the lab |
|---|---|---|---|
| Manifest source | In-repo rendered `bundle.yaml` | External `sdp-addons` lib | **Keep in-repo** — no library repo exists. `base/<addon>` refs the local rendered channel. |
| Uprev surface | `wave{1,2}` channel selection + `promote.sh` | `base/<addon>` `?ref=` tag | `base/<addon>` → `addons/<addon>/base/stable`; `promote.sh` is the local "uprev". |
| Indirection depth | cluster → wave → channel | cluster → base | Collapse waves; one hop. |
| Promotion | wave1 (stable) / wave2 (latest) | branch merge-up | Optional 2-branch (`development`/`main`); single branch is fine at lab scale. |
| Onboarding | none (`k8s-lab` is hand-rolled) | `clusters/_template/` | Add `_template` so a 2nd cluster (e.g. multipass) is a copy. |
| Cluster count | 1 (`k8s-lab`) | 15 nonprod | Keep 1; `namesuffix` + `_template` make N trivial. |
| Addon→cluster matching | implicit (all addons, one cluster) | tag sets (`application`/`utility`/`kubeadm`/`appservice`) | Encode lab cluster as `kubeadm application` tags; document which addons land. |

---

## 2. ApplicationSet deltas — keep the lab's, don't copy Gen2's

The lab's `applicationset.yaml` carries hard-won single-node correctness that the
Gen2 AppSet does **not** have. Preserve these when reshaping; only the
`directories[].path` and naming need touching.

| Field | Lab (keep) | Gen2 | Why the lab differs |
|---|---|---|---|
| `destination` | `server: https://kubernetes.default.svc` | `name: <cluster>` | Lab Argo manages its own cluster in-process; Gen2 registers external AKS/on-prem clusters by name. |
| Apply strategy | **client-side** (SSA omitted) | `ServerSideApply=true` | Lab relies on `ignoreDifferences` for runtime-injected fields; SSA breaks that workaround. |
| `ignoreDifferences` | CRD `selectableFields`, webhook `caBundle`/`failurePolicy`, `last-applied-configuration`, `/status` | none | k8s 1.30+ and cert-manager/istiod mutate these at runtime; without the ignores Argo never reaches Synced. |
| `syncOptions` | `CreateNamespace=true`, `SkipDryRunOnMissingResource=true` | `ServerSideApply=true` | Lab has no namespace-provisioning addon path; Gen2 clusters get namespaces from the library. |
| AppSet/App name | `cluster-addons-k8s-lab`, `{{path.basename}}-k8s-lab` | `cluster`, `{{path.basename}}-<cluster>` | Cosmetic; align to `{{path.basename}}-<cluster>` if adopting `namesuffix`. |

**Action:** the `_template` AppSet should be the *lab's* AppSet with `k8s-lab`
replaced by a `__CLUSTER_NAME__` placeholder — not a copy of the Gen2 file.

---

## 3. Addon disposition (adapted to single-node ARM64)

Gen2 ships 27 base addons gated by tag sets. The lab cluster is effectively
`kubeadm application` (on-prem-style, single node, hosts its own Argo). Mapping:

### Keep — direct lab equivalent already exists

| Gen2 addon | Lab addon | Note |
|---|---|---|
| `cert-manager` | `cert-manager` | 1:1 |
| `core-dns` | `core-dns` | 1:1 (custom DNS ConfigMap) |
| `crds` | `crds` | 1:1 (Gateway API + Envoy Gateway CRDs) |
| `descheduler` | `descheduler` | 1:1 |
| `external-dns` | `external-dns` | 1:1 |
| `external-secrets` | `external-secrets` | 1:1 |

### Keep but restructure (split / rename to match Gen2 granularity)

| Gen2 | Lab today | Change |
|---|---|---|
| `istio-base`, `istio-cni`, `istiod`, `istio-ztunnel` (4 addons) | single `istio` addon (one bundle, `values-cni.yaml`/`values-istiod.yaml`) | **Optional split** into 4 `base/` entries for independent uprev + correct sync ordering. Lower priority — the lab's Ambient bundle works as one unit. |
| `envoy-gateway-operator` (controller, all clusters) + `envoy-gateways` (Gateway/EnvoyProxy, `application` only) | single `envoy-gateway` (Gateway + EnvoyProxy only) | **Rename/split**: lab's current addon ≈ `envoy-gateways` (data plane). The operator install must be accounted for (today implicit via `crds`/bootstrap). Mirror the operator-vs-instances split if you want fidelity. |
| `cilium` (addon) | installed at **bootstrap** (`bootstrap/*/bootstrap-cluster.sh`), `hubble` is the addon | Lab bootstraps Cilium before Argo exists (it's the CNI — chicken/egg). **Keep at bootstrap.** Optionally add a `cilium` base that only manages config drift, but do not move CNI install into Argo. |

### Lab-only — no Gen2 equivalent, keep as-is

| Lab addon | Disposition |
|---|---|
| `hubble` | Lab observability; Gen2 folds Hubble into `cilium`. Keep as a standalone lab addon. |
| `argocd-config` | Insecure-mode ConfigMap + Argo HTTPRoute. Closest Gen2 analog is `argocd-resources` (utility clusters only). Keep; optionally rename to `argocd-resources` for naming parity. |
| `namespaces` | Gen2 ships namespaces inside each library addon; lab centralizes them. Keep until/unless addons own their namespaces. |

### Drop — AKS / enterprise-on-prem only, no place on a single-node lab

`aqua-enforcer`, `aqua-kube-enforcer` (Aqua runtime security), `azwi-webhook`
(Azure Workload Identity), `dynatrace-operator` (APM SaaS), `trident` (NetApp
storage CSI), `metallb` (bare-metal LB — single node needs none),
`kured` (reboot daemon — pointless on one node), `argocd`/`argocd-resources`
(Gen2 utility clusters *host* Argo for others; the lab's Argo is bootstrapped).

### Optional adds — lab-relevant and cheap on one node

`metrics-server` (HPA/`kubectl top`), `kube-state-metrics` (metrics), `vpa`
(right-sizing study), `keda` (event-driven autoscaling demos),
`node-local-dns` (DNS caching). Add only if they serve a study goal — each is a
`base/<addon>` + a `clusters/k8s-lab/addons/<addon>` subscription.

> **Tag encoding for the lab:** treat `k8s-lab` as carrying tags
> `application kubeadm`. That admits the istio stack + envoy-gateways
> (`application`) and the kubeadm-only set *minus* the ones dropped above for
> single-node reasons. Document this in the cluster's README so future addons
> resolve unambiguously.

---

## 4. The library question (decide first — everything hangs off it)

Gen2's `base/<addon>` points at a *separate* `sdp-addons` repo at a version tag.
The lab has no such repo; its manifests are rendered in-tree. Two options:

- **Option A — local library (recommended).** Keep rendered bundles where they
  are. `base/<addon>/kustomization.yaml` becomes a thin ref to the pinned
  channel: `resources: [../../addons/<addon>/base/stable]`. `promote.sh`
  (`latest → stable`) *is* the uprev. One repo, one branch, minimal churn.
  Preserves the lab's offline/self-contained property.

- **Option B — extract a library repo.** Split `addons/*/base/*` into a separate
  `rayabueg/sdp-addons`-style repo, tag it `platform-vX.Y.Z`, and make
  `base/<addon>` a true remote ref. High fidelity to Gen2, but adds a second repo
  + submodule + tagging workflow for a one-node lab. Only worth it if studying
  the *cross-repo uprev automation* is itself a goal.

This plan assumes **Option A**.

---

## 5. Target lab layout (Option A)

```
cluster-addons/
  base/<addon>/kustomization.yaml       # NEW: thin ref → ../../addons/<addon>/base/stable
  addons/<addon>/                       # UNCHANGED: rendered channels + hack/
    base/{stable,latest,rc}/
    hack/{render.sh,promote.sh,values*.yaml}
  clusters/
    _template/                          # NEW: copyable skeleton
      applicationset.yaml               # lab AppSet with __CLUSTER_NAME__
      kustomization.yaml                # namesuffix: -__CLUSTER_NAME__
      addons/<addon>/kustomization.yaml # → ../../../../base/<addon>
      README.md
    k8s-lab/
      applicationset.yaml               # path → clusters/k8s-lab/addons/*  (keep lab fields)
      kustomization.yaml                # namesuffix: -k8s-lab
      addons/<addon>/kustomization.yaml # → ../../../../base/<addon> + patches
  bootstrap/argocd/root-app.yaml        # UNCHANGED
  # waves/  ← DELETED
```

### Concrete file contents

`base/cert-manager/kustomization.yaml` (one per addon; the uprev surface):

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

# Local addon "library": the pinned channel is the uprev surface.
# `promote.sh` (latest → stable) is the lab equivalent of bumping ?ref=.
resources:
  - ../../addons/cert-manager/base/stable
```

`clusters/k8s-lab/addons/cert-manager/kustomization.yaml` (the subscription):

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../../../base/cert-manager

# patches: []   # cluster-specific patches go here
```

> Relative-ref depth check: from `clusters/k8s-lab/addons/<addon>/`, four `..`
> reach the repo root, then `base/<addon>` — matches Gen2's `../../../../base/<addon>`.
> From `base/<addon>/`, two `..` reach root, then `addons/<addon>/base/stable`.

`clusters/_template/kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namesuffix: -__CLUSTER_NAME__

resources:
  - applicationset.yaml
```

---

## 6. Phased execution

Each phase is independently committable and leaves the cluster reconcilable.

**Phase 0 — Decide.** Confirm Option A (local library) and the addon disposition
in §3. No file changes. *(Blocks everything.)*

**Phase 1 — Introduce `base/`.** Add `base/<addon>/kustomization.yaml` for every
kept addon, each ref'ing `../../addons/<addon>/base/stable`. Purely additive;
nothing consumes it yet. Validate `kustomize build base/<addon>` per addon.

**Phase 2 — Repoint subscriptions.** Rewrite each
`clusters/k8s-lab/addons/<addon>/kustomization.yaml` from
`../../../../waves/wave1/<addon>` to `../../../../base/<addon>`. Diff the rendered
output before/after — it must be identical (both ultimately resolve to
`base/stable`). This is the cutover; Argo sees no manifest change.

**Phase 3 — Retire waves.** Delete `waves/`. Confirm nothing references it
(`grep -r waves/ cluster-addons`). Update `cluster-addons/README.md`.

**Phase 4 — Add `_template`.** Create `clusters/_template/` as a placeholder-ised
copy of `clusters/k8s-lab/` (AppSet with `__CLUSTER_NAME__`, `namesuffix`,
`addons/<addon>` stubs, README cloned from Gen2's `_template/README.md` but
pointing at the lab's repo + branch and the lab's apply semantics).

**Phase 5 — Optional restructure.** Split `istio` → 4 base entries; split
`envoy-gateway` → `envoy-gateway-operator` + `envoy-gateways`; add any §3
optional addons. Each is its own PR-sized change.

**Phase 6 — Optional branches.** If you want to study merge-up promotion, add a
`development` branch; `base/*` channel bumps land there first, then merge to
`main`. Skip if single-branch is sufficient.

---

## 7. Validation

Same gates Gen2's `AGENTS.md` prescribes, runnable locally:

```bash
# every cluster builds
for c in clusters/*/; do
  [ "$(basename "$c")" = "_template" ] && continue   # _template has placeholders
  kustomize build "$c" >/dev/null && echo "ok: $c" || echo "FAIL: $c"
done

# every base builds
for b in base/*/; do
  kustomize build "$b" >/dev/null && echo "ok: $b" || echo "FAIL: $b"
done

# Phase 2 equivalence — rendered output must not change at cutover
git stash; kustomize build clusters/k8s-lab > /tmp/before.yaml; git stash pop
kustomize build clusters/k8s-lab > /tmp/after.yaml
diff /tmp/before.yaml /tmp/after.yaml && echo "no drift"
```

`_template` won't `kustomize build` until placeholders are substituted — that's
expected; exclude it from the loop (Gen2 does the same).

Then the operational check: Argo CD shows every `*-k8s-lab` Application
`Synced/Healthy` after each phase, and `kubectl -n argocd get applicationset`
still lists the single AppSet.

---

## 8. Open questions for Ray

1. **Library model** — confirm Option A (local), or do you want to study the
   cross-repo `sdp-addons` extraction (Option B)?
2. **Istio split** — split into 4 base addons now (fidelity) or keep the single
   Ambient bundle (less churn)? Recommended: defer to Phase 5.
3. **Branches** — single `main`, or add `development` to exercise merge-up?
4. **Optional addons** — want `metrics-server`/`kube-state-metrics`/`vpa`/`keda`
   added as part of the reshape, or keep scope to a pure structural migration?
```
