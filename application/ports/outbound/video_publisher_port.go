package outbound

import "context"

type PublishVideoRequest struct {
	VideoFileName string
	StoryID       string
	UserID        string
}

type PublishVideoResponse struct {
	VideoKey    string
	StoreRegion string
}

type VideoPublisherPort interface {
	Publish(ctx context.Context, req PublishVideoRequest) (*PublishVideoResponse, error)
}
