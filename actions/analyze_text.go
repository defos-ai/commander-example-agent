package actions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/defos-ai/commander/framework/agent/action"
	actionmodel "github.com/defos-ai/commander/framework/domain/actions"
)

const (
	AnalyzeTextName = "analyze-text"
	MaxTextBytes    = 64 * 1024
)

type AnalyzeTextInput struct {
	Text string `json:"text" jsonschema:"Nonblank UTF-8 text, at most 65536 bytes."`
}

func (input AnalyzeTextInput) Validate() error {
	if strings.TrimSpace(input.Text) == "" {
		return errors.New("text must not be blank")
	}
	if len(input.Text) > MaxTextBytes {
		return errors.New("text must not exceed 65536 bytes")
	}
	if !utf8.ValidString(input.Text) {
		return errors.New("text must be valid UTF-8")
	}
	return nil
}

type AnalyzeTextOutput struct {
	CharacterCount int `json:"character_count" jsonschema:"Number of Unicode code points, including whitespace."`
	WordCount      int `json:"word_count" jsonschema:"Number of words separated by Unicode whitespace."`
	LineCount      int `json:"line_count" jsonschema:"Number of newline characters plus one, including an empty final line."`
}

func NewAnalyzeText() (*action.Func[AnalyzeTextInput, AnalyzeTextOutput], error) {
	return action.NewFunc(action.FuncConfig[AnalyzeTextInput, AnalyzeTextOutput]{
		Action: actionmodel.Action{
			Name:        AnalyzeTextName,
			Description: "Count the characters, words and lines in text, then propose writing a Markdown summary.",
		},
		Run: func(ctx context.Context, input AnalyzeTextInput) (AnalyzeTextOutput, error) {
			if err := ctx.Err(); err != nil {
				return AnalyzeTextOutput{}, err
			}
			return AnalyzeTextOutput{
				CharacterCount: utf8.RuneCountInString(input.Text),
				WordCount:      len(strings.Fields(input.Text)),
				LineCount:      strings.Count(input.Text, "\n") + 1,
			}, nil
		},
		NextActions: func(ctx context.Context, _ AnalyzeTextInput, output AnalyzeTextOutput) ([]action.NextAction, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			input, err := json.Marshal(WriteSummaryInput{
				CharacterCount: output.CharacterCount,
				WordCount:      output.WordCount,
				LineCount:      output.LineCount,
			})
			if err != nil {
				return nil, err
			}
			// The controller decides which agent executes this proposal.
			return []action.NextAction{{
				ActionName: WriteSummaryName,
				Input:      input,
				Reason:     "Save the text analysis as a downloadable Markdown file.",
			}}, nil
		},
	})
}
