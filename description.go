package exampleagent

import (
	"github.com/defos-ai/commander-example-agent/actions"
	actionmodel "github.com/defos-ai/commander/framework/domain/actions"
)

// Responsibilities returns the default capabilities without starting an agent.
func Responsibilities() []string {
	return []string{"Analyze text and produce a Markdown summary"}
}

// ActionDescriptions returns release metadata without schemas or discovery state.
func ActionDescriptions() []actionmodel.Action {
	return []actionmodel.Action{
		{Name: actions.AnalyzeTextName, Description: actions.AnalyzeTextDescription},
		{Name: actions.WriteSummaryName, Description: actions.WriteSummaryDescription},
	}
}
