package services

import (
	"context"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/domain"
	"github.com/panjf2000/ants/v2"
)

type segmentMetadataSaver struct {
	workerPool   *ants.Pool
	segmentCache outbound.SegmentCachePort
}

func NewSegmentMetadataSaver(workerPool *ants.Pool, segmentCache outbound.SegmentCachePort) inbound.SegmentMetadataSaverPort {
	return &segmentMetadataSaver{
		workerPool:   workerPool,
		segmentCache: segmentCache,
	}
}

func (s segmentMetadataSaver) Save(ctx context.Context, segments <-chan domain.SegmentWithMediaUrl) (<-chan domain.SegmentEvent, <-chan error) {
	out := make(chan domain.SegmentEvent)
	errCh := make(chan error)

	newCtx, cancel := context.WithCancel(ctx)
	err := s.workerPool.Submit(func() {
		defer close(errCh)
		defer cancel()
		for {
			select {
			case <-newCtx.Done():
				return
			case segmentWithMedia := <-segments:
				err := s.segmentCache.Save(newCtx, segmentWithMedia)
				if err != nil {
					errCh <- err
					return
				} else {
					out <- segmentWithMedia.ToEvent()
				}
			}
		}
	})

	if err != nil {
		errCh <- err
		cancel()
	}

	return out, errCh
}
