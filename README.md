# Commander agent template

A starting point for building your own [Commander](https://github.com/defos-ai/commander)
agent with typed actions, questions, checkpoints, proposals and artifacts.
The included Hacker News workflow is an example you can replace.

## Build your agent

Ask your agent:

> Clone https://github.com/defos-ai/commander-example-agent and build a Commander
> agent for [describe your workflow]. Follow AGENTS.md, replace the example actions
> and mission with the capabilities needed, and run the tests.

1. Set your module path and imports, `AgentType` in [agent.go](agent.go), and
   responsibilities in [description.go](description.go).
2. Implement actions in [actions/](actions/); register fresh constructors in
   `agent.go` and list their descriptions in `description.go`.
3. Replace the [example mission](missions/hacker-news.json) and [tests](tests/).
   See [commented policy examples](missions/README.md) and [implementation guidance](AGENTS.md).

Validate with Go 1.26.7 or newer:

```sh
GOWORK=off go test -race -count=1 -timeout=3m ./... ./.github/tests
GOWORK=off go vet ./... ./.github/scripts ./.github/tests
```

## Run locally

With Docker Compose, start the shared hosts and your agent, import its mission,
then open Playground:

```sh
docker compose up --build --wait
docker compose exec -T api commander-api credentials --raw | docker compose run --rm --no-deps -T missions
docker compose run --rm --no-deps playground
```

Choose a model provider and enter its key. To try the unchanged example, ask
“Run hacker-news-digest using the top 10 stories,” then answer its topic question.
HN needs no credentials; the example produces Markdown and JSON from story metadata.

[Admin](http://localhost:4285) shows tasks and artifacts. Connect it to
`http://localhost:4284` with credentials from
`docker compose exec api commander-api credentials`.
Use `/exit` to close Playground and `docker compose down` to stop the stack.

## Connect to an existing deployment

Supply `BUS_URL` and `STORAGE_URL` at runtime through environment variables or
flags; the Docker image sets neither. Use shared filesystem paths, RabbitMQ
(`amqp://` or `amqps://`) and S3-compatible storage (`s3://bucket`) as appropriate.
The agent needs the controller's bus and artifact storage, without a database.

```sh
BUS_URL=/shared/bus STORAGE_URL=/shared/storage go run ./cmd
```

Run `go run ./cmd --help` for flags. `--agent-id-suffix=0123abcd` sets the instance
suffix; never reuse one across running instances of the same agent type.

The template pins Commander v0.4.0. Upgrade hosts and rebuilt agents together;
review the [upgrade notes](https://github.com/defos-ai/commander/releases/tag/v0.4.0)
for retained state and saved mission changes.

## Release your agent

Run **Actions → Make release** on the default branch with a new version, such as
`v0.4.0`. After validation, it publishes binaries for six platforms, a Linux
amd64/arm64 image and a digest-pinned `agent.yaml`. Names follow `AgentType`;
make the GHCR package public for unauthenticated pulls. See
[release guidance](.github/AGENTS.md) for details.

Template source: [MIT](LICENSE). Commander framework and provider dependencies
remain AGPL-3.0-only; their requirements apply to combined builds.
