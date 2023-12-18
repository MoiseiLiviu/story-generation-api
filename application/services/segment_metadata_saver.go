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

func (s *segmentMetadataSaver) Save(ctx context.Context, segments <-chan domain.SegmentWithMediaUrl) (<-chan domain.SegmentEvent, <-chan error) {
	out := make(chan domain.SegmentEvent)
	errCh := make(chan error)

	newCtx, cancel := context.WithCancel(ctx)

	err := s.workerPool.Submit(func() {
		defer close(out)
		defer close(errCh)
		defer cancel()
		for segmentWithMedia := range segments {
			select {
			case <-newCtx.Done():
				return
			default:
				err := s.segmentCache.Save(newCtx, segmentWithMedia)
				if err != nil {
					errCh <- err
					return
				} else {
					s.logger.DebugWithFields("segment saved", map[string]interface{}{
						"type": segmentWithMedia.Type,
						"id":   segmentWithMedia.ID,
					})
					out <- segmentWithMedia.ToEvent()
				}
			}
		}
		s.logger.Debug("segment metadata saving complete")
	})

	if err != nil {
		errCh <- err
		cancel()
	}

	return out, errCh
}
