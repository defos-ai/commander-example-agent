# Commander agent template

A small Go template for extending [Commander](https://github.com/defos-ai/commander)
with new agents. It gives builder-agents and coding agents such as Codex or Claude
a standard starting point: typed JSON actions, validation, proposals and artifacts.

Start with [AGENTS.md](AGENTS.md) for implementation conventions and validation.

## Build your agent

1. Clone this template. Set your module path in [go.mod](go.mod) and update local
   imports. Set `AgentType` and responsibilities in [agent.go](agent.go).
2. Implement actions in [actions/](actions/). Follow [analyze_text.go](actions/analyze_text.go)
   for typed inputs, outputs and follow-up proposals, and [write_summary.go](actions/write_summary.go)
   for file artifacts. Register each action constructor in [agent.go](agent.go).
3. Update the example [mission](missions/analyze-text.json) and [tests](tests/)
   to match your agent's capabilities.

Validate your changes:

```sh
go test -race -count=1 -timeout=3m ./...
go vet ./...
```

Publish from **Actions → Push agent image → Run workflow**. The
[workflow](.github/workflows/push.yml) reads `AgentType` from `agent.go` and pushes
`ghcr.io/<owner>/commander-<AgentType>` with your chosen tag (default `stable`)
and a `sha-<commit>` tag.

## Try it locally

Requires Docker Compose. Builds your agent alongside Commander `stable` hosts.
Set `COMMANDER_VERSION` to pin a release.

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

Template source: [MIT](LICENSE). Commander framework and provider dependencies
remain AGPL-3.0-only. Their license requirements apply to combined builds.
