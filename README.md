# Commander agent template

A template for building agents for [DEFOS Commander](https://github.com/defos-ai/commander).
The replaceable Hacker News example demonstrates questions, checkpoints,
follow-up proposals and Markdown/JSON artifacts.

## Build your agent

Ask your agent:

> Clone https://github.com/defos-ai/commander-example-agent and build a Commander
> agent for [describe your workflow]. Follow AGENTS.md and replace the example
> actions, mission and tests with the capabilities needed.

See [implementation guidance](AGENTS.md) and [commented mission policies](missions/README.md).

## Try it locally

```sh
docker compose up --build --wait
docker compose exec -T api commander-api credentials --raw | docker compose run --rm --no-deps -T missions
docker compose run --rm --no-deps playground
```

Choose a model provider in Playground and enter its key. Try:
“Run hacker-news-digest using the top 10 stories.” Answer the topic question;
reading HN needs no credentials.

For existing deployments, configure `BUS_URL` and `STORAGE_URL`;
see [connection guidance](cli/AGENTS.md) and [release guidance](.github/AGENTS.md).

## License

Template & derived agents: [MIT](LICENSE). Commander: [AGPL-3.0-only](https://github.com/defos-ai/commander/blob/main/LICENSE).
