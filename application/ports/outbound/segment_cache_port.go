package outbound

import (
	"context"
	"generate-script-lambda/domain"
)

type SegmentCachePort interface {
	Save(ctx context.Context, segment domain.VideoSegment, storyID string) error
}
