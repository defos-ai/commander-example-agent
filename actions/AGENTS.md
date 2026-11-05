# Working on actions

- Follow the root `AGENTS.md`. Keep this directory focused on registered action
  implementations, typed JSON inputs/outputs, validation and readiness.
- Replace or remove the starter actions when building a new agent. Update its
  catalog, descriptions, proposals, mission and tests together.
- Give each action a unique name and a static description constant. Register a
  fresh constructor in `../agent.go` and use the same description in
  `../description.go`. Keep mission action lists and follow-up proposals aligned.
- Use `action.Func` for typed handlers and `action.Base` when implementing an
  executor directly, such as the digest writer. Validate inputs before requesting
  operator answers, making HTTP requests or returning proposals.
- Keep shared input/output types and validation together. The starter uses
  `hn.go`; rename or replace it for your domain. Put substantial HTTP, checkpoint
  and filesystem support in `../internal/<purpose>/`.
- When retaining the HN example, keep its choose/fetch/filter/rank/write chain
  aligned. Empty topics include all stories. Rank at most five by descending
  score, descending comment count, then ascending ID.
- Use stable question keys and validate answers. HN is public and needs no
  credentials; demonstrate secret requests only for capabilities that need them.
- Honor context cancellation and return errors before proposing follow-up work.
  The controller selects executors and accepts results; actions do not finish
  operations or choose a specific agent instance.
- Return artifact bytes through the runtime without writing artifact paths
  directly. Escape untrusted Markdown text and links; the starter digest handles
  empty results as valid output.
- Add public behavior tests and fixtures under `../tests/`, using external test
  packages. Keep ordinary tests deterministic with injected HTTP clients; update
  the API scenario when registration, proposals or artifacts change. Use the
  validation commands and dedicated-stack guidance in the root `AGENTS.md`.
