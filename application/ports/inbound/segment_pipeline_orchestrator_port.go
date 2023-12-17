package inbound

import (
	"context"
	"generate-script-lambda/domain"
)

type StartPipelineParams struct {
	StoryID string
	Input   string
	VoiceID string
}

type SegmentPipelineOrchestrator interface {
	StartPipeline(ctx context.Context, request StartPipelineParams) (<-chan domain.SegmentEvent, <-chan error)
}
