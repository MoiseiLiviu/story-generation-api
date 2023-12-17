package inbound

import (
	"context"
	"generate-script-lambda/domain"
)

type SegmentMetadataSaverPort interface {
	Save(ctx context.Context, segments <-chan domain.SegmentWithMediaUrl) (<-chan domain.SegmentEvent, <-chan error)
}
