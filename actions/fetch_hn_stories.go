package actions

import (
	"context"
	"net/http"

	"github.com/defos-ai/commander-example-agent/internal/hn"
	"github.com/defos-ai/commander/framework/agent/action"
	actionmodel "github.com/defos-ai/commander/framework/domain/actions"
)

const FetchHNStoriesName = "fetch-hn-stories"
const FetchHNStoriesDescription = "Fetch up to 20 public Hacker News stories, checkpoint the snapshot for execution retries and propose filtering their titles."

// A supplied client lets embedding hosts use local fixtures without changing task inputs.
func NewFetchHNStories(client *http.Client) (*action.Func[HNFetchInput, HNStoryBatch], error) {
	return action.NewFunc(action.FuncConfig[HNFetchInput, HNStoryBatch]{
		Action: actionmodel.Action{Name: FetchHNStoriesName, Description: FetchHNStoriesDescription},
		Run: func(ctx context.Context, input HNFetchInput) (HNStoryBatch, error) {
			batch := HNStoryBatch{Feed: input.Feed, Topics: input.Topics, Stories: []HNStory{}}
			err := action.RunWithProgress(ctx, "Fetch Hacker News", func() error {
				stories, err := hn.Fetch(ctx, client, input.Feed, input.Limit)
				if err != nil {
					return err
				}
				for _, story := range stories {
					batch.Stories = append(batch.Stories, HNStory{ID: story.ID, Title: story.Title, URL: story.URL, Score: story.Score, Comments: story.Comments})
				}
				return batch.Validate()
			})
			return batch, err
		},
		NextActions: func(ctx context.Context, _ HNFetchInput, output HNStoryBatch) ([]action.NextAction, error) {
			return hnProposal(ctx, FilterHNStoriesName, output)
		},
	})
}
