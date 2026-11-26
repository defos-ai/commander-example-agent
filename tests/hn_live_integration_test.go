//go:build integration

package exampleagent_test

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/defos-ai/commander-example-agent/actions"
	"github.com/defos-ai/commander/framework/domain/operations"
	"github.com/defos-ai/commander/framework/testing/missiontest"
)

// Opt into real public HN access using the same dedicated stack as the saved-data
// scenario. Ordinary CI remains deterministic and requires no model calls.
func TestAgentLiveHNThroughAPI(t *testing.T) {
	if os.Getenv("COMMANDER_HN_LIVE") != "1" {
		t.Skip("set COMMANDER_HN_LIVE=1 to fetch live public Hacker News metadata")
	}
	driver, mission := connectHNMission(t)
	names := []string{actions.ChooseHNTopicsName, actions.FetchHNStoriesName, actions.FilterHNStoriesName, actions.RankHNStoriesName, actions.WriteHNDigestName}
	input := map[string]any{"feed": "top", "limit": 3}
	op := driver.Start(mission, input)
	root := driver.Assign(op.ID, missiontest.Assignment{ActionName: names[0], Input: input})
	var questionID string
	driver.WaitFor(op.ID, func(op operations.Operation) bool {
		for _, question := range op.Questions {
			if question.TaskID == root.ID && question.Key == "hn-topics" && question.Pending() {
				questionID = question.ID
				return true
			}
		}
		return false
	})
	driver.Answer(op.ID, questionID, "all")
	tasks := waitForHNChain(t, driver, op.ID, root.ID, names...)
	questions := driver.Get(op.ID).Questions
	if len(questions) != 1 || questions[0].ID != questionID || questions[0].TaskID != root.ID || questions[0].Status != operations.QuestionAnswered || questions[0].Answer != "all" || questions[0].AnsweredBy != operations.AnswerOperator {
		t.Fatal("operator topic answer was not persisted")
	}
	var fetched, filtered, ranked actions.HNStoryBatch
	for name, batch := range map[string]*actions.HNStoryBatch{names[1]: &fetched, names[2]: &filtered, names[3]: &ranked} {
		mustHN(t, json.Unmarshal(tasks[name].Result.Output, batch))
		mustHN(t, batch.Validate())
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
	files := downloadHNDigest(t, driver, digest)
	var saved actions.HNStoryBatch
	mustHN(t, json.Unmarshal(files["stories.json"], &saved))
	if !reflect.DeepEqual(saved, ranked) || !strings.HasPrefix(string(files["digest.md"]), "# Hacker News reading digest\n") {
		t.Fatal("live artifact contents differ from the ranked snapshot")
	}
	for _, story := range ranked.Stories {
		if !strings.Contains(string(files["digest.md"]), fmt.Sprintf("https://news.ycombinator.com/item?id=%d", story.ID)) {
			t.Fatal("live Markdown digest omitted a discussion link")
		}
	}
	waitForHNProgress(t, op.ID, tasks[names[1]].ID)
	completeHN(t, driver, op.ID, digest.ID, "Fetched live Hacker News metadata and saved Markdown and JSON digests.")
	t.Logf("read %d live stories through all five actions and verified both artifacts", len(fetched.Stories))
}

// missiontest does not expose execution logs. Poll only this read-only endpoint;
// operation mutations, decisions, evidence downloads and cleanup use the driver.
func waitForHNProgress(t *testing.T, operationID, taskID string) {
	t.Helper()
	config := hnMissionConfig(t)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	endpoint := strings.TrimRight(config.URL, "/") + "/api/v1/operations/" + operationID + "/logs?task_id=" + taskID
	for {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		mustHN(t, err)
		request.SetBasicAuth(config.Credentials.Username, config.Credentials.Password)
		response, err := config.HTTPClient.Do(request)
		if err != nil {
			t.Fatal("could not observe saved HN progress logs")
		}
		var logs struct {
			Items []struct{ Message string } `json:"items"`
		}
		err = json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&logs)
		response.Body.Close()
		if response.StatusCode != http.StatusOK || err != nil {
			t.Fatal("could not decode saved HN progress logs")
		}
		messages := map[string]bool{}
		for _, log := range logs.Items {
			messages[log.Message] = true
		}
		if messages["Fetch Hacker News started"] && messages["Fetch Hacker News completed"] {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal("saved HN progress logs did not arrive")
		case <-time.After(100 * time.Millisecond):
		}
	}
}
