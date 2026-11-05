# Working on public tests

- Follow the root `AGENTS.md`. Keep tests and fixtures here, using external test
  packages to exercise public action and host APIs. Use existing dependency
  injection for local fixtures; do not add runtime settings or export helpers
  solely for tests, or put tests in action/CLI packages to access private code.
- Use Go 1.26.7 or newer. Action behavior changes need public action tests;
  registration, proposals and artifacts are also covered through API integration.
  Keep ordinary tests deterministic and avoid paid model calls.
- Replace the starter scenarios and fixtures when customizing the action catalog;
  test the new agent's public contracts instead of retaining unrelated HN checks.
- The supplied unit tests inject saved HN HTTP responses and exercise the topic
  question, complete action chain, bounds, cancellation and checkpoint replay/errors.
- The starter `TestAgentThroughAPI` waits for its five HN actions, submits a saved
  snapshot to `filter-hn-stories`, checks rank/write proposals and both artifacts,
  then settles work and cleans operation state. It needs neither live HN nor a model.

Run from the repository root with published modules and no workspace:

```sh
GOWORK=off go test -race -count=1 -timeout=3m ./... ./.github/tests
GOWORK=off go vet ./... ./.github/scripts ./.github/tests
```

## Dedicated Compose integration

These tests call test-only global cleanup. Never point them at a development or
shared API. Keep Docker lifecycle outside Go tests and always stop the stack after
failure as well. From the repository root with ports 4283/4284/4285 free:

```sh
export COMPOSE_PROJECT_NAME=commander-example-test API_USERNAME=commander
export API_PASSWORD="$(openssl rand -hex 32)"
docker compose -f compose.yaml -f .github/composes/compose.test.yaml up --build --wait --wait-timeout 120
docker compose -f compose.yaml -f .github/composes/compose.test.yaml run --rm --no-deps -T missions
docker compose -f compose.yaml -f .github/composes/compose.test.yaml run --rm --no-deps -T playground --help
GOWORK=off COMMANDER_API_URL=http://127.0.0.1:4284 go test -tags=integration -race -count=1 -timeout=3m -run '^TestAgentThroughAPI$' -v ./tests/...
docker compose -f compose.yaml -f .github/composes/compose.test.yaml down --volumes --remove-orphans
unset API_PASSWORD API_USERNAME COMPOSE_PROJECT_NAME
```

If retaining the HN example, optionally run its live public API smoke test before
stopping that same dedicated stack. It answers the saved topic question and checks
the action chain, progress logs and artifact downloads without a model:

```sh
GOWORK=off COMMANDER_HN_LIVE=1 COMMANDER_API_URL=http://127.0.0.1:4284 go test -tags=integration -race -count=1 -timeout=3m -run '^TestAgentLiveHNThroughAPI$' -v ./tests/...
```

The Playground help smoke check also exercises its shared-token wrapper and needs
no model key. For an interactive MCP connection check, choose provider `other`,
a fixed model name and local base URL, then exit after connection without sending
chat. Keep credentials out of failure logs. CI runs unit checks before Docker
integration and always removes the dedicated stack's volumes.
