# Mission policy examples

Replace the starter mission with one matching your agent's actions. This populated
example shows all Commander v0.6.0 policy groups. Replace the action names with
capabilities you implement; the policies alone do not create tasks or resources.
Comments are shown as JSONC here; importable mission files use plain JSON.

```jsonc
{
  "policies": {
    "continuation": {
      // plan-research proposes parallel child tasks. Wait for those saved direct
      // children before deciding what follows; this does not merge their data.
      "aggregate_results": ["plan-research"],
      // Review this action's result before running its proposed next actions.
      "model_review": ["review-report"]
    },
    "completion": {
      // Model submissions become drafts until an operator explicitly completes
      // the operation. Use "model" to permit model/client completion instead.
      "decision_by": "operator"
    },
    "sharing": [
      {
        // Forward accepted outputs, artifact references and answered questions
        // from these producers to a concurrently running evidence reviewer.
        "from_actions": ["ask-research-scope", "research-topic", "write-report"],
        "to_actions": ["watch-evidence"],
        // "decisions" means answered task questions, not completion approvals.
        "include": ["results", "artifacts", "decisions"]
      }
    ]
  }
}
```

`plan-research` must propose its child work. Aggregation covers only those saved
direct children, not all descendants. `model_review` requires a model/client
continuation decision even for successful results with valid follow-up proposals.

Sessions are action contracts, not mission policies: declare `session_mode: "open"`
or `"use"` and use `action.OpenSession(ctx)` or `action.UseSession(ctx)`. Follow-ups
inherit their direct parent's binding; keep session IDs out of action inputs.
See [session usage](https://github.com/defos-ai/commander/blob/v0.6.0/docs/features/sessions.md#open-and-use).

Start `watch-evidence` before its producers (before batch creation for grouped
work). It consumes updates with `action.ReceiveSharedData(ctx)`. Sharing is live
and best effort, with no replay; artifact bytes stay in shared file storage.

The bundled `hacker-news.json` keeps automatic continuation, model completion and
an empty sharing list for its sequential example. Copy only the policies
your new workflow needs. Import preserves existing mission names; use Admin's
explicit update command to replace a saved definition.
