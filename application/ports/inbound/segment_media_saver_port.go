package inbound

import (
	"context"
	"generate-script-lambda/domain"
)

type SegmentMediaSaverPort interface {
	Save(ctx context.Context, segmentCh <-chan domain.SegmentWithMedia, userID string) (<-chan domain.SegmentWithMediaUrl, <-chan error)
}
