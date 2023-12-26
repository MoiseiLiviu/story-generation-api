package services

import (
	"context"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/channel_utils"
	"generate-script-lambda/domain"
)

type videoCreatorPipeline struct {
	segmentGenerator   inbound.SegmentsGeneratorPort
	metadataSaver      inbound.SegmentMetadataSaverPort
	mediaFileGenerator inbound.SegmentMediaFileGenerator
	videoGenerator     inbound.SegmentVideoGenerator
	mediaBinder        inbound.SegmentMediaBinderPort
	concatenateVideos  outbound.ConcatenateVideosPort
	logger             outbound.LoggerPort
	workerPool         outbound.TaskDispatcher
	videoPublisher     outbound.VideoPublisherPort
}

func NewVideoCreatorPipeline(
	segmentGenerator inbound.SegmentsGeneratorPort,
	metadataSaver inbound.SegmentMetadataSaverPort,
	mediaFileGenerator inbound.SegmentMediaFileGenerator,
	videoGenerator inbound.SegmentVideoGenerator,
	mediaBinder inbound.SegmentMediaBinderPort,
	concatenateVideos outbound.ConcatenateVideosPort,
	logger outbound.LoggerPort,
	workerPool outbound.TaskDispatcher,
	videoPublisher outbound.VideoPublisherPort) inbound.VideoCreatorPipelinePort {
	return &videoCreatorPipeline{
		segmentGenerator:   segmentGenerator,
		metadataSaver:      metadataSaver,
		mediaFileGenerator: mediaFileGenerator,
		videoGenerator:     videoGenerator,
		mediaBinder:        mediaBinder,
		concatenateVideos:  concatenateVideos,
		logger:             logger,
		workerPool:         workerPool,
		videoPublisher:     videoPublisher,
	}
}

func (s *videoCreatorPipeline) StartPipeline(ctx context.Context, request inbound.StartPipelineParams) (*inbound.VideoCreatorResponse, error) {
	newCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	segmentsCh, generatorErrCh := s.segmentGenerator.Generate(newCtx, inbound.GenerateSegmentsParams{
		Input:   request.Input,
		StoryID: request.StoryID,
	})
	segmentsMediaFileCh, mediaFileErrCh := s.mediaFileGenerator.Generate(newCtx, segmentsCh, request.VoiceID)
	coupledMediaCh, coupledMediaErrCh := s.mediaBinder.Bind(newCtx, segmentsMediaFileCh)
	videoSegmentsCh, videoErrCh := s.videoGenerator.Generate(newCtx, coupledMediaCh)
	cachedVideoSegments, metadataSaverErrCh := s.metadataSaver.Save(newCtx, videoSegmentsCh, request.StoryID)
	mergedErrCh, err := channel_utils.MergeChannels(s.workerPool, generatorErrCh, mediaFileErrCh, coupledMediaErrCh, videoErrCh, metadataSaverErrCh)
	if err != nil {
		s.logger.Error(err, "error merging error channels")
		return nil, err
	}

	videoSegments, err := s.collectVideoSegments(newCtx, cachedVideoSegments, mergedErrCh)
	if err != nil {
		s.logger.Error(err, "error collecting video segments")
		return nil, err
	}

	mergedVideoFileName, err := s.concatenateVideos.Concatenate(videoSegments)
	if err != nil {
		s.logger.Error(err, "error concatenating video segments")
		return nil, err
	}

	res, err := s.videoPublisher.Publish(newCtx, outbound.PublishVideoRequest{
		UserID:        request.UserID,
		StoryID:       request.StoryID,
		VideoFileName: mergedVideoFileName,
	})
	if err != nil {
		s.logger.Error(err, "error publishing video")
		return nil, err
	}

	return &inbound.VideoCreatorResponse{
		VideoKey:    res.VideoKey,
		VideoRegion: res.StoreRegion,
	}, nil
}

func (s *videoCreatorPipeline) collectVideoSegments(ctx context.Context,
	videoSegmentsCh <-chan domain.VideoSegment, errCh <-chan error) ([]domain.VideoSegment, error) {
	videoSegments := make([]domain.VideoSegment, 0)
	for {
		select {
		case err := <-errCh:
			s.logger.Error(err, "error in pipeline")
			return nil, err
		case <-ctx.Done():
			s.logger.Info("context cancelled")
			return nil, nil
		case videoSegment, ok := <-videoSegmentsCh:
			if !ok {
				return videoSegments, nil
			} else {
				videoSegments = append(videoSegments, videoSegment)
			}
		}
	}
}
