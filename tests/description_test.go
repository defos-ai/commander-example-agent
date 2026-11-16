package exampleagent_test

import (
	"reflect"
	"testing"

	described "github.com/defos-ai/commander-example-agent"
	"github.com/defos-ai/commander/framework/agent"
	agentmodel "github.com/defos-ai/commander/framework/domain/agents"
	fsartifacts "github.com/defos-ai/commander/providers/fs/artifacts"
)

func TestStaticDescriptionsMatchRuntimeCatalog(t *testing.T) {
	store, err := fsartifacts.NewArtifactFileFsStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	instance, err := described.NewAgent(agent.AgentConfig{ArtifactFileStore: store})
	if err != nil {
		t.Fatal(err)
	}
	runtime := instance.Describe()
	if dependencies := described.Dependencies(); len(dependencies) != 0 || !reflect.DeepEqual(dependencies, runtime.Dependencies) {
		t.Fatal("HTTP-only example should declare no external software dependencies")
	}
	responsibilities := described.Responsibilities()
	if len(responsibilities) == 0 || !reflect.DeepEqual(responsibilities, runtime.Responsibilities) {
		t.Fatal("static responsibilities differ from registration")
	}
	descriptions := described.ActionDescriptions()
	if len(descriptions) == 0 || len(descriptions) != len(runtime.Actions) {
		t.Fatal("static action descriptions differ from the registered catalog")
	}
	for _, registered := range runtime.Actions {
		matched := false
		for _, value := range descriptions {
			if value.Name == registered.Name && value.Description == registered.Description && value.Description != "" {
				matched = true
			}
		}
		if !matched {
			t.Fatalf("registered action %q is missing its static description", registered.Name)
		}
	}
	responsibilities[0] = "changed"
	descriptions[0].Description = "changed"
	if described.Responsibilities()[0] == "changed" || described.ActionDescriptions()[0].Description == "changed" {
		t.Fatal("description calls share mutable slices")
	}
}

func TestDependencyMetadataOverridesAndSnapshotIsolation(t *testing.T) {
	store, err := fsartifacts.NewArtifactFileFsStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	want := agentmodel.Dependency{Name: "git", Description: "Install Git in the execution environment.", Version: "2.40 or newer"}
	configured := []agentmodel.Dependency{want}
	instance, err := described.NewAgent(agent.AgentConfig{ArtifactFileStore: store, Dependencies: configured})
	if err != nil {
		t.Fatal(err)
	}
	configured[0].Description = "caller mutation"
	snapshot := instance.Describe()
	if !reflect.DeepEqual(snapshot.Dependencies, []agentmodel.Dependency{want}) {
		t.Fatal("caller dependency declarations were lost or shared with registration")
	}
	snapshot.Dependencies[0].Description = "snapshot mutation"
	if !reflect.DeepEqual(instance.Describe().Dependencies, []agentmodel.Dependency{want}) || len(described.Dependencies()) != 0 {
		t.Fatal("dependency snapshots share mutable metadata")
	}
}
