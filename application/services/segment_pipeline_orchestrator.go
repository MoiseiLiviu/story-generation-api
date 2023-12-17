package services

import (
	"context"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/domain"
	"github.com/panjf2000/ants/v2"
	"sync"
)

type segmentPipelineOrchestrator struct {
	workerPool       *ants.Pool
	segmentGenerator inbound.SegmentsGeneratorPort
	mediaEnhancer    inbound.SegmentMediaEnhancerPort
	mediaSaver       inbound.SegmentMediaSaverPort
	metadataSaver    inbound.SegmentMetadataSaverPort
}

func NewSegmentPipelineOrchestrator(workerPool *ants.Pool, segmentGenerator inbound.SegmentsGeneratorPort,
	mediaEnhancer inbound.SegmentMediaEnhancerPort, mediaSaver inbound.SegmentMediaSaverPort,
	metadataSaver inbound.SegmentMetadataSaverPort) inbound.SegmentPipelineOrchestrator {
	return &segmentPipelineOrchestrator{
		workerPool:       workerPool,
		segmentGenerator: segmentGenerator,
		mediaEnhancer:    mediaEnhancer,
		mediaSaver:       mediaSaver,
		metadataSaver:    metadataSaver,
	}
}

func (s *segmentPipelineOrchestrator) StartPipeline(ctx context.Context, request inbound.StartPipelineParams) (<-chan domain.SegmentEvent, <-chan error) {
	newCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	segmentCh, segmentGeneratorErrCh := s.segmentGenerator.Generate(newCtx, inbound.GenerateSegmentsParams{
		Input:   request.Input,
		StoryID: request.StoryID,
	})

	segmentWithMediaCh, mediaEnhancerErrCh := s.mediaEnhancer.Generate(newCtx, segmentCh, request.VoiceID)

	segmentWithMediaUrlCh, mediaSaverErrCh := s.mediaSaver.Save(newCtx, segmentWithMediaCh, request.StoryID)

	segmentEventsCh, metadataSaverErrCh := s.metadataSaver.Save(newCtx, segmentWithMediaUrlCh)

	mergerErrCh := s.mergeErrorChannels(segmentGeneratorErrCh, mediaEnhancerErrCh, mediaSaverErrCh, metadataSaverErrCh)

	return segmentEventsCh, mergerErrCh
}

func (s *segmentPipelineOrchestrator) mergeErrorChannels(channels ...<-chan error) <-chan error {
	var wg sync.WaitGroup
	merged := make(chan error)

	output := func(c <-chan error) {
		for err := range c {
			merged <- err
		}
		wg.Done()
	}

	wg.Add(len(channels))
	for _, c := range channels {
		err := s.workerPool.Submit(func() {
			output(c)
		})
		if err != nil {
			merged <- err
			wg.Done()
		}
	}

	err := s.workerPool.Submit(func() {
		wg.Wait()
		close(merged)
	})
	if err != nil {
		merged <- err
	}

	return merged
}
