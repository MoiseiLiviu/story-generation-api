package services

import (
	"context"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/domain"
	"sync"
)

type segmentVideoGenerator struct {
	workerPool   outbound.TaskDispatcher
	videoCreator outbound.SegmentVideoCreator
}

func NewSegmentVideoGenerator(workerPool outbound.TaskDispatcher, videoCreator outbound.SegmentVideoCreator) inbound.SegmentVideoGenerator {
	return &segmentVideoGenerator{
		workerPool:   workerPool,
		videoCreator: videoCreator,
	}
}

func (g *segmentVideoGenerator) Generate(ctx context.Context, segments <-chan domain.AudioWithImageBackground) (<-chan domain.VideoSegment, <-chan error) {
	out := make(chan domain.VideoSegment)
	errCh := make(chan error, 5)
	newCtx, cancel := context.WithCancel(ctx)
	err := g.workerPool.Submit(func() {
		defer close(out)
		defer close(errCh)
		defer cancel()
		var wg sync.WaitGroup
		for s := range segments {
			select {
			case <-newCtx.Done():
				return
			default:
				segment := s
				wg.Add(1)
				err := g.workerPool.Submit(func() {
					defer wg.Done()
					videoCreationRes, err := g.videoCreator.Create(segment.FileName, segment.BackgroundImageFileName)
					if err != nil {
						select {
						case errCh <- err:
						case <-newCtx.Done():
						}
						return
					}
					select {
					case out <- domain.VideoSegment{
						Ordinal:  segment.Ordinal,
						FileName: videoCreationRes.FileName,
						Duration: videoCreationRes.Duration,
					}:
					case <-newCtx.Done():
					}
				})
				if err != nil {
					select {
					case errCh <- err:
					case <-newCtx.Done():
					}
					return
				}
			}
		}
	})
	if err != nil {
		errCh <- err
	}

	return out, errCh
}
