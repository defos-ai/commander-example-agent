package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/defos-ai/commander/framework/agent/action"
	actionmodel "github.com/defos-ai/commander/framework/domain/actions"
	"github.com/defos-ai/commander/framework/domain/artifacts"
)

const WriteHNDigestName = "write-hn-digest"
const WriteHNDigestDescription = "Save a Hacker News reading list as digest.md and stories.json artifacts."

type WriteHNDigestOutput struct {
	FileNames []string `json:"file_names"`
}

type WriteHNDigest struct{ *action.Base }

func NewWriteHNDigest() (*WriteHNDigest, error) {
	input, err := action.SchemaFor[HNStoryBatch]()
	if err != nil {
		return nil, err
	}
	output, err := action.SchemaFor[WriteHNDigestOutput]()
	if err != nil {
		return nil, err
	}
	return &WriteHNDigest{action.NewBase(actionmodel.Action{Name: WriteHNDigestName, Description: WriteHNDigestDescription, InputSchema: input, OutputSchema: output})}, nil
}

func (a *WriteHNDigest) Run(ctx context.Context, raw json.RawMessage) (action.Result, error) {
	if err := ctx.Err(); err != nil {
		return action.Result{}, err
	}
	if err := a.Check(); err != nil {
		return action.Result{}, err
	}
	var batch HNStoryBatch
	if err := action.DecodeInput(raw, &batch); err != nil {
		return action.Result{}, err
	}
	if err := batch.Validate(); err != nil {
		return action.Result{}, err
	}
	data, err := json.MarshalIndent(batch, "", "  ")
	if err != nil {
		return action.Result{}, err
	}
	var markdown strings.Builder
	fmt.Fprintf(&markdown, "# Hacker News reading digest\n\nFeed: %s\n\n", batch.Feed)
	if len(batch.Stories) == 0 {
		markdown.WriteString("No matching stories.\n")
	}
	for _, story := range batch.Stories {
		discussion := "https://news.ycombinator.com/item?id=" + strconv.Itoa(story.ID)
		link := discussion
		if story.URL != "" {
			link = strings.NewReplacer(`\`, "%5C", "(", "%28", ")", "%29", "<", "%3C", ">", "%3E", " ", "%20").Replace(story.URL)
		}
		fmt.Fprintf(&markdown, "- [%s](%s) — %d points, %d comments · [discussion](%s)\n", markdownText(story.Title), link, story.Score, story.Comments, discussion)
	}
	result, err := action.JSONResult(WriteHNDigestOutput{FileNames: []string{"digest.md", "stories.json"}})
	if err != nil {
		return action.Result{}, err
	}
	result.Artifacts = []action.Artifact{{Name: "Hacker News digest", Files: []artifacts.FileSource{
		{Name: "digest.md", MediaType: "text/markdown", Data: []byte(markdown.String())},
		{Name: "stories.json", MediaType: "application/json", Data: append(data, '\n')},
	}}}
	if err := ctx.Err(); err != nil {
		return action.Result{}, err
	}
	return result, nil
}

func markdownText(text string) string {
	text = html.EscapeString(strings.Join(strings.Fields(text), " "))
	return strings.NewReplacer(`\`, `\\`, `[`, `\[`, `]`, `\]`, `*`, `\*`, `_`, `\_`, "`", "\\`", `!`, `\!`).Replace(text)
}
