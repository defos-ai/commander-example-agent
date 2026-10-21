# Working on the example agent

## Purpose and boundaries

- Keep this a small, runnable template for Commander builder-agents and AI coding
  tools to extend. README briefly explains customize/run/connect;
  this file holds implementation guidance. Keep one root `.gitignore` and a
  companion `AGENTS.md` beside every README.
- Preserve uncommitted work. Do not commit, publish, or delete demo state without
  authorization. Do not log credentials or commit environment files.
- Template source is MIT; Commander framework and provider dependencies remain
  AGPL-3.0-only. Preserve third-party license notices and the README distinction
  between the template license and dependency requirements for combined builds.
- The module imports Commander framework and providers only. Do not import
  Commander apps, add a controller inside the agent, or depend on a sibling
  checkout in published `go.mod`.
- Commander → Controller → Agent: actions propose next work; the controller
  selects who executes it. The controller loads `missions/analyze-text.json`.
  Mission `objectives` is a required nonempty array of nonblank strings. Keep
  goals or instruction groups in ordered entries; all apply. Admin lists them
  in the same style as Agent Responsibilities. Steps keep a singular `objective`.
  `starting_actions` guides the first assignment; creating an operation alone
  does not queue that task. Valid proposals execute automatically unless a
  matching mission rule requires a model decision.
- Agents upload artifact bytes; the controller persists accepted metadata.
  Broker acknowledgements are separate from result acceptance. Do not claim
  exactly-once execution.

## Customize the template

When building a new agent from this repository:

1. Set the module path in `go.mod` and update this module's imports.
2. Set `AgentType` in `agent.go` and default responsibilities in `description.go`
   to describe the agent's capabilities.
3. Implement actions in `actions/` with typed JSON schemas and input validation.
   Register a fresh constructor for each action once in `agent.go` and add its
   name/description to `ActionDescriptions` in `description.go`. Keep action
   names unique across the deployment and update follow-up proposals when renaming.
4. Align the demo mission's required/starting actions and schemas with the new
   capabilities. Update public action tests and the API integration scenario.
5. Keep README short: purpose, extension points, local UI workflow and connection
   settings. Keep implementation and test-stack details in this file.

## Entry points

- `agent.go` defines the stable agent type and creates a fresh catalog. Action
  executors hold discovery state: do not share instances between agents. Keep an
  explicitly empty action catalog empty when callers supply one.
- `.github/scripts/release.go` derives package names from the repository owner
  and `AgentType`: `commander-<AgentType>`. Keep `AgentType` declared directly in
  `agent.go` as a lowercase Docker-compatible string literal. The release tool
  fills `agent.yaml.tmpl` with the version, checksums of all six native binaries
  and the pushed Docker image digest. Never hand-maintain checksums or use a
  moving image tag in the manifest. The manifest includes delivery assets and
  static responsibilities/action descriptions, never host commands, privileges,
  networks or secrets.
- `description.go` exports pure `Responsibilities()` and `ActionDescriptions()`
  functions returning fresh slices. Runtime defaults and release manifests share
  these values; each registered action uses its corresponding description constant
  in `actions/`. Keep descriptions in sync with registration. Metadata functions
  must not read environment variables, construct agents/providers, run discovery,
  or perform I/O. Descriptions advertise capabilities, never current readiness.
- Manifest generation imports the module path read from `go.mod` through a small
  temporary Go exporter. It collects only static names/descriptions with bounded
  output and an environment excluding release/model tokens. Keep the four-field
  `metadata` command stable; only `manifest` reads the exported descriptions.
- `actions/analyze_text.go` demonstrates `action.NewFunc` with typed JSON schemas,
  input validation, deterministic output and a proposal. Keep size limits and
  cancellation checks. Add an action in its own file and register it once.
- `actions/write_summary.go` implements `action.Executor` with `Base` for a file
  artifact. Supply bytes to the runtime; do not write artifact paths directly.
- Keep `actions/` focused on registered action implementations, their input/output
  types and readiness. Put substantial subprocess, Git, archive and filesystem
  support in `internal/<purpose>/` packages grouped by responsibility. Tiny helpers
  used by one action may stay beside that action when clearer. Keep tests and
  fixtures in `tests/`; do not export support code solely to test it.
- `cli/cli.go` selects the instance ID and owns host lifetime. Shutdown
  cancels serving and closes dependencies. `cmd/main.go` only translates errors
  into a process exit status.
- `--agent-id-suffix` overrides `COMMANDER_AGENT_ID_SUFFIX`, including an
  explicitly empty flag. Nonempty values accept exactly eight lowercase hex
  characters; unset/empty generates a fresh ID. Compose `AgentConfig.ID` from
  the fixed `AgentType` and this suffix, and validate it with `id.RuntimeType`
  before opening providers. Never accept an externally supplied type prefix.
  Keep the ID out of `PublicConfig`.
  The spawner must save the full ID before launch and reconcile retries against its
  owned instance; new or replacement launches get fresh IDs. Never start two
  instances with the same ID. This setting alone provides neither duplicate
  prevention nor authentication. Registration alone
  does not guarantee that its actions are ready.
- `cli/config.go` owns flags/environment; `cli/factory.go` selects concrete FS,
  RabbitMQ and S3 providers. Reject unsupported schemes. Agents need a bus and
  artifact file store, never a database.
- `compose.yaml` is an isolated FS demonstration with one controller. All state
  volumes belong to UID 1000. API and MCP run at PID 1 so their credential lookup
  can read generated secrets from the live environment. Bind MCP/API/Admin ports
  4283/4284/4285 to loopback. The demo fixes API `ENV` to `development`;
  `.github/composes/compose.test.yaml` enables test-only API cleanup in a dedicated
  stack. Keep fixtures and test-only services in that override, not the demo.
