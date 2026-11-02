package exampleagent

import (
	"github.com/defos-ai/commander-example-agent/actions"
	actionmodel "github.com/defos-ai/commander/framework/domain/actions"
)

// Responsibilities returns the default capabilities without starting an agent.
func Responsibilities() []string {
	return []string{"Build a Hacker News reading list from selected topics", "Filter and rank stories, then produce Markdown and JSON digests"}
}

// ActionDescriptions returns release metadata without schemas or discovery state.
func ActionDescriptions() []actionmodel.Action {
	return []actionmodel.Action{
		{Name: actions.ChooseHNTopicsName, Description: actions.ChooseHNTopicsDescription},
		{Name: actions.FetchHNStoriesName, Description: actions.FetchHNStoriesDescription},
		{Name: actions.FilterHNStoriesName, Description: actions.FilterHNStoriesDescription},
		{Name: actions.RankHNStoriesName, Description: actions.RankHNStoriesDescription},
		{Name: actions.WriteHNDigestName, Description: actions.WriteHNDigestDescription},
	}
}
