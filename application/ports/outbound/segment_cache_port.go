package outbound

import (
	"context"
	"generate-script-lambda/domain"
)

type SegmentCachePort interface {
	Save(ctx context.Context, segment domain.SegmentWithMediaUrl) error
}