- Playground uses the published `ghcr.io/defos-ai/commander-playground` image,
  using the same `COMMANDER_VERSION` as the other hosts (default `stable`). Keep
  Commander app dependencies out of the agent module. It is an optional
  interactive Compose profile connecting to the demo MCP. Never launch it with
  `--all-in-one`, which would start separate hosts and ignore this demo's mission
  and agent.
- `mcp-credentials` generates a shared token in a dedicated Docker volume; MCP
  and Playground mount it read-only and export it before exec. This Compose
  bootstrap file is the only credential-file exception. Preserve configured
  `MCP_TOKEN` and client `COMMANDER_MCP_TOKEN` overrides, reuse generated tokens
  across restarts, and never log tokens. Do not mount the Docker socket or expose
  an unauthenticated credential endpoint.
- Commander images currently target linux/amd64; Compose requests that
  platform explicitly so ARM hosts can use Docker emulation. The template agent
  builds for the Docker builder's platform; its release image supports both
  linux/amd64 and linux/arm64.

## Release workflows

- `make-release.yml` accepts a stable `vMAJOR.MINOR.PATCH` on the default branch.
  It calls `test.yml` (unit → integration), then `build-binaries.yml` (native
  tests/build/help on all six platforms), then `push.yml`. Keep build/push
  workflows reusable-only and checkout the triggering SHA in every job.
  Keep optional `expected_commit` and `request_id` inputs for builder dispatch.
  The first `prepare` step rejects a nonempty expected commit that differs from
  `GITHUB_SHA`, before checkout or dependent jobs. Every release job depends on
  `prepare`; never add conditions or ignored errors that bypass its failure.
  Include the request ID in `run-name` so an uncertain dispatch can be recovered
  without starting another release. Manual version-only dispatch remains available.
- Publishing uses `GITHUB_TOKEN`. The image is first pushed under a run-specific
  build tag. The parent generates the digest-pinned manifest, attaches binaries,
  manifest and LICENSE to a draft, then assigns the version image tag and
  publishes the release. Only after complete publication, `push-stable.yml`
  promotes Docker `stable` and GitHub `Latest`, rejecting backward version moves.
- Serialize releases and keep test concurrency separate from release concurrency.
  Existing Git tags must match the tested SHA. Before reserving a release, reject
  an existing Docker version tag. Never overwrite published versions or assets:
  retries reuse identical draft assets and reject differing bytes.
  Retry failed jobs in the same run so binaries and image digest stay together.

## Validation

Keep tests and fixtures in `tests/`, using external test packages to exercise
public action and host APIs. Use existing dependency injection for local
fixtures. Do not place tests in `actions/` or `cli/` merely to access private
helpers, or export implementation details and add runtime settings solely
for tests.

Use Go 1.26.7 or newer. Changes to action behavior need public action tests;
registration, proposals and artifacts are covered through the API integration
test. Keep fixtures deterministic and avoid paid model calls.

```sh
go test -race -count=1 -timeout=3m ./... ./.github/tests
go vet ./... ./.github/scripts ./.github/tests
```

Integration tests require a **dedicated stack**: they call test-only global
cleanup. Never point them at a development or shared API. Keep Docker lifecycle
outside Go tests. From a terminal with ports 4283/4284/4285 free:

```sh
export COMPOSE_PROJECT_NAME=commander-example-test API_USERNAME=commander
export API_PASSWORD="$(openssl rand -hex 32)"
docker compose -f compose.yaml -f .github/composes/compose.test.yaml up --build --wait --wait-timeout 120
docker compose -f compose.yaml -f .github/composes/compose.test.yaml run --rm --no-deps -T playground --help
COMMANDER_API_URL=http://127.0.0.1:4284 go test -tags=integration -race -count=1 -timeout=3m -run '^TestAgentThroughAPI$' -v ./tests/...
docker compose -f compose.yaml -f .github/composes/compose.test.yaml down --volumes --remove-orphans
unset API_PASSWORD API_USERNAME COMPOSE_PROJECT_NAME
```

Always stop the test stack after a failure as well. The integration test waits
for registration, submits work, checks the follow-up/artifact/report, then settles
work and cleans operation state. CI gates the Docker job on unit checks and
always tears down its volumes. The Playground help smoke check also exercises
the shared-token wrapper; it needs no model key. For interactive MCP connection
checks, select provider `other` with a fixed model name and local base URL, then
exit after connection without sending chat. Keep failure logs free of secrets.

## Develop alongside Commander

The Go module pins the published `v0.1.0` release; demo images default to the
moving `stable` tag, with `COMMANDER_VERSION` available to pin a release. Go
dependencies use published semantic versions, not the Docker `stable` tag. For
framework development, use an ignored `go.work` with this module plus the
checkout's `framework`, `providers/fs`, `providers/rabbitmq` and `providers/s3`
modules.
Map their versioned requirements to local paths with workspace replacements.
Do not commit that workspace or put local replacements in `go.mod`.

Mission files target the current Commander development schema. Published
`v0.1.0` controller hosts do not accept `objectives`; use matching development
host images until that schema is released. Do not add legacy aliases or change
published module pins merely to make the fixture load.

Docker and CI use `GOWORK=off` with the published modules. Before updating module
pins or image tags, verify module downloads, tests and the documented Compose run
without a workspace, and retain the resulting module checksums.
