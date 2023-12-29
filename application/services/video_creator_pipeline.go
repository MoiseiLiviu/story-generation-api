package services

import (
	"context"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/channel_utils"
)

type videoCreatorPipeline struct {
	segmentGenerator   inbound.SegmentsGeneratorPort
	mediaFileGenerator inbound.SegmentMediaFileGenerator
	mediaBinder        inbound.SegmentMediaBinderPort
	logger             outbound.LoggerPort
	workerPool         outbound.TaskDispatcher
	videoCreator       outbound.VideoCreatorPort
	videoPublisher     outbound.VideoPublisherPort
}

func NewVideoCreatorPipeline(
	segmentGenerator inbound.SegmentsGeneratorPort,
	mediaFileGenerator inbound.SegmentMediaFileGenerator,
	mediaBinder inbound.SegmentMediaBinderPort,
	logger outbound.LoggerPort,
	workerPool outbound.TaskDispatcher,
	videoPublisher outbound.VideoPublisherPort,
	videoCreator outbound.VideoCreatorPort) inbound.VideoCreatorPipelinePort {
	return &videoCreatorPipeline{
		segmentGenerator:   segmentGenerator,
		mediaFileGenerator: mediaFileGenerator,
		mediaBinder:        mediaBinder,
		logger:             logger,
		workerPool:         workerPool,
		videoPublisher:     videoPublisher,
		videoCreator:       videoCreator,
	}
}

func (s *videoCreatorPipeline) StartPipeline(ctx context.Context, request inbound.StartPipelineParams) (*inbound.VideoCreatorResponse, error) {
	newCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	segmentsCh, generatorErrCh := s.segmentGenerator.Generate(newCtx, inbound.GenerateSegmentsParams{
		Input:         request.Input,
		StoryID:       request.StoryID,
		WordsPerStory: request.WordsPerStory,
	})
	segmentsMediaFileCh, mediaFileErrCh := s.mediaFileGenerator.Generate(newCtx, segmentsCh, request.VoiceID)

	mergedErrCh, err := channel_utils.MergeChannels(s.workerPool, generatorErrCh, mediaFileErrCh)
	if err != nil {
		s.logger.Error(err, "error merging error channels")
		return nil, err
	}
	err = s.workerPool.Submit(func() {
		for {
			select {
			case <-newCtx.Done():
				return
			case err, ok := <-mergedErrCh:
				if !ok {
					return
				} else {
					s.logger.Error(err, "error in pipeline")
					cancel()
					return
				}
			}
		}
	})

	if err != nil {
		s.logger.Error(err, "error submitting error handler")
		return nil, err
	}

	coupledMediaCh, err := s.mediaBinder.Bind(newCtx, segmentsMediaFileCh)
	if err != nil {
		s.logger.Error(err, "error binding media")
		return nil, err
	}
	createResponse, err := s.videoCreator.Create(coupledMediaCh)
	if err != nil {
		s.logger.Error(err, "error creating video")
		return nil, err
	}

	res, err := s.videoPublisher.Publish(newCtx, outbound.PublishVideoRequest{
		UserID:        request.UserID,
		StoryID:       request.StoryID,
		VideoFileName: createResponse.VideoFileName,
	})
	if err != nil {
		s.logger.Error(err, "error publishing video")
		return nil, err
	}

	return &inbound.VideoCreatorResponse{
		VideoKey:      res.VideoKey,
		VideoRegion:   res.StoreRegion,
		VideoSegments: createResponse.VideoSegments,
	}, nil
}
