package services

import (
	"context"
	"fmt"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/domain"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"regexp"
	"strings"
)

const (
	MaxSentencesBeforePublishing = 3
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

	tokenCh, scriptErr := s.scriptGenerator.Generate(newCtx, params.Input)

	err := s.workerPool.Submit(func() {
		defer close(out)
		defer close(errCh)
		defer cancel()

		var builder strings.Builder
		audioSegmentsCounter := 0
		imageSegmentsCounter := 0

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
					newBuffer, segments, err := s.extractSegments(builder.String())
					if err != nil {
						errCh <- err
						return
					}
					builder.Reset()
					builder.WriteString(newBuffer)
					for _, segment := range segments {
						if segment.Type == domain.AudioSegmentType {
							segment.Ordinal = audioSegmentsCounter
							audioSegmentsCounter++
							segment.StoryID = params.StoryID
						} else if segment.Type == domain.ImageSegmentType {
							segment.Ordinal = imageSegmentsCounter
							imageSegmentsCounter++
							segment.StoryID = params.StoryID
						}
						out <- segment
					}
				} else {
					if builder.Len() > 0 {
						out <- domain.NewSegment(builder.String(), domain.AudioSegmentType, uuid.NewString(), params.StoryID, audioSegmentsCounter)
						log.Info().Msg("Finished reading from stream.")
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

func (s *segmentTextGenerator) extractSegments(buffer string) (resultBuffer string, segments []domain.Segment, err error) {
	segments = make([]domain.Segment, 0)
	if s.canExtractImageSegment(buffer) {
		segment, newBuffer, extractErr := s.extractImageSegment(buffer)
		if extractErr != nil {
			err = extractErr
			return
		}
		resultBuffer, segments, err = s.extractSegments(newBuffer)
		segments = append([]domain.Segment{segment}, segments...)
		return
	}
	if s.isParserInsideBrackets(buffer) {
		resultBuffer = buffer
		return
	}
	if s.canExtractAudioSegment(buffer) {
		segment, newBuffer, extractErr := s.extractAudioSegment(buffer)
		if extractErr != nil {
			err = extractErr
			return
		}
		resultBuffer, segments, err = s.extractSegments(newBuffer)
		segments = append([]domain.Segment{segment}, segments...)
		return
	}
	resultBuffer = buffer

	return
}

func (s *segmentTextGenerator) canExtractImageSegment(buffer string) bool {
	return s.descRegexp.MatchString(buffer)
}

func (s *segmentTextGenerator) canExtractAudioSegment(buffer string) bool {
	matches := s.punctuationRegexp.FindAllString(buffer, -1)

	return len(matches) >= 3
}

func (s *segmentTextGenerator) isParserInsideBrackets(buffer string) bool {
	return strings.Contains(buffer, "[")
}

func (s *segmentTextGenerator) indexOfLastAdmissiblePunctuationMark(str string) int {
	matches := s.punctuationRegexp.FindAllStringIndex(str, -1)

	if len(matches) >= MaxSentencesBeforePublishing {
		return matches[MaxSentencesBeforePublishing-1][0]
	}

	return -1
}

func (s *segmentTextGenerator) extractAudioSegment(buffer string) (audioSegment domain.Segment, newBuffer string, err error) {
	lastPunctuationIndex := s.indexOfLastAdmissiblePunctuationMark(buffer)
	if lastPunctuationIndex == -1 {
		err = fmt.Errorf("no punctuation mark found in buffer")
		return
	}

	audioSegment = domain.Segment{
		Text: buffer[:lastPunctuationIndex+1],
		Type: domain.AudioSegmentType,
		ID:   uuid.NewString(),
	}
	newBuffer = buffer[lastPunctuationIndex+1:]

	log.Debug().
		Str("segmentID", audioSegment.ID).
		Str("text", audioSegment.Text).
		Msg("Extracted text segment for audio handling")

	return
}

func (s *segmentTextGenerator) extractImageSegment(buffer string) (imageSegment domain.Segment, newBuffer string, err error) {
	match := s.descRegexp.FindStringSubmatch(buffer)
	if match == nil {
		err = fmt.Errorf("description enclosing character found, but no expression matching description clause could be extracted")
	}

	imageSegment = domain.Segment{
		Text: match[1],
		Type: domain.ImageSegmentType,
		ID:   uuid.NewString(),
	}

	replaced := false
	newBuffer = s.descRegexp.ReplaceAllStringFunc(buffer, func(match string) string {
		if !replaced {
			replaced = true
			return "{" + imageSegment.ID + "}"
		}
		return match
	})

	log.Debug().
		Str("segmentID", imageSegment.ID).
		Str("description", imageSegment.Text).
		Msg("Extracted description for image handling")

	return
}
