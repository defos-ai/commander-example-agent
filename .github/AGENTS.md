# Working on release tooling and CI

Follow the root `AGENTS.md`. Keep reusable release jobs and their safeguards intact.

## Manifest generation

- `scripts/release.go` derives `commander-<AgentType>` package names from the
  repository owner and the literal `AgentType` in `../agent.go`.
- Render/validate YAML through `framework/agent/manifest.Render`, supplying the
  version, computed checksums of all six native binaries and the pushed Docker
  image digest. Do not add a local template or `--template` flag, hand-maintained
  checksums or moving image tag.
- Retain the comment-only root `agent.yaml.tmpl` marker for the released v0.2.0
  builder's package checks. The generated manifest contains delivery assets and
  static responsibilities/action descriptions, never host commands, privileges,
  networks or secrets.
- Read the module path from `go.mod` for the temporary Go metadata exporter.
  Export only static names/descriptions with bounded output and an environment
  excluding release/model tokens. Keep the four-field `metadata` command stable;
  only `manifest` reads exported descriptions.
- `go mod tidy` skips this hidden directory. Restore renderer dependencies with
  `go get github.com/defos-ai/commander/framework/agent/manifest@v0.4.0` and explicitly
  test/vet `.github/scripts` and `.github/tests` from the repository root.

## Release workflow

- `make-release.yml` accepts stable `vMAJOR.MINOR.PATCH` on the default branch.
  Run `test.yml` (unit then integration), `build-binaries.yml` (native test/build/
  help on six platforms), then `push.yml`. Build/push remain reusable-only; every
  job checks out the triggering SHA.
- Preserve optional `expected_commit` and `request_id` builder dispatch inputs.
  The first `prepare` step rejects a nonempty expected commit unlike `GITHUB_SHA`,
  before checkout or dependent jobs. Every release job depends on successful
  preparation; never bypass it through conditions or ignored errors. Include the
  request ID in `run-name` for uncertain-dispatch recovery. Manual version-only
  dispatch remains available.
- Publish with `GITHUB_TOKEN`. Push the image under a run-specific build tag,
  generate its digest-pinned manifest, attach binaries/manifest/LICENSE to a draft,
  then assign the version image tag and publish. Only complete publication may
  trigger `push-stable.yml` to promote Docker `stable` and GitHub `Latest`.
- Serialize releases separately from test concurrency. Existing Git tags must
  match the tested SHA; reject an existing Docker version tag before reserving
  a release. Never overwrite published versions/assets. Retries reuse identical
  draft assets and reject differing bytes. Retry failed jobs in the same run so
  binaries and image digest remain paired. Reject backward stable/latest moves.

## Test-stack boundary

- CI unit checks gate Docker integration. Keep fixtures and test-only services in
  `composes/compose.test.yaml`, not the demo Compose file. API `ENV=tests` permits
  global cleanup only in the dedicated stack.
- Always tear down test containers and volumes, including on failure; keep
  failure logs free of secrets. The exact local workflow is in
  [../tests/AGENTS.md](../tests/AGENTS.md).
