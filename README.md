# oap-bootstrap

Stand up an [open-agentic-platform](https://github.com/stagecraft-ing/open-agentic-platform)
instance in a new GitHub org and bring its Hetzner K3s estate online, in one
resumable CLI. Fork the platform into your org, register the GitHub App, wire
every secret, provision the cluster, and verify, without the multi-hour manual
choreography.

The authoritative design is the spec at
[`specs/001-oap-instance-bootstrap-cli`](specs/001-oap-instance-bootstrap-cli/spec.md).
This repo is governed by [spec-spine](https://crates.io/crates/spec-spine-cli):
run `spec-spine compile` and `spec-spine lint` to check the corpus.

## Status

Early. **M1** ships today: the config model and two working commands, `init`
and `doctor`. The provisioning phases (`github`, `cluster`, `dns`, `identity`,
`platform`, `verify`, `apply`) are registered stubs landing in later milestones.

## Quickstart

```bash
go build -o oap-bootstrap ./cmd/oap-bootstrap

./oap-bootstrap init      # collect/generate config into oap.env
./oap-bootstrap doctor    # preflight: required tools + config readiness
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
