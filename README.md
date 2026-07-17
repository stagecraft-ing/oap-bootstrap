# oap-bootstrap

Stand up an [open-agentic-platform](https://github.com/statecrafting/open-agentic-platform)
instance in a new GitHub org and bring its Hetzner K3s estate online, in one
resumable CLI. Fork the platform into your org, register the GitHub App, wire
every secret, provision the cluster, and verify, without the multi-hour manual
choreography.

The authoritative design is the spec at
[`specs/001-oap-instance-bootstrap-cli`](specs/001-oap-instance-bootstrap-cli/spec.md).
This repo is governed by [spec-spine](https://crates.io/crates/spec-spine-cli):
run `spec-spine compile` and `spec-spine lint` to check the corpus.

## Status

All phases implemented (milestones M0 through M5). `init` and `doctor` plus the
full provisioning sequence (`github`, `cluster`, `dns`, `identity`, `platform`,
`verify`) and the unattended `apply --yes` are wired and unit-tested. The live
cloud/cluster/Rauthy legs have not yet been exercised against a real target; a
first real run is the outstanding acceptance step (success criteria SC-001
through SC-005 in the spec).

## Quickstart

```bash
go build -o oap-bootstrap ./cmd/oap-bootstrap

# Guided, phase by phase. Every phase is idempotent (detect-or-create), so
# re-running one is how you resume after a fix.
./oap-bootstrap init       # collect/generate config into oap.env
./oap-bootstrap doctor     # preflight: required tools + config readiness
./oap-bootstrap github     # fork + register the GitHub App + Actions secrets
./oap-bootstrap cluster    # wrap setup.sh phase 1 (K3s + GitOps); capture NODE_IP
./oap-bootstrap dns        # Cloudflare A records -> NODE_IP; wait for certs
./oap-bootstrap identity   # Rauthy OIDC clients (+ guided provider leg)
./oap-bootstrap platform   # wrap setup.sh phase 2 (secrets + deploy)
./oap-bootstrap verify     # endpoint / cert / webhook health report

# Or, from a complete oap.env, run every phase unattended:
./oap-bootstrap apply --yes
```

`init` prompts for the values only you can supply (target org, domain, cloud and
GitHub tokens), generates the secrets that should be random, and computes every
service URL from your domain. The result is written to `oap.env`.

## oap.env and secrets at rest

`oap.env` is the single source of truth and the resumable state; re-running a
phase reads it and continues. Secret-classed keys are encrypted at rest with
[SOPS](https://github.com/getsops/sops) + [age](https://github.com/FiloSottile/age),
reusing your existing age key at `~/.config/sops/age/keys.txt` (or
`$SOPS_AGE_KEY_FILE`); non-secret config (domain, URLs) stays cleartext so diffs
stay readable. If sops/age are not installed, `oap.env` falls back to plaintext
with a warning.

## Prerequisites

`doctor` checks for: `git`, `gh`, `hetzner-k3s`, `flux`, `kubectl`, `helm`,
`sops`, `age`, and (optionally) `spec-spine`. Install the missing ones before
provisioning.

## License

See [LICENSE](LICENSE).
