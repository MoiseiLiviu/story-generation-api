package inbound

import (
	"context"
	"generate-script-lambda/domain"
)

type SegmentMediaEnhancerPort interface {
	Enhance(context context.Context, segmentCh <-chan domain.Segment, voiceID string) (<-chan domain.SegmentWithMedia, <-chan error)
}
