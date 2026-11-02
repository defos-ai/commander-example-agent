package exampleagent_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/defos-ai/commander-example-agent/actions"
	"github.com/defos-ai/commander/framework/agent/action"
)

func TestHNDigestFromQuestionThroughHTTPToArtifacts(t *testing.T) {
	var requests atomic.Int32
	client := hnHTTP(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodGet || r.Host != "hacker-news.firebaseio.com" {
			t.Error("fetch did not use the official read-only API")
		}
		w.Header().Set("Content-Type", "application/json")
		responses := map[string]string{
			"/v0/topstories.json": `[1,2,3,4,5,6,7,8]`,
			"/v0/item/1.json":     `{"id":1,"type":"story","title":"Go &amp; concurrency","url":"https://go.dev/doc/","score":100,"descendants":10}`,
			"/v0/item/2.json":     `{"id":2,"type":"story","title":"Google search news","score":500,"descendants":1}`,
			"/v0/item/3.json":     `{"id":3,"type":"story","title":"AI agents [escape] <script>","url":"https://example.com/a(b)\\","score":100,"descendants":12}`,
			"/v0/item/4.json":     `{"id":4,"type":"story","deleted":true}`,
			"/v0/item/5.json":     `{"id":5,"type":"story","dead":true}`,
			"/v0/item/6.json":     `{"id":6,"type":"comment","text":"not a story"}`,
			"/v0/item/7.json":     `null`,
			"/v0/item/8.json":     `{"id":8,"type":"story","title":"Security in Go","score":90,"descendants":4}`,
		}
		body, ok := responses[r.URL.Path]
		if !ok {
			t.Errorf("unexpected request path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = fmt.Fprint(w, body)
	}))
	choose, err := actions.NewChooseHNTopics()
	mustHN(t, err)
	fetch, err := actions.NewFetchHNStories(client)
	mustHN(t, err)
	filter, err := actions.NewFilterHNStories()
	mustHN(t, err)
	rank, err := actions.NewRankHNStories()
	mustHN(t, err)
	write, err := actions.NewWriteHNDigest()
	mustHN(t, err)
	ctx := hnTopics(t, " Go, AI, security ")
	var logs []string
	ctx = action.WithLogs(ctx, hnLogFunc(func(_ context.Context, message string) error {
		logs = append(logs, message)
		return nil
	}))
	input := json.RawMessage(`{}`)
	executors := []action.Executor{choose, fetch, filter, rank, write}
	var ranked actions.HNStoryBatch
	for i, executor := range executors {
		discover(t, executor)
		result, err := executor.Run(ctx, input)
		mustHN(t, err)
		if i == 0 {
			var selected actions.HNFetchInput
			mustHN(t, json.Unmarshal(result.Output, &selected))
			if selected.Feed != "top" || selected.Limit != 10 || len(selected.Topics) != 3 {
				t.Fatal("topic selection did not apply defaults and parse the saved answer")
			}
		}
		if i == 1 {
			var fetched actions.HNStoryBatch
			mustHN(t, json.Unmarshal(result.Output, &fetched))
			if len(fetched.Stories) != 4 || fetched.Stories[0].Title != "Go & concurrency" {
				t.Fatal("fetch did not skip unavailable items and decode story titles")
			}
		}
		if i < len(executors)-1 {
			if len(result.NextActions) != 1 || result.NextActions[0].ActionName != executors[i+1].Action().Name || len(result.Artifacts) != 0 {
				t.Fatalf("%s did not propose the next stage", executor.Action().Name)
			}
			input = result.NextActions[0].Input
			if i == 3 {
				mustHN(t, json.Unmarshal(input, &ranked))
				if !reflect.DeepEqual(hnIDs(ranked), []int{3, 1, 8}) {
					t.Fatal("topic filtering or score/comment ordering changed")
				}
			}
			continue
		}
		if len(result.NextActions) != 0 || len(result.Artifacts) != 1 || len(result.Artifacts[0].Files) != 2 {
			t.Fatal("digest did not produce two final downloadable files")
		}
		var output actions.WriteHNDigestOutput
		mustHN(t, json.Unmarshal(result.Output, &output))
		if !reflect.DeepEqual(output.FileNames, []string{"digest.md", "stories.json"}) {
			t.Fatal("digest metadata differs from the artifact filenames")
		}
		for _, file := range result.Artifacts[0].Files {
			if file.Path != "" {
				t.Fatal("action wrote an artifact path instead of returning bytes")
			}
			switch file.Name {
			case "digest.md":
				text := string(file.Data)
				if file.MediaType != "text/markdown" || strings.Contains(text, "<script>") || strings.Contains(text, "[escape]") || !strings.Contains(text, "https://news.ycombinator.com/item?id=3") || !strings.Contains(text, "https://example.com/a%28b%29%5C") {
					t.Fatal("digest lacks discussion links or contains unescaped remote text")
				}
			case "stories.json":
				var saved actions.HNStoryBatch
				mustHN(t, json.Unmarshal(file.Data, &saved))
				if file.MediaType != "application/json" || !reflect.DeepEqual(saved, ranked) {
					t.Fatal("JSON artifact differs from the ranked snapshot")
				}
			default:
				t.Fatalf("unexpected artifact file %q", file.Name)
			}
		}
	}
	if requests.Load() != 9 || !reflect.DeepEqual(logs, []string{"Fetch Hacker News started", "Fetch Hacker News completed"}) {
		t.Fatalf("unexpected request count or progress: requests=%d logs=%v", requests.Load(), logs)
	}
}

