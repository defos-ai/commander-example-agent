# Commander agent template

A small Go template for extending [Commander](https://github.com/defos-ai/commander)
with new agents. It gives builder-agents and coding agents such as Codex or Claude
a standard starting point: typed JSON actions, validation, proposals and artifacts.

Start with [AGENTS.md](AGENTS.md) for implementation conventions and validation.

## Build your agent

1. Clone this template. Set your module path in [go.mod](go.mod) and update local
   imports. Set `AgentType` in [agent.go](agent.go), then update
   [description.go](description.go) with responsibilities and action descriptions.
2. Implement actions in [actions/](actions/). Follow [analyze_text.go](actions/analyze_text.go)
   for typed inputs, outputs and follow-up proposals, and [write_summary.go](actions/write_summary.go)
   for file artifacts. Register each action constructor in [agent.go](agent.go) and
   list it in [description.go](description.go).
3. Update the example [mission](missions/analyze-text.json) and [tests](tests/)
   to match your agent's capabilities. Mission `objectives` is an ordered array
   of goals or instruction groups; all apply and appear separately in Admin.

Validate your changes:

```sh
go test -race -count=1 -timeout=3m ./... ./.github/tests
go vet ./... ./.github/scripts ./.github/tests
```

## Release your agent

Run **Actions → Make release** on the default branch with a version such as
`v0.2.1`. After tests pass, the [workflow](.github/workflows/make-release.yml) publishes:

- Linux, macOS and Windows binaries for amd64 and arm64.
- `ghcr.io/<owner>/commander-<AgentType>:v0.2.1` for Linux amd64 and arm64.
- `agent.yaml` with responsibilities, action descriptions, binary checksums and
  the Docker image digest.

Names come from `AgentType` in `agent.go`. Set the GHCR package visibility to
public if others should pull it without signing in. Use a new version for each
release; existing Docker version tags are never overwritten. Retry failed jobs
in the same run. Complete releases promote Docker `stable` and GitHub `Latest`
when they are the newest version. Commander spawner uses the package manifest
to select the right executable or image.

## Try it locally

Requires Docker Compose. The demo uses Commander `v0.2.0`; set
`COMMANDER_VERSION` to try another release.

```sh
docker compose up --build --wait
docker compose run --rm --no-deps playground
```

Choose a model provider and enter its API key in Playground. Try:

> Run analyze-text on "Commander turns plans into useful results."

Open [Admin](http://localhost:4285) to inspect tasks and download `summary.md`.
Add `http://localhost:4284` using the credentials shown by:

```sh
docker compose exec api commander-api credentials
```

Use `/exit` to close Playground and `docker compose down` to stop the stack.

## Connect to Commander

Agents run as independent processes or containers across machines. Each instance
advertises its actions; the controller assigns tasks to available instances.
Scale execution by running more instances of an agent type.

For distributed deployments, Commander supports:

| Service | Role | Setting |
| --- | --- | --- |
| RabbitMQ | Task and result messaging | `BUS_URL` (`amqp://` or `amqps://`) |
| MongoDB | Shared orchestration state and artifact metadata | `DATABASE_URL` (`mongodb://` or `mongodb+srv://`) |
| S3-compatible storage | Artifact files | `STORAGE_URL` (`s3://bucket`) |

Agents use `BUS_URL` and `STORAGE_URL`; configure MongoDB on the controller, API
and MCP hosts. Supply service credentials through your deployment's environment.
With the agent's connection settings configured, start it with `go run ./cmd`.

The local Compose demo uses shared filesystem providers. To connect an agent to
an existing FS deployment:

```sh
BUS_URL=/shared/bus STORAGE_URL=/shared/storage go run ./cmd
```

FS paths must point to the same shared directories as the controller.
Run `go run ./cmd --help` for flags.

`--agent-id-suffix=0123abcd` overrides `COMMANDER_AGENT_ID_SUFFIX`. The agent
adds its own type: `example-agent-0123abcd`. Use eight lowercase hex characters.
Omit both settings or pass `--agent-id-suffix=` for a fresh ID.
Never give two running instances of the same type the same suffix.

Template source: [MIT](LICENSE). Commander framework and provider dependencies
remain AGPL-3.0-only. Their license requirements apply to combined builds.
