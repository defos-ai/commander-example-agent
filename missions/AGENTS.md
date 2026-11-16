# Working on mission definitions

- Follow the root `AGENTS.md`; keep importable mission definitions in this directory.
- Keep `.json` files valid JSON without comments. Explain policies with annotated
  `jsonc` examples in the companion README. Distinguish populated customization
  examples from the defaults in importable missions.
- Replace or remove the starter mission for a new agent. Update the Compose
  import configuration, public tests and documentation to use the new definition.
- Use Commander v0.6.0 typed `policies`: `continuation`, `completion` and `sharing`.
  Do not add legacy root `rules`, `completion`, placement or `policies.session` fields.
- Keep `objectives` a nonempty ordered array of nonblank strings; every entry
  applies. Steps retain their singular `objective` field.
- Align required and starting actions, input/output schemas and proposal payloads
  with the registered actions. `starting_actions` guides the first assignment;
  creating an operation alone does not queue that task.
- Choose continuation and completion policies for the intended workflow. Named
  model review requires a model/client decision before proposed work continues.
- Sharing requires live receivers; implement that capability before enabling its
  policy example. The sequential HN starter keeps sharing empty. Sessions use action
  `session_mode: open`/`use` and controller-dispatched context helpers, not mission
  policies or IDs in action inputs; follow-ups inherit their direct parent's binding.
- Validate definitions through the published Admin CLI and the dedicated Compose
  workflow documented at the root. Import preserves existing mission names;
  use Admin's explicit update command when replacing a saved definition.
- Keep public integration tests and fixtures in `tests/`. Run test-only cleanup
  only against a disposable dedicated stack, and stop that stack after failures.
