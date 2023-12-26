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
	imageRegexp       *regexp.Regexp
}

func NewSegmentTextGenerator(logger outbound.LoggerPort, scriptGenerator outbound.StoryScriptGeneratorPort,
	workerPool outbound.TaskDispatcher) inbound.SegmentsGeneratorPort {
	return &segmentTextGenerator{
		logger:            logger,
		scriptGenerator:   scriptGenerator,
		workerPool:        workerPool,
		descRegexp:        regexp.MustCompile(`\[(.*?)]`),
		punctuationRegexp: regexp.MustCompile(`[.!?:;]`),
		imageRegexp:       regexp.MustCompile(`\{[^}]*\}`),
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
		previousImageID := ""

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
						select {
						case errCh <- err:
						case <-newCtx.Done():
						}
						return
					}
					builder.Reset()
					builder.WriteString(newBuffer)
					for _, segment := range segments {
						if segment.Type == domain.AudioSegmentType {
							segment.Ordinal = audioSegmentsCounter
							audioSegmentsCounter++
							segment.StoryID = params.StoryID
							if segment.BackgroundImageID == "" && previousImageID == "" {
								segment.BackgroundImageID = domain.DefaultImageID
							} else if segment.BackgroundImageID == "" {
								segment.BackgroundImageID = previousImageID
							}
						} else if segment.Type == domain.ImageSegmentType {
							segment.Ordinal = imageSegmentsCounter
							imageSegmentsCounter++
							segment.StoryID = params.StoryID
							previousImageID = segment.ID
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
	if untilIndex := s.canExtractAudioSegment(buffer); untilIndex != -1 {
		segment, newBuffer, extractErr := s.extractAudioSegment(buffer, untilIndex)
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

func (s *segmentTextGenerator) isParserInsideBrackets(buffer string) bool {
	return strings.Contains(buffer, "[")
}

func (s *segmentTextGenerator) canExtractAudioSegment(buffer string) int {
	matches := s.punctuationRegexp.FindAllStringIndex(buffer, -1)

	if len(matches) >= MaxSentencesBeforePublishing {
		return matches[MaxSentencesBeforePublishing-1][0]
	}

	return -1
}

func (s *segmentTextGenerator) extractAudioSegment(buffer string, untilIndex int) (audioSegment domain.Segment, newBuffer string, err error) {
	accText := buffer[:untilIndex+1]
	var extractedText string

	imageMatches := s.imageRegexp.FindAllStringIndex(accText, -1)
	nrOfImages := len(imageMatches)

	if nrOfImages > 1 {
		extractedText = accText[:imageMatches[1][0]]
		newBuffer = accText[imageMatches[1][0]:]
	} else {
		extractedText = accText
		newBuffer = buffer[untilIndex+1:]
	}

	imageID := s.imageRegexp.FindString(extractedText)
	if imageID != "" {
		imageID = imageID[1 : len(imageID)-1]
	}

	audioSegment = domain.Segment{
		Text:              s.prepareTextForTTS(extractedText),
		Type:              domain.AudioSegmentType,
		BackgroundImageID: imageID,
		ID:                uuid.NewString(),
	}

	log.Debug().
		Str("segmentID", audioSegment.ID).
		Str("text", audioSegment.Text).
		Str("BackgroundImageID", audioSegment.BackgroundImageID).
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

func (s *segmentTextGenerator) prepareTextForTTS(input string) string {
	result := s.imageRegexp.ReplaceAllString(input, "")
	result = s.removeEmptySpaces(result)

	return result
}

func (s *segmentTextGenerator) removeEmptySpaces(input string) string {
	result := strings.Replace(input, "\n", "", -1)
	result = strings.Replace(result, "\r", "", -1)
	result = strings.Replace(result, "\t", "", -1)
	result = strings.Replace(result, "\\", "", -1)
	result = strings.TrimSpace(result)

	return result
}
