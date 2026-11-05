# Mission policy examples

Replace the starter mission with one matching your agent's actions. This populated
example shows all Commander v0.4.0 policy groups. Replace the action names with
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
    "session": [
      {
        // This action creates a workspace retained by the executing agent.
        "create_action": "create-workspace",
        // Route workspace use to its exact owner using the saved session binding.
        "use_actions": ["write-report", "review-report"]
      }
    ],
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

Session creators declare `SessionMode: "create"` and users declare
`SessionMode: "use"`. Their schemas accept session routing fields, and the resource
owner must provide the use actions. A session policy supplies the binding; the
actions implement the resource.

Start `watch-evidence` before its producers (before batch creation for grouped
work). It consumes updates with `action.ReceiveSharedData(ctx)`. Sharing is live
and best effort, with no replay; artifact bytes stay in shared file storage.

The bundled `hacker-news.json` keeps automatic continuation, model completion and
empty session/sharing lists for its sequential example. Copy only the policies
your new workflow needs. Import preserves existing mission names; use Admin's
explicit update command to replace a saved definition.
