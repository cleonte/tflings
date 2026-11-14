# Spike 07 — plugin cache, offline `init`, plan per save

Run 2026-11-14. Terraform 1.12.2 and 1.17.2, hashicorp/aws 6.73.0. Offline `init` tests: Debian 13 container (linux/arm64), provider already in the shared cache volume, then `/etc/resolv.conf` replaced by an unreachable nameserver (`192.0.2.1`). Timing tests: native macOS arm64, Floci 2.4.0 in Docker, warm cache, endpoints as in spike 04. Scripts: `offline-init.sh`, `timing.sh`.

## What the plugin cache is

The AWS provider is one 729 MB program. Every exercise folder needs it for `init`. With `TF_PLUGIN_CACHE_DIR` set, Terraform keeps one copy in that folder and links to it from each exercise's `.terraform`, so about 120 exercises share one copy and a later `init` takes about 1.3 s instead of about 11 s (spike 01).

## Offline `init`

`terraform init` with the cache filled but **no network** (Terraform latest and 1.12, same results):

| Case | Result |
|---|---|
| A. cache filled, no lock file | **FAIL**: `Failed to query available provider packages ... could not connect to registry.terraform.io: failed to request discovery` (1.1 s on 1.12, 4.8 s on 1.17.2) |
| B. cache filled + full lock file (h1 and zh hashes) | **FAIL**, same error |
| C. cache filled + lock file with only `zh:` hashes | **FAIL**, same error |
| D. C + `TF_PLUGIN_CACHE_MAY_BREAK_DEPENDENCY_LOCK_FILE=true` | **FAIL**, same error |
| E. empty cache + full lock file | **FAIL**, same error |
| F. `init -plugin-dir=/cache`, no `TF_PLUGIN_CACHE_DIR`, zh-only lock | **PASS**, 0.35–0.45 s |
| F2. `init -plugin-dir=/cache`, no lock file | PASS, 0.35–0.38 s; writes a lock file with a hash for this platform only (Terraform prints a note that other platforms' checksums are not included) |
| G. CLI config `provider_installation { filesystem_mirror { path = "/cache" } }`, no lock file | PASS, 0.35–0.38 s |
| G2. same mirror + full lock file | PASS, 0.95–1.0 s |
| G3. same mirror + zh-only lock file | PASS, 0.35–0.37 s |
| H. mirror only for `hashicorp/aws` (`include`), `direct {}` with `exclude` for it | PASS, 0.35–0.37 s |

Findings:
1. **The plugin cache alone does not make `init` work offline.** Terraform still asks the registry which versions exist, even when the cache holds exactly the locked version. It fails fast with a clear error, it does not hang.
2. **The cache folder is already a valid mirror.** Its layout (`registry.terraform.io/hashicorp/aws/6.73.0/<os_arch>/terraform-provider-aws_v6.73.0_x5`) works unchanged as `-plugin-dir` or as a `filesystem_mirror` path. No extra download or conversion step.
3. **The mirror keeps the saving.** Each exercise's `.terraform` is 28–32 KB (a symlink to the cache), not 729 MB, in both `-plugin-dir` and mirror modes.
4. **A lock file with only `zh:` hashes is accepted with a mirror** (G3, H), so tflings does not need to ship lock files with `h1:` hashes for five platforms (which would need downloading every platform's provider to compute).
5. Case H keeps online fallback for other providers (`random`, `local`, small version-constraint lessons), which therefore still need the network the first time.

Not tested: macOS and Windows for the mirror recipe (Linux container only); `-plugin-dir` and mirror with a lock file that pins a different version than the cache holds; what happens when a learner's exercise asks for a provider version the cache lacks.

## Proposed recipe

- `doctor --prefetch` (online): `terraform init` in a scratch folder with `TF_PLUGIN_CACHE_DIR=<tflings cache>` to fill it. This is what the tests did.
- Runner, every `init`: write a CLI config file and point `TF_CLI_CONFIG_FILE` at it:
  `provider_installation { filesystem_mirror { path = "<cache>" include = ["hashicorp/aws"] } direct { exclude = ["hashicorp/aws"] } }`
  The AWS provider then installs from the cache with no network; other providers still come from the registry.
- Ship exercise lock files with `zh:` hashes only.

## Plan per save (native macOS, 5 runs each, median seconds)

Each Terraform command starts the provider process once, so `validate`, `plan` and `show -json` each cost about 1 s on their own.

| Config | validate | plan | plan `-refresh=false` | plan `-out` | show `-json` (plan) | validate + plan `-out` + show `-json` |
|---|---|---|---|---|---|---|
| Terraform 1.17.2, no resources | 1.02 | 1.08 | 1.09 | 1.11 | 1.05 | 3.12 |
| 1.17.2, 1 resource (SSM parameter) | 1.24 | 1.57 | 1.54 | 1.54 | 1.04 | 3.84 |
| 1.17.2, 13 resources (spike 01 config) | 1.26 | 2.83 | 1.54 | 2.82 | 1.03 | 5.07 |
| Terraform 1.12.2, no resources | 1.01 | 1.08 | 1.07 | 1.08 | 1.02 | 3.10 |
| 1.12.2, 1 resource | 1.23 | 1.51 | 1.51 | 1.53 | 1.01 | 3.76 |
| 1.12.2, 13 resources | 1.24 | 2.81 | 1.55 | 2.82 | 1.03 | 5.06 |

The last column is the measured time of the three commands run back to back. A first version of this run omitted the flag set that `validate` and `show` accept and reported 0.05 s for them; those commands had failed instantly, so those numbers were discarded and the run repeated with checked exit codes (all 0).

Observations:
- A per-save check with all three steps is 3.1 s (no resources), 3.8 s (one resource) and 5.1 s (13 resources). The 5 s target is met for typical small exercises and missed by a hair for a 13-resource one.
- **Refresh is the cost in bigger configs**: 13 resources 2.8 s with refresh, 1.5 s with `-refresh=false`. For plan-mode checks, which do not need refreshed state, `-refresh=false` saves about 1.3 s.
- `validate` can probably be dropped from the save loop: `plan` already validates, so the loop could be `plan -out` + `show -json` (about 2.1 s, 2.6 s, 3.9 s by adding the measured medians; not measured as a pair). Not tested whether `show -json` includes enough of the configuration for the CEL `config` checks.
- Terraform 1.12.2 and 1.17.2 are within 0.05 s of each other everywhere.
- I did not test why a bare provider start costs about 1 s; the 729 MB binary is the obvious suspect.

## Addendum (2026-11-14): provider-free configurations need no network at all

`nonet.sh`, run in a Debian container with `--network none`, Terraform 1.12.2 and 1.17.2: a configuration with only `terraform_data` and an output (no `provider` block) runs `init` ("terraform.io/builtin/terraform is built in to Terraform"), then `plan` ("Plan: 1 to add") with an empty environment and `CHECKPOINT_DISABLE=1`. So plan-mode exercises that do not declare the AWS provider need neither Floci, nor the provider download, nor DNS. This is what the first plan-mode topics (`hcl`, `variables`, `outputs`, `types`, `expressions`, `functions`) can use.

## Addendum 2 (2026-11-14): the mirror folder must differ from the plugin cache folder

Found while building tflings: with `TF_PLUGIN_CACHE_DIR=<dir>` **and** `filesystem_mirror { path = "<dir>" }` pointing at the same folder, `init` fails with `Error while installing hashicorp/aws v6.73.0: cannot install existing provider directory <dir>/registry.terraform.io/hashicorp/aws/6.73.0/darwin_arm64 to itself`. The spike's test cases F, G and H all used a different (empty) cache folder, so they did not show it. tflings uses `<home>/plugins` as the mirror and `<home>/plugins-extra` as `TF_PLUGIN_CACHE_DIR`. Verified on macOS: `init` from the mirror, then a plan with the real AWS provider against Floci, 2.9 s per run.
