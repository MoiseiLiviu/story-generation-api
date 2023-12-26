package services

import (
	"context"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/domain"
	"github.com/google/uuid"
	"io"
	"os"
	"sync"
)

type segmentMediaFileGenerator struct {
	logger       outbound.LoggerPort
	imageCreator outbound.ImageGeneratorPort
	audioCreator outbound.AudioGeneratorPort
	workerPool   outbound.TaskDispatcher
}

func NewSegmentMediaFileGenerator(logger outbound.LoggerPort, imageCreator outbound.ImageGeneratorPort,
	audioCreator outbound.AudioGeneratorPort, workerPool outbound.TaskDispatcher) inbound.SegmentMediaFileGenerator {
	return &segmentMediaFileGenerator{
		logger:       logger,
		imageCreator: imageCreator,
		audioCreator: audioCreator,
		workerPool:   workerPool,
	}
}

func (s *segmentMediaFileGenerator) Generate(ctx context.Context, segmentCh <-chan domain.Segment, voiceID string) (<-chan domain.SegmentWithMediaFile, <-chan error) {
	out := make(chan domain.SegmentWithMediaFile)
	errCh := make(chan error, 5)

	newCtx, cancel := context.WithCancel(ctx)

	err := s.workerPool.Submit(func() {
		defer close(out)
		defer close(errCh)
		defer cancel()

		var wg sync.WaitGroup

		for seg := range segmentCh {
			select {
			case <-newCtx.Done():
				return
			default:
				wg.Add(1)
				segment := seg
				err := s.workerPool.Submit(func() {
					defer wg.Done()

					segmentWithMediaFile, err := s.generateMediaFile(newCtx, segment, voiceID)
					if err != nil {
						select {
						case errCh <- err:
						case <-newCtx.Done():
						}
						return
					}

					select {
					case out <- *segmentWithMediaFile:
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

		wg.Wait()
	})

	if err != nil {
		errCh <- err
	}

	return out, errCh
}

func (s *segmentMediaFileGenerator) writeMediaToFile(reader io.Reader) (string, error) {
	fileName := "/tmp/" + uuid.NewString()
	file, err := os.Create(fileName)
	if err != nil {
		return "", err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			s.logger.Error(err, "Failed to close the file")
		}
	}(file)

	_, err = io.Copy(file, reader)
	if err != nil {
		return "", err
	}

	return file.Name(), nil
}

func (s *segmentMediaFileGenerator) generateMediaFile(ctx context.Context, segment domain.Segment, voiceID string) (*domain.SegmentWithMediaFile, error) {
	var mediaReader io.ReadCloser
	if segment.Type == domain.AudioSegmentType {
		reader, err := s.audioCreator.Generate(ctx, outbound.GenerateAudioRequest{
			Text:    segment.Text,
			VoiceID: voiceID,
		})
		if err != nil {
			s.logger.Error(err, "Failed to generate audio")
			return nil, err
		}
		mediaReader = reader
	} else {
		reader, err := s.imageCreator.Generate(ctx, segment.Text)
		if err != nil {
			s.logger.Error(err, "Failed to generate image")
			return nil, err
		}
		mediaReader = reader
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			s.logger.Error(err, "Failed to close the response body")
		}
	}(mediaReader)

	fileName, err := s.writeMediaToFile(mediaReader)
	if err != nil {
		s.logger.Error(err, "Failed to write media to file")
		return nil, err
	}

	return &domain.SegmentWithMediaFile{
		FileName: fileName,
		Segment:  segment,
	}, nil
}
