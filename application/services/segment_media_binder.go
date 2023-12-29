package services

import (
	"context"
	"fmt"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/domain"
)

const DefaultBackgroundImage = "../../assets/default.jpg"

type segmentMediaBinder struct {
	logger     outbound.LoggerPort
	workerPool outbound.TaskDispatcher
}

func NewSegmentMediaBinder(logger outbound.LoggerPort, workerPool outbound.TaskDispatcher) inbound.SegmentMediaBinderPort {
	return &segmentMediaBinder{
		logger:     logger,
		workerPool: workerPool,
	}
}

func (s *segmentMediaBinder) Bind(ctx context.Context, segmentsCh <-chan domain.SegmentWithMediaFile) ([]domain.AudioWithImageBackground, error) {
	imageSegments := make([]domain.SegmentWithMediaFile, 0)
	audioSegments := make([]domain.SegmentWithMediaFile, 0)
	coupledSegments := make([]domain.AudioWithImageBackground, 0)
	for segment := range segmentsCh {
		select {
		case <-ctx.Done():
			s.logger.Debug("Media Binder context done")
			return nil, fmt.Errorf("context done")
		default:
			s.logger.DebugWithFields("Media Binder processing segment", map[string]interface{}{
				"id":   segment.ID,
				"type": segment.Type,
				"bg":   segment.BackgroundImageID,
				"fn":   segment.FileName,
			})
			if segment.Type == domain.AudioSegmentType {
				if segment.BackgroundImageID == domain.DefaultImageID {
					s.logger.DebugWithFields("Found audio segment with default background image", map[string]interface{}{
						"id":   segment.ID,
						"bg":   segment.BackgroundImageID,
						"fn":   segment.FileName,
						"bgfn": DefaultBackgroundImage,
					})
					coupledSegments = append(coupledSegments, domain.AudioWithImageBackground{
						SegmentWithMediaFile:    segment,
						BackgroundImageFileName: DefaultBackgroundImage,
					})
				} else {
					image := s.getSegmentById(segment.BackgroundImageID, imageSegments)
					if image == nil {
						audioSegments = append(audioSegments, segment)
					} else {
						s.logger.DebugWithFields("Found audio segment image", map[string]interface{}{
							"id":   segment.ID,
							"bg":   segment.BackgroundImageID,
							"fn":   segment.FileName,
							"bgfn": image.FileName,
						})
						coupledSegments = append(coupledSegments, domain.AudioWithImageBackground{
							SegmentWithMediaFile:    segment,
							BackgroundImageFileName: image.FileName,
						})
					}
				}
			} else {
				pendingAudio := s.getPendingAudioByImageID(segment.ID, audioSegments)
				for _, audio := range pendingAudio {
					s.logger.DebugWithFields("Found audio segment image", map[string]interface{}{
						"id":   audio.ID,
						"bg":   audio.BackgroundImageID,
						"fn":   audio.FileName,
						"bgfn": segment.FileName,
					})
					coupledSegments = append(coupledSegments, domain.AudioWithImageBackground{
						SegmentWithMediaFile:    audio,
						BackgroundImageFileName: segment.FileName,
					})
				}
				audioSegments = s.removePendingSegments(audioSegments, pendingAudio)
				imageSegments = append(imageSegments, segment)
			}
		}
	}

	return coupledSegments, nil
}

func (s *segmentMediaBinder) removePendingSegments(allSegments, pendingSegments []domain.SegmentWithMediaFile) []domain.SegmentWithMediaFile {
	pendingMap := make(map[string]struct{})
	for _, seg := range pendingSegments {
		pendingMap[seg.ID] = struct{}{}
	}

	var filteredSegments []domain.SegmentWithMediaFile
	for _, seg := range allSegments {
		if _, exists := pendingMap[seg.ID]; !exists {
			filteredSegments = append(filteredSegments, seg)
		}
	}

	return filteredSegments
}

func (s *segmentMediaBinder) getPendingAudioByImageID(id string, segments []domain.SegmentWithMediaFile) []domain.SegmentWithMediaFile {
	pending := make([]domain.SegmentWithMediaFile, 0)
	for _, segment := range segments {
		if segment.BackgroundImageID == id {
			pending = append(pending, segment)
		}
	}
	return pending
}

func (s *segmentMediaBinder) getSegmentById(id string, segments []domain.SegmentWithMediaFile) *domain.SegmentWithMediaFile {
	for _, segment := range segments {
		if segment.ID == id {
			return &segment
		}
	}
	return nil
}
