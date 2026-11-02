package actions

import (
	"context"
	"strings"

	"github.com/defos-ai/commander/framework/agent/action"
	actionmodel "github.com/defos-ai/commander/framework/domain/actions"
)

const FilterHNStoriesName = "filter-hn-stories"
const FilterHNStoriesDescription = "Keep stories whose titles contain a selected topic, or every story when topics is empty."

func NewFilterHNStories() (*action.Func[HNStoryBatch, HNStoryBatch], error) {
	return action.NewFunc(action.FuncConfig[HNStoryBatch, HNStoryBatch]{
		Action: actionmodel.Action{Name: FilterHNStoriesName, Description: FilterHNStoriesDescription},
		Run: func(ctx context.Context, input HNStoryBatch) (HNStoryBatch, error) {
			output := HNStoryBatch{Feed: input.Feed, Topics: input.Topics, Stories: []HNStory{}}
			for _, story := range input.Stories {
				if err := ctx.Err(); err != nil {
					return HNStoryBatch{}, err
				}
				matches := len(input.Topics) == 0
				for _, topic := range input.Topics {
					matches = matches || strings.Contains(" "+normalizedWords(story.Title)+" ", " "+normalizedWords(topic)+" ")
				}
				if matches {
					output.Stories = append(output.Stories, story)
				}
			}
			return output, nil
		},
		NextActions: func(ctx context.Context, _ HNStoryBatch, output HNStoryBatch) ([]action.NextAction, error) {
			return hnProposal(ctx, RankHNStoriesName, output)
		},
	})
}
