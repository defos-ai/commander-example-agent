# Working on the agent template

Keep this a small, runnable starting point for creating new Commander agents.
README covers customize/run/connect; implementation and validation guidance belongs
in the scoped files below. Keep one root `.gitignore` and a companion `AGENTS.md`
beside every README.

## Boundaries

- Preserve uncommitted work. Do not commit, publish or delete demo state without
  authorization. Never log credentials or commit environment files.
- Template source is MIT; Commander framework/provider dependencies remain
  AGPL-3.0-only. Preserve notices and the README distinction for combined builds.
- Import Commander framework and providers only. Do not import Commander apps,
  embed a controller or publish sibling-checkout replacements in `go.mod`.
- Commander → Controller → Agent: actions propose work; the controller selects
  executors and accepts results. Agents upload artifact bytes; the controller
  persists accepted metadata. Broker acknowledgement is separate from acceptance;
  never claim exactly-once execution.

## Find implementation guidance

| Area | Guide |
| --- | --- |
| Typed actions, proposals and artifact output | [actions/AGENTS.md](actions/AGENTS.md) |
| Support packages, HTTP clients and checkpoints | [internal/AGENTS.md](internal/AGENTS.md) |
| Instance IDs, flags, providers and shutdown | [cli/AGENTS.md](cli/AGENTS.md) |
| Image build and runtime settings | [docker/AGENTS.md](docker/AGENTS.md) |
| Mission definitions and policy examples | [missions/AGENTS.md](missions/AGENTS.md) |
| Public tests and dedicated-stack validation | [tests/AGENTS.md](tests/AGENTS.md) |
| Release workflows, manifest exporter and CI | [.github/AGENTS.md](.github/AGENTS.md) |

## Customize and keep contracts aligned

1. Change the module path and local imports, then set `AgentType` directly in
   `agent.go` as a lowercase Docker-compatible string literal.
2. Update `Responsibilities()` and `ActionDescriptions()` in `description.go`.
   Both return fresh slices and pure static metadata: no environment reads,
   provider construction, discovery or I/O. Advertise capabilities, not readiness.
3. Implement typed, validated actions in `actions/`. Register one fresh constructor
   per action in `agent.go`; executors hold discovery state and must not be shared
   between agents. Preserve an explicitly empty catalog supplied by a caller.
4. Replace, rename or remove the example actions, HN client, mission and fixtures
   to suit the new agent. Keep deployment-wide action names unique. Reuse action
   description constants in static metadata; update registration, proposals,
   mission lists/schemas, public action tests and API integration together.
5. Keep substantial subprocess, Git, archive, filesystem and HTTP support under
   `internal/<purpose>/`. Small single-action helpers may stay beside their action.
   Keep tests/fixtures in `tests/`; never export helpers solely for tests.

The bundled HN reading-list workflow demonstrates framework capabilities.
Replace it freely when creating an agent for a different purpose.

## Compose and mission runtime

- `compose.yaml` is an isolated filesystem demo with one controller. State volumes
  belong to UID 1000. Bind MCP/API/Admin ports 4283/4284/4285 to loopback; API uses
  `ENV=development`. Test fixtures/services and cleanup belong only in the test
  override and dedicated stack described in [tests/AGENTS.md](tests/AGENTS.md).
- Keep API and MCP at PID 1 so credential commands can inspect their live
  environment. Keep `BUS_URL`/`STORAGE_URL` out of Dockerfile defaults; deployments
  supply environment variables or flags, and Compose owns local mounts/settings.
- Import mission definitions through the API with the published Admin CLI
  `missions` Compose tool; the bundled demo uses `missions/hacker-news.json`.
  There is no controller `--with-missions` flag. Import preserves existing names;
  use explicit Admin update to replace a definition.
  Pass generated API passwords over stdin without printing or saving them.
- Mission `objectives` is a nonempty ordered array of nonblank strings; all apply
  and Admin shows each separately like Agent Responsibilities. Steps retain singular
  `objective`. `starting_actions` guides assignment but creates no queued task.
  Valid proposals continue automatically unless a continuation policy needs review.
  Use typed `policies`, never legacy root `rules`, `completion` or placement.
- Playground is an optional interactive profile using published
  `ghcr.io/defos-ai/commander-playground` and the hosts' `COMMANDER_VERSION`.
  Never use `--all-in-one`: it starts separate hosts and bypasses this demo.
- `mcp-credentials` generates a shared token in its dedicated Docker volume; MCP
  and Playground mount it read-only and export before exec. This bootstrap is the
  only credential-file exception. Preserve `MCP_TOKEN`/`COMMANDER_MCP_TOKEN`
  overrides, reuse tokens across restarts and never log them. Do not mount the
  Docker socket or expose an unauthenticated credential endpoint.
- Commander host images use explicit `linux/amd64` for ARM emulation. The agent
  builds for the Docker builder's platform and releases for linux/amd64 and arm64.

## Dependencies and checks

Use Go 1.26.7 or newer. Run the unit/race/vet checks and, when relevant, the
**dedicated** Compose integration workflow in [tests/AGENTS.md](tests/AGENTS.md).

Published modules and demo images pin Commander `v0.4.0`; Go uses semantic versions,
not Docker `stable`. Upgrade hosts and rebuilt agents together: v0.4.0 replaces
nested execution `work` with `operation_id`/`task_id` and is incompatible with old
agents. Before retained-state upgrades, back up state and its encryption key,
review unmigrated legacy policies/execution records, and explicitly update saved
missions. See the [upgrade notes](https://github.com/defos-ai/commander/releases/tag/v0.4.0).

For framework development, use an ignored `go.work` with this module and the
checkout's `framework`, `providers/fs`, `providers/rabbitmq`, `providers/s3`, mapping
versioned requirements with workspace replacements. Never publish that workspace
or add local replacements to `go.mod`. Docker and CI use `GOWORK=off`.

Before changing pins/images, verify published downloads, tests and the documented
Compose run without a workspace; retain checksums. `go mod tidy` skips `.github`:
restore renderer dependencies with
`go get github.com/defos-ai/commander/framework/agent/manifest@v0.4.0`, then explicitly
test/vet `.github/scripts` and `.github/tests`. Keep the comment-only
`agent.yaml.tmpl` marker required by the v0.2.0 builder; manifest details live in
[.github/AGENTS.md](.github/AGENTS.md).
