//go:build integration

package exampleagent_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/defos-ai/commander-example-agent/actions"
	"github.com/defos-ai/commander/framework/domain/missions"
	"github.com/defos-ai/commander/framework/domain/operations"
	"github.com/defos-ai/commander/framework/host"
	"github.com/defos-ai/commander/framework/id"
	"github.com/defos-ai/commander/framework/testing/missiontest"
)

// Docker lifecycle belongs to setup/CI. The driver cleans up only its own work
// and preserves saved evidence; it never clears another operation's state.
func TestAgentThroughAPI(t *testing.T) {
	driver, mission := connectHNMission(t)
	// Use a saved snapshot so controller proposals and artifact downloads stay
	// independent of live HN and models. Unit tests cover the question/HTTP stages.
	fixture, err := os.ReadFile("fixtures/hn-stories.json")
	mustHN(t, err)
	var input actions.HNStoryBatch
	mustHN(t, json.Unmarshal(fixture, &input))
	op := driver.Start(mission, map[string]any{"feed": "top", "limit": 10})
	root := driver.Assign(op.ID, missiontest.Assignment{ActionName: actions.FilterHNStoriesName, Input: input})
	tasks := waitForHNChain(t, driver, op.ID, root.ID,
		actions.FilterHNStoriesName, actions.RankHNStoriesName, actions.WriteHNDigestName)
	want := actions.HNStoryBatch{Feed: input.Feed, Topics: input.Topics, Stories: []actions.HNStory{}}
	for _, storyID := range []int{105, 108, 104, 103, 102} {
		for _, story := range input.Stories {
			if story.ID == storyID {
				want.Stories = append(want.Stories, story)
			}
		}
	}
	var ranked actions.HNStoryBatch
	mustHN(t, json.Unmarshal(tasks[actions.RankHNStoriesName].Result.Output, &ranked))
	if !reflect.DeepEqual(ranked, want) {
		t.Fatal("ranking did not keep the expected five matching stories in order")
	}
	digest := tasks[actions.WriteHNDigestName]
	files := downloadHNDigest(t, driver, digest)
	var downloaded actions.HNStoryBatch
	mustHN(t, json.Unmarshal(files["stories.json"], &downloaded))
	if !reflect.DeepEqual(downloaded, want) {
		t.Fatal("downloaded JSON digest differs from the ranked stories")
	}
	markdown := string(files["digest.md"])
	if strings.Contains(markdown, "Security keys explained") || strings.Contains(markdown, "Google browser update") || strings.Contains(markdown, "Go templates quick start") {
		t.Fatal("digest contains an excluded topic or a story below the shortlist")
	}
	last := -1
	for _, story := range want.Stories {
		position := strings.Index(markdown, story.Title)
		if position <= last || !strings.Contains(markdown, fmt.Sprintf("https://news.ycombinator.com/item?id=%d", story.ID)) {
			t.Fatal("Markdown digest omitted a title/discussion link or changed ranking order")
		}
		last = position
	}
	completeHN(t, driver, op.ID, digest.ID, "Selected five Hacker News stories and saved digest.md and stories.json.")
}

func hnMissionConfig(t *testing.T) missiontest.Config {
	t.Helper()
	url := os.Getenv("COMMANDER_API_URL")
	if url == "" {
		url = "http://127.0.0.1:4284"
	}
	client := &http.Client{
		Timeout: 10 * time.Second, Transport: &http.Transport{Proxy: nil},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	t.Cleanup(client.CloseIdleConnections)
	return missiontest.Config{
		URL: url, HTTPClient: client, Timeout: 90 * time.Second,
		Credentials: host.BasicCredentials{Username: os.Getenv("API_USERNAME"), Password: os.Getenv("API_PASSWORD")},
	}
}

func connectHNMission(t *testing.T) (*missiontest.Driver, string) {
	t.Helper()
	driver := missiontest.Connect(t, hnMissionConfig(t))
	raw, err := os.ReadFile("../missions/hacker-news.json")
	mustHN(t, err)
	var definitions []json.RawMessage
	mustHN(t, json.Unmarshal(raw, &definitions))
	if len(definitions) != 1 {
		t.Fatal("expected one starter mission")
	}
	definition, err := missions.LoadMission(definitions[0])
	mustHN(t, err)
	// Use an isolated fixture name so an existing imported demo is never replaced.
	definition.Name += "-test-" + id.New()
	driver.ImportMission(definition)
	driver.WaitForActions(definition.RequiredActions...)
	return driver, definition.Name
}

func waitForHNChain(t *testing.T, driver *missiontest.Driver, operationID, rootID string, names ...string) map[string]operations.Task {
	t.Helper()
	op := driver.WaitFor(operationID, func(op operations.Operation) bool {
		for _, task := range op.Tasks {
			if task.Status == operations.TaskFailed || task.Status == operations.TaskCancelled || task.Status == operations.TaskInterrupted {
				t.Fatalf("%s task %s", task.ActionName, task.Status)
			}
			if task.Status != operations.TaskSucceeded {
				return false
			}
		}
		return len(op.Tasks) == len(names)
	})
	tasks := make(map[string]operations.Task, len(op.Tasks))
	for _, task := range op.Tasks {
		tasks[task.ActionName] = task
	}
	parentID := ""
	for _, name := range names {
		task, ok := tasks[name]
		if !ok || task.ParentTaskID != parentID || !strings.HasPrefix(task.AgentID, "example-agent-") || task.Result == nil {
			t.Fatalf("controller lost the proposed action chain at %s", name)
		}
		parentID = task.ID
	}
	if tasks[names[0]].ID != rootID {
		t.Fatal("action chain did not start from the assigned root task")
	}
	return tasks
}

func downloadHNDigest(t *testing.T, driver *missiontest.Driver, task operations.Task) map[string][]byte {
	t.Helper()
	if task.Result == nil || len(task.Result.Artifacts) != 1 || len(task.Result.Artifacts[0].Files) != 2 {
		t.Fatal("digest did not produce a two-file downloadable artifact")
	}
	var output actions.WriteHNDigestOutput
	mustHN(t, json.Unmarshal(task.Result.Output, &output))
	if !reflect.DeepEqual(output.FileNames, []string{"digest.md", "stories.json"}) {
		t.Fatal("digest metadata does not name both downloadable files")
	}
	artifact := task.Result.Artifacts[0]
	files := map[string][]byte{}
	for _, file := range artifact.Files {
		// Download verifies the accepted size and SHA-256 before returning bytes.
		files[file.Name] = driver.Download(artifact.ID, file)
	}
	return files
}

func completeHN(t *testing.T, driver *missiontest.Driver, operationID, lastTaskID, summary string) {
	t.Helper()
	// The scenario explicitly finishes the final review and completes the operation.
	driver.Resolve(operationID, driver.WaitForContinuation(operationID, lastTaskID))
	completed := driver.Complete(operationID, map[string]string{"summary": summary})
	report := driver.Report(operationID)
	if completed.Status != operations.Completed || completed.ReportID != report.ID || !bytes.Equal(completed.Output, report.Output) {
		t.Fatal("completed operation did not retain its matching final report")
	}
}
