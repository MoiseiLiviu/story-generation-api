package services

import (
	"context"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/domain"
	"github.com/panjf2000/ants/v2"
	"sync"
)

type segmentMediaEnhancer struct {
	imageGenerator outbound.ImageGeneratorPort
	audioGenerator outbound.AudioGeneratorPort
	workerPool     *ants.Pool
}

func NewSegmentMediaEnhancer(imageGenerator outbound.ImageGeneratorPort, audioGenerator outbound.AudioGeneratorPort,
	workerPool *ants.Pool) inbound.SegmentMediaEnhancerPort {
	return &segmentMediaEnhancer{
		imageGenerator: imageGenerator,
		audioGenerator: audioGenerator,
		workerPool:     workerPool,
	}
}

func (s *segmentMediaEnhancer) Generate(ctx context.Context, segmentCh <-chan domain.Segment, voiceID string) (<-chan domain.SegmentWithMedia, <-chan error) {
	out := make(chan domain.SegmentWithMedia)
	errCh := make(chan error)

	newCtx, cancel := context.WithCancel(ctx)

	err := s.workerPool.Submit(func() {
		defer close(out)
		defer close(errCh)
		defer cancel()

		var wg sync.WaitGroup

		for segment := range segmentCh {
			select {
			case <-newCtx.Done():
				return
			default:
				wg.Add(1)
				err := s.workerPool.Submit(func() {
					defer wg.Done()
					select {
					case <-newCtx.Done():
						return
					default:
						if segment.Type == domain.ImageSegmentType {
							result, err := s.useImageGenerator(newCtx, segment)
							if err != nil {
								errCh <- err
								cancel()
							}
							out <- result
						} else if segment.Type == domain.AudioSegmentType {
							result, err := s.useAudioGenerator(newCtx, segment, voiceID)
							if err != nil {
								errCh <- err
								cancel()
							}
							out <- result
						}
					}
				})
				if err != nil {
					wg.Done()
					errCh <- err
					cancel()
				}
			}
		}

		wg.Wait()
	})
	if err != nil {
		errCh <- err
	}

	return out, errCh
}

func (s *segmentMediaEnhancer) useImageGenerator(newCtx context.Context, segment domain.Segment) (domain.SegmentWithMedia, error) {
	content, err := s.imageGenerator.Generate(newCtx, segment.Text)
	if err != nil {
		return domain.SegmentWithMedia{}, err
	}
	return domain.SegmentWithMedia{
		MediaContent: content,
		Segment:      segment,
	}, nil
}

func (s *segmentMediaEnhancer) useAudioGenerator(newCtx context.Context, segment domain.Segment, voiceID string) (domain.SegmentWithMedia, error) {
	content, err := s.audioGenerator.Generate(newCtx, outbound.GenerateAudioParams{Text: segment.Text, VoiceID: voiceID})
	if err != nil {
		return domain.SegmentWithMedia{}, err
	}
	return domain.SegmentWithMedia{
		MediaContent: content,
		Segment:      segment,
	}, nil
}
