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
		analyze, err := actions.NewAnalyzeText()
		if err != nil {
			return nil, fmt.Errorf("create analyze-text: %w", err)
		}
		summary, err := actions.NewWriteSummary()
		if err != nil {
			return nil, fmt.Errorf("create write-summary: %w", err)
		}
		config.Actions = []action.Executor{analyze, summary}
	}
	return agent.NewAgent(config)
}
