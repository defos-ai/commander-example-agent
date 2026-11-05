# Working on the agent host

- Follow the root `AGENTS.md`. `cli.go` selects the instance ID and owns host
  lifetime; cancellation stops serving and shutdown closes dependencies.
  `../cmd/main.go` only translates errors to a process exit status.
- `config.go` owns flags/environment. `--agent-id-suffix` overrides
  `COMMANDER_AGENT_ID_SUFFIX`, including an explicitly empty flag. Nonempty
  suffixes accept exactly eight lowercase hexadecimal characters; unset/empty
  generates a fresh ID.
- Before opening providers, build `AgentConfig.ID` with `id.AgentIDFromSuffix`
  using the fixed `AgentType`. Never accept an externally supplied type prefix
  or include the ID in `PublicConfig`.
- The spawner must save the full ID before launch and reconcile retries against
  its owned instance. New/replacement launches get fresh IDs. Never run two
  instances with one ID; a supplied suffix provides neither duplicate prevention
  nor authentication. Registration does not guarantee action readiness.
- `factory.go` selects concrete filesystem, RabbitMQ and S3 providers. Reject
  unsupported schemes. Agents require a bus and artifact file store, not a database.
- Keep credentials out of errors, logs and public configuration. Connection URLs
  come from runtime flags/environment; Compose owns local demo settings, not
  Dockerfile defaults.
- Keep host tests external under `../tests/`. Use the checks and disposable-stack
  workflow in [../tests/AGENTS.md](../tests/AGENTS.md).
