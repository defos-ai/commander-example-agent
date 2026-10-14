package exampleagent_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/defos-ai/commander-example-agent/actions"
	"github.com/defos-ai/commander/framework/agent/action"
)

func TestAnalysisProposesDownloadableSummary(t *testing.T) {
	analyze, err := actions.NewAnalyzeText()
	if err != nil {
		t.Fatal(err)
	}
	discover(t, analyze)
	result, err := analyze.Run(t.Context(), json.RawMessage(`{"text":"Hello 世界\nNext line\n"}`))
	if err != nil {
		t.Fatal(err)
	}
	var output actions.AnalyzeTextOutput
	if err := json.Unmarshal(result.Output, &output); err != nil {
		t.Fatal(err)
	}
	if output.CharacterCount != 19 || output.WordCount != 4 || output.LineCount != 3 {
		t.Fatalf("unexpected text analysis: %+v", output)
	}
	if len(result.NextActions) != 1 || result.NextActions[0].ActionName != actions.WriteSummaryName {
		t.Fatalf("expected one write-summary proposal: %+v", result.NextActions)
	}
	summary, err := actions.NewWriteSummary()
	if err != nil {
		t.Fatal(err)
	}
	discover(t, summary)
	// Feed the proposal into the next action just as the controller would.
	report, err := summary.Run(t.Context(), result.NextActions[0].Input)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.NextActions) != 0 || len(report.Artifacts) != 1 {
		t.Fatalf("expected a final summary artifact: %+v", report)
	}
	artifact := report.Artifacts[0]
	if artifact.Name != "Text summary" || len(artifact.Files) != 1 {
		t.Fatalf("unexpected artifact: %+v", artifact)
	}
	file := artifact.Files[0]
	want := "# Text summary\n\n- Characters: 19\n- Words: 4\n- Lines: 3\n"
	if file.Name != "summary.md" || file.MediaType != "text/markdown" || file.Path != "" || string(file.Data) != want {
		t.Fatalf("unexpected summary file: %+v", file)
	}
	var reportOutput actions.WriteSummaryOutput
	if err := json.Unmarshal(report.Output, &reportOutput); err != nil {
		t.Fatal(err)
	}
	if reportOutput.FileName != file.Name {
		t.Fatalf("output filename %q differs from artifact file %q", reportOutput.FileName, file.Name)
	}
}

func TestAnalyzeTextRejectsBlankAndOversizedInput(t *testing.T) {
	executor, err := actions.NewAnalyzeText()
	if err != nil {
		t.Fatal(err)
	}
	discover(t, executor)
	for _, test := range []struct{ name, text, wantError string }{
		{"blank", " \t\n", "text must not be blank"},
		{"oversized", strings.Repeat("a", actions.MaxTextBytes+1), "text must not exceed"},
		{"multibyte limit", strings.Repeat("界", actions.MaxTextBytes/3+1), "text must not exceed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			raw, err := json.Marshal(actions.AnalyzeTextInput{Text: test.text})
			if err != nil {
				t.Fatal(err)
			}
			_, err = executor.Run(t.Context(), raw)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("expected %q, got %v", test.wantError, err)
			}
		})
	}
}

func TestWriteSummaryRejectsInvalidCounts(t *testing.T) {
	executor, err := actions.NewWriteSummary()
	if err != nil {
		t.Fatal(err)
	}
	discover(t, executor)
	for _, raw := range []string{
		`{"character_count":0,"word_count":1,"line_count":1}`,
		`{"character_count":65537,"word_count":1,"line_count":1}`,
		`{"character_count":10,"word_count":-1,"line_count":1}`,
		`{"character_count":10,"word_count":11,"line_count":1}`,
		`{"character_count":10,"word_count":1,"line_count":11}`,
		`{"character_count":10,"word_count":1}`,
	} {
		if _, err := executor.Run(t.Context(), json.RawMessage(raw)); err == nil {
			t.Errorf("accepted invalid counts: %s", raw)
		}
	}
}

func TestActionsHonorCancellation(t *testing.T) {
	analyze, err := actions.NewAnalyzeText()
	if err != nil {
		t.Fatal(err)
	}
	summary, err := actions.NewWriteSummary()
	if err != nil {
		t.Fatal(err)
	}
	for _, executor := range []action.Executor{analyze, summary} {
		t.Run(executor.Action().Name, func(t *testing.T) {
			discover(t, executor)
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			if _, err := executor.Run(ctx, json.RawMessage(`{}`)); !errors.Is(err, context.Canceled) {
				t.Fatalf("expected cancellation, got %v", err)
			}
		})
	}
}

func discover(t *testing.T, executor action.Executor) {
	t.Helper()
	if _, err := executor.Discover(t.Context(), nil, nil); err != nil {
		t.Fatal(err)
	}
}
