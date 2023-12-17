package outbound

import (
	"context"
	"generate-script-lambda/domain"
)

type SegmentMediaStorePort interface {
	Save(ctx context.Context, segment domain.SegmentWithMedia, userID string) (string, error)
}
