package inbound

import (
	"context"
	"generate-script-lambda/domain"
)

type SegmentMetadataSaverPort interface {
	Save(ctx context.Context, segments <-chan domain.VideoSegment, storyID string) (<-chan domain.VideoSegment, <-chan error)
}
