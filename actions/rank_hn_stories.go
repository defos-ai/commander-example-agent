package actions

import (
	"cmp"
	"context"
	"slices"

	"github.com/defos-ai/commander/framework/agent/action"
	actionmodel "github.com/defos-ai/commander/framework/domain/actions"
)

const RankHNStoriesName = "rank-hn-stories"
const RankHNStoriesDescription = "Select up to five stories by score, then comment count, and propose writing a reading digest."

func NewRankHNStories() (*action.Func[HNStoryBatch, HNStoryBatch], error) {
	return action.NewFunc(action.FuncConfig[HNStoryBatch, HNStoryBatch]{
		Action: actionmodel.Action{Name: RankHNStoriesName, Description: RankHNStoriesDescription},
		Run: func(ctx context.Context, input HNStoryBatch) (HNStoryBatch, error) {
			if err := ctx.Err(); err != nil {
				return HNStoryBatch{}, err
			}
			input.Stories = slices.Clone(input.Stories)
			slices.SortFunc(input.Stories, func(a, b HNStory) int {
				if a.Score != b.Score {
					return cmp.Compare(b.Score, a.Score)
				}
				if a.Comments != b.Comments {
					return cmp.Compare(b.Comments, a.Comments)
				}
				return cmp.Compare(a.ID, b.ID)
			})
			if len(input.Stories) > 5 {
				input.Stories = input.Stories[:5]
			}
			return input, nil
		},
		NextActions: func(ctx context.Context, _ HNStoryBatch, output HNStoryBatch) ([]action.NextAction, error) {
			return hnProposal(ctx, WriteHNDigestName, output)
		},
	})
}
