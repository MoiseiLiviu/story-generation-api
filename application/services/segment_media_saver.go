package services

import (
	"context"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/domain"
)

type segmentMediaSaver struct {
	logger     outbound.LoggerPort
	mediaStore outbound.SegmentMediaStorePort
	workerPool outbound.TaskDispatcher
}

func NewSegmentMediaSaver(logger outbound.LoggerPort, mediaStore outbound.SegmentMediaStorePort, workerPool outbound.TaskDispatcher) inbound.SegmentMediaSaverPort {
	return &segmentMediaSaver{
		logger:     logger,
		mediaStore: mediaStore,
		workerPool: workerPool,
	}
}

func (s *segmentMediaSaver) Save(ctx context.Context, segmentCh <-chan domain.SegmentWithMedia, userID string) (<-chan domain.SegmentWithMediaUrl, <-chan error) {
	out := make(chan domain.SegmentWithMediaUrl)
	errCh := make(chan error)
	newCtx, cancel := context.WithCancel(ctx)

	err := s.workerPool.Submit(func() {
		defer close(out)
		defer close(errCh)
		defer cancel()

		for {
			select {
			case <-newCtx.Done():
				return
			case segment, ok := <-segmentCh:
				if !ok {
					return
				}
				url, err := s.mediaStore.Save(newCtx, segment, userID)
				if err != nil {
					errCh <- err
					cancel()
					return
				}

				out <- domain.SegmentWithMediaUrl{
					Segment:  segment.Segment,
					MediaURL: url,
				}
			}
		}
	})

	if err != nil {
		errCh <- err
	}

	return out, errCh
}
