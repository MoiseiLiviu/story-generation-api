package services

import (
	"context"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/domain"
	"github.com/panjf2000/ants/v2"
)

type segmentMediaSaver struct {
	mediaStore outbound.SegmentMediaStorePort
	workerPool *ants.Pool
}

func NewSegmentMediaSaver(mediaStore outbound.SegmentMediaStorePort, workerPool *ants.Pool) inbound.SegmentMediaSaverPort {
	return &segmentMediaSaver{
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

		for segment := range segmentCh {
			select {
			case <-newCtx.Done():
				return
			default:
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
