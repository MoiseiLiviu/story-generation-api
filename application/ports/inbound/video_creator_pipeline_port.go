package inbound

import (
	"context"
	"generate-script-lambda/domain"
)

type StartPipelineParams struct {
	StoryID       string
	Input         string
	VoiceID       string
	UserID        string
	WordsPerStory int
}

type VideoCreatorResponse struct {
	VideoKey      string
	VideoRegion   string
	VideoSegments []domain.VideoSegment
}

type VideoCreatorPipelinePort interface {
	StartPipeline(ctx context.Context, request StartPipelineParams) (*VideoCreatorResponse, error)
}
