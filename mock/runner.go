package mock_generator

import (
	"context"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/channel_utils"
	"generate-script-lambda/domain"
	"github.com/google/uuid"
	"time"
)

type Runner struct {
	logger        outbound.LoggerPort
	workerPool    outbound.TaskDispatcher
	segmentReader SegmentReader
	metadataSaver inbound.SegmentMetadataSaverPort
}

func NewRunner(workerPool outbound.TaskDispatcher, segmentReader SegmentReader, metadataSaver inbound.SegmentMetadataSaverPort, logger outbound.LoggerPort) *Runner {
	return &Runner{
		logger:        logger,
		workerPool:    workerPool,
		segmentReader: segmentReader,
		metadataSaver: metadataSaver,
	}
}

func (r *Runner) Run(ctx context.Context) (<-chan domain.SegmentEvent, <-chan error) {
	segmentCh, segmentErrCh := r.createSegmentStream(ctx, uuid.NewString())

	segmentEvents, metadataSaverErrCh := r.metadataSaver.Save(ctx, segmentCh)

	mergedErrCh, err := channel_utils.MergeChannels(r.workerPool, segmentErrCh, metadataSaverErrCh)
	if err != nil {
		out := make(chan domain.SegmentEvent)
		errCh := make(chan error)
		errCh <- err
		close(out)
		close(errCh)
		return out, errCh
	}

	return segmentEvents, mergedErrCh
}

func (r *Runner) createSegmentStream(ctx context.Context, storyID string) (<-chan domain.SegmentWithMediaUrl, <-chan error) {
	out := make(chan domain.SegmentWithMediaUrl)
	errCh := make(chan error, 5)

	newCtx, cancel := context.WithCancel(ctx)

	err := r.workerPool.Submit(func() {
		defer close(out)
		defer close(errCh)
		defer cancel()
		audioSegmentsCh, err := r.streamSegmentsFromFileWithDelay(newCtx, "mock/audio.json", storyID)
		if err != nil {
			r.logger.Error(err, "failed to stream segments from file")
			errCh <- err
			cancel()
			return
		}
		imageSegmentsCh, err := r.streamSegmentsFromFileWithDelay(newCtx, "mock/image.json", storyID)
		if err != nil {
			r.logger.Error(err, "failed to stream segments from file")
			errCh <- err
			cancel()
			return
		}

		mergedCh, err := channel_utils.MergeChannels(r.workerPool, audioSegmentsCh, imageSegmentsCh)
		if err != nil {
			r.logger.Error(err, "failed to merge channels")
			errCh <- err
			cancel()
			return
		}
		for segment := range mergedCh {
			select {
			case <-newCtx.Done():
				return
			default:
				out <- segment
			}
		}
		r.logger.Info("Finished reading from stream.")
	})
	if err != nil {
		errCh <- err
	}

	return out, errCh
}

func (r *Runner) streamSegmentsFromFileWithDelay(ctx context.Context, fileName string, storyID string) (<-chan domain.SegmentWithMediaUrl, error) {
	out := make(chan domain.SegmentWithMediaUrl)
	mockSegments, err := r.segmentReader.Read(fileName)
	if err != nil {
		return nil, err
	}
	err = r.workerPool.Submit(func() {
		defer close(out)
		for _, s := range mockSegments {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(time.Duration(s.Delay) * time.Second)
				out <- domain.SegmentWithMediaUrl{
					MediaURL: s.Url,
					Segment: domain.Segment{
						StoryID: storyID,
						Text:    s.Text,
						Type:    s.Type,
						ID:      s.SegmentId,
						Ordinal: s.Ordinal,
					},
				}
			}
		}
	})

	return out, err
}
