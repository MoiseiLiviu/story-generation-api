package services

import (
	"context"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/channel_utils"
	"generate-script-lambda/domain"
)

type segmentPipelineOrchestrator struct {
	logger           outbound.LoggerPort
	workerPool       outbound.TaskDispatcher
	segmentGenerator inbound.SegmentsGeneratorPort
	mediaEnhancer    inbound.SegmentMediaEnhancerPort
	mediaSaver       inbound.SegmentMediaSaverPort
	metadataSaver    inbound.SegmentMetadataSaverPort
}

func NewSegmentPipelineOrchestrator(logger outbound.LoggerPort, workerPool outbound.TaskDispatcher,
	segmentGenerator inbound.SegmentsGeneratorPort, mediaEnhancer inbound.SegmentMediaEnhancerPort,
	mediaSaver inbound.SegmentMediaSaverPort, metadataSaver inbound.SegmentMetadataSaverPort) inbound.SegmentPipelineOrchestrator {
	return &segmentPipelineOrchestrator{
		logger:           logger,
		workerPool:       workerPool,
		segmentGenerator: segmentGenerator,
		mediaEnhancer:    mediaEnhancer,
		mediaSaver:       mediaSaver,
		metadataSaver:    metadataSaver,
	}
}

func (s *segmentPipelineOrchestrator) StartPipeline(ctx context.Context, request inbound.StartPipelineParams) (<-chan domain.SegmentEvent, <-chan error) {
	segmentCh, segmentGeneratorErrCh := s.segmentGenerator.Generate(ctx, inbound.GenerateSegmentsParams{
		Input:   request.Input,
		StoryID: request.StoryID,
	})

	segmentWithMediaCh, mediaEnhancerErrCh := s.mediaEnhancer.Enhance(ctx, segmentCh, request.VoiceID)

	segmentWithMediaUrlCh, mediaSaverErrCh := s.mediaSaver.Save(ctx, segmentWithMediaCh, request.UserID)

	segmentEventsCh, metadataSaverErrCh := s.metadataSaver.Save(ctx, segmentWithMediaUrlCh)

	mergerErrCh, err := channel_utils.MergeChannels(s.workerPool, segmentGeneratorErrCh, mediaEnhancerErrCh, mediaSaverErrCh, metadataSaverErrCh)

	if err != nil {
		out := make(chan domain.SegmentEvent)
		errCh := make(chan error)
		errCh <- err
		close(out)
		close(errCh)
		return out, errCh
	}

	return segmentEventsCh, mergerErrCh
}
