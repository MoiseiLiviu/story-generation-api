package inbound

import (
	"context"
	"generate-script-lambda/domain"
)

type SegmentMediaFileGenerator interface {
	Generate(context context.Context, segmentCh <-chan domain.Segment, voiceID string) (<-chan domain.SegmentWithMediaFile, <-chan error)
}
