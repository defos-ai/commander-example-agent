package exampleagent_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/defos-ai/commander-example-agent/actions"
	"github.com/defos-ai/commander/framework/agent/action"
	"github.com/defos-ai/commander/framework/domain/operations"
)

func TestHNQuestionAndInputValidation(t *testing.T) {
	choose, err := actions.NewChooseHNTopics()
	mustHN(t, err)
	discover(t, choose)
	if result, err := choose.Run(t.Context(), json.RawMessage(`{}`)); !errors.Is(err, action.ErrQuestionsUnavailable) || !hnEmpty(result) {
		t.Fatal("choose must require an execution question")
	}
	invalidInputContext := action.WithQuestions(t.Context(), hnQuestionFunc(func(context.Context, string, string, []string) (operations.Question, error) {
		t.Fatal("invalid feed input requested an operator answer")
		return operations.Question{}, nil
	}))
	for _, input := range []string{`null`, `{"feed":"jobs"}`, `{"limit":0}`, `{"limit":21}`, `{"extra":true}`} {
		if result, err := choose.Run(invalidInputContext, json.RawMessage(input)); err == nil || !hnEmpty(result) {
			t.Fatal("invalid feed selection produced a result")
		}
	}
	for _, answer := range []string{" ", "Go,,AI", "one,two,three,four,five,six", strings.Repeat("a", 41)} {
		if result, err := choose.Run(hnTopics(t, answer), json.RawMessage(`{}`)); err == nil || !hnEmpty(result) {
			t.Fatal("invalid topics produced a result")
		}
	}
	result, err := choose.Run(hnTopics(t, "all"), json.RawMessage(`{}`))
	mustHN(t, err)
	var selected actions.HNFetchInput
	mustHN(t, json.Unmarshal(result.Output, &selected))
	if selected.Topics == nil || len(selected.Topics) != 0 {
		t.Fatal("all topics must be an empty array")
	}
	writer, err := actions.NewWriteHNDigest()
	mustHN(t, err)
	discover(t, writer)
	valid := actions.HNStoryBatch{Feed: "top", Topics: []string{}, Stories: []actions.HNStory{{ID: 1, Title: "A story"}}}
	for _, change := range []func(*actions.HNStoryBatch){
		func(b *actions.HNStoryBatch) { b.Topics = nil },
		func(b *actions.HNStoryBatch) { b.Stories = nil },
		func(b *actions.HNStoryBatch) { b.Stories[0].ID = 0 },
		func(b *actions.HNStoryBatch) { b.Stories[0].Score = -1 },
		func(b *actions.HNStoryBatch) { b.Stories[0].Title = strings.Repeat("a", 1025) },
		func(b *actions.HNStoryBatch) { b.Stories[0].URL = "javascript:alert(1)" },
		func(b *actions.HNStoryBatch) { b.Stories = append(b.Stories, b.Stories[0]) },
	} {
		batch := valid
		batch.Stories = append([]actions.HNStory(nil), valid.Stories...)
		change(&batch)
		if result, err := writer.Run(t.Context(), hnJSON(t, batch)); err == nil || !hnEmpty(result) {
			t.Fatal("invalid snapshot produced a digest")
		}
	}
}

func TestHNFetchFailuresAndBounds(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
	}{
		{"HTTP failure", 503, "upstream unavailable"}, {"redirect", 302, ""},
		{"invalid JSON", 200, "{"}, {"invalid feed", 200, `{"id":1}`},
		{"oversized feed", 200, `[1]` + strings.Repeat(" ", 1024*1024)},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			client := hnHTTP(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if strings.HasPrefix(r.URL.Path, "/v0/item/") {
					_, _ = w.Write([]byte(`{"id":1,"type":"story","title":"One"}`))
					return
				}
				w.Header().Set("Location", "https://another-host.invalid/")
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			fetch, err := actions.NewFetchHNStories(client)
			mustHN(t, err)
			discover(t, fetch)
			input := actions.HNFetchInput{Feed: "top", Limit: 1, Topics: []string{}}
			if result, err := fetch.Run(t.Context(), hnJSON(t, input)); err == nil || !hnEmpty(result) || calls.Load() != 1 {
				t.Fatal("failed fetch produced work or followed a redirect")
			}
		})
	}
	var calls atomic.Int32
	client := hnHTTP(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if strings.HasSuffix(r.URL.Path, "stories.json") {
			_, _ = w.Write([]byte(`[1,2,3]`))
			return
		}
		_, _ = w.Write([]byte(`{"id":1,"type":"story","title":"One"}`))
	}))
	fetch, err := actions.NewFetchHNStories(client)
	mustHN(t, err)
	discover(t, fetch)
	_, err = fetch.Run(t.Context(), hnJSON(t, actions.HNFetchInput{Feed: "top", Limit: 1, Topics: []string{}}))
	mustHN(t, err)
	if calls.Load() != 2 {
		t.Fatal("fetch read beyond the requested story limit")
	}
}

func TestHNActionsHonorCancellation(t *testing.T) {
	choose, err := actions.NewChooseHNTopics()
	mustHN(t, err)
	fetch, err := actions.NewFetchHNStories(nil)
	mustHN(t, err)
	filter, err := actions.NewFilterHNStories()
	mustHN(t, err)
	rank, err := actions.NewRankHNStories()
	mustHN(t, err)
	write, err := actions.NewWriteHNDigest()
	mustHN(t, err)
	for _, executor := range []action.Executor{choose, fetch, filter, rank, write} {
		discover(t, executor)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		if result, err := executor.Run(ctx, json.RawMessage(`{}`)); !errors.Is(err, context.Canceled) || !hnEmpty(result) {
			t.Fatalf("%s ignored cancellation", executor.Action().Name)
		}
	}
	started := make(chan struct{})
	client := hnHTTP(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	fetch, err = actions.NewFetchHNStories(client)
	mustHN(t, err)
	discover(t, fetch)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	input := hnJSON(t, actions.HNFetchInput{Feed: "top", Limit: 1, Topics: []string{}})
	go func() {
		_, err := fetch.Run(ctx, input)
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("fetch did not reach HTTP request")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("fetch lost cancellation: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("fetch did not stop after cancellation")
	}
}

func hnTopics(t *testing.T, answer string) context.Context {
	t.Helper()
	return action.WithQuestions(t.Context(), hnQuestionFunc(func(_ context.Context, key, text string, options []string) (operations.Question, error) {
		if key != "hn-topics" || text != "Which topics should the reading list cover? Use all or comma-separated keywords." || strings.Join(options, ",") != "all,Go,AI,security" {
			t.Error("question content changed")
		}
		return operations.Question{Key: key, Text: text, Options: options, Status: operations.QuestionAnswered, Answer: answer, AnsweredBy: operations.AnswerOperator}, nil
	}))
}

type hnQuestionFunc func(context.Context, string, string, []string) (operations.Question, error)

func (f hnQuestionFunc) Question(ctx context.Context, key, text string, options []string) (operations.Question, error) {
	return f(ctx, key, text, options)
}

type hnLogFunc func(context.Context, string) error

func (f hnLogFunc) Log(ctx context.Context, message string) error { return f(ctx, message) }

func hnEmpty(result action.Result) bool {
	return len(result.Output) == 0 && len(result.NextActions) == 0 && len(result.Artifacts) == 0
}

func hnJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	mustHN(t, err)
	return raw
}

func mustHN(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
