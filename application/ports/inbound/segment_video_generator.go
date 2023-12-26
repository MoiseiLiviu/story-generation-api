package inbound

import (
	"context"
	"generate-script-lambda/domain"
)

type SegmentVideoGenerator interface {
	Generate(ctx context.Context, segments <-chan domain.AudioWithImageBackground) (<-chan domain.VideoSegment, <-chan error)
}
