package exampleagent

import (
	"fmt"

	"github.com/defos-ai/commander-example-agent/actions"
	"github.com/defos-ai/commander/framework/agent"
	"github.com/defos-ai/commander/framework/agent/action"
)

const AgentType agent.AgentType = "example-agent"

// NewAgent registers a fresh action catalog for each agent instance.
func NewAgent(config agent.AgentConfig) (*agent.Agent, error) {
	config.Type = AgentType
	if config.Responsibilities == nil {
		config.Responsibilities = Responsibilities()
	}
	if config.Actions == nil {
		choose, err := actions.NewChooseHNTopics()
		if err != nil {
			return nil, fmt.Errorf("create choose-hn-topics: %w", err)
		}
		fetch, err := actions.NewFetchHNStories(nil)
		if err != nil {
			return nil, fmt.Errorf("create fetch-hn-stories: %w", err)
		}
		filter, err := actions.NewFilterHNStories()
		if err != nil {
			return nil, fmt.Errorf("create filter-hn-stories: %w", err)
		}
		rank, err := actions.NewRankHNStories()
		if err != nil {
			return nil, fmt.Errorf("create rank-hn-stories: %w", err)
		}
		digest, err := actions.NewWriteHNDigest()
		if err != nil {
			return nil, fmt.Errorf("create write-hn-digest: %w", err)
		}
		config.Actions = []action.Executor{choose, fetch, filter, rank, digest}
	}
	return agent.NewAgent(config)
}
