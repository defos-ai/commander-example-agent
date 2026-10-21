package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/defos-ai/commander/framework/agent/action"
	actionmodel "github.com/defos-ai/commander/framework/domain/actions"
	"github.com/defos-ai/commander/framework/domain/artifacts"
)

const WriteSummaryDescription = "Write text-analysis counts to a summary.md artifact."

const WriteSummaryName = "write-summary"

type WriteSummaryInput struct {
	CharacterCount int `json:"character_count" jsonschema:"Character count from analyze-text, from 1 to 65536."`
	WordCount      int `json:"word_count" jsonschema:"Word count from analyze-text, from 1 to character_count."`
	LineCount      int `json:"line_count" jsonschema:"Line count from analyze-text, from 1 to character_count."`
}

func (input WriteSummaryInput) Validate() error {
	if input.CharacterCount < 1 || input.CharacterCount > MaxTextBytes {
		return errors.New("character_count must be between 1 and 65536")
	}
	if input.WordCount < 1 || input.WordCount > input.CharacterCount {
		return errors.New("word_count must be between 1 and character_count")
	}
	if input.LineCount < 1 || input.LineCount > input.CharacterCount {
		return errors.New("line_count must be between 1 and character_count")
	}
	return nil
}

type WriteSummaryOutput struct {
	FileName string `json:"file_name" jsonschema:"Name of the Markdown file in the returned artifact."`
}

// WriteSummary implements Executor directly because it returns artifact bytes.
type WriteSummary struct{ *action.Base }

func NewWriteSummary() (*WriteSummary, error) {
	inputSchema, err := action.SchemaFor[WriteSummaryInput]()
	if err != nil {
		return nil, err
	}
	outputSchema, err := action.SchemaFor[WriteSummaryOutput]()
	if err != nil {
		return nil, err
	}
	return &WriteSummary{action.NewBase(actionmodel.Action{
		Name:         WriteSummaryName,
		Description:  WriteSummaryDescription,
		InputSchema:  inputSchema,
		OutputSchema: outputSchema,
	})}, nil
}

func (a *WriteSummary) Run(ctx context.Context, raw json.RawMessage) (action.Result, error) {
	if err := ctx.Err(); err != nil {
		return action.Result{}, err
	}
	if err := a.Check(); err != nil {
		return action.Result{}, err
	}
	var input WriteSummaryInput
	if err := action.DecodeInput(raw, &input); err != nil {
		return action.Result{}, err
	}
	if err := input.Validate(); err != nil {
		return action.Result{}, err
	}
	content := fmt.Sprintf("# Text summary\n\n- Characters: %d\n- Words: %d\n- Lines: %d\n",
		input.CharacterCount, input.WordCount, input.LineCount)
	result, err := action.JSONResult(WriteSummaryOutput{FileName: "summary.md"})
	if err != nil {
		return action.Result{}, err
	}
	// Return bytes, not storage paths. The runtime uploads files before reporting
	// the result; the controller persists the accepted artifact metadata.
	result.Artifacts = []action.Artifact{{
		Name: "Text summary",
		Files: []artifacts.FileSource{{
			Name: "summary.md", MediaType: "text/markdown", Data: []byte(content),
		}},
	}}
	if err := ctx.Err(); err != nil {
		return action.Result{}, err
	}
	return result, nil
}
