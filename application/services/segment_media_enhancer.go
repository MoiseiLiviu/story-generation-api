package services

import (
	"context"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/domain"
	"regexp"
	"strings"
	"sync"
)

type segmentMediaEnhancer struct {
	logger         outbound.LoggerPort
	imageGenerator outbound.ImageGeneratorPort
	audioGenerator outbound.AudioGeneratorPort
	workerPool     outbound.TaskDispatcher
	imageRegexp    *regexp.Regexp
}

func NewSegmentMediaEnhancer(logger outbound.LoggerPort, imageGenerator outbound.ImageGeneratorPort, audioGenerator outbound.AudioGeneratorPort,
	workerPool outbound.TaskDispatcher) inbound.SegmentMediaEnhancerPort {
	return &segmentMediaEnhancer{
		logger:         logger,
		imageGenerator: imageGenerator,
		audioGenerator: audioGenerator,
		workerPool:     workerPool,
		imageRegexp:    regexp.MustCompile(`\{[^}]*}`),
	}
}

func (s *segmentMediaEnhancer) Enhance(ctx context.Context, segmentCh <-chan domain.Segment, voiceID string) (<-chan domain.SegmentWithMedia, <-chan error) {
	out := make(chan domain.SegmentWithMedia)
	errCh := make(chan error)

	newCtx, cancel := context.WithCancel(ctx)

	err := s.workerPool.Submit(func() {
		defer close(out)
		defer close(errCh)
		defer cancel()

		var wg sync.WaitGroup

	outer:
		for {
			select {
			case <-newCtx.Done():
				return
			case segment, ok := <-segmentCh:
				if !ok {
					break outer
				}
				wg.Add(1)
				err := s.workerPool.Submit(func() {
					defer wg.Done()
					select {
					case <-newCtx.Done():
						return
					default:
						s.logger.DebugWithFields("Enhancing segment", map[string]interface{}{
							"segment_id": segment.ID,
							"type":       segment.Type,
						})
						if segment.Type == domain.ImageSegmentType {
							result, err := s.useImageGenerator(newCtx, segment)
							if err != nil {
								s.logger.ErrorWithFields(err, "Failed to generate image", map[string]interface{}{
									"description": segment.Text,
									"segment_id":  segment.ID,
								})
								errCh <- err
								cancel()
							}
							out <- result
						} else if segment.Type == domain.AudioSegmentType {
							result, err := s.useAudioGenerator(newCtx, segment, voiceID)
							if err != nil {
								s.logger.ErrorWithFields(err, "Failed to generate audio", map[string]interface{}{
									"text":       segment.Text,
									"segment_id": segment.ID,
								})
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
	preparedText := s.prepareTextForTTS(segment.Text)
	content, err := s.audioGenerator.Generate(newCtx, outbound.GenerateAudioParams{Text: preparedText, VoiceID: voiceID})
	if err != nil {
		return domain.SegmentWithMedia{}, err
	}
	return domain.SegmentWithMedia{
		MediaContent: content,
		Segment:      segment,
	}, nil
}

func (s *segmentMediaEnhancer) prepareTextForTTS(input string) string {
	result := s.imageRegexp.ReplaceAllString(input, "")
	result = s.removeEmptySpaces(result)

	return result
}

func (s *segmentMediaEnhancer) removeEmptySpaces(input string) string {
	result := strings.Replace(input, "\n", "", -1)
	result = strings.Replace(result, "\r", "", -1)
	result = strings.Replace(result, "\t", "", -1)
	result = strings.TrimSpace(result)

	return result
}
