package actions

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/defos-ai/commander/framework/agent/action"
)

type HNFeedInput struct {
	Feed  string `json:"feed,omitempty" jsonschema:"Hacker News feed: top, new or best; default top."`
	Limit int    `json:"limit,omitempty" jsonschema:"Number of feed entries to inspect, from 1 to 20; default 10."`
}

func (input HNFeedInput) Validate() error {
	if input.Feed != "top" && input.Feed != "new" && input.Feed != "best" {
		return errors.New("feed must be top, new or best")
	}
	if input.Limit < 1 || input.Limit > 20 {
		return errors.New("limit must be between 1 and 20")
	}
	return nil
}

type HNFetchInput struct {
	Feed   string   `json:"feed"`
	Limit  int      `json:"limit"`
	Topics []string `json:"topics" jsonschema:"Up to five title keywords or phrases; an empty array includes every topic."`
}

func (input HNFetchInput) Validate() error {
	if err := (HNFeedInput{Feed: input.Feed, Limit: input.Limit}).Validate(); err != nil {
		return err
	}
	return validateTopics(input.Topics)
}

type HNStory struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Score    int    `json:"score"`
	Comments int    `json:"comments"`
}

type HNStoryBatch struct {
	Feed    string    `json:"feed"`
	Topics  []string  `json:"topics"`
	Stories []HNStory `json:"stories"`
}

func (batch HNStoryBatch) Validate() error {
	if err := (HNFetchInput{Feed: batch.Feed, Limit: 1, Topics: batch.Topics}).Validate(); err != nil {
		return err
	}
	if batch.Stories == nil || len(batch.Stories) > 20 {
		return errors.New("stories must be an array containing at most 20 entries")
	}
	seen := make(map[int]bool)
	for _, story := range batch.Stories {
		if story.ID <= 0 || seen[story.ID] || story.Score < 0 || story.Comments < 0 {
			return errors.New("story IDs must be positive and unique; counts must be nonnegative")
		}
		seen[story.ID] = true
		if strings.TrimSpace(story.Title) == "" || len(story.Title) > 1024 || !utf8.ValidString(story.Title) {
			return errors.New("story titles must be nonblank UTF-8 text of at most 1024 bytes")
		}
		if story.URL != "" {
			link, err := url.Parse(story.URL)
			if err != nil || len(story.URL) > 2048 || !utf8.ValidString(story.URL) || link.Hostname() == "" || link.User != nil || (link.Scheme != "https" && link.Scheme != "http") {
				return errors.New("story URL must be an HTTP or HTTPS URL without credentials, at most 2048 bytes")
			}
		}
	}
	return nil
}

func validateTopics(topics []string) error {
	if topics == nil || len(topics) > 5 {
		return errors.New("topics must be an array containing at most five entries")
	}
	for _, topic := range topics {
		if normalizedWords(topic) == "" || len(topic) > 40 || !utf8.ValidString(topic) {
			return errors.New("topics must contain words and be valid UTF-8 of at most 40 bytes")
		}
	}
	return nil
}

func normalizedWords(text string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}), " ")
}

func hnProposal(ctx context.Context, name string, input any) ([]action.NextAction, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	return []action.NextAction{{ActionName: name, Input: raw, Reason: "Continue building the Hacker News reading digest."}}, nil
}
