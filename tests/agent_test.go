package exampleagent_test

import (
	"strings"
	"testing"

	exampleagent "github.com/defos-ai/commander-example-agent"
	"github.com/defos-ai/commander/framework/agent"
	"github.com/defos-ai/commander/framework/agent/action"
	fsartifacts "github.com/defos-ai/commander/providers/fs/artifacts"
)

func TestAgentsHaveIndependentCatalogs(t *testing.T) {
	store, err := fsartifacts.NewArtifactFileFsStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	config := agent.AgentConfig{ArtifactFileStore: store}
	first, err := exampleagent.NewAgent(config)
	if err != nil {
		t.Fatal(err)
	}
	second, err := exampleagent.NewAgent(config)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID() == second.ID() || !strings.HasPrefix(first.ID(), "example-agent-") || second.Describe().Type != exampleagent.AgentType {
		t.Fatalf("unexpected agent identities: %q and %q", first.ID(), second.ID())
	}
	if err := first.DiscoverActions(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	for _, instance := range first.Describe().Actions {
		if instance.Status != action.Ready {
			t.Fatalf("first agent action is not ready: %+v", instance)
		}
	}
	if len(second.Describe().Actions) != 2 {
		t.Fatalf("expected two example actions, got %+v", second.Describe().Actions)
	}
	for _, instance := range second.Describe().Actions {
		if instance.Status != action.Unknown {
			t.Fatalf("discovery leaked between agents: %+v", instance)
		}
	}
}
