//go:build integration

package exampleagent_test

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/defos-ai/commander-example-agent/actions"
	"github.com/defos-ai/commander/framework/id"
)

// This optional smoke test needs public HN access and the same disposable
// ENV=tests stack as TestAgentThroughAPI. Ordinary CI stays offline and repeatable.
func TestAgentLiveHNThroughAPI(t *testing.T) {
	if os.Getenv("COMMANDER_HN_LIVE") != "1" {
		t.Skip("set COMMANDER_HN_LIVE=1 to fetch live public Hacker News metadata")
	}
	a := connectAPI(t)
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	a.request(t, ctx, http.MethodPost, "/api/v1/testing/cleanup", nil, http.StatusNoContent, nil)
	names := []string{"choose-hn-topics", "fetch-hn-stories", "filter-hn-stories", "rank-hn-stories", "write-hn-digest"}
	await(t, ctx, func() bool {
		var agents agentList
		a.request(t, ctx, http.MethodGet, "/api/v1/agents", nil, http.StatusOK, &agents)
		for _, agent := range agents.Items {
			if agent.Type != "example-agent" || agent.Status != "ready" || agent.Stale {
				continue
			}
			ready := map[string]bool{}
			for _, action := range agent.Actions {
				ready[action.Name] = action.Status == "ready"
			}
			if slices.ContainsFunc(names, func(name string) bool { return !ready[name] }) {
				continue
			}
			return true
		}
		return false
	})
	operationID := id.New()
	base := "/api/v1/operations/" + operationID
	t.Cleanup(func() {
		if err := a.cleanup(operationID); err != nil {
			t.Errorf("clean up live HN operation: %v", err)
		}
	})
	input := map[string]any{"feed": "top", "limit": 3}
	var operation operationResponse
	a.request(t, ctx, http.MethodPost, "/api/v1/operations", map[string]any{
		"operation_id": operationID, "mission_name": "hacker-news-digest", "input": input,
	}, http.StatusCreated, &operation)
	var root taskResponse
	a.request(t, ctx, http.MethodPost, base+"/tasks", map[string]any{
		"action_name": names[0], "input": input,
	}, http.StatusCreated, &root)
	var questions struct {
		Items []struct {
			ID         string `json:"id"`
			TaskID     string `json:"task_id"`
			Key        string `json:"key"`
			Status     string `json:"status"`
			Answer     string `json:"answer"`
			AnsweredBy string `json:"answered_by"`
		} `json:"items"`
	}
	// Persist the operator answer through the API, exercising the real action
	// question provider rather than injecting a canned answer into the action.
	var questionID string
	await(t, ctx, func() bool {
		a.request(t, ctx, http.MethodGet, base+"/questions", nil, http.StatusOK, &questions)
		for _, question := range questions.Items {
			if question.Key == "hn-topics" && question.Status == "pending" {
				questionID = question.ID
				return true
			}
		}
		return false
	})
	a.request(t, ctx, http.MethodPost, base+"/questions/"+questionID+"/answer", map[string]string{"answer": "all"}, http.StatusOK, nil)
	await(t, ctx, func() bool {
		a.request(t, ctx, http.MethodGet, base, nil, http.StatusOK, &operation)
		for _, task := range operation.Tasks {
			if task.Status == "failed" || task.Status == "cancelled" {
				t.Fatalf("live %s task %s: %s", task.ActionName, task.Status, task.Error)
			}
			if task.Status != "succeeded" {
				return false
			}
		}
		return len(operation.Tasks) == len(names)
	})
	tasks := map[string]taskResponse{}
	for _, task := range operation.Tasks {
		tasks[task.ActionName] = task
	}
	parentID := ""
	for _, name := range names {
		task, ok := tasks[name]
		if !ok || task.ParentTaskID != parentID || !strings.HasPrefix(task.AgentID, "example-agent-") || task.Result == nil {
			t.Fatalf("live controller proposals lost their action chain at %s", name)
		}
		parentID = task.ID
	}
	if tasks[names[0]].ID != root.ID {
		t.Fatal("live action chain did not start from the assigned topic task")
	}
	a.request(t, ctx, http.MethodGet, base+"/questions", nil, http.StatusOK, &questions)
	if len(questions.Items) != 1 || questions.Items[0].ID != questionID || questions.Items[0].TaskID != root.ID || questions.Items[0].Status != "answered" || questions.Items[0].Answer != "all" || questions.Items[0].AnsweredBy != "operator" {
		t.Fatal("operator topic answer was not persisted")
	}
	var fetched, filtered, ranked actions.HNStoryBatch
	for name, batch := range map[string]*actions.HNStoryBatch{names[1]: &fetched, names[2]: &filtered, names[3]: &ranked} {
		if err := json.Unmarshal(tasks[name].Result.Output, batch); err != nil {
			t.Fatal(err)
		}
		if err := batch.Validate(); err != nil {
			t.Fatalf("live %s output: %v", name, err)
		}
	}
	if fetched.Feed != "top" || len(fetched.Topics) != 0 || len(fetched.Stories) > 3 || !reflect.DeepEqual(fetched, filtered) {
		t.Fatal("live fetch ignored its bounds or all-topic filtering changed the snapshot")
	}
	want := fetched
	want.Stories = slices.Clone(fetched.Stories)
	slices.SortFunc(want.Stories, func(a, b actions.HNStory) int {
		if n := cmp.Compare(b.Score, a.Score); n != 0 {
			return n
		}
		if n := cmp.Compare(b.Comments, a.Comments); n != 0 {
			return n
		}
		return cmp.Compare(a.ID, b.ID)
	})
	if !reflect.DeepEqual(ranked, want) {
		t.Fatal("live ranking differs from the collected story metadata")
	}
	digest := tasks[names[4]]
	if len(digest.Result.Artifacts) != 1 || len(digest.Result.Artifacts[0].Files) != 2 {
		t.Fatal("live digest did not produce both artifact files")
	}
	files := map[string][]byte{}
	artifact := digest.Result.Artifacts[0]
	for _, file := range artifact.Files {
		content := a.request(t, ctx, http.MethodGet, "/api/v1/artifacts/"+artifact.ID+"/files/"+file.ID, nil, http.StatusOK, nil)
		hash := sha256.Sum256(content)
		if file.SHA256 != hex.EncodeToString(hash[:]) {
			t.Fatalf("downloaded live %s does not match its hash", file.Name)
		}
		files[file.Name] = content
	}
	var saved actions.HNStoryBatch
	if json.Unmarshal(files["stories.json"], &saved) != nil || !reflect.DeepEqual(saved, ranked) || !strings.HasPrefix(string(files["digest.md"]), "# Hacker News reading digest\n") {
		t.Fatal("live artifact contents differ from the ranked snapshot")
	}
	for _, story := range ranked.Stories {
		if !strings.Contains(string(files["digest.md"]), fmt.Sprintf("https://news.ycombinator.com/item?id=%d", story.ID)) {
			t.Fatal("live Markdown digest omitted a discussion link")
		}
	}
	// Progress logs travel through the v0.4.0 execution protocol independently of
	// the accepted task result, so wait for their durable API view separately.
	await(t, ctx, func() bool {
		var logs struct {
			Items []struct{ Message string } `json:"items"`
		}
		a.request(t, ctx, http.MethodGet, base+"/logs?task_id="+tasks[names[1]].ID, nil, http.StatusOK, &logs)
		messages := map[string]bool{}
		for _, log := range logs.Items {
			messages[log.Message] = true
		}
		return messages["Fetch Hacker News started"] && messages["Fetch Hacker News completed"]
	})
	a.request(t, ctx, http.MethodPost, base+"/complete", map[string]any{
		"output": map[string]string{"summary": "Fetched live Hacker News metadata and saved Markdown and JSON digests."},
	}, http.StatusOK, &operation)
	if operation.Status != "completed" || operation.ReportID == "" {
		t.Fatal("live HN operation did not produce its final report")
	}
	t.Logf("read %d live stories through all five actions and verified both artifacts", len(fetched.Stories))
}
