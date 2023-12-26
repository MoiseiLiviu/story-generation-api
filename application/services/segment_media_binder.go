package services

import (
	"context"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/domain"
)

const DefaultBackgroundImage = "/assets/images/default.jpg"

type segmentMediaBinder struct {
	workerPool outbound.TaskDispatcher
}

func NewSegmentMediaBinder(workerPool outbound.TaskDispatcher) inbound.SegmentMediaBinderPort {
	return &segmentMediaBinder{
		workerPool: workerPool,
	}
}

func (s *segmentMediaBinder) Bind(ctx context.Context, segments <-chan domain.SegmentWithMediaFile) (<-chan domain.AudioWithImageBackground, <-chan error) {
	out := make(chan domain.AudioWithImageBackground)
	errCh := make(chan error, 5)
	err := s.workerPool.Submit(func() {
		defer close(out)
		defer close(errCh)
		imageSegments := make([]domain.SegmentWithMediaFile, 0)
		audioSegments := make([]domain.SegmentWithMediaFile, 0)
		for segment := range segments {
			select {
			case <-ctx.Done():
				return
			default:
				if segment.Type == domain.AudioSegmentType {
					if segment.BackgroundImageID == domain.DefaultImageID {
						out <- domain.AudioWithImageBackground{
							SegmentWithMediaFile:    segment,
							BackgroundImageFileName: DefaultBackgroundImage,
						}
					} else {
						image := s.getSegmentById(segment.BackgroundImageID, imageSegments)
						if image == nil {
							audioSegments = append(audioSegments, segment)
						}
					}
				} else {
					pendingAudio := s.getAudioSegmentByBackgroundImageID(segment.ID, audioSegments)
					for _, audio := range pendingAudio {
						out <- domain.AudioWithImageBackground{
							SegmentWithMediaFile:    *audio,
							BackgroundImageFileName: segment.FileName,
						}
					}
					imageSegments = append(imageSegments, segment)
				}
			}

		}
	})
	if err != nil {
		errCh <- err
	}

	return out, errCh
}

func (s *segmentMediaBinder) getAudioSegmentByBackgroundImageID(id string, segments []domain.SegmentWithMediaFile) []*domain.SegmentWithMediaFile {
	audioSegments := make([]*domain.SegmentWithMediaFile, 0)
	for _, segment := range segments {
		if segment.BackgroundImageID == id {
			audioSegments = append(audioSegments, &segment)
		}
	}
	return audioSegments
}

func (s *segmentMediaBinder) getSegmentById(id string, segments []domain.SegmentWithMediaFile) *domain.SegmentWithMediaFile {
	for _, segment := range segments {
		if segment.ID == id {
			return &segment
		}
	}
	return nil
}
