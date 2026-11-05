package hn

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/defos-ai/commander/framework/agent/action"
)

const feedCheckpoint = "hn-feed-v1"

type feedSnapshot struct {
	Feed  string `json:"feed"`
	Limit int    `json:"limit"`
	IDs   []int  `json:"ids"`
}

func (snapshot feedSnapshot) valid(feed string, limit int) bool {
	if snapshot.Feed != feed || snapshot.Limit != limit || snapshot.IDs == nil || len(snapshot.IDs) > limit {
		return false
	}
	for _, id := range snapshot.IDs {
		if id <= 0 {
			return false
		}
	}
	return true
}

type itemSnapshot struct {
	ID      int  `json:"id"`
	Skipped bool `json:"skipped"`
	// Bytes use base64 in the checkpoint envelope. This bounds HTML escaping of
	// public titles/URLs while preserving their exact normalized metadata.
	Data []byte `json:"data"`
}

func loadSnapshot(ctx context.Context, name string, output any) (bool, error) {
	raw, found, err := action.LoadCheckpoint(ctx, name)
	if err != nil || !found {
		return found, err
	}
	if err := decodeSnapshot(raw, output); err != nil {
		return false, errors.New("invalid Hacker News checkpoint")
	}
	return true, nil
}

func decodeSnapshot(data []byte, output any) error {
	if !utf8.Valid(data) {
		return errors.New("checkpoint contains invalid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return errors.New("checkpoint contains extra JSON")
	}
	return nil
}

func saveSnapshot(ctx context.Context, name string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return action.SaveCheckpoint(ctx, name, raw)
}

func (snapshot itemSnapshot) story(id int) (*Story, error) {
	if snapshot.ID != id || snapshot.Skipped != (snapshot.Data == nil) {
		return nil, errors.New("invalid Hacker News item checkpoint")
	}
	if snapshot.Skipped {
		return nil, nil
	}
	var story Story
	if decodeSnapshot(snapshot.Data, &story) != nil || story.ID != id || !validStory(story) {
		return nil, errors.New("invalid Hacker News story checkpoint")
	}
	return &story, nil
}

func snapshotItem(id int, story *Story) (itemSnapshot, error) {
	snapshot := itemSnapshot{ID: id, Skipped: story == nil}
	if story != nil {
		var data bytes.Buffer
		encoder := json.NewEncoder(&data)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(story); err != nil {
			return itemSnapshot{}, err
		}
		snapshot.Data = data.Bytes()
	}
	return snapshot, nil
}

// Validate normalized data before persisting it, including replayed snapshots.
func validStory(story Story) bool {
	if story.ID <= 0 || story.Type != "story" || story.Dead || story.Deleted || story.Score < 0 || story.Comments < 0 ||
		strings.TrimSpace(story.Title) == "" || len(story.Title) > 1024 || !utf8.ValidString(story.Title) {
		return false
	}
	if story.URL != "" {
		link, err := url.Parse(story.URL)
		return err == nil && len(story.URL) <= 2048 && utf8.ValidString(story.URL) && link.Hostname() != "" && link.User == nil && (link.Scheme == "https" || link.Scheme == "http")
	}
	return true
}
