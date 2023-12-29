package inbound

import (
	"context"
	"generate-script-lambda/domain"
)

type SegmentMediaBinderPort interface {
	Bind(ctx context.Context, segments <-chan domain.SegmentWithMediaFile) ([]domain.AudioWithImageBackground, error)
}