func TestHNFilteringRankingAndEmptyDigest(t *testing.T) {
	filter, err := actions.NewFilterHNStories()
	mustHN(t, err)
	rank, err := actions.NewRankHNStories()
	mustHN(t, err)
	write, err := actions.NewWriteHNDigest()
	mustHN(t, err)
	for _, executor := range []action.Executor{filter, rank, write} {
		discover(t, executor)
	}
	batch := actions.HNStoryBatch{Feed: "best", Topics: []string{"Go"}, Stories: []actions.HNStory{
		{ID: 7, Title: "Go seven", Score: 5, Comments: 1}, {ID: 6, Title: "Go six", Score: 5, Comments: 2},
		{ID: 5, Title: "Go five", Score: 6}, {ID: 4, Title: "Go four", Score: 6, Comments: 2},
		{ID: 3, Title: "Go three", Score: 6, Comments: 2}, {ID: 2, Title: "Go two", Score: 7},
		{ID: 1, Title: "Google should not match", Score: 1000},
	}}
	filtered, err := filter.Run(t.Context(), hnJSON(t, batch))
	mustHN(t, err)
	ranked, err := rank.Run(t.Context(), filtered.NextActions[0].Input)
	mustHN(t, err)
	var result actions.HNStoryBatch
	mustHN(t, json.Unmarshal(ranked.Output, &result))
	if !reflect.DeepEqual(hnIDs(result), []int{2, 3, 4, 5, 6}) {
		t.Fatal("ranking did not apply deterministic ties and the five-story limit")
	}
	batch.Topics = []string{"machine learning"}
	batch.Stories = []actions.HNStory{{ID: 1, Title: "MACHINE-learning notes"}, {ID: 2, Title: "Machine learners"}}
	filtered, err = filter.Run(t.Context(), hnJSON(t, batch))
	mustHN(t, err)
	mustHN(t, json.Unmarshal(filtered.Output, &result))
	if !reflect.DeepEqual(hnIDs(result), []int{1}) {
		t.Fatal("phrase matching did not normalize case and punctuation")
	}
	batch.Topics = []string{"Rust"}
	filtered, err = filter.Run(t.Context(), hnJSON(t, batch))
	mustHN(t, err)
	ranked, err = rank.Run(t.Context(), filtered.NextActions[0].Input)
	mustHN(t, err)
	mustHN(t, json.Unmarshal(ranked.Output, &result))
	if result.Stories == nil || len(result.Stories) != 0 {
		t.Fatal("unmatched topics must produce an empty story array")
	}
	empty, err := write.Run(t.Context(), ranked.NextActions[0].Input)
	mustHN(t, err)
	if len(empty.Artifacts) != 1 {
		t.Fatal("no matches should still produce a digest")
	}
	batch.Topics = []string{}
	filtered, err = filter.Run(t.Context(), hnJSON(t, batch))
	mustHN(t, err)
	mustHN(t, json.Unmarshal(filtered.Output, &result))
	if len(result.Stories) != 2 {
		t.Fatal("all topics should preserve every fetched story")
	}
}

func hnIDs(batch actions.HNStoryBatch) []int {
	ids := make([]int, 0, len(batch.Stories))
	for _, story := range batch.Stories {
		ids = append(ids, story.ID)
	}
	return ids
}

func discover(t *testing.T, executor action.Executor) {
	t.Helper()
	_, err := executor.Discover(t.Context(), nil, nil)
	mustHN(t, err)
}
