// Package hn reads bounded story metadata from the public Hacker News API.
package hn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type Story struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Score    int    `json:"score"`
	Comments int    `json:"descendants"`
	Type     string `json:"type"`
	Deleted  bool   `json:"deleted"`
	Dead     bool   `json:"dead"`
}

func Fetch(ctx context.Context, client *http.Client, feed string, limit int) ([]Story, error) {
	configured := http.Client{}
	if client != nil {
		configured = *client
	}
	if configured.Timeout <= 0 || configured.Timeout > 5*time.Second {
		configured.Timeout = 5 * time.Second
	}
	configured.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	get := func(path string, output any) error {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://hacker-news.firebaseio.com/v0/"+path+".json", nil)
		if err != nil {
			return errors.New("could not prepare Hacker News request")
		}
		request.Header.Set("User-Agent", "commander-example-agent")
		response, err := configured.Do(request)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return errors.New("Hacker News request failed")
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("Hacker News returned HTTP %d", response.StatusCode)
		}
		data, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024+1))
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return errors.New("could not read Hacker News response")
		}
		if len(data) > 1024*1024 || !utf8.Valid(data) || json.Unmarshal(data, output) != nil {
			return errors.New("Hacker News returned an invalid or oversized response")
		}
		return nil
	}
	var ids []int
	if err := get(feed+"stories", &ids); err != nil {
		return nil, err
	}
	if ids == nil || len(ids) > 500 {
		return nil, errors.New("Hacker News returned an invalid feed")
	}
	if len(ids) > limit {
		ids = ids[:limit]
	}
	stories := []Story{}
	seen := make(map[int]bool)
	for _, id := range ids {
		if id <= 0 {
			return nil, errors.New("Hacker News returned an invalid story ID")
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		var story *Story
		if err := get("item/"+strconv.Itoa(id), &story); err != nil {
			return nil, err
		}
		if story == nil || story.Deleted || story.Dead || story.Type != "story" || strings.TrimSpace(story.Title) == "" {
			continue
		}
		if story.ID != id {
			return nil, errors.New("Hacker News returned a mismatched story ID")
		}
		story.Title = html.UnescapeString(story.Title)
		stories = append(stories, *story)
	}
	return stories, nil
}
