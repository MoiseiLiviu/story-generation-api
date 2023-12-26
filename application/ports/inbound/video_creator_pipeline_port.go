package inbound

import (
	"context"
)

type StartPipelineParams struct {
	StoryID string
	Input   string
	VoiceID string
	UserID  string
}

type VideoCreatorResponse struct {
	VideoKey    string
	VideoRegion string
}

type VideoCreatorPipelinePort interface {
	StartPipeline(ctx context.Context, request StartPipelineParams) (*VideoCreatorResponse, error)
}
