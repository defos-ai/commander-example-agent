package exampleagent_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/defos-ai/commander-example-agent/actions"
	"github.com/defos-ai/commander/framework/agent/action"
)

func TestHNFetchCheckpointsResumeAndReplay(t *testing.T) {
	var mu sync.Mutex
	calls := map[string]int{}
	client := hnHTTP(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		calls[r.URL.Path]++
		responses := map[string]string{
			"/v0/topstories.json": `[1,2,3,4,5,6,6,7]`,
			"/v0/item/1.json":     `{"id":1,"type":"story","title":"First &amp; original","score":10}`,
			"/v0/item/2.json":     `null`,
			"/v0/item/3.json":     `{"id":3,"type":"story","dead":true}`,
			"/v0/item/4.json":     `{"id":4,"type":"story","deleted":true}`,
			"/v0/item/5.json":     `{"id":5,"type":"comment","title":"Comment"}`,
			"/v0/item/6.json":     `{"id":6,"type":"story","title":" "}`,
			"/v0/item/7.json":     `{"id":7,"type":"story","title":"Last","score":20}`,
		}
		if r.URL.Path == "/v0/item/7.json" && calls[r.URL.Path] == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprint(w, responses[r.URL.Path])
	}))
	fetch, err := actions.NewFetchHNStories(client)
	mustHN(t, err)
	discover(t, fetch)
	store := &hnCheckpointMemory{data: map[string]json.RawMessage{}}
	ctx := action.WithCheckpoints(t.Context(), store)
	input := hnJSON(t, actions.HNFetchInput{Feed: "top", Limit: 8, Topics: []string{}})
	if result, err := fetch.Run(ctx, input); err == nil || !hnEmpty(result) {
		t.Fatal("interrupted fetch produced a proposal")
	}
	result, err := fetch.Run(ctx, input)
	mustHN(t, err)
	var batch actions.HNStoryBatch
	mustHN(t, json.Unmarshal(result.Output, &batch))
	if !reflect.DeepEqual(hnIDs(batch), []int{1, 7}) || batch.Stories[0].Title != "First & original" || len(result.NextActions) != 1 {
		t.Fatal("resumed snapshot lost normalized stories or skipped-item outcomes")
	}
	replayed, err := fetch.Run(ctx, input)
	mustHN(t, err)
	if !bytes.Equal(replayed.Output, result.Output) || len(store.data) != 8 {
		t.Fatal("replay changed the fixed snapshot or did not checkpoint skipped items")
	}
	mu.Lock()
	defer mu.Unlock()
	for path, count := range calls {
		want := 1
		if path == "/v0/item/7.json" {
			want = 2
		}
		if count != want {
			t.Fatalf("%s requested %d times, want %d; checkpointed items must not be fetched again", path, count, want)
		}
	}
}

func TestHNFetchCheckpointErrorsStopProposals(t *testing.T) {
	sentinel := errors.New("checkpoint persistence failed")
	for _, test := range []struct {
		name               string
		loadFail, saveFail int
		wantRequests       int
	}{
		{"feed load", 1, 0, 0},
		{"feed save", 0, 1, 1},
		{"item load", 2, 0, 1},
		{"item save", 0, 2, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			requests := 0
			client := hnHTTP(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if strings.HasSuffix(r.URL.Path, "stories.json") {
					_, _ = fmt.Fprint(w, `[1]`)
				} else {
					_, _ = fmt.Fprint(w, `{"id":1,"type":"story","title":"One"}`)
				}
			}))
			fetch, err := actions.NewFetchHNStories(client)
			mustHN(t, err)
			discover(t, fetch)
			store := &hnCheckpointMemory{data: map[string]json.RawMessage{}, loadFail: test.loadFail, saveFail: test.saveFail, failure: sentinel}
			result, err := fetch.Run(action.WithCheckpoints(t.Context(), store), hnJSON(t, actions.HNFetchInput{Feed: "top", Limit: 1, Topics: []string{}}))
			if !errors.Is(err, sentinel) || !hnEmpty(result) || requests != test.wantRequests {
				t.Fatalf("checkpoint failure continued fetching or proposed work: err=%v requests=%d", err, requests)
			}
		})
	}
}

func TestHNFetchWithoutCheckpointsKeepsIndependentRequests(t *testing.T) {
	var requests atomic.Int32
	client := hnHTTP(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if strings.HasSuffix(r.URL.Path, "stories.json") {
			_, _ = fmt.Fprint(w, `[1]`)
		} else {
			_, _ = fmt.Fprint(w, `{"id":1,"type":"story","title":"One"}`)
		}
	}))
	fetch, err := actions.NewFetchHNStories(client)
	mustHN(t, err)
	discover(t, fetch)
	input := hnJSON(t, actions.HNFetchInput{Feed: "top", Limit: 1, Topics: []string{}})
	for range 2 {
		result, err := fetch.Run(t.Context(), input)
		mustHN(t, err)
		if len(result.NextActions) != 1 {
			t.Fatal("standalone fetch did not propose filtering")
		}
	}
	if requests.Load() != 4 {
		t.Fatal("independent standalone calls shared a cached snapshot")
	}
}

