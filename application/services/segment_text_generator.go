package services

import (
	"context"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/domain"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"regexp"
	"strings"
)

type segmentTextGenerator struct {
	logger            outbound.LoggerPort
	scriptGenerator   outbound.StoryScriptGeneratorPort
	workerPool        outbound.TaskDispatcher
	descRegexp        *regexp.Regexp
	punctuationRegexp *regexp.Regexp
}

func NewSegmentTextGenerator(logger outbound.LoggerPort, scriptGenerator outbound.StoryScriptGeneratorPort,
	workerPool outbound.TaskDispatcher) inbound.SegmentsGeneratorPort {
	return &segmentTextGenerator{
		logger:            logger,
		scriptGenerator:   scriptGenerator,
		workerPool:        workerPool,
		descRegexp:        regexp.MustCompile(`\[(.*?)]`),
		punctuationRegexp: regexp.MustCompile(`[.!?:;]`),
	}
}

func (s *segmentTextGenerator) Generate(ctx context.Context, params inbound.GenerateSegmentsParams) (<-chan domain.Segment, <-chan error) {
	log.Debug().Msg("Starting creating script segments")

	out := make(chan domain.Segment)
	errCh := make(chan error, 5)

	newCtx, cancel := context.WithCancel(ctx)

	tokenCh, scriptErr := s.scriptGenerator.Generate(newCtx, outbound.GenerateStoryScriptRequest{
		Input:         params.Input,
		WordsPerStory: params.WordsPerStory,
	})

	err := s.workerPool.Submit(func() {
		defer close(out)
		defer close(errCh)
		defer cancel()

		var builder strings.Builder
		audioSegmentsCounter := 0
		imageSegmentsCounter := 0
		previousImageID := domain.DefaultImageID

		for {
			select {
			case err, ok := <-scriptErr:
				if ok {
					errCh <- err
					return
				}
			case <-newCtx.Done():
				return
			case token, ok := <-tokenCh:
				if ok {
					builder.WriteString(token)
					segments, newBuffer := s.extractSegments(builder.String())
					builder.Reset()
					builder.WriteString(newBuffer)
					for _, segment := range segments {
						if segment.Type == domain.AudioSegmentType {
							segment.Ordinal = audioSegmentsCounter
							audioSegmentsCounter++
							segment.StoryID = params.StoryID
							segment.BackgroundImageID = previousImageID
						} else if segment.Type == domain.ImageSegmentType {
							segment.Ordinal = imageSegmentsCounter
							imageSegmentsCounter++
							segment.StoryID = params.StoryID
							previousImageID = segment.ID
						}
						s.logger.DebugWithFields("Generated segment", map[string]interface{}{
							"id":   segment.ID,
							"type": segment.Type,
							"ord":  segment.Ordinal,
							"bg":   segment.BackgroundImageID,
							"txt":  segment.Text,
						})

						out <- segment
					}
				} else {
					if builder.Len() > 0 {
						segment := domain.Segment{
							Text:              s.prepareForMediaGeneration(builder.String()),
							Type:              domain.AudioSegmentType,
							ID:                uuid.NewString(),
							BackgroundImageID: previousImageID,
							Ordinal:           audioSegmentsCounter,
						}
						s.logger.DebugWithFields("Generated segment", map[string]interface{}{
							"id":   segment.ID,
							"type": segment.Type,
							"ord":  segment.Ordinal,
							"bg":   segment.BackgroundImageID,
							"txt":  segment.Text,
						})
						out <- segment
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

func (s *segmentTextGenerator) extractSegments(buffer string) ([]domain.Segment, string) {
	segments := make([]domain.Segment, 0)
	imageIndex := s.descRegexp.FindStringIndex(buffer)
	if imageIndex != nil {
		audioText := buffer[:imageIndex[0]]
		if audioText != "" {
			audioSegment := domain.Segment{
				Text: s.prepareForMediaGeneration(audioText),
				Type: domain.AudioSegmentType,
				ID:   uuid.NewString(),
			}
			segments = append(segments, audioSegment)
		}
		imageDescription := buffer[imageIndex[0]+1 : imageIndex[1]-1]
		imageSegment := domain.Segment{
			Text: s.prepareForMediaGeneration(imageDescription),
			Type: domain.ImageSegmentType,
			ID:   uuid.NewString(),
		}
		segments = append(segments, imageSegment)
		furtherSegments, newBuffer := s.extractSegments(buffer[imageIndex[1]:])
		segments = append(segments, furtherSegments...)
		return segments, newBuffer
	} else {
		return segments, buffer
	}
}

func (s *segmentTextGenerator) prepareForMediaGeneration(input string) string {
	result := strings.Replace(input, "\n", "", -1)
	result = strings.Replace(result, "\r", "", -1)
	result = strings.Replace(result, "\t", "", -1)
	result = strings.Replace(result, "\\", "", -1)
	result = strings.TrimSpace(result)

	return result
}
