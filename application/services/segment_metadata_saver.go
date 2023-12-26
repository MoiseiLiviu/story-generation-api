package services

import (
	"context"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/domain"
)

type segmentMetadataSaver struct {
	logger       outbound.LoggerPort
	workerPool   outbound.TaskDispatcher
	segmentCache outbound.SegmentCachePort
}

func NewSegmentMetadataSaver(logger outbound.LoggerPort, workerPool outbound.TaskDispatcher,
	segmentCache outbound.SegmentCachePort) inbound.SegmentMetadataSaverPort {
	return &segmentMetadataSaver{
		logger:       logger,
		workerPool:   workerPool,
		segmentCache: segmentCache,
	}
}

func (s *segmentMetadataSaver) Save(ctx context.Context, segments <-chan domain.VideoSegment, storyID string) (<-chan domain.VideoSegment, <-chan error) {
	out := make(chan domain.VideoSegment)
	errCh := make(chan error, 5)

	newCtx, cancel := context.WithCancel(ctx)

	err := s.workerPool.Submit(func() {
		defer close(out)
		defer close(errCh)
		defer cancel()
		for segment := range segments {
			select {
			case <-newCtx.Done():
				return
			default:
				err := s.segmentCache.Save(newCtx, segment, storyID)
				if err != nil {
					select {
					case errCh <- err:
					case <-newCtx.Done():
					}
					return
				}
				select {
				case out <- segment:
				case <-newCtx.Done():
				}
			}
		}
	})

	if err != nil {
		errCh <- err
	}

	return out, errCh
}
