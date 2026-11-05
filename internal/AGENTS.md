# Working on internal support

- Follow the root `AGENTS.md`. Group substantial support code by responsibility
  in `internal/<purpose>/`; keep action contracts and registration in `../actions/`
  and the root agent catalog. Export helpers only when production callers need them.
- Add support packages for your agent's domain and remove unused starter code.
  The HN-specific guidance below applies when retaining `hn/`.
- `hn/` owns the public HN API client, response mapping and execution checkpoints.
  Keep the fixed official API origin, injected HTTP client support, disabled
  redirects, five-second request timeout, thirty-second overall timeout and
  one-MiB response limit. Inspect at most twenty feed entries.
- Skip null, dead, deleted, non-story and blank-title items. Deduplicate feed IDs,
  verify item IDs, decode title entities and validate normalized public metadata
  before persisting it. Never fetch linked articles or comment trees.
- Checkpoints bind the selected feed and limit to its IDs, then save each unique
  item's outcome, including skipped items. Keep stable versioned names and at
  most twenty-one facts within the same execution, each below sixteen KiB.
- Preserve the checkpoint byte encoding: it prevents JSON envelope escaping from
  exceeding the size limit for valid titles and URLs. Validate decoded UTF-8,
  structure, metadata and ID bindings before using saved values.
- Reuse completed snapshots on retry. Fall back to ordinary fetching only when
  the initial checkpoint provider is unavailable; propagate actual load/save
  failures. A failed save may have an unknown outcome, and checkpoints do not
  guarantee exactly-once requests or recovery in a different execution.
- Keep secrets, artifact files and deployment connection settings out of these
  checkpoints. Honor cancellation and keep errors free of credentials.
- Exercise support through public action tests in `../tests/`, including HTTP
  limits, cancellation, checkpoint replay, corruption and persistence failures.
  Do not add runtime configuration or export private helpers solely for tests.