func TestHNFetchRejectsCorruptOrMismatchedCheckpoints(t *testing.T) {
	validFeed := json.RawMessage(`{"feed":"top","limit":1,"ids":[1]}`)
	for _, test := range []struct {
		name, checkpoint string
		data             json.RawMessage
	}{
		{"feed binding", "hn-feed-v1", json.RawMessage(`{"feed":"best","limit":1,"ids":[1]}`)},
		{"limit binding", "hn-feed-v1", json.RawMessage(`{"feed":"top","limit":2,"ids":[1]}`)},
		{"null IDs", "hn-feed-v1", json.RawMessage(`{"feed":"top","limit":1,"ids":null}`)},
		{"invalid ID", "hn-feed-v1", json.RawMessage(`{"feed":"top","limit":1,"ids":[0]}`)},
		{"extra fields", "hn-feed-v1", json.RawMessage(`{"feed":"top","limit":1,"ids":[1],"extra":true}`)},
		{"item binding", "hn-item-1-v1", json.RawMessage(`{"id":2,"skipped":true,"data":null}`)},
		{"missing outcome", "hn-item-1-v1", json.RawMessage(`{"id":1}`)},
		{"invalid encoded data", "hn-item-1-v1", json.RawMessage(`{"id":1,"skipped":false,"data":"not base64"}`)},
		{"invalid UTF-8 metadata", "hn-item-1-v1", hnJSON(t, map[string]any{"id": 1, "skipped": false, "data": []byte("{\"id\":1,\"type\":\"story\",\"title\":\"\xff\"}")})},
		{"invalid metadata", "hn-item-1-v1", hnJSON(t, map[string]any{"id": 1, "skipped": false, "data": []byte(`{"id":1,"type":"story","title":"One","score":-1}`)})},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := hnHTTP(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("corrupt checkpoint triggered an HTTP request")
				w.WriteHeader(http.StatusServiceUnavailable)
			}))
			fetch, err := actions.NewFetchHNStories(client)
			mustHN(t, err)
			discover(t, fetch)
			store := &hnCheckpointMemory{data: map[string]json.RawMessage{"hn-feed-v1": validFeed}}
			store.data[test.checkpoint] = test.data
			result, err := fetch.Run(action.WithCheckpoints(t.Context(), store), hnJSON(t, actions.HNFetchInput{Feed: "top", Limit: 1, Topics: []string{}}))
			if err == nil || !hnEmpty(result) {
				t.Fatal("corrupt checkpoint produced a proposal")
			}
		})
	}
}

func TestHNFetchCheckpointsFitMaximumValidMetadata(t *testing.T) {
	// HTML and control characters can expand when JSON is put in an envelope.
	for _, title := range []string{strings.Repeat("<", 1024), strings.Repeat("\x01", 1023) + "a"} {
		t.Run(fmt.Sprintf("title-%x", title[0]), func(t *testing.T) {
			link := "https://example.com/" + strings.Repeat("<", 2048-len("https://example.com/"))
			client := hnHTTP(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "stories.json") {
					_, _ = fmt.Fprint(w, `[1]`)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "type": "story", "title": title, "url": link})
			}))
			fetch, err := actions.NewFetchHNStories(client)
			mustHN(t, err)
			discover(t, fetch)
			store := &hnCheckpointMemory{data: map[string]json.RawMessage{}}
			ctx := action.WithCheckpoints(t.Context(), store)
			input := hnJSON(t, actions.HNFetchInput{Feed: "top", Limit: 1, Topics: []string{}})
			first, err := fetch.Run(ctx, input)
			mustHN(t, err)
			second, err := fetch.Run(ctx, input)
			mustHN(t, err)
			var batch actions.HNStoryBatch
			mustHN(t, json.Unmarshal(second.Output, &batch))
			if !bytes.Equal(first.Output, second.Output) || len(batch.Stories) != 1 || batch.Stories[0].Title != title || batch.Stories[0].URL != link {
				t.Fatal("checkpoint encoding changed valid public metadata")
			}
		})
	}
}

type hnCheckpointMemory struct {
	data               map[string]json.RawMessage
	loads, saves       int
	loadFail, saveFail int
	failure            error
}

func (store *hnCheckpointMemory) LoadCheckpoint(_ context.Context, name string) (json.RawMessage, bool, error) {
	store.loads++
	if store.loads == store.loadFail {
		return nil, false, store.failure
	}
	data, found := store.data[name]
	return bytes.Clone(data), found, nil
}

func (store *hnCheckpointMemory) SaveCheckpoint(_ context.Context, name string, data json.RawMessage) error {
	store.saves++
	if store.saves == store.saveFail {
		return store.failure
	}
	store.data[name] = bytes.Clone(data)
	return nil
}
