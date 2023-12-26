package inbound

import (
	"context"
	"generate-script-lambda/domain"
)

type GenerateSegmentsParams struct {
	Input         string
	StoryID       string
	WordsPerStory int
}

type SegmentsGeneratorPort interface {
	Generate(ctx context.Context, params GenerateSegmentsParams) (<-chan domain.Segment, <-chan error)
}
