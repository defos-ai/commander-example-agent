package actions

import (
	"context"
	"strings"

	"github.com/defos-ai/commander/framework/agent/action"
	actionmodel "github.com/defos-ai/commander/framework/domain/actions"
)

const ChooseHNTopicsName = "choose-hn-topics"
const ChooseHNTopicsDescription = "Ask which topics a Hacker News reading digest should cover, then propose fetching its feed."

func NewChooseHNTopics() (*action.Func[HNFeedInput, HNFetchInput], error) {
	return action.NewFunc(action.FuncConfig[HNFeedInput, HNFetchInput]{
		Action:   actionmodel.Action{Name: ChooseHNTopicsName, Description: ChooseHNTopicsDescription},
		NewInput: func() HNFeedInput { return HNFeedInput{Feed: "top", Limit: 10} },
		Run: func(ctx context.Context, input HNFeedInput) (HNFetchInput, error) {
			question, err := action.Question(ctx, "hn-topics", "Which topics should the reading list cover? Use all or comma-separated keywords.", "all", "Go", "AI", "security")
			if err != nil {
				return HNFetchInput{}, err
			}
			topics := []string{}
			if !strings.EqualFold(strings.TrimSpace(question.Answer), "all") {
				for _, topic := range strings.Split(question.Answer, ",") {
					topics = append(topics, strings.TrimSpace(topic))
				}
			}
			output := HNFetchInput{Feed: input.Feed, Limit: input.Limit, Topics: topics}
			return output, output.Validate()
		},
		NextActions: func(ctx context.Context, _ HNFeedInput, output HNFetchInput) ([]action.NextAction, error) {
			return hnProposal(ctx, FetchHNStoriesName, output)
		},
	})
}
